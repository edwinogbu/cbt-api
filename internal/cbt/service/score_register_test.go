package service

import (
	"context"
	"testing"
	"time"

	"cbt-api/internal/cbt/dto"
	"gorm.io/gorm"
)

// seedRegisterExam seeds an Exam fully scoped to class/term/session -
// seedTermlyExam (exam_service_test.go) deliberately leaves those
// empty since report-card tests don't need them, but the score
// register's own resolveScoreComponents matches on all four.
func seedRegisterExam(t *testing.T, db *gorm.DB, id, subjectID, classID, termID, sessionID, examType string, totalMarks int, theoryMaxMarks *int) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO exams (id, subject_id, class_id, term_id, session_id, exam_type, duration_minutes, total_marks, theory_max_marks, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 60, ?, ?, ?, ?)",
		id, subjectID, classID, termID, sessionID, examType, totalMarks, theoryMaxMarks, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed register exam: %v", err)
	}
}

func seedRegisterClass(t *testing.T, db *gorm.DB, classID, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO classes (id, school_id, class_level_id, class_arm_id) VALUES (?, ?, 'level-1', 'arm-1')", classID, schoolID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedRegisterStudent(t *testing.T, db *gorm.DB, studentID, classID, schoolID, admissionNo, firstName string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES (?, ?, 'Test', 'student')", studentID, firstName).Error; err != nil {
		t.Fatalf("failed to seed student user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO students (id, user_id, school_id, class_id, admission_no, is_active) VALUES (?, ?, ?, ?, ?, 1)",
		studentID, studentID, schoolID, classID, admissionNo,
	).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func TestGetScoreRegister_NoExamsYetReturnsEmptyComponents(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")

	resp, err := svc.GetScoreRegister(context.Background(), "school-1", false, "class-1", "subj-1", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetScoreRegister failed: %v", err)
	}
	if len(resp.Components) != 0 {
		t.Errorf("expected 0 components before any Exam is created for this scope, got %d", len(resp.Components))
	}
	if len(resp.Rows) != 1 || resp.Rows[0].StudentID != "student-1" {
		t.Fatalf("expected the enrolled student to still appear with an empty row, got %+v", resp.Rows)
	}
	if resp.SubmissionStatus != "draft" {
		t.Errorf("SubmissionStatus = %q, want draft (no SubjectResultSubmission row yet)", resp.SubmissionStatus)
	}
}

func TestSaveScoreRegister_EntersAndClearsAPlainComponent(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-ca-1", "subj-1", "class-1", "term-1", "session-1", "continuous_assessment", 20, nil)

	score := 18
	saveResp, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &score}},
	})
	if err != nil {
		t.Fatalf("SaveScoreRegister failed: %v", err)
	}
	if len(saveResp.Saved) != 1 || len(saveResp.Failed) != 0 {
		t.Fatalf("expected 1 saved, 0 failed, got saved=%d failed=%d (%+v)", len(saveResp.Saved), len(saveResp.Failed), saveResp.Failed)
	}

	reg, err := svc.GetScoreRegister(context.Background(), "school-1", false, "class-1", "subj-1", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetScoreRegister failed: %v", err)
	}
	if len(reg.Components) != 1 || reg.Components[0].Code != "continuous_assessment" {
		t.Fatalf("expected 1 'continuous_assessment' component, got %+v", reg.Components)
	}
	cell := reg.Rows[0].Cells[0]
	if cell.Score == nil || *cell.Score != 18 {
		t.Fatalf("expected the saved score 18 to round-trip, got %v", cell.Score)
	}
	if cell.ReadOnly {
		t.Error("a plain manually-scored component (no CBT questions) must not be ReadOnly")
	}

	// Clearing (nil score) must delete the row entirely - "not
	// recorded", not a stored 0.
	if _, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: nil}},
	}); err != nil {
		t.Fatalf("SaveScoreRegister (clear) failed: %v", err)
	}
	reg2, _ := svc.GetScoreRegister(context.Background(), "school-1", false, "class-1", "subj-1", "term-1", "session-1")
	if reg2.Rows[0].Cells[0].Score != nil {
		t.Errorf("expected the cleared cell's score to be nil, got %v", *reg2.Rows[0].Cells[0].Score)
	}
}

func TestSaveScoreRegister_RejectsScoreAboveMax(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-ca-1", "subj-1", "class-1", "term-1", "session-1", "continuous_assessment", 20, nil)

	tooHigh := 25
	resp, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &tooHigh}},
	})
	if err != nil {
		t.Fatalf("SaveScoreRegister failed: %v", err)
	}
	if len(resp.Saved) != 0 || len(resp.Failed) != 1 {
		t.Fatalf("expected the over-max score to be rejected (0 saved, 1 failed), got saved=%d failed=%d", len(resp.Saved), len(resp.Failed))
	}
}

