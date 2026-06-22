package payment

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/shopspring/decimal"
    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/client"
    "github.com/stripe/stripe-go/v81/webhook"

    "cbt-api/internal/models"
)

type StripeGateway struct {
    secretKey     string
    webhookSecret string
    api           *client.API
}

func NewStripeGateway() *StripeGateway {
    secretKey := os.Getenv("STRIPE_SECRET_KEY")
    webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
    stripe.Key = secretKey

    return &StripeGateway{
        secretKey:     secretKey,
        webhookSecret: webhookSecret,
        api:           client.New(secretKey, nil),
    }
}

// ============================================
// ONE‑TIME PAYMENTS (PaymentIntent)
// ============================================

func (g *StripeGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
    idempotencyKey := fmt.Sprintf("payment_%s_%s_%d", req.SchoolID, req.UserID, time.Now().UnixNano())

    params := &stripe.PaymentIntentParams{
        Amount:             stripe.Int64(int64(req.Amount.Mul(decimal.NewFromInt(100)).IntPart())),
        Currency:           stripe.String(string(req.Currency)),
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
        Metadata: map[string]string{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
        },
    }
    params.IdempotencyKey = stripe.String(idempotencyKey)

    pi, err := g.api.PaymentIntents.New(params)
    if err != nil {
        return nil, fmt.Errorf("stripe payment intent error: %w", err)
    }

    return &models.PaymentIntent{
        ID:               pi.ID,
        Gateway:          models.GatewayStripe,
        Reference:        pi.ClientSecret,
        ClientSecret:     pi.ClientSecret,
        Amount:           decimal.NewFromInt(pi.Amount),
        Currency:         models.Currency(pi.Currency),
        Status:           models.PaymentIntentStatus(pi.Status),
        CreatedAt:        time.Unix(pi.Created, 0),
        AuthorizationURL: "",
    }, nil
}

func (g *StripeGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
    pi, err := g.api.PaymentIntents.Get(reference, nil)
    if err != nil {
        return nil, fmt.Errorf("stripe payment intent retrieve error: %w", err)
    }

    status := "pending"
    if pi.Status == stripe.PaymentIntentStatusSucceeded {
        status = "succeeded"
    } else if pi.Status == stripe.PaymentIntentStatusCanceled {
        status = "failed"
    }

    return &PaymentVerification{
        Status:    status,
        Amount:    decimal.NewFromInt(pi.Amount),
        Reference: pi.ID,
        GatewayData: map[string]interface{}{
            "status":         pi.Status,
            "payment_method": pi.PaymentMethod,
        },
    }, nil
}

// ============================================
// RECURRING SUBSCRIPTIONS (Checkout Sessions)
// ============================================

func (g *StripeGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
    // 1. Find or create customer (reuse by email)
    cust, err := g.findOrCreateCustomer(req)
    if err != nil {
        return nil, err
    }

    priceID := getStripePriceID(req.Tier, req.Interval)
    if priceID == "" {
        return nil, fmt.Errorf("no Stripe price ID found for tier %s interval %s", req.Tier, req.Interval)
    }

    successURL := req.SuccessURL
    if successURL == "" {
        successURL = "https://yourapp.com/success"
    }
    cancelURL := req.CancelURL
    if cancelURL == "" {
        cancelURL = "https://yourapp.com/cancel"
    }

    idempotencyKey := fmt.Sprintf("subscription_%s_%s_%d", req.SchoolID, req.UserID, time.Now().UnixNano())

    sessionParams := &stripe.CheckoutSessionParams{
        Customer:   stripe.String(cust.ID),
        Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
        SuccessURL: stripe.String(successURL),
        CancelURL:  stripe.String(cancelURL),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                Price:    stripe.String(priceID),
                Quantity: stripe.Int64(1),
            },
        },
        Metadata: map[string]string{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
            "tier":      string(req.Tier),
        },
    }
    sessionParams.IdempotencyKey = stripe.String(idempotencyKey)

    sess, err := g.api.CheckoutSessions.New(sessionParams)
    if err != nil {
        return nil, fmt.Errorf("stripe checkout session error: %w", err)
    }

    return &SubscriptionResult{
        SubscriptionID:        sess.ID,       // Checkout Session ID (not subscription ID)
        GatewaySubscriptionID: "",            // will be filled by webhook
        GatewayCustomerID:     cust.ID,
        AuthorizationURL:      sess.URL,
        Status:                string(sess.Status),
    }, nil
}

func (g *StripeGateway) findOrCreateCustomer(req *SubscriptionRequest) (*stripe.Customer, error) {
    // Search for existing customer with the given email
    params := &stripe.CustomerListParams{}
    params.Limit = stripe.Int64(1)
    params.Filters.AddFilter("email", "", req.Email)

    iter := g.api.Customers.List(params)
    if iter.Next() {
        return iter.Customer(), nil
    }
    if err := iter.Err(); err != nil {
        return nil, fmt.Errorf("stripe customer search error: %w", err)
    }

    // Create new customer
    customerParams := &stripe.CustomerParams{
        Email: stripe.String(req.Email),
        Name:  stripe.String(req.CustomerName),
        Metadata: map[string]string{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
        },
    }
    cust, err := g.api.Customers.New(customerParams)
    if err != nil {
        return nil, fmt.Errorf("stripe customer creation error: %w", err)
    }
    return cust, nil
}

