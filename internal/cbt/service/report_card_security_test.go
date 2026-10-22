package service

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"github.com/google/uuid"
)

func TestWriteReportCardAudit_ContentHashVerifiesClean(t *testing.T) {
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
	seedReportCardResult(t, db, "student-1", examID, 70, true)

	if _, err := svc.ApproveReportCard(ctx, schoolID, "admin-1", &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{StudentID: "student-1", ReportType: "termly", TermID: termID, SessionID: sessionID},
		Notes:           "looks good",
	}); err != nil {
		t.Fatalf("ApproveReportCard failed: %v", err)
	}

	report, err := svc.VerifyReportCardAuditIntegrity(ctx, schoolID)
	if err != nil {
		t.Fatalf("VerifyReportCardAuditIntegrity failed: %v", err)
	}
	if report.Checked != 1 {
		t.Fatalf("Checked = %d, want 1", report.Checked)
	}
	if len(report.Issues) != 0 {
		t.Errorf("Issues = %v, want none for an untouched audit log", report.Issues)
	}
}

func TestVerifyReportCardAuditIntegrity_DetectsDirectDBTamper(t *testing.T) {
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
	seedReportCardResult(t, db, "student-1", examID, 70, true)

	if _, err := svc.ApproveReportCard(ctx, schoolID, "admin-1", &dto.ApproveReportCardRequest{
		ReportCardScope: dto.ReportCardScope{StudentID: "student-1", ReportType: "termly", TermID: termID, SessionID: sessionID},
	}); err != nil {
		t.Fatalf("ApproveReportCard failed: %v", err)
	}

	// Simulate tampering: a direct DB edit of the stored audit entry's
	// Notes, bypassing the approve/reject/batch-approve API entirely -
	// the only legitimate writer of this table.
	var entry models.ReportCardAuditLog
	if err := db.Where("school_id = ?", schoolID).First(&entry).Error; err != nil {
		t.Fatalf("failed to load seeded audit entry: %v", err)
	}
	if err := db.Model(&models.ReportCardAuditLog{}).Where("id = ?", entry.ID).Update("notes", "tampered notes - approved under duress").Error; err != nil {
		t.Fatalf("failed to simulate tamper: %v", err)
	}

	report, err := svc.VerifyReportCardAuditIntegrity(ctx, schoolID)
	if err != nil {
		t.Fatalf("VerifyReportCardAuditIntegrity failed: %v", err)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("Issues = %v, want exactly 1 flagged tampered entry", report.Issues)
	}
	if report.Issues[0].ID != entry.ID {
		t.Errorf("flagged issue ID = %q, want %q", report.Issues[0].ID, entry.ID)
	}
}

func TestBatchApproveReportCards_RejectsOversizedExplicitStudentList(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	seedSchool(t, db, "school-1", "Test School", "", "")
	seedReportCardClass(t, db, "class-1", "school-1", "")

	studentIDs := make([]string, 1001)
	for i := range studentIDs {
		studentIDs[i] = uuid.NewString()
	}

	_, err := svc.BatchApproveReportCards(ctx, "school-1", "admin-1", false, &dto.BatchApproveRequest{
		ClassID: "class-1", ReportType: "termly", TermID: "term-1", SessionID: "session-1",
		StudentIDs: studentIDs,
	})
	if err == nil {
		t.Fatal("expected an error for a batch exceeding the max explicit student_ids size, got nil")
	}
	if !strings.Contains(err.Error(), "too many student_ids") {
		t.Errorf("error = %q, want it to mention the batch-size limit", err.Error())
	}
}

func TestExportClassReportCards_SanitizesFormulaInjectionInCSV(t *testing.T) {
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
	// seedReportCardStudent hardcodes the user's first/last name as
	// "Test"/"Student" - directly craft a student row with a formula-
	// injection-shaped admission number instead, since that is the field
	// this test needs to control.
	if err := db.Exec("INSERT INTO students (id, user_id, school_id, class_id, admission_no) VALUES (?, ?, ?, ?, ?)",
		"student-1", userID, schoolID, classID, "=cmd|'/c calc'!A1").Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES (?, ?, ?, 'student')", userID, "Test", "Student").Error; err != nil {
		t.Fatalf("failed to seed student's user: %v", err)
	}
	seedReportCardResult(t, db, "student-1", examID, 70, true)

	csvData, _, _, err := svc.ExportClassReportCards(ctx, schoolID, false, classID, "termly", termID, sessionID, "csv")
	if err != nil {
		t.Fatalf("ExportClassReportCards failed: %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(csvData))).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse exported CSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("CSV rows = %d, want 2", len(rows))
	}
	admissionCell := rows[1][0]
	if !strings.HasPrefix(admissionCell, "'") {
		t.Errorf("admission number cell = %q, want it prefixed with a leading apostrophe to defuse the formula", admissionCell)
	}
	if !strings.Contains(admissionCell, "=cmd") {
		t.Errorf("admission number cell = %q, want the original value preserved after the apostrophe", admissionCell)
	}
}
