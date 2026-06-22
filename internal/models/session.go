package models

import (
    "time"
)

type UserSession struct {
    ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
    UserID       string    `gorm:"type:uuid;not null;index" json:"user_id"`
    RefreshToken string    `gorm:"not null;index" json:"-"`
    UserAgent    string    `json:"user_agent"`
    ClientIP     string    `json:"client_ip"`
    ExpiresAt    time.Time `gorm:"not null;index" json:"expires_at"`
    CreatedAt    time.Time `json:"created_at"`
}

func (UserSession) TableName() string { return "user_sessions" }