// ============================================
// SUBSCRIPTION MANAGEMENT
// ============================================

func (g *StripeGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
    params := &stripe.SubscriptionParams{
        CancelAtPeriodEnd: stripe.Bool(true),
    }
    _, err := g.api.Subscriptions.Update(subscriptionID, params)
    if err != nil {
        return fmt.Errorf("stripe subscription cancellation error: %w", err)
    }
    return nil
}

func (g *StripeGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
    sub, err := g.api.Subscriptions.Get(subscriptionID, nil)
    if err != nil {
        return nil, fmt.Errorf("stripe subscription get error: %w", err)
    }

    active := sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing
    cancelAtPeriodEnd := sub.CancelAtPeriodEnd

    var currentPeriodStart, currentPeriodEnd time.Time
    if sub.CurrentPeriodStart > 0 {
        currentPeriodStart = time.Unix(sub.CurrentPeriodStart, 0)
    }
    if sub.CurrentPeriodEnd > 0 {
        currentPeriodEnd = time.Unix(sub.CurrentPeriodEnd, 0)
    }

    return &SubscriptionStatus{
        Active:             active,
        Status:             string(sub.Status),
        CurrentPeriodStart: currentPeriodStart,
        CurrentPeriodEnd:   currentPeriodEnd,
        CancelAtPeriodEnd:  cancelAtPeriodEnd,
    }, nil
}

// ============================================
// WEBHOOK HANDLING
// ============================================

func (g *StripeGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
    // Use the webhook package for signature verification
    event, err := webhook.ConstructEvent(payload, signature, g.webhookSecret)
    if err != nil {
        return nil, fmt.Errorf("stripe webhook signature verification failed: %w", err)
    }

    var rawMap map[string]interface{}
    if err := json.Unmarshal(event.Data.Raw, &rawMap); err != nil {
        return nil, err
    }
    // Store the unique Stripe event ID for idempotency
    rawMap["stripe_event_id"] = event.ID

    we := &WebhookEvent{
        Type:    string(event.Type),
        RawData: rawMap,
    }

    switch event.Type {
    case "checkout.session.completed":
        var sess stripe.CheckoutSession
        if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
            return nil, err
        }
        we.Reference = sess.ID
        we.Type = "payment_success"
        if sess.Subscription != nil {
            we.GatewaySubscriptionID = sess.Subscription.ID
            we.Type = "subscription_activated"
        }

    case "customer.subscription.created", "customer.subscription.updated":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            return nil, err
        }
        we.GatewaySubscriptionID = sub.ID
        if sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing {
            we.Type = "subscription_activated"
        } else if sub.CancelAtPeriodEnd {
            we.Type = "subscription_cancelled"
        }

    case "customer.subscription.deleted":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            return nil, err
        }
        we.GatewaySubscriptionID = sub.ID
        we.Type = "subscription_cancelled"

    case "invoice.payment_succeeded":
        var invoice stripe.Invoice
        if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
            return nil, err
        }
        if invoice.Subscription != nil {
            we.GatewaySubscriptionID = invoice.Subscription.ID
        }
        if invoice.PaymentIntent != nil {
            we.Reference = invoice.PaymentIntent.ID
        }
        we.Amount = decimal.NewFromInt(invoice.AmountPaid)
        we.Currency = string(invoice.Currency)
        we.Type = "payment_success"
    }

    return we, nil
}

func (g *StripeGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
    _, err := g.ParseWebhook(ctx, payload, signature)
    return err
}

// ============================================
// HELPER: Get Stripe Price ID from environment
// ============================================

func getStripePriceID(tier models.SubscriptionTier, interval models.PaymentInterval) string {
    envMap := map[models.SubscriptionTier]map[models.PaymentInterval]string{
        models.TierBasic: {
            models.IntervalMonthly: os.Getenv("STRIPE_PRICE_BASIC_MONTHLY"),
            models.IntervalYearly:  os.Getenv("STRIPE_PRICE_BASIC_YEARLY"),
        },
        models.TierPremium: {
            models.IntervalMonthly: os.Getenv("STRIPE_PRICE_PREMIUM_MONTHLY"),
            models.IntervalYearly:  os.Getenv("STRIPE_PRICE_PREMIUM_YEARLY"),
        },
        models.TierEnterprise: {
            models.IntervalMonthly: os.Getenv("STRIPE_PRICE_ENTERPRISE_MONTHLY"),
            models.IntervalYearly:  os.Getenv("STRIPE_PRICE_ENTERPRISE_YEARLY"),
        },
    }
    if m, ok := envMap[tier]; ok {
        return m[interval]
    }
    return ""
}


