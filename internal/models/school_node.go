package models

import (
	"time"

	"gorm.io/gorm"
)

// SchoolNodeCredential authenticates a School CBT Node's push/pull calls to
// the cloud sync endpoints (internal/nodesync). One school can have several
// credentials (e.g. to rotate one without downtime); a node authenticates
// with "Authorization: Bearer <plaintext key>" and the server looks it up by
// KeyHash - the plaintext key is shown to the operator exactly once, at
// creation, and never stored or logged.
type SchoolNodeCredential struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID   string         `gorm:"type:uuid;not null;index" json:"school_id"`
	Label      string         `gorm:"type:varchar(100)" json:"label"`
	KeyHash    string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	LastUsedAt *time.Time     `json:"last_used_at"`
	RevokedAt  *time.Time     `json:"revoked_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SchoolNodeCredential) TableName() string { return "school_node_credentials" }
