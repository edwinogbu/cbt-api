package payment

import (
    "context"
    "time"

    "github.com/shopspring/decimal"

    "cbt-api/internal/models"
)

type Gateway interface {
    // Payment Methods
    CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error)
    VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error)
    
    // Subscription Methods
    CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error)
    CancelSubscription(ctx context.Context, subscriptionID string) error
    GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error)
    
    // Webhook
    HandleWebhook(ctx context.Context, payload []byte, signature string) error
    
    // NEW: Parse webhook payload into a standard event
    ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
    
}




// WebhookEvent – standard structure for all gateways
type WebhookEvent struct {
    Type                  string // "subscription_activated", "payment_success", "subscription_cancelled", etc.
    GatewaySubscriptionID string // the subscription ID from the gateway
    Reference             string // payment reference (if any)
    Amount                decimal.Decimal
    Currency              string
    RawData               map[string]interface{} // original payload
}

type PaymentRequest struct {
    SchoolID     string
    UserID       string
    Amount       decimal.Decimal
    Currency     models.Currency
    Email        string
    Metadata     map[string]interface{}
    CallbackURL  string
    SuccessURL   string
    CancelURL    string
}

type SubscriptionRequest struct {
    SchoolID        string
    UserID          string
    Tier            models.SubscriptionTier
    Interval        models.PaymentInterval
    Email           string
    CustomerName    string
    PaymentMethodID string
    SuccessURL      string
    CancelURL       string
}

type SubscriptionResult struct {
    SubscriptionID        string // gateway's own ID
    GatewayCustomerID     string
    GatewaySubscriptionID string // same as SubscriptionID, kept for clarity
    ClientSecret          string
    AuthorizationURL      string
    Status                string
}

type PaymentVerification struct {
    Status      string
    Amount      decimal.Decimal
    Reference   string
    GatewayData map[string]interface{}
}

type SubscriptionStatus struct {
    Active      bool
    Status      string
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    CancelAtPeriodEnd  bool
}

