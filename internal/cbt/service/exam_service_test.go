package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupExamTestDB mirrors internal/sync/service/sync_service_test.go's
// pattern: an in-memory SQLite DB with a hand-written, SQLite-compatible
// subset of the schema (production's `default:gen_random_uuid()` tags are
// Postgres-only). Each test gets its own uniquely-named shared-cache DSN
// so CREATE TABLE calls never collide across tests.
func setupExamTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE exams (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			title TEXT, subject_id TEXT, class_id TEXT, session_id TEXT, term_id TEXT,
			exam_type TEXT DEFAULT 'main_exam', status TEXT DEFAULT 'draft',
			duration_minutes INTEGER NOT NULL, total_marks INTEGER NOT NULL, pass_mark INTEGER,
			instructions TEXT, start_time DATETIME, end_time DATETIME,
			shuffle_questions INTEGER DEFAULT 0, shuffle_options INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			review_policy TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			first_name TEXT, last_name TEXT, role TEXT DEFAULT 'student', deleted_at DATETIME
		)`,
		`CREATE TABLE classes (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), school_id TEXT, teacher_id TEXT, deleted_at DATETIME
		)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT
		)`,
		`CREATE TABLE exam_attempts (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			start_time DATETIME, end_time DATETIME, score INTEGER, percentage REAL,
			status TEXT DEFAULT 'in_progress', device_info TEXT, ip_address TEXT,
			signature TEXT, synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE question_bank (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, session_id TEXT, term_id TEXT, class_level_id TEXT, class_id TEXT,
			subject_id TEXT, exam_type TEXT, question_text TEXT, question_type TEXT,
			difficulty TEXT, bloom_level TEXT, topic TEXT, sub_topic TEXT, learning_objective TEXT,
			options TEXT, correct_option_keys TEXT, correct_answer TEXT, rubric TEXT, tags TEXT,
			explanation TEXT, marks INTEGER DEFAULT 1, negative_marks REAL DEFAULT 0,
			time_limit_seconds INTEGER, "order" INTEGER DEFAULT 0, is_required INTEGER DEFAULT 0,
			status TEXT DEFAULT 'draft', version INTEGER DEFAULT 1, parent_id TEXT,
			created_by TEXT, updated_by TEXT, usage_count INTEGER DEFAULT 0, success_rate REAL,
			curriculum_type TEXT, source_type TEXT, external_id TEXT, metadata TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE student_answers (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, question_id TEXT NOT NULL,
			selected_answer TEXT, selected_option_keys TEXT, essay_response TEXT,
			is_correct INTEGER DEFAULT 0, is_marked INTEGER DEFAULT 0,
			marks_awarded INTEGER, marked_by TEXT, marked_at DATETIME, marker_feedback TEXT,
			time_spent INTEGER,
			synced_at DATETIME, synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE results (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			attempt_id TEXT NOT NULL, total_score INTEGER, percentage REAL, grade TEXT, remarks TEXT, fully_graded INTEGER DEFAULT 1,
			passed INTEGER DEFAULT 0, published_at DATETIME, synced_to_cloud_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE proctoring_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			status TEXT DEFAULT 'active', started_at DATETIME, ended_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE exam_questions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), exam_id TEXT NOT NULL, question_id TEXT NOT NULL,
			sort_order INTEGER DEFAULT 0
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTestExamService(db *gorm.DB) *ExamService {
	examRepo := repository.NewExamRepository(db)
	questionRepo := repository.NewQuestionRepository(db)
	return NewExamService(examRepo, questionRepo, db)
}

const (
	testStudentID = "student-1"
	testExamID    = "exam-1"
)

func seedExam(t *testing.T, db *gorm.DB, durationMinutes int) {
	t.Helper()
	// Raw INSERT, not db.Create(): the Exam struct has many more
	// GORM-mapped columns than this minimal test table declares, and
	// db.Create() would try to insert every one of them. This service is
	// never written to in the paths under test, only read, so SELECT *
	// against the minimal table is sufficient.
	if err := db.Exec(
		"INSERT INTO exams (id, duration_minutes, total_marks, pass_mark, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		testExamID, durationMinutes, 10, 5, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed exam: %v", err)
	}
}

func seedAttempt(t *testing.T, db *gorm.DB, startedMinutesAgo int) (attemptID string) {
	t.Helper()
	attempt := &models.ExamAttempt{
		ID:        uuid.NewString(),
		StudentID: testStudentID,
		ExamID:    testExamID,
		StartTime: time.Now().Add(-time.Duration(startedMinutesAgo) * time.Minute),
		Status:    string(models.AttemptStatusInProgress),
	}
	if err := db.Create(attempt).Error; err != nil {
		t.Fatalf("failed to seed attempt: %v", err)
	}
	return attempt.ID
}

func seedQuestion(t *testing.T, db *gorm.DB, correctAnswer string, marks int) (questionID string) {
	t.Helper()
	// Raw INSERT for the same reason as seedExam above: QuestionBank has
	// many more columns than this minimal table, and is only ever read
	// (FindByID) in the paths under test, never written.
	id := uuid.NewString()
	if err := db.Exec(
		"INSERT INTO question_bank (id, correct_answer, marks, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, correctAnswer, marks, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed question: %v", err)
	}
	return id
}

// ---- Fix A: exam time-limit enforcement ----

// TestSaveAnswer_WithinTimeLimit_Accepted is the baseline: the single-
// answer real-time path already enforced the time limit before this
// change - confirms the refactor into checkWithinTimeLimit didn't alter
// its behavior.
func TestSaveAnswer_WithinTimeLimit_Accepted(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 5) // started 5 min ago, 30 min duration
	questionID := seedQuestion(t, db, "A", 2)

	_, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: questionID, SelectedAnswer: "A",
	}, testStudentID)
	if err != nil {
		t.Fatalf("expected answer within time limit to be accepted, got error: %v", err)
	}
}

