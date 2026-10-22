package models

import "time"

// AttendanceRecord is one student's attendance mark for one calendar
// date - the atomic unit everything else in this file is computed from.
// There is deliberately no separate "school calendar"/"term day" table:
// a date counts as a school day for a class/term purely by having at
// least one AttendanceRecord against it (see
// ExamService.attendanceSchoolDaysOpen) - the day a teacher actually
// took the register, nothing configured in advance. This keeps the
// feature to exactly what report cards need (days present/absent/open)
// without building an academic-calendar subsystem nothing else in this
// codebase has asked for.
//
// Scoped uniquely by (StudentID, TermID, Date): a student has at most
// one mark for one day within one term - marking the same day again
// (e.g. a correction) updates that row rather than creating a second
// one. TermID is included (not just StudentID+Date) for the same reason
// ReportCardRemark scopes by term rather than date alone: a student
// repeating a term in a later session must get its own independent
// attendance record for any date that happens to recur, rather than
// silently colliding with the first time through.
type AttendanceRecord struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID  string    `gorm:"type:uuid;not null;index" json:"school_id"`
	StudentID string    `gorm:"type:uuid;not null;uniqueIndex:idx_attendance_scope" json:"student_id"`
	ClassID   string    `gorm:"type:uuid;not null;index" json:"class_id"`
	TermID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_attendance_scope" json:"term_id"`
	SessionID string    `gorm:"type:uuid;not null;index" json:"session_id"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex:idx_attendance_scope" json:"date"`
	// Status is one of "present", "absent", "late", "excused" - a short,
	// school-facing set validated at the service layer (see
	// ExamService.validAttendanceStatuses), not a DB enum, matching this
	// codebase's established preference (PromotionDecision.Decision,
	// ReportCardApproval.Status) for free-text/service-validated status
	// columns over a Postgres enum type.
	Status   string `gorm:"type:varchar(20);not null" json:"status"`
	MarkedBy string `gorm:"type:uuid;not null" json:"marked_by"`
	Notes    string `gorm:"type:text" json:"notes,omitempty"`
	// SyncedToCloudAt - see models.ReportCardApproval's own field of the
	// same name/purpose: nil until a School Node has pushed this row to
	// the cloud (internal/nodesync), feeding the same node-side outbox
	// query pattern (WHERE synced_to_cloud_at IS NULL).
	SyncedToCloudAt *time.Time `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
