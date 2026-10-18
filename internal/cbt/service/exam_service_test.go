package service

import (
	"context"
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
			duration_minutes INTEGER NOT NULL, total_marks INTEGER NOT NULL, pass_mark INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE exam_attempts (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			start_time DATETIME, end_time DATETIME, score INTEGER, percentage REAL,
			status TEXT DEFAULT 'in_progress', device_info TEXT, ip_address TEXT,
			signature TEXT, synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE question_bank (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			correct_answer TEXT, marks INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE student_answers (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, question_id TEXT NOT NULL,
			selected_answer TEXT, selected_option_keys TEXT, essay_response TEXT,
			is_correct INTEGER DEFAULT 0, is_marked INTEGER DEFAULT 0, time_spent INTEGER,
			synced_at DATETIME, synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE results (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			attempt_id TEXT NOT NULL, total_score INTEGER, percentage REAL, grade TEXT, remarks TEXT,
			passed INTEGER DEFAULT 0, published_at DATETIME, synced_to_cloud_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE proctoring_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			status TEXT DEFAULT 'active', started_at DATETIME, ended_at DATETIME,
			created_at DATETIME, updated_at DATETIME
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