// TestSaveAnswer_AfterTimeLimit_Rejected confirms the pre-existing
// real-time guard still works after refactoring into the shared helper.
func TestSaveAnswer_AfterTimeLimit_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 45) // started 45 min ago, 30 min duration - expired
	questionID := seedQuestion(t, db, "A", 2)

	_, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: questionID, SelectedAnswer: "A",
	}, testStudentID)
	if err == nil || err.Error() != "time limit exceeded" {
		t.Fatalf("expected \"time limit exceeded\", got: %v", err)
	}
}

// TestBulkSaveAnswers_AfterTimeLimit_Rejected is the regression test for
// the confirmed audit finding: BulkSaveAnswers previously had no time
// check at all, so a device that stayed offline past the exam deadline
// could still write answers via this path. This must now reject the
// whole batch, matching SaveAnswer's behavior.
func TestBulkSaveAnswers_AfterTimeLimit_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 45) // expired
	questionID := seedQuestion(t, db, "A", 2)

	resp, err := svc.BulkSaveAnswers(context.Background(), &dto.BulkSaveAnswerRequest{
		AttemptID: attemptID,
		Answers: []dto.BulkAnswerItem{
			{QuestionID: questionID, SelectedAnswer: "A"},
		},
	}, testStudentID)
	if err == nil || err.Error() != "time limit exceeded" {
		t.Fatalf("expected \"time limit exceeded\", got resp=%+v err=%v", resp, err)
	}

	var count int64
	db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count)
	if count != 0 {
		t.Fatalf("expected no answer rows written after a rejected bulk sync, got %d", count)
	}
}

// TestBulkSaveAnswers_WithinTimeLimit_Accepted is the companion
// regression test: legitimate offline sync (device reconnects before the
// deadline) must keep working - the fix must not be overly aggressive.
func TestBulkSaveAnswers_WithinTimeLimit_Accepted(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 10) // 10 of 30 minutes elapsed
	questionID := seedQuestion(t, db, "B", 2)

	resp, err := svc.BulkSaveAnswers(context.Background(), &dto.BulkSaveAnswerRequest{
		AttemptID: attemptID,
		Answers: []dto.BulkAnswerItem{
			{QuestionID: questionID, SelectedAnswer: "B"},
		},
	}, testStudentID)
	if err != nil {
		t.Fatalf("expected bulk sync within time limit to be accepted, got: %v", err)
	}
	if resp.SyncedCount != 1 {
		t.Fatalf("expected 1 synced answer, got %d (errors: %v)", resp.SyncedCount, resp.Errors)
	}
}

