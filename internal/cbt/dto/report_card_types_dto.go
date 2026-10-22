package dto

// ============================================
// MID-TERM PROGRESS REPORT CARD
// ============================================
//
// A progress snapshot partway through a term, distinct from the
// Main Termly Academic Report Card (TermlyReportCardResponse) - no
// backend computation service produces this yet (the shared grading
// engine, internal/grading, currently only resolves a subject's FINAL
// termly total). This type is the shape that service is expected to
// fill once built; internal/cbt/pdf's Mid-Term generator already
// renders it.

// MidTermReportCardResponse - Mid-Term Progress Report Card data.
type MidTermReportCardResponse struct {
	School   SchoolInfo              `json:"school"`
	Student  StudentInfo             `json:"student"`
	Class    ClassInfo               `json:"class"`
	Term     TermInfo                `json:"term"`
	Session  SessionInfo             `json:"session"`
	Subjects []MidTermSubjectResult  `json:"subjects"`
	Summary  ReportCardSummary       `json:"summary"`
	// Attendance is nil until this class/term has had at least one
	// day's register taken (see ExamService.attendanceSummary) - never
	// an all-zero fabricated figure.
	Attendance          *MidTermAttendance `json:"attendance"`
	ClassTeacherComment string             `json:"class_teacher_comment"`
	NextSteps           string             `json:"next_steps"`
}

// MidTermSubjectResult - one subject's in-progress components. Grade
// is deliberately never assigned at mid-term (matches the reference
// template's GRADE column, always "—") - a mid-term snapshot is not a
// final result.
type MidTermSubjectResult struct {
	SubjectName string                   `json:"subject_name"`
	SubjectCode string                   `json:"subject_code"`
	Components  []SubjectComponentResult `json:"components,omitempty"`
	// TotalScore/TotalMarks are the raw Σscore/Σmarks across every
	// included (non-final-exam-type) result - the same plain-sum
	// convention SubjectTermlyResult/ReportCardSubject use, kept
	// separate from Percentage for the identical reason: Percentage is
	// the resolved grading-engine figure (weighted when a scheme is
	// configured), which a raw score/marks pair can't represent.
	TotalScore int     `json:"total_score"`
	TotalMarks int     `json:"total_marks"`
	Percentage float64 `json:"percentage"`
	IsComplete bool    `json:"is_complete"`
}

// MidTermAttendance - a mid-term snapshot's attendance and homework
// habits figures.
type MidTermAttendance struct {
	DaysPossible int    `json:"days_possible"`
	DaysPresent  int    `json:"days_present"`
	DaysAbsent   int    `json:"days_absent"`
	// HomeworkNote is always empty for now - homework/assignment
	// tracking is a separate, not-yet-built feature this field
	// anticipates; attendance tracking (ExamService.attendanceSummary)
	// only ever populates DaysPossible/DaysPresent/DaysAbsent above.
	HomeworkNote string `json:"homework_note"`
}

// SetMidTermRemarkRequest - set a teacher's mid-term progress comment
// and next-steps note on a student's Mid-Term Progress Report Card.
type SetMidTermRemarkRequest struct {
	StudentID string `json:"student_id" binding:"required"`
	TermID    string `json:"term_id" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
	Comment   string `json:"comment" binding:"required"`
	NextSteps string `json:"next_steps,omitempty"`
}

// ============================================
// PROMOTION EXAMINATION REPORT CARD
// ============================================
//
// No promotion-policy evaluation service exists in this codebase yet
// (eligibility thresholds, core-subject rules, an approval workflow) -
// this type is the shape that service is expected to fill once built;
// internal/cbt/pdf's Promotion generator already renders it, and
// enforces the same safety rule at the rendering layer: a system
// recommendation is never presented as a final decision.

// PromotionReportCardResponse - Promotion Examination Report Card
// data. SystemRecommendation is a computed RECOMMENDATION ONLY - never
// an automatic promotion/demotion decision. FinalDecision must stay
// "Pending authorized approval" (the zero value the PDF renderer
// falls back to) until a human with the appropriate permission records
// an actual decision through an approval workflow - no code may ever
// copy SystemRecommendation into FinalDecision automatically.
type PromotionReportCardResponse struct {
	School            SchoolInfo           `json:"school"`
	Student           StudentInfo          `json:"student"`
	Class             ClassInfo            `json:"class"` // current class
	Session           SessionInfo          `json:"session"`
	Subjects          []ReportCardSubject  `json:"subjects"`
	Summary           ReportCardSummary    `json:"summary"`
	ProposedNextClass string               `json:"proposed_next_class"`
	PolicyChecksSummary   string `json:"policy_checks_summary"`
	ResultCompleteness    string `json:"result_completeness"`
	SystemRecommendation  string `json:"system_recommendation"`
	FinalDecision         string `json:"final_decision"`
	Strengths             string `json:"strengths"`
	FollowUp              string `json:"follow_up"`
	RemedialPlan          string `json:"remedial_plan"`
}

// RecordPromotionDecisionRequest - record the authorized, final
// promotion decision for one student's session. Deliberately has no
// field for "copy the system recommendation" - whoever calls this
// must state the decision themselves.
type RecordPromotionDecisionRequest struct {
	StudentID            string  `json:"student_id" binding:"required"`
	SessionID            string  `json:"session_id" binding:"required"`
	Decision             string  `json:"decision" binding:"required"`
	Notes                string  `json:"notes,omitempty"`
	ProposedClassLevelID *string `json:"proposed_class_level_id,omitempty"`
}
