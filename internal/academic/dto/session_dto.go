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