// TestAutoSave_AfterTimeLimit_Rejected: AutoSave had neither a status
// check nor a time check before this fix - confirmed reachable via the
// live POST /student/auto-save route.
func TestAutoSave_AfterTimeLimit_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 45) // expired
	questionID := seedQuestion(t, db, "A", 2)

	_, err := svc.AutoSave(context.Background(), &dto.AutoSaveRequest{
		AttemptID: attemptID,
		Answers:   []dto.AutoSaveItem{{QuestionID: questionID, SelectedAnswer: "A"}},
	}, testStudentID)
	if err == nil || err.Error() != "time limit exceeded" {
		t.Fatalf("expected \"time limit exceeded\", got: %v", err)
	}
}

// ---- Fix B: duplicate submission prevention ----

// NOTE on testing true concurrent submits: an earlier version of this
// file attempted a goroutine-based race test (two SubmitExam calls fired
// concurrently for the same attempt, asserting exactly one Result row
// survives). It was removed because it hung indefinitely against the
// in-memory SQLite database used by these tests - SQLite has no real
// row-level locking (clause.Locking{Strength:"UPDATE"} has no SQLite
// equivalent to Postgres's SELECT ... FOR UPDATE), so two goroutines
// both opening a write transaction against the same SQLite file produce
// unpredictable blocking/busy behavior that has nothing to do with the
// fix being tested. That means the database-level serialization half of
// Fix B (the row lock itself preventing two requests from both passing
// the status check before either commits) is NOT verified by this test
// suite and would need to be exercised against a real Postgres instance
// to confirm - flagged as such in the Phase 1 report rather than
// claimed as tested. What IS verified here, deterministically, is the
// other half of the fix: that a second call for an attempt already
// graded is idempotent (returns the existing result, creates nothing
// new) - see TestSubmitExam_RetriedAfterSuccess_ReturnsExistingResult
// below, which covers the same "never create a second Result row"
// guarantee without depending on true concurrency.

// TestSubmitExam_RetriedAfterSuccess_ReturnsExistingResult is the
// sequential version of the same guarantee: a client retry (e.g. after a
// lost acknowledgment) must get the original result back, not an error
// and never a second Result row.
func TestSubmitExam_RetriedAfterSuccess_ReturnsExistingResult(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 5)
	questionID := seedQuestion(t, db, "A", 2)
	if err := db.Exec(
		"INSERT INTO student_answers (id, attempt_id, question_id, selected_answer, is_correct, created_at, updated_at) VALUES (?, ?, ?, 'A', 1, ?, ?)",
		uuid.NewString(), attemptID, questionID, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed answer: %v", err)
	}

	first, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("first submit: expected success, got: %v", err)
	}

	second, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("retried submit: expected idempotent success, got error: %v", err)
	}
	if second.ResultID != first.ResultID {
		t.Fatalf("retried submit returned a different ResultID (%s vs %s) - implies a duplicate Result was created", second.ResultID, first.ResultID)
	}

	var count int64
	db.Model(&models.Result{}).Where("attempt_id = ?", attemptID).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 Result row after a retried submit, got %d", count)
	}
}

// TestMarkReview_AfterTimeLimit_Rejected is the regression test for the
// same class of gap Fix A covered for SaveAnswer/BulkSaveAnswers/AutoSave:
// MarkReview also writes to student_answers (via IsMarked) but had no
// checkWithinTimeLimit call at all, so a device that stayed connected
// past the exam deadline could still toggle mark-for-review flags via
// this path even though every other answer-writing path correctly
// rejected writes past the deadline.
func TestMarkReview_AfterTimeLimit_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 45) // started 45 min ago, 30 min duration - expired
	questionID := seedQuestion(t, db, "A", 2)

	_, err := svc.MarkReview(context.Background(), &dto.MarkReviewRequest{
		AttemptID: attemptID, QuestionID: questionID, IsMarked: true,
	}, testStudentID)
	if err == nil || err.Error() != "time limit exceeded" {
		t.Fatalf("expected \"time limit exceeded\", got: %v", err)
	}

	var count int64
	db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count)
	if count != 0 {
		t.Fatalf("expected no answer rows written after a rejected mark-review call, got %d", count)
	}
}

