package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"
)

// ---- Assessment settings CRUD (UpsertAssessmentScheme/GradingScale) ----

func TestUpsertAssessmentScheme_RejectsWeightsNotSummingTo100(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	req := &dto.AssessmentSchemeRequest{
		Components: []dto.AssessmentComponentInput{
			{Code: "ca", Name: "CA", Weight: 30, ExamTypes: []string{"continuous_assessment"}},
			{Code: "exam", Name: "Exam", Weight: 60, ExamTypes: []string{"main_exam"}}, // 90, not 100
		},
	}
	if _, err := svc.UpsertAssessmentScheme(context.Background(), "school-1", false, "", "admin-1", req); err == nil {
		t.Fatal("expected an error for components whose weights do not sum to 100, got nil")
	}
}

func TestUpsertAssessmentScheme_RejectsExamTypeClaimedByTwoComponents(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	req := &dto.AssessmentSchemeRequest{
		Components: []dto.AssessmentComponentInput{
			{Code: "ca", Name: "CA", Weight: 50, ExamTypes: []string{"continuous_assessment"}},
			{Code: "ca2", Name: "CA2", Weight: 50, ExamTypes: []string{"continuous_assessment"}},
		},
	}
	if _, err := svc.UpsertAssessmentScheme(context.Background(), "school-1", false, "", "admin-1", req); err == nil {
		t.Fatal("expected an error for two components claiming the same exam type, got nil")
	}
}

// TestUpsertAssessmentScheme_SecondCallReplacesNotDuplicates proves the
// upsert is keyed by scope: creating a scheme for the same (empty
// class-level, empty subject) scope twice must update the one row, not
// create a second - otherwise grading.ResolveScheme's unique-index-
// backed lookup would eventually find two "school-wide" schemes.
func TestUpsertAssessmentScheme_SecondCallReplacesNotDuplicates(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	first := &dto.AssessmentSchemeRequest{Components: []dto.AssessmentComponentInput{
		{Code: "ca", Name: "CA", Weight: 30, ExamTypes: []string{"continuous_assessment"}},
		{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: []string{"main_exam"}},
	}}
	resp1, err := svc.UpsertAssessmentScheme(context.Background(), "school-1", false, "", "admin-1", first)
	if err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}

	second := &dto.AssessmentSchemeRequest{Components: []dto.AssessmentComponentInput{
		{Code: "ca", Name: "CA", Weight: 40, ExamTypes: []string{"continuous_assessment"}},
		{Code: "exam", Name: "Exam", Weight: 60, ExamTypes: []string{"main_exam"}},
	}}
	resp2, err := svc.UpsertAssessmentScheme(context.Background(), "school-1", false, "", "admin-1", second)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if resp1.ID != resp2.ID {
		t.Fatalf("second upsert created a new scheme (ID %s) instead of replacing the first (ID %s)", resp2.ID, resp1.ID)
	}

	list, err := svc.ListAssessmentSchemes(context.Background(), "school-1", false, "", "")
	if err != nil {
		t.Fatalf("ListAssessmentSchemes failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 scheme after two upserts to the same scope, got %d", len(list))
	}
	if list[0].Components[0].Weight != 40 && list[0].Components[1].Weight != 40 {
		t.Errorf("expected the second upsert's weights (40/60) to have replaced the first (30/70)")
	}
}

func TestUpsertGradingScale_RequiresZeroBand(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	req := &dto.GradingScaleRequest{
		Name: "Custom Scale", PassMark: 50,
		Bands: []dto.GradingScaleBandInput{
			{Grade: "A", MinPercent: 70},
			{Grade: "F", MinPercent: 40}, // no band starting at 0
		},
	}
	if _, err := svc.UpsertGradingScale(context.Background(), "school-1", false, "", "admin-1", req); err == nil {
		t.Fatal("expected an error for a band set with no 0% band, got nil")
	}
}

func TestUpsertGradingScale_ValidRoundTrips(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	req := &dto.GradingScaleRequest{
		Name: "WAEC Style", PassMark: 50,
		Bands: []dto.GradingScaleBandInput{
			{Grade: "A", MinPercent: 75, GradePoint: 5},
			{Grade: "F", MinPercent: 0, GradePoint: 0},
		},
	}
	resp, err := svc.UpsertGradingScale(context.Background(), "school-1", false, "", "admin-1", req)
	if err != nil {
		t.Fatalf("UpsertGradingScale failed: %v", err)
	}
	if len(resp.Bands) != 2 {
		t.Fatalf("expected 2 bands back, got %d", len(resp.Bands))
	}

	// Confirm it's actually reachable through the real grading engine's
	// own resolution path, not just stored and ignored.
	resolved := svc.gradingSvc.ResolveScale(context.Background(), "school-1", "")
	if resolved.Source != "school" {
		t.Fatalf("grading.Service.ResolveScale did not pick up the newly configured scale, got Source=%s", resolved.Source)
	}
}

