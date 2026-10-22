package service

import (
	"context"
	"testing"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/repository"
	"cbt-api/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupTestDB mirrors internal/sync/service/sync_service_test.go's pattern:
// an in-memory SQLite DB with hand-written, SQLite-compatible DDL for just
// the tables NodeSyncService.Push/AuthenticateNode touch (production uses
// AutoMigrate against Postgres, whose gen_random_uuid() default SQLite's
// DDL parser rejects).
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE school_node_credentials (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, label TEXT,
			key_hash TEXT NOT NULL, last_used_at DATETIME, revoked_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, school_id TEXT NOT NULL,
			class_id TEXT, admission_no TEXT, date_of_birth DATETIME, gender TEXT,
			address TEXT, guardian_name TEXT, guardian_phone TEXT, guardian_email TEXT,
			is_active INTEGER DEFAULT 1,
			status TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE exam_attempts (
			id TEXT PRIMARY KEY, student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			start_time DATETIME, end_time DATETIME, score INTEGER, percentage REAL,
			status TEXT, device_info TEXT, ip_address TEXT, signature TEXT,
			synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE student_answers (
			id TEXT PRIMARY KEY, attempt_id TEXT NOT NULL, question_id TEXT NOT NULL,
			selected_answer TEXT, selected_option_keys TEXT, essay_response TEXT,
			is_correct INTEGER DEFAULT 0, is_marked INTEGER DEFAULT 0, time_spent INTEGER,
			synced_at DATETIME, synced_to_cloud_at DATETIME,
			marks_awarded INTEGER, marked_by TEXT, marked_at DATETIME, marker_feedback TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE results (
			id TEXT PRIMARY KEY, student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			attempt_id TEXT, total_score INTEGER, percentage REAL, grade TEXT,
			remarks TEXT, passed INTEGER DEFAULT 0, theory_score INTEGER, fully_graded INTEGER DEFAULT 1, published_at DATETIME,
			synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE exam_event_logs (
			id TEXT PRIMARY KEY, attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			event_type TEXT NOT NULL, client_sequence INTEGER, payload_json TEXT,
			client_timestamp TEXT, synced_to_cloud_at DATETIME, created_at DATETIME
		)`,
		`CREATE TABLE report_card_approvals (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, student_id TEXT NOT NULL,
			report_type TEXT NOT NULL, term_id TEXT DEFAULT '', session_id TEXT NOT NULL,
			status TEXT NOT NULL, version INTEGER DEFAULT 1, snapshot_json TEXT, notes TEXT,
			actor_id TEXT NOT NULL, synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE report_card_audit_logs (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, actor_id TEXT NOT NULL,
			action TEXT NOT NULL, report_type TEXT, student_id TEXT, class_id TEXT,
			term_id TEXT, session_id TEXT, notes TEXT, content_hash TEXT NOT NULL,
			synced_to_cloud_at DATETIME, created_at DATETIME
		)`,
		`CREATE TABLE attendance_records (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, student_id TEXT NOT NULL,
			class_id TEXT NOT NULL, term_id TEXT NOT NULL, session_id TEXT NOT NULL,
			date DATE NOT NULL, status TEXT NOT NULL, marked_by TEXT NOT NULL, notes TEXT,
			synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME,
			UNIQUE(student_id, term_id, date)
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create table: %v\nDDL: %s", err, stmt)
		}
	}
	return db
}

func seedStudent(t *testing.T, db *gorm.DB, studentID, schoolID string) {
	t.Helper()
	s := models.Student{ID: studentID, UserID: uuid.NewString(), SchoolID: schoolID, AdmissionNo: uuid.NewString(), IsActive: true}
	if err := db.Table("students").Create(&s).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func seedCredential(t *testing.T, db *gorm.DB, schoolID, plaintextKey string) {
	t.Helper()
	hash, err := utils.HashPassword(plaintextKey)
	if err != nil {
		t.Fatalf("failed to hash credential: %v", err)
	}
	cred := models.SchoolNodeCredential{ID: uuid.NewString(), SchoolID: schoolID, Label: "test node", KeyHash: hash}
	if err := db.Table("school_node_credentials").Create(&cred).Error; err != nil {
		t.Fatalf("failed to seed credential: %v", err)
	}
}

// TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool is the
// regression test for Push's core security boundary (see the function's
// own comment: "this is what stops one compromised or misconfigured node
// from writing data attributed to another school"). A node authenticated
// as School A must never be able to write a session/result/event for a
// student who belongs to School B, even if it knows that student's real
// ID - this is exactly the "cross-school data leakage" / tenant-isolation
// risk the project's own audit trail (config.go) flags as a known gap
// elsewhere; this at least locks down the one place that already has an
// explicit check so a future refactor can't silently remove it.
func TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	schoolB := uuid.NewString()
	studentInSchoolA := uuid.NewString()
	studentInSchoolB := uuid.NewString()
	seedStudent(t, db, studentInSchoolA, schoolA)
	seedStudent(t, db, studentInSchoolB, schoolB)

	req := &dto.PushRequest{
		Sessions: []dto.SessionPush{
			{ID: uuid.NewString(), StudentID: studentInSchoolA, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"},
			{ID: uuid.NewString(), StudentID: studentInSchoolB, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"},
		},
		Results: []dto.ResultPush{
			{ID: uuid.NewString(), StudentID: studentInSchoolA, ExamID: uuid.NewString(), AttemptID: uuid.NewString(), TotalScore: 10, Grade: "A"},
			{ID: uuid.NewString(), StudentID: studentInSchoolB, ExamID: uuid.NewString(), AttemptID: uuid.NewString(), TotalScore: 10, Grade: "A"},
		},
		Events: []dto.EventPush{
			{ID: uuid.NewString(), AttemptID: uuid.NewString(), StudentID: studentInSchoolA, EventType: "tab_switch"},
			{ID: uuid.NewString(), AttemptID: uuid.NewString(), StudentID: studentInSchoolB, EventType: "tab_switch"},
		},
	}

	resp, err := svc.Push(context.Background(), schoolA, req)
	if err != nil {
		t.Fatalf("Push returned an error: %v", err)
	}

	if resp.SessionsAccepted != 1 {
		t.Errorf("expected exactly 1 session accepted (School A's own), got %d", resp.SessionsAccepted)
	}
	if resp.ResultsAccepted != 1 {
		t.Errorf("expected exactly 1 result accepted (School A's own), got %d", resp.ResultsAccepted)
	}
	if resp.EventsAccepted != 1 {
		t.Errorf("expected exactly 1 event accepted (School A's own), got %d", resp.EventsAccepted)
	}

	if len(resp.Rejected) != 3 {
		t.Fatalf("expected exactly 3 rejected records (School B's session/result/event), got %d: %+v", len(resp.Rejected), resp.Rejected)
	}
	for _, r := range resp.Rejected {
		if r.Reason != "student does not belong to authenticated school" {
			t.Errorf("unexpected rejection reason for %s %s: %q", r.Type, r.ID, r.Reason)
		}
	}

	// Verify School B's session was genuinely never written, not just
	// reported as rejected.
	var count int64
	db.Table("exam_attempts").Where("student_id = ?", studentInSchoolB).Count(&count)
	if count != 0 {
		t.Errorf("School B's session was written to the DB despite being rejected: count=%d", count)
	}
}

// TestAuthenticateNode_RejectsWrongKeyAndAcceptsRight verifies the
// bcrypt-based node credential lookup actually distinguishes keys rather
// than accepting anything, and that a correct key resolves to the right
// school.
func TestAuthenticateNode_RejectsWrongKeyAndAcceptsRight(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	seedCredential(t, db, schoolA, "correct-plaintext-key")

	if _, _, err := svc.AuthenticateNode(context.Background(), "wrong-plaintext-key"); err == nil {
		t.Error("expected AuthenticateNode to reject a wrong key, got no error")
	}

	resolvedSchool, credID, err := svc.AuthenticateNode(context.Background(), "correct-plaintext-key")
	if err != nil {
		t.Fatalf("expected the correct key to authenticate, got error: %v", err)
	}
	if resolvedSchool != schoolA {
		t.Errorf("expected resolved school %q, got %q", schoolA, resolvedSchool)
	}
	if credID == "" {
		t.Error("expected a non-empty credential ID")
	}
}

// TestPush_UpsertIsIdempotent verifies that pushing the identical session
// twice (the real-world case: a node retries a push whose response it
// never received) does not create a duplicate row or error - this is the
// "safe retry and conflict handling during synchronisation" requirement.
func TestPush_UpsertIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)

	session := dto.SessionPush{ID: uuid.NewString(), StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"}
	req := &dto.PushRequest{Sessions: []dto.SessionPush{session}}

	if _, err := svc.Push(context.Background(), schoolA, req); err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if _, err := svc.Push(context.Background(), schoolA, req); err != nil {
		t.Fatalf("retried push of the same session failed: %v", err)
	}

	var count int64
	db.Table("exam_attempts").Where("id = ?", session.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 row after retrying the same push, got %d (not idempotent)", count)
	}
}

// ---- Conflict detection: an incoming push must never silently revert a
// more final locally-stored record. ----

// TestPush_RejectsSessionDowngrade_FinalSubmissionCannotBeReverted is the
// regression test for the directive's explicit requirement: "a final
// exam submission cannot be silently reverted by delayed synchronization."
// A session already recorded as completed must survive a later, stale,
// or out-of-order push that claims the same attempt is still in_progress.
func TestPush_RejectsSessionDowngrade_FinalSubmissionCannotBeReverted(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	attemptID := uuid.NewString()

	completed := dto.SessionPush{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "completed"}
	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Sessions: []dto.SessionPush{completed}}); err != nil {
		t.Fatalf("initial completed push failed: %v", err)
	}

	stale := dto.SessionPush{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"}
	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Sessions: []dto.SessionPush{stale}})
	if err != nil {
		t.Fatalf("stale push returned an error: %v", err)
	}
	if resp.SessionsAccepted != 0 {
		t.Errorf("expected the stale downgrade to be rejected, not accepted, got SessionsAccepted=%d", resp.SessionsAccepted)
	}
	if len(resp.Rejected) != 1 {
		t.Fatalf("expected exactly 1 rejection, got %d: %+v", len(resp.Rejected), resp.Rejected)
	}

	var stored models.ExamAttempt
	if err := db.Table("exam_attempts").Where("id = ?", attemptID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored attempt: %v", err)
	}
	if stored.Status != "completed" {
		t.Errorf("expected the stored attempt to remain completed, got %q - the final submission was reverted", stored.Status)
	}
}

// TestPush_AcceptsSessionStatusAdvance confirms the conflict guard is
// one-directional: a legitimate forward transition (in_progress ->
// completed, the normal case of a node syncing a just-finished exam)
// must still be accepted, not blocked by the new check.
func TestPush_AcceptsSessionStatusAdvance(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	attemptID := uuid.NewString()

	inProgress := dto.SessionPush{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"}
	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Sessions: []dto.SessionPush{inProgress}}); err != nil {
		t.Fatalf("initial in_progress push failed: %v", err)
	}

	completed := dto.SessionPush{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "completed"}
	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Sessions: []dto.SessionPush{completed}})
	if err != nil {
		t.Fatalf("advancing push failed: %v", err)
	}
	if resp.SessionsAccepted != 1 {
		t.Fatalf("expected the forward transition to be accepted, got SessionsAccepted=%d, rejected=%+v", resp.SessionsAccepted, resp.Rejected)
	}

	var stored models.ExamAttempt
	if err := db.Table("exam_attempts").Where("id = ?", attemptID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored attempt: %v", err)
	}
	if stored.Status != "completed" {
		t.Errorf("expected stored status completed, got %q", stored.Status)
	}
}