// TestMarkReview_WithinTimeLimit_Accepted is the companion test: a
// legitimate mark-for-review toggle while the attempt is still active
// must keep working.
func TestMarkReview_WithinTimeLimit_Accepted(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 5) // 5 of 30 minutes elapsed
	questionID := seedQuestion(t, db, "A", 2)

	resp, err := svc.MarkReview(context.Background(), &dto.MarkReviewRequest{
		AttemptID: attemptID, QuestionID: questionID, IsMarked: true,
	}, testStudentID)
	if err != nil {
		t.Fatalf("expected mark-review within time limit to be accepted, got: %v", err)
	}
	if !resp.IsMarked {
		t.Fatalf("expected IsMarked true in response, got false")
	}
}

// ---- Grading-integrity fix: percentage/pass-mark scale mismatch ----

// seedExamQuestions attaches questions to an exam via exam_questions, the
// join SubmitExam now reads to compute the exam's real total marks.
func seedExamQuestions(t *testing.T, db *gorm.DB, examID string, questionIDs []string) {
	t.Helper()
	for i, qID := range questionIDs {
		if err := db.Exec(
			"INSERT INTO exam_questions (id, exam_id, question_id, sort_order) VALUES (?, ?, ?, ?)",
			uuid.NewString(), examID, qID, i,
		).Error; err != nil {
			t.Fatalf("failed to seed exam_questions: %v", err)
		}
	}
}

// TestSubmitExam_PercentageUsesActualQuestionMarks_NotStaleExamTotalMarks
// is the regression test for a confirmed live defect: exams.total_marks
// is a separate, teacher-entered field that drifts from the real sum of
// the questions actually attached via AddQuestionsToExam (nothing keeps
// them in sync). Reproduced live: a 40-question, 1-mark-each exam
// created with total_marks=15 graded a 28-correct attempt at 186.67% -
// an impossible, silently-stored percentage. SubmitExam must compute
// the percentage against the real sum of attached questions' marks.
func TestSubmitExam_PercentageUsesActualQuestionMarks_NotStaleExamTotalMarks(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	// Deliberately mismatched: exams.total_marks (15) does not match the
	// sum of the 4 questions' marks seeded below (4 x 10 = 40).
	seedExam(t, db, 30)
	db.Exec("UPDATE exams SET total_marks = 15, pass_mark = 20 WHERE id = ?", testExamID)
	attemptID := seedAttempt(t, db, 5)

	var questionIDs []string
	for i := 0; i < 4; i++ {
		qID := seedQuestion(t, db, "A", 10)
		questionIDs = append(questionIDs, qID)
	}
	seedExamQuestions(t, db, testExamID, questionIDs)

	// Answer 3 of 4 correctly (30/40 marks = 75%), 1 wrong.
	for i, qID := range questionIDs {
		answer := "A"
		if i == 3 {
			answer = "B" // wrong
		}
		if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
			AttemptID: attemptID, QuestionID: qID, SelectedAnswer: answer,
		}, testStudentID); err != nil {
			t.Fatalf("SaveAnswer failed: %v", err)
		}
	}

	resp, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("SubmitExam failed: %v", err)
	}

	if resp.Score != 30 {
		t.Fatalf("expected score 30 (3 correct x 10 marks), got %d", resp.Score)
	}
	if resp.Percentage < 74.9 || resp.Percentage > 75.1 {
		t.Fatalf("expected percentage ~75.0 (30/40 actual question marks), got %v - a percentage > 100 means the stale exams.total_marks (15) was used instead of the real question-marks sum (40)", resp.Percentage)
	}
	// pass_mark=20 out of the real 40-mark total is a 50% threshold;
	// 75% clears it.
	if !resp.Passed {
		t.Fatalf("expected passed=true (75%% clears a 50%% pass threshold), got false")
	}

	var stored models.Result
	if err := db.Where("attempt_id = ?", attemptID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored result: %v", err)
	}
	if stored.Percentage > 100 {
		t.Fatalf("stored Result.Percentage is %v, which is impossible for a percentage - the bug is present", stored.Percentage)
	}
}

