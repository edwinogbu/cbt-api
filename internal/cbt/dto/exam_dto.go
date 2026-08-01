package dto

import (
	"time"
)

// ============================================
// EXAM MANAGEMENT REQUEST DTOs
// ============================================

type CreateExamRequest struct {
	Title            string     `json:"title" binding:"required"`
	SubjectID        string     `json:"subject_id" binding:"required,uuid"`
	ClassID          string     `json:"class_id,omitempty"`
	DurationMinutes  int        `json:"duration_minutes" binding:"required,min=1"`
	TotalMarks       int        `json:"total_marks" binding:"required,min=1"`
	PassMark         int        `json:"pass_mark" binding:"required,min=0"`
	Instructions     string     `json:"instructions"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	ShuffleOptions   bool       `json:"shuffle_options"`
	IsActive         bool       `json:"is_active"`
}

type UpdateExamRequest struct {
	Title            *string    `json:"title"`
	SubjectID        *string    `json:"subject_id"`
	ClassID          *string    `json:"class_id"`
	DurationMinutes  *int       `json:"duration_minutes"`
	TotalMarks       *int       `json:"total_marks"`
	PassMark         *int       `json:"pass_mark"`
	Instructions     *string    `json:"instructions"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	ShuffleQuestions *bool      `json:"shuffle_questions"`
	ShuffleOptions   *bool      `json:"shuffle_options"`
	IsActive         *bool      `json:"is_active"`
	Status           *string    `json:"status"`
	ExamType         *string    `json:"exam_type"`
	SessionID        *string    `json:"session_id"`
	TermID           *string    `json:"term_id"`
}

type AddQuestionsToExamRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required"`
}

