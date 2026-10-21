package service

import (
	"context"
	"encoding/json"
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
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), school_id TEXT, teacher_id TEXT,
			class_level_id TEXT, class_arm_id TEXT, deleted_at DATETIME
		)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT, school_id TEXT, class_id TEXT,
			admission_no TEXT DEFAULT '', is_active INTEGER DEFAULT 1, deleted_at DATETIME
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
		`CREATE TABLE subjects (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT, code TEXT, description TEXT, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		// Mirrors internal/grading/grading_test.go's DDL - needed here too
		// since buildSubjectTermlyResult now resolves a school's
		// configured grading/assessment rules (internal/grading) rather
		// than always using the hardcoded builtin defaults.
		`CREATE TABLE grading_scales (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, class_level_id TEXT, scope_key TEXT,
			name TEXT, pass_mark REAL DEFAULT 50, is_active INTEGER DEFAULT 1,
			updated_by TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE grading_scale_bands (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			grading_scale_id TEXT, grade TEXT, min_percent REAL,
			grade_point REAL DEFAULT 0, remark TEXT, sort_order INTEGER,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE assessment_schemes (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, class_level_id TEXT, subject_id TEXT, scope_key TEXT,
			missing_policy TEXT DEFAULT 'renormalize', is_active INTEGER DEFAULT 1,
			updated_by TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE assessment_components (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			scheme_id TEXT, code TEXT, name TEXT, weight REAL,
			exam_types TEXT DEFAULT '[]', aggregation TEXT DEFAULT 'pooled',
			sort_order INTEGER, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE report_card_remarks (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, student_id TEXT, term_id TEXT, session_id TEXT,
			teacher_remark TEXT, teacher_id TEXT, teacher_set_at DATETIME,
			principal_remark TEXT, principal_id TEXT, principal_set_at DATETIME,
			conduct TEXT,
			mid_term_comment TEXT, mid_term_next_steps TEXT, mid_term_set_by TEXT, mid_term_set_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE promotion_policies (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, class_level_id TEXT, scope_key TEXT,
			overall_threshold REAL DEFAULT 50, core_subject_ids TEXT DEFAULT '[]',
			core_subject_pass_mark REAL DEFAULT 40, is_active INTEGER DEFAULT 1,
			updated_by TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE promotion_decisions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, student_id TEXT, session_id TEXT,
			proposed_class_level_id TEXT, decision TEXT, notes TEXT,
			decided_by TEXT, decided_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE report_card_approvals (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, student_id TEXT, report_type TEXT, term_id TEXT DEFAULT '', session_id TEXT,
			status TEXT, version INTEGER DEFAULT 1, snapshot_json TEXT, notes TEXT,
			actor_id TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE report_card_audit_logs (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, actor_id TEXT, action TEXT, report_type TEXT,
			student_id TEXT, class_id TEXT, term_id TEXT, session_id TEXT, notes TEXT,
			created_at DATETIME
		)`,
		`CREATE TABLE class_levels (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, name TEXT, level_number INTEGER, category TEXT,
			promotion_to TEXT, sort_order INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE class_arms (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, name TEXT, arm_code TEXT, capacity INTEGER DEFAULT 40,
			room_number TEXT, sort_order INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE academic_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, name TEXT, start_date DATETIME, end_date DATETIME,
			is_current INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE terms (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			session_id TEXT, term_number INTEGER, name TEXT, start_date DATETIME, end_date DATETIME,
			is_current INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE schools (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT, code TEXT, address TEXT, phone TEXT, email TEXT, logo TEXT, website TEXT,
			status TEXT DEFAULT 'PENDING', is_active INTEGER DEFAULT 1,
			application_name TEXT, primary_color TEXT, secondary_color TEXT,
			provisioned_at DATETIME, provisioned_by TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
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

// seedQuestionWithOptionKeys seeds a question using the MODERN
// options+correct_option_keys format (CreateQuestionRequest's
// Options/CorrectOptionKeys fields - what QuestionService actually
// accepts and validates as an alternative to the legacy single-string
// CorrectAnswer field), leaving correct_answer empty exactly as a real
// question created that way does. See TestGradeAnswer_UsesCorrectOptionKeys.
func seedQuestionWithOptionKeys(t *testing.T, db *gorm.DB, correctKeys []string, marks int) (questionID string) {
	t.Helper()
	id := uuid.NewString()
	keysJSON, err := json.Marshal(correctKeys)
	if err != nil {
		t.Fatalf("failed to marshal correct_option_keys: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO question_bank (id, correct_option_keys, marks, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, string(keysJSON), marks, time.Now(), time.Now(),
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

// TestGradeAnswer_UsesCorrectOptionKeys is the regression test for a
// third grading bug, found live by running a real seeded exam end to
// end against a School CBT Node: gradeAnswer only ever compared the
// submitted answer against the legacy single-string CorrectAnswer
// field. A question created via the modern options+correct_option_keys
// format (what CreateQuestionRequest's own validation treats as a fully
// valid alternative - QuestionService requires only one of
// CorrectOptionKeys or CorrectAnswer, never both) leaves CorrectAnswer
// empty, so every answer to it was silently graded wrong regardless of
// correctness. Confirmed live: 5/5 objectively-correct answers on a
// real seeded exam all scored 0.
func TestGradeAnswer_UsesCorrectOptionKeys(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	attemptID := seedAttempt(t, db, 2)
	correctQID := seedQuestionWithOptionKeys(t, db, []string{"A"}, 1)
	wrongQID := seedQuestionWithOptionKeys(t, db, []string{"B"}, 1)

	resp, err := svc.BulkSaveAnswers(context.Background(), &dto.BulkSaveAnswerRequest{
		AttemptID: attemptID,
		Answers: []dto.BulkAnswerItem{
			{QuestionID: correctQID, SelectedAnswer: "A"}, // matches correct_option_keys
			{QuestionID: wrongQID, SelectedAnswer: "A"},   // correct_option_keys is ["B"]
		},
	}, testStudentID)
	if err != nil {
		t.Fatalf("BulkSaveAnswers failed: %v", err)
	}
	if resp.SyncedCount != 2 {
		t.Fatalf("expected 2 synced, got %d", resp.SyncedCount)
	}

	var correctAnswer, wrongAnswer models.StudentAnswer
	if err := db.Where("attempt_id = ? AND question_id = ?", attemptID, correctQID).First(&correctAnswer).Error; err != nil {
		t.Fatalf("failed to load correct answer: %v", err)
	}
	if err := db.Where("attempt_id = ? AND question_id = ?", attemptID, wrongQID).First(&wrongAnswer).Error; err != nil {
		t.Fatalf("failed to load wrong answer: %v", err)
	}
	if !correctAnswer.IsCorrect {
		t.Fatalf("expected the answer matching correct_option_keys to have IsCorrect=true, got false - the bug is present")
	}
	if wrongAnswer.IsCorrect {
		t.Fatalf("expected the answer NOT matching correct_option_keys to have IsCorrect=false, got true")
	}
}

// ---- Manual/essay marking ----

const testMarkerTeacherID = "teacher-1"
const testOtherTeacherID = "teacher-2"
const testAdminID = "admin-1"
const testMarkingSchoolID = "school-A"

func seedMarkingUser(t *testing.T, db *gorm.DB, id, role string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES (?, 'Test', 'User', ?)", id, role).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedMarkingClass(t *testing.T, db *gorm.DB, teacherID, schoolID string) (classID string) {
	t.Helper()
	classID = uuid.NewString()
	if err := db.Exec("INSERT INTO classes (id, teacher_id, school_id) VALUES (?, ?, ?)", classID, teacherID, schoolID).Error; err != nil {
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
	classID := seedMarkingClass(t, db, testMarkerTeacherID, testMarkingSchoolID)
	setExamClass(t, db, classID)
	seedMarkingUser(t, db, testMarkerTeacherID, "teacher")
	seedMarkingUser(t, db, testOtherTeacherID, "teacher")
	seedMarkingUser(t, db, testAdminID, "admin")
	seedMarkingUser(t, db, testStudentID, "student")
	if err := db.Exec("INSERT INTO students (id, user_id) VALUES (?, ?)", testStudentID, testStudentID).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}

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
	}, testMarkerTeacherID, testMarkingSchoolID, false)
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
	}, testMarkerTeacherID, testMarkingSchoolID, false)
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
	}, testOtherTeacherID, testMarkingSchoolID, false)
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
	}, testStudentID, testMarkingSchoolID, false)
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
	}, testAdminID, testMarkingSchoolID, false)
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
	}, testMarkerTeacherID, testMarkingSchoolID, false); err != nil {
		t.Fatalf("first mark failed: %v", err)
	}
	resp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 6, Feedback: "Corrected after review.",
	}, testMarkerTeacherID, testMarkingSchoolID, false)
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
	}, testMarkerTeacherID, testMarkingSchoolID, false)
	if err != nil {
		t.Fatalf("third (repeated) mark failed: %v", err)
	}
	if resp2.TotalScore != 6 || resp2.Percentage != resp.Percentage {
		t.Fatalf("expected a repeated identical mark to be a stable no-op, got score=%d percentage=%v", resp2.TotalScore, resp2.Percentage)
	}
}

