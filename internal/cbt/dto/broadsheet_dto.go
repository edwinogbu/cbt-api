package dto

import "time"

// ============================================
// ASSESSMENT SETTINGS (AssessmentScheme/Component, GradingScale/Band)
// ============================================

// AssessmentComponentInput is one weighted component of an
// AssessmentScheme being created/updated - mirrors
// models.AssessmentComponent's own fields.
type AssessmentComponentInput struct {
	Code        string   `json:"code" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Weight      float64  `json:"weight" binding:"required,min=0,max=100"`
	ExamTypes   []string `json:"exam_types" binding:"required,min=1"`
	Aggregation string   `json:"aggregation"` // pooled | average | best | latest - defaults to "pooled"
	SortOrder   int      `json:"sort_order"`
}

// AssessmentSchemeRequest creates or replaces (upserts, keyed by this
// exact scope) one AssessmentScheme and its full set of components.
// ClassLevelID/SubjectID both empty = school-wide; ClassLevelID only =
// class-level-wide; both set = subject-within-class-level - the same
// three scopes grading.ResolveScheme already resolves in that order.
type AssessmentSchemeRequest struct {
	ClassLevelID  string                      `json:"class_level_id"`
	SubjectID     string                      `json:"subject_id"`
	MissingPolicy string                      `json:"missing_policy"` // "renormalize" | "zero" - defaults to "renormalize"
	Components    []AssessmentComponentInput `json:"components" binding:"required,min=1"`
}

type AssessmentComponentResponse struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Weight      float64  `json:"weight"`
	ExamTypes   []string `json:"exam_types"`
	Aggregation string   `json:"aggregation"`
	SortOrder   int      `json:"sort_order"`
}

type AssessmentSchemeResponse struct {
	ID            string                         `json:"id"`
	SchoolID      string                         `json:"school_id"`
	ClassLevelID  string                         `json:"class_level_id,omitempty"`
	SubjectID     string                         `json:"subject_id,omitempty"`
	MissingPolicy string                         `json:"missing_policy"`
	IsActive      bool                           `json:"is_active"`
	Components    []AssessmentComponentResponse `json:"components"`
	CreatedAt     time.Time                      `json:"created_at"`
	UpdatedAt     time.Time                      `json:"updated_at"`
}

// GradingScaleBandInput mirrors models.GradingScaleBand.
type GradingScaleBandInput struct {
	Grade      string  `json:"grade" binding:"required"`
	MinPercent float64 `json:"min_percent" binding:"min=0,max=100"`
	GradePoint float64 `json:"grade_point"`
	Remark     string  `json:"remark"`
	SortOrder  int     `json:"sort_order"`
}

// GradingScaleRequest creates or replaces (upserts, keyed by scope)
// one GradingScale and its full set of bands.
type GradingScaleRequest struct {
	ClassLevelID string                  `json:"class_level_id"`
	Name         string                  `json:"name" binding:"required"`
	PassMark     float64                 `json:"pass_mark" binding:"min=0,max=100"`
	Bands        []GradingScaleBandInput `json:"bands" binding:"required,min=1"`
}

type GradingScaleBandResponse struct {
	ID         string  `json:"id"`
	Grade      string  `json:"grade"`
	MinPercent float64 `json:"min_percent"`
	GradePoint float64 `json:"grade_point"`
	Remark     string  `json:"remark"`
	SortOrder  int     `json:"sort_order"`
}

type GradingScaleResponse struct {
	ID           string                      `json:"id"`
	SchoolID     string                      `json:"school_id"`
	ClassLevelID string                      `json:"class_level_id,omitempty"`
	Name         string                      `json:"name"`
	PassMark     float64                     `json:"pass_mark"`
	IsActive     bool                        `json:"is_active"`
	Bands        []GradingScaleBandResponse `json:"bands"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
}

// ============================================
// TEACHER SUBJECT ASSIGNMENTS
// ============================================