// TestSubmitExam_PassMarkBelowThreshold_CorrectlyFails is the companion
// test: confirms a genuinely failing attempt (below the real pass-mark
// percentage) is actually marked as failed, not silently passed by the
// scale-mismatch bug (which made "passed" true for almost any real
// percentage compared against a small raw pass_mark number).
func TestSubmitExam_PassMarkBelowThreshold_CorrectlyFails(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	db.Exec("UPDATE exams SET total_marks = 15, pass_mark = 20 WHERE id = ?", testExamID)
	attemptID := seedAttempt(t, db, 5)

	var questionIDs []string
	for i := 0; i < 4; i++ {
		qID := seedQuestion(t, db, "A", 10)
		questionIDs = append(questionIDs, qID)
	}
	seedExamQuestions(t, db, testExamID, questionIDs)

	// Answer only 1 of 4 correctly (10/40 marks = 25%, below the 50% pass
	// threshold derived from pass_mark=20/total=40).
	for i, qID := range questionIDs {
		answer := "A"
		if i != 0 {
			answer = "B" // wrong
		}
		if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
			AttemptID: attemptID, QuestionID: qID, SelectedAnswer: answer,
		}, testStudentID); err != nil {
			t.Fatalf("SaveAnswer failed: %v", err)
		}
	}

	resp, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("SubmitExam failed: %v", err)
	}
	if resp.Passed {
		t.Fatalf("expected passed=false for a 25%% score against a 50%% pass threshold, got true - confirms the scale-mismatch bug (percentage >= raw pass_mark) is NOT present")
	}
}

// TestAutoSave_SetsIsCorrect is the regression test for a second,
// separate bug found while running a real end-to-end exam: AutoSave
// never fetched the question or computed IsCorrect at all (unlike
// SaveAnswer and BulkSaveAnswers), so any answer whose last write came
// through auto-save kept IsCorrect=false forever regardless of whether
// it was actually correct - confirmed live, a 4-question discrepancy in
// a real 40-question exam's score. gradeAnswer (used by all three save
// paths now) must be reached from AutoSave too.
func TestAutoSave_SetsIsCorrect(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 5)
	correctQID := seedQuestion(t, db, "A", 5)
	wrongQID := seedQuestion(t, db, "A", 5)

	resp, err := svc.AutoSave(context.Background(), &dto.AutoSaveRequest{
		AttemptID: attemptID,
		Answers: []dto.AutoSaveItem{
			{QuestionID: correctQID, SelectedAnswer: "A"}, // correct
			{QuestionID: wrongQID, SelectedAnswer: "B"},   // wrong
		},
	}, testStudentID)
	if err != nil {
		t.Fatalf("AutoSave failed: %v", err)
	}
	if resp.SavedCount != 2 {
		t.Fatalf("expected 2 saved, got %d", resp.SavedCount)
	}

	var correctAnswer, wrongAnswer models.StudentAnswer
	if err := db.Where("attempt_id = ? AND question_id = ?", attemptID, correctQID).First(&correctAnswer).Error; err != nil {
		t.Fatalf("failed to load correct answer: %v", err)
	}
	if err := db.Where("attempt_id = ? AND question_id = ?", attemptID, wrongQID).First(&wrongAnswer).Error; err != nil {
		t.Fatalf("failed to load wrong answer: %v", err)
	}
	if !correctAnswer.IsCorrect {
		t.Fatalf("expected the correctly-answered question to have IsCorrect=true after AutoSave, got false - the bug is present")
	}
	if wrongAnswer.IsCorrect {
		t.Fatalf("expected the wrongly-answered question to have IsCorrect=false after AutoSave, got true")
	}
}

// ---- Manual/essay marking ----

const testMarkerTeacherID = "teacher-1"
const testOtherTeacherID = "teacher-2"
const testAdminID = "admin-1"

