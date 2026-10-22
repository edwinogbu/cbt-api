package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreMiddleware "cbt-api/internal/middleware"
	"cbt-api/internal/academic/repository"
	"cbt-api/internal/academic/service"
	"cbt-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- GetAllSchools: school.code redaction ----
//
// Code is a registration secret - presented at POST /auth/register to
// self-provision a teacher/parent account (or, before a separate fix,
// an admin account) against that school. Before this fix, this
// list-every-school endpoint (reachable by any teacher, not just
// admins) returned every school's code in full, including every OTHER
// school's - handing any teacher at School A a ready-to-use
// registration secret for School B. A caller's own single school by ID
// (GetSchool) is unaffected; this only redacts the field from the
// cross-school listing, and only for non-platform-owner callers.

func setupSchoolHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE schools (
		id TEXT PRIMARY KEY, name TEXT, code TEXT, address TEXT, phone TEXT,
		email TEXT, logo TEXT, website TEXT, status TEXT DEFAULT 'PENDING',
		is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func seedSchoolForHandlerTest(t *testing.T, db *gorm.DB, id, name, code string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO schools (id, name, code, created_at) VALUES (?, ?, ?, ?)",
		id, name, code, time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed school: %v", err)
	}
}

func schoolTokenFor(t *testing.T, role, schoolID string) string {
	t.Helper()
	token, err := utils.GenerateJWT("user-"+role, role+"@example.com", role, schoolID, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT(%s) failed: %v", role, err)
	}
	return token
}

func getAllSchools(t *testing.T, srv *httptest.Server, token string) map[string]interface{} {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/schools", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return body
}

func TestGetAllSchools_Teacher_CodeRedacted(t *testing.T) {
	db := setupSchoolHandlerTestDB(t)
	seedSchoolForHandlerTest(t, db, "school-a", "School A", "SCH-AAA111")
	seedSchoolForHandlerTest(t, db, "school-b", "School B", "SCH-BBB222")

	svc := service.NewSchoolService(repository.NewSchoolRepository(db))
	h := NewSchoolHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	academic := r.Group("/")
	academic.Use(coreMiddleware.AuthMiddleware(), coreMiddleware.RoleMiddleware("admin", "super_admin", "teacher", "platform_owner"))
	{
		academic.GET("/schools", h.GetAllSchools)
	}
	srv := httptest.NewServer(r)
	defer srv.Close()

	token := schoolTokenFor(t, "teacher", "school-a")
	body := getAllSchools(t, srv, token)

	data, ok := body["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T", body["data"])
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 schools, got %d", len(data))
	}
	for _, item := range data {
		school, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected school object, got %T", item)
		}
		if code, exists := school["code"]; exists && code != "" {
			t.Fatalf("expected code to be redacted for a teacher caller, got %q for school %v", code, school["id"])
		}
	}
}

func TestGetAllSchools_PlatformOwner_SeesRealCodes(t *testing.T) {
	db := setupSchoolHandlerTestDB(t)
	seedSchoolForHandlerTest(t, db, "school-a", "School A", "SCH-AAA111")

	svc := service.NewSchoolService(repository.NewSchoolRepository(db))
	h := NewSchoolHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	academic := r.Group("/")
	academic.Use(coreMiddleware.AuthMiddleware(), coreMiddleware.RoleMiddleware("admin", "super_admin", "teacher", "platform_owner"))
	{
		academic.GET("/schools", h.GetAllSchools)
	}
	srv := httptest.NewServer(r)
	defer srv.Close()

	token := schoolTokenFor(t, "platform_owner", "")
	body := getAllSchools(t, srv, token)

	data, ok := body["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 school, got %v", body["data"])
	}
	school := data[0].(map[string]interface{})
	if school["code"] != "SCH-AAA111" {
		t.Fatalf("expected platform_owner to see the real code, got %v", school["code"])
	}
}