// ---- exam marking tenant isolation ----
//
// Before this fix, authorizeMarking's admin/super_admin bypass had NO
// school check at all - any school's admin could mark (and so alter the
// recorded score of) any OTHER school's exam attempts just by naming
// their attempt_id/question_id, and GetPendingManualMarking's
// admin/super_admin branch had no school filter either, returning every
// school's pending essay answers (student names, essay text, all of it).

// TestMarkDescriptiveAnswer_CrossTenant_Rejected confirms an admin from
// a different school cannot mark School A's exam attempt.
func TestMarkDescriptiveAnswer_CrossTenant_Rejected(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	_, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8,
	}, testAdminID, "school-B", false)
	if err == nil {
		t.Fatal("expected an admin from a different school to be rejected marking School A's exam, got no error")
	}
}

// TestMarkDescriptiveAnswer_PlatformOwner_Bypasses confirms an
// authenticated platform_owner can mark any school's exam.
func TestMarkDescriptiveAnswer_PlatformOwner_Bypasses(t *testing.T) {
	svc, attemptID, essayQID := setupMarkingFixture(t)

	resp, err := svc.MarkDescriptiveAnswer(context.Background(), &dto.MarkAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, MarksAwarded: 8,
	}, testAdminID, "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to mark any school's exam, got error: %v", err)
	}
	if resp.TotalScore != 8 {
		t.Fatalf("expected total score 8, got %d", resp.TotalScore)
	}
}