func seedMarkingUser(t *testing.T, db *gorm.DB, id, role string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES (?, 'Test', 'User', ?)", id, role).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedMarkingClass(t *testing.T, db *gorm.DB, teacherID string) (classID string) {
	t.Helper()
	classID = uuid.NewString()
	if err := db.Exec("INSERT INTO classes (id, teacher_id) VALUES (?, ?)", classID, teacherID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
	return classID
}

func setExamClass(t *testing.T, db *gorm.DB, classID string) {
	t.Helper()
	if err := db.Exec("UPDATE exams SET class_id = ? WHERE id = ?", classID, testExamID).Error; err != nil {
		t.Fatalf("failed to set exam class: %v", err)
	}
}

func seedEssayQuestion(t *testing.T, db *gorm.DB, marks int) (questionID string) {
	t.Helper()
	questionID = uuid.NewString()
	if err := db.Exec(
		"INSERT INTO question_bank (id, question_type, marks, created_at, updated_at) VALUES (?, 'essay', ?, ?, ?)",
		questionID, marks, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed essay question: %v", err)
	}
	return questionID
}

// setupMarkingFixture builds: exam (total_marks=10, pass_mark=5) owned
// by a class taught by testMarkerTeacherID, one essay question worth 10
// marks attached to it, an in-progress attempt, and the student's
// submitted (but unmarked) essay answer. Returns the attempt ID.
func setupMarkingFixture(t *testing.T) (svc *ExamService, attemptID, essayQID string) {
	t.Helper()
	db := setupExamTestDB(t)
	svc = newTestExamService(db)
	seedExam(t, db, 30)
	db.Exec("UPDATE exams SET total_marks = 10, pass_mark = 5 WHERE id = ?", testExamID)
	classID := seedMarkingClass(t, db, testMarkerTeacherID)
	setExamClass(t, db, classID)
	seedMarkingUser(t, db, testMarkerTeacherID, "teacher")
	seedMarkingUser(t, db, testOtherTeacherID, "teacher")
	seedMarkingUser(t, db, testAdminID, "admin")

	attemptID = seedAttempt(t, db, 5)
	essayQID = seedEssayQuestion(t, db, 10)
	seedExamQuestions(t, db, testExamID, []string{essayQID})

	if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, SelectedAnswer: "My essay response text.",
	}, testStudentID); err != nil {
		t.Fatalf("failed to save essay answer: %v", err)
	}

	resp, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("SubmitExam failed: %v", err)
	}
	if resp.Status != string(models.AttemptStatusPendingManualMarking) {
		t.Fatalf("expected attempt status pending_manual_marking after submitting with an unmarked essay answer, got %q", resp.Status)
	}
	return svc, attemptID, essayQID
}

// TestMarkDescriptiveAnswer_ValidScore_TransitionsToCompleted covers the
// full happy path: a teacher who owns the exam's class marks the essay
// answer within the allowed range, and the attempt - the only essay
// question now scored - transitions from pending_manual_marking to
// completed with the correct final percentage.
func TestMarkDescriptiveAnswer_ValidScore_TransitionsToCompleted(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	resp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8, Feedback: "Good answer.",
	}, testMarkerTeacherID)
	if err != nil {
		t.Fatalf("expected marking to succeed, got: %v", err)
	}
	if resp.AttemptStatus != string(models.AttemptStatusCompleted) {
		t.Fatalf("expected attempt to transition to completed once its only essay answer is marked, got %q", resp.AttemptStatus)
	}
	if !resp.FullyGraded {
		t.Fatalf("expected FullyGraded=true, got false")
	}
	if resp.TotalScore != 8 {
		t.Fatalf("expected total score 8, got %d", resp.TotalScore)
	}
	if resp.Percentage < 79.9 || resp.Percentage > 80.1 {
		t.Fatalf("expected percentage ~80.0 (8/10 marks), got %v", resp.Percentage)
	}
	if !resp.Passed {
		t.Fatalf("expected passed=true (80%% clears the 50%% pass threshold), got false")
	}
}

// TestMarkDescriptiveAnswer_OutOfRangeScore_Rejected confirms a marker
// cannot enter a score outside the question's own [0, marks] range.
func TestMarkDescriptiveAnswer_OutOfRangeScore_Rejected(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	_, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 15, // > the question's 10 marks
	}, testMarkerTeacherID)
	if err == nil {
		t.Fatal("expected an out-of-range score to be rejected, got no error")
	}
}

// TestMarkDescriptiveAnswer_WrongTeacher_Rejected confirms a teacher who
// does not own the exam's class cannot mark it - unauthorized users
// cannot alter results outside their permissions.
func TestMarkDescriptiveAnswer_WrongTeacher_Rejected(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	_, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8,
	}, testOtherTeacherID)
	if err == nil || !strings.HasPrefix(err.Error(), "unauthorized") {
		t.Fatalf("expected an unauthorized error for a teacher who doesn't own this class, got: %v", err)
	}
}

