package models

import (
    "time"
    
    "gorm.io/gorm"
)

type UserRole string
type UserStatus string

const (
    RoleAdmin     UserRole = "admin"
    RoleTeacher   UserRole = "teacher"
    RoleStudent   UserRole = "student"
    RoleParent    UserRole = "parent"
    RoleSuperAdmin UserRole = "super_admin"
    
    StatusActive    UserStatus = "active"
    StatusInactive  UserStatus = "inactive"
    StatusSuspended UserStatus = "suspended"
    StatusPending   UserStatus = "pending"
)

type User struct {
    ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Username         string         `gorm:"type:varchar(100);uniqueIndex" json:"username"`
    Email            *string        `gorm:"uniqueIndex" json:"email"`
    Password         string         `gorm:"not null" json:"-"`
    FirstName        string         `json:"first_name"`
    LastName         string         `json:"last_name"`
    PhoneNumber      string         `json:"phone_number"`
    Role             UserRole       `gorm:"default:'student'" json:"role"`
    Status           UserStatus     `gorm:"default:'pending'" json:"status"`
    EmailVerified    bool           `gorm:"default:false" json:"email_verified"`
    IsActive         bool           `gorm:"default:true" json:"is_active"`
    TwoFactorSecret  string         `gorm:"type:text" json:"-"`
    TwoFactorEnabled bool           `gorm:"default:false" json:"two_factor_enabled"`
    LastLoginAt      *time.Time     `json:"last_login_at"`
   // NEW: School ID (TEXT to support both UUID and string codes)
    SchoolID         *string        `gorm:"type:text;index" json:"school_id,omitempty"`
    CreatedAt        time.Time      `json:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at"`
    DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
    
    // ⭐ This is the field - make sure it's correct
    PlainPasswordEncrypted string `gorm:"column:plain_password_encrypted;type:text" json:"-"`
}

func (User) TableName() string { return "users" }

