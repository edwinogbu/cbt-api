package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Shared fixture helpers for Mid-Term/Promotion report tests ----

func seedReportCardSession(t *testing.T, db *gorm.DB, id, schoolID, name string) {
	t.Helper()
	if err := db.Exec("INSERT INTO academic_sessions (id, school_id, name) VALUES (?, ?, ?)", id, schoolID, name).Error; err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
}

func seedReportCardTerm(t *testing.T, db *gorm.DB, id, sessionID string, termNumber int, name string) {
	t.Helper()
	if err := db.Exec("INSERT INTO terms (id, session_id, term_number, name) VALUES (?, ?, ?, ?)", id, sessionID, termNumber, name).Error; err != nil {
		t.Fatalf("failed to seed term: %v", err)
	}
}

func seedReportCardClassLevel(t *testing.T, db *gorm.DB, id, schoolID, name string, promotionTo *string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO class_levels (id, school_id, name, promotion_to, is_active) VALUES (?, ?, ?, ?, 1)",
		id, schoolID, name, promotionTo,
	).Error; err != nil {
		t.Fatalf("failed to seed class level: %v", err)
	}
}

func seedReportCardClass(t *testing.T, db *gorm.DB, id, schoolID, classLevelID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO classes (id, school_id, class_level_id) VALUES (?, ?, ?)",
		id, schoolID, classLevelID,
	).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedReportCardExam(t *testing.T, db *gorm.DB, id, subjectID, classID, termID, sessionID, examType string, totalMarks int) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO exams (id, subject_id, class_id, term_id, session_id, exam_type, duration_minutes, total_marks, pass_mark, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 60, ?, 50, ?, ?)",
		id, subjectID, classID, termID, sessionID, examType, totalMarks, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed exam: %v", err)
	}
}

func seedReportCardResult(t *testing.T, db *gorm.DB, studentID, examID string, totalScore int, fullyGraded bool) {
	t.Helper()
	if err := db.Create(&models.Result{
		ID: uuid.NewString(), StudentID: studentID, ExamID: examID, AttemptID: uuid.NewString(),
		TotalScore: totalScore, FullyGraded: fullyGraded,
	}).Error; err != nil {
		t.Fatalf("failed to seed result: %v", err)
	}
}

func seedReportCardStudent(t *testing.T, db *gorm.DB, id, userID, schoolID, classID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO students (id, user_id, school_id, class_id) VALUES (?, ?, ?, ?)",
		id, userID, schoolID, classID,
	).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES (?, ?, ?, 'student')", userID, "Test", "Student").Error; err != nil {
		t.Fatalf("failed to seed student's user: %v", err)
	}
}

// ---- Mid-Term Progress Report ----

func TestGetMidTermProgressReportForUser_ExcludesMainExamFromComputation(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)

	caExamID, mainExamID := "exam-ca", "exam-main"
	seedReportCardExam(t, db, caExamID, "subj-1", classID, termID, sessionID, "continuous_assessment", 20)
	seedReportCardExam(t, db, mainExamID, "subj-1", classID, termID, sessionID, "main_exam", 80)
	seedReportCardResult(t, db, "student-1", caExamID, 18, true)  // 90%
	seedReportCardResult(t, db, "student-1", mainExamID, 10, true) // 12.5% - must be excluded

	report, err := svc.GetMidTermProgressReportForUser(context.Background(), termID, sessionID, userID)
	if err != nil {
		t.Fatalf("GetMidTermProgressReportForUser failed: %v", err)
	}
	if len(report.Subjects) != 1 {
		t.Fatalf("Subjects length = %d, want 1", len(report.Subjects))
	}
	got := report.Subjects[0]
	if got.Percentage != 90 {
		t.Errorf("Percentage = %.2f, want 90 (only the CA exam's 18/20 - the main_exam result must be excluded from a mid-term snapshot)", got.Percentage)
	}
	if !got.IsComplete {
		t.Errorf("IsComplete = false, want true - the CA component has a result")
	}
}