// TestPush_RejectsAnswerConflict_AttemptAlreadyFinalized confirms an
// answer push with a DIFFERENT value than what's stored is rejected once
// its attempt has moved past in_progress - "prevent an older answer or
// submission from overwriting a newer authoritative record."
func TestPush_RejectsAnswerConflict_AttemptAlreadyFinalized(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	attemptID := uuid.NewString()
	answerID := uuid.NewString()
	questionID := uuid.NewString()

	// Attempt completed, with an answer already recorded as "A".
	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{
		Sessions: []dto.SessionPush{{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "completed"}},
		Answers:  []dto.AnswerPush{{ID: answerID, AttemptID: attemptID, QuestionID: questionID, SelectedAnswer: "A", IsCorrect: true}},
	}); err != nil {
		t.Fatalf("initial push failed: %v", err)
	}

	// A delayed/out-of-order push now claims a different answer for the
	// same ID - must be rejected, not silently applied.
	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{
		Answers: []dto.AnswerPush{{ID: answerID, AttemptID: attemptID, QuestionID: questionID, SelectedAnswer: "B", IsCorrect: false}},
	})
	if err != nil {
		t.Fatalf("conflicting answer push returned an error: %v", err)
	}
	if resp.AnswersAccepted != 0 {
		t.Errorf("expected the conflicting answer to be rejected, got AnswersAccepted=%d", resp.AnswersAccepted)
	}

	var stored models.StudentAnswer
	if err := db.Table("student_answers").Where("id = ?", answerID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored answer: %v", err)
	}
	if stored.SelectedAnswer != "A" {
		t.Errorf("expected the stored answer to remain %q, got %q - a finalized attempt's answer was silently overwritten", "A", stored.SelectedAnswer)
	}
}