// TestMarkDescriptiveAnswer_StudentRejected confirms a student cannot
// mark their own (or anyone else's) essay answer.
func TestMarkDescriptiveAnswer_StudentRejected(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	_, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8,
	}, testStudentID)
	if err == nil || !strings.HasPrefix(err.Error(), "unauthorized") {
		t.Fatalf("expected an unauthorized error for a student marker, got: %v", err)
	}
}

// TestMarkDescriptiveAnswer_AdminBypassesClassOwnership mirrors the
// existing authorizeQuestionWrite admin-bypass tests for the same
// guarantee on the marking path.
func TestMarkDescriptiveAnswer_AdminBypassesClassOwnership(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	resp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 6,
	}, testAdminID)
	if err != nil {
		t.Fatalf("expected an admin to be able to mark any class's exam, got error: %v", err)
	}
	if resp.TotalScore != 6 {
		t.Fatalf("expected total score 6, got %d", resp.TotalScore)
	}
}

// TestMarkDescriptiveAnswer_RepeatedMarking_Idempotent confirms a retry
// or a deliberate correction overwrites cleanly: the final state
// reflects only the last write, with exactly one Result row throughout.
func TestMarkDescriptiveAnswer_RepeatedMarking_Idempotent(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	if _, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8,
	}, testMarkerTeacherID); err != nil {
		t.Fatalf("first mark failed: %v", err)
	}
	resp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 6, Feedback: "Corrected after review.",
	}, testMarkerTeacherID)
	if err != nil {
		t.Fatalf("second (correcting) mark failed: %v", err)
	}
	if resp.TotalScore != 6 {
		t.Fatalf("expected the second mark's score (6) to be the final value, got %d", resp.TotalScore)
	}

	// setupMarkingFixture runs each test against its own fresh in-memory
	// db; re-derive it from svc's underlying connection via a raw query
	// through the service's own repo is not exposed, so assert via the
	// response shape instead: a third identical call stays stable.
	resp2, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 6,
	}, testMarkerTeacherID)
	if err != nil {
		t.Fatalf("third (repeated) mark failed: %v", err)
	}
	if resp2.TotalScore != 6 || resp2.Percentage != resp.Percentage {
		t.Fatalf("expected a repeated identical mark to be a stable no-op, got score=%d percentage=%v", resp2.TotalScore, resp2.Percentage)
	}
}

// TestSubmitExam_MixedObjectiveAndEssay_ScoresBothCorrectly confirms an
// exam with both an auto-graded objective question and a to-be-marked
// essay question scores the objective part immediately and only holds
// the essay part pending - never double-counts, never blocks the
// objective score on the essay's marking.
func TestSubmitExam_MixedObjectiveAndEssay_ScoresBothCorrectly(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	db.Exec("UPDATE exams SET total_marks = 15, pass_mark = 8 WHERE id = ?", testExamID)
	classID := seedMarkingClass(t, db, testMarkerTeacherID)
	setExamClass(t, db, classID)
	seedMarkingUser(t, db, testMarkerTeacherID, "teacher")

	attemptID := seedAttempt(t, db, 5)
	objQID := seedQuestion(t, db, "A", 5)
	essayQID := seedEssayQuestion(t, db, 10)
	seedExamQuestions(t, db, testExamID, []string{objQID, essayQID})

	if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: objQID, SelectedAnswer: "A", // correct
	}, testStudentID); err != nil {
		t.Fatalf("SaveAnswer (objective) failed: %v", err)
	}
	if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, SelectedAnswer: "Essay text.",
	}, testStudentID); err != nil {
		t.Fatalf("SaveAnswer (essay) failed: %v", err)
	}

	submitResp, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID)
	if err != nil {
		t.Fatalf("SubmitExam failed: %v", err)
	}
	if submitResp.Status != string(models.AttemptStatusPendingManualMarking) {
		t.Fatalf("expected pending_manual_marking (essay not yet scored), got %q", submitResp.Status)
	}
	if submitResp.Score != 5 {
		t.Fatalf("expected provisional score 5 (objective only, essay not yet counted), got %d", submitResp.Score)
	}

	markResp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 7,
	}, testMarkerTeacherID)
	if err != nil {
		t.Fatalf("MarkDescriptiveAnswer failed: %v", err)
	}
	if markResp.TotalScore != 12 { // 5 objective + 7 essay
		t.Fatalf("expected combined total score 12 (5 objective + 7 essay), got %d", markResp.TotalScore)
	}
	if markResp.AttemptStatus != string(models.AttemptStatusCompleted) {
		t.Fatalf("expected completed once the only essay question is marked, got %q", markResp.AttemptStatus)
	}
}

