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