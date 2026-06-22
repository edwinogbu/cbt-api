package models

import (
    "time"
    
    // "github.com/google/uuid"
    "gorm.io/gorm"
)

type School struct {
    ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    // ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Name      string         `gorm:"uniqueIndex;not null" json:"name"`
    Code      string         `gorm:"uniqueIndex" json:"code"`
    Address   string         `json:"address"`
    Phone     string         `json:"phone"`
    Email     string         `json:"email"`
    Logo      string         `json:"logo"`
    Website   string         `json:"website"`
    IsActive  bool           `gorm:"default:true" json:"is_active"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	   // Relationships
    Subscriptions []Subscription `gorm:"foreignKey:SchoolID" json:"subscriptions,omitempty"`
    CurrentSubscription *Subscription `gorm:"-" json:"current_subscription,omitempty"`
}

func (School) TableName() string { return "schools" }