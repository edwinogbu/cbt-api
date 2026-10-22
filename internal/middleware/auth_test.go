package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cbt-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

// newPlatformOwnerTestServer builds a real HTTP server running the
// actual AuthMiddleware + PlatformOwnerOnly chain on a dummy protected
// route, exactly as any real platform-owner-only endpoint (the tenant
// registry, platform branding config, the LAN Setup Dashboard - Phases
// 2-10) will be wired. No mocking of the auth layer: requests carry real
// JWTs minted by the same utils.GenerateJWT the real login flow uses.
func newPlatformOwnerTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	protected := r.Group("/platform-only")
	protected.Use(AuthMiddleware(), PlatformOwnerOnly())
	protected.GET("/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func tokenFor(t *testing.T, role string) string {
	t.Helper()
	token, err := utils.GenerateJWT("user-"+role, role+"@example.com", role, "school-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT(%s) failed: %v", role, err)
	}
	return token
}

// TestPlatformOwnerOnly_RejectsEverySchoolScopedRole is the Phase 1
// acceptance test: a platform-owner-only endpoint must accept exactly
// "platform_owner" and reject every other real role this system has -
// critically including admin and super_admin, which is the whole point
// (today's super_admin is a school-level role; it must not inherit
// platform authority just because the name sounds privileged).
func TestPlatformOwnerOnly_RejectsEverySchoolScopedRole(t *testing.T) {
	srv := newPlatformOwnerTestServer(t)

	cases := []struct {
		role       string
		wantStatus int
	}{
		{"platform_owner", http.StatusOK},
		{"admin", http.StatusForbidden},
		{"super_admin", http.StatusForbidden},
		{"teacher", http.StatusForbidden},
		{"student", http.StatusForbidden},
		{"parent", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/platform-only/tenants", nil)
			req.Header.Set("Authorization", "Bearer "+tokenFor(t, tc.role))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("role=%s: got status %d, want %d", tc.role, resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

// TestPlatformOwnerOnly_RejectsUnauthenticated confirms there is no
// anonymous/default-allow path - a request with no token and a request
// with a garbage token must both be rejected before PlatformOwnerOnly
// ever runs (AuthMiddleware itself aborts the chain).
func TestPlatformOwnerOnly_RejectsUnauthenticated(t *testing.T) {
	srv := newPlatformOwnerTestServer(t)

	t.Run("no_authorization_header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/platform-only/tenants", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("got status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("garbage_token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/platform-only/tenants", nil)
		req.Header.Set("Authorization", "Bearer not-a-real-jwt")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("got status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("expired_token", func(t *testing.T) {
		expired, err := utils.GenerateJWT("user-platform_owner", "owner@example.com", "platform_owner", "", -time.Hour)
		if err != nil {
			t.Fatalf("GenerateJWT failed: %v", err)
		}
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/platform-only/tenants", nil)
		req.Header.Set("Authorization", "Bearer "+expired)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("got status %d, want 401 (an expired platform_owner token must still be rejected)", resp.StatusCode)
		}
	})
}

// TestPlatformOwnerOnly_RoleValueCannotBeSpoofedViaHeader confirms the
// role comes only from the verified JWT claim - a client-supplied
// X-Role or similar header must have no effect.
func TestPlatformOwnerOnly_RoleValueCannotBeSpoofedViaHeader(t *testing.T) {
	srv := newPlatformOwnerTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/platform-only/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "admin"))
	req.Header.Set("X-Role", "platform_owner")
	req.Header.Set("X-User-Role", "platform_owner")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a spoofed X-Role header granted access: got status %d, want 403", resp.StatusCode)
	}
}

// newAdminGroupTestServer mirrors the real /admin route group's gate in
// api/routes/routes.go: AuthMiddleware() + RoleMiddleware("admin",
// "super_admin", "platform_owner"). AdminOnly() (RoleMiddleware("admin")
// alone) used to gate this group and silently locked out every real
// super_admin and platform_owner account - every AdminHandler method
// behind it is already role-agnostic once past the gate (it scopes or
// bypasses purely on schoolID/isPlatformOwner from the verified JWT), so
// the gate itself was the only thing out of step with the rest of the
// codebase's admin-ish route groups (academic/questions/subscriptions/exams).
func newAdminGroupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/admin")
	admin.Use(AuthMiddleware(), RoleMiddleware("admin", "super_admin", "platform_owner"))
	admin.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestAdminGroup_AllowsAdminSuperAdminAndPlatformOwner(t *testing.T) {
	srv := newAdminGroupTestServer(t)

	cases := []struct {
		role       string
		wantStatus int
	}{
		{"admin", http.StatusOK},
		{"super_admin", http.StatusOK},
		{"platform_owner", http.StatusOK},
		{"teacher", http.StatusForbidden},
		{"student", http.StatusForbidden},
		{"parent", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/users", nil)
			req.Header.Set("Authorization", "Bearer "+tokenFor(t, tc.role))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("role=%s: got status %d, want %d", tc.role, resp.StatusCode, tc.wantStatus)
			}
		})
	}
}