type AssignExamRequest struct {
	StudentIDs []string   `json:"student_ids"`
	ClassID    string     `json:"class_id,omitempty"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
}

// ============================================
// EXAM TAKING REQUEST DTOs
// ============================================

type StartExamRequest struct {
	// ExamID     string `json:"exam_id" binding:"required,uuid"`
	// // StudentID  string `json:"student_id" binding:"required,uuid"`
	// StudentID  string `json:"student_id,omitempty"` // ← Remove required tag
	// DeviceInfo string `json:"device_info"`
	// IPAddress  string `json:"ip_address"`
	ExamID     string `json:"exam_id,omitempty"`      // Optional - taken from URL
    StudentID  string `json:"student_id,omitempty"`   // Internal - never from body
    DeviceInfo string `json:"device_info,omitempty"`  // Optional - auto-detected
    IPAddress  string `json:"ip_address,omitempty"`   // Optional - auto-detected
}

type SaveAnswerRequest struct {
	AttemptID      string `json:"attempt_id" binding:"required,uuid"`
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	SelectedAnswer string `json:"selected_answer" binding:"required,oneof=A B C D"`
	TimeSpent      int    `json:"time_spent"`
	IsMarked       bool   `json:"is_marked"`
}

type BulkSaveAnswerRequest struct {
	AttemptID  string           `json:"attempt_id" binding:"required,uuid"`
	Answers    []BulkAnswerItem `json:"answers" binding:"required"`
	DeviceInfo DeviceInfo       `json:"device_info"`
}

type BulkAnswerItem struct {
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	SelectedAnswer string `json:"selected_answer" binding:"required,oneof=A B C D"`
	TimeSpent      int    `json:"time_spent"`
	IsMarked       bool   `json:"is_marked"`
}

type SubmitExamRequest struct {
	AttemptID  string     `json:"attempt_id" binding:"required,uuid"`
	DeviceInfo DeviceInfo `json:"device_info"`
	Signature  string     `json:"signature"`
}

type SyncAnswersRequest struct {
	AttemptID  string           `json:"attempt_id" binding:"required,uuid"`
	Answers    []SyncAnswerItem `json:"answers" binding:"required"`
	SyncToken  string           `json:"sync_token"`
	DeviceInfo DeviceInfo       `json:"device_info"`
}

type SyncAnswerItem struct {
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	SelectedAnswer string `json:"selected_answer" binding:"required,oneof=A B C D"`
	TimeSpent      int    `json:"time_spent"`
	IsMarked       bool   `json:"is_marked"`
}

type AutoSaveRequest struct {
	AttemptID  string          `json:"attempt_id" binding:"required,uuid"`
	Answers    []AutoSaveItem  `json:"answers"`
	Progress   ProgressData    `json:"progress"`
	Timer      TimerData       `json:"timer"`
	SyncStatus SyncStatusData  `json:"sync_status"`
}

type AutoSaveItem struct {
	QuestionID     string `json:"question_id"`
	SelectedAnswer string `json:"selected_answer"`
	TimeSpent      int    `json:"time_spent"`
	IsMarked       bool   `json:"is_marked"`
}

type MarkReviewRequest struct {
	AttemptID  string `json:"attempt_id" binding:"required,uuid"`
	QuestionID string `json:"question_id" binding:"required,uuid"`
	IsMarked   bool   `json:"is_marked"`
}

type OfflineSubmitRequest struct {
	AttemptID       string              `json:"attempt_id" binding:"required,uuid"`
	Answers         []OfflineAnswerItem `json:"answers"`
	FinalSubmission bool                `json:"final_submission"`
	SubmittedAt     string              `json:"submitted_at"`
	Signature       string              `json:"signature"`
	DeviceInfo      DeviceInfo          `json:"device_info"`
}

type OfflineAnswerItem struct {
	QuestionID     string `json:"question_id"`
	SelectedAnswer string `json:"selected_answer"`
	TimeSpent      int    `json:"time_spent"`
	IsMarked       bool   `json:"is_marked"`
}

type ProctoringViolationRequest struct {
	AttemptID     string `json:"attempt_id" binding:"required,uuid"`
	ViolationType string `json:"violation_type" binding:"required,oneof=tab_switch copy_paste screenshot face_missing"`
	Details       string `json:"details"`
}

// ============================================
// RESPONSE DTOs - EXAM MANAGEMENT
// ============================================

type ExamResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	SubjectID        string     `json:"subject_id"`
	SubjectName      string     `json:"subject_name"`
	ClassID          *string    `json:"class_id,omitempty"`
	ClassName        string     `json:"class_name,omitempty"`
	SessionID        string     `json:"session_id"`
	SessionName      string     `json:"session_name"`
	TermID           string     `json:"term_id"`
	TermName         string     `json:"term_name"`
	DurationMinutes  int        `json:"duration_minutes"`
	TotalMarks       int        `json:"total_marks"`
	PassMark         int        `json:"pass_mark"`
	Instructions     string     `json:"instructions"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	ShuffleOptions   bool       `json:"shuffle_options"`
	IsActive         bool       `json:"is_active"`
	QuestionCount    int        `json:"question_count"`
	Status           string     `json:"status"`
	ExamType         string     `json:"exam_type"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ExamListResponse struct {
	Exams      []ExamResponse `json:"exams"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

// ============================================
// RESPONSE DTOs - EXAM TAKING
// ============================================

type ExamAttemptResponse struct {
	ID             string     `json:"id"`
	StudentID      string     `json:"student_id"`
	ExamID         string     `json:"exam_id"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Score          *int       `json:"score,omitempty"`
	Percentage     *float64   `json:"percentage,omitempty"`
	Status         string     `json:"status"`
	TimeRemaining  int        `json:"time_remaining"`
	AnsweredCount  int        `json:"answered_count"`
	TotalQuestions int        `json:"total_questions"`
	CreatedAt      time.Time  `json:"created_at"`
}

type StartExamResponse struct {
	Attempt      *ExamAttemptResponse `json:"attempt"`
	Exam         *ExamDetailResponse  `json:"exam"`
	Questions    []QuestionResponse   `json:"questions"`
	ProctoringID string               `json:"proctoring_id,omitempty"`
}

type ExamDetailResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	SubjectID        string     `json:"subject_id"`
	SubjectName      string     `json:"subject_name"`
	DurationMinutes  int        `json:"duration_minutes"`
	TotalMarks       int        `json:"total_marks"`
	PassMark         int        `json:"pass_mark"`
	Instructions     string     `json:"instructions"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	ShuffleOptions   bool       `json:"shuffle_options"`
}

type QuestionResponse struct {
	ID           string `json:"id"`
	QuestionText string `json:"question_text"`
	OptionA      string `json:"option_a"`
	OptionB      string `json:"option_b"`
	OptionC      string `json:"option_c"`
	OptionD      string `json:"option_d"`
	Marks        int    `json:"marks"`
	SortOrder    int    `json:"sort_order"`
}

type SaveAnswerResponse struct {
	QuestionID     string      `json:"question_id"`
	SelectedAnswer string      `json:"selected_answer"`
	IsAnswered     bool        `json:"is_answered"`
	IsMarked       bool        `json:"is_marked"`
	TimeSpent      int         `json:"time_spent"`
	SavedAt        time.Time   `json:"saved_at"`
	Progress       ProgressData `json:"progress"`
}

type SubmitExamResponse struct {
	AttemptID        string    `json:"attempt_id"`
	ResultID         string    `json:"result_id"`
	Score            int       `json:"score"`
	TotalMarks       int       `json:"total_marks"`
	Percentage       float64   `json:"percentage"`
	Grade            string    `json:"grade"`
	Passed           bool      `json:"passed"`
	TimeTakenMinutes int       `json:"time_taken_minutes"`
	Status           string    `json:"status"`
	SubmittedAt      time.Time `json:"submitted_at"`
}

type BulkSaveAnswerResponse struct {
	SyncedCount int          `json:"synced_count"`
	FailedCount int          `json:"failed_count"`
	Errors      []string     `json:"errors"`
	Progress    ProgressData `json:"progress"`
	SyncID      string       `json:"sync_id"`
	SyncedAt    time.Time    `json:"synced_at"`
}

type AutoSaveResponse struct {
	AttemptID         string       `json:"attempt_id"`
	SavedAt           time.Time    `json:"saved_at"`
	SavedCount        int          `json:"saved_count"`
	TotalAnswersSaved int          `json:"total_answers_saved"`
	NextSaveAt        time.Time    `json:"next_save_at"`
	LocalStorageKey   string       `json:"local_storage_key"`
	SyncStatus        SyncStatusData `json:"sync_status"`
}

type SyncAnswersResponse struct {
	AttemptID     string    `json:"attempt_id"`
	SyncedCount   int       `json:"synced_count"`
	FailedCount   int       `json:"failed_count"`
	Errors        []string  `json:"errors"`
	SyncTimestamp time.Time `json:"sync_timestamp"`
	SyncID        string    `json:"sync_id"`
	NextSyncAfter time.Time `json:"next_sync_after"`
	SyncStatus    string    `json:"sync_status"`
}

type OfflineSubmitResponse struct {
	AttemptID        string    `json:"attempt_id"`
	SubmissionStatus string    `json:"submission_status"`
	ResultID         *string   `json:"result_id"`
	Message          string    `json:"message"`
	SubmittedAt      time.Time `json:"submitted_at"`
	NextSync         time.Time `json:"next_sync"`
}

type ProctoringStatusResponse struct {
	IsActive        bool      `json:"is_active"`
	ViolationsCount int       `json:"violations_count"`
	Status          string    `json:"status"`
	StartedAt       time.Time `json:"started_at"`
}


// Add these after the ProctoringStatusResponse or before helper types

// MarkReviewResponse - Response for marking a question for review
type MarkReviewResponse struct {
    QuestionID string      `json:"question_id"`
    IsMarked   bool        `json:"is_marked"`
    MarkedAt   time.Time   `json:"marked_at"`
    Progress   ProgressData `json:"progress"`
}

// PracticeSessionResponse - Response for practice session
type PracticeSessionResponse struct {
    ID             string     `json:"id"`
    SubjectID      string     `json:"subject_id"`
    SubjectName    string     `json:"subject_name"`
    TotalQuestions int        `json:"total_questions"`
    Answered       int        `json:"answered"`
    Score          int        `json:"score"`
    Status         string     `json:"status"`
    StartedAt      time.Time  `json:"started_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// ============================================
// STUDENT DASHBOARD RESPONSE DTOs
// ============================================

type StudentDashboardResponse struct {
	Student         StudentInfo           `json:"student"`
	Summary         DashboardSummary      `json:"summary"`
	AvailableExams  []AvailableExam       `json:"available_exams"`
	InProgressExams []InProgressExam      `json:"in_progress_exams"`
	CompletedExams  []CompletedExam       `json:"completed_exams"`
	UpcomingExams   []StudentUpcomingExam `json:"upcoming_exams"`
}

type StudentInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	AdmissionNumber string `json:"admission_number"`
	Class           string `json:"class"`
	ClassID         string `json:"class_id"`
	School          string `json:"school"`
	SchoolID        string `json:"school_id"`
}

type DashboardSummary struct {
	TotalAvailable  int     `json:"total_available"`
	TotalInProgress int     `json:"total_in_progress"`
	TotalCompleted  int     `json:"total_completed"`
	TotalUpcoming   int     `json:"total_upcoming"`
	OverallAverage  float64 `json:"overall_average"`
}

type AvailableExam struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	ExamType         string       `json:"exam_type"`
	Subject          SubjectInfo  `json:"subject"`
	DurationMinutes  int          `json:"duration_minutes"`
	TotalMarks       int          `json:"total_marks"`
	PassMark         int          `json:"pass_mark"`
	Schedule         ScheduleInfo `json:"schedule"`
	Status           string       `json:"status"`
	OfflineAvailable bool         `json:"offline_available"`
	HasStarted       bool         `json:"has_started"`
	HasCompleted     bool         `json:"has_completed"`
	Instructions     string       `json:"instructions,omitempty"`
}

type InProgressExam struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	ExamType         string       `json:"exam_type"`
	Subject          SubjectInfo  `json:"subject"`
	DurationMinutes  int          `json:"duration_minutes"`
	TotalMarks       int          `json:"total_marks"`
	PassMark         int          `json:"pass_mark"`
	Schedule         ScheduleInfo `json:"schedule"`
	Attempt          AttemptInfo  `json:"attempt"`
	OfflineAvailable bool         `json:"offline_available"`
	HasStarted       bool         `json:"has_started"`
	HasCompleted     bool         `json:"has_completed"`
}

type CompletedExam struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	ExamType         string       `json:"exam_type"`
	Subject          SubjectInfo  `json:"subject"`
	DurationMinutes  int          `json:"duration_minutes"`
	TotalMarks       int          `json:"total_marks"`
	PassMark         int          `json:"pass_mark"`
	Schedule         ScheduleInfo `json:"schedule"`
	Result           ResultInfo   `json:"result"`
	HasCompleted     bool         `json:"has_completed"`
	CanReview        bool         `json:"can_review"`
	ReviewAvailable  bool         `json:"review_available"`
}

type StudentUpcomingExam struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	ExamType         string       `json:"exam_type"`
	Subject          SubjectInfo  `json:"subject"`
	DurationMinutes  int          `json:"duration_minutes"`
	TotalMarks       int          `json:"total_marks"`
	PassMark         int          `json:"pass_mark"`
	Schedule         ScheduleInfo `json:"schedule"`
	Status           string       `json:"status"`
	DaysUntil        int          `json:"days_until"`
	OfflineAvailable bool         `json:"offline_available"`
	HasStarted       bool         `json:"has_started"`
	HasCompleted     bool         `json:"has_completed"`
	Instructions     string       `json:"instructions,omitempty"`
}

type SubjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type ScheduleInfo struct {
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	IsAvailable bool       `json:"is_available"`
}

type AttemptInfo struct {
	AttemptID          string    `json:"attempt_id"`
	StartTime          time.Time `json:"start_time"`
	TimeRemaining      int       `json:"time_remaining"`
	TimeElapsed        int       `json:"time_elapsed"`
	AnsweredCount      int       `json:"answered_count"`
	TotalQuestions     int       `json:"total_questions"`
	ProgressPercentage int       `json:"progress_percentage"`
	MarkedForReview    int       `json:"marked_for_review"`
	LastActivity       time.Time `json:"last_activity"`
}

type ResultInfo struct {
	Score       int       `json:"score"`
	Percentage  float64   `json:"percentage"`
	Grade       string    `json:"grade"`
	Passed      bool      `json:"passed"`
	CompletedAt time.Time `json:"completed_at"`
}

// ============================================
// TEACHER DASHBOARD DTOs
// ============================================

type TeacherDashboardResponse struct {
	Teacher          TeacherInfo           `json:"teacher"`
	ClassPerformance []ClassPerformance    `json:"class_performance"`
	StudentTrends    StudentTrends         `json:"student_performance_trends"`
	UpcomingExams    []TeacherUpcomingExam `json:"upcoming_exams"`
	RecentActivities []RecentActivity      `json:"recent_activities"`
}

type TeacherInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	SubjectsTaught []string `json:"subjects_taught"`
}

type ClassPerformance struct {
	ClassName       string  `json:"class_name"`
	Subject         string  `json:"subject"`
	TotalStudents   int     `json:"total_students"`
	AverageScore    float64 `json:"average_score"`
	PassRate        float64 `json:"pass_rate"`
	TopPerformer    string  `json:"top_performer"`
	BottomPerformer string  `json:"bottom_performer"`
}

type StudentTrends struct {
	Improving int `json:"improving"`
	Declining int `json:"declining"`
	Stable    int `json:"stable"`
	AtRisk    int `json:"at_risk"`
}

type TeacherUpcomingExam struct {
	ExamTitle         string `json:"exam_title"`
	Date              string `json:"date"`
	StudentsScheduled int    `json:"students_scheduled"`
}

type RecentActivity struct {
	Action  string `json:"action"`
	Student string `json:"student"`
	Exam    string `json:"exam"`
	Score   int    `json:"score"`
	Time    string `json:"time"`
}

// ============================================
// RESPONSE DTOs - RESULTS & PERFORMANCE
// ============================================

type ClassResultResponse struct {
	Exam       ExamSummary       `json:"exam"`
	Class      ClassSummary      `json:"class"`
	Students   []StudentResult   `json:"students"`
	Statistics ResultStatistics  `json:"statistics"`
	Pagination PaginationInfo    `json:"pagination"`
}

type ExamSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	TotalMarks  int    `json:"total_marks"`
	PassMark    int    `json:"pass_mark"`
}

type ClassSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StudentResult struct {
	StudentID        string     `json:"student_id"`
	StudentName      string     `json:"student_name"`
	AdmissionNumber  string     `json:"admission_number"`
	Score            int        `json:"score"`
	Percentage       float64    `json:"percentage"`
	Grade            string     `json:"grade"`
	Passed           bool       `json:"passed"`
	Rank             int        `json:"rank"`
	AttemptStatus    string     `json:"attempt_status"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	TimeTakenMinutes int        `json:"time_taken_minutes"`
	Remarks          string     `json:"remarks,omitempty"`
}

type ResultStatistics struct {
	TotalStudents     int             `json:"total_students"`
	Attempted         int             `json:"attempted"`
	NotAttempted      int             `json:"not_attempted"`
	Passed            int             `json:"passed"`
	Failed            int             `json:"failed"`
	PassPercentage    float64         `json:"pass_percentage"`
	AverageScore      float64         `json:"average_score"`
	AveragePercentage float64         `json:"average_percentage"`
	HighestScore      int             `json:"highest_score"`
	LowestScore       int             `json:"lowest_score"`
	GradeDistribution map[string]int  `json:"grade_distribution"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ExamRankingResponse struct {
	ExamTitle         string         `json:"exam_title"`
	TotalParticipants int            `json:"total_participants"`
	Rankings          []RankingEntry `json:"rankings"`
}

type RankingEntry struct {
	Rank        int     `json:"rank"`
	StudentID   string  `json:"student_id"`
	StudentName string  `json:"student_name"`
	Score       int     `json:"score"`
	Percentage  float64 `json:"percentage"`
	Grade       string  `json:"grade"`
	Badge       string  `json:"badge,omitempty"`
}

type ExamStatisticsResponse struct {
	TotalAttempts     int            `json:"total_attempts"`
	CompletedCount    int            `json:"completed_count"`
	InProgressCount   int            `json:"in_progress_count"`
	PassedCount       int            `json:"passed_count"`
	FailedCount       int            `json:"failed_count"`
	AverageScore      float64        `json:"average_score"`
	PassRate          float64        `json:"pass_rate"`
	HighestScore      int            `json:"highest_score"`
	LowestScore       int            `json:"lowest_score"`
	GradeDistribution map[string]int `json:"grade_distribution"`
}

type StudentPerformanceResponse struct {
	Student            StudentInfo          `json:"student"`
	SubjectPerformance []SubjectPerformance `json:"subject_performance"`
	OverallPerformance OverallPerformance   `json:"overall_performance"`
	Attendance         AttendanceInfo       `json:"attendance"`
}

type SubjectPerformance struct {
	Subject           string        `json:"subject"`
	SubjectID         string        `json:"subject_id"`
	ExamsTaken        int           `json:"exams_taken"`
	AverageScore      float64       `json:"average_score"`
	AveragePercentage float64       `json:"average_percentage"`
	HighestScore      int           `json:"highest_score"`
	LowestScore       int           `json:"lowest_score"`
	GradeAverage      string        `json:"grade_average"`
	PerformanceTrend  string        `json:"performance_trend"`
	ExamHistory       []ExamHistory `json:"exam_history"`
}

type ExamHistory struct {
	ExamID     string    `json:"exam_id"`
	ExamTitle  string    `json:"exam_title"`
	Score      int       `json:"score"`
	Percentage float64   `json:"percentage"`
	Grade      string    `json:"grade"`
	Passed     bool      `json:"passed"`
	Date       time.Time `json:"date"`
}

type OverallPerformance struct {
	TotalExamsTaken          int     `json:"total_exams_taken"`
	OverallAveragePercentage float64 `json:"overall_average_percentage"`
	OverallGrade             string  `json:"overall_grade"`
	SubjectsPassed           int     `json:"subjects_passed"`
	SubjectsFailed           int     `json:"subjects_failed"`
	BestSubject              string  `json:"best_subject"`
	WorstSubject             string  `json:"worst_subject"`
}

type AttendanceInfo struct {
	Present             int     `json:"present"`
	Absent              int     `json:"absent"`
	AttendancePercentage float64 `json:"attendance_percentage"`
}

type StudentExamResultResponse struct {
	Result        ResultDetail    `json:"result"`
	Answers       []AnswerDetail  `json:"answers"`
	ResultSummary ResultSummary   `json:"result_summary"`
}

type ResultDetail struct {
	ID               string    `json:"id"`
	ExamTitle        string    `json:"exam_title"`
	Subject          string    `json:"subject"`
	TotalScore       int       `json:"total_score"`
	TotalMarks       int       `json:"total_marks"`
	Percentage       float64   `json:"percentage"`
	Grade            string    `json:"grade"`
	Passed           bool      `json:"passed"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	TimeTakenMinutes int       `json:"time_taken_minutes"`
}

type AnswerDetail struct {
	QuestionID     string `json:"question_id"`
	QuestionText   string `json:"question_text"`
	StudentAnswer  string `json:"student_answer"`
	CorrectAnswer  string `json:"correct_answer"`
	IsCorrect      bool   `json:"is_correct"`
	MarksObtained  int    `json:"marks_obtained"`
	TotalMarks     int    `json:"total_marks"`
}

type ResultSummary struct {
	CorrectAnswers int     `json:"correct_answers"`
	WrongAnswers   int     `json:"wrong_answers"`
	SkippedAnswers int     `json:"skipped_answers"`
	TotalQuestions int     `json:"total_questions"`
	Accuracy       float64 `json:"accuracy"`
}

// ============================================
// OFFLINE PACKAGE RESPONSE
// ============================================

type DownloadExamPackageResponse struct {
	Manifest      PackageManifest   `json:"manifest"`
	Exam          ExamPackageInfo   `json:"exam"`
	Context       PackageContext    `json:"context"`
	Questions     []PackageQuestion `json:"questions"`
	Security      PackageSecurity   `json:"security"`
	Metadata      PackageMetadata   `json:"metadata"`
	DownloadToken string            `json:"download_token"`
}

type PackageManifest struct {
	ExamID      string    `json:"exam_id"`
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Signature   string    `json:"signature"`
	Integrity   string    `json:"integrity"`
}

type ExamPackageInfo struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	ExamType        string          `json:"exam_type"`
	DurationMinutes int             `json:"duration_minutes"`
	TotalMarks      int             `json:"total_marks"`
	PassMark        int             `json:"pass_mark"`
	Instructions    string          `json:"instructions"`
	Settings        PackageSettings `json:"settings"`
}