func TestGetMidTermProgressReportForUser_NoResultsYetIsIncompleteNotZero(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedReportCardExam(t, db, "exam-ca", "subj-1", classID, termID, sessionID, "continuous_assessment", 20)

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	// No result seeded at all for this student.

	report, err := svc.GetMidTermProgressReportForUser(context.Background(), termID, sessionID, userID)
	if err != nil {
		t.Fatalf("GetMidTermProgressReportForUser failed: %v", err)
	}
	if len(report.Subjects) != 1 {
		t.Fatalf("Subjects length = %d, want 1", len(report.Subjects))
	}
	if report.Subjects[0].IsComplete {
		t.Error("IsComplete = true, want false - no result exists yet for this subject")
	}
}

func TestSetMidTermRemark_DoesNotOverwriteMainTermlyRemark(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-1")
	ctx := context.Background()

	teacherReq := &dto.SetReportCardRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Remark: "End of term: excellent work.",
	}
	if err := svc.SetTeacherReportCardRemark(ctx, "school-1", "teacher-1", teacherReq); err != nil {
		t.Fatalf("SetTeacherReportCardRemark failed: %v", err)
	}

	midTermReq := &dto.SetMidTermRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Comment: "Mid-term: participating well so far.", NextSteps: "Keep practicing.",
	}
	if err := svc.SetMidTermRemark(ctx, "school-1", "teacher-1", midTermReq); err != nil {
		t.Fatalf("SetMidTermRemark failed: %v", err)
	}

	got := svc.getReportCardRemark(ctx, "student-1", "term-1", "session-1")
	if got == nil {
		t.Fatal("getReportCardRemark returned nil")
	}
	if got.TeacherRemark != teacherReq.Remark {
		t.Errorf("TeacherRemark = %q, want %q (must survive the later mid-term write)", got.TeacherRemark, teacherReq.Remark)
	}
	if got.MidTermComment != midTermReq.Comment || got.MidTermNextSteps != midTermReq.NextSteps {
		t.Errorf("MidTermComment/MidTermNextSteps = %q/%q, want %q/%q", got.MidTermComment, got.MidTermNextSteps, midTermReq.Comment, midTermReq.NextSteps)
	}
}

func TestSetMidTermRemark_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-A")

	req := &dto.SetMidTermRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Comment: "Should never be written.",
	}
	if err := svc.SetMidTermRemark(context.Background(), "school-B", "teacher-1", req); err == nil {
		t.Fatal("expected School B's token to be rejected setting a mid-term remark on School A's student")
	}
}

// ---- Promotion Examination Report ----

