package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cbt-api/internal/middleware"
	"cbt-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

// newPlatformRouteTestServer wires the real AuthMiddleware + PlatformOwnerOnly
// chain (the same two-middleware stack api/routes/routes.go's
// setupPlatformRoutes uses) in front of a handful of dummy routes mirroring
// the real /platform/* surface, WITHOUT a live database - this test suite
// exists to prove the authorization gate itself (role in, status out),
// not the handlers' business logic, which needs a real DB and is exercised
// separately. No request here reaches a PlatformHandler method at all.
func newPlatformRouteTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	platform := r.Group("/api/v1/platform")
	platform.Use(middleware.AuthMiddleware(), middleware.PlatformOwnerOnly())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
	{
		platform.GET("/tenants", ok)
		platform.GET("/tenants/:id", ok)
		platform.PUT("/tenants/:id/branding", ok)
		platform.PUT("/tenants/:id/provision", ok)
		platform.PUT("/tenants/:id/suspend", ok)
		platform.GET("/audit-logs", ok)
		platform.GET("/lan-config/:schoolId", ok)
		platform.PUT("/lan-config/:schoolId", ok)
		platform.POST("/lan-config/:schoolId/test", ok)
		platform.GET("/lan-config/:schoolId/history", ok)
		platform.POST("/lan-config/:schoolId/rollback", ok)
	}
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func platformTestToken(t *testing.T, role string) string {
	t.Helper()
	token, err := utils.GenerateJWT("user-"+role, role+"@example.com", role, "school-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT(%s) failed: %v", role, err)
	}
	return token
}

// TestPlatformRoutes_OnlyPlatformOwnerAllowed is the Phase 10.5-required
// authorization matrix for the LAN Setup Dashboard and tenant-registry
// surface: platform_owner allowed, every school-scoped role (including
// admin and super_admin - a school admin must never reach another
// school's tenant record or LAN config) and unauthenticated denied, for
// every route this handler exposes.
func TestPlatformRoutes_OnlyPlatformOwnerAllowed(t *testing.T) {
	srv := newPlatformRouteTestServer(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/platform/tenants"},
		{http.MethodGet, "/api/v1/platform/tenants/school-1"},
		{http.MethodPut, "/api/v1/platform/tenants/school-1/branding"},
		{http.MethodPut, "/api/v1/platform/tenants/school-1/provision"},
		{http.MethodPut, "/api/v1/platform/tenants/school-1/suspend"},
		{http.MethodGet, "/api/v1/platform/audit-logs"},
		{http.MethodGet, "/api/v1/platform/lan-config/school-1"},
		{http.MethodPut, "/api/v1/platform/lan-config/school-1"},
		{http.MethodPost, "/api/v1/platform/lan-config/school-1/test"},
		{http.MethodGet, "/api/v1/platform/lan-config/school-1/history"},
		{http.MethodPost, "/api/v1/platform/lan-config/school-1/rollback"},
	}

	roles := []struct {
		role       string
		wantStatus int
	}{
		{"platform_owner", http.StatusOK},
		{"admin", http.StatusForbidden},
		{"super_admin", http.StatusForbidden},
		{"teacher", http.StatusForbidden},
		{"invigilator", http.StatusForbidden},
		{"student", http.StatusForbidden},
		{"parent", http.StatusForbidden},
	}

	for _, rt := range routes {
		for _, rc := range roles {
			name := rc.role + "_" + rt.method + "_" + rt.path
			t.Run(name, func(t *testing.T) {
				req, _ := http.NewRequest(rt.method, srv.URL+rt.path, nil)
				req.Header.Set("Authorization", "Bearer "+platformTestToken(t, rc.role))
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != rc.wantStatus {
					t.Fatalf("role=%s %s %s: got status %d, want %d", rc.role, rt.method, rt.path, resp.StatusCode, rc.wantStatus)
				}
			})
		}

		// Unauthenticated: no Authorization header at all must also be
		// denied (401, before PlatformOwnerOnly ever runs), proving there
		// is no anonymous/default-allow path onto this surface.
		t.Run("unauthenticated_"+rt.method+"_"+rt.path, func(t *testing.T) {
			req, _ := http.NewRequest(rt.method, srv.URL+rt.path, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s unauthenticated: got status %d, want 401", rt.method, rt.path, resp.StatusCode)
			}
		})
	}
}

// TestPlatformRoutes_TamperedRoleHeaderIsIgnored proves a request carrying
// a non-platform_owner JWT role claim cannot escalate by also sending an
// arbitrary client-supplied "X-Role" (or similarly named) header or query
// parameter: only the server-verified JWT claim (set by AuthMiddleware from
// a cryptographically validated token) ever reaches PlatformOwnerOnly's
// check - RoleMiddleware reads exclusively from the Gin context key
// AuthMiddleware itself set, never from any request header/param.
func TestPlatformRoutes_TamperedRoleHeaderIsIgnored(t *testing.T) {
	srv := newPlatformRouteTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/platform/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+platformTestToken(t, "admin"))
	// Attempt to spoof a platform_owner identity via headers/params a naive
	// implementation might have trusted instead of the verified JWT claim.
	req.Header.Set("X-Role", "platform_owner")
	req.Header.Set("X-User-Role", "platform_owner")
	q := req.URL.Query()
	q.Set("role", "platform_owner")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("tampered-header admin request: got status %d, want 403 (spoofed role header must be ignored)", resp.StatusCode)
	}
}
