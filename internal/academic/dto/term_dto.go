package dto

import "time"

// Request DTOs
type CreateTermRequest struct {
    SessionID string    `json:"session_id" binding:"required"`
    TermNumber int      `json:"term_number" binding:"required,min=1,max=3"`
    Name      string    `json:"name"`
    StartDate time.Time `json:"start_date" binding:"required"`
    EndDate   time.Time `json:"end_date" binding:"required"`
    IsCurrent bool      `json:"is_current"`
}

type UpdateTermRequest struct {
    Name      string     `json:"name"`
    StartDate *time.Time `json:"start_date"`
    EndDate   *time.Time `json:"end_date"`
    IsCurrent *bool      `json:"is_current"`
    IsActive  *bool      `json:"is_active"`
}


type TermResponse struct {
    ID          string    `json:"id"`
    SessionID   string    `json:"session_id"`
    SessionName string    `json:"session_name,omitempty"`
    TermNumber  int       `json:"term_number"`
    Name        string    `json:"name"`
    StartDate   time.Time `json:"start_date"`
    EndDate     time.Time `json:"end_date"`
    IsCurrent   bool      `json:"is_current"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type TermListResponse struct {
    Terms      []TermResponse `json:"terms"`
    Total      int64          `json:"total"`
    Page       int            `json:"page"`
    Limit      int            `json:"limit"`
    TotalPages int            `json:"total_pages"`
}

// ============================================
// NEW DTO TYPES FOR MISSING ENDPOINTS
// ============================================

// TermBulkDeleteRequest - for bulk delete
type TermBulkDeleteRequest struct {
    IDs []string `json:"ids" binding:"required"`
}

// TermStatsResponse - statistics response
type TermStatsResponse struct {
    Total      int64            `json:"total"`
    Active     int64            `json:"active"`
    Inactive   int64            `json:"inactive"`
    Current    int64            `json:"current"`
    BySession  map[string]int64 `json:"by_session"`
    TotalTerms int64            `json:"total_terms"`
}

// ListTermsRequest - ALL PARAMETERS OPTIONAL
type ListTermsRequest struct {
    Page       int    `form:"page" json:"page"`
    Limit      int    `form:"limit" json:"limit"`
    Search     string `form:"search" json:"search"`
    SessionID  string `form:"session_id" json:"session_id"`
    SchoolID   string `form:"school_id" json:"school_id"`
    IsActive   *bool  `form:"is_active" json:"is_active"`
    IsCurrent  *bool  `form:"is_current" json:"is_current"`
    SortBy     string `form:"sort_by" json:"sort_by"`
    SortOrder  string `form:"sort_order" json:"sort_order"`
}