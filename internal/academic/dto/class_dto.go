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

// ============================================
// NEW DTO TYPES FOR MISSING ENDPOINTS
// ============================================

// ClassBulkDeleteRequest - for bulk delete
type ClassBulkDeleteRequest struct {
    IDs []string `json:"ids" binding:"required"`
}

// ClassStatsResponse - statistics response
type ClassStatsResponse struct {
    Total          int64            `json:"total"`
    Active         int64            `json:"active"`
    Inactive       int64            `json:"inactive"`
    BySchool       map[string]int64 `json:"by_school"`
    ByLevel        map[string]int64 `json:"by_level"`
    ByArm          map[string]int64 `json:"by_arm"`
    TotalStudents  int64            `json:"total_students"`
    AvgPerClass    float64          `json:"avg_per_class"`
}

// ListClassesRequest - ALL PARAMETERS OPTIONAL
type ListClassesRequest struct {
    Page         int    `form:"page" json:"page"`
    Limit        int    `form:"limit" json:"limit"`
    Search       string `form:"search" json:"search"`
    SchoolID     string `form:"school_id" json:"school_id"`
    SessionID    string `form:"session_id" json:"session_id"`
    ClassLevelID string `form:"class_level_id" json:"class_level_id"`
    ClassArmID   string `form:"class_arm_id" json:"class_arm_id"`
    TeacherID    string `form:"teacher_id" json:"teacher_id"`
    IsActive     *bool  `form:"is_active" json:"is_active"`
    SortBy       string `form:"sort_by" json:"sort_by"`
    SortOrder    string `form:"sort_order" json:"sort_order"`
}

// ClassListResponse - paginated list response
type ClassListResponse struct {
    Items      []ClassResponse `json:"items"`
    Total      int64           `json:"total"`
    Page       int             `json:"page"`
    Limit      int             `json:"limit"`
    TotalPages int             `json:"total_pages"`
}