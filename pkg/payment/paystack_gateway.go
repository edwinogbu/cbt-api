package payment

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha512"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/google/uuid"          
    "github.com/shopspring/decimal"

    "cbt-api/internal/models"
)

// PaystackGateway handles one‑time payments and webhooks.
type PaystackGateway struct {
    secretKey string
    client    *http.Client
}

func NewPaystackGateway() *PaystackGateway {
    rawKey := os.Getenv("PAYSTACK_SECRET_KEY")
    // Trim whitespace and newlines
    secretKey := strings.TrimSpace(rawKey)
    if secretKey == "" {
        log.Printf("[Paystack] WARNING: PAYSTACK_SECRET_KEY is not set or empty")
    } else {
        // Log first 10 characters for debugging (never log full key)
        prefix := secretKey
        if len(prefix) > 10 {
            prefix = prefix[:10] + "..."
        }
        log.Printf("[Paystack] Secret key loaded (prefix: %s)", prefix)
    }
    return &PaystackGateway{
        secretKey: secretKey,
        client:    &http.Client{Timeout: 30 * time.Second},
    }
}

type PaystackResponse struct {
    Status  bool                   `json:"status"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data"`
}

func (g *PaystackGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
    log.Printf("[Paystack] CreatePayment started: email=%s, amount=%s %s, callback=%s",
        req.Email, req.Amount.String(), req.Currency, req.CallbackURL)

    // Validate secret key
    if g.secretKey == "" {
        log.Printf("[Paystack] CreatePayment error: PAYSTACK_SECRET_KEY is not set")
        return nil, fmt.Errorf("paystack secret key missing: set PAYSTACK_SECRET_KEY in environment")
    }

    // Amount in kobo (smallest currency unit)
    amountKobo := int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart())
    payload := map[string]interface{}{
        "email":    req.Email,
        "amount":   amountKobo,
        "currency": string(req.Currency),
        "metadata": map[string]interface{}{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
        },
    }

    if req.CallbackURL != "" {
        payload["callback_url"] = req.CallbackURL
    }

    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        log.Printf("[Paystack] CreatePayment marshal error: %v", err)
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    log.Printf("[Paystack] Request payload: %s", string(jsonPayload))

    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(jsonPayload))
    if err != nil {
        log.Printf("[Paystack] CreatePayment request creation error: %v", err)
        return nil, err
    }
    httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(httpReq)
    if err != nil {
        log.Printf("[Paystack] CreatePayment HTTP error: %v", err)
        return nil, fmt.Errorf("paystack request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("[Paystack] CreatePayment read body error: %v", err)
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    log.Printf("[Paystack] Response status: %d", resp.StatusCode)
    if len(body) > 0 {
        log.Printf("[Paystack] Response body: %s", string(body))
    }

    // Handle HTTP errors
    if resp.StatusCode != http.StatusOK {
        // Special handling for 401 – invalid key
        if resp.StatusCode == http.StatusUnauthorized {
            return nil, fmt.Errorf("paystack authentication failed: invalid secret key (check your PAYSTACK_SECRET_KEY environment variable)")
        }
        return nil, fmt.Errorf("paystack returned HTTP %d: %s", resp.StatusCode, string(body))
    }

    var paystackResp PaystackResponse
    if err := json.Unmarshal(body, &paystackResp); err != nil {
        log.Printf("[Paystack] CreatePayment parse error: %v", err)
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if !paystackResp.Status {
        errMsg := paystackResp.Message
        if errMsg == "" {
            if errData, ok := paystackResp.Data["error"].(string); ok {
                errMsg = errData
            } else if msg, ok := paystackResp.Data["message"].(string); ok {
                errMsg = msg
            } else {
                errMsg = "unknown error"
            }
        }
        log.Printf("[Paystack] CreatePayment API error: %s", errMsg)
        return nil, fmt.Errorf("paystack error: %s", errMsg)
    }

    reference, ok := paystackResp.Data["reference"].(string)
    if !ok {
        return nil, fmt.Errorf("paystack response missing reference")
    }
    authURL, ok := paystackResp.Data["authorization_url"].(string)
    if !ok {
        return nil, fmt.Errorf("paystack response missing authorization_url")
    }
    accessCode, ok := paystackResp.Data["access_code"].(string)
    if !ok {
        return nil, fmt.Errorf("paystack response missing access_code")
    }

    log.Printf("[Paystack] CreatePayment success: reference=%s, auth_url=%s", reference, authURL)

    // Generate a local ID (UUID) for the PaymentIntent (Paystack does not return an id)
    localID := uuid.New().String()

    return &models.PaymentIntent{
        ID:               localID,
        Gateway:          models.GatewayPaystack,
        AuthorizationURL: authURL,
        Reference:        reference,
        AccessCode:       accessCode,
        Amount:           req.Amount,
        Currency:         req.Currency,
        Status:           models.IntentPending,
        CreatedAt:        time.Now(),
    }, nil
}

// VerifyPayment remains unchanged (already correct)
func (g *PaystackGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
    log.Printf("[Paystack] VerifyPayment started: reference=%s", reference)

    if g.secretKey == "" {
        return nil, fmt.Errorf("paystack secret key missing")
    }

    url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+g.secretKey)

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("verification request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    var paystackResp PaystackResponse
    if err := json.Unmarshal(body, &paystackResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    status := "pending"
    amount := decimal.Zero

    if paystackResp.Status {
        data := paystackResp.Data
        if txStatus, ok := data["status"].(string); ok {
            switch txStatus {
            case "success":
                status = "succeeded"
            case "failed":
                status = "failed"
            default:
                status = "pending"
            }
        }
        if amt, ok := data["amount"].(float64); ok {
            amount = decimal.NewFromFloat(amt / 100)
        }
    } else {
        return nil, fmt.Errorf("paystack verification error: %s", paystackResp.Message)
    }

    return &PaymentVerification{
        Status:      status,
        Amount:      amount,
        Reference:   reference,
        GatewayData: paystackResp.Data,
    }, nil
}

// ParseWebhook (unchanged, correct)
func (g *PaystackGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
    log.Printf("[Paystack] ParseWebhook: signature present=%v", signature != "")

    if signature == "" {
        return nil, fmt.Errorf("missing signature")
    }
    computed := computeHMAC512(payload, g.secretKey)
    if computed != signature {
        return nil, fmt.Errorf("invalid signature")
    }

    var event map[string]interface{}
    if err := json.Unmarshal(payload, &event); err != nil {
        return nil, err
    }

    eventType, ok := event["event"].(string)
    if !ok {
        return nil, fmt.Errorf("missing event type")
    }
    data, ok := event["data"].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("missing event data")
    }

    we := &WebhookEvent{Type: eventType, RawData: data}

    switch eventType {
    case "charge.success":
        if ref, ok := data["reference"].(string); ok {
            we.Reference = ref
        }
        if amount, ok := data["amount"].(float64); ok {
            we.Amount = decimal.NewFromFloat(amount / 100)
        }
        if currency, ok := data["currency"].(string); ok {
            we.Currency = currency
        }
        we.Type = "payment_success"
    case "charge.failed":
        we.Type = "payment_failed"
    default:
        we.Type = "unhandled"
    }

    return we, nil
}

func (g *PaystackGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
    _, err := g.ParseWebhook(ctx, payload, signature)
    return err
}

func computeHMAC512(payload []byte, secret string) string {
    h := hmac.New(sha512.New, []byte(secret))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}

// Subscription stubs (unchanged)
func (g *PaystackGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
    return nil, fmt.Errorf("subscription creation is not supported for Paystack (use one‑time payments only)")
}

func (g *PaystackGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
    return fmt.Errorf("subscription cancellation is not supported for Paystack")
}

func (g *PaystackGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
    return nil, fmt.Errorf("subscription retrieval is not supported for Paystack")
}



// package payment

// import (
//     "bytes"
//     "context"
//     "crypto/hmac"
//     "crypto/sha512"
//     "encoding/hex"
//     "encoding/json"
//     "fmt"
//     "io"
//     "log"
//     "net/http"
//     "os"
//     "time"

//     "github.com/shopspring/decimal"

//     "cbt-api/internal/models"
// )

// // PaystackGateway handles one‑time payments and webhooks.
// // No subscription plans – all recurring logic is managed in your own database.
// type PaystackGateway struct {
//     secretKey string
//     client    *http.Client
// }

// func NewPaystackGateway() *PaystackGateway {
//     return &PaystackGateway{
//         secretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
//         client:    &http.Client{Timeout: 30 * time.Second},
//     }
// }

// // PaystackResponse represents the standard Paystack API response.
// type PaystackResponse struct {
//     Status  bool                   `json:"status"`
//     Message string                 `json:"message"`
//     Data    map[string]interface{} `json:"data"`
// }

// // CreatePayment initializes a one‑time transaction.
// // Returns a PaymentIntent (mapped to your DB model) with the authorization URL.
// func (g *PaystackGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
//     log.Printf("[Paystack] CreatePayment started: email=%s, amount=%s %s",
//         req.Email, req.Amount.String(), req.Currency)

//     // Validate secret key
//     if g.secretKey == "" {
//         log.Printf("[Paystack] CreatePayment error: PAYSTACK_SECRET_KEY is not set")
//         return nil, fmt.Errorf("paystack secret key missing")
//     }

//     // Build request payload (amount in kobo)
//     amountKobo := int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart())
//     payload := map[string]interface{}{
//         "email":    req.Email,
//         "amount":   amountKobo,
//         "currency": string(req.Currency),
//         "metadata": map[string]interface{}{
//             "school_id": req.SchoolID,
//             "user_id":   req.UserID,
//         },
//     }

//     if req.CallbackURL != "" {
//         payload["callback_url"] = req.CallbackURL
//     }

//     jsonPayload, err := json.Marshal(payload)
//     if err != nil {
//         log.Printf("[Paystack] CreatePayment marshal error: %v", err)
//         return nil, fmt.Errorf("failed to marshal request: %w", err)
//     }
//     log.Printf("[Paystack] Request payload: %s", string(jsonPayload))

//     // Create HTTP request
//     httpReq, err := http.NewRequestWithContext(ctx, "POST",
//         "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(jsonPayload))
//     if err != nil {
//         log.Printf("[Paystack] CreatePayment request creation error: %v", err)
//         return nil, err
//     }
//     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
//     httpReq.Header.Set("Content-Type", "application/json")

//     // Execute request
//     resp, err := g.client.Do(httpReq)
//     if err != nil {
//         log.Printf("[Paystack] CreatePayment HTTP error: %v", err)
//         return nil, fmt.Errorf("paystack request failed: %w", err)
//     }
//     defer resp.Body.Close()

//     // Read response
//     body, err := io.ReadAll(resp.Body)
//     if err != nil {
//         log.Printf("[Paystack] CreatePayment read body error: %v", err)
//         return nil, fmt.Errorf("failed to read response: %w", err)
//     }
//     log.Printf("[Paystack] Response status: %d, body length: %d", resp.StatusCode, len(body))
//     if len(body) > 0 {
//         log.Printf("[Paystack] Response body: %s", string(body))
//     } else {
//         log.Printf("[Paystack] Response body is EMPTY")
//     }

//     // Handle non‑200 HTTP status
//     if resp.StatusCode != http.StatusOK {
//         return nil, fmt.Errorf("paystack returned HTTP %d: %s", resp.StatusCode, string(body))
//     }

//     // Parse JSON
//     var paystackResp PaystackResponse
//     if err := json.Unmarshal(body, &paystackResp); err != nil {
//         log.Printf("[Paystack] CreatePayment parse error: %v", err)
//         return nil, fmt.Errorf("failed to parse response: %w", err)
//     }

//     // Check API error
//     if !paystackResp.Status {
//         errMsg := paystackResp.Message
//         if errMsg == "" {
//             if errData, ok := paystackResp.Data["error"].(string); ok {
//                 errMsg = errData
//             } else if msg, ok := paystackResp.Data["message"].(string); ok {
//                 errMsg = msg
//             } else {
//                 errMsg = "unknown error"
//             }
//         }
//         log.Printf("[Paystack] CreatePayment API error: %s", errMsg)
//         return nil, fmt.Errorf("paystack error: %s", errMsg)
//     }

//     // Extract required fields
//     reference, ok := paystackResp.Data["reference"].(string)
//     if !ok {
//         log.Printf("[Paystack] CreatePayment missing reference in response")
//         return nil, fmt.Errorf("paystack response missing reference")
//     }
//     authURL, ok := paystackResp.Data["authorization_url"].(string)
//     if !ok {
//         log.Printf("[Paystack] CreatePayment missing authorization_url")
//         return nil, fmt.Errorf("paystack response missing authorization_url")
//     }
//     accessCode, ok := paystackResp.Data["access_code"].(string)
//     if !ok {
//         log.Printf("[Paystack] CreatePayment missing access_code")
//         return nil, fmt.Errorf("paystack response missing access_code")
//     }

//     log.Printf("[Paystack] CreatePayment success: reference=%s, auth_url=%s", reference, authURL)

//     // Map to your DB PaymentIntent model
//     return &models.PaymentIntent{
//         ID:               reference,
//         Gateway:          models.GatewayPaystack,
//         AuthorizationURL: authURL,
//         Reference:        reference,
//         AccessCode:       accessCode,
//         Amount:           req.Amount,
//         Currency:         req.Currency,
//         Status:           models.IntentPending,
//         CreatedAt:        time.Now(),
//     }, nil
// }

// // VerifyPayment checks the status of a transaction.
// // Returns PaymentVerification with amount, status, and raw gateway data.
// func (g *PaystackGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
//     log.Printf("[Paystack] VerifyPayment started: reference=%s", reference)

//     if g.secretKey == "" {
//         log.Printf("[Paystack] VerifyPayment error: secret key missing")
//         return nil, fmt.Errorf("paystack secret key missing")
//     }

//     url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)
//     req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
//     if err != nil {
//         log.Printf("[Paystack] VerifyPayment request creation error: %v", err)
//         return nil, err
//     }
//     req.Header.Set("Authorization", "Bearer "+g.secretKey)

//     resp, err := g.client.Do(req)
//     if err != nil {
//         log.Printf("[Paystack] VerifyPayment HTTP error: %v", err)
//         return nil, fmt.Errorf("verification request failed: %w", err)
//     }
//     defer resp.Body.Close()

//     body, err := io.ReadAll(resp.Body)
//     if err != nil {
//         log.Printf("[Paystack] VerifyPayment read body error: %v", err)
//         return nil, fmt.Errorf("failed to read response: %w", err)
//     }
//     log.Printf("[Paystack] Verification response status: %d", resp.StatusCode)

//     var paystackResp PaystackResponse
//     if err := json.Unmarshal(body, &paystackResp); err != nil {
//         log.Printf("[Paystack] VerifyPayment parse error: %v, body: %s", err, string(body))
//         return nil, fmt.Errorf("failed to parse response: %w", err)
//     }

//     status := "pending"
//     amount := decimal.Zero

//     if paystackResp.Status {
//         data := paystackResp.Data
//         if txStatus, ok := data["status"].(string); ok {
//             switch txStatus {
//             case "success":
//                 status = "succeeded"
//             case "failed":
//                 status = "failed"
//             default:
//                 status = "pending"
//             }
//         }
//         // Amount is returned in kobo – convert to NGN
//         if amt, ok := data["amount"].(float64); ok {
//             amount = decimal.NewFromFloat(amt / 100)
//         }
//         log.Printf("[Paystack] VerifyPayment result: status=%s, amount=%s", status, amount.String())
//     } else {
//         log.Printf("[Paystack] VerifyPayment API error: %s", paystackResp.Message)
//         return nil, fmt.Errorf("paystack verification error: %s", paystackResp.Message)
//     }

//     return &PaymentVerification{
//         Status:      status,
//         Amount:      amount,
//         Reference:   reference,
//         GatewayData: paystackResp.Data,
//     }, nil
// }

// // ParseWebhook verifies the Paystack signature and extracts event data.
// // It only processes relevant events: charge.success (payment success).
// // Subscription‑related events (subscription.create, subscription.disable) are ignored
// // because we manage all subscription state locally.
// func (g *PaystackGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
//     log.Printf("[Paystack] ParseWebhook: signature present=%v", signature != "")

//     if signature == "" {
//         log.Printf("[Paystack] ParseWebhook error: missing signature")
//         return nil, fmt.Errorf("missing signature")
//     }
//     // Verify HMAC-SHA512 signature
//     computed := computeHMAC512(payload, g.secretKey)
//     if computed != signature {
//         log.Printf("[Paystack] ParseWebhook error: invalid signature (computed=%s, given=%s)",
//             computed, signature)
//         return nil, fmt.Errorf("invalid signature")
//     }

//     var event map[string]interface{}
//     if err := json.Unmarshal(payload, &event); err != nil {
//         log.Printf("[Paystack] ParseWebhook unmarshal error: %v", err)
//         return nil, err
//     }

//     eventType, ok := event["event"].(string)
//     if !ok {
//         log.Printf("[Paystack] ParseWebhook error: missing event type")
//         return nil, fmt.Errorf("missing event type")
//     }
//     data, ok := event["data"].(map[string]interface{})
//     if !ok {
//         log.Printf("[Paystack] ParseWebhook error: missing event data")
//         return nil, fmt.Errorf("missing event data")
//     }

//     we := &WebhookEvent{Type: eventType, RawData: data}

//     switch eventType {
//     case "charge.success":
//         // Payment succeeded – the most important event for one‑time payments
//         if ref, ok := data["reference"].(string); ok {
//             we.Reference = ref
//         }
//         if amount, ok := data["amount"].(float64); ok {
//             we.Amount = decimal.NewFromFloat(amount / 100)
//         }
//         if currency, ok := data["currency"].(string); ok {
//             we.Currency = currency
//         }
//         we.Type = "payment_success"
//         log.Printf("[Paystack] Webhook: payment_success for reference=%s", we.Reference)

//     case "charge.failed":
//         we.Type = "payment_failed"
//         log.Printf("[Paystack] Webhook: payment_failed for reference=%v", data["reference"])

//     case "subscription.create", "subscription.disable", "subscription.not_renew":
//         // Ignore – we don't use Paystack subscriptions
//         log.Printf("[Paystack] Webhook: ignoring subscription event %s", eventType)
//         we.Type = "ignored"

//     default:
//         log.Printf("[Paystack] Webhook: unhandled event type %s", eventType)
//         we.Type = "unhandled"
//     }

//     return we, nil
// }

// // HandleWebhook is a placeholder for backward compatibility.
// // You can implement your own logic that calls ParseWebhook and then processes the event.
// func (g *PaystackGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
//     log.Printf("[Paystack] HandleWebhook called")
//     event, err := g.ParseWebhook(ctx, payload, signature)
//     if err != nil {
//         return err
//     }
//     // In a real implementation, you would call your subscription service here.
//     log.Printf("[Paystack] Webhook event processed: %+v", event)
//     return nil
// }

// // computeHMAC512 is a helper for Paystack signature verification.
// func computeHMAC512(payload []byte, secret string) string {
//     h := hmac.New(sha512.New, []byte(secret))
//     h.Write(payload)
//     return hex.EncodeToString(h.Sum(nil))
// }


// // -------------------------------------------------------------------
// // Subscription methods (stubs – Paystack subscriptions are not used)
// // -------------------------------------------------------------------

// func (g *PaystackGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
//     log.Printf("[Paystack] CreateSubscription called – not implemented")
//     return nil, fmt.Errorf("subscription creation is not supported for Paystack (use one‑time payments only)")
// }

// func (g *PaystackGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
//     log.Printf("[Paystack] CancelSubscription called – not implemented")
//     return fmt.Errorf("subscription cancellation is not supported for Paystack")
// }

// func (g *PaystackGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
//     log.Printf("[Paystack] GetSubscription called – not implemented")
//     return nil, fmt.Errorf("subscription retrieval is not supported for Paystack")
// }

// // -------------------------------------------------------------------
// // The following methods (CreateSubscription, CancelSubscription, GetSubscription)
// // are removed because we no longer use Paystack plans.
// // If your code expects these methods due to an interface, you can either:
// // - Remove them from the interface, or
// // - Keep them as stubs that return an error: "not implemented (no plans used)".
// // -------------------------------------------------------------------


// // package payment

// // import (
// //     "bytes"
// //     "context"
// //     "crypto/hmac"
// //     "crypto/sha512"
// //     "encoding/hex"
// //     "encoding/json"
// //     "fmt"
// //     "io"
// //     "log"
// //     "net/http"
// //     "os"
// //     "time"

// //     "github.com/shopspring/decimal"

// //     "cbt-api/internal/models"
// // )

// // type PaystackGateway struct {
// //     secretKey string
// //     client    *http.Client
// // }

// // func NewPaystackGateway() *PaystackGateway {
// //     return &PaystackGateway{
// //         secretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
// //         client:    &http.Client{Timeout: 30 * time.Second},
// //     }
// // }

// // type PaystackResponse struct {
// //     Status  bool                   `json:"status"`
// //     Message string                 `json:"message"`
// //     Data    map[string]interface{} `json:"data"`
// // }

// // // CreatePayment initializes a one-time payment transaction
// // func (g *PaystackGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
// //     log.Printf("[Paystack] CreatePayment started for email=%s amount=%s", req.Email, req.Amount.String())

// //     payload := map[string]interface{}{
// //         "email":    req.Email,
// //         "amount":   int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()),
// //         "currency": string(req.Currency),
// //         "metadata": map[string]interface{}{
// //             "school_id": req.SchoolID,
// //             "user_id":   req.UserID,
// //         },
// //     }

// //     if req.CallbackURL != "" {
// //         payload["callback_url"] = req.CallbackURL
// //     }

// //     jsonPayload, err := json.Marshal(payload)
// //     if err != nil {
// //         log.Printf("[Paystack] CreatePayment marshal error: %v", err)
// //         return nil, err
// //     }

// //     httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(jsonPayload))
// //     if err != nil {
// //         log.Printf("[Paystack] CreatePayment request creation error: %v", err)
// //         return nil, err
// //     }
// //     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     httpReq.Header.Set("Content-Type", "application/json")

// //     resp, err := g.client.Do(httpReq)
// //     if err != nil {
// //         log.Printf("[Paystack] CreatePayment HTTP error: %v", err)
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, err := io.ReadAll(resp.Body)
// //     if err != nil {
// //         log.Printf("[Paystack] CreatePayment read body error: %v", err)
// //         return nil, err
// //     }

// //     var paystackResp PaystackResponse
// //     if err := json.Unmarshal(body, &paystackResp); err != nil {
// //         log.Printf("[Paystack] CreatePayment parse error: %v, body: %s", err, string(body))
// //         return nil, err
// //     }

// //     if !paystackResp.Status {
// //         log.Printf("[Paystack] CreatePayment failed: %s", paystackResp.Message)
// //         return nil, fmt.Errorf("paystack error: %s", paystackResp.Message)
// //     }

// //     log.Printf("[Paystack] CreatePayment success, reference=%s", paystackResp.Data["reference"])
// //     return &models.PaymentIntent{
// //         ID:               paystackResp.Data["reference"].(string),
// //         Gateway:          models.GatewayPaystack,
// //         AuthorizationURL: paystackResp.Data["authorization_url"].(string),
// //         Reference:        paystackResp.Data["reference"].(string),
// //         AccessCode:       paystackResp.Data["access_code"].(string),
// //         Amount:           req.Amount,
// //         Currency:         req.Currency,
// //         Status:           models.IntentPending,
// //         CreatedAt:        time.Now(),
// //     }, nil
// // }

// // // VerifyPayment checks the status of a transaction
// // func (g *PaystackGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
// //     log.Printf("[Paystack] VerifyPayment started for reference=%s", reference)

// //     url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)

// //     req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
// //     if err != nil {
// //         log.Printf("[Paystack] VerifyPayment request error: %v", err)
// //         return nil, err
// //     }
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         log.Printf("[Paystack] VerifyPayment HTTP error: %v", err)
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, err := io.ReadAll(resp.Body)
// //     if err != nil {
// //         log.Printf("[Paystack] VerifyPayment read error: %v", err)
// //         return nil, err
// //     }

// //     var paystackResp PaystackResponse
// //     if err := json.Unmarshal(body, &paystackResp); err != nil {
// //         log.Printf("[Paystack] VerifyPayment parse error: %v", err)
// //         return nil, err
// //     }

// //     status := "pending"
// //     if paystackResp.Status {
// //         data := paystackResp.Data
// //         if data["status"] == "success" {
// //             status = "succeeded"
// //         } else if data["status"] == "failed" {
// //             status = "failed"
// //         }
// //     }

// //     amount := decimal.Zero
// //     if amt, ok := paystackResp.Data["amount"].(float64); ok {
// //         amount = decimal.NewFromFloat(amt / 100)
// //     }

// //     log.Printf("[Paystack] VerifyPayment result: status=%s", status)
// //     return &PaymentVerification{
// //         Status:      status,
// //         Amount:      amount,
// //         Reference:   reference,
// //         GatewayData: paystackResp.Data,
// //     }, nil
// // }

// // // CreateSubscription initializes a recurring subscription on Paystack
// // // func (g *PaystackGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
// // //     log.Printf("[Paystack] CreateSubscription started: tier=%s interval=%s email=%s",
// // //         req.Tier, req.Interval, req.Email)

// // //     // 1. Get plan code from environment (must exist)
// // //     planCode := g.getPaystackPlanCode(req.Tier, req.Interval)
// // //     if planCode == "" {
// // //         log.Printf("[Paystack] CreateSubscription error: plan code not configured for %s/%s", req.Tier, req.Interval)
// // //         return nil, fmt.Errorf("paystack plan code not configured for tier=%s interval=%s", req.Tier, req.Interval)
// // //     }
// // //     log.Printf("[Paystack] Using plan code: %s", planCode)

// // //     // 2. Build payload according to Paystack subscription initialization API
// // //     payload := map[string]interface{}{
// // //         "email":    req.Email,
// // //         "plan":     planCode,
// // //         "callback_url": req.SuccessURL, // mandatory for redirect
// // //         "metadata": map[string]interface{}{
// // //             "school_id": req.SchoolID,
// // //             "user_id":   req.UserID,
// // //             "tier":      string(req.Tier),
// // //             "interval":  string(req.Interval),
// // //         },
// // //     }

// // //     if req.CustomerName != "" {
// // //         payload["name"] = req.CustomerName
// // //     }

// // //     jsonPayload, err := json.Marshal(payload)
// // //     if err != nil {
// // //         log.Printf("[Paystack] CreateSubscription marshal error: %v", err)
// // //         return nil, fmt.Errorf("failed to marshal request: %w", err)
// // //     }

// // //     // 3. Make API request
// // //     httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/subscription/initialize", bytes.NewBuffer(jsonPayload))
// // //     if err != nil {
// // //         log.Printf("[Paystack] CreateSubscription request creation error: %v", err)
// // //         return nil, err
// // //     }
// // //     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
// // //     httpReq.Header.Set("Content-Type", "application/json")

// // //     resp, err := g.client.Do(httpReq)
// // //     if err != nil {
// // //         log.Printf("[Paystack] CreateSubscription HTTP error: %v", err)
// // //         return nil, fmt.Errorf("paystack request failed: %w", err)
// // //     }
// // //     defer resp.Body.Close()

// // //     body, err := io.ReadAll(resp.Body)
// // //     if err != nil {
// // //         log.Printf("[Paystack] CreateSubscription read body error: %v", err)
// // //         return nil, fmt.Errorf("failed to read paystack response: %w", err)
// // //     }

// // //     // 4. Parse response
// // //     var paystackResp PaystackResponse
// // //     if err := json.Unmarshal(body, &paystackResp); err != nil {
// // //         log.Printf("[Paystack] CreateSubscription parse error: %v, body: %s", err, string(body))
// // //         return nil, fmt.Errorf("failed to parse paystack response: %w", err)
// // //     }

// // //     // 5. Handle failure with detailed error extraction
// // //     if !paystackResp.Status {
// // //         errMsg := paystackResp.Message
// // //         if errMsg == "" {
// // //             // Try to extract from data.error or data.message
// // //             if errData, ok := paystackResp.Data["error"].(string); ok {
// // //                 errMsg = errData
// // //             } else if msg, ok := paystackResp.Data["message"].(string); ok {
// // //                 errMsg = msg
// // //             } else {
// // //                 errMsg = "unknown error"
// // //             }
// // //         }
// // //         log.Printf("[Paystack] CreateSubscription API error: %s", errMsg)
// // //         return nil, fmt.Errorf("paystack subscription error: %s", errMsg)
// // //     }

// // //     // 6. Extract subscription data
// // //     subscriptionCode, ok := paystackResp.Data["subscription_code"].(string)
// // //     if !ok {
// // //         log.Printf("[Paystack] CreateSubscription missing subscription_code in response")
// // //         return nil, fmt.Errorf("paystack response missing subscription_code")
// // //     }
// // //     authorizationURL, ok := paystackResp.Data["authorization_url"].(string)
// // //     if !ok {
// // //         log.Printf("[Paystack] CreateSubscription missing authorization_url")
// // //         return nil, fmt.Errorf("paystack response missing authorization_url")
// // //     }

// // //     // Customer code or ID (optional)
// // //     var customerID string
// // //     if customer, ok := paystackResp.Data["customer"].(map[string]interface{}); ok {
// // //         if code, ok := customer["customer_code"].(string); ok {
// // //             customerID = code
// // //         }
// // //     }

// // //     log.Printf("[Paystack] CreateSubscription success: subscription_code=%s, auth_url=%s",
// // //         subscriptionCode, authorizationURL)

// // //     return &SubscriptionResult{
// // //         SubscriptionID:        subscriptionCode,
// // //         AuthorizationURL:      authorizationURL,
// // //         GatewayCustomerID:     customerID,
// // //         GatewaySubscriptionID: subscriptionCode,
// // //         Status:                "pending",
// // //     }, nil
// // // }

// // func (g *PaystackGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
// //     log.Printf("[Paystack] CreateSubscription started: tier=%s interval=%s email=%s", req.Tier, req.Interval, req.Email)

// //     // 1. Validate secret key
// //     if g.secretKey == "" {
// //         log.Printf("[Paystack] CreateSubscription error: PAYSTACK_SECRET_KEY is not set")
// //         return nil, fmt.Errorf("paystack secret key is missing")
// //     }

// //     // 2. Get plan code
// //     planCode := g.getPaystackPlanCode(req.Tier, req.Interval)
// //     if planCode == "" {
// //         log.Printf("[Paystack] CreateSubscription error: plan code not configured for %s/%s", req.Tier, req.Interval)
// //         return nil, fmt.Errorf("paystack plan code not configured for tier=%s interval=%s", req.Tier, req.Interval)
// //     }
// //     log.Printf("[Paystack] Using plan code: %s", planCode)

// //     // 3. Build payload (ensure callback_url is not empty – if empty, use a default or skip)
// //     if req.SuccessURL == "" {
// //         log.Printf("[Paystack] Warning: SuccessURL is empty, Paystack may reject the request")
// //     }
// //     payload := map[string]interface{}{
// //         "email":    req.Email,
// //         "plan":     planCode,
// //         "metadata": map[string]interface{}{
// //             "school_id": req.SchoolID,
// //             "user_id":   req.UserID,
// //             "tier":      string(req.Tier),
// //             "interval":  string(req.Interval),
// //         },
// //     }
// //     if req.SuccessURL != "" {
// //         payload["callback_url"] = req.SuccessURL
// //     }
// //     if req.CustomerName != "" {
// //         payload["name"] = req.CustomerName
// //     }

// //     jsonPayload, err := json.Marshal(payload)
// //     if err != nil {
// //         log.Printf("[Paystack] CreateSubscription marshal error: %v", err)
// //         return nil, fmt.Errorf("failed to marshal request: %w", err)
// //     }
// //     log.Printf("[Paystack] Request payload: %s", string(jsonPayload))

// //     // 4. Create HTTP request
// //     httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/subscription/initialize", bytes.NewBuffer(jsonPayload))
// //     if err != nil {
// //         log.Printf("[Paystack] CreateSubscription request creation error: %v", err)
// //         return nil, err
// //     }
// //     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     httpReq.Header.Set("Content-Type", "application/json")

// //     // 5. Execute request
// //     resp, err := g.client.Do(httpReq)
// //     if err != nil {
// //         log.Printf("[Paystack] CreateSubscription HTTP error: %v", err)
// //         return nil, fmt.Errorf("paystack request failed: %w", err)
// //     }
// //     defer resp.Body.Close()

// //     // 6. Read response body and log status
// //     body, err := io.ReadAll(resp.Body)
// //     if err != nil {
// //         log.Printf("[Paystack] CreateSubscription read body error: %v", err)
// //         return nil, fmt.Errorf("failed to read paystack response: %w", err)
// //     }
// //     log.Printf("[Paystack] Response status: %d, body length: %d", resp.StatusCode, len(body))
// //     if len(body) > 0 {
// //         log.Printf("[Paystack] Response body: %s", string(body))
// //     } else {
// //         log.Printf("[Paystack] Response body is EMPTY")
// //     }

// //     // 7. Handle non-200 HTTP status
// //     if resp.StatusCode != http.StatusOK {
// //         return nil, fmt.Errorf("paystack returned HTTP %d: %s", resp.StatusCode, string(body))
// //     }

// //     // 8. Parse JSON response
// //     var paystackResp PaystackResponse
// //     if err := json.Unmarshal(body, &paystackResp); err != nil {
// //         log.Printf("[Paystack] CreateSubscription parse error: %v", err)
// //         return nil, fmt.Errorf("failed to parse paystack response: %w", err)
// //     }

// //     // 9. Check for API error
// //     if !paystackResp.Status {
// //         errMsg := paystackResp.Message
// //         if errMsg == "" {
// //             if errData, ok := paystackResp.Data["error"].(string); ok {
// //                 errMsg = errData
// //             } else if msg, ok := paystackResp.Data["message"].(string); ok {
// //                 errMsg = msg
// //             } else {
// //                 errMsg = "unknown error"
// //             }
// //         }
// //         log.Printf("[Paystack] CreateSubscription API error: %s", errMsg)
// //         return nil, fmt.Errorf("paystack subscription error: %s", errMsg)
// //     }

// //     // 10. Extract subscription data
// //     subscriptionCode, ok := paystackResp.Data["subscription_code"].(string)
// //     if !ok {
// //         log.Printf("[Paystack] CreateSubscription missing subscription_code in response")
// //         return nil, fmt.Errorf("paystack response missing subscription_code")
// //     }
// //     authorizationURL, ok := paystackResp.Data["authorization_url"].(string)
// //     if !ok {
// //         log.Printf("[Paystack] CreateSubscription missing authorization_url")
// //         return nil, fmt.Errorf("paystack response missing authorization_url")
// //     }

// //     var customerID string
// //     if customer, ok := paystackResp.Data["customer"].(map[string]interface{}); ok {
// //         if code, ok := customer["customer_code"].(string); ok {
// //             customerID = code
// //         }
// //     }

// //     log.Printf("[Paystack] CreateSubscription success: subscription_code=%s, auth_url=%s", subscriptionCode, authorizationURL)
// //     return &SubscriptionResult{
// //         SubscriptionID:        subscriptionCode,
// //         AuthorizationURL:      authorizationURL,
// //         GatewayCustomerID:     customerID,
// //         GatewaySubscriptionID: subscriptionCode,
// //         Status:                "pending",
// //     }, nil
// // }

// // // CancelSubscription disables an active subscription on Paystack
// // func (g *PaystackGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
// //     log.Printf("[Paystack] CancelSubscription for subscription_id=%s", subscriptionID)

// //     payload := map[string]interface{}{
// //         "code":  subscriptionID,
// //         "token": g.secretKey,
// //     }

// //     jsonPayload, err := json.Marshal(payload)
// //     if err != nil {
// //         log.Printf("[Paystack] CancelSubscription marshal error: %v", err)
// //         return err
// //     }

// //     req, err := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/subscription/disable", bytes.NewBuffer(jsonPayload))
// //     if err != nil {
// //         log.Printf("[Paystack] CancelSubscription request error: %v", err)
// //         return err
// //     }
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     req.Header.Set("Content-Type", "application/json")

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         log.Printf("[Paystack] CancelSubscription HTTP error: %v", err)
// //         return err
// //     }
// //     defer resp.Body.Close()

// //     body, _ := io.ReadAll(resp.Body)
// //     var paystackResp PaystackResponse
// //     json.Unmarshal(body, &paystackResp)

// //     if !paystackResp.Status {
// //         log.Printf("[Paystack] CancelSubscription failed: %s", paystackResp.Message)
// //         return fmt.Errorf("paystack cancel error: %s", paystackResp.Message)
// //     }

// //     log.Printf("[Paystack] CancelSubscription successful for %s", subscriptionID)
// //     return nil
// // }

// // // GetSubscription retrieves subscription status from Paystack
// // func (g *PaystackGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
// //     log.Printf("[Paystack] GetSubscription for %s", subscriptionID)

// //     url := fmt.Sprintf("https://api.paystack.co/subscription/%s", subscriptionID)

// //     req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
// //     if err != nil {
// //         log.Printf("[Paystack] GetSubscription request error: %v", err)
// //         return nil, err
// //     }
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         log.Printf("[Paystack] GetSubscription HTTP error: %v", err)
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, err := io.ReadAll(resp.Body)
// //     if err != nil {
// //         log.Printf("[Paystack] GetSubscription read error: %v", err)
// //         return nil, err
// //     }

// //     var paystackResp PaystackResponse
// //     if err := json.Unmarshal(body, &paystackResp); err != nil {
// //         log.Printf("[Paystack] GetSubscription parse error: %v", err)
// //         return nil, err
// //     }

// //     active := false
// //     status := "inactive"
// //     if paystackResp.Status {
// //         data := paystackResp.Data
// //         if s, ok := data["status"].(string); ok && s == "active" {
// //             active = true
// //             status = "active"
// //         }
// //     }

// //     log.Printf("[Paystack] GetSubscription result: active=%v status=%s", active, status)
// //     return &SubscriptionStatus{
// //         Active: active,
// //         Status: status,
// //     }, nil
// // }

// // // ParseWebhook verifies signature and extracts webhook data
// // func (g *PaystackGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
// //     log.Printf("[Paystack] ParseWebhook: signature present=%v", signature != "")

// //     if signature == "" {
// //         log.Printf("[Paystack] ParseWebhook error: missing signature")
// //         return nil, fmt.Errorf("missing signature")
// //     }

// //     computed := computeHMAC512(payload, g.secretKey)
// //     if computed != signature {
// //         log.Printf("[Paystack] ParseWebhook error: invalid signature (computed=%s, given=%s)", computed, signature)
// //         return nil, fmt.Errorf("invalid signature")
// //     }

// //     var event map[string]interface{}
// //     if err := json.Unmarshal(payload, &event); err != nil {
// //         log.Printf("[Paystack] ParseWebhook unmarshal error: %v", err)
// //         return nil, err
// //     }

// //     eventType, ok := event["event"].(string)
// //     if !ok {
// //         log.Printf("[Paystack] ParseWebhook error: missing event type")
// //         return nil, fmt.Errorf("missing event type")
// //     }
// //     data, ok := event["data"].(map[string]interface{})
// //     if !ok {
// //         log.Printf("[Paystack] ParseWebhook error: missing event data")
// //         return nil, fmt.Errorf("missing event data")
// //     }

// //     we := &WebhookEvent{Type: eventType, RawData: data}

// //     switch eventType {
// //     case "subscription.create":
// //         if subCode, ok := data["subscription_code"].(string); ok {
// //             we.GatewaySubscriptionID = subCode
// //             we.Type = "subscription_activated"
// //             log.Printf("[Paystack] Webhook: subscription_activated for %s", subCode)
// //         }
// //     case "charge.success":
// //         if ref, ok := data["reference"].(string); ok {
// //             we.Reference = ref
// //         }
// //         if amount, ok := data["amount"].(float64); ok {
// //             we.Amount = decimal.NewFromFloat(amount / 100)
// //         }
// //         if currency, ok := data["currency"].(string); ok {
// //             we.Currency = currency
// //         }
// //         we.Type = "payment_success"
// //         log.Printf("[Paystack] Webhook: payment_success for reference=%s", we.Reference)
// //     case "subscription.disable":
// //         if subCode, ok := data["subscription_code"].(string); ok {
// //             we.GatewaySubscriptionID = subCode
// //             we.Type = "subscription_cancelled"
// //             log.Printf("[Paystack] Webhook: subscription_cancelled for %s", subCode)
// //         }
// //     default:
// //         log.Printf("[Paystack] Webhook: unhandled event type %s", eventType)
// //     }

// //     return we, nil
// // }

// // // HandleWebhook is a placeholder for processing webhooks (can be implemented later)
// // func (g *PaystackGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
// //     log.Printf("[Paystack] HandleWebhook called")
// //     return nil
// // }

// // // getPaystackPlanCode returns the plan code from environment variables
// // func (g *PaystackGateway) getPaystackPlanCode(tier models.SubscriptionTier, interval models.PaymentInterval) string {
// //     var envKey string
// //     switch tier {
// //     case models.TierBasic:
// //         switch interval {
// //         case models.IntervalMonthly:
// //             envKey = "PAYSTACK_PLAN_BASIC_MONTHLY"
// //         case models.IntervalQuarterly:
// //             envKey = "PAYSTACK_PLAN_BASIC_QUARTERLY"
// //         case models.IntervalYearly:
// //             envKey = "PAYSTACK_PLAN_BASIC_YEARLY"
// //         }
// //     case models.TierPremium:
// //         switch interval {
// //         case models.IntervalMonthly:
// //             envKey = "PAYSTACK_PLAN_PREMIUM_MONTHLY"
// //         case models.IntervalQuarterly:
// //             envKey = "PAYSTACK_PLAN_PREMIUM_QUARTERLY"
// //         case models.IntervalYearly:
// //             envKey = "PAYSTACK_PLAN_PREMIUM_YEARLY"
// //         }
// //     case models.TierEnterprise:
// //         switch interval {
// //         case models.IntervalMonthly:
// //             envKey = "PAYSTACK_PLAN_ENTERPRISE_MONTHLY"
// //         case models.IntervalQuarterly:
// //             envKey = "PAYSTACK_PLAN_ENTERPRISE_QUARTERLY"
// //         case models.IntervalYearly:
// //             envKey = "PAYSTACK_PLAN_ENTERPRISE_YEARLY"
// //         }
// //     }

// //     if envKey == "" {
// //         return ""
// //     }
// //     return os.Getenv(envKey)
// // }

// // // computeHMAC512 is a helper for webhook signature verification
// // func computeHMAC512(payload []byte, secret string) string {
// //     h := hmac.New(sha512.New, []byte(secret))
// //     h.Write(payload)
// //     return hex.EncodeToString(h.Sum(nil))
// // }


// // package payment

// // import (
// //     "bytes"
// //     "context"
// //     "crypto/hmac"
// //     "crypto/sha512"
// //     "encoding/hex"
// //     "encoding/json"
// //     "fmt"
// //     "io"
// //     "net/http"
// //     "os"
// //     "time"


// //     "github.com/shopspring/decimal"

// //     "cbt-api/internal/models"
// // )

// // type PaystackGateway struct {
// //     secretKey string
// //     client    *http.Client
// // }

// // func NewPaystackGateway() *PaystackGateway {
// //     return &PaystackGateway{
// //         secretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
// //         client:    &http.Client{Timeout: 30 * time.Second},
// //     }
// // }

// // type PaystackResponse struct {
// //     Status  bool                   `json:"status"`
// //     Message string                 `json:"message"`
// //     Data    map[string]interface{} `json:"data"`
// // }

// // func (g *PaystackGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
// //     payload := map[string]interface{}{
// //         "email":    req.Email,
// //         "amount":   int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()),
// //         "currency": string(req.Currency),
// //         "metadata": map[string]interface{}{
// //             "school_id": req.SchoolID,
// //             "user_id":   req.UserID,
// //         },
// //     }

// //     if req.CallbackURL != "" {
// //         payload["callback_url"] = req.CallbackURL
// //     }

// //     jsonPayload, _ := json.Marshal(payload)

// //     httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(jsonPayload))
// //     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     httpReq.Header.Set("Content-Type", "application/json")

// //     resp, err := g.client.Do(httpReq)
// //     if err != nil {
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, _ := io.ReadAll(resp.Body)

// //     var paystackResp PaystackResponse
// //     json.Unmarshal(body, &paystackResp)

// //     if !paystackResp.Status {
// //         return nil, fmt.Errorf("paystack error: %s", paystackResp.Message)
// //     }

// //     return &models.PaymentIntent{
// //         ID:               paystackResp.Data["reference"].(string),
// //         Gateway:          models.GatewayPaystack,
// //         AuthorizationURL: paystackResp.Data["authorization_url"].(string),
// //         Reference:        paystackResp.Data["reference"].(string),
// //         AccessCode:       paystackResp.Data["access_code"].(string),
// //         Amount:           req.Amount,
// //         Currency:         req.Currency,
// //         // Status:           string(models.IntentPending),
// //         Status:           models.IntentPending,

// //         CreatedAt:        time.Now(),
// //     }, nil
// // }

// // func (g *PaystackGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
// //     url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)

// //     req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, _ := io.ReadAll(resp.Body)

// //     var paystackResp PaystackResponse
// //     json.Unmarshal(body, &paystackResp)

// //     status := "pending"
// //     if paystackResp.Status {
// //         data := paystackResp.Data
// //         if data["status"] == "success" {
// //             status = "succeeded"
// //         } else if data["status"] == "failed" {
// //             status = "failed"
// //         }
// //     }

// //     return &PaymentVerification{
// //         Status:    status,
// //         Amount:    decimal.NewFromFloat(paystackResp.Data["amount"].(float64) / 100),
// //         Reference: reference,
// //         GatewayData: paystackResp.Data,
// //     }, nil
// // }

// // func (g *PaystackGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
// //     planCode := getPaystackPlanCode(req.Tier, req.Interval)

// //     payload := map[string]interface{}{
// //         "email":    req.Email,
// //         "plan":     planCode,
// //         "metadata": map[string]string{
// //             "school_id": req.SchoolID,
// //             "user_id":   req.UserID,
// //             "tier":      string(req.Tier),
// //         },
// //     }

// //     jsonPayload, _ := json.Marshal(payload)

// //     httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/subscription/initialize", bytes.NewBuffer(jsonPayload))
// //     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     httpReq.Header.Set("Content-Type", "application/json")

// //     resp, err := g.client.Do(httpReq)
// //     if err != nil {
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, _ := io.ReadAll(resp.Body)

// //     var paystackResp PaystackResponse
// //     json.Unmarshal(body, &paystackResp)

// //     if !paystackResp.Status {
// //         return nil, fmt.Errorf("paystack subscription error: %s", paystackResp.Message)
// //     }

// //     return &SubscriptionResult{
// //         SubscriptionID:        paystackResp.Data["subscription_code"].(string),
// //         AuthorizationURL:     paystackResp.Data["authorization_url"].(string),
// //         GatewayCustomerID:     fmt.Sprintf("%v", paystackResp.Data["customer"]),
// //         GatewaySubscriptionID: paystackResp.Data["subscription_code"].(string),
// //         Status:                "pending",
// //     }, nil
// // }

// // func (g *PaystackGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
// //     payload := map[string]interface{}{
// //         "code":  subscriptionID,
// //         "token": g.secretKey,
// //     }

// //     jsonPayload, _ := json.Marshal(payload)

// //     req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/subscription/disable", bytes.NewBuffer(jsonPayload))
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)
// //     req.Header.Set("Content-Type", "application/json")

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         return err
// //     }
// //     defer resp.Body.Close()

// //     return nil
// // }

// // func (g *PaystackGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
// //     url := fmt.Sprintf("https://api.paystack.co/subscription/%s", subscriptionID)

// //     req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
// //     req.Header.Set("Authorization", "Bearer "+g.secretKey)

// //     resp, err := g.client.Do(req)
// //     if err != nil {
// //         return nil, err
// //     }
// //     defer resp.Body.Close()

// //     body, _ := io.ReadAll(resp.Body)

// //     var paystackResp PaystackResponse
// //     json.Unmarshal(body, &paystackResp)

// //     active := false
// //     status := "inactive"
// //     if paystackResp.Status {
// //         data := paystackResp.Data
// //         if data["status"] == "active" {
// //             active = true
// //             status = "active"
// //         }
// //     }

// //     return &SubscriptionStatus{
// //         Active: active,
// //         Status: status,
// //     }, nil
// // }


// // // ParseWebhook verifies signature and extracts webhook data
// // func (g *PaystackGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
// //     // Verify signature (Paystack sends a x-paystack-signature header)
// //     if signature == "" {
// //         return nil, fmt.Errorf("missing signature")
// //     }
// //     // Compute HMAC-SHA512 of payload using secret key
// //     computed := computeHMAC512(payload, g.secretKey)
// //     if computed != signature {
// //         return nil, fmt.Errorf("invalid signature")
// //     }

// //     var event map[string]interface{}
// //     if err := json.Unmarshal(payload, &event); err != nil {
// //         return nil, err
// //     }

// //     eventType, ok := event["event"].(string)
// //     if !ok {
// //         return nil, fmt.Errorf("missing event type")
// //     }
// //     data, ok := event["data"].(map[string]interface{})
// //     if !ok {
// //         return nil, fmt.Errorf("missing event data")
// //     }

// //     we := &WebhookEvent{Type: eventType, RawData: data}

// //     switch eventType {
// //     case "subscription.create":
// //         if subCode, ok := data["subscription_code"].(string); ok {
// //             we.GatewaySubscriptionID = subCode
// //             we.Type = "subscription_activated"
// //         }
// //     case "charge.success":
// //         if ref, ok := data["reference"].(string); ok {
// //             we.Reference = ref
// //         }
// //         if amount, ok := data["amount"].(float64); ok {
// //             we.Amount = decimal.NewFromFloat(amount / 100)
// //         }
// //         if currency, ok := data["currency"].(string); ok {
// //             we.Currency = currency
// //         }
// //         // If this is a subscription payment, we may need to find the subscription from metadata
// //         // For simplicity, we can also look up by reference.
// //         we.Type = "payment_success"
// //     case "subscription.disable":
// //         if subCode, ok := data["subscription_code"].(string); ok {
// //             we.GatewaySubscriptionID = subCode
// //             we.Type = "subscription_cancelled"
// //         }
// //     }

// //     return we, nil
// // }

// // // helper for HMAC‑SHA512
// // func computeHMAC512(payload []byte, secret string) string {
// //     h := hmac.New(sha512.New, []byte(secret))
// //     h.Write(payload)
// //     return hex.EncodeToString(h.Sum(nil))
// // }


// // func (g *PaystackGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
// //     // Webhook handling will be implemented
// //     return nil
// // }

// // func getPaystackPlanCode(tier models.SubscriptionTier, interval models.PaymentInterval) string {
// //     plans := map[models.SubscriptionTier]map[models.PaymentInterval]string{
// //         models.TierBasic: {
// //             models.IntervalMonthly: os.Getenv("PAYSTACK_PLAN_BASIC_MONTHLY"),
// //             models.IntervalYearly:  os.Getenv("PAYSTACK_PLAN_BASIC_YEARLY"),
// //         },
// //         models.TierPremium: {
// //             models.IntervalMonthly: os.Getenv("PAYSTACK_PLAN_PREMIUM_MONTHLY"),
// //             models.IntervalYearly:  os.Getenv("PAYSTACK_PLAN_PREMIUM_YEARLY"),
// //         },
// //         models.TierEnterprise: {
// //             models.IntervalMonthly: os.Getenv("PAYSTACK_PLAN_ENTERPRISE_MONTHLY"),
// //             models.IntervalYearly:  os.Getenv("PAYSTACK_PLAN_ENTERPRISE_YEARLY"),
// //         },
// //     }
    
// //     if plan, ok := plans[tier][interval]; ok && plan != "" {
// //         return plan
// //     }
// //     return "PLN_test_123"
// // }