package models

import (
	// "database/sql/driver"
	// "encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ============================================
// ENUM TYPES
// ============================================

type SubscriptionTier string
type SubscriptionStatus string
type PaymentStatus string
type PaymentIntentStatus string
type PaymentMethod string
type PaymentInterval string
type PaymentGateway string
type Currency string
type EmailType string
type InvoiceStatus string
type WebhookStatus string

// ============================================
// ENUM VALUES
// ============================================

const (
	// Subscription tiers
	TierBasic      SubscriptionTier = "basic"
	TierPremium    SubscriptionTier = "premium"
	TierEnterprise SubscriptionTier = "enterprise"

	// Subscription statuses (renamed to avoid conflict with user.go)
	SubStatusPending   SubscriptionStatus = "pending"
	SubStatusTrial     SubscriptionStatus = "trial"
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPastDue   SubscriptionStatus = "past_due"
	SubStatusExpired   SubscriptionStatus = "expired"
	SubStatusCancelled SubscriptionStatus = "cancelled"

	// Payment transaction statuses
	PaymentPending   PaymentStatus = "pending"
	PaymentPaid      PaymentStatus = "paid"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
	PaymentCancelled PaymentStatus = "cancelled"

	// Payment intent statuses
	IntentPending        PaymentIntentStatus = "pending"
	IntentRequiresAction PaymentIntentStatus = "requires_action"
	IntentSucceeded      PaymentIntentStatus = "succeeded"
	IntentFailed         PaymentIntentStatus = "failed"
	IntentExpired        PaymentIntentStatus = "expired"
	IntentCancelled      PaymentIntentStatus = "cancelled"

	// Invoice statuses
	InvoicePending  InvoiceStatus = "pending"
	InvoicePaid     InvoiceStatus = "paid"
	InvoiceOverdue  InvoiceStatus = "overdue"
	InvoiceVoid     InvoiceStatus = "void"
	InvoiceRefunded InvoiceStatus = "refunded"

	// Payment methods
	MethodCard   PaymentMethod = "card"
	MethodBank   PaymentMethod = "bank_transfer"
	MethodWallet PaymentMethod = "wallet"
	MethodUSSD   PaymentMethod = "ussd"
	MethodQR     PaymentMethod = "qr"

	// Billing intervals
	IntervalMonthly   PaymentInterval = "monthly"
	IntervalQuarterly PaymentInterval = "quarterly"
	IntervalYearly    PaymentInterval = "yearly"

	// Gateways
	GatewayStripe      PaymentGateway = "stripe"
	GatewayPaystack    PaymentGateway = "paystack"
	GatewayFlutterwave PaymentGateway = "flutterwave"

	// Currencies
	CurrencyNGN Currency = "NGN"
	CurrencyUSD Currency = "USD"

	// Webhook statuses
	WebhookPending   WebhookStatus = "pending"
	WebhookProcessed WebhookStatus = "processed"
	WebhookFailed    WebhookStatus = "failed"

	// Email notifications
	EmailSubscriptionCreated   EmailType = "subscription_created"
	EmailSubscriptionRenewed   EmailType = "subscription_renewed"
	EmailSubscriptionExpiring  EmailType = "subscription_expiring"
	EmailSubscriptionExpired   EmailType = "subscription_expired"
	EmailSubscriptionCancelled EmailType = "subscription_cancelled"
	EmailPaymentSuccess        EmailType = "payment_success"
	EmailPaymentFailed         EmailType = "payment_failed"
	EmailInvoiceGenerated      EmailType = "invoice_generated"
)

// ============================================
// TIER LIMITS & PRICING
// ============================================

type TierLimit struct {
	MaxStudents  int `json:"max_students"`
	MaxTeachers  int `json:"max_teachers"`
	MaxExams     int `json:"max_exams"`
	MaxQuestions int `json:"max_questions"`
	MaxStorageMB int `json:"max_storage_mb"`
}

var TierLimits = map[SubscriptionTier]TierLimit{
	TierBasic: {
		MaxStudents:  100,
		MaxTeachers:  10,
		MaxExams:     50,
		MaxQuestions: 500,
		MaxStorageMB: 1024,
	},
	TierPremium: {
		MaxStudents:  500,
		MaxTeachers:  50,
		MaxExams:     200,
		MaxQuestions: 2000,
		MaxStorageMB: 5120,
	},
	TierEnterprise: {
		MaxStudents:  5000,
		MaxTeachers:  500,
		MaxExams:     1000,
		MaxQuestions: 10000,
		MaxStorageMB: 51200,
	},
}

var Pricing = map[SubscriptionTier]map[PaymentInterval]decimal.Decimal{
	TierBasic: {
		IntervalMonthly:   decimal.NewFromFloat(29.99),
		IntervalQuarterly: decimal.NewFromFloat(79.99),
		IntervalYearly:    decimal.NewFromFloat(299.99),
	},
	TierPremium: {
		IntervalMonthly:   decimal.NewFromFloat(99.99),
		IntervalQuarterly: decimal.NewFromFloat(279.99),
		IntervalYearly:    decimal.NewFromFloat(999.99),
	},
	TierEnterprise: {
		IntervalMonthly:   decimal.NewFromFloat(299.99),
		IntervalQuarterly: decimal.NewFromFloat(849.99),
		IntervalYearly:    decimal.NewFromFloat(2999.99),
	},
}

// ============================================
// JSON MAP HELPER (Using type alias to avoid duplication)
// ============================================

// Note: JSONMap is defined in cbt.go. This is a type alias for convenience.
// The actual implementation is in cbt.go with Value() and Scan() methods.

// ============================================
// SUBSCRIPTION MODEL
// ============================================

type Subscription struct {
	ID     string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID string `gorm:"type:uuid;not null;index:idx_user_subscription" json:"user_id"`
	SchoolID string `gorm:"type:uuid;not null;index:idx_school_subscription" json:"school_id"`

	Tier   SubscriptionTier   `gorm:"type:varchar(30);not null" json:"tier"`
	Status SubscriptionStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`

	// Gateway Information
	Gateway               PaymentGateway `gorm:"type:varchar(30);not null" json:"gateway"`
	GatewayCustomerID     string         `gorm:"index" json:"gateway_customer_id,omitempty"`
	GatewaySubscriptionID string         `gorm:"index" json:"gateway_subscription_id,omitempty"`

	// Billing Details
	Amount          decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
	Currency        Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`
	PaymentInterval PaymentInterval `gorm:"type:varchar(20);default:'monthly'" json:"payment_interval"`

	// Dates
	StartDate       time.Time  `gorm:"not null" json:"start_date"`
	EndDate         time.Time  `gorm:"not null;index" json:"end_date"`
	TrialEndsAt     *time.Time `json:"trial_ends_at,omitempty"`
	LastPaymentDate *time.Time `json:"last_payment_date,omitempty"`
	NextPaymentDate *time.Time `json:"next_payment_date,omitempty"`

	// Renewal Settings
	AutoRenew       bool `gorm:"default:false" json:"auto_renew"`
	CancelAtEndDate bool `gorm:"default:false" json:"cancel_at_end_date"`

	// Resource Limits
	MaxStudents  int `gorm:"default:100" json:"max_students"`
	MaxTeachers  int `gorm:"default:10" json:"max_teachers"`
	MaxExams     int `gorm:"default:50" json:"max_exams"`
	MaxQuestions int `gorm:"default:500" json:"max_questions"`
	MaxStorageMB int `gorm:"default:1024" json:"max_storage_mb"`

	// Current Invoice Reference
	CurrentInvoiceID *string `gorm:"type:uuid" json:"current_invoice_id,omitempty"`

	// JSON Data Fields - Using JSONMap from cbt.go
	Features           JSONMap `gorm:"type:jsonb;default:'{}'" json:"features"`
	Metadata           JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	GatewayData        JSONMap `gorm:"type:jsonb;default:'{}'" json:"gateway_data"`
	EmailNotifications JSONMap `gorm:"type:jsonb;default:'{}'" json:"email_notifications"`

	// Audit Fields
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`

	// Relationships
	Invoices            []Invoice            `gorm:"foreignKey:SubscriptionID" json:"invoices,omitempty"`
	PaymentIntents      []PaymentIntent      `gorm:"foreignKey:SubscriptionID" json:"payment_intents,omitempty"`
	PaymentTransactions []PaymentTransaction `gorm:"foreignKey:SubscriptionID" json:"payment_transactions,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ============================================
// INVOICE MODEL
// ============================================

type Invoice struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

	InvoiceNumber string `gorm:"uniqueIndex;not null" json:"invoice_number"`

	// Financial Details
	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
	Tax      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"tax"`
	Discount decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"discount"`
	Total    decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"total"`

	Currency Currency `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

	Status InvoiceStatus `gorm:"type:varchar(20);default:'pending';index" json:"status"`

	// Dates
	DueDate time.Time  `gorm:"index" json:"due_date"`
	PaidAt  *time.Time `json:"paid_at,omitempty"`

	// Documents
	PDFURL string `json:"pdf_url,omitempty"`

	// JSON Data
	Items    JSONMap `gorm:"type:jsonb;default:'{}'" json:"items"`
	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relationships
	Subscription    Subscription     `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	PaymentIntents  []PaymentIntent  `gorm:"foreignKey:InvoiceID" json:"payment_intents,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ============================================
// PAYMENT INTENT MODEL
// ============================================

type PaymentIntent struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
	InvoiceID      string `gorm:"type:uuid;not null;index" json:"invoice_id"`
	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

	// Idempotency (Prevents duplicate charges)
	IdempotencyKey string `gorm:"uniqueIndex;not null" json:"idempotency_key"`

	// Gateway
	Gateway PaymentGateway `gorm:"type:varchar(30);not null;index" json:"gateway"`

	// Amount
	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
	Currency Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

	// Gateway References
	Reference            string `gorm:"uniqueIndex:idx_gateway_reference" json:"reference"`
	GatewayPaymentID     string `gorm:"index" json:"gateway_payment_id,omitempty"`
	GatewayTransactionID string `gorm:"index" json:"gateway_transaction_id,omitempty"`

	// Gateway Response Data
	ClientSecret     string `json:"client_secret,omitempty"`
	AuthorizationURL string `json:"authorization_url,omitempty"`
	AccessCode       string `json:"access_code,omitempty"`

	// Payment Details
	PaymentMethod PaymentMethod        `gorm:"type:varchar(30)" json:"payment_method,omitempty"`
	Status        PaymentIntentStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`

	// Retry Handling
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	LastRetryAt *time.Time `json:"last_retry_at,omitempty"`

	// Lifecycle
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`

	// Immutable Lock (Prevents modification after finalization)
	IsFinalized bool `gorm:"default:false" json:"is_finalized"`

	// JSON Data
	GatewayResponse JSONMap `gorm:"type:jsonb;default:'{}'" json:"gateway_response"`
	Metadata        JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relationships
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	Invoice      Invoice      `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ============================================
// PAYMENT TRANSACTION MODEL
// ============================================

type PaymentTransaction struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID  string `gorm:"type:uuid;not null;index" json:"subscription_id"`
	PaymentIntentID string `gorm:"type:uuid;not null;index" json:"payment_intent_id"`
	InvoiceID       string `gorm:"type:uuid;not null;index" json:"invoice_id"`
	SchoolID        string `gorm:"type:uuid;not null;index" json:"school_id"`
	UserID          string `gorm:"type:uuid;not null;index" json:"user_id"`

	// Financial Details
	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
	Fee      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"fee"`
	Tax      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"tax"`
	Currency Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

	// Payment Details
	PaymentMethod PaymentMethod  `gorm:"type:varchar(30)" json:"payment_method"`
	PaymentStatus PaymentStatus  `gorm:"type:varchar(30);default:'pending';index" json:"payment_status"`
	Gateway       PaymentGateway `gorm:"type:varchar(30);index" json:"gateway"`

	// References
	Reference            string `gorm:"uniqueIndex" json:"reference"`
	TransactionID        string `gorm:"uniqueIndex" json:"transaction_id"`
	GatewayTransactionID string `gorm:"index" json:"gateway_transaction_id"`

	Description string `json:"description"`

	// Dates
	PaidAt *time.Time `json:"paid_at,omitempty"`

	// Immutable Financial Lock (Prevents modification after reconciliation)
	IsLocked bool `gorm:"default:false" json:"is_locked"`

	// JSON Data
	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relationships
	Subscription  Subscription  `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	Invoice       Invoice       `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	PaymentIntent PaymentIntent `gorm:"foreignKey:PaymentIntentID" json:"payment_intent,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================
// PAYMENT EVENT LOG MODEL (AUDIT TRAIL)
// ============================================

type PaymentEventLog struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	PaymentIntentID string `gorm:"type:uuid;index" json:"payment_intent_id"`
	TransactionID   string `gorm:"type:uuid;index" json:"transaction_id"`

	EventType    string `gorm:"index;not null" json:"event_type"`
	StatusBefore string `json:"status_before,omitempty"`
	StatusAfter  string `json:"status_after,omitempty"`

	Message string `json:"message,omitempty"`

	Payload JSONMap `gorm:"type:jsonb;default:'{}'" json:"payload"`

	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// WEBHOOK EVENT MODEL
// ============================================

type WebhookEvent struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	Gateway   PaymentGateway `gorm:"type:varchar(30);index" json:"gateway"`
	EventType string         `gorm:"index" json:"event_type"`

	Reference string `gorm:"index" json:"reference"`

	IdempotencyKey string `gorm:"uniqueIndex" json:"idempotency_key"`

	Status WebhookStatus `gorm:"type:varchar(20);default:'pending';index" json:"status"`

	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`

	Payload JSONMap `gorm:"type:jsonb;default:'{}'" json:"payload"`

	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// SUBSCRIPTION HISTORY MODEL
// ============================================

type SubscriptionHistory struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

	OldTier   SubscriptionTier   `json:"old_tier,omitempty"`
	NewTier   SubscriptionTier   `json:"new_tier,omitempty"`
	OldStatus SubscriptionStatus `json:"old_status,omitempty"`
	NewStatus SubscriptionStatus `json:"new_status,omitempty"`

	OldAmount decimal.Decimal `gorm:"type:decimal(20,2)" json:"old_amount,omitempty"`
	NewAmount decimal.Decimal `gorm:"type:decimal(20,2)" json:"new_amount,omitempty"`

	ChangeReason string `json:"change_reason"`
	ChangedBy    string `json:"changed_by"`

	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// EMAIL NOTIFICATION MODEL
// ============================================

type EmailNotification struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

	EmailType      EmailType `json:"email_type"`
	RecipientEmail string    `gorm:"index" json:"recipient_email"`
	Subject        string    `json:"subject"`

	Status       string `gorm:"index" json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`

	SentAt time.Time `json:"sent_at"`

	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// REMINDER SCHEDULE MODEL
// ============================================

type ReminderSchedule struct {
	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`

	ReminderType string `gorm:"index" json:"reminder_type"`
	DaysBefore   int    `json:"days_before"`

	Status string `gorm:"index" json:"status"`

	SentAt *time.Time `json:"sent_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================
// TABLE NAMES
// ============================================

func (Subscription) TableName() string        { return "subscriptions" }
func (Invoice) TableName() string             { return "invoices" }
func (PaymentIntent) TableName() string       { return "payment_intents" }
func (PaymentTransaction) TableName() string  { return "payment_transactions" }
func (PaymentEventLog) TableName() string     { return "payment_event_logs" }
func (WebhookEvent) TableName() string        { return "webhook_events" }
func (SubscriptionHistory) TableName() string { return "subscription_histories" }
func (EmailNotification) TableName() string   { return "email_notifications" }
func (ReminderSchedule) TableName() string    { return "reminder_schedules" }

// package models

// import (
// 	"database/sql/driver"
// 	"encoding/json"
// 	"time"

// 	"github.com/shopspring/decimal"
// 	"gorm.io/gorm"
// )

// // ============================================
// // ENUM TYPES
// // ============================================

// type SubscriptionTier string
// type SubscriptionStatus string
// type PaymentStatus string
// type PaymentIntentStatus string
// type PaymentMethod string
// type PaymentInterval string
// type PaymentGateway string
// type Currency string
// type EmailType string
// type InvoiceStatus string
// type WebhookStatus string

// // ============================================
// // ENUM VALUES
// // ============================================

// const (
// 	// Subscription tiers
// 	TierBasic      SubscriptionTier = "basic"
// 	TierPremium    SubscriptionTier = "premium"
// 	TierEnterprise SubscriptionTier = "enterprise"

// 	// Subscription statuses
// 	StatusPending   SubscriptionStatus = "pending"
// 	StatusTrial     SubscriptionStatus = "trial"
// 	StatusActive    SubscriptionStatus = "active"
// 	StatusPastDue   SubscriptionStatus = "past_due"
// 	StatusExpired   SubscriptionStatus = "expired"
// 	StatusCancelled SubscriptionStatus = "cancelled"

// 	// Payment transaction statuses
// 	PaymentPending   PaymentStatus = "pending"
// 	PaymentPaid      PaymentStatus = "paid"
// 	PaymentFailed    PaymentStatus = "failed"
// 	PaymentRefunded  PaymentStatus = "refunded"
// 	PaymentCancelled PaymentStatus = "cancelled"

// 	// Payment intent statuses
// 	IntentPending        PaymentIntentStatus = "pending"
// 	IntentRequiresAction PaymentIntentStatus = "requires_action"
// 	IntentSucceeded      PaymentIntentStatus = "succeeded"
// 	IntentFailed         PaymentIntentStatus = "failed"
// 	IntentExpired        PaymentIntentStatus = "expired"
// 	IntentCancelled      PaymentIntentStatus = "cancelled"

// 	// Invoice statuses
// 	InvoicePending  InvoiceStatus = "pending"
// 	InvoicePaid     InvoiceStatus = "paid"
// 	InvoiceOverdue  InvoiceStatus = "overdue"
// 	InvoiceVoid     InvoiceStatus = "void"
// 	InvoiceRefunded InvoiceStatus = "refunded"

// 	// Payment methods
// 	MethodCard   PaymentMethod = "card"
// 	MethodBank   PaymentMethod = "bank_transfer"
// 	MethodWallet PaymentMethod = "wallet"
// 	MethodUSSD   PaymentMethod = "ussd"
// 	MethodQR     PaymentMethod = "qr"

// 	// Billing intervals
// 	IntervalMonthly   PaymentInterval = "monthly"
// 	IntervalQuarterly PaymentInterval = "quarterly"
// 	IntervalYearly    PaymentInterval = "yearly"

// 	// Gateways
// 	GatewayStripe      PaymentGateway = "stripe"
// 	GatewayPaystack    PaymentGateway = "paystack"
// 	GatewayFlutterwave PaymentGateway = "flutterwave"

// 	// Currencies
// 	CurrencyNGN Currency = "NGN"
// 	CurrencyUSD Currency = "USD"

// 	// Webhook statuses
// 	WebhookPending   WebhookStatus = "pending"
// 	WebhookProcessed WebhookStatus = "processed"
// 	WebhookFailed    WebhookStatus = "failed"

// 	// Email notifications
// 	EmailSubscriptionCreated   EmailType = "subscription_created"
// 	EmailSubscriptionRenewed   EmailType = "subscription_renewed"
// 	EmailSubscriptionExpiring  EmailType = "subscription_expiring"
// 	EmailSubscriptionExpired   EmailType = "subscription_expired"
// 	EmailSubscriptionCancelled EmailType = "subscription_cancelled"
// 	EmailPaymentSuccess        EmailType = "payment_success"
// 	EmailPaymentFailed         EmailType = "payment_failed"
// 	EmailInvoiceGenerated      EmailType = "invoice_generated"
// )

// // ============================================
// // TIER LIMITS & PRICING
// // ============================================

// type TierLimit struct {
// 	MaxStudents  int `json:"max_students"`
// 	MaxTeachers  int `json:"max_teachers"`
// 	MaxExams     int `json:"max_exams"`
// 	MaxQuestions int `json:"max_questions"`
// 	MaxStorageMB int `json:"max_storage_mb"`
// }

// var TierLimits = map[SubscriptionTier]TierLimit{
// 	TierBasic: {
// 		MaxStudents:  100,
// 		MaxTeachers:  10,
// 		MaxExams:     50,
// 		MaxQuestions: 500,
// 		MaxStorageMB: 1024,
// 	},
// 	TierPremium: {
// 		MaxStudents:  500,
// 		MaxTeachers:  50,
// 		MaxExams:     200,
// 		MaxQuestions: 2000,
// 		MaxStorageMB: 5120,
// 	},
// 	TierEnterprise: {
// 		MaxStudents:  5000,
// 		MaxTeachers:  500,
// 		MaxExams:     1000,
// 		MaxQuestions: 10000,
// 		MaxStorageMB: 51200,
// 	},
// }

// var Pricing = map[SubscriptionTier]map[PaymentInterval]decimal.Decimal{
// 	TierBasic: {
// 		IntervalMonthly:   decimal.NewFromFloat(29.99),
// 		IntervalQuarterly: decimal.NewFromFloat(79.99),
// 		IntervalYearly:    decimal.NewFromFloat(299.99),
// 	},
// 	TierPremium: {
// 		IntervalMonthly:   decimal.NewFromFloat(99.99),
// 		IntervalQuarterly: decimal.NewFromFloat(279.99),
// 		IntervalYearly:    decimal.NewFromFloat(999.99),
// 	},
// 	TierEnterprise: {
// 		IntervalMonthly:   decimal.NewFromFloat(299.99),
// 		IntervalQuarterly: decimal.NewFromFloat(849.99),
// 		IntervalYearly:    decimal.NewFromFloat(2999.99),
// 	},
// }

// // ============================================
// // JSON MAP HELPER
// // ============================================

// // type JSONMap map[string]interface{}

// // func (j JSONMap) Value() (driver.Value, error) {
// // 	if j == nil {
// // 		return nil, nil
// // 	}
// // 	return json.Marshal(j)
// // }

// func (j *JSONMap) Scan(value interface{}) error {
// 	if value == nil {
// 		*j = make(JSONMap)
// 		return nil
// 	}
// 	bytes, ok := value.([]byte)
// 	if !ok {
// 		return nil
// 	}
// 	return json.Unmarshal(bytes, j)
// }

// // ============================================
// // SUBSCRIPTION MODEL
// // ============================================

// type Subscription struct {
// 	ID     string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	UserID string `gorm:"type:uuid;not null;index:idx_user_subscription" json:"user_id"`
// 	SchoolID string `gorm:"type:uuid;not null;index:idx_school_subscription" json:"school_id"`

// 	Tier   SubscriptionTier   `gorm:"type:varchar(30);not null" json:"tier"`
// 	Status SubscriptionStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`

// 	// Gateway Information
// 	Gateway               PaymentGateway `gorm:"type:varchar(30);not null" json:"gateway"`
// 	GatewayCustomerID     string         `gorm:"index" json:"gateway_customer_id,omitempty"`
// 	GatewaySubscriptionID string         `gorm:"index" json:"gateway_subscription_id,omitempty"`

// 	// Billing Details
// 	Amount          decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
// 	Currency        Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`
// 	PaymentInterval PaymentInterval `gorm:"type:varchar(20);default:'monthly'" json:"payment_interval"`

// 	// Dates
// 	StartDate       time.Time  `gorm:"not null" json:"start_date"`
// 	EndDate         time.Time  `gorm:"not null;index" json:"end_date"`
// 	TrialEndsAt     *time.Time `json:"trial_ends_at,omitempty"`
// 	LastPaymentDate *time.Time `json:"last_payment_date,omitempty"`
// 	NextPaymentDate *time.Time `json:"next_payment_date,omitempty"`

// 	// Renewal Settings
// 	AutoRenew       bool `gorm:"default:false" json:"auto_renew"`
// 	CancelAtEndDate bool `gorm:"default:false" json:"cancel_at_end_date"`

// 	// Resource Limits
// 	MaxStudents  int `gorm:"default:100" json:"max_students"`
// 	MaxTeachers  int `gorm:"default:10" json:"max_teachers"`
// 	MaxExams     int `gorm:"default:50" json:"max_exams"`
// 	MaxQuestions int `gorm:"default:500" json:"max_questions"`
// 	MaxStorageMB int `gorm:"default:1024" json:"max_storage_mb"`

// 	// Current Invoice Reference
// 	CurrentInvoiceID *string `gorm:"type:uuid" json:"current_invoice_id,omitempty"`

// 	// JSON Data Fields
// 	Features           JSONMap `gorm:"type:jsonb;default:'{}'" json:"features"`
// 	Metadata           JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`
// 	GatewayData        JSONMap `gorm:"type:jsonb;default:'{}'" json:"gateway_data"`
// 	EmailNotifications JSONMap `gorm:"type:jsonb;default:'{}'" json:"email_notifications"`

// 	// Audit Fields
// 	CreatedBy string `json:"created_by,omitempty"`
// 	UpdatedBy string `json:"updated_by,omitempty"`

// 	// Relationships
// 	Invoices            []Invoice            `gorm:"foreignKey:SubscriptionID" json:"invoices,omitempty"`
// 	PaymentIntents      []PaymentIntent      `gorm:"foreignKey:SubscriptionID" json:"payment_intents,omitempty"`
// 	PaymentTransactions []PaymentTransaction `gorm:"foreignKey:SubscriptionID" json:"payment_transactions,omitempty"`

// 	CreatedAt time.Time      `json:"created_at"`
// 	UpdatedAt time.Time      `json:"updated_at"`
// 	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // ============================================
// // INVOICE MODEL
// // ============================================

// type Invoice struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
// 	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
// 	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

// 	InvoiceNumber string `gorm:"uniqueIndex;not null" json:"invoice_number"`

// 	// Financial Details
// 	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
// 	Tax      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"tax"`
// 	Discount decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"discount"`
// 	Total    decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"total"`

// 	Currency Currency `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

// 	Status InvoiceStatus `gorm:"type:varchar(20);default:'pending';index" json:"status"`

// 	// Dates
// 	DueDate time.Time  `gorm:"index" json:"due_date"`
// 	PaidAt  *time.Time `json:"paid_at,omitempty"`

// 	// Documents
// 	PDFURL string `json:"pdf_url,omitempty"`

// 	// JSON Data
// 	Items    JSONMap `gorm:"type:jsonb;default:'{}'" json:"items"`
// 	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

// 	// Relationships
// 	Subscription    Subscription     `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
// 	PaymentIntents  []PaymentIntent  `gorm:"foreignKey:InvoiceID" json:"payment_intents,omitempty"`

// 	CreatedAt time.Time      `json:"created_at"`
// 	UpdatedAt time.Time      `json:"updated_at"`
// 	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // ============================================
// // PAYMENT INTENT MODEL
// // ============================================

// type PaymentIntent struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
// 	InvoiceID      string `gorm:"type:uuid;not null;index" json:"invoice_id"`
// 	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
// 	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

// 	// Idempotency (Prevents duplicate charges)
// 	IdempotencyKey string `gorm:"uniqueIndex;not null" json:"idempotency_key"`

// 	// Gateway
// 	Gateway PaymentGateway `gorm:"type:varchar(30);not null;index" json:"gateway"`

// 	// Amount
// 	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
// 	Currency Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

// 	// Gateway References
// 	Reference            string `gorm:"uniqueIndex:idx_gateway_reference" json:"reference"`
// 	GatewayPaymentID     string `gorm:"index" json:"gateway_payment_id,omitempty"`
// 	GatewayTransactionID string `gorm:"index" json:"gateway_transaction_id,omitempty"`

// 	// Gateway Response Data
// 	ClientSecret     string `json:"client_secret,omitempty"`
// 	AuthorizationURL string `json:"authorization_url,omitempty"`
// 	AccessCode       string `json:"access_code,omitempty"`

// 	// Payment Details
// 	PaymentMethod PaymentMethod        `gorm:"type:varchar(30)" json:"payment_method,omitempty"`
// 	Status        PaymentIntentStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`

// 	// Retry Handling
// 	RetryCount  int        `gorm:"default:0" json:"retry_count"`
// 	LastRetryAt *time.Time `json:"last_retry_at,omitempty"`

// 	// Lifecycle
// 	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
// 	PaidAt    *time.Time `json:"paid_at,omitempty"`

// 	// Immutable Lock (Prevents modification after finalization)
// 	IsFinalized bool `gorm:"default:false" json:"is_finalized"`

// 	// JSON Data
// 	GatewayResponse JSONMap `gorm:"type:jsonb;default:'{}'" json:"gateway_response"`
// 	Metadata        JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

// 	// Relationships
// 	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
// 	Invoice      Invoice      `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`

// 	CreatedAt time.Time      `json:"created_at"`
// 	UpdatedAt time.Time      `json:"updated_at"`
// 	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // ============================================
// // PAYMENT TRANSACTION MODEL
// // ============================================

// type PaymentTransaction struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID  string `gorm:"type:uuid;not null;index" json:"subscription_id"`
// 	PaymentIntentID string `gorm:"type:uuid;not null;index" json:"payment_intent_id"`
// 	InvoiceID       string `gorm:"type:uuid;not null;index" json:"invoice_id"`
// 	SchoolID        string `gorm:"type:uuid;not null;index" json:"school_id"`
// 	UserID          string `gorm:"type:uuid;not null;index" json:"user_id"`

// 	// Financial Details
// 	Amount   decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"amount"`
// 	Fee      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"fee"`
// 	Tax      decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"tax"`
// 	Currency Currency        `gorm:"type:varchar(10);default:'NGN'" json:"currency"`

// 	// Payment Details
// 	PaymentMethod PaymentMethod  `gorm:"type:varchar(30)" json:"payment_method"`
// 	PaymentStatus PaymentStatus  `gorm:"type:varchar(30);default:'pending';index" json:"payment_status"`
// 	Gateway       PaymentGateway `gorm:"type:varchar(30);index" json:"gateway"`

// 	// References
// 	Reference            string `gorm:"uniqueIndex" json:"reference"`
// 	TransactionID        string `gorm:"uniqueIndex" json:"transaction_id"`
// 	GatewayTransactionID string `gorm:"index" json:"gateway_transaction_id"`

// 	Description string `json:"description"`

// 	// Dates
// 	PaidAt *time.Time `json:"paid_at,omitempty"`

// 	// Immutable Financial Lock (Prevents modification after reconciliation)
// 	IsLocked bool `gorm:"default:false" json:"is_locked"`

// 	// JSON Data
// 	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

// 	// Relationships
// 	Subscription  Subscription  `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
// 	Invoice       Invoice       `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
// 	PaymentIntent PaymentIntent `gorm:"foreignKey:PaymentIntentID" json:"payment_intent,omitempty"`

// 	CreatedAt time.Time `json:"created_at"`
// 	UpdatedAt time.Time `json:"updated_at"`
// }

// // ============================================
// // PAYMENT EVENT LOG MODEL (AUDIT TRAIL)
// // ============================================

// type PaymentEventLog struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	PaymentIntentID string `gorm:"type:uuid;index" json:"payment_intent_id"`
// 	TransactionID   string `gorm:"type:uuid;index" json:"transaction_id"`

// 	EventType    string `gorm:"index;not null" json:"event_type"`
// 	StatusBefore string `json:"status_before,omitempty"`
// 	StatusAfter  string `json:"status_after,omitempty"`

// 	Message string `json:"message,omitempty"`

// 	Payload JSONMap `gorm:"type:jsonb;default:'{}'" json:"payload"`

// 	CreatedAt time.Time `json:"created_at"`
// }

// // ============================================
// // WEBHOOK EVENT MODEL
// // ============================================

// type WebhookEvent struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	Gateway   PaymentGateway `gorm:"type:varchar(30);index" json:"gateway"`
// 	EventType string         `gorm:"index" json:"event_type"`

// 	Reference string `gorm:"index" json:"reference"`

// 	IdempotencyKey string `gorm:"uniqueIndex" json:"idempotency_key"`

// 	Status WebhookStatus `gorm:"type:varchar(20);default:'pending';index" json:"status"`

// 	RetryCount   int        `gorm:"default:0" json:"retry_count"`
// 	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
// 	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`
// 	ErrorMessage string     `json:"error_message,omitempty"`

// 	Payload JSONMap `gorm:"type:jsonb;default:'{}'" json:"payload"`

// 	CreatedAt time.Time `json:"created_at"`
// }

// // ============================================
// // SUBSCRIPTION HISTORY MODEL
// // ============================================

// type SubscriptionHistory struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
// 	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
// 	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

// 	OldTier   SubscriptionTier   `json:"old_tier,omitempty"`
// 	NewTier   SubscriptionTier   `json:"new_tier,omitempty"`
// 	OldStatus SubscriptionStatus `json:"old_status,omitempty"`
// 	NewStatus SubscriptionStatus `json:"new_status,omitempty"`

// 	OldAmount decimal.Decimal `gorm:"type:decimal(20,2)" json:"old_amount,omitempty"`
// 	NewAmount decimal.Decimal `gorm:"type:decimal(20,2)" json:"new_amount,omitempty"`

// 	ChangeReason string `json:"change_reason"`
// 	ChangedBy    string `json:"changed_by"`

// 	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

// 	CreatedAt time.Time `json:"created_at"`
// }

// // ============================================
// // EMAIL NOTIFICATION MODEL
// // ============================================

// type EmailNotification struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`
// 	SchoolID       string `gorm:"type:uuid;not null;index" json:"school_id"`
// 	UserID         string `gorm:"type:uuid;not null;index" json:"user_id"`

// 	EmailType      EmailType `json:"email_type"`
// 	RecipientEmail string    `gorm:"index" json:"recipient_email"`
// 	Subject        string    `json:"subject"`

// 	Status       string `gorm:"index" json:"status"`
// 	ErrorMessage string `json:"error_message,omitempty"`

// 	SentAt time.Time `json:"sent_at"`

// 	Metadata JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`

// 	CreatedAt time.Time `json:"created_at"`
// }

// // ============================================
// // REMINDER SCHEDULE MODEL
// // ============================================

// type ReminderSchedule struct {
// 	ID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

// 	SubscriptionID string `gorm:"type:uuid;not null;index" json:"subscription_id"`

// 	ReminderType string `gorm:"index" json:"reminder_type"`
// 	DaysBefore   int    `json:"days_before"`

// 	Status string `gorm:"index" json:"status"`

// 	SentAt *time.Time `json:"sent_at,omitempty"`

// 	CreatedAt time.Time `json:"created_at"`
// 	UpdatedAt time.Time `json:"updated_at"`
// }

// // ============================================
// // TABLE NAMES
// // ============================================

// func (Subscription) TableName() string        { return "subscriptions" }
// func (Invoice) TableName() string             { return "invoices" }
// func (PaymentIntent) TableName() string       { return "payment_intents" }
// func (PaymentTransaction) TableName() string  { return "payment_transactions" }
// func (PaymentEventLog) TableName() string     { return "payment_event_logs" }
// func (WebhookEvent) TableName() string        { return "webhook_events" }
// func (SubscriptionHistory) TableName() string { return "subscription_histories" }
// func (EmailNotification) TableName() string   { return "email_notifications" }
// func (ReminderSchedule) TableName() string    { return "reminder_schedules" }