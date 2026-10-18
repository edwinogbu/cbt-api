package repository

import (
	"cbt-api/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type SyncRepository struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) *SyncRepository {
	return &SyncRepository{db: db}
}

// FindByIdempotencyKey looks up a previously processed sync operation.
// Returns (nil, nil) - not an error - when no such operation exists yet.
func (r *SyncRepository) FindByIdempotencyKey(idempotencyKey string) (*models.SyncOperation, error) {
	var op models.SyncOperation
	err := r.db.Where("idempotency_key = ?", idempotencyKey).First(&op).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &op, nil
}

// RecordOperation durably records a successfully processed sync operation
// so a retried request with the same idempotencyKey is detected. Relies on
// the unique index on idempotency_key: a concurrent duplicate insert (e.g.
// two retries racing) fails here rather than double-processing silently -
// the caller should treat that as ALREADY_PROCESSED and re-fetch.
func (r *SyncRepository) RecordOperation(op *models.SyncOperation) error {
	op.ProcessedAt = time.Now()
	return r.db.Create(op).Error
}

// FindAttemptOwner returns the studentID that owns the given exam attempt,
// for the same ownership check pattern used by internal/cbt/service
// (see BE-03 in the security audit: never trust a client-supplied
// student/attempt pairing without verifying it server-side).
func (r *SyncRepository) FindAttemptOwner(attemptID string) (studentID string, err error) {
	var attempt models.ExamAttempt
	err = r.db.Select("student_id").Where("id = ? AND deleted_at IS NULL", attemptID).First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return attempt.StudentID, nil
}

// FindProctoringSessionID returns the ProctoringSession.ID for an attempt,
// required as ProctoringViolation.ProctoringID (not-null FK). Returns
// ("", nil) - not an error - if the attempt has no proctoring session.
func (r *SyncRepository) FindProctoringSessionID(attemptID string) (string, error) {
	var session models.ProctoringSession
	err := r.db.Select("id").Where("attempt_id = ?", attemptID).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return session.ID, nil
}

// SaveViolation inserts a proctoring violation record. Reused by both the
// sync endpoint (type="violation") and the real-time violation endpoint,
// so violations are recorded identically regardless of which path a client
// happens to be online for at the time.
func (r *SyncRepository) SaveViolation(v *models.ProctoringViolation) error {
	return r.db.Create(v).Error
}

// SaveEvent inserts an audit/event-log entry from the client's exam-taking
// event stream. Best-effort, never used for grading.
func (r *SyncRepository) SaveEvent(e *models.ExamEventLog) error {
	return r.db.Create(e).Error
}