func TestGetPromotionReportCardForUser_UsesLastTermOfSession(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID := "school-1", "class-1", "session-1"
	term1, term2 := "term-1", "term-2"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, term1, sessionID, 1, "First Term")
	seedReportCardTerm(t, db, term2, sessionID, 2, "Second Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)

	// Term 1: low score. Term 2 (the later term): high score. The
	// promotion report must reflect term 2, not term 1.
	examTerm1 := "exam-term1"
	examTerm2 := "exam-term2"
	seedReportCardExam(t, db, examTerm1, "subj-1", classID, term1, sessionID, "main_exam", 100)
	seedReportCardExam(t, db, examTerm2, "subj-1", classID, term2, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", examTerm1, 20, true) // 20%
	seedReportCardResult(t, db, "student-1", examTerm2, 90, true) // 90%

	report, err := svc.GetPromotionReportCardForUser(context.Background(), sessionID, userID)
	if err != nil {
		t.Fatalf("GetPromotionReportCardForUser failed: %v", err)
	}
	if len(report.Subjects) != 1 {
		t.Fatalf("Subjects length = %d, want 1", len(report.Subjects))
	}
	if report.Subjects[0].Percentage != 90 {
		t.Errorf("Percentage = %.2f, want 90 (term 2's result, the session's later term, not term 1's 20%%)", report.Subjects[0].Percentage)
	}
}

func TestGetPromotionReportCardForUser_EligibleWhenAboveBuiltinThreshold(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "Final Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)

	examID := "exam-1"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", examID, 75, true) // well above the builtin 50% threshold

	report, err := svc.GetPromotionReportCardForUser(context.Background(), sessionID, userID)
	if err != nil {
		t.Fatalf("GetPromotionReportCardForUser failed: %v", err)
	}
	if report.SystemRecommendation != "ELIGIBLE FOR REVIEW" {
		t.Errorf("SystemRecommendation = %q, want ELIGIBLE FOR REVIEW", report.SystemRecommendation)
	}
	if report.FinalDecision != "" {
		t.Errorf("FinalDecision = %q, want empty - no decision has been recorded yet, and a recommendation must never be copied into it", report.FinalDecision)
	}
}

func TestGetPromotionReportCardForUser_BelowThreshold_NotEligible(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "Final Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)

	examID := "exam-1"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", examID, 30, true) // below the builtin 50% threshold

	report, err := svc.GetPromotionReportCardForUser(context.Background(), sessionID, userID)
	if err != nil {
		t.Fatalf("GetPromotionReportCardForUser failed: %v", err)
	}
	if report.SystemRecommendation != "NOT ELIGIBLE FOR REVIEW" {
		t.Errorf("SystemRecommendation = %q, want NOT ELIGIBLE FOR REVIEW", report.SystemRecommendation)
	}
}

func TestGetPromotionReportCardForUser_ResolvesProposedNextClassFromRealPromotionTo(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, sessionID, termID := "school-1", "session-1", "term-1"
	currentLevelID, nextLevelID := "level-jss2", "level-jss3"
	classID := "class-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClassLevel(t, db, nextLevelID, schoolID, "JSS 3", nil)
	seedReportCardClassLevel(t, db, currentLevelID, schoolID, "JSS 2", &nextLevelID)
	seedReportCardClass(t, db, classID, schoolID, currentLevelID)
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "Final Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	examID := "exam-1"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", examID, 80, true)

	report, err := svc.GetPromotionReportCardForUser(context.Background(), sessionID, userID)
	if err != nil {
		t.Fatalf("GetPromotionReportCardForUser failed: %v", err)
	}
	if report.ProposedNextClass != "JSS 3" {
		t.Errorf("ProposedNextClass = %q, want \"JSS 3\" (this class level's real configured PromotionTo)", report.ProposedNextClass)
	}
}

func TestRecordPromotionDecision_AppearsOnSubsequentReportFetch(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "Final Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")

	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	examID := "exam-1"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)
	seedReportCardResult(t, db, "student-1", examID, 80, true)

	ctx := context.Background()
	decisionReq := &dto.RecordPromotionDecisionRequest{
		StudentID: "student-1", SessionID: sessionID, Decision: "Promoted", Notes: "Approved by academic board.",
	}
	if err := svc.RecordPromotionDecision(ctx, schoolID, "admin-1", decisionReq); err != nil {
		t.Fatalf("RecordPromotionDecision failed: %v", err)
	}

	report, err := svc.GetPromotionReportCardForUser(ctx, sessionID, userID)
	if err != nil {
		t.Fatalf("GetPromotionReportCardForUser failed: %v", err)
	}
	if report.FinalDecision == "" {
		t.Fatal("FinalDecision is empty after a decision was recorded")
	}
	if !strings.Contains(report.FinalDecision, "Promoted") {
		t.Errorf("FinalDecision = %q, want it to contain the real recorded decision \"Promoted\"", report.FinalDecision)
	}
}

func TestRecordPromotionDecision_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-A")

	req := &dto.RecordPromotionDecisionRequest{StudentID: "student-1", SessionID: "session-1", Decision: "Promoted"}
	if err := svc.RecordPromotionDecision(context.Background(), "school-B", "admin-1", req); err == nil {
		t.Fatal("expected School B's token to be rejected recording a decision for School A's student")
	}
}

func TestRecordPromotionDecision_RequiresDecisionField(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-1")

	req := &dto.RecordPromotionDecisionRequest{StudentID: "student-1", SessionID: "session-1", Decision: ""}
	if err := svc.RecordPromotionDecision(context.Background(), "school-1", "admin-1", req); err == nil {
		t.Fatal("expected an empty Decision to be rejected")
	}
}