type PackageSettings struct {
	ShuffleQuestions bool `json:"shuffle_questions"`
	ShuffleOptions   bool `json:"shuffle_options"`
	ShowTimer        bool `json:"show_timer"`
	AllowMarking     bool `json:"allow_marking"`
	AutoSaveInterval int  `json:"auto_save_interval"`
}

type PackageContext struct {
	School  SubjectInfo `json:"school"`
	Subject SubjectInfo `json:"subject"`
}

type PackageQuestion struct {
	ID           string       `json:"id"`
	QuestionText string       `json:"question_text"`
	QuestionType string       `json:"question_type"`
	Marks        int          `json:"marks"`
	SortOrder    int          `json:"sort_order"`
	Options      []OptionInfo `json:"options"`
}

type OptionInfo struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type PackageSecurity struct {
	Encrypted          bool   `json:"encrypted"`
	StorageKey         string `json:"storage_key"`
	EncryptionAlgorithm string `json:"encryption_algorithm"`
}

type PackageMetadata struct {
	TotalQuestions  int `json:"total_questions"`
	TotalMarks      int `json:"total_marks"`
	EstimatedSizeKB int `json:"estimated_size_kb"`
}

// ============================================
// SCHOOL PERFORMANCE DTOs
// ============================================

type SchoolPerformanceResponse struct {
	School             SchoolInfo          `json:"school"`
	Summary            SchoolSummary       `json:"summary"`
	PerformanceMetrics PerformanceMetrics  `json:"performance_metrics"`
	SubjectPerformance []SubjectPerf       `json:"subject_performance"`
	ClassRankings      []ClassRanking      `json:"class_rankings"`
	GenderMetrics      GenderMetrics       `json:"gender_metrics"`
}

type SchoolInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SchoolSummary struct {
	TotalStudents       int    `json:"total_students"`
	TotalTeachers       int    `json:"total_teachers"`
	TotalExamsConducted int    `json:"total_exams_conducted"`
	CurrentSession      string `json:"current_session"`
	CurrentTerm         string `json:"current_term"`
}

type PerformanceMetrics struct {
	OverallPassRate         float64 `json:"overall_pass_rate"`
	AverageClassPerformance float64 `json:"average_class_performance"`
	TopPerformingClass      string  `json:"top_performing_class"`
	LowestPerformingClass   string  `json:"lowest_performing_class"`
}

type SubjectPerf struct {
	Subject       string  `json:"subject"`
	PassRate      float64 `json:"pass_rate"`
	AverageScore  float64 `json:"average_score"`
	TotalStudents int     `json:"total_students"`
}

type ClassRanking struct {
	ClassName    string  `json:"class_name"`
	AverageScore float64 `json:"average_score"`
	Rank         int     `json:"rank"`
}

type GenderMetrics struct {
	MaleAverage   float64 `json:"male_average"`
	FemaleAverage float64 `json:"female_average"`
}

// ============================================
// PROFESSIONAL CBT FLOW DTOS
// ============================================

type CreateExamWithContextRequest struct {
	SchoolID         string       `json:"school_id" binding:"required,uuid"`
	SessionID        string       `json:"session_id" binding:"required,uuid"`
	TermID           string       `json:"term_id" binding:"required,uuid"`
	SubjectID        string       `json:"subject_id" binding:"required,uuid"`
	ClassID          string       `json:"class_id" binding:"required,uuid"`
	Title            string       `json:"title" binding:"required"`
	ExamType         string       `json:"exam_type" binding:"required,oneof=main_exam mid_term class_test practice quiz weekly_test"`
	DurationMinutes  int          `json:"duration_minutes" binding:"required,min=1"`
	TotalMarks       int          `json:"total_marks" binding:"required,min=1"`
	PassMark         int          `json:"pass_mark" binding:"required,min=0"`
	Instructions     string       `json:"instructions"`
	StartTime        *time.Time   `json:"start_time"`
	EndTime          *time.Time   `json:"end_time"`
	ShuffleQuestions bool         `json:"shuffle_questions"`
	ShuffleOptions   bool         `json:"shuffle_options"`
	IsActive         bool         `json:"is_active"`
	ReviewPolicy     ReviewPolicy `json:"review_policy"`
}

