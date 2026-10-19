package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreMiddleware "cbt-api/internal/middleware"
	"cbt-api/internal/nodesync/repository"
	"cbt-api/internal/nodesync/service"
	"cbt-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- LAN Setup Dashboard: CreateCredential tenant isolation ----
//
// Before this fix, POST /admin/school-nodes/credentials took its target
// school_id straight from the client request body with no check at all
// against the caller's own school - any authenticated admin, from ANY
// school, could mint a valid, indefinitely-lived node credential for ANY
// OTHER school by naming its ID, then use that credential (via
// NodeAuthMiddleware, which trusts it alone) to push forged results into
// that school or pull its students/exams. This is a full HTTP-level
// regression test (real AuthMiddleware + RoleMiddleware chain, real
// JWTs, real handler, real in-memory-SQLite-backed service), mirroring
// internal/middleware/auth_test.go's pattern for the same reason: the
// vulnerability was in the handler/route wiring, not in any function a
// narrower unit test could call directly.

func setupCredentialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE school_node_credentials (
		id TEXT PRIMARY KEY, school_id TEXT NOT NULL, label TEXT,
		key_hash TEXT NOT NULL, last_used_at DATETIME, revoked_at DATETIME,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func newCredentialTestServer(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db := setupCredentialTestDB(t)
	svc := service.NewNodeSyncService(repository.NewNodeSyncRepository(db))
	h := NewNodeSyncHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminNodes := r.Group("/admin/school-nodes")
	adminNodes.Use(coreMiddleware.AuthMiddleware(), coreMiddleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
	{
		adminNodes.POST("/credentials", h.CreateCredential)
	}
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, db
}

func credentialTokenFor(t *testing.T, role, schoolID string) string {
	t.Helper()
	token, err := utils.GenerateJWT("user-"+role, role+"@example.com", role, schoolID, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT(%s) failed: %v", role, err)
	}
	return token
}

func postCreateCredential(t *testing.T, srv *httptest.Server, token, targetSchoolID string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"school_id": targetSchoolID, "label": "test-node"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/admin/school-nodes/credentials", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestCreateCredential_CrossTenant_Rejected(t *testing.T) {
	srv, db := newCredentialTestServer(t)
	token := credentialTokenFor(t, "admin", "11111111-1111-1111-1111-111111111111")

	resp := postCreateCredential(t, srv, token, "22222222-2222-2222-2222-222222222222")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected School A's admin to be rejected minting a credential for School B, got status %d", resp.StatusCode)
	}

	var count int64
	db.Table("school_node_credentials").Where("school_id = ?", "22222222-2222-2222-2222-222222222222").Count(&count)
	if count != 0 {
		t.Fatal("a node credential for School B was created by a cross-tenant request that should have been rejected")
	}
}

func TestCreateCredential_SameTenant_Allowed(t *testing.T) {
	srv, db := newCredentialTestServer(t)
	token := credentialTokenFor(t, "admin", "11111111-1111-1111-1111-111111111111")

	resp := postCreateCredential(t, srv, token, "11111111-1111-1111-1111-111111111111")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected School A's admin to mint a credential for its own school, got status %d", resp.StatusCode)
	}

	var count int64
	db.Table("school_node_credentials").Where("school_id = ?", "11111111-1111-1111-1111-111111111111").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 credential to be created for School A, got %d", count)
	}
}

func TestCreateCredential_SuperAdmin_SameTenant_Allowed(t *testing.T) {
	srv, _ := newCredentialTestServer(t)
	token := credentialTokenFor(t, "super_admin", "11111111-1111-1111-1111-111111111111")

	resp := postCreateCredential(t, srv, token, "11111111-1111-1111-1111-111111111111")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected School A's super_admin to mint a credential for its own school, got status %d", resp.StatusCode)
	}
}

func TestCreateCredential_PlatformOwner_Bypasses(t *testing.T) {
	srv, db := newCredentialTestServer(t)
	token := credentialTokenFor(t, "platform_owner", "")

	resp := postCreateCredential(t, srv, token, "22222222-2222-2222-2222-222222222222")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected platform_owner to mint a credential for any school, got status %d", resp.StatusCode)
	}

	var count int64
	db.Table("school_node_credentials").Where("school_id = ?", "22222222-2222-2222-2222-222222222222").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 credential to be created for School B, got %d", count)
	}
}

func TestCreateCredential_Teacher_RejectedByRoute(t *testing.T) {
	srv, _ := newCredentialTestServer(t)
	token := credentialTokenFor(t, "teacher", "11111111-1111-1111-1111-111111111111")

	resp := postCreateCredential(t, srv, token, "11111111-1111-1111-1111-111111111111")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected a teacher to be rejected at the route's role gate, got status %d", resp.StatusCode)
	}
}
