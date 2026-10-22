package dto

import "time"

// ============================================
// REPORT CARD APPROVALS, BATCH REPORTING & EXPORTS
// ============================================
//
// A school's sign-off on a report card is tracked separately from the
// report card's own live, always-recomputed figures
// (GetTermlyReportCard/GetMidTermProgressReport/GetPromotionReportCard) -
// approving freezes a snapshot of what was approved, so a later
// correction to underlying results never silently changes what was
// already signed off and printed. See models.ReportCardApproval's own
// doc comment for the full rationale.

// ReportCardScope identifies one student's one report card instance -
// the same four-part scope models.ReportCardApproval and
// models.ReportCardAuditLog key on. TermID is empty for "promotion"
// (session-scoped, no single term).
type ReportCardScope struct {
	StudentID  string `json:"student_id" binding:"required"`
	ReportType string `json:"report_type" binding:"required"` // "termly" | "midterm" | "promotion"
	TermID     string `json:"term_id,omitempty"`
	SessionID  string `json:"session_id" binding:"required"`
}

// ApproveReportCardRequest approves one student's report card,
// freezing a snapshot of its current, live-computed figures as the
// official approved copy.
type ApproveReportCardRequest struct {
	ReportCardScope
	Notes string `json:"notes,omitempty"`
}

// RejectReportCardRequest marks one student's report card as rejected
// (sent back for correction) - Notes is required, since a rejection
// with no stated reason gives whoever has to fix it nothing to act on.
type RejectReportCardRequest struct {
	ReportCardScope
	Notes string `json:"notes" binding:"required"`
}

// ReportCardApprovalResponse is the current sign-off state of one
// student's report card. Status is "not_submitted" when no
// models.ReportCardApproval row exists yet - never inferred as
// "approved" or "rejected" by default.
type ReportCardApprovalResponse struct {
	StudentID  string     `json:"student_id"`
	ReportType string     `json:"report_type"`
	TermID     string     `json:"term_id,omitempty"`
	SessionID  string     `json:"session_id"`
	Status     string     `json:"status"`
	Version    int        `json:"version"`
	Notes      string     `json:"notes,omitempty"`
	ActorID    string     `json:"actor_id,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// BatchApproveRequest approves every named student's report card for
// one class/report-type/term-or-session scope in one call.
type BatchApproveRequest struct {
	ClassID    string `json:"class_id" binding:"required"`
	ReportType string `json:"report_type" binding:"required"`
	TermID     string `json:"term_id,omitempty"`
	SessionID  string `json:"session_id" binding:"required"`
	// StudentIDs, left empty, approves every currently active student
	// in the class - an explicit list approves only those named.
	StudentIDs []string `json:"student_ids,omitempty"`
	Notes      string   `json:"notes,omitempty"`
}

// BatchApprovalFailure is one student's reason for not being approved
// in a BatchApproveRequest - a batch call never fails all-or-nothing;
// every other student's approval in the same request still proceeds.
type BatchApprovalFailure struct {
	StudentID string `json:"student_id"`
	Error     string `json:"error"`
}

type BatchApproveResponse struct {
	ApprovedStudentIDs []string               `json:"approved_student_ids"`
	Failed             []BatchApprovalFailure `json:"failed"`
}

// ReportCardAuditEntry is one historical action from
// models.ReportCardAuditLog.
type ReportCardAuditEntry struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	ReportType string    `json:"report_type"`
	StudentID  string    `json:"student_id,omitempty"`
	ClassID    string    `json:"class_id,omitempty"`
	TermID     string    `json:"term_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	ActorID    string    `json:"actor_id"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ClassReportCardSummaryRow is one student's row in a class-wide
// report-card roster - the review screen an admin/teacher uses before
// approving or exporting a whole class, not a full per-subject
// breakdown (that stays in the individual report card / PDF).
type ClassReportCardSummaryRow struct {
	StudentID         string  `json:"student_id"`
	StudentName       string  `json:"student_name"`
	AdmissionNumber   string  `json:"admission_number"`
	OverallPercentage float64 `json:"overall_percentage"`
	// OverallGrade is always empty for a Mid-Term report - no letter
	// grade is ever assigned at mid-term, matching
	// dto.MidTermSubjectResult having no Grade field at all.
	OverallGrade    string `json:"overall_grade,omitempty"`
	ClassPosition   int    `json:"class_position,omitempty"`
	ApprovalStatus  string `json:"approval_status"`
	ApprovalVersion int    `json:"approval_version,omitempty"`
	// Error is set instead of the figures above when this student's
	// report card could not be generated (e.g. no results recorded at
	// all yet) - the student is still listed, never silently dropped
	// from the class roster.
	Error string `json:"error,omitempty"`
}

type ClassReportCardSummaryResponse struct {
	ClassID    string                      `json:"class_id"`
	ClassName  string                      `json:"class_name"`
	ReportType string                      `json:"report_type"`
	TermID     string                      `json:"term_id,omitempty"`
	SessionID  string                      `json:"session_id"`
	Rows       []ClassReportCardSummaryRow `json:"rows"`
}

// AuditIntegrityIssue flags one models.ReportCardAuditLog row whose
// stored ContentHash no longer matches its own columns - see that
// field's doc comment for what this can and can't detect.
type AuditIntegrityIssue struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	StudentID string    `json:"student_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditIntegrityReport struct {
	Checked int                   `json:"checked"`
	Issues  []AuditIntegrityIssue `json:"issues"`
}
