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
	// Attendance is nil until attendance tracking exists in this
	// codebase (it does not yet) - never a fabricated figure.
	Attendance          *MidTermAttendance `json:"attendance"`
	ClassTeacherComment string             `json:"class_teacher_comment"`
	NextSteps           string             `json:"next_steps"`
}

// MidTermSubjectResult - one subject's in-progress components. Grade
// is deliberately never assigned at mid-term (matches the reference
// template's GRADE column, always "—") - a mid-term snapshot is not a
// final result.
type MidTermSubjectResult struct {
	SubjectName string                    `json:"subject_name"`
	SubjectCode string                    `json:"subject_code"`
	Components  []SubjectComponentResult  `json:"components,omitempty"`
	Percentage  float64                   `json:"percentage"`
	IsComplete  bool                      `json:"is_complete"`
}

// MidTermAttendance - a mid-term snapshot's attendance and homework
// habits figures.
type MidTermAttendance struct {
	DaysPossible int    `json:"days_possible"`
	DaysPresent  int    `json:"days_present"`
	DaysAbsent   int    `json:"days_absent"`
	HomeworkNote string `json:"homework_note"`
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
