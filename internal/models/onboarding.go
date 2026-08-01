package models

import (
    "time"
    "gorm.io/gorm"
)

// ============================================
// ONBOARDING SESSION - As per your review
// ============================================

type OnboardingStatus string

const (
    OnboardingStatusPlanSelected    OnboardingStatus = "PLAN_SELECTED"
    OnboardingStatusAccountCreated  OnboardingStatus = "ACCOUNT_CREATED"
    OnboardingStatusEmailVerified   OnboardingStatus = "EMAIL_VERIFIED"
    OnboardingStatusSchoolCreated   OnboardingStatus = "SCHOOL_CREATED"
    OnboardingStatusPaymentPending  OnboardingStatus = "PAYMENT_PENDING"
    OnboardingStatusComplete        OnboardingStatus = "COMPLETE"
    OnboardingStatusExpired         OnboardingStatus = "EXPIRED"
)

type OnboardingSession struct {
    ID             string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SessionToken   string           `gorm:"uniqueIndex;not null" json:"session_token"`
    Plan           string           `json:"plan"`
    Status         OnboardingStatus `gorm:"type:varchar(30);default:'PLAN_SELECTED'" json:"status"`
    UserID         *string          `gorm:"type:uuid" json:"user_id,omitempty"`
    SchoolID       *string          `gorm:"type:uuid" json:"school_id,omitempty"`
    SubscriptionID *string          `gorm:"type:uuid" json:"subscription_id,omitempty"`
    ExpiresAt      time.Time        `gorm:"not null;index" json:"expires_at"`
    CreatedAt      time.Time        `json:"created_at"`
    UpdatedAt      time.Time        `json:"updated_at"`
    DeletedAt      gorm.DeletedAt   `gorm:"index" json:"-"`
}

func (OnboardingSession) TableName() string { return "onboarding_sessions" }

// // internal/models/onboarding.go
// package models

// import (
//     "time"
// )

// type OnboardingSession struct {
//     ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     Plan           string    `gorm:"type:varchar(30);not null" json:"plan"`
//     Email          string    `gorm:"type:varchar(255);not null" json:"email"`
//     Status         string    `gorm:"type:varchar(30);default:'plan_selected'" json:"status"`
//     Step           int       `gorm:"default:1" json:"step"`
//     SchoolID       string    `gorm:"type:uuid" json:"school_id,omitempty"`
//     UserID         string    `gorm:"type:uuid" json:"user_id,omitempty"`
//     SubscriptionID string    `gorm:"type:uuid" json:"subscription_id,omitempty"`
//     ExpiresAt      time.Time `json:"expires_at"`
//     CreatedAt      time.Time `json:"created_at"`
//     UpdatedAt      time.Time `json:"updated_at"`
// }

// func (OnboardingSession) TableName() string { return "onboarding_sessions" }