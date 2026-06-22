package dto

import (
    "time"
    // "github.com/google/uuid"
)

// ============================================
// REQUEST DTOs
// ============================================

type StartExamRequest struct {
    ExamID       string `json:"exam_id" binding:"required,uuid"`
    StudentID    string `json:"student_id" binding:"required,uuid"`
    DeviceInfo   string `json:"device_info"`
    IPAddress    string `json:"ip_address"`
}

type SaveAnswerRequest struct {
    AttemptID      string `json:"attempt_id" binding:"required,uuid"`
    QuestionID     string `json:"question_id" binding:"required,uuid"`
    SelectedAnswer string `json:"selected_answer" binding:"required,oneof=A B C D"`
    TimeSpent      int    `json:"time_spent"`
}

type SubmitExamRequest struct {
    AttemptID string `json:"attempt_id" binding:"required,uuid"`
}

type SyncAnswersRequest struct {
    DeviceID  string           `json:"device_id" binding:"required"`
    Answers   []SyncAnswerItem `json:"answers" binding:"required"`
}

type SyncAnswerItem struct {
    QuestionID     string `json:"question_id" binding:"required,uuid"`
    SelectedAnswer string `json:"selected_answer" binding:"required,oneof=A B C D"`
    TimeSpent      int    `json:"time_spent"`
}

type ProctoringViolationRequest struct {
    AttemptID     string `json:"attempt_id" binding:"required,uuid"`
    ViolationType string `json:"violation_type" binding:"required,oneof=tab_switch copy_paste screenshot face_missing"`
    Details       string `json:"details"`
}

// ============================================
// RESPONSE DTOs
// ============================================

type ExamAttemptResponse struct {
    ID            string     `json:"id"`
    StudentID     string     `json:"student_id"`
    ExamID        string     `json:"exam_id"`
    StartTime     time.Time  `json:"start_time"`
    EndTime       *time.Time `json:"end_time,omitempty"`
    Score         *int       `json:"score,omitempty"`
    Percentage    *float64   `json:"percentage,omitempty"`
    Status        string     `json:"status"`
    TimeRemaining int        `json:"time_remaining"`
    AnsweredCount int        `json:"answered_count"`
    TotalQuestions int       `json:"total_questions"`
    CreatedAt     time.Time  `json:"created_at"`
}

type StartExamResponse struct {
    Attempt      *ExamAttemptResponse   `json:"attempt"`
    Exam         *ExamDetailResponse    `json:"exam"`
    Questions    []QuestionResponse     `json:"questions"`
    ProctoringID string                 `json:"proctoring_id,omitempty"`
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
    IsCorrect     bool   `json:"is_correct"`
    CorrectAnswer string `json:"correct_answer,omitempty"`
    Explanation   string `json:"explanation,omitempty"`
}

type SubmitExamResponse struct {
    AttemptID    string  `json:"attempt_id"`
    Score        int     `json:"score"`
    TotalMarks   int     `json:"total_marks"`
    Percentage   float64 `json:"percentage"`
    Grade        string  `json:"grade"`
    Passed       bool    `json:"passed"`
    ResultID     string  `json:"result_id,omitempty"`
}

type SyncAnswersResponse struct {
    SyncedCount int      `json:"synced_count"`
    FailedCount int      `json:"failed_count"`
    Errors      []string `json:"errors,omitempty"`
}

type ProctoringStatusResponse struct {
    IsActive        bool      `json:"is_active"`
    ViolationsCount int       `json:"violations_count"`
    Status          string    `json:"status"`
    StartedAt       time.Time `json:"started_at"`
}

type PracticeSessionResponse struct {
    ID             string    `json:"id"`
    SubjectID      string    `json:"subject_id"`
    SubjectName    string    `json:"subject_name"`
    TotalQuestions int       `json:"total_questions"`
    Answered       int       `json:"answered"`
    Score          int       `json:"score"`
    Status         string    `json:"status"`
    StartedAt      time.Time `json:"started_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}