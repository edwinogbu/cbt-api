package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// ============================================
// STEP 1: Plan Selection
// ============================================

type StartOnboardingRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Tier     string `json:"tier" binding:"required,oneof=basic premium enterprise"`
	Interval string `json:"interval" binding:"required,oneof=monthly quarterly yearly"`
	Gateway  string `json:"gateway" binding:"required,oneof=paystack flutterwave stripe"`
}

type StartOnboardingResponse struct {
	SessionID string          `json:"session_id"`
	Amount    decimal.Decimal `json:"amount"`
	Tier      string          `json:"tier"`
	Interval  string          `json:"interval"`
	Gateway   string          `json:"gateway"`
	Message   string          `json:"message"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// ============================================
// STEP 2: Create Administrator
// ============================================

type CreateAdminRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=8"`
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	PhoneNumber string `json:"phone_number"`
}

type CreateAdminResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	NextStep  string `json:"next_step"`
}

// ============================================
// STEP 3: Verify Email
// ============================================

type VerifyEmailRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	OTPCode   string `json:"otp_code" binding:"required,len=6"`
}

type VerifyEmailResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Message  string `json:"message"`
	NextStep string `json:"next_step"`
}

// ============================================
// STEP 4: School Information
// ============================================

type RegisterSchoolRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	SchoolName  string `json:"school_name" binding:"required,min=3,max=100"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	SchoolEmail string `json:"school_email" binding:"required,email"`
}

type RegisterSchoolResponse struct {
	SchoolID   string `json:"school_id"`
	SchoolCode string `json:"school_code"`
	Message    string `json:"message"`
	NextStep   string `json:"next_step"`
}

// ============================================
// STEP 6: Create Subscription (Payment)
// ============================================

type CreateSubscriptionRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	CallbackURL string `json:"callback_url"`
	SuccessURL  string `json:"success_url"`
	CancelURL   string `json:"cancel_url"`
}

type CreateSubscriptionResponse struct {
	SubscriptionID  string          `json:"subscription_id"`
	PaymentIntentID string          `json:"payment_intent_id"`
	PaymentLink     string          `json:"payment_link"`
	Amount          decimal.Decimal `json:"amount"`
	Gateway         string          `json:"gateway"`
	Reference       string          `json:"reference"`
	Message         string          `json:"message"`
	NextStep        string          `json:"next_step"`
	ExpiresAt       time.Time       `json:"expires_at"`
}

// ============================================
// GET AVAILABLE GATEWAYS
// ============================================

type GatewayInfo struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsAvailable bool   `json:"is_available"`
	Currency    string `json:"currency"`
}

// ============================================
// PAYMENT STATUS
// ============================================

type PaymentStatusResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IsComplete  bool   `json:"is_complete"`
	PaymentLink string `json:"payment_link,omitempty"`
}

// ============================================
// ACTIVATION RESPONSE
// ============================================

type ActivationResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	SchoolID       string `json:"school_id"`
	AdminID        string `json:"admin_id"`
	SubscriptionID string `json:"subscription_id"`
	RedirectURL    string `json:"redirect_url"`
}

// ============================================
// ONBOARDING STATUS
// ============================================

type OnboardingStatusResponse struct {
	SessionID       string    `json:"session_id"`
	Status          string    `json:"status"`
	Tier            string    `json:"tier"`
	Interval        string    `json:"interval"`
	Gateway         string    `json:"gateway"`
	Amount          string    `json:"amount"`
	HasUser         bool      `json:"has_user"`
	HasSchool       bool      `json:"has_school"`
	HasSubscription bool      `json:"has_subscription"`
	EmailVerified   bool      `json:"email_verified"`
	NextStep        string    `json:"next_step"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// ============================================
// GET SESSION BY EMAIL
// ============================================

