package dto

import "time"

// Request DTOs
type CreateClassLevelRequest struct {
    SchoolID    string `json:"school_id" binding:"required"`
    Name        string `json:"name" binding:"required"`
    LevelNumber int    `json:"level_number" binding:"required"`
    Category    string `json:"category" binding:"required,oneof=JSS SSS PRIMARY"`
    PromotionTo *string `json:"promotion_to"`
    SortOrder   int    `json:"sort_order"`
}

type UpdateClassLevelRequest struct {
    Name        string  `json:"name"`
    LevelNumber *int    `json:"level_number"`
    Category    string  `json:"category"`
    PromotionTo *string `json:"promotion_to"`
    SortOrder   *int    `json:"sort_order"`
    IsActive    *bool   `json:"is_active"`
}

// NEW: List request with ALL OPTIONAL parameters
type ListClassLevelsRequest struct {
    Page     int    `form:"page" json:"page"`
    Limit    int    `form:"limit" json:"limit"`
    Search   string `form:"search" json:"search"`
    SchoolID string `form:"school_id" json:"school_id"`
    Category string `form:"category" json:"category"`
    SortBy   string `form:"sort_by" json:"sort_by"`
    SortDir  string `form:"sort_order" json:"sort_order"`
    IsActive *bool  `form:"is_active" json:"is_active"`
}

// NEW: Bulk delete request - renamed to avoid conflict
type ClassLevelBulkDeleteRequest struct {
    IDs []string `json:"ids" binding:"required"`
}

// NEW: Search response
type SearchClassLevelsResponse struct {
    Items      []ClassLevelResponse `json:"items"`
    Total      int64                `json:"total"`
    Page       int                  `json:"page"`
    Limit      int                  `json:"limit"`
    TotalPages int                  `json:"total_pages"`
}

// NEW: Statistics response
type ClassLevelStatsResponse struct {
    Total          int64            `json:"total"`
    Active         int64            `json:"active"`
    Inactive       int64            `json:"inactive"`
    ByCategory     map[string]int64 `json:"by_category"`
    TotalLevels    int64            `json:"total_levels"`
    AvgLevelNumber float64          `json:"avg_level_number"`
}

// Response DTOs
type ClassLevelResponse struct {
    ID          string    `json:"id"`
    SchoolID    string    `json:"school_id"`
    Name        string    `json:"name"`
    LevelNumber int       `json:"level_number"`
    Category    string    `json:"category"`
    PromotionTo *string   `json:"promotion_to"`
    SortOrder   int       `json:"sort_order"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}


