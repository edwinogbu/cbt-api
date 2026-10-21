package models

import (
    "time"
    "gorm.io/gorm"
)

// ============================================
// SCHOOL STATUS - As per your review
// ============================================

type SchoolStatus string

const (
    SchoolStatusPending          SchoolStatus = "PENDING"
    SchoolStatusEmailVerified    SchoolStatus = "EMAIL_VERIFIED"
    SchoolStatusPaymentPending   SchoolStatus = "PAYMENT_PENDING"
    SchoolStatusActive           SchoolStatus = "ACTIVE"
    SchoolStatusSuspended        SchoolStatus = "SUSPENDED"
    SchoolStatusExpired          SchoolStatus = "EXPIRED"
    SchoolStatusDeactivated      SchoolStatus = "DEACTIVATED"
)

type School struct {
    ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Name      string         `gorm:"uniqueIndex;not null" json:"name"`
    Code      string         `gorm:"uniqueIndex" json:"code"`
    Address   string         `json:"address"`
    Phone     string         `json:"phone"`
    Email     string         `json:"email"`
    Logo      string         `json:"logo"`
    Website   string         `json:"website"`
    Status    SchoolStatus   `gorm:"type:varchar(30);default:'PENDING'" json:"status"`
    IsActive  bool           `gorm:"default:true" json:"is_active"`

    // Platform-owner-controlled tenant branding (Master Implementation
    // Plan Phase 5). ApplicationName is distinct from Name: Name is the
    // school's own registered name (what they submitted at
    // registration), ApplicationName is what the Platform Owner
    // approves for display throughout the product (dashboard headings,
    // login screen, document titles, PWA/Electron presentation) -
    // school-editable in neither case; only a platform_owner-gated
    // endpoint may write these four fields. A school owner sees them
    // read-only. Defaults to empty so the frontend can fall back to
    // Name (or a neutral platform default) rather than ever hardcoding
    // one tenant's identity as the application's permanent branding.
    ApplicationName string `gorm:"column:application_name" json:"application_name"`
    PrimaryColor    string `gorm:"column:primary_color" json:"primary_color"`
    SecondaryColor  string `gorm:"column:secondary_color" json:"secondary_color"`

    // Set when a platform_owner provisions/activates this tenant (see
    // internal/platform). Nil until the first provisioning action.
    ProvisionedAt *time.Time `gorm:"column:provisioned_at" json:"provisioned_at,omitempty"`
    ProvisionedBy *string    `gorm:"column:provisioned_by;type:uuid" json:"provisioned_by,omitempty"`

    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
     // Relationships
    Subscriptions []Subscription `gorm:"foreignKey:SchoolID" json:"subscriptions,omitempty"`
    CurrentSubscription *Subscription `gorm:"-" json:"current_subscription,omitempty"`

}

func (School) TableName() string { return "schools" }


// package models

// import (
//     "time"
    
//     // "github.com/google/uuid"
//     "gorm.io/gorm"
// )

// type School struct {
//     ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     // ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     Name      string         `gorm:"uniqueIndex;not null" json:"name"`
//     Code      string         `gorm:"uniqueIndex" json:"code"`
//     Address   string         `json:"address"`
//     Phone     string         `json:"phone"`
//     Email     string         `json:"email"`
//     Logo      string         `json:"logo"`
//     Website   string         `json:"website"`
//     IsActive  bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt time.Time      `json:"created_at"`
//     UpdatedAt time.Time      `json:"updated_at"`
//     DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// 	   // Relationships
//     Subscriptions []Subscription `gorm:"foreignKey:SchoolID" json:"subscriptions,omitempty"`
//     CurrentSubscription *Subscription `gorm:"-" json:"current_subscription,omitempty"`
// }

// func (School) TableName() string { return "schools" }