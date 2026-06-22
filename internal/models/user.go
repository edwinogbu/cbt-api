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
    
    StatusActive    UserStatus = "active"
    StatusInactive  UserStatus = "inactive"
    StatusSuspended UserStatus = "suspended"
    StatusPending   UserStatus = "pending"
)

type User struct {
    ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Username         string         `gorm:"type:varchar(100);uniqueIndex" json:"username"`          // NEW: used for student login
    Email            *string        `gorm:"uniqueIndex" json:"email"`                               // optional for students – pointer to allow NULL
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
    CreatedAt        time.Time      `json:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at"`
    DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }







// // package models

// // import (
// //     "time"
    
// //     "gorm.io/gorm"
// // )

// // type UserRole string
// // type UserStatus string

// // const (
// //     RoleAdmin     UserRole = "admin"
// //     RoleTeacher   UserRole = "teacher"
// //     RoleStudent   UserRole = "student"
// //     RoleParent    UserRole = "parent"
    
// //     UserStatusActive    UserStatus = "active"
// //     UserStatusInactive  UserStatus = "inactive"
// //     UserStatusSuspended UserStatus = "suspended"
// //     UserStatusPending   UserStatus = "pending"
// // )

// // type User struct {
// //     ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     Email            string         `gorm:"uniqueIndex;not null" json:"email"`
// //     Password         string         `gorm:"not null" json:"-"`
// //     FirstName        string         `json:"first_name"`
// //     LastName         string         `json:"last_name"`
// //     PhoneNumber      string         `json:"phone_number"`
// //     Role             UserRole       `gorm:"default:'student'" json:"role"`
// //     Status           UserStatus     `gorm:"default:'pending'" json:"status"`
// //     EmailVerified    bool           `gorm:"default:false" json:"email_verified"`
// //     IsActive         bool           `gorm:"default:true" json:"is_active"`
// //     TwoFactorSecret  string         `gorm:"type:text" json:"-"`
// //     TwoFactorEnabled bool           `gorm:"default:false" json:"two_factor_enabled"`
// //     LastLoginAt      *time.Time     `json:"last_login_at"`
// //     CreatedAt        time.Time      `json:"created_at"`
// //     UpdatedAt        time.Time      `json:"updated_at"`
// //     DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // func (User) TableName() string { return "users" }


// package models

// import (
//     "time"
    
//     // "github.com/google/uuid"
//     "gorm.io/gorm"
// )

// type UserRole string
// type UserStatus string

// const (
//     RoleAdmin     UserRole = "admin"
//     RoleTeacher   UserRole = "teacher"
//     RoleStudent   UserRole = "student"
//     RoleParent    UserRole = "parent"
    
//     StatusActive    UserStatus = "active"
//     StatusInactive  UserStatus = "inactive"
//     StatusSuspended UserStatus = "suspended"
//     StatusPending   UserStatus = "pending"
// )

// type User struct {
//     // ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     Email         string         `gorm:"uniqueIndex;not null" json:"email"`
//     Password      string         `gorm:"not null" json:"-"`
//     FirstName     string         `json:"first_name"`
//     LastName      string         `json:"last_name"`
//     PhoneNumber   string         `json:"phone_number"`
//     Role          UserRole       `gorm:"default:'student'" json:"role"`
//     Status        UserStatus     `gorm:"default:'pending'" json:"status"`
//     EmailVerified bool           `gorm:"default:false" json:"email_verified"`
//     IsActive      bool           `gorm:"default:true" json:"is_active"`
// 	TwoFactorSecret  string `gorm:"type:text" json:"-"`
//     TwoFactorEnabled bool   `gorm:"default:false" json:"two_factor_enabled"`
//     LastLoginAt   *time.Time     `json:"last_login_at"`
//     CreatedAt     time.Time      `json:"created_at"`
//     UpdatedAt     time.Time      `json:"updated_at"`
//     DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
// }

// func (User) TableName() string { return "users" }