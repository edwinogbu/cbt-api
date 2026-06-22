package dto

import "time"

// Request DTOs
type CreateClassRequest struct {
    SchoolID     string  `json:"school_id" binding:"required"`
    SessionID    string  `json:"session_id" binding:"required"`
    ClassLevelID string  `json:"class_level_id" binding:"required"`
    ClassArmID   string  `json:"class_arm_id" binding:"required"`
    TeacherID    *string `json:"teacher_id"`
    RoomNumber   string  `json:"room_number"`
}

type UpdateClassRequest struct {
    TeacherID    *string `json:"teacher_id"`
    RoomNumber   string  `json:"room_number"`
    IsActive     *bool   `json:"is_active"`
}

// Response DTOs
type ClassResponse struct {
    ID           string              `json:"id"`
    SchoolID     string              `json:"school_id"`
    SessionID    string              `json:"session_id"`
    ClassLevel   ClassLevelBriefDTO  `json:"class_level"`
    ClassArm     ClassArmBriefDTO    `json:"class_arm"`
    ClassCode    string              `json:"class_code"`
    TeacherID    *string             `json:"teacher_id"`
    RoomNumber   string              `json:"room_number"`
    StudentCount int                 `json:"student_count"`
    IsActive     bool                `json:"is_active"`
    CreatedAt    time.Time           `json:"created_at"`
    UpdatedAt    time.Time           `json:"updated_at"`
}

type ClassLevelBriefDTO struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    LevelNumber int    `json:"level_number"`
    Category    string `json:"category"`
}

type ClassArmBriefDTO struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    ArmCode string `json:"arm_code"`
}