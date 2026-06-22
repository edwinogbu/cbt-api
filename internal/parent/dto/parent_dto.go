package dto

import "time"

// ParentRegisterRequest is used when a parent registers (will be embedded in auth register)
type ParentRegisterRequest struct {
	AdmissionNumber string `json:"admission_number" binding:"required"`
}

// ChildResponse represents a linked child for the parent dashboard
type ChildResponse struct {
	StudentID    string `json:"student_id"`
	FullName     string `json:"full_name"`
	AdmissionNo  string `json:"admission_no"`
	ClassName    string `json:"class_name"`
	ClassTeacher string `json:"class_teacher"`
}

// ExamResultResponse for a child's exam result
type ExamResultResponse struct {
	ExamID      string    `json:"exam_id"`
	ExamTitle   string    `json:"exam_title"`
	Subject     string    `json:"subject"`
	Score       float64   `json:"score"`
	TotalScore  float64   `json:"total_score"`
	Percentage  float64   `json:"percentage"`
	Grade       string    `json:"grade"`
	TakenAt     time.Time `json:"taken_at"`
}

// ChildResultResponse includes child info and exam results
type ChildResultResponse struct {
	Child      ChildResponse       `json:"child"`
	Results    []ExamResultResponse `json:"results"`
}