package dto

import (
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

// ============================================
// REQUEST DTOs
// ============================================

type CreateSubscriptionRequest struct {
    SchoolID      string                 `json:"school_id" binding:"required,uuid"`
    Tier          string                 `json:"tier" binding:"required,oneof=basic premium enterprise"`
    Interval      string                 `json:"interval" binding:"required,oneof=monthly quarterly yearly"`
    Gateway       string                 `json:"gateway" binding:"required,oneof=stripe paystack flutterwave"`
    PaymentMethod string                 `json:"payment_method" binding:"omitempty,oneof=card bank_transfer wallet ussd qr"`
    AutoRenew     bool                   `json:"auto_renew"`
    SuccessURL    string                 `json:"success_url"`
    CancelURL     string                 `json:"cancel_url"`
    CallbackURL   string                 `json:"callback_url"`
    Email         string                 `json:"email" binding:"required,email"`
    Metadata      map[string]interface{} `json:"metadata"`
}

type UpdateSubscriptionRequest struct {
    Tier            *string `json:"tier" binding:"omitempty,oneof=basic premium enterprise"`
    AutoRenew       *bool   `json:"auto_renew"`
    CancelAtEndDate *bool   `json:"cancel_at_end_date"`
    Status          *string `json:"status" binding:"omitempty,oneof=active cancelled pending trial"`
}

type RenewSubscriptionRequest struct {
    Interval string `json:"interval" binding:"required,oneof=monthly quarterly yearly"`
}

type CancelSubscriptionRequest struct {
    CancelImmediately bool   `json:"cancel_immediately"`
    Reason            string `json:"reason"`
}

type CreatePaymentIntentRequest struct {
    SubscriptionID string `json:"subscription_id" binding:"required,uuid"`
    SuccessURL     string `json:"success_url"`
    CancelURL      string `json:"cancel_url"`
}

type VerifyPaymentRequest struct {
    Reference string `json:"reference" binding:"required"`
    Gateway   string `json:"gateway" binding:"required,oneof=stripe paystack flutterwave"`
}

// ============================================
// RESPONSE DTOs
// ============================================

type SubscriptionResponse struct {
    ID              string                 `json:"id"`
    UserID          string                 `json:"user_id,omitempty"`
    SchoolID        string                 `json:"school_id"`
    Tier            string                 `json:"tier"`
    Status          string                 `json:"status"`
    Gateway         string                 `json:"gateway"`
    Amount          decimal.Decimal        `json:"amount"`
    Currency        string                 `json:"currency"`
    PaymentInterval string                 `json:"payment_interval"`
    StartDate       time.Time              `json:"start_date"`
    EndDate         time.Time              `json:"end_date"`
    TrialEndsAt     *time.Time             `json:"trial_ends_at,omitempty"`
    AutoRenew       bool                   `json:"auto_renew"`
    CancelAtEndDate bool                   `json:"cancel_at_end_date"`
    LastPaymentDate *time.Time             `json:"last_payment_date,omitempty"`
    NextPaymentDate *time.Time             `json:"next_payment_date,omitempty"`
    MaxStudents     int                    `json:"max_students"`
    MaxTeachers     int                    `json:"max_teachers"`
    MaxExams        int                    `json:"max_exams"`
    MaxQuestions    int                    `json:"max_questions"`
    MaxStorageMB    int                    `json:"max_storage_mb"`
    Features        map[string]interface{} `json:"features"`
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}

type PaymentIntentResponse struct {
    ID               string          `json:"id"`
    SubscriptionID   string          `json:"subscription_id"`
    InvoiceID        string          `json:"invoice_id"`
    Gateway          string          `json:"gateway"`
    Amount           decimal.Decimal `json:"amount"`
    Currency         string          `json:"currency"`
    Reference        string          `json:"reference"`
    ClientSecret     string          `json:"client_secret,omitempty"`
    AuthorizationURL string          `json:"authorization_url,omitempty"`
    AccessCode       string          `json:"access_code,omitempty"`
    Status           string          `json:"status"`
    ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
    PaymentMethod    string          `json:"payment_method,omitempty"`
    CreatedAt        time.Time       `json:"created_at"`
}

type InvoiceResponse struct {
    ID            string                 `json:"id"`
    InvoiceNumber string                 `json:"invoice_number"`
    Amount        decimal.Decimal        `json:"amount"`
    Tax           decimal.Decimal        `json:"tax"`
    Discount      decimal.Decimal        `json:"discount"`
    Total         decimal.Decimal        `json:"total"`
    Currency      string                 `json:"currency"`
    Status        string                 `json:"status"`
    DueDate       time.Time              `json:"due_date"`
    PaidAt        *time.Time             `json:"paid_at,omitempty"`
    PDFURL        string                 `json:"pdf_url,omitempty"`
    Items         map[string]interface{} `json:"items,omitempty"`
    CreatedAt     time.Time              `json:"created_at"`
    UpdatedAt     time.Time              `json:"updated_at"`
}

type PaymentTransactionResponse struct {
    ID             string          `json:"id"`
    SubscriptionID string          `json:"subscription_id"`
    InvoiceID      string          `json:"invoice_id"`
    Amount         decimal.Decimal `json:"amount"`
    Fee            decimal.Decimal `json:"fee"`
    Tax            decimal.Decimal `json:"tax"`
    Currency       string          `json:"currency"`
    PaymentMethod  string          `json:"payment_method"`
    Status         string          `json:"status"`
    Gateway        string          `json:"gateway"`
    Reference      string          `json:"reference"`
    Description    string          `json:"description"`
    PaidAt         *time.Time      `json:"paid_at,omitempty"`
    CreatedAt      time.Time       `json:"created_at"`
}

type CreateSubscriptionResponse struct {
    Subscription  *SubscriptionResponse  `json:"subscription"`
    PaymentIntent *PaymentIntentResponse `json:"payment_intent"`
    Invoice       *InvoiceResponse       `json:"invoice"`
}

type SubscriptionStatusResponse struct {
    IsActive      bool      `json:"is_active"`
    Status        string    `json:"status"`
    Tier          string    `json:"tier"`
    StartDate     time.Time `json:"start_date"`
    EndDate       time.Time `json:"end_date"`
    DaysRemaining int       `json:"days_remaining"`
    AutoRenew     bool      `json:"auto_renew"`
}

type SubscriptionUsageResponse struct {
    CurrentStudents  int     `json:"current_students"`
    CurrentTeachers  int     `json:"current_teachers"`
    CurrentExams     int     `json:"current_exams"`
    CurrentQuestions int     `json:"current_questions"`
    StorageUsedMB    int     `json:"storage_used_mb"`
    MaxStudents      int     `json:"max_students"`
    MaxTeachers      int     `json:"max_teachers"`
    MaxExams         int     `json:"max_exams"`
    MaxQuestions     int     `json:"max_questions"`
    MaxStorageMB     int     `json:"max_storage_mb"`
    StudentPercent   float64 `json:"student_percent"`
    TeacherPercent   float64 `json:"teacher_percent"`
    ExamPercent      float64 `json:"exam_percent"`
    StoragePercent   float64 `json:"storage_percent"`
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func GenerateInvoiceNumber() string {
    return fmt.Sprintf("INV-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
}

func GenerateReference() string {
    return fmt.Sprintf("REF-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
}


// package dto

// import (
//     "fmt"
//     "time"

//     "github.com/google/uuid"
//     "github.com/shopspring/decimal"
// )

// // ============================================
// // REQUEST DTOs
// // ============================================

// type CreateSubscriptionRequest struct {
//     SchoolID     string `json:"school_id" binding:"required,uuid"`
//     Tier         string `json:"tier" binding:"required,oneof=basic premium enterprise"`
//     Interval     string `json:"interval" binding:"required,oneof=monthly quarterly yearly"`
//     Gateway      string `json:"gateway" binding:"required,oneof=stripe paystack flutterwave"`
//     PaymentMethod string `json:"payment_method" binding:"omitempty,oneof=card bank_transfer wallet ussd qr"`
//     AutoRenew    bool   `json:"auto_renew"`
//     SuccessURL   string `json:"success_url"`
//     CancelURL    string `json:"cancel_url"`
//     CallbackURL  string `json:"callback_url"`
//     Email        string `json:"email" binding:"required,email"`
//     Metadata     map[string]interface{} `json:"metadata"`
// }

// type UpdateSubscriptionRequest struct {
//     Tier            *string `json:"tier" binding:"omitempty,oneof=basic premium enterprise"`
//     AutoRenew       *bool   `json:"auto_renew"`
//     CancelAtEndDate *bool   `json:"cancel_at_end_date"`
//     Status          *string `json:"status" binding:"omitempty,oneof=active cancelled pending"`
// }

// type RenewSubscriptionRequest struct {
//     Interval string `json:"interval" binding:"required,oneof=monthly quarterly yearly"`
// }

// type CancelSubscriptionRequest struct {
//     CancelImmediately bool   `json:"cancel_immediately"`
//     Reason            string `json:"reason"`
// }

// type CreatePaymentIntentRequest struct {
//     SubscriptionID string `json:"subscription_id" binding:"required,uuid"`
//     SuccessURL     string `json:"success_url"`
//     CancelURL      string `json:"cancel_url"`
// }

// type VerifyPaymentRequest struct {
//     Reference string `json:"reference" binding:"required"`
//     Gateway   string `json:"gateway" binding:"required,oneof=stripe paystack flutterwave"`
// }

// // ============================================
// // RESPONSE DTOs
// // ============================================

// type SubscriptionResponse struct {
//     ID              string               `json:"id"`
//     UserID          string               `json:"user_id"`
//     SchoolID        string               `json:"school_id"`
//     Tier            string               `json:"tier"`
//     Status          string               `json:"status"`
//     Gateway         string               `json:"gateway"`
//     Amount          decimal.Decimal      `json:"amount"`
//     Currency        string               `json:"currency"`
//     PaymentInterval string               `json:"payment_interval"`
//     StartDate       time.Time            `json:"start_date"`
//     EndDate         time.Time            `json:"end_date"`
//     TrialEndsAt     *time.Time           `json:"trial_ends_at,omitempty"`
//     AutoRenew       bool                 `json:"auto_renew"`
//     CancelAtEndDate bool                 `json:"cancel_at_end_date"`
//     LastPaymentDate *time.Time           `json:"last_payment_date,omitempty"`
//     NextPaymentDate *time.Time           `json:"next_payment_date,omitempty"`
//     MaxStudents     int                  `json:"max_students"`
//     MaxTeachers     int                  `json:"max_teachers"`
//     MaxExams        int                  `json:"max_exams"`
//     MaxQuestions    int                  `json:"max_questions"`
//     MaxStorageMB    int                  `json:"max_storage_mb"`
//     Features        map[string]interface{} `json:"features"`
//     CreatedAt       time.Time            `json:"created_at"`
//     UpdatedAt       time.Time            `json:"updated_at"`
// }

// type PaymentIntentResponse struct {
//     ID               string          `json:"id"`
//     SubscriptionID   string          `json:"subscription_id"`
//     InvoiceID        string          `json:"invoice_id"`
//     Gateway          string          `json:"gateway"`
//     Amount           decimal.Decimal `json:"amount"`
//     Currency         string          `json:"currency"`
//     Reference        string          `json:"reference"`
//     ClientSecret     string          `json:"client_secret,omitempty"`
//     AuthorizationURL string          `json:"authorization_url,omitempty"`
//     AccessCode       string          `json:"access_code,omitempty"`
//     Status           string          `json:"status"`
//     ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
//     PaymentMethod    string          `json:"payment_method,omitempty"`
//     CreatedAt        time.Time       `json:"created_at"`
// }

// type InvoiceResponse struct {
//     ID            string               `json:"id"`
//     InvoiceNumber string               `json:"invoice_number"`
//     Amount        decimal.Decimal      `json:"amount"`
//     Tax           decimal.Decimal      `json:"tax"`
//     Discount      decimal.Decimal      `json:"discount"`
//     Total         decimal.Decimal      `json:"total"`
//     Currency      string               `json:"currency"`
//     Status        string               `json:"status"`
//     DueDate       time.Time            `json:"due_date"`
//     PaidAt        *time.Time           `json:"paid_at,omitempty"`
//     PDFURL        string               `json:"pdf_url,omitempty"`
//     Items         map[string]interface{} `json:"items,omitempty"`
//     CreatedAt     time.Time            `json:"created_at"`
//     UpdatedAt     time.Time            `json:"updated_at"`
// }

// type PaymentTransactionResponse struct {
//     ID             string          `json:"id"`
//     SubscriptionID string          `json:"subscription_id"`
//     InvoiceID      string          `json:"invoice_id"`
//     Amount         decimal.Decimal `json:"amount"`
//     Fee            decimal.Decimal `json:"fee"`
//     Tax            decimal.Decimal `json:"tax"`
//     Currency       string          `json:"currency"`
//     PaymentMethod  string          `json:"payment_method"`
//     Status         string          `json:"status"`
//     Gateway        string          `json:"gateway"`
//     Reference      string          `json:"reference"`
//     Description    string          `json:"description"`
//     PaidAt         *time.Time      `json:"paid_at,omitempty"`
//     CreatedAt      time.Time       `json:"created_at"`
// }

// type CreateSubscriptionResponse struct {
//     Subscription  *SubscriptionResponse  `json:"subscription"`
//     PaymentIntent *PaymentIntentResponse `json:"payment_intent"`
//     Invoice       *InvoiceResponse       `json:"invoice"`
// }

// type SubscriptionStatusResponse struct {
//     IsActive      bool      `json:"is_active"`
//     Status        string    `json:"status"`
//     Tier          string    `json:"tier"`
//     StartDate     time.Time `json:"start_date"`
//     EndDate       time.Time `json:"end_date"`
//     DaysRemaining int       `json:"days_remaining"`
//     AutoRenew     bool      `json:"auto_renew"`
// }

// type SubscriptionUsageResponse struct {
//     CurrentStudents  int     `json:"current_students"`
//     CurrentTeachers  int     `json:"current_teachers"`
//     CurrentExams     int     `json:"current_exams"`
//     CurrentQuestions int     `json:"current_questions"`
//     StorageUsedMB    int     `json:"storage_used_mb"`
//     MaxStudents      int     `json:"max_students"`
//     MaxTeachers      int     `json:"max_teachers"`
//     MaxExams         int     `json:"max_exams"`
//     MaxQuestions     int     `json:"max_questions"`
//     MaxStorageMB     int     `json:"max_storage_mb"`
//     StudentPercent   float64 `json:"student_percent"`
//     TeacherPercent   float64 `json:"teacher_percent"`
//     ExamPercent      float64 `json:"exam_percent"`
//     StoragePercent   float64 `json:"storage_percent"`
// }

// // ============================================
// // HELPER FUNCTIONS
// // ============================================

// func GenerateInvoiceNumber() string {
//     return fmt.Sprintf("INV-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
// }

// func GenerateReference() string {
//     return fmt.Sprintf("REF-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
// }