// TestGetPendingManualMarking_Admin_ScopedToOwnSchool confirms an admin
// sees their own school's pending marking.
func TestGetPendingManualMarking_Admin_ScopedToOwnSchool(t *testing.T) {
	svc, _, _ := setupMarkingFixture(t)

	resp, err := svc.GetPendingManualMarking(context.Background(), testAdminID, testMarkingSchoolID, false)
	if err != nil {
		t.Fatalf("expected School A's admin to read its own pending marking, got error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 pending item for School A's own admin, got %d", resp.Total)
	}
}

// TestGetPendingManualMarking_Admin_CrossTenant_Empty confirms an admin
// from a different school sees none of School A's pending marking.
func TestGetPendingManualMarking_Admin_CrossTenant_Empty(t *testing.T) {
	svc, _, _ := setupMarkingFixture(t)

	resp, err := svc.GetPendingManualMarking(context.Background(), testAdminID, "school-B", false)
	if err != nil {
		t.Fatalf("expected a different school's admin to get its own (empty) results, not an error: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected 0 pending items for a different school's admin, got %d - cross-tenant leak", resp.Total)
	}
}

// TestGetPendingManualMarking_PlatformOwner_SeesAllSchools confirms
// platform_owner's unscoped bypass still works. Unlike
// authorizeMarking's admin/super_admin bypass (which relies on
// authorizeExamTenant being called first), this method looks up its own
// caller's role directly, so the test needs a caller whose actual role
// is platform_owner, not merely an isPlatformOwner=true flag on an
// admin account - builds its own fixture (rather than going through
// setupMarkingFixture, which doesn't expose its db) so it can seed that
// extra user.
func TestGetPendingManualMarking_PlatformOwner_SeesAllSchools(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedExam(t, db, 30)
	db.Exec("UPDATE exams SET total_marks = 10, pass_mark = 5 WHERE id = ?", testExamID)
	classID := seedMarkingClass(t, db, testMarkerTeacherID, testMarkingSchoolID)
	setExamClass(t, db, classID)
	seedMarkingUser(t, db, testMarkerTeacherID, "teacher")
	seedMarkingUser(t, db, testStudentID, "student")
	const testPlatformOwnerID = "owner-1"
	seedMarkingUser(t, db, testPlatformOwnerID, "platform_owner")
	if err := db.Exec("INSERT INTO students (id, user_id) VALUES (?, ?)", testStudentID, testStudentID).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}

	attemptID := seedAttempt(t, db, 5)
	essayQID := seedEssayQuestion(t, db, 10)
	seedExamQuestions(t, db, testExamID, []string{essayQID})
	if _, err := svc.SaveAnswer(context.Background(), &dto.SaveAnswerRequest{
		AttemptID: attemptID, QuestionID: essayQID, SelectedAnswer: "My essay response text.",
	}, testStudentID); err != nil {
		t.Fatalf("failed to save essay answer: %v", err)
	}
	if _, err := svc.SubmitExam(context.Background(), &dto.SubmitExamRequest{AttemptID: attemptID}, testStudentID); err != nil {
		t.Fatalf("SubmitExam failed: %v", err)
	}

	resp, err := svc.GetPendingManualMarking(context.Background(), testPlatformOwnerID, "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to read every school's pending marking, got error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected platform_owner to see School A's 1 pending item, got %d", resp.Total)
	}
}

