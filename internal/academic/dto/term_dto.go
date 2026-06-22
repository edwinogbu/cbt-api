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

// Response DTOs
type TermResponse struct {
    ID         string    `json:"id"`
    SessionID  string    `json:"session_id"`
    TermNumber int       `json:"term_number"`
    Name       string    `json:"name"`
    StartDate  time.Time `json:"start_date"`
    EndDate    time.Time `json:"end_date"`
    IsCurrent  bool      `json:"is_current"`
    IsActive   bool      `json:"is_active"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}