package dto

import "time"

// TenantListQuery - query params for GET /platform/tenants
type TenantListQuery struct {
	Status string `form:"status"`
	Search string `form:"search"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// TenantSummary - one row in the Platform Owner's tenant registry list
type TenantSummary struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Code               string     `json:"code"`
	ApplicationName    string     `json:"application_name"`
	Email              string     `json:"email"`
	Status             string     `json:"status"`
	IsActive           bool       `json:"is_active"`
	SubscriptionStatus string     `json:"subscription_status,omitempty"`
	SubscriptionTier   string     `json:"subscription_tier,omitempty"`
	ProvisionedAt      *time.Time `json:"provisioned_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// TenantListResponse - paginated tenant registry
type TenantListResponse struct {
	Data []TenantSummary `json:"data"`
	Meta struct {
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Total int64 `json:"total"`
	} `json:"meta"`
}

// TenantDetail - GET /platform/tenants/:id
type TenantDetail struct {
	TenantSummary
	Address        string `json:"address"`
	Phone          string `json:"phone"`
	Website        string `json:"website"`
	Logo           string `json:"logo"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
}

// UpdateBrandingRequest - PUT /platform/tenants/:id/branding (platform_owner only)
type UpdateBrandingRequest struct {
	ApplicationName string `json:"application_name" binding:"required,max=120"`
	Logo            string `json:"logo" binding:"omitempty,max=2048"`
	PrimaryColor    string `json:"primary_color" binding:"omitempty,hexcolor"`
	SecondaryColor  string `json:"secondary_color" binding:"omitempty,hexcolor"`
}

// ProvisionTenantRequest - PUT /platform/tenants/:id/provision
type ProvisionTenantRequest struct {
	ApplicationName string `json:"application_name" binding:"omitempty,max=120"`
}

// SuspendTenantRequest - PUT /platform/tenants/:id/suspend
type SuspendTenantRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=500"`
}

// AuditLogQuery - GET /platform/audit-logs
type AuditLogQuery struct {
	Action string `form:"action"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// AuditLogEntry
type AuditLogEntry struct {
	ID          string    `json:"id"`
	ActorUserID string    `json:"actor_user_id"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Details     string    `json:"details,omitempty"`
	Outcome     string    `json:"outcome"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLogListResponse
type AuditLogListResponse struct {
	Data []AuditLogEntry `json:"data"`
	Meta struct {
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Total int64 `json:"total"`
	} `json:"meta"`
}

// LANConfigResponse - GET /platform/lan-config/:schoolId
type LANConfigResponse struct {
	ID             string     `json:"id"`
	SchoolID       string     `json:"school_id"`
	ServerAddress  string     `json:"server_address"`
	Port           int        `json:"port"`
	IsActive       bool       `json:"is_active"`
	LastTestedAt   *time.Time `json:"last_tested_at,omitempty"`
	LastTestStatus string     `json:"last_test_status"`
	LastTestError  string     `json:"last_test_error,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// UpdateLANConfigRequest - PUT /platform/lan-config/:schoolId
type UpdateLANConfigRequest struct {
	ServerAddress string `json:"server_address" binding:"required,max=255"`
	Port          int    `json:"port" binding:"required,min=1,max=65535"`
}

// LANConfigHistoryEntry
type LANConfigHistoryEntry struct {
	ID            string    `json:"id"`
	ServerAddress string    `json:"server_address"`
	Port          int       `json:"port"`
	IsActive      bool      `json:"is_active"`
	ChangeType    string    `json:"change_type"`
	ChangedBy     string    `json:"changed_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// LANConnectivityTestResponse - POST /platform/lan-config/:schoolId/test
type LANConnectivityTestResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ResponseTime int64  `json:"response_time_ms,omitempty"`
}
