package payment

import (
    "log" 
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/shopspring/decimal"

    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type FlutterwaveGateway struct {
    secretKey string
    client    *http.Client
}

func NewFlutterwaveGateway() *FlutterwaveGateway {
    return &FlutterwaveGateway{
        secretKey: os.Getenv("FLUTTERWAVE_SECRET_KEY"),
        client:    &http.Client{Timeout: 30 * time.Second},
    }
}

type FlutterwaveResponse struct {
    Status  string                 `json:"status"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data"`
}

// func (g *FlutterwaveGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
//     // ... (your existing implementation, unchanged)
//     reference := fmt.Sprintf("REF_%d_%s", time.Now().UnixNano(), req.SchoolID)

//     payload := map[string]interface{}{
//         "tx_ref":       reference,
//         "amount":       req.Amount.InexactFloat64(),
//         "currency":     string(req.Currency),
//         "redirect_url": req.CallbackURL,
//         "customer": map[string]string{
//             "email": req.Email,
//         },
//         "meta": map[string]interface{}{
//             "school_id": req.SchoolID,
//             "user_id":   req.UserID,
//         },
//     }

//     jsonPayload, _ := json.Marshal(payload)

//     httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.flutterwave.com/v3/payments", bytes.NewBuffer(jsonPayload))
//     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
//     httpReq.Header.Set("Content-Type", "application/json")

//     resp, err := g.client.Do(httpReq)
//     if err != nil {
//         return nil, err
//     }
//     defer resp.Body.Close()

//     body, _ := io.ReadAll(resp.Body)

//     var fwResp FlutterwaveResponse
//     json.Unmarshal(body, &fwResp)

//     if fwResp.Status != "success" {
//         return nil, fmt.Errorf("flutterwave error: %s", fwResp.Message)
//     }

//     return &models.PaymentIntent{
//         ID:               fwResp.Data["id"].(string),
//         Gateway:          models.GatewayFlutterwave,
//         AuthorizationURL: fwResp.Data["link"].(string),
//         Reference:        reference,
//         Amount:           req.Amount,
//         Currency:         req.Currency,
//         Status:           models.IntentPending,
//         CreatedAt:        time.Now(),
//     }, nil
// }

// func (g *FlutterwaveGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
//     log.Printf("[Flutterwave] CreatePayment started: email=%s, amount=%s %s, callback=%s",
//         req.Email, req.Amount.String(), req.Currency, req.CallbackURL)

//     if g.secretKey == "" {
//         return nil, fmt.Errorf("FLUTTERWAVE_SECRET_KEY is not set")
//     }

//     reference := fmt.Sprintf("REF_%d_%s", time.Now().UnixNano(), req.SchoolID)

//     // Ensure required fields are present
//     if req.Email == "" {
//         return nil, fmt.Errorf("customer email is required")
//     }
//     if req.CallbackURL == "" {
//         log.Printf("[Flutterwave] Warning: CallbackURL is empty, Flutterwave may reject the request")
//         // You can set a default here if desired
//         // req.CallbackURL = "https://yourapp.com/payment/verify"
//     }

//     payload := map[string]interface{}{
//         "tx_ref":       reference,
//         "amount":       req.Amount.InexactFloat64(),
//         "currency":     string(req.Currency),
//         "redirect_url": req.CallbackURL,
//         "customer": map[string]string{
//             "email": req.Email,
//         },
//         "meta": map[string]interface{}{
//             "school_id": req.SchoolID,
//             "user_id":   req.UserID,
//         },
//     }

//     jsonPayload, err := json.Marshal(payload)
//     if err != nil {
//         return nil, fmt.Errorf("failed to marshal request: %w", err)
//     }
//     log.Printf("[Flutterwave] Request payload: %s", string(jsonPayload))

//     httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.flutterwave.com/v3/payments", bytes.NewBuffer(jsonPayload))
//     if err != nil {
//         return nil, err
//     }
//     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
//     httpReq.Header.Set("Content-Type", "application/json")

//     resp, err := g.client.Do(httpReq)
//     if err != nil {
//         return nil, fmt.Errorf("HTTP request failed: %w", err)
//     }
//     defer resp.Body.Close()

//     body, err := io.ReadAll(resp.Body)
//     if err != nil {
//         return nil, fmt.Errorf("failed to read response: %w", err)
//     }

//     var fwResp FlutterwaveResponse
//     if err := json.Unmarshal(body, &fwResp); err != nil {
//         return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
//     }

//     if fwResp.Status != "success" {
//         // Enhanced error message
//         errMsg := fwResp.Message
//         if errMsg == "" {
//             if dataMsg, ok := fwResp.Data["message"].(string); ok {
//                 errMsg = dataMsg
//             } else {
//                 errMsg = "unknown error"
//             }
//         }
//         // Log full response for debugging
//         log.Printf("[Flutterwave] API error: status=%s, message=%s, full response=%s", fwResp.Status, errMsg, string(body))
//         return nil, fmt.Errorf("flutterwave error: %s", errMsg)
//     }

//     // Extract data
//     id, ok := fwResp.Data["id"].(string)
//     if !ok {
//         // Might be float64
//         if idFloat, ok := fwResp.Data["id"].(float64); ok {
//             id = fmt.Sprintf("%.0f", idFloat)
//         } else {
//             return nil, fmt.Errorf("missing or invalid id in response")
//         }
//     }
//     link, ok := fwResp.Data["link"].(string)
//     if !ok {
//         return nil, fmt.Errorf("missing authorization link in response")
//     }

//     log.Printf("[Flutterwave] Payment created: id=%s, link=%s", id, link)

//     return &models.PaymentIntent{
//         ID:               id,
//         Gateway:          models.GatewayFlutterwave,
//         AuthorizationURL: link,
//         Reference:        reference,
//         Amount:           req.Amount,
//         Currency:         req.Currency,
//         Status:           models.IntentPending,
//         CreatedAt:        time.Now(),
//     }, nil
// }

// func (g *FlutterwaveGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
//     log.Printf("[Flutterwave] CreatePayment started: email=%s, amount=%s %s, callback=%s",
//         req.Email, req.Amount.String(), req.Currency, req.CallbackURL)

//     if g.secretKey == "" {
//         return nil, fmt.Errorf("FLUTTERWAVE_SECRET_KEY is not set")
//     }

//     // Generate unique transaction reference
//     reference := fmt.Sprintf("REF_%d_%s", time.Now().UnixNano(), req.SchoolID)

//     // Validate required fields
//     if req.Email == "" {
//         return nil, fmt.Errorf("customer email is required")
//     }
//     if req.CallbackURL == "" {
//         log.Printf("[Flutterwave] Warning: CallbackURL is empty; setting default")
//         req.CallbackURL = "https://yourapp.com/payment/verify"
//     }

//     payload := map[string]interface{}{
//         "tx_ref":       reference,
//         "amount":       req.Amount.InexactFloat64(),
//         "currency":     string(req.Currency),
//         "redirect_url": req.CallbackURL,
//         "customer": map[string]string{
//             "email": req.Email,
//         },
//         "meta": map[string]interface{}{
//             "school_id": req.SchoolID,
//             "user_id":   req.UserID,
//         },
//     }

//     jsonPayload, err := json.Marshal(payload)
//     if err != nil {
//         return nil, fmt.Errorf("failed to marshal request: %w", err)
//     }
//     log.Printf("[Flutterwave] Request payload: %s", string(jsonPayload))

//     httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.flutterwave.com/v3/payments", bytes.NewBuffer(jsonPayload))
//     if err != nil {
//         return nil, err
//     }
//     httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
//     httpReq.Header.Set("Content-Type", "application/json")

//     resp, err := g.client.Do(httpReq)
//     if err != nil {
//         return nil, fmt.Errorf("HTTP request failed: %w", err)
//     }
//     defer resp.Body.Close()

//     body, err := io.ReadAll(resp.Body)
//     if err != nil {
//         return nil, fmt.Errorf("failed to read response: %w", err)
//     }
//     log.Printf("[Flutterwave] Raw response body: %s", string(body))

//     var fwResp FlutterwaveResponse
//     if err := json.Unmarshal(body, &fwResp); err != nil {
//         return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
//     }

//     // Check API status
//     if fwResp.Status != "success" {
//         errMsg := fwResp.Message
//         if errMsg == "" {
//             if dataMsg, ok := fwResp.Data["message"].(string); ok {
//                 errMsg = dataMsg
//             } else {
//                 errMsg = "unknown error"
//             }
//         }
//         log.Printf("[Flutterwave] API error: status=%s, message=%s", fwResp.Status, errMsg)
//         return nil, fmt.Errorf("flutterwave error: %s", errMsg)
//     }

//     // Extract data – safely handle ID as number or string
//     if fwResp.Data == nil {
//         return nil, fmt.Errorf("missing data in response")
//     }

//     var id string
//     switch v := fwResp.Data["id"].(type) {
//     case string:
//         id = v
//     case float64:
//         id = fmt.Sprintf("%.0f", v)
//     case int:
//         id = fmt.Sprintf("%d", v)
//     case int64:
//         id = fmt.Sprintf("%d", v)
//     default:
//         log.Printf("[Flutterwave] Unexpected type for id: %T, value: %v", fwResp.Data["id"], fwResp.Data["id"])
//         return nil, fmt.Errorf("missing or invalid id in response")
//     }

//     // Extract authorization link
//     link, ok := fwResp.Data["link"].(string)
//     if !ok {
//         log.Printf("[Flutterwave] Missing link in response data: %+v", fwResp.Data)
//         return nil, fmt.Errorf("missing authorization link in response")
//     }

//     log.Printf("[Flutterwave] Payment created: id=%s, link=%s", id, link)

//     return &models.PaymentIntent{
//         ID:               id,
//         Gateway:          models.GatewayFlutterwave,
//         AuthorizationURL: link,
//         Reference:        reference,
//         Amount:           req.Amount,
//         Currency:         req.Currency,
//         Status:           models.IntentPending,
//         CreatedAt:        time.Now(),
//     }, nil
// }


func (g *FlutterwaveGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*models.PaymentIntent, error) {
    log.Printf("[Flutterwave] CreatePayment started: email=%s, amount=%s %s, callback=%s",
        req.Email, req.Amount.String(), req.Currency, req.CallbackURL)

    if g.secretKey == "" {
        return nil, fmt.Errorf("FLUTTERWAVE_SECRET_KEY is not set")
    }

    // Generate a unique transaction reference (used as tx_ref and stored as Reference)
    reference := fmt.Sprintf("REF_%d_%s", time.Now().UnixNano(), req.SchoolID)

    // Validate required fields
    if req.Email == "" {
        return nil, fmt.Errorf("customer email is required")
    }
    if req.CallbackURL == "" {
        log.Printf("[Flutterwave] Warning: CallbackURL is empty; setting default")
        req.CallbackURL = "https://yourapp.com/payment/verify"
    }

    payload := map[string]interface{}{
        "tx_ref":       reference,
        "amount":       req.Amount.InexactFloat64(),
        "currency":     string(req.Currency),
        "redirect_url": req.CallbackURL,
        "customer": map[string]string{
            "email": req.Email,
        },
        "meta": map[string]interface{}{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
        },
    }

    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    log.Printf("[Flutterwave] Request payload: %s", string(jsonPayload))

    httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.flutterwave.com/v3/payments", bytes.NewBuffer(jsonPayload))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    log.Printf("[Flutterwave] Raw response body: %s", string(body))

    var fwResp FlutterwaveResponse
    if err := json.Unmarshal(body, &fwResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
    }

    // Check API status
    if fwResp.Status != "success" {
        errMsg := fwResp.Message
        if errMsg == "" {
            if dataMsg, ok := fwResp.Data["message"].(string); ok {
                errMsg = dataMsg
            } else {
                errMsg = "unknown error"
            }
        }
        log.Printf("[Flutterwave] API error: status=%s, message=%s", fwResp.Status, errMsg)
        return nil, fmt.Errorf("flutterwave error: %s", errMsg)
    }

    // Extract the payment link (required)
    link, ok := fwResp.Data["link"].(string)
    if !ok {
        log.Printf("[Flutterwave] Missing link in response data: %+v", fwResp.Data)
        return nil, fmt.Errorf("missing authorization link in response")
    }

    // Generate a local ID (UUID) for the PaymentIntent record
    localID := uuid.New().String()

    log.Printf("[Flutterwave] Payment created: reference=%s, link=%s", reference, link)

    return &models.PaymentIntent{
        ID:               localID,                     // local primary key
        Gateway:          models.GatewayFlutterwave,
        AuthorizationURL: link,
        Reference:        reference,                   // the tx_ref we sent
        Amount:           req.Amount,
        Currency:         req.Currency,
        Status:           models.IntentPending,
        CreatedAt:        time.Now(),
    }, nil
}



func (g *FlutterwaveGateway) VerifyPayment(ctx context.Context, reference string) (*PaymentVerification, error) {
    // ... (your existing implementation)
    url := fmt.Sprintf("https://api.flutterwave.com/v3/transactions/%s/verify", reference)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+g.secretKey)

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var fwResp FlutterwaveResponse
    json.Unmarshal(body, &fwResp)

    status := "pending"
    if fwResp.Status == "success" {
        data := fwResp.Data
        if data["status"] == "successful" {
            status = "succeeded"
        } else if data["status"] == "failed" {
            status = "failed"
        }
    }

    return &PaymentVerification{
        Status:      status,
        Amount:      decimal.NewFromFloat(fwResp.Data["amount"].(float64)),
        Reference:   reference,
        GatewayData: fwResp.Data,
    }, nil
}