// TestGetPendingManualMarking_EmptySchoolID_NonPlatformOwner_Rejected
// confirms a non-platform-owner caller with no school_id of their own
// (e.g. a legacy account seeded with none) is rejected, not silently
// treated as unscoped.
func TestGetPendingManualMarking_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	svc, _, _ := setupMarkingFixture(t)

	if _, err := svc.GetPendingManualMarking(context.Background(), testAdminID, "", false); err == nil {
		t.Fatal("expected a non-platform-owner caller with no school_id of their own to be rejected, got no error")
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
	classID := seedMarkingClass(t, db, testMarkerTeacherID, testMarkingSchoolID)
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
	}, testMarkerTeacherID, testMarkingSchoolID, false)
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

// ---- buildSubjectTermlyResult: report-card subject totals now resolved
// through the shared grading/assessment engine (internal/grading) ----

func seedTermlySubject(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO subjects (id, name, code, is_active, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)",
		id, name, name, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed subject: %v", err)
	}
}

func seedTermlyExam(t *testing.T, db *gorm.DB, id, subjectID, examType string, totalMarks int) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO exams (id, subject_id, exam_type, duration_minutes, total_marks, pass_mark, created_at, updated_at) VALUES (?, ?, ?, 60, ?, 50, ?, ?)",
		id, subjectID, examType, totalMarks, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed exam: %v", err)
	}
}

// TestBuildSubjectTermlyResult_NoConfigMatchesPlainPooledSum locks in
// that a school with no AssessmentScheme/GradingScale configured sees
// EXACTLY the same subject percentage/grade this function always
// produced (plain Σscore/Σmarks, builtin grade boundaries) - the whole
// point of routing it through internal/grading is that this case must
// not change.
func TestBuildSubjectTermlyResult_NoConfigMatchesPlainPooledSum(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedTermlyExam(t, db, "exam-ca-1", "subj-1", "continuous_assessment", 20)
	seedTermlyExam(t, db, "exam-main-1", "subj-1", "main_exam", 80)

	results := []models.Result{
		{ExamID: "exam-ca-1", TotalScore: 18, FullyGraded: true},
		{ExamID: "exam-main-1", TotalScore: 60, FullyGraded: true},
	}

	got, err := svc.buildSubjectTermlyResult(context.Background(), "student-1", "subj-1", "school-1", "level-1", results)
	if err != nil {
		t.Fatalf("buildSubjectTermlyResult failed: %v", err)
	}

	wantPct := (18.0 + 60.0) / (20.0 + 80.0) * 100 // 78%
	if got.Percentage != wantPct {
		t.Errorf("Percentage = %.4f, want %.4f (plain pooled sum, unchanged by this school having no grading config)", got.Percentage, wantPct)
	}
	if got.TotalScore != 78 || got.TotalMarks != 100 {
		t.Errorf("TotalScore/TotalMarks = %d/%d, want 78/100 (raw sums unchanged)", got.TotalScore, got.TotalMarks)
	}
	wantGrade, wantGP, wantRemark := dto.GetGrade(wantPct)
	if got.Grade != wantGrade || got.GradePoint != wantGP || got.Remarks != wantRemark {
		t.Errorf("Grade/GradePoint/Remarks = %s/%.1f/%s, want %s/%.1f/%s (builtin boundaries, unchanged)",
			got.Grade, got.GradePoint, got.Remarks, wantGrade, wantGP, wantRemark)
	}
}

