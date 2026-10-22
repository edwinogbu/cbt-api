package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"testing"

	"cbt-api/internal/cbt/dto"

	"github.com/google/uuid"
)

func TestApproveReportCard_FreezesSnapshotAndIncrementsVersionOnReapproval(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 72, true)

	req := &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{
			StudentID: "student-1", ReportType: "termly", TermID: termID, SessionID: sessionID,
		},
		Notes: "First review pass",
	}
	resp, err := svc.ApproveReportCard(ctx, schoolID, "admin-1", req)
	if err != nil {
		t.Fatalf("ApproveReportCard failed: %v", err)
	}
	if resp.Status != "approved" {
		t.Errorf("Status = %q, want approved", resp.Status)
	}
	if resp.Version != 1 {
		t.Errorf("Version = %d, want 1 on first approval", resp.Version)
	}
	if resp.ActorID != "admin-1" {
		t.Errorf("ActorID = %q, want admin-1", resp.ActorID)
	}

	// Re-approving the same scope must bump the version, not create a
	// second row (models.ReportCardApproval's own unique index on
	// student/report-type/term/session would reject a second row
	// anyway, but this also checks the service's own find-or-init path).
	resp2, err := svc.ApproveReportCard(ctx, schoolID, "admin-2", req)
	if err != nil {
		t.Fatalf("second ApproveReportCard failed: %v", err)
	}
	if resp2.Version != 2 {
		t.Errorf("Version after re-approval = %d, want 2", resp2.Version)
	}
	if resp2.ActorID != "admin-2" {
		t.Errorf("ActorID after re-approval = %q, want admin-2 (the latest approver)", resp2.ActorID)
	}

	history, err := svc.GetApprovalHistory(ctx, schoolID, "student-1", "termly")
	if err != nil {
		t.Fatalf("GetApprovalHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2 (one audit entry per approval, even though the ReportCardApproval row itself was overwritten in place)", len(history))
	}
}

func TestApproveReportCard_RejectsCrossTenantStudent(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	seedSchool(t, db, "school-1", "School One", "", "")
	seedSchool(t, db, "school-2", "School Two", "", "")
	seedReportCardClass(t, db, "class-1", "school-1", "")
	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, "school-1", "class-1")

	req := &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{
			StudentID: "student-1", ReportType: "termly", TermID: "term-1", SessionID: "session-1",
		},
	}
	// Caller is authenticated as school-2's admin, approving a
	// school-1 student - must be refused, not silently scoped away.
	if _, err := svc.ApproveReportCard(ctx, "school-2", "admin-1", req); err == nil {
		t.Fatal("expected an error approving another school's student, got nil")
	}
}

func TestApproveReportCard_PromotionNormalizesTermIDToEmpty(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 60, true)

	req := &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{
			StudentID: "student-1", ReportType: "promotion",
			// A caller mistakenly (or maliciously) sending a TermID for
			// a session-scoped promotion report must not be able to
			// create a second, distinct approval row for the same
			// promotion report under a different TermID value.
			TermID: "some-stray-term-id", SessionID: sessionID,
		},
	}
	resp, err := svc.ApproveReportCard(ctx, schoolID, "admin-1", req)
	if err != nil {
		t.Fatalf("ApproveReportCard failed: %v", err)
	}
	if resp.TermID != "" {
		t.Errorf("TermID = %q, want empty string for a normalized promotion approval", resp.TermID)
	}

	status, err := svc.GetApprovalStatus(ctx, schoolID, false, dto.ReportCardScope{
		StudentID: "student-1", ReportType: "promotion", SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("GetApprovalStatus failed: %v", err)
	}
	if status.Status != "approved" {
		t.Errorf("GetApprovalStatus (TermID omitted) = %q, want approved - must resolve to the same normalized row", status.Status)
	}
}

func TestRejectReportCard_RequiresNotes(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	seedSchool(t, db, "school-1", "Test School", "", "")
	seedReportCardClass(t, db, "class-1", "school-1", "")
	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, "school-1", "class-1")

	req := &dto.RejectReportCardRequest{
		ReportCardScope: dto.ReportCardScope{
			StudentID: "student-1", ReportType: "termly", TermID: "term-1", SessionID: "session-1",
		},
		// Notes deliberately left empty.
	}
	if _, err := svc.RejectReportCard(ctx, "school-1", "admin-1", req); err == nil {
		t.Fatal("expected an error rejecting with no notes, got nil")
	}
}

func TestGetApprovalStatus_NotSubmittedWhenNoRow(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	seedSchool(t, db, "school-1", "Test School", "", "")
	seedReportCardClass(t, db, "class-1", "school-1", "")
	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, "school-1", "class-1")

	status, err := svc.GetApprovalStatus(ctx, "school-1", false, dto.ReportCardScope{
		StudentID: "student-1", ReportType: "termly", TermID: "term-1", SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("GetApprovalStatus failed: %v", err)
	}
	if status.Status != "not_submitted" {
		t.Errorf("Status = %q, want not_submitted for a report card never approved or rejected", status.Status)
	}
	if status.Version != 0 {
		t.Errorf("Version = %d, want 0 for a never-approved report card", status.Version)
	}
}

