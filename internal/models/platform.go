package models

import (
	"time"

	"gorm.io/gorm"
)

// ============================================
// TENANT AUDIT LOG (Master Implementation Plan Phase 11)
// ============================================
//
// Records every privileged platform_owner / tenant-provisioning
// operation: who did what, to which tenant, when, and the outcome.
// Deliberately stores no secrets - Details is a short, non-sensitive
// JSON blob (never passwords, tokens, or LAN credentials - see
// internal/platform/service.go's writers for what actually goes in it).
type TenantAuditLog struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ActorUserID string    `gorm:"type:uuid;not null;index" json:"actor_user_id"`
	ActorRole   string    `gorm:"type:varchar(30);not null" json:"actor_role"`
	Action      string    `gorm:"type:varchar(100);not null;index" json:"action"`
	TargetType  string    `gorm:"type:varchar(50);not null" json:"target_type"`
	TargetID    string    `gorm:"type:uuid;index" json:"target_id"`
	Details     string    `gorm:"type:text" json:"details,omitempty"`
	Outcome     string    `gorm:"type:varchar(20);not null;default:'success'" json:"outcome"`
	CreatedAt   time.Time `json:"created_at"`
}

func (TenantAuditLog) TableName() string { return "tenant_audit_logs" }

// ============================================
// LAN SERVER CONFIGURATION (Master Implementation Plan Phase 10)
// ============================================
//
// One row per school tenant's School CBT Node LAN configuration.
// Read/write access is restricted to platform_owner only - see
// internal/platform's handler, gated with middleware.PlatformOwnerOnly().
// Schools never get technician-level access to this, per the plan's
// explicit requirement.
type LANServerConfig struct {
	ID             string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID       string         `gorm:"type:uuid;not null;uniqueIndex" json:"school_id"`
	ServerAddress  string         `gorm:"type:varchar(255)" json:"server_address"`
	Port           int            `gorm:"default:8080" json:"port"`
	IsActive       bool           `gorm:"default:false" json:"is_active"`
	LastTestedAt   *time.Time     `json:"last_tested_at,omitempty"`
	LastTestStatus string         `gorm:"type:varchar(20);default:'untested'" json:"last_test_status"`
	LastTestError  string         `gorm:"type:text" json:"last_test_error,omitempty"`
	UpdatedBy      string         `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LANServerConfig) TableName() string { return "lan_server_configs" }

// LANServerConfigHistory is an append-only log of every change to a
// school's LANServerConfig, letting the Platform Owner review history
// and roll back to a previously-known-valid configuration (Phase 10.2).
type LANServerConfigHistory struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConfigID      string    `gorm:"type:uuid;not null;index" json:"config_id"`
	SchoolID      string    `gorm:"type:uuid;not null;index" json:"school_id"`
	ServerAddress string    `gorm:"type:varchar(255)" json:"server_address"`
	Port          int       `json:"port"`
	IsActive      bool      `json:"is_active"`
	ChangeType    string    `gorm:"type:varchar(20);not null" json:"change_type"`
	ChangedBy     string    `gorm:"type:uuid;not null" json:"changed_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func (LANServerConfigHistory) TableName() string { return "lan_server_config_history" }

func init() {
	Register(&TenantAuditLog{})
	Register(&LANServerConfig{})
	Register(&LANServerConfigHistory{})
}
