package models

import "time"

// ReportCardRemark holds one student's teacher/principal comments for a
// single term's report card - the "Comments/Feedback" section every
// report-card template calls for. Before this model existed,
// TermlyReportCardResponse.TeacherRemarks/PrincipalRemarks were always
// hardcoded empty strings; there was nowhere to persist them.
//
// Scoped by (StudentID, TermID, SessionID) rather than just (StudentID,
// TermID): a student can, in principle, repeat a term across two
// different sessions, and each occurrence needs its own remark, not a
// shared one.
type ReportCardRemark struct {
	ID              string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID        string     `gorm:"type:uuid;not null;index" json:"school_id"`
	StudentID       string     `gorm:"type:uuid;not null;uniqueIndex:idx_report_card_remark_scope" json:"student_id"`
	TermID          string     `gorm:"type:uuid;not null;uniqueIndex:idx_report_card_remark_scope" json:"term_id"`
	SessionID       string     `gorm:"type:uuid;not null;uniqueIndex:idx_report_card_remark_scope" json:"session_id"`
	TeacherRemark   string     `gorm:"type:text" json:"teacher_remark"`
	TeacherID       *string    `gorm:"type:uuid" json:"teacher_id"`
	TeacherSetAt    *time.Time `json:"teacher_set_at"`
	PrincipalRemark string     `gorm:"type:text" json:"principal_remark"`
	PrincipalID     *string    `gorm:"type:uuid" json:"principal_id"`
	PrincipalSetAt  *time.Time `json:"principal_set_at"`
	Conduct         string     `gorm:"type:varchar(50)" json:"conduct"`
	// Mid-term fields are deliberately separate from TeacherRemark/
	// Conduct above, not a reused alias of them: a teacher's mid-term
	// progress note and their end-of-term comment are different pieces
	// of writing about different points in the term, and collapsing
	// them onto one pair of columns would let setting one silently
	// overwrite the other.
	MidTermComment   string     `gorm:"type:text" json:"mid_term_comment"`
	MidTermNextSteps string     `gorm:"type:text" json:"mid_term_next_steps"`
	MidTermSetBy     *string    `gorm:"type:uuid" json:"mid_term_set_by"`
	MidTermSetAt     *time.Time `json:"mid_term_set_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