func TestBatchApproveReportCards_ApprovesWholeClassRoster(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	for i, admission := range []string{"student-1", "student-2"} {
		userID := uuid.NewString()
		seedReportCardStudent(t, db, admission, userID, schoolID, classID)
		seedReportCardResult(t, db, admission, examID, 50+i*10, true)
	}

	resp, err := svc.BatchApproveReportCards(ctx, schoolID, "admin-1", false, &dto.BatchApproveRequest{
		ClassID: classID, ReportType: "termly", TermID: termID, SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("BatchApproveReportCards failed: %v", err)
	}
	if len(resp.ApprovedStudentIDs) != 2 {
		t.Fatalf("ApprovedStudentIDs = %v, want 2 students approved", resp.ApprovedStudentIDs)
	}
	if len(resp.Failed) != 0 {
		t.Errorf("Failed = %v, want none", resp.Failed)
	}

	for _, studentID := range []string{"student-1", "student-2"} {
		status, err := svc.GetApprovalStatus(ctx, schoolID, false, dto.ReportCardScope{
			StudentID: studentID, ReportType: "termly", TermID: termID, SessionID: sessionID,
		})
		if err != nil || status.Status != "approved" {
			t.Errorf("student %s approval status = %+v, err=%v, want approved", studentID, status, err)
		}
	}
}

func TestGetClassReportCardSummary_IncludesFiguresAndApprovalStatus(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 80, true)

	if _, err := svc.ApproveReportCard(ctx, schoolID, "admin-1", &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{StudentID: "student-1", ReportType: "termly", TermID: termID, SessionID: sessionID},
	}); err != nil {
		t.Fatalf("ApproveReportCard failed: %v", err)
	}

	summary, err := svc.GetClassReportCardSummary(ctx, schoolID, false, classID, "termly", termID, sessionID)
	if err != nil {
		t.Fatalf("GetClassReportCardSummary failed: %v", err)
	}
	if len(summary.Rows) != 1 {
		t.Fatalf("Rows length = %d, want 1", len(summary.Rows))
	}
	row := summary.Rows[0]
	if row.StudentID != "student-1" {
		t.Errorf("StudentID = %q, want student-1", row.StudentID)
	}
	if row.ApprovalStatus != "approved" || row.ApprovalVersion != 1 {
		t.Errorf("approval status/version = %q/%d, want approved/1", row.ApprovalStatus, row.ApprovalVersion)
	}
	if row.OverallPercentage != 80 {
		t.Errorf("OverallPercentage = %.1f, want 80", row.OverallPercentage)
	}
}

func TestGenerateClassReportCardsZIP_ProducesOnePDFPerStudent(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	for i, admission := range []string{"student-1", "student-2"} {
		userID := uuid.NewString()
		seedReportCardStudent(t, db, admission, userID, schoolID, classID)
		seedReportCardResult(t, db, admission, examID, 60+i*5, true)
	}

	data, filename, skipped, err := svc.GenerateClassReportCardsZIP(ctx, schoolID, false, classID, "termly", termID, sessionID)
	if err != nil {
		t.Fatalf("GenerateClassReportCardsZIP failed: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	if filename == "" {
		t.Error("filename is empty")
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader failed: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("zip entries = %d, want 2 (one PDF per student)", len(zr.File))
	}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open zip entry %s: %v", f.Name, err)
		}
		buf := make([]byte, 4)
		if _, err := rc.Read(buf); err != nil {
			t.Fatalf("failed to read zip entry %s: %v", f.Name, err)
		}
		rc.Close()
		if string(buf) != "%PDF" {
			t.Errorf("zip entry %s does not start with a PDF magic header, got %q", f.Name, buf)
		}
	}
}

func TestExportClassReportCards_CSVAndExcel(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 65, true)

	csvData, contentType, filename, err := svc.ExportClassReportCards(ctx, schoolID, false, classID, "termly", termID, sessionID, "csv")
	if err != nil {
		t.Fatalf("ExportClassReportCards (csv) failed: %v", err)
	}
	if contentType != "text/csv" {
		t.Errorf("contentType = %q, want text/csv", contentType)
	}
	if filename == "" {
		t.Error("filename is empty")
	}
	rows, err := csv.NewReader(bytes.NewReader(csvData)).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse exported CSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("CSV rows = %d, want 2 (header + 1 student)", len(rows))
	}
	// seedReportCardStudent does not set admission_no (production's
	// Student.AdmissionNo is populated by the real academic-admissions
	// flow, out of scope for this fixture) - check the student name
	// column instead, which seedReportCardStudent does set.
	if rows[1][1] != "Test Student" {
		t.Errorf("CSV data row student name = %q, want %q", rows[1][1], "Test Student")
	}

	excelData, excelContentType, _, err := svc.ExportClassReportCards(ctx, schoolID, false, classID, "termly", termID, sessionID, "excel")
	if err != nil {
		t.Fatalf("ExportClassReportCards (excel) failed: %v", err)
	}
	if excelContentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("excel contentType = %q, unexpected", excelContentType)
	}
	if len(excelData) == 0 {
		t.Error("excel export produced no bytes")
	}
}