func (g *FlutterwaveGateway) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResult, error) {
    planID := getFlutterwavePlanID(req.Tier, req.Interval)

    payload := map[string]interface{}{
        "email":    req.Email,
        "name":     req.CustomerName,
        "plan":     planID,
        "metadata": map[string]string{
            "school_id": req.SchoolID,
            "user_id":   req.UserID,
            "tier":      string(req.Tier),
        },
    }

    jsonPayload, _ := json.Marshal(payload)

    httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.flutterwave.com/v3/subscriptions", bytes.NewBuffer(jsonPayload))
    httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var fwResp FlutterwaveResponse
    json.Unmarshal(body, &fwResp)

    if fwResp.Status != "success" {
        return nil, fmt.Errorf("flutterwave subscription error: %s", fwResp.Message)
    }

    return &SubscriptionResult{
        SubscriptionID:        fmt.Sprintf("%v", fwResp.Data["id"]),
        GatewaySubscriptionID: fmt.Sprintf("%v", fwResp.Data["id"]),
        GatewayCustomerID:     fmt.Sprintf("%v", fwResp.Data["customer"]),
        Status:                fmt.Sprintf("%v", fwResp.Data["status"]),
    }, nil
}

func (g *FlutterwaveGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
    url := fmt.Sprintf("https://api.flutterwave.com/v3/subscriptions/%s/cancel", subscriptionID)

    req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
    req.Header.Set("Authorization", "Bearer "+g.secretKey)

    resp, err := g.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

func (g *FlutterwaveGateway) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionStatus, error) {
    url := fmt.Sprintf("https://api.flutterwave.com/v3/subscriptions/%s", subscriptionID)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+g.secretKey)

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var fwResp FlutterwaveResponse
    json.Unmarshal(body, &fwResp)

    active := false
    status := "inactive"
    if fwResp.Status == "success" {
        if fwResp.Data["status"] == "active" {
            active = true
            status = "active"
        }
    }

    return &SubscriptionStatus{
        Active: active,
        Status: status,
    }, nil
}

