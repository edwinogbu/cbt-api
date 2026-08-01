package dto

import "time"

// Request DTOs
type CreateSessionRequest struct {
    SchoolID  string    `json:"school_id" binding:"required"`
    Name      string    `json:"name" binding:"required"`
    StartDate time.Time `json:"start_date" binding:"required"`
    EndDate   time.Time `json:"end_date" binding:"required"`
    IsCurrent bool      `json:"is_current"`
}

type UpdateSessionRequest struct {
    Name      string     `json:"name"`
    StartDate *time.Time `json:"start_date"`
    EndDate   *time.Time `json:"end_date"`
    IsCurrent *bool      `json:"is_current"`
    IsActive  *bool      `json:"is_active"`
}

// Response DTOs
type SessionResponse struct {
    ID        string    `json:"id"`
    SchoolID  string    `json:"school_id"`
    Name      string    `json:"name"`
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
    IsCurrent bool      `json:"is_current"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// internal/academic/dto/session_dto.go

// Add these new DTOs

// ListSessionsRequest for filtering and pagination
type ListSessionsRequest struct {
    Page       int    `form:"page" binding:"omitempty,min=1"`
    Limit      int    `form:"limit" binding:"omitempty,min=1,max=100"`
    Search     string `form:"search"`
    SchoolID   string `form:"school_id"`
    Status     string `form:"status" binding:"omitempty,oneof=active upcoming ended inactive all"`
    SortBy     string `form:"sort_by" binding:"omitempty,oneof=name start_date end_date created_at"`
    SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
    StartDate  string `form:"start_date"`
    EndDate    string `form:"end_date"`
    IsActive   *bool  `form:"is_active"`
}

// SessionStatsResponse for statistics
type SessionStatsResponse struct {
    Total    int64 `json:"total"`
    Active   int64 `json:"active"`
    Upcoming int64 `json:"upcoming"`
    Ended    int64 `json:"ended"`
    Inactive int64 `json:"inactive"`
}

// SessionListResponse for paginated list
type SessionListResponse struct {
    Items      []SessionResponse     `json:"items"`
    Pagination PaginationMeta        `json:"pagination"`
    Stats      *SessionStatsResponse `json:"stats,omitempty"`
}

// PaginationMeta for pagination info
type PaginationMeta struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}

// SessionSummaryResponse for dashboard overview
type SessionSummaryResponse struct {
    Current    *SessionResponse     `json:"current,omitempty"`
    Stats      SessionStatsResponse `json:"stats"`
    Timeline   []SessionResponse    `json:"timeline,omitempty"`
}

// BulkDeleteRequest for bulk operations
type BulkDeleteRequest struct {
    IDs []string `json:"ids" binding:"required,min=1"`
}