package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// These cover DownloadTermlyReportCardForUser's parameter/auth
// validation, which runs before any service/DB call - the happy path
// (service call -> pdf.GenerateTermlyReportCard -> response) is already
// covered end-to-end by internal/cbt/service's GetReportCardForUser
// tests and internal/cbt/pdf's round-trip PDF-content tests; this
// handler's own logic beyond param validation is a few lines of direct
// glue between those two, already-tested layers.

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDownloadTermlyReportCardForUser_MissingTermID_Rejected(t *testing.T) {
	h := &ExamHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/student/exams/report-card/pdf?sessionId=session-1", nil)

	h.DownloadTermlyReportCardForUser(c)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400 for a missing termId", w.Code)
	}
	if bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF")) {
		t.Error("a request missing termId must not produce PDF output")
	}
}

func TestDownloadTermlyReportCardForUser_MissingSessionID_Rejected(t *testing.T) {
	h := &ExamHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/student/exams/report-card/pdf?termId=term-1", nil)

	h.DownloadTermlyReportCardForUser(c)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400 for a missing sessionId", w.Code)
	}
}

func TestDownloadTermlyReportCardForUser_Unauthenticated_Rejected(t *testing.T) {
	h := &ExamHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/student/exams/report-card/pdf?termId=term-1&sessionId=session-1", nil)
	// Deliberately no "user_id" set in the gin context - simulates a
	// request that reached this handler without AuthMiddleware ever
	// populating it.

	h.DownloadTermlyReportCardForUser(c)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 when no authenticated user_id is present", w.Code)
	}
}

func TestSanitizeFilename_StripsPathSeparatorsAndQuotes(t *testing.T) {
	cases := map[string]string{
		"report-card-First Term-2025/2026.pdf":  "report-card-First Term-2025-2026.pdf",
		`weird"name.pdf`:                        "weirdname.pdf",
		"back\\slash.pdf":                       "back-slash.pdf",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
