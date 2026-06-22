package models

import (
    "time"
)

type OTP struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
    Code      string    `gorm:"not null" json:"-"`
    Type      string    `gorm:"not null;index" json:"type"` // email_verification, password_reset
    Used      bool      `gorm:"default:false" json:"-"`
    ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}

func (OTP) TableName() string { return "otps" }