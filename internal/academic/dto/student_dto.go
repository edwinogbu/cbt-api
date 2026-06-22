package dto

import "time"

// Request DTOs
type CreateStudentRequest struct {
    UserID        string     `json:"user_id" binding:"required"`
    SchoolID      string     `json:"school_id" binding:"required"`
    ClassID       string     `json:"class_id"`
    AdmissionNo   string     `json:"admission_no" binding:"required"`
    DateOfBirth   *time.Time `json:"date_of_birth"`
    Gender        string     `json:"gender" binding:"omitempty,oneof=Male Female Other"`
    Address       string     `json:"address"`
    GuardianName  string     `json:"guardian_name"`
    GuardianPhone string     `json:"guardian_phone"`
    GuardianEmail string     `json:"guardian_email" binding:"omitempty,email"`
}

type UpdateStudentRequest struct {
    ClassID       *string    `json:"class_id"`
    DateOfBirth   *time.Time `json:"date_of_birth"`
    Gender        string     `json:"gender"`
    Address       string     `json:"address"`
    GuardianName  string     `json:"guardian_name"`
    GuardianPhone string     `json:"guardian_phone"`
    GuardianEmail string     `json:"guardian_email"`
    IsActive      *bool      `json:"is_active"`
}

// Response DTOs
type StudentResponse struct {
    ID            string     `json:"id"`
    UserID        string     `json:"user_id"`
    SchoolID      string     `json:"school_id"`
    ClassID       string     `json:"class_id"`
    AdmissionNo   string     `json:"admission_no"`
    DateOfBirth   *time.Time `json:"date_of_birth"`
    Gender        string     `json:"gender"`
    Address       string     `json:"address"`
    GuardianName  string     `json:"guardian_name"`
    GuardianPhone string     `json:"guardian_phone"`
    GuardianEmail string     `json:"guardian_email"`
    IsActive      bool       `json:"is_active"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

type StudentDetailResponse struct {
    StudentResponse
    User UserBriefDTO `json:"user"`
}

type UserBriefDTO struct {
    ID          string `json:"id"`
    Email       string `json:"email"`
    FirstName   string `json:"first_name"`
    LastName    string `json:"last_name"`
    PhoneNumber string `json:"phone_number"`
}