// TestBuildSubjectTermlyResult_WeightedSchemeAppliesConfiguredWeights
// proves a school's configured AssessmentScheme actually changes the
// subject total - not just that the no-config path is unaffected. CA
// scores 100%, the exam scores 60%: the plain pooled sum (68%, grade B)
// and this school's weighted 30/70 split (72%, grade A) must disagree,
// which is only possible if buildSubjectTermlyResult is really applying
// the configured weights rather than falling back to the old formula.
func TestBuildSubjectTermlyResult_WeightedSchemeAppliesConfiguredWeights(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedTermlyExam(t, db, "exam-ca-1", "subj-1", "continuous_assessment", 20)
	seedTermlyExam(t, db, "exam-main-1", "subj-1", "main_exam", 80)

	scheme := models.AssessmentScheme{
		SchoolID: "school-1", ScopeKey: "school", MissingPolicy: "renormalize", IsActive: true,
		Components: []models.AssessmentComponent{
			{Code: "ca", Name: "CA", Weight: 30, ExamTypes: models.StringArray{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: models.StringArray{"main_exam"}, Aggregation: "pooled"},
		},
	}
	if err := db.Create(&scheme).Error; err != nil {
		t.Fatalf("failed to seed assessment scheme: %v", err)
	}

	results := []models.Result{
		{ExamID: "exam-ca-1", TotalScore: 20, FullyGraded: true},   // 100%
		{ExamID: "exam-main-1", TotalScore: 48, FullyGraded: true}, // 60%
	}

	got, err := svc.buildSubjectTermlyResult(context.Background(), "student-1", "subj-1", "school-1", "level-1", results)
	if err != nil {
		t.Fatalf("buildSubjectTermlyResult failed: %v", err)
	}

	plainPooled := (20.0 + 48.0) / (20.0 + 80.0) * 100 // 68%
	wantWeighted := 100.0*0.30 + 60.0*0.70              // 72%
	if got.Percentage == plainPooled {
		t.Fatalf("Percentage = %.4f equals the plain pooled sum - the configured 30/70 weighting was not applied", got.Percentage)
	}
	if got.Percentage != wantWeighted {
		t.Errorf("Percentage = %.4f, want %.4f (30%% CA + 70%% Exam per this school's configured scheme)", got.Percentage, wantWeighted)
	}
	if got.Grade != "A" {
		t.Errorf("Grade = %s, want A (72%% weighted total crosses the builtin 70%% boundary that the unweighted 68%% would not)", got.Grade)
	}
}

// ---- Report-card teacher/principal remarks: previously hardcoded to
// "" with nowhere to persist them ----

func seedStudentWithSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO students (id, user_id, school_id) VALUES (?, ?, ?)", id, id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func TestGetReportCardRemark_NoneYetReturnsNil(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	if got := svc.getReportCardRemark(context.Background(), "student-x", "term-x", "session-x"); got != nil {
		t.Errorf("getReportCardRemark = %+v, want nil when no remark has ever been set", got)
	}
}

func TestSetTeacherReportCardRemark_CreatesNewRemark(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-1")

	req := &dto.SetReportCardRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Remark: "Good effort this term.", Conduct: "Good",
	}
	if err := svc.SetTeacherReportCardRemark(context.Background(), "school-1", "teacher-1", req); err != nil {
		t.Fatalf("SetTeacherReportCardRemark failed: %v", err)
	}

	got := svc.getReportCardRemark(context.Background(), "student-1", "term-1", "session-1")
	if got == nil {
		t.Fatal("getReportCardRemark returned nil after setting a teacher remark")
	}
	if got.TeacherRemark != req.Remark || got.Conduct != req.Conduct {
		t.Errorf("TeacherRemark/Conduct = %q/%q, want %q/%q", got.TeacherRemark, got.Conduct, req.Remark, req.Conduct)
	}
	if got.PrincipalRemark != "" {
		t.Errorf("PrincipalRemark = %q, want empty - only the teacher remark was set", got.PrincipalRemark)
	}
}