type ReviewPolicy struct {
	ReleaseMode        string `json:"release_mode"`
	ShowScore          bool   `json:"show_score"`
	ShowGrade          bool   `json:"show_grade"`
	ShowCorrectAnswers bool   `json:"show_correct_answers"`
	ShowExplanations   bool   `json:"show_explanations"`
	ShowQuestionReview bool   `json:"show_question_review"`
	AllowRetake        bool   `json:"allow_retake"`
}

type SubjectQuestionsResponse struct {
	SubjectID      string                  `json:"subject_id"`
	SubjectName    string                  `json:"subject_name"`
	ClassID        string                  `json:"class_id"`
	ClassName      string                  `json:"class_name"`
	SessionID      string                  `json:"session_id"`
	SessionName    string                  `json:"session_name"`
	TermID         string                  `json:"term_id"`
	TermName       string                  `json:"term_name"`
	TotalQuestions int                     `json:"total_questions"`
	Questions      []QuestionWithSelection `json:"questions"`
	Statistics     QuestionStatistics      `json:"statistics"`
}

type QuestionWithSelection struct {
	ID           string          `json:"id"`
	QuestionText string          `json:"question_text"`
	QuestionType string          `json:"question_type"`
	Difficulty   string          `json:"difficulty"`
	BloomLevel   string          `json:"bloom_level"`
	Marks        int             `json:"marks"`
	Topic        string          `json:"topic"`
	Selected     bool            `json:"selected"`
	Options      []OptionPreview `json:"options,omitempty"`
}

