package service

import (
	"context"
	"testing"
	"time"

	cbtRepo "cbt-api/internal/cbt/repository"
	cbtService "cbt-api/internal/cbt/service"
	"cbt-api/internal/models"
	"cbt-api/internal/sync/dto"
	"cbt-api/internal/sync/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with just the tables
// this test exercises. Production uses AutoMigrate(models.GetAllModels()...)
// against Postgres, whose `default:gen_random_uuid()` GORM tags SQLite's
// DDL parser rejects - so schema here is created with plain SQLite-
// compatible DDL instead, and primary keys are generated client-side in
// the test helpers below rather than relying on a DB default.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// A uniquely-named shared-cache in-memory DB per test: SQLite's
	// connection pool can open more than one connection to the same DSN,
	// and a bare ":memory:" DSN gives each connection its own empty
	// database (data written on one connection would be invisible on the
	// next), so the shared-cache DSN is required - but sharing the exact
	// same DSN string ("file::memory:?cache=shared") across every test in
	// this file would also share the same database, letting the
	// CREATE TABLE calls collide across parallel/sequential subtests.
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	// id columns default to a random hex blob (SQLite's nearest equivalent
	// to Postgres's gen_random_uuid()) so that GORM's Create(), which omits
	// zero-value primary keys from the INSERT column list because the
	// production struct tag says `default:gen_random_uuid()`, still gets a
	// non-empty, non-colliding primary key back for rows this test doesn't
	// seed explicitly (i.e. everything the sync service itself creates).
	ddl := []string{
		`CREATE TABLE exam_attempts (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			start_time DATETIME, end_time DATETIME, score INTEGER, percentage REAL,
			status TEXT DEFAULT 'in_progress', device_info TEXT, ip_address TEXT,
			signature TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE proctoring_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			status TEXT DEFAULT 'active', started_at DATETIME, ended_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE proctoring_violations (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), proctoring_id TEXT NOT NULL, attempt_id TEXT NOT NULL,
			violation_type TEXT, severity TEXT DEFAULT 'warning', details TEXT,
			timestamp DATETIME, created_at DATETIME
		)`,
		`CREATE TABLE exam_event_logs (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			event_type TEXT NOT NULL, client_sequence INTEGER, payload_json TEXT,
			client_timestamp TEXT, created_at DATETIME
		)`,
		`CREATE TABLE sync_operations (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), idempotency_key TEXT NOT NULL UNIQUE, operation_id TEXT,
			type TEXT NOT NULL, record_id TEXT NOT NULL, attempt_id TEXT NOT NULL,
			student_id TEXT NOT NULL, server_version INTEGER DEFAULT 1, server_sequence INTEGER,
			result_json TEXT, processed_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTestSyncService(db *gorm.DB) *SyncService {
	syncRepository := repository.NewSyncRepository(db)
	// examService is unused by the violation/event dispatch paths this
	// test exercises; a real one wired to the same test db is still
	// constructed to match production wiring exactly (see
	// api/routes/routes.go:initSyncHandler).
	examRepo := cbtRepo.NewExamRepository(db)
	questionRepo := cbtRepo.NewQuestionRepository(db)
	examSvc := cbtService.NewExamService(examRepo, questionRepo, db)
	return NewSyncService(syncRepository, examSvc)
}

func seedAttemptWithProctoring(t *testing.T, db *gorm.DB, studentID string) (attemptID string) {
	t.Helper()
	attempt := &models.ExamAttempt{
		ID:        uuid.NewString(),
		StudentID: studentID,
		ExamID:    "exam-1",
		StartTime: time.Now(),
		Status:    "in_progress",
	}
	if err := db.Create(attempt).Error; err != nil {
		t.Fatalf("failed to seed attempt: %v", err)
	}
	proctoring := &models.ProctoringSession{
		ID:        uuid.NewString(),
		AttemptID: attempt.ID,
		StudentID: studentID,
		Status:    "active",
		StartedAt: time.Now(),
	}
	if err := db.Create(proctoring).Error; err != nil {
		t.Fatalf("failed to seed proctoring session: %v", err)
	}
	return attempt.ID
}

func violationRequest(attemptID, idempotencyKey string) *dto.SyncRequest {
	return &dto.SyncRequest{
		OperationID:    "op-1",
		IdempotencyKey: idempotencyKey,
		RecordID:       "record-1",
		SessionID:      attemptID,
		Type:           "violation",
		Payload:        []byte(`{"violationType":"tab_switch","severity":"warning","details":"switched tabs"}`),
	}
}

// TestSync_Idempotency_SecondCallDoesNotDuplicate is the single most
// important behavior in this package: a retried sync request (same
// idempotencyKey, e.g. after the client's retry/backoff logic re-sends it
// following a network blip) must never be processed twice. Getting this
// wrong would mean a violation - or worse, an answer or submission -
// could be silently duplicated or double-counted.
func TestSync_Idempotency_SecondCallDoesNotDuplicate(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestSyncService(db)
	studentID := "student-1"
	attemptID := seedAttemptWithProctoring(t, db, studentID)

	req := violationRequest(attemptID, "idem-key-abc")

	first := svc.Process(context.Background(), studentID, req)
	if first.Outcome != dto.OutcomeAccepted {
		t.Fatalf("expected first call ACCEPTED, got %s (%s: %s)", first.Outcome, first.ErrorCode, first.ErrorMessage)
	}

	second := svc.Process(context.Background(), studentID, req)
	if second.Outcome != dto.OutcomeAlreadyProcessed {
		t.Fatalf("expected second call ALREADY_PROCESSED, got %s (%s: %s)", second.Outcome, second.ErrorCode, second.ErrorMessage)
	}

	var count int64
	if err := db.Model(&models.ProctoringViolation{}).Where("attempt_id = ?", attemptID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count violations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 violation row after a duplicate sync, got %d", count)
	}
}

// TestSync_DifferentIdempotencyKeys_BothProcessed confirms the ledger
// isn't overly aggressive: two genuinely distinct operations (different
// idempotencyKeys) for the same attempt must both go through.
func TestSync_DifferentIdempotencyKeys_BothProcessed(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestSyncService(db)
	studentID := "student-1"
	attemptID := seedAttemptWithProctoring(t, db, studentID)

	first := svc.Process(context.Background(), studentID, violationRequest(attemptID, "idem-key-1"))
	second := svc.Process(context.Background(), studentID, violationRequest(attemptID, "idem-key-2"))

	if first.Outcome != dto.OutcomeAccepted || second.Outcome != dto.OutcomeAccepted {
		t.Fatalf("expected both distinct operations ACCEPTED, got %s and %s", first.Outcome, second.Outcome)
	}

	var count int64
	db.Model(&models.ProctoringViolation{}).Where("attempt_id = ?", attemptID).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 violation rows for 2 distinct operations, got %d", count)
	}
}

// TestSync_OwnershipCheck_RejectsOtherStudentsAttempt is the sync
// package's equivalent of the BE-03 fix: a sync request must never be
// allowed to act on an attempt belonging to a different student, even if
// the caller supplies a valid, otherwise-well-formed idempotencyKey.
func TestSync_OwnershipCheck_RejectsOtherStudentsAttempt(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestSyncService(db)
	ownerID := "student-owner"
	attackerID := "student-attacker"
	attemptID := seedAttemptWithProctoring(t, db, ownerID)

	result := svc.Process(context.Background(), attackerID, violationRequest(attemptID, "idem-key-attack"))

	if result.Outcome != dto.OutcomeRejected || result.ErrorCode != "FORBIDDEN" {
		t.Fatalf("expected REJECTED/FORBIDDEN for cross-student sync attempt, got %s/%s", result.Outcome, result.ErrorCode)
	}

	var count int64
	db.Model(&models.ProctoringViolation{}).Where("attempt_id = ?", attemptID).Count(&count)
	if count != 0 {
		t.Fatalf("expected no violation row to be created for a rejected cross-student sync, got %d", count)
	}
}

// TestSync_UnknownAttempt_RejectsNotFound ensures a sessionId that does
// not correspond to any real ExamAttempt is rejected cleanly rather than
// panicking or silently succeeding.
func TestSync_UnknownAttempt_RejectsNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestSyncService(db)

	result := svc.Process(context.Background(), "student-1", violationRequest("00000000-0000-0000-0000-000000000000", "idem-key-missing"))

	if result.Outcome != dto.OutcomeRejected || result.ErrorCode != "NOT_FOUND" {
		t.Fatalf("expected REJECTED/NOT_FOUND for unknown attempt, got %s/%s", result.Outcome, result.ErrorCode)
	}
}