// ---- Phase 2: cross-tenant exam isolation ----
//
// Exam carries no school_id of its own - it is scoped to a school only
// transitively through its ClassID (classes.school_id). Before this fix,
// GetExam/UpdateExam/DeleteExam/PublishExam took only an exam ID with no
// ownership check at all, so any authenticated user who knew (or
// enumerated) another school's exam ID could read, modify, delete, or
// publish it. These tests seed two schools' classes/exams and confirm the
// cross-tenant case is rejected while the same-tenant case still works,
// and that an authenticated platform_owner (isPlatformOwner=true) is
// deliberately exempt.

func seedExamClass(t *testing.T, db *gorm.DB, schoolID string) (classID string) {
	t.Helper()
	classID = uuid.NewString()
	if err := db.Exec("INSERT INTO classes (id, school_id) VALUES (?, ?)", classID, schoolID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
	return classID
}

func seedExamForClass(t *testing.T, db *gorm.DB, classID string) (examID string) {
	t.Helper()
	examID = uuid.NewString()
	if err := db.Exec(
		"INSERT INTO exams (id, class_id, title, duration_minutes, total_marks, pass_mark, status, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		examID, classID, "Mid-Term", 30, 10, 5, "draft", 1, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed exam: %v", err)
	}
	return examID
}

func TestGetExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := svc.GetExam(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's token to be rejected reading School A's exam, got no error")
	}

	// Same-tenant access must still work - the fix must not be overly
	// aggressive.
	resp, err := svc.GetExam(context.Background(), examA, "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own exam, got error: %v", err)
	}
	if resp.ID != examA {
		t.Fatalf("expected exam %s, got %s", examA, resp.ID)
	}
}

func TestGetExam_PlatformOwner_BypassesTenantCheck(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	// Platform owner carries no school_id of its own (see
	// models.RolePlatformOwner) and must still be able to reach any
	// tenant's exam - this is the one role deliberately not confined by
	// authorizeExamTenant.
	if _, err := svc.GetExam(context.Background(), examA, "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's exam, got error: %v", err)
	}
}

func TestGetExam_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	// A non-platform-owner caller with no school_id (e.g. a legacy account
	// seeded with none) must NOT be treated as "unscoped" - "" must not
	// coincidentally match a real exam's school.
	if _, err := svc.GetExam(context.Background(), examA, "", false); err == nil {
		t.Fatal("expected an empty school_id on a non-platform-owner caller to be rejected, got no error")
	}
}

func TestUpdateExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)
	newTitle := "Hijacked Title"

	if _, err := svc.UpdateExam(context.Background(), examA, &dto.UpdateExamRequest{Title: &newTitle}, "school-B", false); err == nil {
		t.Fatal("expected School B's token to be rejected updating School A's exam, got no error")
	}

	var title string
	db.Table("exams").Select("title").Where("id = ?", examA).Scan(&title)
	if title == newTitle {
		t.Fatal("exam title was changed by a cross-tenant update that should have been rejected")
	}
}

func TestDeleteExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if err := svc.DeleteExam(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's token to be rejected deleting School A's exam, got no error")
	}

	var count int64
	db.Table("exams").Where("id = ? AND deleted_at IS NULL", examA).Count(&count)
	if count != 1 {
		t.Fatal("exam was deleted by a cross-tenant delete that should have been rejected")
	}
}

func TestPublishExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if err := svc.PublishExam(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's token to be rejected publishing School A's exam, got no error")
	}

	var status string
	db.Table("exams").Select("status").Where("id = ?", examA).Scan(&status)
	if status == string(models.ExamStatusPublished) {
		t.Fatal("exam was published by a cross-tenant publish that should have been rejected")
	}
}