type OptionPreview struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type QuestionStatistics struct {
	ByDifficulty   map[string]int `json:"by_difficulty"`
	ByBloomLevel   map[string]int `json:"by_bloom_level"`
	ByQuestionType map[string]int `json:"by_question_type"`
	AverageMarks   float64        `json:"average_marks"`
}

type BulkAddQuestionsToExamRequest struct {
	QuestionIDs        []string `json:"question_ids,omitempty"`
	SelectAll          bool     `json:"select_all"`
	FilterByDifficulty []string `json:"filter_by_difficulty,omitempty"`
	FilterByBloom      []string `json:"filter_by_bloom,omitempty"`
	FilterByTopic      string   `json:"filter_by_topic,omitempty"`
}

type ExamPreviewResponse struct {
	ExamID          string                `json:"exam_id"`
	ExamTitle       string                `json:"exam_title"`
	SubjectID       string                `json:"subject_id"`
	SubjectName     string                `json:"subject_name"`
	ClassID         string                `json:"class_id"`
	ClassName       string                `json:"class_name"`
	TotalQuestions  int                   `json:"total_questions"`
	TotalMarks      int                   `json:"total_marks"`
	DurationMinutes int                   `json:"duration_minutes"`
	PassMark        int                   `json:"pass_mark"`
	Statistics      ExamPreviewStatistics `json:"statistics"`
	Questions       []QuestionPreviewItem `json:"questions"`
}

type ExamPreviewStatistics struct {
	TotalQuestions     int            `json:"total_questions"`
	TotalMarks         int            `json:"total_marks"`
	AverageMarks       float64        `json:"average_marks"`
	ByDifficulty       map[string]int `json:"by_difficulty"`
	ByBloomLevel       map[string]int `json:"by_bloom_level"`
	ByQuestionType     map[string]int `json:"by_question_type"`
	PassMark           int            `json:"pass_mark"`
	PassMarkPercentage float64        `json:"pass_mark_percentage"`
}

type QuestionPreviewItem struct {
	ID           string          `json:"id"`
	QuestionText string          `json:"question_text"`
	QuestionType string          `json:"question_type"`
	Difficulty   string          `json:"difficulty"`
	BloomLevel   string          `json:"bloom_level"`
	Marks        int             `json:"marks"`
	Options      []OptionPreview `json:"options,omitempty"`
}

type ExamWithContextResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	ExamType         string     `json:"exam_type"`
	SubjectID        string     `json:"subject_id"`
	SubjectName      string     `json:"subject_name"`
	ClassID          string     `json:"class_id"`
	ClassName        string     `json:"class_name"`
	SessionID        string     `json:"session_id"`
	SessionName      string     `json:"session_name"`
	TermID           string     `json:"term_id"`
	TermName         string     `json:"term_name"`
	DurationMinutes  int        `json:"duration_minutes"`
	TotalMarks       int        `json:"total_marks"`
	PassMark         int        `json:"pass_mark"`
	Instructions     string     `json:"instructions"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	ShuffleOptions   bool       `json:"shuffle_options"`
	IsActive         bool       `json:"is_active"`
	QuestionCount    int        `json:"question_count"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ============================================
// TERMLY RESULT DTOs (WAEC/NECO Style)
// ============================================

// TermlyResultResponse - Complete termly result for a student
type TermlyResultResponse struct {
	Student     StudentInfo          `json:"student"`
	Class       ClassInfo            `json:"class"`
	Term        TermInfo             `json:"term"`
	Session     SessionInfo          `json:"session"`
	School      SchoolInfo           `json:"school"`
	Subjects    []SubjectTermlyResult `json:"subjects"`
	Summary     TermlySummary        `json:"summary"`
	Statistics  TermlyStatistics     `json:"statistics"`
	GeneratedAt time.Time            `json:"generated_at"`
}

type ClassInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TermInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type SessionInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubjectTermlyResult - Individual subject result for a term
type SubjectTermlyResult struct {
	Subject          SubjectInfo      `json:"subject"`
	Exams            []ExamResultItem `json:"exams"`
	TotalScore       int              `json:"total_score"`
	TotalMarks       int              `json:"total_marks"`
	Percentage       float64          `json:"percentage"`
	Grade            string           `json:"grade"`
	GradePoint       float64          `json:"grade_point"`
	Position         int              `json:"position"`
	ClassAverage     float64          `json:"class_average"`
	HighestScore     int              `json:"highest_score"`
	LowestScore      int              `json:"lowest_score"`
	PerformanceTrend string           `json:"performance_trend"`
	Remarks          string           `json:"remarks"`
}

// ExamResultItem - Individual exam result within a subject
type ExamResultItem struct {
	ExamID     string    `json:"exam_id"`
	ExamTitle  string    `json:"exam_title"`
	ExamType   string    `json:"exam_type"`
	Score      int       `json:"score"`
	TotalMarks int       `json:"total_marks"`
	Percentage float64   `json:"percentage"`
	Grade      string    `json:"grade"`
	Date       time.Time `json:"date"`
	IsFinal    bool      `json:"is_final"`
}

// TermlySummary - Overall summary for the term
type TermlySummary struct {
	TotalSubjects          int     `json:"total_subjects"`
	TotalScore             int     `json:"total_score"`
	TotalMarks             int     `json:"total_marks"`
	OverallPercentage      float64 `json:"overall_percentage"`
	OverallGrade           string  `json:"overall_grade"`
	OverallGradePoint      float64 `json:"overall_grade_point"`
	NumberPassed           int     `json:"number_passed"`
	NumberFailed           int     `json:"number_failed"`
	ClassPosition          int     `json:"class_position"`
	TotalStudents          int     `json:"total_students"`
	BestPerformingSubject  string  `json:"best_performing_subject"`
	WorstPerformingSubject string  `json:"worst_performing_subject"`
}