// ParseWebhook for Flutterwave
func (g *FlutterwaveGateway) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
    // Flutterwave sends a 'verif-hash' header
    expected := computeHMAC256(payload, g.secretKey)
    if expected != signature {
        return nil, fmt.Errorf("invalid signature")
    }

    var event map[string]interface{}
    if err := json.Unmarshal(payload, &event); err != nil {
        return nil, err
    }

    eventType, _ := event["event"].(string)
    data, _ := event["data"].(map[string]interface{})

    we := &WebhookEvent{Type: eventType, RawData: data}

    switch eventType {
    case "subscription.created":
        if subID, ok := data["id"].(float64); ok {
            we.GatewaySubscriptionID = fmt.Sprintf("%.0f", subID)
            we.Type = "subscription_activated"
        }
    case "charge.completed":
        if ref, ok := data["tx_ref"].(string); ok {
            we.Reference = ref
        }
        if amount, ok := data["amount"].(float64); ok {
            we.Amount = decimal.NewFromFloat(amount)
        }
        if currency, ok := data["currency"].(string); ok {
            we.Currency = currency
        }
        we.Type = "payment_success"
    case "subscription.cancelled":
        if subID, ok := data["id"].(float64); ok {
            we.GatewaySubscriptionID = fmt.Sprintf("%.0f", subID)
            we.Type = "subscription_cancelled"
        }
    }

    return we, nil
}