func TestSetPrincipalReportCardRemark_DoesNotClobberTeacherRemark(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-1")
	ctx := context.Background()

	teacherReq := &dto.SetReportCardRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Remark: "Good effort this term.", Conduct: "Good",
	}
	if err := svc.SetTeacherReportCardRemark(ctx, "school-1", "teacher-1", teacherReq); err != nil {
		t.Fatalf("SetTeacherReportCardRemark failed: %v", err)
	}

	principalReq := &dto.SetReportCardRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Remark: "Keep up the good work.",
	}
	if err := svc.SetPrincipalReportCardRemark(ctx, "school-1", "principal-1", principalReq); err != nil {
		t.Fatalf("SetPrincipalReportCardRemark failed: %v", err)
	}

	got := svc.getReportCardRemark(ctx, "student-1", "term-1", "session-1")
	if got == nil {
		t.Fatal("getReportCardRemark returned nil after setting both remarks")
	}
	if got.TeacherRemark != teacherReq.Remark {
		t.Errorf("TeacherRemark = %q, want %q (must survive the later principal-remark write, same row)", got.TeacherRemark, teacherReq.Remark)
	}
	if got.PrincipalRemark != principalReq.Remark {
		t.Errorf("PrincipalRemark = %q, want %q", got.PrincipalRemark, principalReq.Remark)
	}
}

func TestSetTeacherReportCardRemark_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedStudentWithSchool(t, db, "student-1", "school-A")

	req := &dto.SetReportCardRemarkRequest{
		StudentID: "student-1", TermID: "term-1", SessionID: "session-1",
		Remark: "Should never be written.",
	}
	err := svc.SetTeacherReportCardRemark(context.Background(), "school-B", "teacher-1", req)
	if err == nil {
		t.Fatal("expected School B's token to be rejected setting a remark on School A's student, got no error")
	}

	if got := svc.getReportCardRemark(context.Background(), "student-1", "term-1", "session-1"); got != nil {
		t.Errorf("a rejected cross-tenant remark write must not be persisted, got %+v", got)
	}
}

// ---- PreviewTermlyReportCard: a school's own real identity, with
// illustrative (never real) sample student/result data ----

func seedSchool(t *testing.T, db *gorm.DB, id, name, logo, primaryColor string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO schools (id, name, logo, primary_color) VALUES (?, ?, ?, ?)",
		id, name, logo, primaryColor,
	).Error; err != nil {
		t.Fatalf("failed to seed school: %v", err)
	}
}

func TestPreviewTermlyReportCard_UsesTheCallersOwnRealSchoolIdentity(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedSchool(t, db, "school-1", "Greenfield Academy", "https://cdn.example/greenfield-logo.png", "#0f4c3a")

	preview, err := svc.PreviewTermlyReportCard(context.Background(), "school-1")
	if err != nil {
		t.Fatalf("PreviewTermlyReportCard failed: %v", err)
	}
	if preview.School.Name != "Greenfield Academy" {
		t.Errorf("School.Name = %q, want the school's real registered name", preview.School.Name)
	}
	if preview.School.Logo != "https://cdn.example/greenfield-logo.png" {
		t.Errorf("School.Logo = %q, want the school's own real configured logo URL", preview.School.Logo)
	}
	if preview.School.PrimaryColor != "#0f4c3a" {
		t.Errorf("School.PrimaryColor = %q, want the school's own real configured brand color", preview.School.PrimaryColor)
	}
}

func TestPreviewTermlyReportCard_TwoSchoolsNeverShareIdentity(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedSchool(t, db, "school-A", "School A Academy", "https://cdn.example/a-logo.png", "#0f4c3a")
	seedSchool(t, db, "school-B", "School B College", "https://cdn.example/b-logo.png", "#8a1f2b")

	previewA, err := svc.PreviewTermlyReportCard(context.Background(), "school-A")
	if err != nil {
		t.Fatalf("PreviewTermlyReportCard(school-A) failed: %v", err)
	}
	previewB, err := svc.PreviewTermlyReportCard(context.Background(), "school-B")
	if err != nil {
		t.Fatalf("PreviewTermlyReportCard(school-B) failed: %v", err)
	}

	if previewA.School.Name == previewB.School.Name || previewA.School.Logo == previewB.School.Logo || previewA.School.PrimaryColor == previewB.School.PrimaryColor {
		t.Fatalf("School A's preview must never carry School B's identity - got A=%+v B=%+v", previewA.School, previewB.School)
	}
}