// TermlyStatistics - Statistical data for the term
type TermlyStatistics struct {
	ClassAverage      float64            `json:"class_average"`
	ClassHighest      float64            `json:"class_highest"`
	ClassLowest       float64            `json:"class_lowest"`
	PassRate          float64            `json:"pass_rate"`
	GradeDistribution map[string]int     `json:"grade_distribution"`
	SubjectAverages   map[string]float64 `json:"subject_averages"`
	SubjectRankings   []SubjectRanking   `json:"subject_rankings"`
}

// SubjectRanking - Ranking of subjects by performance
type SubjectRanking struct {
	SubjectName  string  `json:"subject_name"`
	AverageScore float64 `json:"average_score"`
	PassRate     float64 `json:"pass_rate"`
	Rank         int     `json:"rank"`
}

// TermlyReportCardResponse - Full report card format
type TermlyReportCardResponse struct {
	School           SchoolInfo           `json:"school"`
	Student          StudentInfo          `json:"student"`
	Class            ClassInfo            `json:"class"`
	Term             TermInfo             `json:"term"`
	Session          SessionInfo          `json:"session"`
	Subjects         []ReportCardSubject  `json:"subjects"`
	Summary          ReportCardSummary    `json:"summary"`
	TeacherRemarks   string               `json:"teacher_remarks"`
	PrincipalRemarks string               `json:"principal_remarks"`
	NextTermBegins   *time.Time           `json:"next_term_begins"`
}

// ReportCardSubject - Subject entry in report card
type ReportCardSubject struct {
	SubjectName   string  `json:"subject_name"`
	SubjectCode   string  `json:"subject_code"`
	FirstTest     *int    `json:"first_test,omitempty"`
	SecondTest    *int    `json:"second_test,omitempty"`
	ThirdTest     *int    `json:"third_test,omitempty"`
	ExamScore     *int    `json:"exam_score,omitempty"`
	TotalScore    int     `json:"total_score"`
	Grade         string  `json:"grade"`
	GradePoint    float64 `json:"grade_point"`
	Position      int     `json:"position"`
	ClassAverage  float64 `json:"class_average"`
	HighestScore  int     `json:"highest_score"`
	LowestScore   int     `json:"lowest_score"`
	Remarks       string  `json:"remarks"`
}

// ReportCardSummary - Summary section of report card
type ReportCardSummary struct {
	TotalSubjects          int     `json:"total_subjects"`
	TotalScore             int     `json:"total_score"`
	TotalMarks             int     `json:"total_marks"`
	OverallPercentage      float64 `json:"overall_percentage"`
	OverallGrade           string  `json:"overall_grade"`
	OverallGradePoint      float64 `json:"overall_grade_point"`
	NumberPassed           int     `json:"number_passed"`
	NumberFailed           int     `json:"number_failed"`
	ClassPosition          int     `json:"class_position"`
	TotalStudentsInClass   int     `json:"total_students_in_class"`
	BestPerformingSubject  string  `json:"best_performing_subject"`
	WorstPerformingSubject string  `json:"worst_performing_subject"`
	Conduct                string  `json:"conduct,omitempty"`
	TeacherComment         string  `json:"teacher_comment,omitempty"`
	PrincipalComment       string  `json:"principal_comment,omitempty"`
}

// ============================================
// WAEC/NECO GRADE SCALE HELPERS
// ============================================

// GradeScale - WAEC/NECO Grade Scale
type GradeScale struct {
	Grade      string  `json:"grade"`
	ScoreRange string  `json:"score_range"`
	GradePoint float64 `json:"grade_point"`
	Remark     string  `json:"remark"`
}

// GetGrade - Get grade based on percentage
func GetGrade(percentage float64) (grade string, gradePoint float64, remark string) {
	switch {
	case percentage >= 70:
		return "A", 5.0, "Excellent"
	case percentage >= 60:
		return "B", 4.0, "Very Good"
	case percentage >= 50:
		return "C", 3.0, "Good"
	case percentage >= 45:
		return "D", 2.0, "Fair"
	case percentage >= 40:
		return "E", 1.0, "Pass"
	default:
		return "F", 0.0, "Fail"
	}
}

// CalculateGradeFromScore - Calculate grade from score and total marks
func CalculateGradeFromScore(score, totalMarks int) (string, float64, string) {
	if totalMarks == 0 {
		return "F", 0.0, "Fail"
	}
	percentage := float64(score) / float64(totalMarks) * 100
	return GetGrade(percentage)
}

// ============================================
// HELPER TYPES
// ============================================

type DeviceInfo struct {
	UserAgent          string `json:"user_agent"`
	Platform           string `json:"platform"`
	ScreenResolution   string `json:"screen_resolution"`
	SyncedFrom         string `json:"synced_from,omitempty"`
	SyncTimestamp      string `json:"sync_timestamp,omitempty"`
	PreviousSync       string `json:"previous_sync,omitempty"`
	SubmittedFrom      string `json:"submitted_from,omitempty"`
	SubmissionTimestamp string `json:"submission_timestamp,omitempty"`
}

type ProgressData struct {
	Total           int `json:"total"`
	Answered        int `json:"answered"`
	Unanswered      int `json:"unanswered"`
	MarkedForReview int `json:"marked_for_review"`
	Percentage      int `json:"percentage"`
}

type TimerData struct {
	Elapsed   int `json:"elapsed"`
	Remaining int `json:"remaining"`
}

type SyncStatusData struct {
	Online      bool       `json:"online"`
	PendingSync int        `json:"pending_sync"`
	LastSync    *time.Time `json:"last_sync"`
}

type QuestionStatus struct {
	QuestionID string `json:"question_id"`
	IsAnswered bool   `json:"is_answered"`
	IsMarked   bool   `json:"is_marked"`
	TimeSpent  int    `json:"time_spent"`
	Status     string `json:"status"`
}

