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



// ListClassArmsRequest for filtering and pagination (all fields optional)
type ListClassArmsRequest struct {
    Page        int    `form:"page,omitempty" binding:"omitempty,min=1"`
    Limit       int    `form:"limit,omitempty" binding:"omitempty,min=1,max=100"`
    Search      string `form:"search,omitempty"`
    SchoolID    string `form:"school_id,omitempty"`
    MinCapacity int    `form:"min_capacity,omitempty"`
    MaxCapacity int    `form:"max_capacity,omitempty"`
    SortBy      string `form:"sort_by,omitempty" binding:"omitempty,oneof=name arm_code capacity sort_order created_at"`
    SortOrder   string `form:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
    IsActive    *bool  `form:"is_active,omitempty"`
}

// ClassArmStatsResponse for statistics
type ClassArmStatsResponse struct {
    Total       int64   `json:"total"`
    Active      int64   `json:"active"`
    Inactive    int64   `json:"inactive"`
    TotalCapacity int64 `json:"total_capacity"`
    AvgCapacity   float64 `json:"avg_capacity"`
}

// ClassArmListResponse for paginated list
type ClassArmListResponse struct {
    Items      []ClassArmResponse     `json:"items"`
    Pagination PaginationMeta         `json:"pagination"`
    Stats      *ClassArmStatsResponse `json:"stats,omitempty"`
}

// BulkDeleteRequest for bulk operations
type BulkDeleteClassArmRequest struct {
    IDs []string `json:"ids" binding:"required,min=1"`
}