type GetSessionByEmailResponse struct {
	SessionID string    `json:"session_id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}


// package dto

// // ============================================
// // STEP 1: Plan Selection
// // ============================================

// type PlanSelectionRequest struct {
//     Plan string `json:"plan" binding:"required,oneof=starter professional institutional enterprise"`
// }

// type PlanSelectionResponse struct {
//     SessionID string `json:"session_id"`
//     Plan      string `json:"plan"`
//     Status    string `json:"status"`
//     NextStep  string `json:"next_step"`
//     ExpiresAt string `json:"expires_at"`
// }

// // ============================================
// // STEP 2: Create Administrator
// // ============================================

// type CreateAdminRequest struct {
//     SessionID string `json:"session_id" binding:"required"`
//     FirstName string `json:"first_name" binding:"required"`
//     LastName  string `json:"last_name" binding:"required"`
//     Email     string `json:"email" binding:"required,email"`
//     Phone     string `json:"phone" binding:"required"`
//     Password  string `json:"password" binding:"required,min=8"`
// }

// type CreateAdminResponse struct {
//     UserID    string `json:"user_id"`
//     Email     string `json:"email"`
//     FirstName string `json:"first_name"`
//     LastName  string `json:"last_name"`
//     Status    string `json:"status"`
//     NextStep  string `json:"next_step"`
// }

// // ============================================
// // STEP 3: Verify Email
// // ============================================

// type VerifyEmailRequest struct {
//     SessionID string `json:"session_id" binding:"required"`
//     Code      string `json:"code" binding:"required,len=6"`
// }

// type VerifyEmailResponse struct {
//     Status   string `json:"status"`
//     NextStep string `json:"next_step"`
// }

// // ============================================
// // STEP 4: School Information
// // ============================================

// type SchoolRegistrationRequest struct {
//     SessionID      string `json:"session_id" binding:"required"`
//     SchoolName     string `json:"school_name" binding:"required"`
//     Address        string `json:"address" binding:"required"`
//     City           string `json:"city" binding:"required"`
//     State          string `json:"state" binding:"required"`
//     Country        string `json:"country" binding:"required"`
//     Phone          string `json:"phone" binding:"required"`
//     Email          string `json:"email" binding:"required,email"`
//     Website        string `json:"website"`
//     PrincipalName  string `json:"principal_name" binding:"required"`
//     PrincipalEmail string `json:"principal_email" binding:"required,email"`
//     PrincipalPhone string `json:"principal_phone" binding:"required"`
//     SchoolType     string `json:"school_type"`
//     LGA            string `json:"lga"`
// }

// type SchoolRegistrationResponse struct {
//     SchoolID   string `json:"school_id"`
//     SchoolName string `json:"school_name"`
//     SchoolCode string `json:"school_code"`
//     Status     string `json:"status"`
//     NextStep   string `json:"next_step"`
// }

// // ============================================
// // STEP 6: Create Subscription (Payment)
// // ============================================

// type CreateSubscriptionOnboardingRequest struct {
//     SessionID string `json:"session_id" binding:"required"`
// }

// type CreateSubscriptionOnboardingResponse struct {
//     SubscriptionID string `json:"subscription_id"`
//     PaymentURL     string `json:"payment_url"`
//     Reference      string `json:"reference"`
//     Amount         string `json:"amount"`
//     Currency       string `json:"currency"`
//     Status         string `json:"status"`
//     NextStep       string `json:"next_step"`
// }

// // ============================================
// // ACTIVATION RESPONSE
// // ============================================

// type ActivationResponse struct {
//     Success        bool   `json:"success"`
//     Message        string `json:"message"`
//     SchoolID       string `json:"school_id"`
//     AdminID        string `json:"admin_id"`
//     SubscriptionID string `json:"subscription_id"`
//     RedirectURL    string `json:"redirect_url"`
// }

// // ============================================
// // ONBOARDING STATUS
// // ============================================

// type OnboardingStatusResponse struct {
//     SessionID       string `json:"session_id"`
//     Status          string `json:"status"`
//     Plan            string `json:"plan"`
//     HasUser         bool   `json:"has_user"`
//     HasSchool       bool   `json:"has_school"`
//     HasSubscription bool   `json:"has_subscription"`
//     EmailVerified   bool   `json:"email_verified"`
//     NextStep        string `json:"next_step"`
// }