type GetProgressResponse struct {
	AttemptID      string          `json:"attempt_id"`
	Status         string          `json:"status"`
	Timer          TimerInfo       `json:"timer"`
	Progress       ProgressData    `json:"progress"`
	QuestionStatus []QuestionStatus `json:"question_status"`
	Pace           PaceInfo        `json:"pace"`
}

type TimerInfo struct {
	Remaining              int     `json:"remaining"`
	Elapsed                int     `json:"elapsed"`
	TimeRemainingFormatted string  `json:"time_remaining_formatted"`
	TimeElapsedFormatted   string  `json:"time_elapsed_formatted"`
	PercentageRemaining    float64 `json:"percentage_remaining"`
	IsWarning              bool    `json:"is_warning"`
	IsCritical             bool    `json:"is_critical"`
	WarningThreshold       int     `json:"warning_threshold"`
	CriticalThreshold      int     `json:"critical_threshold"`
}

type PaceInfo struct {
	AnsweredPerMinute   float64 `json:"answered_per_minute"`
	ProjectedCompletion string  `json:"projected_completion"`
	IsOnTrack           bool    `json:"is_on_track"`
	RecommendedPace     float64 `json:"recommended_pace"`
	BehindBy            int     `json:"behind_by"`
}

type GetAttemptStateResponse struct {
	Attempt    AttemptState    `json:"attempt"`
	Questions  []QuestionState `json:"questions"`
	Progress   ProgressData    `json:"progress"`
	Timer      TimerInfo       `json:"timer"`
	SyncStatus SyncStatusData  `json:"sync_status"`
}

type AttemptState struct {
	ID                   string    `json:"id"`
	Status               string    `json:"status"`
	StartTime            time.Time `json:"start_time"`
	TimeRemaining        int       `json:"time_remaining"`
	TimeElapsed          int       `json:"time_elapsed"`
	TotalQuestions       int       `json:"total_questions"`
	AnsweredCount        int       `json:"answered_count"`
	MarkedForReviewCount int       `json:"marked_for_review_count"`
	ProgressPercentage   int       `json:"progress_percentage"`
	LastActivity         time.Time `json:"last_activity"`
}

type QuestionState struct {
	ID             string       `json:"id"`
	QuestionText   string       `json:"question_text"`
	QuestionType   string       `json:"question_type"`
	Marks          int          `json:"marks"`
	SortOrder      int          `json:"sort_order"`
	Options        []OptionInfo `json:"options"`
	SelectedAnswer string       `json:"selected_answer"`
	IsAnswered     bool         `json:"is_answered"`
	IsMarked       bool         `json:"is_marked"`
	TimeSpent      int          `json:"time_spent"`
}

type GetAnswersResponse struct {
	AttemptID    string        `json:"attempt_id"`
	Answers      []AnswerState `json:"answers"`
	TotalAnswers int           `json:"total_answers"`
	SyncedCount  int           `json:"synced_count"`
	PendingCount int           `json:"pending_count"`
	SyncStatus   string        `json:"sync_status"`
}

type AnswerState struct {
	QuestionID     string    `json:"question_id"`
	SelectedAnswer string    `json:"selected_answer"`
	TimeSpent      int       `json:"time_spent"`
	IsMarked       bool      `json:"is_marked"`
	SavedAt        time.Time `json:"saved_at"`
	Synced         bool      `json:"synced"`
}

type CompletedExamsResponse struct {
	Exams      []CompletedExamDetail `json:"exams"`
	Statistics CompletedStats        `json:"statistics"`
}

type CompletedExamDetail struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	ExamType        string              `json:"exam_type"`
	Subject         SubjectInfo         `json:"subject"`
	DurationMinutes int                 `json:"duration_minutes"`
	TotalMarks      int                 `json:"total_marks"`
	PassMark        int                 `json:"pass_mark"`
	Attempt         CompletedAttemptInfo `json:"attempt"`
	Result          ResultInfo          `json:"result"`
	CanReview       bool                `json:"can_review"`
	ReviewAvailable bool                `json:"review_available"`
}

type CompletedAttemptInfo struct {
	AttemptID string    `json:"attempt_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	TimeTaken int       `json:"time_taken"`
}

type CompletedStats struct {
	TotalCompleted int     `json:"total_completed"`
	AverageScore   float64 `json:"average_score"`
	PassRate       float64 `json:"pass_rate"`
	HighestScore   int     `json:"highest_score"`
	LowestScore    int     `json:"lowest_score"`
}

type InProgressExamsResponse struct {
	Exams           []InProgressExam `json:"exams"`
	TotalInProgress int              `json:"total_in_progress"`
}

// ============================================
// EXAM TYPE CONFIGURATION - ADD TO dto/exam_dto.go
// ============================================

const (
    MinQuestionsWeeklyTest = 10
    MaxQuestionsWeeklyTest = 20
    MinQuestionsMidTerm    = 30
    MaxQuestionsMidTerm    = 40
    MinQuestionsMainExam   = 40
    MaxQuestionsMainExam   = 60
    MinQuestionsPractice   = 5
    MaxQuestionsPractice   = 20
    MinQuestionsQuiz       = 5
    MaxQuestionsQuiz       = 15
    MinQuestionsClassTest  = 10
    MaxQuestionsClassTest  = 25
)

// GetQuestionCountRange returns the valid question count range for an exam type
func GetQuestionCountRange(examType string) (min, max int) {
    switch examType {
    case "weekly_test":
        return MinQuestionsWeeklyTest, MaxQuestionsWeeklyTest
    case "mid_term":
        return MinQuestionsMidTerm, MaxQuestionsMidTerm
    case "main_exam":
        return MinQuestionsMainExam, MaxQuestionsMainExam
    case "practice":
        return MinQuestionsPractice, MaxQuestionsPractice
    case "quiz":
        return MinQuestionsQuiz, MaxQuestionsQuiz
    case "class_test":
        return MinQuestionsClassTest, MaxQuestionsClassTest
    default:
        return 5, 100 // Safe fallback
    }
}