package models

import "time"

// ReportCardApproval records one student's one report card instance's
// current sign-off state - the gate between a school's internal
// working figures and the version actually released/printed as
// official. A report card with no row here has never been submitted
// for approval; GetApprovalStatus reports that as "not_submitted",
// never infers a status.
//
// Approving freezes SnapshotJSON: the exact report-card payload
// (dto.TermlyReportCardResponse / MidTermReportCardResponse /
// PromotionReportCardResponse, marshaled) as it stood at the moment of
// approval. A later correction to the underlying results (a re-marked
// script, a fixed CA entry) changes what GetTermlyReportCard computes
// live, but never silently changes what was already approved and
// printed - the officially approved copy stays exactly what it was
// approved as until someone explicitly re-approves (which bumps
// Version and replaces the snapshot).
//
// TermID is empty string, never NULL, for "promotion" (session-scoped,
// no single term) - the same ScopeKey-avoidance reasoning
// PromotionPolicy/GradingScale already use: Postgres treats every NULL
// as distinct in a unique index, which would let the same student's
// promotion report be "approved" under more than one row instead of
// exactly one.
type ReportCardApproval struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID   string `gorm:"type:uuid;not null;index" json:"school_id"`
	StudentID  string `gorm:"type:uuid;not null;uniqueIndex:idx_report_card_approval_scope" json:"student_id"`
	ReportType string `gorm:"type:varchar(20);not null;uniqueIndex:idx_report_card_approval_scope" json:"report_type"`
	TermID     string `gorm:"type:varchar(40);not null;default:'';uniqueIndex:idx_report_card_approval_scope" json:"term_id"`
	SessionID  string `gorm:"type:uuid;not null;uniqueIndex:idx_report_card_approval_scope" json:"session_id"`
	// Status is "approved" or "rejected" - there is no row at all for
	// "not yet submitted", mirroring PromotionDecision's own "no row =
	// no decision yet" convention rather than a third enum value
	// nothing ever sets atomically.
	Status       string `gorm:"type:varchar(20);not null" json:"status"`
	Version      int    `gorm:"not null;default:1" json:"version"`
	SnapshotJSON string `gorm:"type:text" json:"-"`
	Notes        string `gorm:"type:text" json:"notes"`
	ActorID      string `gorm:"type:uuid;not null" json:"actor_id"`
	// SyncedToCloudAt is nil until a School Node has pushed this row to
	// the cloud (internal/nodesync) - a row created directly against the
	// cloud deployment (not a node) simply never needs this set. Mirrors
	// models.ExamAttempt/Result's own SyncedToCloudAt column and the
	// node-side "outbox" query pattern (WHERE synced_to_cloud_at IS NULL)
	// in internal/nodesync/repository.
	SyncedToCloudAt *time.Time `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ReportCardAuditLog is the append-only historical trail behind
// ReportCardApproval's own "latest status + version" row, which is
// overwritten in place on every new approve/reject/batch action and so
// cannot by itself answer "what happened before this." Scoped to
// SchoolID (unlike the platform-tier-only TenantAuditLog, which
// records privileged platform_owner/tenant-provisioning operations and
// has no SchoolID column at all) so a school's own admins can see
// their own report-card approval history without any cross-tenant
// data ever appearing in it.
type ReportCardAuditLog struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID   string `gorm:"type:uuid;not null;index" json:"school_id"`
	ActorID    string `gorm:"type:uuid;not null" json:"actor_id"`
	Action     string `gorm:"type:varchar(40);not null;index" json:"action"`
	ReportType string `gorm:"type:varchar(20);not null" json:"report_type"`
	StudentID  string `gorm:"type:uuid;index" json:"student_id,omitempty"`
	ClassID    string `gorm:"type:uuid;index" json:"class_id,omitempty"`
	TermID     string `gorm:"type:varchar(40)" json:"term_id,omitempty"`
	SessionID  string `gorm:"type:uuid" json:"session_id,omitempty"`
	Notes      string `gorm:"type:text" json:"notes,omitempty"`
	// ContentHash is sha256(SchoolID|ActorID|Action|ReportType|StudentID|
	// ClassID|TermID|SessionID|Notes|CreatedAt), computed once at write
	// time (writeReportCardAudit) and never recomputed on read. It makes
	// this row tamper-evident: VerifyReportCardAuditIntegrity recomputes
	// the hash from the row's own stored columns and flags any mismatch -
	// a silent direct-DB edit of an already-written entry (bypassing the
	// approve/reject/batch-approve API, the only legitimate writer)
	// changes the columns but not the hash, so the two stop matching.
	// This detects tampering of an existing row; it cannot detect a row
	// being deleted outright, which no per-row hash can.
	ContentHash string `gorm:"type:varchar(64);not null" json:"-"`
	// SyncedToCloudAt - see ReportCardApproval's own field of the same
	// name/purpose.
	SyncedToCloudAt *time.Time `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
}
