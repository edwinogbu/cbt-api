package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ---- CORSMiddleware: no Access-Control-Allow-Credentials ----
//
// Access-Control-Allow-Origin: * combined with Access-Control-Allow-
// Credentials: true is a textbook CORS misconfiguration: this app
// never sets a cookie anywhere (pure Bearer-token auth), so there is
// no credentialed capability for that header to legitimately grant,
// and browsers already refuse the combination outright per the Fetch
// spec. This test guards against it coming back.
func TestCORSMiddleware_NoAllowCredentialsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if v := w.Header().Get("Access-Control-Allow-Credentials"); v != "" {
		t.Fatalf("expected no Access-Control-Allow-Credentials header alongside a wildcard origin, got %q", v)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %q", v)
	}
}
