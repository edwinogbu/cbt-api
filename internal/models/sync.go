package models

import (
	"time"

	"gorm.io/gorm"
)

// SyncOperation is the durable idempotency ledger for the offline-first
// sync protocol (POST /api/v1/sync/:type). Every accepted sync request is
// recorded here keyed by its client-generated idempotencyKey, so a retried
// or replayed request (e.g. after a network blip during the LAN/Cloud sync
// engine's retry loop) is detected and returns the original result instead
// of being reprocessed.
type SyncOperation struct {
	ID             string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	IdempotencyKey string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"idempotency_key"`
	OperationID    string         `gorm:"type:varchar(255);index" json:"operation_id"`
	Type           string         `gorm:"type:varchar(50);not null;index" json:"type"`
	RecordID       string         `gorm:"type:varchar(255);not null" json:"record_id"`
	AttemptID      string         `gorm:"type:uuid;not null;index" json:"attempt_id"`
	StudentID      string         `gorm:"type:uuid;not null;index" json:"student_id"`
	ServerVersion  int            `gorm:"type:integer;not null;default:1" json:"server_version"`
	ServerSequence *int           `gorm:"type:integer" json:"server_sequence"`
	ResultJSON     string         `gorm:"type:jsonb" json:"result_json"`
	ProcessedAt    time.Time      `json:"processed_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SyncOperation) TableName() string { return "sync_operations" }

// ExamEventLog durably stores the client-side exam-taking event stream
// (answer saved, navigation, network lost/restored, checkpoint, etc.)
// synced from the offline-first Dexie event log, for audit and
// conflict-resolution purposes only. Never used for grading.
type ExamEventLog struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID       string    `gorm:"type:uuid;not null;index" json:"attempt_id"`
	StudentID       string    `gorm:"type:uuid;not null;index" json:"student_id"`
	EventType       string    `gorm:"type:varchar(50);not null" json:"event_type"`
	ClientSequence  int       `gorm:"type:integer" json:"client_sequence"`
	PayloadJSON     string    `gorm:"type:jsonb" json:"payload_json"`
	ClientTimestamp string    `gorm:"type:varchar(64)" json:"client_timestamp"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ExamEventLog) TableName() string { return "exam_event_logs" }
