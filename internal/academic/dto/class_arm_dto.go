package dto

import "time"

// Request DTOs
type CreateClassArmRequest struct {
    SchoolID   string `json:"school_id" binding:"required"`
    Name       string `json:"name" binding:"required"`
    ArmCode    string `json:"arm_code"`
    Capacity   int    `json:"capacity"`
    RoomNumber string `json:"room_number"`
    SortOrder  int    `json:"sort_order"`
}

type UpdateClassArmRequest struct {
    Name       string `json:"name"`
    ArmCode    string `json:"arm_code"`
    Capacity   *int   `json:"capacity"`
    RoomNumber string `json:"room_number"`
    SortOrder  *int   `json:"sort_order"`
    IsActive   *bool  `json:"is_active"`
}

// Response DTOs
type ClassArmResponse struct {
    ID         string    `json:"id"`
    SchoolID   string    `json:"school_id"`
    Name       string    `json:"name"`
    ArmCode    string    `json:"arm_code"`
    Capacity   int       `json:"capacity"`
    RoomNumber string    `json:"room_number"`
    SortOrder  int       `json:"sort_order"`
    IsActive   bool      `json:"is_active"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}