// ---- Teacher subject assignments ----

func TestCreateAndListMySubjectAssignments(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	if err := db.Exec("INSERT INTO classes (id, school_id, class_level_id, class_arm_id) VALUES ('class-1','school-1','level-1','arm-1')").Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("failed to seed teacher: %v", err)
	}

	req := &dto.CreateTeacherSubjectAssignmentRequest{
		TeacherID: "teacher-1", SubjectID: "subj-1", ClassID: "class-1", SessionID: "session-1",
	}
	if _, err := svc.CreateTeacherSubjectAssignment(context.Background(), "school-1", false, req); err != nil {
		t.Fatalf("CreateTeacherSubjectAssignment failed: %v", err)
	}

	mine, err := svc.GetMySubjectAssignments(context.Background(), "teacher-1", "session-1")
	if err != nil {
		t.Fatalf("GetMySubjectAssignments failed: %v", err)
	}
	if len(mine) != 1 {
		t.Fatalf("expected 1 assignment for teacher-1, got %d", len(mine))
	}
	if mine[0].SubjectName != "Mathematics" || mine[0].ClassID != "class-1" {
		t.Errorf("unexpected assignment contents: %+v", mine[0])
	}

	// A different teacher must see none of this.
	otherTeacherAssignments, err := svc.GetMySubjectAssignments(context.Background(), "teacher-2", "session-1")
	if err != nil {
		t.Fatalf("GetMySubjectAssignments(teacher-2) failed: %v", err)
	}
	if len(otherTeacherAssignments) != 0 {
		t.Errorf("expected teacher-2 to see 0 assignments, got %d", len(otherTeacherAssignments))
	}
}

// ---- Theory-score combining (models.Exam.TheoryMaxMarks / models.Result.TheoryScore) ----

func TestBuildSubjectTermlyResult_TheoryScorePendingKeepsExamIncomplete(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	if err := db.Exec(
		"INSERT INTO exams (id, subject_id, class_id, term_id, session_id, exam_type, duration_minutes, total_marks, theory_max_marks) VALUES ('exam-main-1','subj-1','class-1','term-1','session-1','main_exam',60,40,30)",
	).Error; err != nil {
		t.Fatalf("failed to seed exam with theory portion: %v", err)
	}

	// CBT portion graded (28/40), theory not yet entered.
	results := []models.Result{{ExamID: "exam-main-1", TotalScore: 28, FullyGraded: true, TheoryScore: nil}}

	got, err := svc.buildSubjectTermlyResult(context.Background(), "student-1", "subj-1", "school-1", "level-1", results)
	if err != nil {
		t.Fatalf("buildSubjectTermlyResult failed: %v", err)
	}
	if len(got.Exams) != 0 {
		t.Fatalf("expected the theory-pending exam to be excluded entirely (not shown as a partial 28/40), got %d exam rows", len(got.Exams))
	}
	if got.IsComplete {
		t.Error("expected IsComplete=false while a required theory score is still pending")
	}
}

func TestBuildSubjectTermlyResult_CombinesCBTAndTheoryIntoOneExamTotal(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	if err := db.Exec(
		"INSERT INTO exams (id, subject_id, class_id, term_id, session_id, exam_type, duration_minutes, total_marks, theory_max_marks) VALUES ('exam-main-1','subj-1','class-1','term-1','session-1','main_exam',60,40,30)",
	).Error; err != nil {
		t.Fatalf("failed to seed exam with theory portion: %v", err)
	}

	theoryScore := 22
	results := []models.Result{{ExamID: "exam-main-1", TotalScore: 28, Percentage: 70, Grade: "A", FullyGraded: true, TheoryScore: &theoryScore}}

	got, err := svc.buildSubjectTermlyResult(context.Background(), "student-1", "subj-1", "school-1", "level-1", results)
	if err != nil {
		t.Fatalf("buildSubjectTermlyResult failed: %v", err)
	}
	if len(got.Exams) != 1 {
		t.Fatalf("expected 1 combined exam row, got %d", len(got.Exams))
	}
	exam := got.Exams[0]
	if exam.Score != 50 || exam.TotalMarks != 70 {
		t.Errorf("Score/TotalMarks = %d/%d, want 50/70 (28 CBT + 22 theory out of 40+30)", exam.Score, exam.TotalMarks)
	}
	wantPct := 50.0 / 70.0 * 100
	if exam.Percentage != wantPct {
		t.Errorf("Percentage = %.4f, want %.4f (recomputed from the combined score/marks, not the stale CBT-only 70%%)", exam.Percentage, wantPct)
	}
	if got.TotalScore != 50 || got.TotalMarks != 70 {
		t.Errorf("subject TotalScore/TotalMarks = %d/%d, want 50/70", got.TotalScore, got.TotalMarks)
	}
}
