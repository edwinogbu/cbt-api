package dto

import "time"

// Request DTOs
// type CreateSchoolRequest struct {
//     Name    string `json:"name" binding:"required,min=2,max=255"`
//     Code    string `json:"code" binding:"required,min=2,max=50"`
//     Address string `json:"address"`
//     Phone   string `json:"phone"`
//     Email   string `json:"email" binding:"omitempty,email"`
//     Logo    string `json:"logo"`
//     Website string `json:"website"`
// }

type CreateSchoolRequest struct {
    Name    string `json:"name" binding:"required,min=2,max=255"`
    Code    string `json:"code" binding:"omitempty,min=2,max=50"`  // changed: required → omitempty
    Address string `json:"address"`
    Phone   string `json:"phone"`
    Email   string `json:"email" binding:"omitempty,email"`
    Logo    string `json:"logo"`
    Website string `json:"website"`
}

type UpdateSchoolRequest struct {
    Name    string `json:"name" binding:"omitempty,min=2,max=255"`
    Address string `json:"address"`
    Phone   string `json:"phone"`
    Email   string `json:"email" binding:"omitempty,email"`
    Logo    string `json:"logo"`
    Website string `json:"website"`
    IsActive *bool `json:"is_active"`
}

// Response DTOs
type SchoolResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Code      string    `json:"code"`
    Address   string    `json:"address"`
    Phone     string    `json:"phone"`
    Email     string    `json:"email"`
    Logo      string    `json:"logo"`
    Website   string    `json:"website"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}