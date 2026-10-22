package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// These tests exercise ExportClassBroadSheet/ExportSubjectBroadSheet/
// ExportSchoolBroadSheet end to end through the real Get*BroadSheet
// builders (same seed patterns as broadsheet_views_test.go), proving
// the export path renders the same data those functions return rather
// than a second, independently computed figure.

func TestExportClassBroadSheet_AllFormats(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Greenfield Academy", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	seedReportCardStudent(t, db, "student-1", "user-1", schoolID, classID)
	seedReportCardExam(t, db, "exam-math-1", "subj-math", classID, termID, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", "exam-math-1", 80, true)

	ctx := context.Background()

	t.Run("pdf", func(t *testing.T) {
		data, contentType, filename, err := svc.ExportClassBroadSheet(ctx, schoolID, false, classID, "termly", termID, sessionID, "pdf")
		if err != nil {
			t.Fatalf("ExportClassBroadSheet(pdf) failed: %v", err)
		}
		if contentType != "application/pdf" {
			t.Errorf("contentType = %q, want application/pdf", contentType)
		}
		if !strings.HasSuffix(filename, ".pdf") {
			t.Errorf("filename = %q, want a .pdf suffix", filename)
		}
		if !bytes.HasPrefix(data, []byte("%PDF")) {
			t.Fatalf("output does not start with the PDF magic header")
		}
	})

	t.Run("csv", func(t *testing.T) {
		data, contentType, filename, err := svc.ExportClassBroadSheet(ctx, schoolID, false, classID, "termly", termID, sessionID, "csv")
		if err != nil {
			t.Fatalf("ExportClassBroadSheet(csv) failed: %v", err)
		}
		if contentType != "text/csv" {
			t.Errorf("contentType = %q, want text/csv", contentType)
		}
		if !strings.HasSuffix(filename, ".csv") {
			t.Errorf("filename = %q, want a .csv suffix", filename)
		}
		got := string(data)
		if !strings.Contains(got, "Mathematics") {
			t.Errorf("CSV does not contain subject header %q:\n%s", "Mathematics", got)
		}
		if !strings.Contains(got, "80 (A)") {
			t.Errorf("CSV does not contain the student's actual score/grade %q:\n%s", "80 (A)", got)
		}
	})

	t.Run("excel", func(t *testing.T) {
		data, contentType, filename, err := svc.ExportClassBroadSheet(ctx, schoolID, false, classID, "termly", termID, sessionID, "excel")
		if err != nil {
			t.Fatalf("ExportClassBroadSheet(excel) failed: %v", err)
		}
		if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Errorf("unexpected contentType %q", contentType)
		}
		if !strings.HasSuffix(filename, ".xlsx") {
			t.Errorf("filename = %q, want a .xlsx suffix", filename)
		}
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("generated Excel file could not be parsed back: %v", err)
		}
		rows, err := f.GetRows("Class Broad Sheet")
		if err != nil {
			t.Fatalf("GetRows failed: %v", err)
		}
		if len(rows) != 2 { // header + 1 student
			t.Fatalf("expected 2 rows (header + 1 student), got %d: %v", len(rows), rows)
		}
	})

	t.Run("unsupported format rejected", func(t *testing.T) {
		if _, _, _, err := svc.ExportClassBroadSheet(ctx, schoolID, false, classID, "termly", termID, sessionID, "docx"); err == nil {
			t.Fatal("expected an error for an unsupported format, got nil")
		}
	})
}

func TestExportSubjectBroadSheet_PDFAndCSV(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("seed teacher failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a1','school-1','teacher-1','subj-math','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-math-c1", "subj-math", "class-1", "term-1", "session-1", "main_exam", 100, nil)
	if err := db.Exec("INSERT INTO results (id, student_id, exam_id, total_score, percentage, grade, fully_graded) VALUES ('r1','student-1','exam-math-c1',90,90,'A',1)").Error; err != nil {
		t.Fatalf("seed result failed: %v", err)
	}

	ctx := context.Background()
	data, contentType, _, err := svc.ExportSubjectBroadSheet(ctx, "school-1", false, "subj-math", "term-1", "session-1", "pdf")
	if err != nil {
		t.Fatalf("ExportSubjectBroadSheet(pdf) failed: %v", err)
	}
	if contentType != "application/pdf" || !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("ExportSubjectBroadSheet(pdf) did not return a valid PDF")
	}

	csvData, _, _, err := svc.ExportSubjectBroadSheet(ctx, "school-1", false, "subj-math", "term-1", "session-1", "csv")
	if err != nil {
		t.Fatalf("ExportSubjectBroadSheet(csv) failed: %v", err)
	}
	if !strings.Contains(string(csvData), "class-1") && !strings.Contains(string(csvData), "Ada Lovelace") {
		t.Errorf("CSV missing expected row content:\n%s", csvData)
	}
}

func TestExportSchoolBroadSheet_PDFAndCSV(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("seed teacher failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a1','school-1','teacher-1','subj-math','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO subject_result_submissions (id, school_id, class_id, subject_id, term_id, session_id, status, version, teacher_id) VALUES ('s1','school-1','class-1','subj-math','term-1','session-1','approved',1,'teacher-1')",
	).Error; err != nil {
		t.Fatalf("seed submission failed: %v", err)
	}

	ctx := context.Background()
	data, contentType, filename, err := svc.ExportSchoolBroadSheet(ctx, "school-1", false, "", "term-1", "session-1", "termly", "pdf")
	if err != nil {
		t.Fatalf("ExportSchoolBroadSheet(pdf) failed: %v", err)
	}
	if contentType != "application/pdf" || !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("ExportSchoolBroadSheet(pdf) did not return a valid PDF")
	}
	if !strings.HasSuffix(filename, ".pdf") {
		t.Errorf("filename = %q, want a .pdf suffix", filename)
	}

	csvData, _, _, err := svc.ExportSchoolBroadSheet(ctx, "school-1", false, "", "term-1", "session-1", "termly", "csv")
	if err != nil {
		t.Fatalf("ExportSchoolBroadSheet(csv) failed: %v", err)
	}
	if !strings.Contains(string(csvData), "Yes") {
		t.Errorf("CSV does not reflect the approved submission as Yes:\n%s", csvData)
	}
}