type CreateTeacherSubjectAssignmentRequest struct {
	TeacherID string `json:"teacher_id" binding:"required,uuid"`
	SubjectID string `json:"subject_id" binding:"required,uuid"`
	ClassID   string `json:"class_id" binding:"required,uuid"`
	SessionID string `json:"session_id" binding:"required,uuid"`
}

type TeacherSubjectAssignmentResponse struct {
	ID          string `json:"id"`
	TeacherID   string `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	ClassID     string `json:"class_id"`
	ClassName   string `json:"class_name"`
	SessionID   string `json:"session_id"`
	IsActive    bool   `json:"is_active"`
}

// MySubjectAssignment is one row of a teacher's own "my subjects"
// landing view - the scoping list the Student Score Register picks
// from.
type MySubjectAssignment struct {
	AssignmentID string `json:"assignment_id"`
	SubjectID    string `json:"subject_id"`
	SubjectName  string `json:"subject_name"`
	SubjectCode  string `json:"subject_code"`
	ClassID      string `json:"class_id"`
	ClassName    string `json:"class_name"`
	SessionID    string `json:"session_id"`
}

// ============================================
// STUDENT SCORE REGISTER
// ============================================

// ScoreComponentDef is one assessment component the register has a
// column for, resolved from the effective AssessmentScheme (or the
// built-in default) for this class-level/subject - the register must
// never let a cell exceed this MaxMarks.
type ScoreComponentDef struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ExamID    string `json:"exam_id"`    // the per-component Exam row scores are stored against
	MaxMarks  int    `json:"max_marks"`
	ExamType  string `json:"exam_type"`
}

// ScoreCell is one student's one component's current score - nil Score
// means "not yet entered", distinct from a genuine 0. ReadOnly is true
// for a component sourced from a real CBT exam attempt (the objective
// portion of a combined exam, or any pure-CBT component) - a teacher
// may only ever directly edit a manually-scored or theory cell.
type ScoreCell struct {
	ComponentCode string `json:"component_code"`
	Score         *int   `json:"score"`
	MaxMarks      int    `json:"max_marks"`
	ReadOnly      bool   `json:"read_only"`
}

// ScoreRegisterRow is one student's full row in the Subject Teacher
// Score Register (template A) - every configured component's cell,
// plus the authoritative server-computed total/percentage/grade.
type ScoreRegisterRow struct {
	StudentID       string      `json:"student_id"`
	AdmissionNumber string      `json:"admission_number"`
	StudentName     string      `json:"student_name"`
	Cells           []ScoreCell `json:"cells"`
	Total           *int        `json:"total"`
	MaxTotal        int         `json:"max_total"`
	Percentage      *float64    `json:"percentage"`
	Grade           string      `json:"grade,omitempty"`
	IsComplete      bool        `json:"is_complete"`
	// Remark is the teacher's free-text comment on this student's
	// subject result (template A's "Teacher remark" column) - nil
	// means none has been written yet, distinct from an empty string
	// (a remark that was written and then cleared).
	Remark *string `json:"remark,omitempty"`
}

// ScoreRegisterResponse is the full Subject Teacher Score Register for
// one class/subject/term/session - everything template A's "A.
// STUDENT SCORE REGISTER" table needs, plus its header fields.
type ScoreRegisterResponse struct {
	ClassID       string              `json:"class_id"`
	ClassName     string              `json:"class_name"`
	SubjectID     string              `json:"subject_id"`
	SubjectName   string              `json:"subject_name"`
	TeacherID     string              `json:"teacher_id,omitempty"`
	TeacherName   string              `json:"teacher_name,omitempty"`
	TermID        string              `json:"term_id"`
	SessionID     string              `json:"session_id"`
	Components    []ScoreComponentDef `json:"components"`
	Rows          []ScoreRegisterRow  `json:"rows"`
	SubmissionStatus string           `json:"submission_status"` // draft|submitted|returned|reviewed|approved
	IsLocked      bool                `json:"is_locked"`
	StudentsEnrolled int              `json:"students_enrolled"`
	ResultsComplete  int              `json:"results_complete"`
}

// ScoreEntryInput is one cell's new value in a batch save - a nil
// Score clears/leaves-unset that cell (never writes a 0).
type ScoreEntryInput struct {
	StudentID string `json:"student_id" binding:"required,uuid"`
	Score     *int   `json:"score"`
}

// SaveScoreRegisterRequest batches every edited cell for ONE component
// (one column) of one class/subject/term/session register in a single
// save - mirrors the attendance register's own roster -> edit-buffer ->
// one-batch-save shape.
type SaveScoreRegisterRequest struct {
	ClassID       string            `json:"class_id" binding:"required,uuid"`
	SubjectID     string            `json:"subject_id" binding:"required,uuid"`
	TermID        string            `json:"term_id" binding:"required,uuid"`
	SessionID     string            `json:"session_id" binding:"required,uuid"`
	ComponentCode string            `json:"component_code" binding:"required"`
	Entries       []ScoreEntryInput `json:"entries" binding:"required,min=1"`
	// CorrectionReason is required only when this save targets a
	// locked (reviewed/approved) SubjectResultSubmission - see
	// ExamService.SaveScoreRegister.
	CorrectionReason string `json:"correction_reason,omitempty"`
}

// SaveScoreRegisterRemarkRequest sets (or clears, with an empty
// Remark) one student's teacher remark for a subject/term/session -
// kept as its own small request rather than folded into
// SaveScoreRegisterRequest, since a remark isn't a score component and
// has no MaxMarks/ReadOnly semantics to validate against.
type SaveScoreRegisterRemarkRequest struct {
	ClassID          string `json:"class_id" binding:"required,uuid"`
	SubjectID        string `json:"subject_id" binding:"required,uuid"`
	TermID           string `json:"term_id" binding:"required,uuid"`
	SessionID        string `json:"session_id" binding:"required,uuid"`
	StudentID        string `json:"student_id" binding:"required,uuid"`
	Remark           string `json:"remark"`
	CorrectionReason string `json:"correction_reason,omitempty"`
}

type SaveScoreRegisterResult struct {
	StudentID string `json:"student_id"`
	Error     string `json:"error,omitempty"`
}

type SaveScoreRegisterResponse struct {
	Saved   []SaveScoreRegisterResult `json:"saved"`
	Failed  []SaveScoreRegisterResult `json:"failed"`
}

// ============================================
// SUBMISSION / APPROVAL CHAIN
// ============================================

type SubjectResultScopeRequest struct {
	ClassID   string `json:"class_id" binding:"required,uuid"`
	SubjectID string `json:"subject_id" binding:"required,uuid"`
	TermID    string `json:"term_id" binding:"required,uuid"`
	SessionID string `json:"session_id" binding:"required,uuid"`
	Notes     string `json:"notes"`
}

type SubjectResultSubmissionResponse struct {
	ID         string     `json:"id"`
	ClassID    string     `json:"class_id"`
	SubjectID  string     `json:"subject_id"`
	TermID     string     `json:"term_id"`
	SessionID  string     `json:"session_id"`
	Status     string     `json:"status"`
	Version    int        `json:"version"`
	TeacherID  string     `json:"teacher_id"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ReviewedBy string     `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

// ============================================
// BROAD SHEETS (Class Teacher / HOD / VP Academics)
// ============================================

// ClassBroadSheetSubjectCell is one student's one subject's figures in
// the Class Teacher Broad Sheet (template B) - deliberately thin (just
// what the matrix needs), computed from the same authoritative
// GetStudentTermlyResult/buildSubjectTermlyResult pipeline the report
// card uses.
type ClassBroadSheetSubjectCell struct {
	// SubjectCode (not an ID - the report-card payloads this is built
	// from only carry Name/Code, and Subject.Code is already globally
	// unique) keys this cell to ClassBroadSheetResponse's own
	// SubjectCodes/SubjectNames column headers.
	SubjectCode string  `json:"subject_code"`
	Score       *int    `json:"score"`
	MaxMarks    int     `json:"max_marks"`
	Grade       string  `json:"grade,omitempty"`
	IsComplete  bool    `json:"is_complete"`
}

type ClassBroadSheetRow struct {
	StudentID       string                       `json:"student_id"`
	AdmissionNumber string                       `json:"admission_number"`
	StudentName     string                       `json:"student_name"`
	Subjects        []ClassBroadSheetSubjectCell `json:"subjects"`
	OverallAverage  *float64                     `json:"overall_average"`
	Pass            int                          `json:"pass"`
	Fail            int                          `json:"fail"`
	Incomplete      int                          `json:"incomplete"`
	DaysPresent     int                          `json:"days_present"`
	DaysOpen        int                          `json:"days_open"`
}

type ClassBroadSheetResponse struct {
	ClassID          string               `json:"class_id"`
	ClassName        string               `json:"class_name"`
	ClassTeacherName string               `json:"class_teacher_name,omitempty"`
	TermID           string               `json:"term_id"`
	SessionID        string               `json:"session_id"`
	ReportType       string               `json:"report_type"`
	SubjectCodes     []string             `json:"subject_codes"`
	SubjectNames     []string             `json:"subject_names"`
	Rows             []ClassBroadSheetRow `json:"rows"`
	StudentCount     int                  `json:"student_count"`
	AllSubjectsSubmitted bool             `json:"all_subjects_submitted"`
	StudentsWithIncompleteResults int     `json:"students_with_incomplete_results"`
	// Review is this broad sheet's own actionable workflow state
	// (template B, page 4) - nil only if ClassBroadSheetReview lookup
	// itself failed (never blocks the read-only matrix above it from
	// rendering).
	Review *ClassBroadSheetReviewResponse `json:"review,omitempty"`
}

// ClassBroadSheetReviewResponse is the Class Teacher Broad Sheet's own
// workflow state - template B page 4's summary/signature/next-action
// block.
type ClassBroadSheetReviewResponse struct {
	ID             string     `json:"id,omitempty"`
	ClassID        string     `json:"class_id"`
	ReportType     string     `json:"report_type"`
	TermID         string     `json:"term_id,omitempty"`
	SessionID      string     `json:"session_id"`
	Status         string     `json:"status"`
	Summary        string     `json:"summary"`
	ClassTeacherID string     `json:"class_teacher_id,omitempty"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	ReviewerID     string     `json:"reviewer_id,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ReturnNotes    string     `json:"return_notes,omitempty"`
	NextAction     string     `json:"next_action,omitempty"`
}

// SaveClassBroadSheetReviewRequest saves (without submitting) the
// class teacher's draft summary text.
type SaveClassBroadSheetReviewRequest struct {
	ClassID    string `json:"class_id" binding:"required,uuid"`
	ReportType string `json:"report_type" binding:"required"`
	TermID     string `json:"term_id"`
	SessionID  string `json:"session_id" binding:"required,uuid"`
	Summary    string `json:"summary"`
	NextAction string `json:"next_action"`
}

// ClassBroadSheetReviewActionRequest is the shared shape for
// submit/return/approve - Notes is the reviewer's return reason when
// returning, otherwise optional.
type ClassBroadSheetReviewActionRequest struct {
	ClassID    string `json:"class_id" binding:"required,uuid"`
	ReportType string `json:"report_type" binding:"required"`
	TermID     string `json:"term_id"`
	SessionID  string `json:"session_id" binding:"required,uuid"`
	Notes      string `json:"notes"`
}

// SubjectBroadSheetRow is one class's standing for the HOD/
// Examination Officer's own subject-across-classes view (the fourth
// broad sheet the spec describes but the attached templates don't
// render a page for - modeled consistently with the other three).
type SubjectBroadSheetRow struct {
	ClassID        string  `json:"class_id"`
	ClassName      string  `json:"class_name"`
	TeacherID      string  `json:"teacher_id,omitempty"`
	TeacherName    string  `json:"teacher_name,omitempty"`
	Enrolled       int     `json:"enrolled"`
	MarksEntered   int     `json:"marks_entered"`
	Incomplete     int     `json:"incomplete"`
	ClassAverage   float64 `json:"class_average_percent"`
	PassRate       float64 `json:"pass_rate_percent"`
	SubmissionStatus string `json:"submission_status"`
}

type SubjectBroadSheetResponse struct {
	SubjectID   string                 `json:"subject_id"`
	SubjectName string                 `json:"subject_name"`
	TermID      string                 `json:"term_id"`
	SessionID   string                 `json:"session_id"`
	Rows        []SubjectBroadSheetRow `json:"rows"`
}

// SchoolBroadSheetRow is one class+subject's monitor row in the VP
// Academics/School General Broad Sheet (template C, section A).
type SchoolBroadSheetRow struct {
	ClassID          string  `json:"class_id"`
	ClassName        string  `json:"class_name"`
	SubjectID        string  `json:"subject_id"`
	SubjectName      string  `json:"subject_name"`
	TeacherName      string  `json:"teacher_name,omitempty"`
	Enrolled         int     `json:"enrolled"`
	MarksEntered     int     `json:"marks_entered"`
	Incomplete       int     `json:"incomplete"`
	ClassAveragePct  float64 `json:"class_average_percent"`
	PassRatePct      float64 `json:"pass_rate_percent"`
	Submitted        bool    `json:"submitted"`
	Reviewed         bool    `json:"reviewed"`
	ApprovedPublished bool   `json:"approved_published"`
}

type SchoolBroadSheetResponse struct {
	TermID              string                `json:"term_id"`
	SessionID           string                `json:"session_id"`
	ReportType          string                `json:"report_type"`
	Rows                []SchoolBroadSheetRow `json:"rows"`
	TotalStudents       int                   `json:"total_students"`
	ClassesIncluded     int                   `json:"classes_included"`
	SubjectResultSets   int                   `json:"subject_result_sets"`
	SubmissionsComplete int                   `json:"submissions_complete"`
	PendingIncomplete   int                   `json:"pending_incomplete"`
	AwaitingReview      int                   `json:"awaiting_review"`
	Approved            int                   `json:"approved"`
	Published           int                   `json:"published"`
	// GeneratedAt and OverallStatus are template C's "GENERATED ON" and
	// "OVERALL STATUS" header fields - computed fresh on every read,
	// never stored, so they always reflect this exact response's own
	// figures rather than a stale cached rollup.
	GeneratedAt   time.Time `json:"generated_at"`
	OverallStatus string    `json:"overall_status"` // "In progress" | "Ready for review" | "Published"
	// PromotionReview is template C's "Section C" - populated only when
	// ReportType == "promotion"; nil for termly/midterm reports, which
	// have no eligibility decision to monitor.
	PromotionReview *SchoolPromotionReviewSummary `json:"promotion_review,omitempty"`
}

// SchoolPromotionReviewSummary is the VP Academics / School General
// Broad Sheet's "C. PROMOTION REVIEW" section - a school-wide tally of
// the same per-student SystemRecommendation/FinalDecision the
// Promotion Examination Report Card already computes (internal/cbt/
// service/promotion_report_service.go), never a second, independently
// computed eligibility check.
type SchoolPromotionReviewSummary struct {
	EligibilityRulesApplied string `json:"eligibility_rules_applied"`
	EligibleCount           int    `json:"eligible_count"`
	RequiresReviewCount     int    `json:"requires_review_count"`
	IncompleteBlockedCount  int    `json:"incomplete_blocked_count"`
	// DecisionStatus is "Approved" once every evaluated student in
	// scope has a recorded FinalDecision, else "Pending" - mirrors
	// FinalDecision's own rule that only a human-recorded decision
	// (never the system recommendation) can ever make this "Approved".
	DecisionStatus string `json:"decision_status"`
}
