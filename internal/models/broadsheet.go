package models

import "time"

// TeacherSubjectAssignment records which teacher teaches which subject
// in which class for a session - the scoping record the Student Score
// Register, "my subjects" (a teacher's own landing view), and the HOD/
// Examination Officer broad sheet all resolve against. Distinct from
// Class.TeacherID (the one homeroom/class teacher): a class has many
// subject teachers, one assignment per subject, and the same teacher
// can be assigned across several classes. No unique constraint on
// (SubjectID, ClassID, SessionID) alone - co-teaching a subject is a
// real school policy this model doesn't need to forbid - but
// (TeacherID, SubjectID, ClassID, SessionID) is unique so the same
// assignment can't be created twice.
type TeacherSubjectAssignment struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID  string    `gorm:"type:uuid;not null;index" json:"school_id"`
	TeacherID string    `gorm:"type:uuid;not null;uniqueIndex:idx_teacher_subject_assignment" json:"teacher_id"`
	SubjectID string    `gorm:"type:uuid;not null;uniqueIndex:idx_teacher_subject_assignment" json:"subject_id"`
	ClassID   string    `gorm:"type:uuid;not null;uniqueIndex:idx_teacher_subject_assignment" json:"class_id"`
	SessionID string    `gorm:"type:uuid;not null;uniqueIndex:idx_teacher_subject_assignment" json:"session_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TeacherSubjectAssignment) TableName() string { return "teacher_subject_assignments" }

// SubjectResultSubmission is one subject teacher's approval-chain state
// for a class's scores in one term - the Academic Broad Sheet spec's
// "teacher enters/saves -> teacher submits -> HOD/Examination Officer
// reviews -> academic administrator approves -> locked" workflow.
// Distinct from ReportCardApproval (scoped per student + report-type +
// term, freezing one finished document's snapshot): this is scoped per
// teacher's own worksheet - one class, one subject, one term/session -
// tracking the state of the whole Student Score Register for that
// scope rather than any single student's report. The underlying scores
// themselves live in ordinary models.Result rows (see Result.AttemptID/
// TheoryScore); this table never duplicates them, only gates and
// audits who may still edit them and whether they're ready to feed
// report cards/promotion.
//
// Status: "draft" (no row, or Version 0 - still being entered) ->
// "submitted" (teacher is done) -> "returned" (a reviewer sent it back,
// Version unchanged) -> "reviewed" (HOD/Examination Officer checked
// completeness, passed upward) -> "approved" (locked - see IsLocked).
// Re-submitting after a correction bumps Version; the prior approval
// state is recoverable from SubjectResultAuditLog, same pattern
// ReportCardApproval/ReportCardAuditLog already established.
type SubjectResultSubmission struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID   string     `gorm:"type:uuid;not null;uniqueIndex:idx_subject_result_submission_scope" json:"school_id"`
	ClassID    string     `gorm:"type:uuid;not null;uniqueIndex:idx_subject_result_submission_scope" json:"class_id"`
	SubjectID  string     `gorm:"type:uuid;not null;uniqueIndex:idx_subject_result_submission_scope" json:"subject_id"`
	TermID     string     `gorm:"type:uuid;not null;uniqueIndex:idx_subject_result_submission_scope" json:"term_id"`
	SessionID  string     `gorm:"type:uuid;not null;uniqueIndex:idx_subject_result_submission_scope" json:"session_id"`
	Status     string     `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	Version    int        `gorm:"not null;default:0" json:"version"`
	TeacherID  string     `gorm:"type:uuid;not null" json:"teacher_id"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ReviewedBy *string    `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	ApprovedBy *string    `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	Notes      string     `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (SubjectResultSubmission) TableName() string { return "subject_result_submissions" }

// IsLocked reports whether ordinary score-register edits are blocked
// for this submission - true once it has been reviewed or approved.
// "submitted" alone does NOT lock (a teacher who submits too early can
// still fix a typo before a reviewer looks at it); only once a human
// reviewer has acted does a correction start requiring the configured
// correction permission + reason (see the broad-sheet handler).
func (s SubjectResultSubmission) IsLocked() bool {
	return s.Status == "reviewed" || s.Status == "approved"
}

// SubjectResultAuditLog is SubjectResultSubmission's append-only
// historical trail, mirroring ReportCardAuditLog's own tamper-evident
// design exactly (content-hash of its own columns, recomputed and
// compared on verify) - see that model's doc comment for the full
// rationale. Kept as a deliberately separate table/model rather than a
// shared one: the two track different kinds of actions (one student's
// finished report vs. one teacher's whole-class worksheet) and mixing
// their rows would make neither trail easy to read.
type SubjectResultAuditLog struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID    string     `gorm:"type:uuid;not null;index" json:"school_id"`
	ActorID     string     `gorm:"type:uuid;not null" json:"actor_id"`
	Action      string     `gorm:"type:varchar(40);not null;index" json:"action"` // score_saved|submitted|returned|reviewed|approved|corrected
	ClassID     string     `gorm:"type:uuid;index" json:"class_id,omitempty"`
	SubjectID   string     `gorm:"type:uuid;index" json:"subject_id,omitempty"`
	TermID      string     `gorm:"type:varchar(40)" json:"term_id,omitempty"`
	SessionID   string     `gorm:"type:uuid" json:"session_id,omitempty"`
	StudentID   *string    `gorm:"type:uuid;index" json:"student_id,omitempty"`
	Notes       string     `gorm:"type:text" json:"notes,omitempty"`
	ContentHash string     `gorm:"type:varchar(64);not null" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (SubjectResultAuditLog) TableName() string { return "subject_result_audit_logs" }
