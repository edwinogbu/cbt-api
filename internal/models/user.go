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

    // RoleHOD/RoleExamOfficer/RoleVPAcademics are Academic Broad Sheet
    // approval-chain roles (teacher submits -> HOD reviews -> exam
    // officer checks completeness -> VP Academics/admin approves),
    // granted the same way as any additional role - via UserRoleGrant/
    // AuthService.GrantRole, never a user's primary Role at
    // registration (RegisterRequest.Role's oneof allowlist is
    // unchanged). A user holding one of these almost always also holds
    // RoleTeacher as their primary role (see UserRoleGrant's own doc
    // comment, which already anticipated "an HOD also granted
    // teacher") - these are additive school-scoped responsibilities,
    // not a replacement for being a teacher.
    RoleHOD          UserRole = "hod"
    RoleExamOfficer  UserRole = "examination_officer"
    RoleVPAcademics  UserRole = "vp_academics"

    // RolePlatformOwner is a distinct tier above every school-scoped role,
    // including super_admin (which today is - and remains - a school-level
    // elevated admin, not a platform-wide authority). A platform owner is
    // not tied to any school (User.SchoolID is nil for this role) and is
    // the only role authorized for cross-tenant operations: the tenant
    // registry, platform-controlled branding, and the LAN Setup Dashboard.
    //
    // This role can ONLY become a user's primary Role via the server-side
    // bootstrap in pkg/database/postgres.go's seedPlatformOwner() (gated
    // on PLATFORM_OWNER_EMAIL/PLATFORM_OWNER_PASSWORD env vars, never a
    // request parameter). It is explicitly rejected by AuthService.GrantRole
    // so no admin/super_admin can grant it to themselves or anyone else via
    // POST /auth/users/:id/roles, and it is not in RegisterRequest.Role's
    // allowlist (oneof=student teacher admin parent) so public self-
    // registration can never set it either. Do not add it to either path.
    RolePlatformOwner UserRole = "platform_owner"

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

// UserRoleGrant records an ADDITIONAL role granted to a user, on top of
// their primary User.Role (e.g. a teacher also granted the principal
// role, or an HOD also granted teacher). A user's full set of available
// roles is always User.Role plus every row here for that user_id - see
// AuthService.GetAvailableRoles. Switching the active role (POST
// /auth/switch-role) re-issues a JWT with that role as the "role" claim;
// it never changes User.Role itself, so every existing permission check
// that reads the JWT's role claim (middleware.RoleMiddleware, and every
// `switch user.Role` in the service layer) keeps working unchanged - a
// switched-role session is just a normal token whose role claim happens
// to be a granted role instead of the primary one.
type UserRoleGrant struct {
    ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    UserID    string    `gorm:"type:uuid;index;not null;uniqueIndex:idx_user_role_grant" json:"user_id"`
    Role      UserRole  `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_role_grant" json:"role"`
    GrantedBy *string   `gorm:"type:uuid" json:"granted_by,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

func (UserRoleGrant) TableName() string { return "user_role_grants" }