// Helper for HMAC‑SHA256
func computeHMAC256(payload []byte, secret string) string {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}

func (g *FlutterwaveGateway) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
    // Delegate to ParseWebhook if needed, or just return nil
    _, err := g.ParseWebhook(ctx, payload, signature)
    return err
}

func getFlutterwavePlanID(tier models.SubscriptionTier, interval models.PaymentInterval) string {
    plans := map[models.SubscriptionTier]map[models.PaymentInterval]string{
        models.TierBasic: {
            models.IntervalMonthly: os.Getenv("FLUTTERWAVE_PLAN_BASIC_MONTHLY"),
            models.IntervalYearly:  os.Getenv("FLUTTERWAVE_PLAN_BASIC_YEARLY"),
        },
        models.TierPremium: {
            models.IntervalMonthly: os.Getenv("FLUTTERWAVE_PLAN_PREMIUM_MONTHLY"),
            models.IntervalYearly:  os.Getenv("FLUTTERWAVE_PLAN_PREMIUM_YEARLY"),
        },
        models.TierEnterprise: {
            models.IntervalMonthly: os.Getenv("FLUTTERWAVE_PLAN_ENTERPRISE_MONTHLY"),
            models.IntervalYearly:  os.Getenv("FLUTTERWAVE_PLAN_ENTERPRISE_YEARLY"),
        },
    }
    
    if plan, ok := plans[tier][interval]; ok && plan != "" {
        return plan
    }
    return "1234"
}

