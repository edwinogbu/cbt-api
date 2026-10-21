package main

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// newCORSTestServer builds a real HTTP server running gin with the exact
// same corsConfig() the production server uses (see main.go), plus a
// stand-in handler for POST /api/v1/sync/:type - the route the offline
// sync client (cloudSync.ts, lanSync.ts) actually calls. It deliberately
// does not start the DB, Redis, or AI workers: this test is only about
// whether a real HTTP request carrying the sync client's exact headers
// survives the CORS layer, not about sync business logic (already
// covered by internal/sync/service tests).
func newCORSTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cors.New(corsConfig()))
	r.POST("/api/v1/sync/:type", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ACCEPTED"})
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// TestCORSPreflight_AllowsSyncClientHeaders is a regression test for the
// bug found via live checkpoint-logging (see
// CORE-CBT-COMPLETION-REPORT.md §16): the browser's CORS preflight for
// POST /api/v1/sync/:type returned 204, but the response's
// Access-Control-Allow-Headers did not include Idempotency-Key or
// X-Operation-Id - the two custom headers cloudSync.ts/lanSync.ts send
// on every sync request - so the browser silently refused to send the
// real POST. Unlike internal/sync/service's tests, which call
// SyncService.Process directly and never go through an HTTP server, this
// exercises the real CORS middleware over a real HTTP connection, which
// is the only layer that can catch this class of bug.
func TestCORSPreflight_AllowsSyncClientHeaders(t *testing.T) {
	srv := newCORSTestServer(t)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/sync/answer", nil)
	if err != nil {
		t.Fatalf("building preflight request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	// Exactly the header set cloudSync.ts/lanSync.ts send on a real sync
	// POST (Content-Type/Authorization are "simple"/already-allowed
	// headers in the old config; Idempotency-Key and X-Operation-Id are
	// the two that were missing).
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization,idempotency-key,x-operation-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected preflight status 204, got %d", resp.StatusCode)
	}

	allowed := resp.Header.Get("Access-Control-Allow-Headers")
	if allowed == "" {
		t.Fatal("Access-Control-Allow-Headers is empty - preflight grants no headers at all")
	}

	for _, want := range []string{"Idempotency-Key", "X-Operation-Id"} {
		if !headerListContains(allowed, want) {
			t.Errorf("Access-Control-Allow-Headers %q does not include %q - the sync client's "+
				"POST would be silently blocked by the browser even though the preflight itself "+
				"returns 204", allowed, want)
		}
	}
}

// TestCORSPreflight_FullAllowedHeaderSet pins the complete expected
// header list so a future change that removes one of these (including
// the pre-existing four, not just the two this session added) fails
// loudly here instead of surfacing only as a silent browser-side drop
// weeks later.
func TestCORSPreflight_FullAllowedHeaderSet(t *testing.T) {
	srv := newCORSTestServer(t)

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/sync/event", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "origin,content-type,accept,authorization,idempotency-key,x-operation-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}
	defer resp.Body.Close()

	got := splitHeaderList(resp.Header.Get("Access-Control-Allow-Headers"))
	want := []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key", "X-Operation-Id"}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Access-Control-Allow-Headers mismatch:\n got:  %v\n want: %v", got, want)
	}
}

// TestSyncPOST_SucceedsWithClientHeaders is the other half of the real
// bug: it was never that the server actively rejected the POST (it never
// even saw it - the browser blocked it after a bad preflight). This
// confirms the real route handler plus the real CORS middleware don't
// themselves reject a request carrying the exact headers the sync client
// sends, once it does arrive.
func TestSyncPOST_SucceedsWithClientHeaders(t *testing.T) {
	srv := newCORSTestServer(t)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/sync/answer", strings.NewReader(`{}`))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "test-idempotency-key")
	req.Header.Set("X-Operation-Id", "test-operation-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func splitHeaderList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func headerListContains(list, header string) bool {
	for _, h := range splitHeaderList(list) {
		if strings.EqualFold(h, header) {
			return true
		}
	}
	return false
}

// ---- ClientIP spoofing: Gin's unsafe default trusted-proxy config ----
//
// Without an explicit SetTrustedProxies call, Gin's default engine
// trusts 0.0.0.0/0 and ::/0 as proxies (with ForwardedByClientIP: true),
// so c.ClientIP() blindly honors a client-supplied X-Forwarded-For
// header from ANY remote address - a real client could set it to
// whatever IP it likes, spoofing session audit logs, exam attempt IP
// tracking, and any IP-based rate limiting built on top of it.

func TestNewRouter_ClientIPIsNotSpoofableViaForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, err := newRouter()
	if err != nil {
		t.Fatalf("newRouter failed: %v", err)
	}

	var observedIP string
	r.GET("/whoami", func(c *gin.Context) {
		observedIP = c.ClientIP()
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "127.0.0.1:54321"

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if observedIP == "1.2.3.4" {
		t.Fatalf("ClientIP() trusted a client-supplied X-Forwarded-For header: got %q, want the real TCP peer (127.0.0.1)", observedIP)
	}
	if observedIP != "127.0.0.1" {
		t.Fatalf("expected ClientIP() to return the real TCP peer 127.0.0.1, got %q", observedIP)
	}
}