// TestSaveScoreRegister_RejectsEditingACBTComponent proves the backend
// enforces read-only CBT cells itself - "hiding a button is not
// sufficient authorization" (the spec's own words) - by seeding an
// exam_questions row for the exam (making resolveScoreComponents mark
// it ReadOnly) and confirming a direct save attempt is still rejected.
func TestSaveScoreRegister_RejectsEditingACBTComponent(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-main-1", "subj-1", "class-1", "term-1", "session-1", "main_exam", 40, nil)
	if err := db.Exec("INSERT INTO exam_questions (id, exam_id, question_id) VALUES ('eq-1', 'exam-main-1', 'q-1')").Error; err != nil {
		t.Fatalf("failed to seed exam_questions: %v", err)
	}

	score := 30
	_, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "main_exam", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &score}},
	})
	if err == nil {
		t.Fatal("expected SaveScoreRegister to reject a direct edit of a CBT-sourced (ReadOnly) component, got nil error")
	}
}

// TestScoreRegister_CombinedCBTTheoryExam proves the register shows
// the CBT objective cell as ReadOnly and a separate, editable theory
// cell, and that saving the theory score combines correctly into the
// row's authoritative total - the exact 28/40 + 22/30 = 50/70 example
// from the spec, now exercised through the register's own save path
// rather than directly through buildSubjectTermlyResult.
func TestScoreRegister_CombinedCBTTheoryExam(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	theoryMax := 30
	seedRegisterExam(t, db, "exam-main-1", "subj-1", "class-1", "term-1", "session-1", "main_exam", 40, &theoryMax)
	if err := db.Exec("INSERT INTO exam_questions (id, exam_id, question_id) VALUES ('eq-1', 'exam-main-1', 'q-1')").Error; err != nil {
		t.Fatalf("failed to seed exam_questions: %v", err)
	}
	// The CBT objective score, as it would be after a real graded attempt.
	if err := db.Exec("INSERT INTO results (id, student_id, exam_id, attempt_id, total_score, percentage, grade, fully_graded) VALUES ('res-1','student-1','exam-main-1','att-1',28,70,'A',1)").Error; err != nil {
		t.Fatalf("failed to seed CBT result: %v", err)
	}

	reg, err := svc.GetScoreRegister(context.Background(), "school-1", false, "class-1", "subj-1", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetScoreRegister failed: %v", err)
	}
	if len(reg.Components) != 2 {
		t.Fatalf("expected 2 components (objective + theory), got %d: %+v", len(reg.Components), reg.Components)
	}
	var objCell, theoryCell *dto.ScoreCell
	for i, c := range reg.Components {
		if c.Code == "main_exam" {
			objCell = &reg.Rows[0].Cells[i]
		}
		if c.Code == "main_exam_theory" {
			theoryCell = &reg.Rows[0].Cells[i]
		}
	}
	if objCell == nil || !objCell.ReadOnly || objCell.Score == nil || *objCell.Score != 28 {
		t.Fatalf("objective cell wrong: %+v", objCell)
	}
	if theoryCell == nil || theoryCell.ReadOnly || theoryCell.Score != nil {
		t.Fatalf("theory cell should be editable and still unset: %+v", theoryCell)
	}
	// Before theory is entered, the row must show as incomplete with no total.
	if reg.Rows[0].IsComplete {
		t.Error("expected IsComplete=false before the theory score is entered")
	}

	theoryScore := 22
	if _, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "main_exam_theory", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &theoryScore}},
	}); err != nil {
		t.Fatalf("SaveScoreRegister(theory) failed: %v", err)
	}

	reg2, err := svc.GetScoreRegister(context.Background(), "school-1", false, "class-1", "subj-1", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetScoreRegister (after theory save) failed: %v", err)
	}
	row := reg2.Rows[0]
	if row.Total == nil || *row.Total != 50 || row.MaxTotal != 70 {
		t.Fatalf("Total/MaxTotal = %v/%d, want 50/70", row.Total, row.MaxTotal)
	}
	if !row.IsComplete {
		t.Error("expected IsComplete=true once the theory score is entered")
	}
}

// TestSaveScoreRegister_LockedSubmissionRequiresCorrectionReason proves
// SubjectResultSubmission.IsLocked actually gates ordinary saves once a
// reviewer has acted, and that a correction_reason unblocks it.
func TestSaveScoreRegister_LockedSubmissionRequiresCorrectionReason(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-ca-1", "subj-1", "class-1", "term-1", "session-1", "continuous_assessment", 20, nil)
	if err := db.Exec(
		"INSERT INTO subject_result_submissions (id, school_id, class_id, subject_id, term_id, session_id, status, version, teacher_id) VALUES ('sub-1','school-1','class-1','subj-1','term-1','session-1','approved',1,'teacher-1')",
	).Error; err != nil {
		t.Fatalf("failed to seed an approved submission: %v", err)
	}

	score := 15
	_, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &score}},
	})
	if err == nil {
		t.Fatal("expected a save with no correction_reason to be rejected once the submission is approved (locked)")
	}

	_, err = svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", CorrectionReason: "Transcription error found during review",
		Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &score}},
	})
	if err != nil {
		t.Fatalf("expected a save WITH correction_reason to succeed on a locked submission, got error: %v", err)
	}
}