// TestPush_AcceptsIdenticalAnswerRetry_AttemptAlreadyFinalized confirms
// the conflict guard doesn't break legitimate retries: re-pushing the
// exact same answer value for an already-finalized attempt (a node
// resending a batch whose response it never received) must still succeed.
func TestPush_AcceptsIdenticalAnswerRetry_AttemptAlreadyFinalized(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	attemptID := uuid.NewString()
	answerID := uuid.NewString()
	questionID := uuid.NewString()

	answer := dto.AnswerPush{ID: answerID, AttemptID: attemptID, QuestionID: questionID, SelectedAnswer: "A", IsCorrect: true}
	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{
		Sessions: []dto.SessionPush{{ID: attemptID, StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "completed"}},
		Answers:  []dto.AnswerPush{answer},
	}); err != nil {
		t.Fatalf("initial push failed: %v", err)
	}

	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Answers: []dto.AnswerPush{answer}})
	if err != nil {
		t.Fatalf("retried identical answer push returned an error: %v", err)
	}
	if resp.AnswersAccepted != 1 {
		t.Errorf("expected the identical retry to be accepted as a no-op, got AnswersAccepted=%d, rejected=%+v", resp.AnswersAccepted, resp.Rejected)
	}
}

// TestPush_RejectsResultOverwrite_FullyGradedWithDifferentValues confirms
// an already-fully-graded Result (the authoritative outcome of either
// pure objective grading or completed manual marking) resists being
// silently overwritten by a push carrying different score data.
func TestPush_RejectsResultOverwrite_FullyGradedWithDifferentValues(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	resultID := uuid.NewString()
	attemptID := uuid.NewString()
	examID := uuid.NewString()

	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{
		Results: []dto.ResultPush{{ID: resultID, StudentID: studentID, ExamID: examID, AttemptID: attemptID, TotalScore: 80, Grade: "A", Passed: true, FullyGraded: true}},
	}); err != nil {
		t.Fatalf("initial result push failed: %v", err)
	}

	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{
		Results: []dto.ResultPush{{ID: resultID, StudentID: studentID, ExamID: examID, AttemptID: attemptID, TotalScore: 40, Grade: "C", Passed: false, FullyGraded: true}},
	})
	if err != nil {
		t.Fatalf("conflicting result push returned an error: %v", err)
	}
	if resp.ResultsAccepted != 0 {
		t.Errorf("expected the conflicting result to be rejected, got ResultsAccepted=%d", resp.ResultsAccepted)
	}

	var stored models.Result
	if err := db.Table("results").Where("id = ?", resultID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored result: %v", err)
	}
	if stored.TotalScore != 80 {
		t.Errorf("expected the stored score to remain 80, got %d - a fully-graded result was silently overwritten", stored.TotalScore)
	}
}

// TestPush_AcceptsResultRetry_FullyGradedWithIdenticalValues is the
// idempotency companion: re-pushing the exact same fully-graded result
// must succeed, not be mistaken for a conflict.
func TestPush_AcceptsResultRetry_FullyGradedWithIdenticalValues(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)
	result := dto.ResultPush{ID: uuid.NewString(), StudentID: studentID, ExamID: uuid.NewString(), AttemptID: uuid.NewString(), TotalScore: 80, Grade: "A", Passed: true, FullyGraded: true}

	if _, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Results: []dto.ResultPush{result}}); err != nil {
		t.Fatalf("initial result push failed: %v", err)
	}

	resp, err := svc.Push(context.Background(), schoolA, &dto.PushRequest{Results: []dto.ResultPush{result}})
	if err != nil {
		t.Fatalf("retried identical result push returned an error: %v", err)
	}
	if resp.ResultsAccepted != 1 {
		t.Errorf("expected the identical retry to be accepted as a no-op, got ResultsAccepted=%d, rejected=%+v", resp.ResultsAccepted, resp.Rejected)
	}
}
