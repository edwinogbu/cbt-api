package service

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/google/uuid"

    "cbt-api/internal/models"
    "cbt-api/internal/subscription/dto"
    "cbt-api/internal/subscription/repository"
    "cbt-api/pkg/email"
    "cbt-api/pkg/payment"
)

type SubscriptionService struct {
    repo           *repository.SubscriptionRepository
    paymentService *payment.PaymentService
    emailService   *email.EmailService
}

func NewSubscriptionService(
    repo *repository.SubscriptionRepository,
    paymentService *payment.PaymentService,
    emailService *email.EmailService,
) *SubscriptionService {
    return &SubscriptionService{
        repo:           repo,
        paymentService: paymentService,
        emailService:   emailService,
    }
}

// ============================================
// CRUD METHODS (required by handler)
// ============================================

func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
    log.Printf("[SERVICE] GetSubscription: id=%s", id)
    sub, err := s.repo.FindByID(id)
    if err != nil {
        log.Printf("[SERVICE] GetSubscription error: %v", err)
        return nil, errors.New("subscription not found")
    }
    return s.toSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
    log.Printf("[SERVICE] GetSubscriptionsBySchool: school=%s", schoolID)
    subs, err := s.repo.FindBySchool(schoolID)
    if err != nil {
        log.Printf("[SERVICE] GetSubscriptionsBySchool error: %v", err)
        return nil, err
    }
    responses := make([]dto.SubscriptionResponse, len(subs))
    for i, sub := range subs {
        responses[i] = *s.toSubscriptionResponse(&sub)
    }
    return responses, nil
}

func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
    log.Printf("[SERVICE] GetCurrentSubscription: school=%s", schoolID)
    sub, err := s.repo.FindCurrentBySchool(schoolID)
    if err != nil {
        log.Printf("[SERVICE] GetCurrentSubscription error: %v", err)
        return nil, errors.New("no active subscription found")
    }
    return s.toSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
    log.Printf("[SERVICE] UpdateSubscription: id=%s, user=%s", id, userID)
    sub, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("subscription not found")
    }

    if req.Tier != nil {
        newTier := models.SubscriptionTier(*req.Tier)
        sub.Tier = newTier
        sub.Amount = models.Pricing[newTier][sub.PaymentInterval]
        sub.MaxStudents = models.TierLimits[newTier].MaxStudents
        sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
        sub.MaxExams = models.TierLimits[newTier].MaxExams
        sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
        sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
        sub.Features = getFeaturesForTier(newTier)
    }
    if req.AutoRenew != nil {
        sub.AutoRenew = *req.AutoRenew
    }
    if req.CancelAtEndDate != nil {
        sub.CancelAtEndDate = *req.CancelAtEndDate
    }
    if req.Status != nil {
        sub.Status = models.SubscriptionStatus(*req.Status)
    }
    sub.UpdatedAt = time.Now()
    sub.UpdatedBy = userID

    if err := s.repo.Update(sub); err != nil {
        log.Printf("[SERVICE] UpdateSubscription failed: %v", err)
        return nil, err
    }
    log.Printf("[SERVICE] UpdateSubscription success: id=%s", id)
    return s.toSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
    log.Printf("[SERVICE] CancelSubscription: id=%s, immediate=%v, reason=%s, user=%s", id, cancelImmediately, reason, userID)
    sub, err := s.repo.FindByID(id)
    if err != nil {
        return errors.New("subscription not found")
    }
    if cancelImmediately {
        sub.Status = models.SubStatusCancelled
        if sub.GatewaySubscriptionID != "" {
            _ = s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
        }
    } else {
        sub.CancelAtEndDate = true
    }
    sub.UpdatedAt = time.Now()
    sub.UpdatedBy = userID
    if err := s.repo.Update(sub); err != nil {
        log.Printf("[SERVICE] CancelSubscription failed: %v", err)
        return err
    }
    log.Printf("[SERVICE] CancelSubscription success: id=%s", id)
    return nil
}

func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
    log.Printf("[SERVICE] RenewSubscription: id=%s, interval=%s, user=%s", id, req.Interval, userID)
    sub, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("subscription not found")
    }
    newInterval := models.PaymentInterval(req.Interval)
    newAmount := models.Pricing[sub.Tier][newInterval]
    newEndDate := calculateEndDate(time.Now(), newInterval)

    sub.PaymentInterval = newInterval
    sub.Amount = newAmount
    sub.EndDate = newEndDate
    sub.Status = models.SubStatusActive
    sub.AutoRenew = true
    sub.CancelAtEndDate = false
    sub.UpdatedAt = time.Now()
    sub.UpdatedBy = userID

    if err := s.repo.Update(sub); err != nil {
        log.Printf("[SERVICE] RenewSubscription failed: %v", err)
        return nil, err
    }
    log.Printf("[SERVICE] RenewSubscription success: id=%s, new_end=%s", id, newEndDate)
    return s.toSubscriptionResponse(sub), nil
}

// ============================================
// PAYMENT INTENT METHODS (one-time payments)
// ============================================

func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
    log.Printf("[SERVICE] CreatePaymentIntent: subscription=%s, user=%s", subscriptionID, userID)
    sub, err := s.repo.FindByID(subscriptionID)
    if err != nil {
        return nil, errors.New("subscription not found")
    }

    invoice := &models.Invoice{
        ID:             uuid.New().String(),
        SubscriptionID: subscriptionID,
        SchoolID:       sub.SchoolID,
        UserID:         userID,
        InvoiceNumber:  dto.GenerateInvoiceNumber(),
        Amount:         sub.Amount,
        Total:          sub.Amount,
        Currency:       sub.Currency,
        Status:         models.InvoicePending,
        DueDate:        time.Now().AddDate(0, 0, 7),
        Items:          map[string]interface{}{"payment_type": "manual"},
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    if err := s.repo.CreateInvoice(invoice); err != nil {
        log.Printf("[SERVICE] CreatePaymentIntent: failed to create invoice: %v", err)
        return nil, err
    }

    // For a real implementation, fetch the user's email from your user repository
    // using userID. Here we use a placeholder – you must replace this.
    userEmail := "user@example.com"

    // Ensure callback URL is set (required for Flutterwave)
    callbackURL := req.SuccessURL
    if callbackURL == "" {
        callbackURL = "https://yourapp.com/payment/success"
    }

    paymentReq := &payment.PaymentRequest{
        SchoolID:    sub.SchoolID,
        UserID:      userID,
        Amount:      sub.Amount,
        Currency:    sub.Currency,
        Email:       userEmail,
        CallbackURL: callbackURL,
        SuccessURL:  req.SuccessURL,
        CancelURL:   req.CancelURL,
        Metadata: map[string]interface{}{
            "subscription_id": subscriptionID,
            "invoice_id":      invoice.ID,
            "payment_type":    "manual",
        },
    }

    pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
    if err != nil {
        log.Printf("[SERVICE] CreatePaymentIntent: payment creation failed: %v", err)
        return nil, err
    }

    dbPI := &models.PaymentIntent{
        ID:               uuid.New().String(),
        SubscriptionID:   subscriptionID,
        InvoiceID:        invoice.ID,
        SchoolID:         sub.SchoolID,
        UserID:           userID,
        IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
        Gateway:          sub.Gateway,
        Amount:           sub.Amount,
        Currency:         sub.Currency,
        Reference:        pi.Reference,
        ClientSecret:     pi.ClientSecret,
        AuthorizationURL: pi.AuthorizationURL,
        AccessCode:       pi.AccessCode,
        Status:           models.IntentPending,
        ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
        CreatedAt:        time.Now(),
        UpdatedAt:        time.Now(),
    }
    s.repo.CreatePaymentIntent(dbPI)

    log.Printf("[SERVICE] CreatePaymentIntent success: payment_intent=%s, url=%s", dbPI.ID, dbPI.AuthorizationURL)
    return s.toPaymentIntentResponse(dbPI), nil
}

func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
    log.Printf("[SERVICE] ConfirmPaymentIntent: id=%s", paymentIntentID)
    pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
    if err != nil {
        return errors.New("payment intent not found")
    }
    if pi.IsFinalized {
        return errors.New("payment intent already finalized")
    }

    verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
    if err != nil {
        log.Printf("[SERVICE] ConfirmPaymentIntent: verification failed: %v", err)
        return err
    }

    pi.Status = models.PaymentIntentStatus(verification.Status)
    pi.GatewayResponse = verification.GatewayData

    if verification.Status == "succeeded" {
        pi.PaidAt = timePtr(time.Now())
        pi.IsFinalized = true
        s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())

        sub, _ := s.repo.FindByID(pi.SubscriptionID)
        if sub != nil && sub.Status == models.SubStatusPending {
            sub.Status = models.SubStatusActive
            sub.StartDate = time.Now()
            sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
            sub.LastPaymentDate = timePtr(time.Now())
            sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
            s.repo.Update(sub)
            log.Printf("[SERVICE] ConfirmPaymentIntent: subscription %s activated", sub.ID)
        }

        transaction := &models.PaymentTransaction{
            ID:              uuid.New().String(),
            SubscriptionID:  pi.SubscriptionID,
            PaymentIntentID: pi.ID,
            InvoiceID:       pi.InvoiceID,
            SchoolID:        pi.SchoolID,
            UserID:          pi.UserID,
            Amount:          pi.Amount,
            Currency:        pi.Currency,
            PaymentStatus:   models.PaymentPaid,
            Gateway:         pi.Gateway,
            Reference:       pi.Reference,
            PaidAt:          timePtr(time.Now()),
            CreatedAt:       time.Now(),
            UpdatedAt:       time.Now(),
        }
        s.repo.CreatePaymentTransaction(transaction)
    }

    s.repo.UpdatePaymentIntent(pi)
    log.Printf("[SERVICE] ConfirmPaymentIntent: final status=%s", pi.Status)
    return nil
}

// ============================================
// CREATE SUBSCRIPTION (one‑time payment link)
// ============================================

func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
    log.Printf("[SERVICE] CreateSubscription: school=%s tier=%s interval=%s gateway=%s user=%s",
        schoolID, req.Tier, req.Interval, req.Gateway, userID)

    existing, _ := s.repo.FindCurrentBySchool(schoolID)
    if existing != nil {
        log.Printf("[SERVICE] CreateSubscription: active subscription already exists for school %s", schoolID)
        return nil, errors.New("school already has an active subscription")
    }

    tier := models.SubscriptionTier(req.Tier)
    interval := models.PaymentInterval(req.Interval)
    amount, exists := models.Pricing[tier][interval]
    if !exists {
        log.Printf("[SERVICE] CreateSubscription: invalid pricing for tier=%s interval=%s", req.Tier, req.Interval)
        return nil, errors.New("invalid pricing for selected tier and interval")
    }

    now := time.Now()
    subscriptionID := uuid.New().String()
    subscription := &models.Subscription{
        ID:                subscriptionID,
        UserID:            userID,
        SchoolID:          schoolID,
        Tier:              tier,
        Status:            models.SubStatusPending,
        Gateway:           models.PaymentGateway(req.Gateway),
        Amount:            amount,
        Currency:          models.CurrencyNGN,
        PaymentInterval:   interval,
        StartDate:         now,
        EndDate:           now.AddDate(1, 0, 0), // temporary, will be updated after payment
        AutoRenew:         req.AutoRenew,
        MaxStudents:       models.TierLimits[tier].MaxStudents,
        MaxTeachers:       models.TierLimits[tier].MaxTeachers,
        MaxExams:          models.TierLimits[tier].MaxExams,
        MaxQuestions:      models.TierLimits[tier].MaxQuestions,
        MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
        Features:          getFeaturesForTier(tier),
        CreatedAt:         now,
        UpdatedAt:         now,
    }
    if err := s.repo.Create(subscription); err != nil {
        log.Printf("[SERVICE] CreateSubscription: failed to create subscription record: %v", err)
        return nil, err
    }

    invoice := &models.Invoice{
        ID:             uuid.New().String(),
        SubscriptionID: subscriptionID,
        SchoolID:       schoolID,
        UserID:         userID,
        InvoiceNumber:  dto.GenerateInvoiceNumber(),
        Amount:         amount,
        Total:          amount,
        Currency:       models.CurrencyNGN,
        Status:         models.InvoicePending,
        DueDate:        now.AddDate(0, 0, 7),
        Items:          map[string]interface{}{"tier": string(tier), "interval": string(interval)},
        CreatedAt:      now,
        UpdatedAt:      now,
    }
    if err := s.repo.CreateInvoice(invoice); err != nil {
        log.Printf("[SERVICE] CreateSubscription: failed to create invoice: %v", err)
        s.repo.Delete(subscriptionID)
        return nil, err
    }

    // For a real implementation, fetch the user's email from your user repository
    // using userID. Here we use req.Email as provided by the frontend – this is acceptable.
    userEmail := req.Email
    if userEmail == "" {
        userEmail = "customer@example.com"
    }

    // Ensure callback URL is set (required for Flutterwave)
    callbackURL := req.SuccessURL
    if callbackURL == "" {
        callbackURL = "https://yourapp.com/payment/success"
    }

    paymentReq := &payment.PaymentRequest{
        SchoolID:    schoolID,
        UserID:      userID,
        Amount:      amount,
        Currency:    models.CurrencyNGN,
        Email:       userEmail,
        CallbackURL: callbackURL,
        SuccessURL:  req.SuccessURL,
        CancelURL:   req.CancelURL,
        Metadata: map[string]interface{}{
            "subscription_id": subscriptionID,
            "invoice_id":      invoice.ID,
        },
    }

    paymentIntent, err := s.paymentService.CreatePayment(ctx, models.PaymentGateway(req.Gateway), paymentReq)
    if err != nil {
        log.Printf("[SERVICE] CreateSubscription: payment creation failed: %v", err)
        s.repo.Delete(subscriptionID)
        return nil, fmt.Errorf("failed to create payment link: %w", err)
    }

    dbPI := &models.PaymentIntent{
        ID:               uuid.New().String(),
        SubscriptionID:   subscriptionID,
        InvoiceID:        invoice.ID,
        SchoolID:         schoolID,
        UserID:           userID,
        IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
        Gateway:          models.PaymentGateway(req.Gateway),
        Amount:           amount,
        Currency:         models.CurrencyNGN,
        Reference:        paymentIntent.Reference,
        AuthorizationURL: paymentIntent.AuthorizationURL,
        Status:           models.IntentPending,
        ExpiresAt:        timePtr(now.Add(48 * time.Hour)),
        CreatedAt:        now,
        UpdatedAt:        now,
    }
    s.repo.CreatePaymentIntent(dbPI)

    history := &models.SubscriptionHistory{
        ID:             uuid.New().String(),
        SubscriptionID: subscriptionID,
        SchoolID:       schoolID,
        UserID:         userID,
        NewTier:        tier,
        NewStatus:      models.SubStatusPending,
        NewAmount:      amount,
        ChangeReason:   "subscription_created_pending_payment",
        ChangedBy:      userID,
        CreatedAt:      now,
    }
    s.repo.CreateHistory(history)

    go s.sendPaymentLinkEmail(subscription, invoice, dbPI, userEmail)

    log.Printf("[SERVICE] CreateSubscription success: subscription=%s, payment_link=%s", subscriptionID, dbPI.AuthorizationURL)
    return &dto.CreateSubscriptionResponse{
        Subscription:  s.toSubscriptionResponse(subscription),
        PaymentIntent: s.toPaymentIntentResponse(dbPI),
        Invoice:       s.toInvoiceResponse(invoice),
    }, nil
}

// ============================================
// WEBHOOK PROCESSING (only payment_success)
// ============================================

func (s *SubscriptionService) ProcessWebhook(ctx context.Context, gateway models.PaymentGateway, payload []byte, signature string) error {
    log.Printf("[WEBHOOK] Received for gateway %s", gateway)
    gw, err := s.paymentService.GetGateway(gateway)
    if err != nil {
        log.Printf("[WEBHOOK] Gateway not found: %v", err)
        return err
    }
    event, err := gw.ParseWebhook(ctx, payload, signature)
    if err != nil {
        log.Printf("[WEBHOOK] Failed to parse webhook: %v", err)
        return err
    }

    webhookEvent := &models.WebhookEvent{
        ID:             uuid.New().String(),
        Gateway:        gateway,
        EventType:      event.Type,
        Payload:        event.RawData,
        IdempotencyKey: event.Reference,
        Status:         "pending",
        CreatedAt:      time.Now(),
    }
    s.repo.CreateWebhookEvent(webhookEvent)

    if event.Type == "payment_success" {
        log.Printf("[WEBHOOK] Payment success for reference %s", event.Reference)
        pi, err := s.repo.FindPaymentIntentByReference(event.Reference)
        if err != nil {
            log.Printf("[WEBHOOK] Payment intent not found for reference %s", event.Reference)
            s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "failed", "payment intent not found")
            return nil
        }
        sub, _ := s.repo.FindByID(pi.SubscriptionID)
        if sub != nil && sub.Status == models.SubStatusPending {
            sub.Status = models.SubStatusActive
            sub.StartDate = time.Now()
            sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
            sub.LastPaymentDate = timePtr(time.Now())
            sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
            s.repo.Update(sub)
            s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
            log.Printf("[WEBHOOK] Subscription %s activated", sub.ID)
        }
        s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "processed", "")
    } else {
        log.Printf("[WEBHOOK] Ignoring event type %s", event.Type)
        s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "ignored", "")
    }
    return nil
}

// ============================================
// AUTO‑RENEWAL (called by cron)
// ============================================

func (s *SubscriptionService) GenerateRenewalPaymentLink(subscriptionID string) error {
    log.Printf("[RENEWAL] Generating renewal payment link for subscription %s", subscriptionID)
    sub, err := s.repo.FindByID(subscriptionID)
    if err != nil {
        return err
    }
    if sub.Status != models.SubStatusActive {
        log.Printf("[RENEWAL] Subscription %s is not active, skipping", subscriptionID)
        return nil
    }

    invoice := &models.Invoice{
        ID:             uuid.New().String(),
        SubscriptionID: sub.ID,
        SchoolID:       sub.SchoolID,
        UserID:         sub.UserID,
        InvoiceNumber:  dto.GenerateInvoiceNumber(),
        Amount:         sub.Amount,
        Total:          sub.Amount,
        Currency:       sub.Currency,
        Status:         models.InvoicePending,
        DueDate:        time.Now().AddDate(0, 0, 7),
        Items:          map[string]interface{}{"renewal": true},
        CreatedAt:      time.Now(),
    }
    if err := s.repo.CreateInvoice(invoice); err != nil {
        log.Printf("[RENEWAL] Failed to create renewal invoice: %v", err)
        return err
    }

    // Fetch user email – replace with actual user lookup
    userEmail := "customer@example.com"

    // Use a default success URL for the renewal payment
    defaultSuccessURL := "https://yourapp.com/payment/success"

    paymentReq := &payment.PaymentRequest{
        SchoolID:    sub.SchoolID,
        UserID:      sub.UserID,
        Amount:      sub.Amount,
        Currency:    sub.Currency,
        Email:       userEmail,
        CallbackURL: defaultSuccessURL,
        Metadata: map[string]interface{}{
            "subscription_id": sub.ID,
            "invoice_id":      invoice.ID,
            "renewal":         true,
        },
    }
    pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
    if err != nil {
        log.Printf("[RENEWAL] Failed to create payment for renewal: %v", err)
        return err
    }
    dbPI := &models.PaymentIntent{
        ID:               uuid.New().String(),
        SubscriptionID:   sub.ID,
        InvoiceID:        invoice.ID,
        SchoolID:         sub.SchoolID,
        UserID:           sub.UserID,
        Gateway:          sub.Gateway,
        Amount:           sub.Amount,
        Currency:         sub.Currency,
        Reference:        pi.Reference,
        AuthorizationURL: pi.AuthorizationURL,
        Status:           models.IntentPending,
        ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
        CreatedAt:        time.Now(),
    }
    s.repo.CreatePaymentIntent(dbPI)
    log.Printf("[RENEWAL] Renewal payment link generated: %s", dbPI.AuthorizationURL)
    go s.sendRenewalEmail(sub, invoice, dbPI, userEmail)
    return nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
    switch interval {
    case models.IntervalMonthly:
        return start.AddDate(0, 1, 0)
    case models.IntervalQuarterly:
        return start.AddDate(0, 3, 0)
    case models.IntervalYearly:
        return start.AddDate(1, 0, 0)
    default:
        return start.AddDate(0, 1, 0)
    }
}

func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
    next := calculateEndDate(currentEnd, interval)
    return &next
}

func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
    features := make(models.JSONMap)
    switch tier {
    case models.TierBasic:
        features["analytics"] = false
        features["api_access"] = false
        features["priority_support"] = false
        features["bulk_import"] = false
        features["white_label"] = false
    case models.TierPremium:
        features["analytics"] = true
        features["api_access"] = true
        features["priority_support"] = false
        features["bulk_import"] = true
        features["white_label"] = false
        features["custom_reports"] = true
    case models.TierEnterprise:
        features["analytics"] = true
        features["api_access"] = true
        features["priority_support"] = true
        features["bulk_import"] = true
        features["white_label"] = true
        features["custom_reports"] = true
        features["dedicated_server"] = true
        features["sla"] = true
    }
    return features
}

func timePtr(t time.Time) *time.Time {
    return &t
}

// ============================================
// RESPONSE MAPPERS
// ============================================

func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
    return &dto.SubscriptionResponse{
        ID:              sub.ID,
        SchoolID:        sub.SchoolID,
        Tier:            string(sub.Tier),
        Status:          string(sub.Status),
        Gateway:         string(sub.Gateway),
        Amount:          sub.Amount,
        Currency:        string(sub.Currency),
        PaymentInterval: string(sub.PaymentInterval),
        StartDate:       sub.StartDate,
        EndDate:         sub.EndDate,
        TrialEndsAt:     sub.TrialEndsAt,
        AutoRenew:       sub.AutoRenew,
        CancelAtEndDate: sub.CancelAtEndDate,
        LastPaymentDate: sub.LastPaymentDate,
        NextPaymentDate: sub.NextPaymentDate,
        MaxStudents:     sub.MaxStudents,
        MaxTeachers:     sub.MaxTeachers,
        MaxExams:        sub.MaxExams,
        MaxQuestions:    sub.MaxQuestions,
        MaxStorageMB:    sub.MaxStorageMB,
        Features:        sub.Features,
        CreatedAt:       sub.CreatedAt,
        UpdatedAt:       sub.UpdatedAt,
    }
}

func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
    return &dto.PaymentIntentResponse{
        ID:               pi.ID,
        SubscriptionID:   pi.SubscriptionID,
        InvoiceID:        pi.InvoiceID,
        Gateway:          string(pi.Gateway),
        Amount:           pi.Amount,
        Currency:         string(pi.Currency),
        Reference:        pi.Reference,
        ClientSecret:     pi.ClientSecret,
        AuthorizationURL: pi.AuthorizationURL,
        AccessCode:       pi.AccessCode,
        Status:           string(pi.Status),
        ExpiresAt:        pi.ExpiresAt,
        PaymentMethod:    string(pi.PaymentMethod),
    }
}

func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
    return &dto.InvoiceResponse{
        ID:            inv.ID,
        InvoiceNumber: inv.InvoiceNumber,
        Amount:        inv.Amount,
        Tax:           inv.Tax,
        Discount:      inv.Discount,
        Total:         inv.Total,
        Currency:      string(inv.Currency),
        Status:        string(inv.Status),
        DueDate:       inv.DueDate,
        PaidAt:        inv.PaidAt,
        PDFURL:        inv.PDFURL,
        Items:         inv.Items,
        CreatedAt:     inv.CreatedAt,
    }
}

// ============================================
// EMAIL STUBS (replace with real email sending)
// ============================================

func (s *SubscriptionService) sendPaymentLinkEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
    log.Printf("[EMAIL] Payment link to %s: %s", email, pi.AuthorizationURL)
}

func (s *SubscriptionService) sendRenewalEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
    log.Printf("[EMAIL] Renewal payment link to %s: %s", email, pi.AuthorizationURL)
}








// package service

// import (
//     "context"
//     "errors"
//     "fmt"
//     "log"
//     "time"

//     "github.com/google/uuid"
//     "github.com/shopspring/decimal"

//     "cbt-api/internal/models"
//     "cbt-api/internal/subscription/dto"
//     "cbt-api/internal/subscription/repository"
//     "cbt-api/pkg/email"
//     "cbt-api/pkg/payment"
// )

// type SubscriptionService struct {
//     repo           *repository.SubscriptionRepository
//     paymentService *payment.PaymentService
//     emailService   *email.EmailService
// }

// func NewSubscriptionService(
//     repo *repository.SubscriptionRepository,
//     paymentService *payment.PaymentService,
//     emailService *email.EmailService,
// ) *SubscriptionService {
//     return &SubscriptionService{
//         repo:           repo,
//         paymentService: paymentService,
//         emailService:   emailService,
//     }
// }

// // ============================================
// // CRUD METHODS (required by handler)
// // ============================================

// func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
//     log.Printf("[SERVICE] GetSubscription: id=%s", id)
//     sub, err := s.repo.FindByID(id)
//     if err != nil {
//         log.Printf("[SERVICE] GetSubscription error: %v", err)
//         return nil, errors.New("subscription not found")
//     }
//     return s.toSubscriptionResponse(sub), nil
// }

// func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
//     log.Printf("[SERVICE] GetSubscriptionsBySchool: school=%s", schoolID)
//     subs, err := s.repo.FindBySchool(schoolID)
//     if err != nil {
//         log.Printf("[SERVICE] GetSubscriptionsBySchool error: %v", err)
//         return nil, err
//     }
//     responses := make([]dto.SubscriptionResponse, len(subs))
//     for i, sub := range subs {
//         responses[i] = *s.toSubscriptionResponse(&sub)
//     }
//     return responses, nil
// }

// func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
//     log.Printf("[SERVICE] GetCurrentSubscription: school=%s", schoolID)
//     sub, err := s.repo.FindCurrentBySchool(schoolID)
//     if err != nil {
//         log.Printf("[SERVICE] GetCurrentSubscription error: %v", err)
//         return nil, errors.New("no active subscription found")
//     }
//     return s.toSubscriptionResponse(sub), nil
// }

// func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
//     log.Printf("[SERVICE] UpdateSubscription: id=%s, user=%s", id, userID)
//     sub, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("subscription not found")
//     }

//     if req.Tier != nil {
//         newTier := models.SubscriptionTier(*req.Tier)
//         sub.Tier = newTier
//         sub.Amount = models.Pricing[newTier][sub.PaymentInterval]
//         sub.MaxStudents = models.TierLimits[newTier].MaxStudents
//         sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
//         sub.MaxExams = models.TierLimits[newTier].MaxExams
//         sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
//         sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
//         sub.Features = getFeaturesForTier(newTier)
//     }
//     if req.AutoRenew != nil {
//         sub.AutoRenew = *req.AutoRenew
//     }
//     if req.CancelAtEndDate != nil {
//         sub.CancelAtEndDate = *req.CancelAtEndDate
//     }
//     if req.Status != nil {
//         sub.Status = models.SubscriptionStatus(*req.Status)
//     }
//     sub.UpdatedAt = time.Now()
//     sub.UpdatedBy = userID

//     if err := s.repo.Update(sub); err != nil {
//         log.Printf("[SERVICE] UpdateSubscription failed: %v", err)
//         return nil, err
//     }
//     log.Printf("[SERVICE] UpdateSubscription success: id=%s", id)
//     return s.toSubscriptionResponse(sub), nil
// }

// func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
//     log.Printf("[SERVICE] CancelSubscription: id=%s, immediate=%v, reason=%s, user=%s", id, cancelImmediately, reason, userID)
//     sub, err := s.repo.FindByID(id)
//     if err != nil {
//         return errors.New("subscription not found")
//     }
//     if cancelImmediately {
//         sub.Status = models.SubStatusCancelled
//         if sub.GatewaySubscriptionID != "" {
//             _ = s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
//         }
//     } else {
//         sub.CancelAtEndDate = true
//     }
//     sub.UpdatedAt = time.Now()
//     sub.UpdatedBy = userID
//     if err := s.repo.Update(sub); err != nil {
//         log.Printf("[SERVICE] CancelSubscription failed: %v", err)
//         return err
//     }
//     log.Printf("[SERVICE] CancelSubscription success: id=%s", id)
//     return nil
// }

// func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
//     log.Printf("[SERVICE] RenewSubscription: id=%s, interval=%s, user=%s", id, req.Interval, userID)
//     sub, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("subscription not found")
//     }
//     newInterval := models.PaymentInterval(req.Interval)
//     newAmount := models.Pricing[sub.Tier][newInterval]
//     newEndDate := calculateEndDate(time.Now(), newInterval)

//     sub.PaymentInterval = newInterval
//     sub.Amount = newAmount
//     sub.EndDate = newEndDate
//     sub.Status = models.SubStatusActive
//     sub.AutoRenew = true
//     sub.CancelAtEndDate = false
//     sub.UpdatedAt = time.Now()
//     sub.UpdatedBy = userID

//     if err := s.repo.Update(sub); err != nil {
//         log.Printf("[SERVICE] RenewSubscription failed: %v", err)
//         return nil, err
//     }
//     log.Printf("[SERVICE] RenewSubscription success: id=%s, new_end=%s", id, newEndDate)
//     return s.toSubscriptionResponse(sub), nil
// }

// // ============================================
// // PAYMENT INTENT METHODS (one-time payments)
// // ============================================

// func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
//     log.Printf("[SERVICE] CreatePaymentIntent: subscription=%s, user=%s", subscriptionID, userID)
//     sub, err := s.repo.FindByID(subscriptionID)
//     if err != nil {
//         return nil, errors.New("subscription not found")
//     }

//     invoice := &models.Invoice{
//         ID:             uuid.New().String(),
//         SubscriptionID: subscriptionID,
//         SchoolID:       sub.SchoolID,
//         UserID:         userID,
//         InvoiceNumber:  dto.GenerateInvoiceNumber(),
//         Amount:         sub.Amount,
//         Total:          sub.Amount,
//         Currency:       sub.Currency,
//         Status:         models.InvoicePending,
//         DueDate:        time.Now().AddDate(0, 0, 7),
//         Items:          map[string]interface{}{"payment_type": "manual"},
//         CreatedAt:      time.Now(),
//         UpdatedAt:      time.Now(),
//     }
//     if err := s.repo.CreateInvoice(invoice); err != nil {
//         log.Printf("[SERVICE] CreatePaymentIntent: failed to create invoice: %v", err)
//         return nil, err
//     }

//     // IMPORTANT: Set CallbackURL for Flutterwave (required)
//     callbackURL := req.SuccessURL
//     if callbackURL == "" {
//         callbackURL = "https://yourapp.com/payment/success" // fallback
//     }

//     paymentReq := &payment.PaymentRequest{
//         SchoolID:    sub.SchoolID,
//         UserID:      userID,
//         Amount:      sub.Amount,
//         Currency:    sub.Currency,
//         Email:       req.SuccessURL, // placeholder – fetch actual user email
//         CallbackURL: callbackURL,    // ← Fix: Flutterwave needs this
//         SuccessURL:  req.SuccessURL,
//         CancelURL:   req.CancelURL,
//         Metadata: map[string]interface{}{
//             "subscription_id": subscriptionID,
//             "invoice_id":      invoice.ID,
//             "payment_type":    "manual",
//         },
//     }

//     pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
//     if err != nil {
//         log.Printf("[SERVICE] CreatePaymentIntent: payment creation failed: %v", err)
//         return nil, err
//     }

//     dbPI := &models.PaymentIntent{
//         ID:               uuid.New().String(),
//         SubscriptionID:   subscriptionID,
//         InvoiceID:        invoice.ID,
//         SchoolID:         sub.SchoolID,
//         UserID:           userID,
//         IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
//         Gateway:          sub.Gateway,
//         Amount:           sub.Amount,
//         Currency:         sub.Currency,
//         Reference:        pi.Reference,
//         ClientSecret:     pi.ClientSecret,
//         AuthorizationURL: pi.AuthorizationURL,
//         AccessCode:       pi.AccessCode,
//         Status:           models.IntentPending,
//         ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
//         CreatedAt:        time.Now(),
//         UpdatedAt:        time.Now(),
//     }
//     s.repo.CreatePaymentIntent(dbPI)

//     log.Printf("[SERVICE] CreatePaymentIntent success: payment_intent=%s, url=%s", dbPI.ID, dbPI.AuthorizationURL)
//     return s.toPaymentIntentResponse(dbPI), nil
// }

// func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
//     log.Printf("[SERVICE] ConfirmPaymentIntent: id=%s", paymentIntentID)
//     pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
//     if err != nil {
//         return errors.New("payment intent not found")
//     }
//     if pi.IsFinalized {
//         return errors.New("payment intent already finalized")
//     }

//     verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
//     if err != nil {
//         log.Printf("[SERVICE] ConfirmPaymentIntent: verification failed: %v", err)
//         return err
//     }

//     pi.Status = models.PaymentIntentStatus(verification.Status)
//     pi.GatewayResponse = verification.GatewayData

//     if verification.Status == "succeeded" {
//         pi.PaidAt = timePtr(time.Now())
//         pi.IsFinalized = true
//         s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())

//         sub, _ := s.repo.FindByID(pi.SubscriptionID)
//         if sub != nil && sub.Status == models.SubStatusPending {
//             sub.Status = models.SubStatusActive
//             sub.StartDate = time.Now()
//             sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
//             sub.LastPaymentDate = timePtr(time.Now())
//             sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
//             s.repo.Update(sub)
//             log.Printf("[SERVICE] ConfirmPaymentIntent: subscription %s activated", sub.ID)
//         }

//         transaction := &models.PaymentTransaction{
//             ID:              uuid.New().String(),
//             SubscriptionID:  pi.SubscriptionID,
//             PaymentIntentID: pi.ID,
//             InvoiceID:       pi.InvoiceID,
//             SchoolID:        pi.SchoolID,
//             UserID:          pi.UserID,
//             Amount:          pi.Amount,
//             Currency:        pi.Currency,
//             PaymentStatus:   models.PaymentPaid,
//             Gateway:         pi.Gateway,
//             Reference:       pi.Reference,
//             PaidAt:          timePtr(time.Now()),
//             CreatedAt:       time.Now(),
//             UpdatedAt:       time.Now(),
//         }
//         s.repo.CreatePaymentTransaction(transaction)
//     }

//     s.repo.UpdatePaymentIntent(pi)
//     log.Printf("[SERVICE] ConfirmPaymentIntent: final status=%s", pi.Status)
//     return nil
// }

// // ============================================
// // CREATE SUBSCRIPTION (one‑time payment link)
// // ============================================

// func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
//     log.Printf("[SERVICE] CreateSubscription: school=%s tier=%s interval=%s gateway=%s user=%s",
//         schoolID, req.Tier, req.Interval, req.Gateway, userID)

//     existing, _ := s.repo.FindCurrentBySchool(schoolID)
//     if existing != nil {
//         log.Printf("[SERVICE] CreateSubscription: active subscription already exists for school %s", schoolID)
//         return nil, errors.New("school already has an active subscription")
//     }

//     tier := models.SubscriptionTier(req.Tier)
//     interval := models.PaymentInterval(req.Interval)
//     amount, exists := models.Pricing[tier][interval]
//     if !exists {
//         log.Printf("[SERVICE] CreateSubscription: invalid pricing for tier=%s interval=%s", req.Tier, req.Interval)
//         return nil, errors.New("invalid pricing for selected tier and interval")
//     }

//     now := time.Now()
//     subscriptionID := uuid.New().String()
//     subscription := &models.Subscription{
//         ID:                subscriptionID,
//         UserID:            userID,
//         SchoolID:          schoolID,
//         Tier:              tier,
//         Status:            models.SubStatusPending,
//         Gateway:           models.PaymentGateway(req.Gateway),
//         Amount:            amount,
//         Currency:          models.CurrencyNGN,
//         PaymentInterval:   interval,
//         StartDate:         now,
//         EndDate:           now.AddDate(1, 0, 0),
//         AutoRenew:         req.AutoRenew,
//         MaxStudents:       models.TierLimits[tier].MaxStudents,
//         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
//         MaxExams:          models.TierLimits[tier].MaxExams,
//         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
//         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
//         Features:          getFeaturesForTier(tier),
//         CreatedAt:         now,
//         UpdatedAt:         now,
//     }
//     if err := s.repo.Create(subscription); err != nil {
//         log.Printf("[SERVICE] CreateSubscription: failed to create subscription record: %v", err)
//         return nil, err
//     }

//     invoice := &models.Invoice{
//         ID:             uuid.New().String(),
//         SubscriptionID: subscriptionID,
//         SchoolID:       schoolID,
//         UserID:         userID,
//         InvoiceNumber:  dto.GenerateInvoiceNumber(),
//         Amount:         amount,
//         Total:          amount,
//         Currency:       models.CurrencyNGN,
//         Status:         models.InvoicePending,
//         DueDate:        now.AddDate(0, 0, 7),
//         Items:          map[string]interface{}{"tier": string(tier), "interval": string(interval)},
//         CreatedAt:      now,
//         UpdatedAt:      now,
//     }
//     if err := s.repo.CreateInvoice(invoice); err != nil {
//         log.Printf("[SERVICE] CreateSubscription: failed to create invoice: %v", err)
//         s.repo.Delete(subscriptionID)
//         return nil, err
//     }

//     // IMPORTANT: Set CallbackURL for Flutterwave (required)
//     callbackURL := req.SuccessURL
//     if callbackURL == "" {
//         callbackURL = "https://yourapp.com/payment/success"
//     }

//     paymentReq := &payment.PaymentRequest{
//         SchoolID:    schoolID,
//         UserID:      userID,
//         Amount:      amount,
//         Currency:    models.CurrencyNGN,
//         Email:       req.Email,
//         CallbackURL: callbackURL, // ← Fix: Flutterwave needs this
//         SuccessURL:  req.SuccessURL,
//         CancelURL:   req.CancelURL,
//         Metadata: map[string]interface{}{
//             "subscription_id": subscriptionID,
//             "invoice_id":      invoice.ID,
//         },
//     }

//     paymentIntent, err := s.paymentService.CreatePayment(ctx, models.PaymentGateway(req.Gateway), paymentReq)
//     if err != nil {
//         log.Printf("[SERVICE] CreateSubscription: payment creation failed: %v", err)
//         s.repo.Delete(subscriptionID)
//         return nil, fmt.Errorf("failed to create payment link: %w", err)
//     }

//     dbPI := &models.PaymentIntent{
//         ID:               uuid.New().String(),
//         SubscriptionID:   subscriptionID,
//         InvoiceID:        invoice.ID,
//         SchoolID:         schoolID,
//         UserID:           userID,
//         IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
//         Gateway:          models.PaymentGateway(req.Gateway),
//         Amount:           amount,
//         Currency:         models.CurrencyNGN,
//         Reference:        paymentIntent.Reference,
//         AuthorizationURL: paymentIntent.AuthorizationURL,
//         Status:           models.IntentPending,
//         ExpiresAt:        timePtr(now.Add(48 * time.Hour)),
//         CreatedAt:        now,
//         UpdatedAt:        now,
//     }
//     s.repo.CreatePaymentIntent(dbPI)

//     history := &models.SubscriptionHistory{
//         ID:             uuid.New().String(),
//         SubscriptionID: subscriptionID,
//         SchoolID:       schoolID,
//         UserID:         userID,
//         NewTier:        tier,
//         NewStatus:      models.SubStatusPending,
//         NewAmount:      amount,
//         ChangeReason:   "subscription_created_pending_payment",
//         ChangedBy:      userID,
//         CreatedAt:      now,
//     }
//     s.repo.CreateHistory(history)

//     go s.sendPaymentLinkEmail(subscription, invoice, dbPI, req.Email)

//     log.Printf("[SERVICE] CreateSubscription success: subscription=%s, payment_link=%s", subscriptionID, dbPI.AuthorizationURL)
//     return &dto.CreateSubscriptionResponse{
//         Subscription:  s.toSubscriptionResponse(subscription),
//         PaymentIntent: s.toPaymentIntentResponse(dbPI),
//         Invoice:       s.toInvoiceResponse(invoice),
//     }, nil
// }

// // ============================================
// // WEBHOOK PROCESSING
// // ============================================

// func (s *SubscriptionService) ProcessWebhook(ctx context.Context, gateway models.PaymentGateway, payload []byte, signature string) error {
//     log.Printf("[WEBHOOK] Received for gateway %s", gateway)
//     gw, err := s.paymentService.GetGateway(gateway)
//     if err != nil {
//         log.Printf("[WEBHOOK] Gateway not found: %v", err)
//         return err
//     }
//     event, err := gw.ParseWebhook(ctx, payload, signature)
//     if err != nil {
//         log.Printf("[WEBHOOK] Failed to parse webhook: %v", err)
//         return err
//     }

//     webhookEvent := &models.WebhookEvent{
//         ID:             uuid.New().String(),
//         Gateway:        gateway,
//         EventType:      event.Type,
//         Payload:        event.RawData,
//         IdempotencyKey: event.Reference,
//         Status:         "pending",
//         CreatedAt:      time.Now(),
//     }
//     s.repo.CreateWebhookEvent(webhookEvent)

//     if event.Type == "payment_success" {
//         log.Printf("[WEBHOOK] Payment success for reference %s", event.Reference)
//         pi, err := s.repo.FindPaymentIntentByReference(event.Reference)
//         if err != nil {
//             log.Printf("[WEBHOOK] Payment intent not found for reference %s", event.Reference)
//             s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "failed", "payment intent not found")
//             return nil
//         }
//         sub, _ := s.repo.FindByID(pi.SubscriptionID)
//         if sub != nil && sub.Status == models.SubStatusPending {
//             sub.Status = models.SubStatusActive
//             sub.StartDate = time.Now()
//             sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
//             sub.LastPaymentDate = timePtr(time.Now())
//             sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
//             s.repo.Update(sub)
//             s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
//             log.Printf("[WEBHOOK] Subscription %s activated", sub.ID)
//         }
//         s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "processed", "")
//     } else {
//         log.Printf("[WEBHOOK] Ignoring event type %s", event.Type)
//         s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "ignored", "")
//     }
//     return nil
// }

// // ============================================
// // AUTO‑RENEWAL (cron)
// // ============================================

// func (s *SubscriptionService) GenerateRenewalPaymentLink(subscriptionID string) error {
//     log.Printf("[RENEWAL] Generating renewal payment link for subscription %s", subscriptionID)
//     sub, err := s.repo.FindByID(subscriptionID)
//     if err != nil {
//         return err
//     }
//     if sub.Status != models.SubStatusActive {
//         log.Printf("[RENEWAL] Subscription %s is not active, skipping", subscriptionID)
//         return nil
//     }

//     invoice := &models.Invoice{
//         ID:             uuid.New().String(),
//         SubscriptionID: sub.ID,
//         SchoolID:       sub.SchoolID,
//         UserID:         sub.UserID,
//         InvoiceNumber:  dto.GenerateInvoiceNumber(),
//         Amount:         sub.Amount,
//         Total:          sub.Amount,
//         Currency:       sub.Currency,
//         Status:         models.InvoicePending,
//         DueDate:        time.Now().AddDate(0, 0, 7),
//         Items:          map[string]interface{}{"renewal": true},
//         CreatedAt:      time.Now(),
//     }
//     if err := s.repo.CreateInvoice(invoice); err != nil {
//         log.Printf("[RENEWAL] Failed to create renewal invoice: %v", err)
//         return err
//     }

//     email := "customer@example.com" // replace with actual user email

//     // For renewal, we need a redirect URL – use a default success page
//     defaultSuccessURL := "https://yourapp.com/payment/success"
//     paymentReq := &payment.PaymentRequest{
//         SchoolID:    sub.SchoolID,
//         UserID:      sub.UserID,
//         Amount:      sub.Amount,
//         Currency:    sub.Currency,
//         Email:       email,
//         CallbackURL: defaultSuccessURL, // ← Required for Flutterwave
//         Metadata: map[string]interface{}{
//             "subscription_id": sub.ID,
//             "invoice_id":      invoice.ID,
//             "renewal":         true,
//         },
//     }
//     pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
//     if err != nil {
//         log.Printf("[RENEWAL] Failed to create payment for renewal: %v", err)
//         return err
//     }
//     dbPI := &models.PaymentIntent{
//         ID:               uuid.New().String(),
//         SubscriptionID:   sub.ID,
//         InvoiceID:        invoice.ID,
//         SchoolID:         sub.SchoolID,
//         UserID:           sub.UserID,
//         Gateway:          sub.Gateway,
//         Amount:           sub.Amount,
//         Currency:         sub.Currency,
//         Reference:        pi.Reference,
//         AuthorizationURL: pi.AuthorizationURL,
//         Status:           models.IntentPending,
//         ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
//         CreatedAt:        time.Now(),
//     }
//     s.repo.CreatePaymentIntent(dbPI)
//     log.Printf("[RENEWAL] Renewal payment link generated: %s", dbPI.AuthorizationURL)
//     go s.sendRenewalEmail(sub, invoice, dbPI, email)
//     return nil
// }

// // ============================================
// // HELPER FUNCTIONS
// // ============================================

// func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
//     switch interval {
//     case models.IntervalMonthly:
//         return start.AddDate(0, 1, 0)
//     case models.IntervalQuarterly:
//         return start.AddDate(0, 3, 0)
//     case models.IntervalYearly:
//         return start.AddDate(1, 0, 0)
//     default:
//         return start.AddDate(0, 1, 0)
//     }
// }

// func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
//     next := calculateEndDate(currentEnd, interval)
//     return &next
// }

// func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
//     features := make(models.JSONMap)
//     switch tier {
//     case models.TierBasic:
//         features["analytics"] = false
//         features["api_access"] = false
//         features["priority_support"] = false
//         features["bulk_import"] = false
//         features["white_label"] = false
//     case models.TierPremium:
//         features["analytics"] = true
//         features["api_access"] = true
//         features["priority_support"] = false
//         features["bulk_import"] = true
//         features["white_label"] = false
//         features["custom_reports"] = true
//     case models.TierEnterprise:
//         features["analytics"] = true
//         features["api_access"] = true
//         features["priority_support"] = true
//         features["bulk_import"] = true
//         features["white_label"] = true
//         features["custom_reports"] = true
//         features["dedicated_server"] = true
//         features["sla"] = true
//     }
//     return features
// }

// func timePtr(t time.Time) *time.Time {
//     return &t
// }

// // Response mappers
// func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
//     return &dto.SubscriptionResponse{
//         ID:              sub.ID,
//         SchoolID:        sub.SchoolID,
//         Tier:            string(sub.Tier),
//         Status:          string(sub.Status),
//         Gateway:         string(sub.Gateway),
//         Amount:          sub.Amount,
//         Currency:        string(sub.Currency),
//         PaymentInterval: string(sub.PaymentInterval),
//         StartDate:       sub.StartDate,
//         EndDate:         sub.EndDate,
//         TrialEndsAt:     sub.TrialEndsAt,
//         AutoRenew:       sub.AutoRenew,
//         CancelAtEndDate: sub.CancelAtEndDate,
//         LastPaymentDate: sub.LastPaymentDate,
//         NextPaymentDate: sub.NextPaymentDate,
//         MaxStudents:     sub.MaxStudents,
//         MaxTeachers:     sub.MaxTeachers,
//         MaxExams:        sub.MaxExams,
//         MaxQuestions:    sub.MaxQuestions,
//         MaxStorageMB:    sub.MaxStorageMB,
//         Features:        sub.Features,
//         CreatedAt:       sub.CreatedAt,
//         UpdatedAt:       sub.UpdatedAt,
//     }
// }

// func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
//     return &dto.PaymentIntentResponse{
//         ID:               pi.ID,
//         SubscriptionID:   pi.SubscriptionID,
//         InvoiceID:        pi.InvoiceID,
//         Gateway:          string(pi.Gateway),
//         Amount:           pi.Amount,
//         Currency:         string(pi.Currency),
//         Reference:        pi.Reference,
//         ClientSecret:     pi.ClientSecret,
//         AuthorizationURL: pi.AuthorizationURL,
//         AccessCode:       pi.AccessCode,
//         Status:           string(pi.Status),
//         ExpiresAt:        pi.ExpiresAt,
//         PaymentMethod:    string(pi.PaymentMethod),
//     }
// }

// func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
//     return &dto.InvoiceResponse{
//         ID:            inv.ID,
//         InvoiceNumber: inv.InvoiceNumber,
//         Amount:        inv.Amount,
//         Tax:           inv.Tax,
//         Discount:      inv.Discount,
//         Total:         inv.Total,
//         Currency:      string(inv.Currency),
//         Status:        string(inv.Status),
//         DueDate:       inv.DueDate,
//         PaidAt:        inv.PaidAt,
//         PDFURL:        inv.PDFURL,
//         Items:         inv.Items,
//         CreatedAt:     inv.CreatedAt,
//     }
// }

// // Email stubs
// func (s *SubscriptionService) sendPaymentLinkEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
//     log.Printf("[EMAIL] Payment link to %s: %s", email, pi.AuthorizationURL)
// }

// func (s *SubscriptionService) sendRenewalEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
//     log.Printf("[EMAIL] Renewal payment link to %s: %s", email, pi.AuthorizationURL)
// }



// // package service

// // import (
// //     "context"
// //     "errors"
// //     "fmt"
// //     "log"
// //     "time"

// //     "github.com/google/uuid"
// //     // "github.com/shopspring/decimal"

// //     "cbt-api/internal/models"
// //     "cbt-api/internal/subscription/dto"
// //     "cbt-api/internal/subscription/repository"
// //     "cbt-api/pkg/email"
// //     "cbt-api/pkg/payment"
// // )

// // type SubscriptionService struct {
// //     repo           *repository.SubscriptionRepository
// //     paymentService *payment.PaymentService
// //     emailService   *email.EmailService
// // }

// // func NewSubscriptionService(
// //     repo *repository.SubscriptionRepository,
// //     paymentService *payment.PaymentService,
// //     emailService *email.EmailService,
// // ) *SubscriptionService {
// //     return &SubscriptionService{
// //         repo:           repo,
// //         paymentService: paymentService,
// //         emailService:   emailService,
// //     }
// // }

// // // ============================================
// // // CRUD METHODS (required by handler)
// // // ============================================

// // func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
// //     log.Printf("[SERVICE] GetSubscription: id=%s", id)
// //     sub, err := s.repo.FindByID(id)
// //     if err != nil {
// //         log.Printf("[SERVICE] GetSubscription error: %v", err)
// //         return nil, errors.New("subscription not found")
// //     }
// //     return s.toSubscriptionResponse(sub), nil
// // }

// // func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
// //     log.Printf("[SERVICE] GetSubscriptionsBySchool: school=%s", schoolID)
// //     subs, err := s.repo.FindBySchool(schoolID)
// //     if err != nil {
// //         log.Printf("[SERVICE] GetSubscriptionsBySchool error: %v", err)
// //         return nil, err
// //     }
// //     responses := make([]dto.SubscriptionResponse, len(subs))
// //     for i, sub := range subs {
// //         responses[i] = *s.toSubscriptionResponse(&sub)
// //     }
// //     return responses, nil
// // }

// // func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
// //     log.Printf("[SERVICE] GetCurrentSubscription: school=%s", schoolID)
// //     sub, err := s.repo.FindCurrentBySchool(schoolID)
// //     if err != nil {
// //         log.Printf("[SERVICE] GetCurrentSubscription error: %v", err)
// //         return nil, errors.New("no active subscription found")
// //     }
// //     return s.toSubscriptionResponse(sub), nil
// // }

// // func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// //     log.Printf("[SERVICE] UpdateSubscription: id=%s, user=%s", id, userID)
// //     sub, err := s.repo.FindByID(id)
// //     if err != nil {
// //         return nil, errors.New("subscription not found")
// //     }

// //     if req.Tier != nil {
// //         newTier := models.SubscriptionTier(*req.Tier)
// //         sub.Tier = newTier
// //         sub.Amount = models.Pricing[newTier][sub.PaymentInterval]
// //         sub.MaxStudents = models.TierLimits[newTier].MaxStudents
// //         sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
// //         sub.MaxExams = models.TierLimits[newTier].MaxExams
// //         sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
// //         sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
// //         sub.Features = getFeaturesForTier(newTier)
// //     }
// //     if req.AutoRenew != nil {
// //         sub.AutoRenew = *req.AutoRenew
// //     }
// //     if req.CancelAtEndDate != nil {
// //         sub.CancelAtEndDate = *req.CancelAtEndDate
// //     }
// //     if req.Status != nil {
// //         sub.Status = models.SubscriptionStatus(*req.Status)
// //     }
// //     sub.UpdatedAt = time.Now()
// //     sub.UpdatedBy = userID

// //     if err := s.repo.Update(sub); err != nil {
// //         log.Printf("[SERVICE] UpdateSubscription failed: %v", err)
// //         return nil, err
// //     }
// //     log.Printf("[SERVICE] UpdateSubscription success: id=%s", id)
// //     return s.toSubscriptionResponse(sub), nil
// // }

// // func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
// //     log.Printf("[SERVICE] CancelSubscription: id=%s, immediate=%v, reason=%s, user=%s", id, cancelImmediately, reason, userID)
// //     sub, err := s.repo.FindByID(id)
// //     if err != nil {
// //         return errors.New("subscription not found")
// //     }
// //     if cancelImmediately {
// //         sub.Status = models.SubStatusCancelled
// //         // If you ever stored a gateway subscription ID, cancel it (optional)
// //         if sub.GatewaySubscriptionID != "" {
// //             _ = s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
// //         }
// //     } else {
// //         sub.CancelAtEndDate = true
// //     }
// //     sub.UpdatedAt = time.Now()
// //     sub.UpdatedBy = userID
// //     if err := s.repo.Update(sub); err != nil {
// //         log.Printf("[SERVICE] CancelSubscription failed: %v", err)
// //         return err
// //     }
// //     log.Printf("[SERVICE] CancelSubscription success: id=%s", id)
// //     return nil
// // }

// // func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// //     log.Printf("[SERVICE] RenewSubscription: id=%s, interval=%s, user=%s", id, req.Interval, userID)
// //     sub, err := s.repo.FindByID(id)
// //     if err != nil {
// //         return nil, errors.New("subscription not found")
// //     }
// //     newInterval := models.PaymentInterval(req.Interval)
// //     newAmount := models.Pricing[sub.Tier][newInterval]
// //     newEndDate := calculateEndDate(time.Now(), newInterval)

// //     sub.PaymentInterval = newInterval
// //     sub.Amount = newAmount
// //     sub.EndDate = newEndDate
// //     sub.Status = models.SubStatusActive
// //     sub.AutoRenew = true
// //     sub.CancelAtEndDate = false
// //     sub.UpdatedAt = time.Now()
// //     sub.UpdatedBy = userID

// //     if err := s.repo.Update(sub); err != nil {
// //         log.Printf("[SERVICE] RenewSubscription failed: %v", err)
// //         return nil, err
// //     }
// //     log.Printf("[SERVICE] RenewSubscription success: id=%s, new_end=%s", id, newEndDate)
// //     return s.toSubscriptionResponse(sub), nil
// // }

// // // ============================================
// // // PAYMENT INTENT METHODS (one-time payments)
// // // ============================================

// // func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
// //     log.Printf("[SERVICE] CreatePaymentIntent: subscription=%s, user=%s", subscriptionID, userID)
// //     sub, err := s.repo.FindByID(subscriptionID)
// //     if err != nil {
// //         return nil, errors.New("subscription not found")
// //     }

// //     // Create a new invoice for this payment
// //     invoice := &models.Invoice{
// //         ID:             uuid.New().String(),
// //         SubscriptionID: subscriptionID,
// //         SchoolID:       sub.SchoolID,
// //         UserID:         userID,
// //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// //         Amount:         sub.Amount,
// //         Total:          sub.Amount,
// //         Currency:       sub.Currency,
// //         Status:         models.InvoicePending,
// //         DueDate:        time.Now().AddDate(0, 0, 7),
// //         Items:          map[string]interface{}{"payment_type": "manual"},
// //         CreatedAt:      time.Now(),
// //         UpdatedAt:      time.Now(),
// //     }
// //     if err := s.repo.CreateInvoice(invoice); err != nil {
// //         log.Printf("[SERVICE] CreatePaymentIntent: failed to create invoice: %v", err)
// //         return nil, err
// //     }

// //     // Build one‑time payment request – no subscription call
// //     paymentReq := &payment.PaymentRequest{
// //         SchoolID:   sub.SchoolID,
// //         UserID:     userID,
// //         Amount:     sub.Amount,
// //         Currency:   sub.Currency,
// //         Email:      req.SuccessURL, // placeholder – you should fetch the user's email from DB
// //         SuccessURL: req.SuccessURL,
// //         CancelURL:  req.CancelURL,
// //         Metadata: map[string]interface{}{
// //             "subscription_id": subscriptionID,
// //             "invoice_id":      invoice.ID,
// //             "payment_type":    "manual",
// //         },
// //     }

// //     pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
// //     if err != nil {
// //         log.Printf("[SERVICE] CreatePaymentIntent: payment creation failed: %v", err)
// //         return nil, err
// //     }

// //     dbPI := &models.PaymentIntent{
// //         ID:               uuid.New().String(),
// //         SubscriptionID:   subscriptionID,
// //         InvoiceID:        invoice.ID,
// //         SchoolID:         sub.SchoolID,
// //         UserID:           userID,
// //         IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
// //         Gateway:          sub.Gateway,
// //         Amount:           sub.Amount,
// //         Currency:         sub.Currency,
// //         Reference:        pi.Reference,
// //         ClientSecret:     pi.ClientSecret,
// //         AuthorizationURL: pi.AuthorizationURL,
// //         AccessCode:       pi.AccessCode,
// //         Status:           models.IntentPending,
// //         ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
// //         CreatedAt:        time.Now(),
// //         UpdatedAt:        time.Now(),
// //     }
// //     s.repo.CreatePaymentIntent(dbPI)

// //     log.Printf("[SERVICE] CreatePaymentIntent success: payment_intent=%s, url=%s", dbPI.ID, dbPI.AuthorizationURL)
// //     return s.toPaymentIntentResponse(dbPI), nil
// // }

// // func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
// //     log.Printf("[SERVICE] ConfirmPaymentIntent: id=%s", paymentIntentID)
// //     pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
// //     if err != nil {
// //         return errors.New("payment intent not found")
// //     }
// //     if pi.IsFinalized {
// //         return errors.New("payment intent already finalized")
// //     }

// //     // Verify with gateway
// //     verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
// //     if err != nil {
// //         log.Printf("[SERVICE] ConfirmPaymentIntent: verification failed: %v", err)
// //         return err
// //     }

// //     pi.Status = models.PaymentIntentStatus(verification.Status)
// //     pi.GatewayResponse = verification.GatewayData

// //     if verification.Status == "succeeded" {
// //         pi.PaidAt = timePtr(time.Now())
// //         pi.IsFinalized = true
// //         s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())

// //         // Activate subscription if it's still pending
// //         sub, _ := s.repo.FindByID(pi.SubscriptionID)
// //         if sub != nil && sub.Status == models.SubStatusPending {
// //             sub.Status = models.SubStatusActive
// //             sub.StartDate = time.Now()
// //             sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
// //             sub.LastPaymentDate = timePtr(time.Now())
// //             sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// //             s.repo.Update(sub)
// //             log.Printf("[SERVICE] ConfirmPaymentIntent: subscription %s activated", sub.ID)
// //         }

// //         // Record transaction
// //         transaction := &models.PaymentTransaction{
// //             ID:              uuid.New().String(),
// //             SubscriptionID:  pi.SubscriptionID,
// //             PaymentIntentID: pi.ID,
// //             InvoiceID:       pi.InvoiceID,
// //             SchoolID:        pi.SchoolID,
// //             UserID:          pi.UserID,
// //             Amount:          pi.Amount,
// //             Currency:        pi.Currency,
// //             PaymentStatus:   models.PaymentPaid,
// //             Gateway:         pi.Gateway,
// //             Reference:       pi.Reference,
// //             PaidAt:          timePtr(time.Now()),
// //             CreatedAt:       time.Now(),
// //             UpdatedAt:       time.Now(),
// //         }
// //         s.repo.CreatePaymentTransaction(transaction)
// //     }

// //     s.repo.UpdatePaymentIntent(pi)
// //     log.Printf("[SERVICE] ConfirmPaymentIntent: final status=%s", pi.Status)
// //     return nil
// // }

// // // ============================================
// // // CREATE SUBSCRIPTION (one‑time payment link)
// // // ============================================

// // func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
// //     log.Printf("[SERVICE] CreateSubscription: school=%s tier=%s interval=%s gateway=%s user=%s",
// //         schoolID, req.Tier, req.Interval, req.Gateway, userID)

// //     // Check existing active subscription
// //     existing, _ := s.repo.FindCurrentBySchool(schoolID)
// //     if existing != nil {
// //         log.Printf("[SERVICE] CreateSubscription: active subscription already exists for school %s", schoolID)
// //         return nil, errors.New("school already has an active subscription")
// //     }

// //     tier := models.SubscriptionTier(req.Tier)
// //     interval := models.PaymentInterval(req.Interval)
// //     amount, exists := models.Pricing[tier][interval]
// //     if !exists {
// //         log.Printf("[SERVICE] CreateSubscription: invalid pricing for tier=%s interval=%s", req.Tier, req.Interval)
// //         return nil, errors.New("invalid pricing for selected tier and interval")
// //     }

// //     now := time.Now()
// //     subscriptionID := uuid.New().String()
// //     subscription := &models.Subscription{
// //         ID:                subscriptionID,
// //         UserID:            userID,
// //         SchoolID:          schoolID,
// //         Tier:              tier,
// //         Status:            models.SubStatusPending,
// //         Gateway:           models.PaymentGateway(req.Gateway),
// //         Amount:            amount,
// //         Currency:          models.CurrencyNGN,
// //         PaymentInterval:   interval,
// //         StartDate:         now,
// //         EndDate:           now.AddDate(1, 0, 0), // temporary, will be updated after payment
// //         AutoRenew:         req.AutoRenew,
// //         MaxStudents:       models.TierLimits[tier].MaxStudents,
// //         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
// //         MaxExams:          models.TierLimits[tier].MaxExams,
// //         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
// //         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
// //         Features:          getFeaturesForTier(tier),
// //         CreatedAt:         now,
// //         UpdatedAt:         now,
// //     }
// //     if err := s.repo.Create(subscription); err != nil {
// //         log.Printf("[SERVICE] CreateSubscription: failed to create subscription record: %v", err)
// //         return nil, err
// //     }

// //     invoice := &models.Invoice{
// //         ID:             uuid.New().String(),
// //         SubscriptionID: subscriptionID,
// //         SchoolID:       schoolID,
// //         UserID:         userID,
// //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// //         Amount:         amount,
// //         Total:          amount,
// //         Currency:       models.CurrencyNGN,
// //         Status:         models.InvoicePending,
// //         DueDate:        now.AddDate(0, 0, 7),
// //         Items:          map[string]interface{}{"tier": string(tier), "interval": string(interval)},
// //         CreatedAt:      now,
// //         UpdatedAt:      now,
// //     }
// //     if err := s.repo.CreateInvoice(invoice); err != nil {
// //         log.Printf("[SERVICE] CreateSubscription: failed to create invoice: %v", err)
// //         s.repo.Delete(subscriptionID)
// //         return nil, err
// //     }

// //     // Create one‑time payment link – NOT a gateway subscription
// //     paymentReq := &payment.PaymentRequest{
// //         SchoolID:   schoolID,
// //         UserID:     userID,
// //         Amount:     amount,
// //         Currency:   models.CurrencyNGN,
// //         Email:      req.Email,
// //         SuccessURL: req.SuccessURL,
// //         CancelURL:  req.CancelURL,
// //         Metadata: map[string]interface{}{
// //             "subscription_id": subscriptionID,
// //             "invoice_id":      invoice.ID,
// //         },
// //     }
// //     paymentIntent, err := s.paymentService.CreatePayment(ctx, models.PaymentGateway(req.Gateway), paymentReq)
// //     if err != nil {
// //         log.Printf("[SERVICE] CreateSubscription: payment creation failed: %v", err)
// //         s.repo.Delete(subscriptionID)
// //         return nil, fmt.Errorf("failed to create payment link: %w", err)
// //     }

// //     dbPI := &models.PaymentIntent{
// //         ID:               uuid.New().String(),
// //         SubscriptionID:   subscriptionID,
// //         InvoiceID:        invoice.ID,
// //         SchoolID:         schoolID,
// //         UserID:           userID,
// //         IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano()),
// //         Gateway:          models.PaymentGateway(req.Gateway),
// //         Amount:           amount,
// //         Currency:         models.CurrencyNGN,
// //         Reference:        paymentIntent.Reference,
// //         AuthorizationURL: paymentIntent.AuthorizationURL,
// //         Status:           models.IntentPending,
// //         ExpiresAt:        timePtr(now.Add(48 * time.Hour)),
// //         CreatedAt:        now,
// //         UpdatedAt:        now,
// //     }
// //     s.repo.CreatePaymentIntent(dbPI)

// //     // History log
// //     history := &models.SubscriptionHistory{
// //         ID:             uuid.New().String(),
// //         SubscriptionID: subscriptionID,
// //         SchoolID:       schoolID,
// //         UserID:         userID,
// //         NewTier:        tier,
// //         NewStatus:      models.SubStatusPending,
// //         NewAmount:      amount,
// //         ChangeReason:   "subscription_created_pending_payment",
// //         ChangedBy:      userID,
// //         CreatedAt:      now,
// //     }
// //     s.repo.CreateHistory(history)

// //     // Async email with payment link
// //     go s.sendPaymentLinkEmail(subscription, invoice, dbPI, req.Email)

// //     log.Printf("[SERVICE] CreateSubscription success: subscription=%s, payment_link=%s", subscriptionID, dbPI.AuthorizationURL)
// //     return &dto.CreateSubscriptionResponse{
// //         Subscription:  s.toSubscriptionResponse(subscription),
// //         PaymentIntent: s.toPaymentIntentResponse(dbPI),
// //         Invoice:       s.toInvoiceResponse(invoice),
// //     }, nil
// // }

// // // ============================================
// // // WEBHOOK PROCESSING (only payment_success)
// // // ============================================

// // func (s *SubscriptionService) ProcessWebhook(ctx context.Context, gateway models.PaymentGateway, payload []byte, signature string) error {
// //     log.Printf("[WEBHOOK] Received for gateway %s", gateway)
// //     gw, err := s.paymentService.GetGateway(gateway)
// //     if err != nil {
// //         log.Printf("[WEBHOOK] Gateway not found: %v", err)
// //         return err
// //     }
// //     event, err := gw.ParseWebhook(ctx, payload, signature)
// //     if err != nil {
// //         log.Printf("[WEBHOOK] Failed to parse webhook: %v", err)
// //         return err
// //     }

// //     // Store raw event
// //     webhookEvent := &models.WebhookEvent{
// //         ID:             uuid.New().String(),
// //         Gateway:        gateway,
// //         EventType:      event.Type,
// //         Payload:        event.RawData,
// //         IdempotencyKey: event.Reference,
// //         Status:         "pending",
// //         CreatedAt:      time.Now(),
// //     }
// //     s.repo.CreateWebhookEvent(webhookEvent)

// //     if event.Type == "payment_success" {
// //         log.Printf("[WEBHOOK] Payment success for reference %s", event.Reference)
// //         pi, err := s.repo.FindPaymentIntentByReference(event.Reference)
// //         if err != nil {
// //             log.Printf("[WEBHOOK] Payment intent not found for reference %s", event.Reference)
// //             s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "failed", "payment intent not found")
// //             return nil
// //         }
// //         // Activate subscription
// //         sub, _ := s.repo.FindByID(pi.SubscriptionID)
// //         if sub != nil && sub.Status == models.SubStatusPending {
// //             sub.Status = models.SubStatusActive
// //             sub.StartDate = time.Now()
// //             sub.EndDate = calculateEndDate(time.Now(), sub.PaymentInterval)
// //             sub.LastPaymentDate = timePtr(time.Now())
// //             sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// //             s.repo.Update(sub)
// //             s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
// //             log.Printf("[WEBHOOK] Subscription %s activated", sub.ID)
// //         }
// //         s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "processed", "")
// //     } else {
// //         log.Printf("[WEBHOOK] Ignoring event type %s", event.Type)
// //         s.repo.UpdateWebhookEventStatus(webhookEvent.ID, "ignored", "")
// //     }
// //     return nil
// // }

// // // ============================================
// // // AUTO‑RENEWAL (called by cron job)
// // // ============================================

// // func (s *SubscriptionService) GenerateRenewalPaymentLink(subscriptionID string) error {
// //     log.Printf("[RENEWAL] Generating renewal payment link for subscription %s", subscriptionID)
// //     sub, err := s.repo.FindByID(subscriptionID)
// //     if err != nil {
// //         return err
// //     }
// //     if sub.Status != models.SubStatusActive {
// //         log.Printf("[RENEWAL] Subscription %s is not active, skipping", subscriptionID)
// //         return nil
// //     }

// //     // Create new invoice for the next period
// //     invoice := &models.Invoice{
// //         ID:             uuid.New().String(),
// //         SubscriptionID: sub.ID,
// //         SchoolID:       sub.SchoolID,
// //         UserID:         sub.UserID,
// //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// //         Amount:         sub.Amount,
// //         Total:          sub.Amount,
// //         Currency:       sub.Currency,
// //         Status:         models.InvoicePending,
// //         DueDate:        time.Now().AddDate(0, 0, 7),
// //         Items:          map[string]interface{}{"renewal": true},
// //         CreatedAt:      time.Now(),
// //     }
// //     if err := s.repo.CreateInvoice(invoice); err != nil {
// //         log.Printf("[RENEWAL] Failed to create renewal invoice: %v", err)
// //         return err
// //     }

// //     // Fetch user email – replace with actual lookup
// //     email := "customer@example.com"

// //     paymentReq := &payment.PaymentRequest{
// //         SchoolID: sub.SchoolID,
// //         UserID:   sub.UserID,
// //         Amount:   sub.Amount,
// //         Currency: sub.Currency,
// //         Email:    email,
// //         Metadata: map[string]interface{}{
// //             "subscription_id": sub.ID,
// //             "invoice_id":      invoice.ID,
// //             "renewal":         true,
// //         },
// //     }
// //     pi, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
// //     if err != nil {
// //         log.Printf("[RENEWAL] Failed to create payment for renewal: %v", err)
// //         return err
// //     }
// //     dbPI := &models.PaymentIntent{
// //         ID:               uuid.New().String(),
// //         SubscriptionID:   sub.ID,
// //         InvoiceID:        invoice.ID,
// //         SchoolID:         sub.SchoolID,
// //         UserID:           sub.UserID,
// //         Gateway:          sub.Gateway,
// //         Amount:           sub.Amount,
// //         Currency:         sub.Currency,
// //         Reference:        pi.Reference,
// //         AuthorizationURL: pi.AuthorizationURL,
// //         Status:           models.IntentPending,
// //         ExpiresAt:        timePtr(time.Now().Add(48 * time.Hour)),
// //         CreatedAt:        time.Now(),
// //     }
// //     s.repo.CreatePaymentIntent(dbPI)
// //     log.Printf("[RENEWAL] Renewal payment link generated: %s", dbPI.AuthorizationURL)
// //     go s.sendRenewalEmail(sub, invoice, dbPI, email)
// //     return nil
// // }

// // // ============================================
// // // HELPER FUNCTIONS (shared)
// // // ============================================

// // func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
// //     switch interval {
// //     case models.IntervalMonthly:
// //         return start.AddDate(0, 1, 0)
// //     case models.IntervalQuarterly:
// //         return start.AddDate(0, 3, 0)
// //     case models.IntervalYearly:
// //         return start.AddDate(1, 0, 0)
// //     default:
// //         return start.AddDate(0, 1, 0)
// //     }
// // }

// // func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
// //     next := calculateEndDate(currentEnd, interval)
// //     return &next
// // }

// // func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
// //     features := make(models.JSONMap)
// //     switch tier {
// //     case models.TierBasic:
// //         features["analytics"] = false
// //         features["api_access"] = false
// //         features["priority_support"] = false
// //         features["bulk_import"] = false
// //         features["white_label"] = false
// //     case models.TierPremium:
// //         features["analytics"] = true
// //         features["api_access"] = true
// //         features["priority_support"] = false
// //         features["bulk_import"] = true
// //         features["white_label"] = false
// //         features["custom_reports"] = true
// //     case models.TierEnterprise:
// //         features["analytics"] = true
// //         features["api_access"] = true
// //         features["priority_support"] = true
// //         features["bulk_import"] = true
// //         features["white_label"] = true
// //         features["custom_reports"] = true
// //         features["dedicated_server"] = true
// //         features["sla"] = true
// //     }
// //     return features
// // }

// // func timePtr(t time.Time) *time.Time {
// //     return &t
// // }

// // // ============================================
// // // RESPONSE MAPPERS (convert models to DTOs)
// // // ============================================

// // func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
// //     return &dto.SubscriptionResponse{
// //         ID:              sub.ID,
// //         SchoolID:        sub.SchoolID,
// //         Tier:            string(sub.Tier),
// //         Status:          string(sub.Status),
// //         Gateway:         string(sub.Gateway),
// //         Amount:          sub.Amount,
// //         Currency:        string(sub.Currency),
// //         PaymentInterval: string(sub.PaymentInterval),
// //         StartDate:       sub.StartDate,
// //         EndDate:         sub.EndDate,
// //         TrialEndsAt:     sub.TrialEndsAt,
// //         AutoRenew:       sub.AutoRenew,
// //         CancelAtEndDate: sub.CancelAtEndDate,
// //         LastPaymentDate: sub.LastPaymentDate,
// //         NextPaymentDate: sub.NextPaymentDate,
// //         MaxStudents:     sub.MaxStudents,
// //         MaxTeachers:     sub.MaxTeachers,
// //         MaxExams:        sub.MaxExams,
// //         MaxQuestions:    sub.MaxQuestions,
// //         MaxStorageMB:    sub.MaxStorageMB,
// //         Features:        sub.Features,
// //         CreatedAt:       sub.CreatedAt,
// //         UpdatedAt:       sub.UpdatedAt,
// //     }
// // }

// // func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
// //     return &dto.PaymentIntentResponse{
// //         ID:               pi.ID,
// //         SubscriptionID:   pi.SubscriptionID,
// //         InvoiceID:        pi.InvoiceID,
// //         Gateway:          string(pi.Gateway),
// //         Amount:           pi.Amount,
// //         Currency:         string(pi.Currency),
// //         Reference:        pi.Reference,
// //         ClientSecret:     pi.ClientSecret,
// //         AuthorizationURL: pi.AuthorizationURL,
// //         AccessCode:       pi.AccessCode,
// //         Status:           string(pi.Status),
// //         ExpiresAt:        pi.ExpiresAt,
// //         PaymentMethod:    string(pi.PaymentMethod),
// //     }
// // }

// // func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
// //     return &dto.InvoiceResponse{
// //         ID:            inv.ID,
// //         InvoiceNumber: inv.InvoiceNumber,
// //         Amount:        inv.Amount,
// //         Tax:           inv.Tax,
// //         Discount:      inv.Discount,
// //         Total:         inv.Total,
// //         Currency:      string(inv.Currency),
// //         Status:        string(inv.Status),
// //         DueDate:       inv.DueDate,
// //         PaidAt:        inv.PaidAt,
// //         PDFURL:        inv.PDFURL,
// //         Items:         inv.Items,
// //         CreatedAt:     inv.CreatedAt,
// //     }
// // }

// // // ============================================
// // // EMAIL STUBS (replace with real email sending)
// // // ============================================

// // func (s *SubscriptionService) sendPaymentLinkEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
// //     log.Printf("[EMAIL] Payment link to %s: %s", email, pi.AuthorizationURL)
// // }

// // func (s *SubscriptionService) sendRenewalEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
// //     log.Printf("[EMAIL] Renewal payment link to %s: %s", email, pi.AuthorizationURL)
// // }






// // // package service

// // // import (
// // //     "context"
// // //     "errors"
// // //     "fmt"
// // //     "time"

// // //     "github.com/google/uuid"
// // //     "github.com/shopspring/decimal"

// // //     "cbt-api/internal/models"
// // //     "cbt-api/internal/subscription/dto"
// // //     "cbt-api/internal/subscription/repository"
// // //     "cbt-api/pkg/email"
// // //     "cbt-api/pkg/payment"
// // // )

// // // type SubscriptionService struct {
// // //     repo           *repository.SubscriptionRepository
// // //     paymentService *payment.PaymentService
// // //     emailService   *email.EmailService
// // // }

// // // func NewSubscriptionService(
// // //     repo *repository.SubscriptionRepository,
// // //     paymentService *payment.PaymentService,
// // //     emailService *email.EmailService,
// // // ) *SubscriptionService {
// // //     return &SubscriptionService{
// // //         repo:           repo,
// // //         paymentService: paymentService,
// // //         emailService:   emailService,
// // //     }
// // // }

// // // // ============================================
// // // // SUBSCRIPTION MANAGEMENT
// // // // ============================================

// // // func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
// // //     // Check for existing active subscription
// // //     existing, _ := s.repo.FindCurrentBySchool(schoolID)
// // //     if existing != nil {
// // //         return nil, errors.New("school already has an active subscription")
// // //     }

// // //     // Get pricing
// // //     tier := models.SubscriptionTier(req.Tier)
// // //     interval := models.PaymentInterval(req.Interval)
// // //     amount, exists := models.Pricing[tier][interval]
// // //     if !exists {
// // //         return nil, errors.New("invalid pricing for selected tier and interval")
// // //     }

// // //     // Calculate dates
// // //     now := time.Now()
// // //     endDate := calculateEndDate(now, interval)
// // //     trialEndsAt := now.AddDate(0, 0, 7) // 7 days trial

// // //     // Create subscription record
// // //     subscriptionID := uuid.New().String()
// // //     subscription := &models.Subscription{
// // //         ID:                subscriptionID,
// // //         UserID:            userID,
// // //         SchoolID:          schoolID,
// // //         Tier:              tier,
// // //         Status:            models.SubStatusTrial,
// // //         Gateway:           models.PaymentGateway(req.Gateway),
// // //         Amount:            amount,
// // //         Currency:          models.CurrencyNGN,
// // //         PaymentInterval:   interval,
// // //         StartDate:         now,
// // //         EndDate:           endDate,
// // //         TrialEndsAt:       &trialEndsAt,
// // //         AutoRenew:         req.AutoRenew,
// // //         MaxStudents:       models.TierLimits[tier].MaxStudents,
// // //         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
// // //         MaxExams:          models.TierLimits[tier].MaxExams,
// // //         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
// // //         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
// // //         Features:          getFeaturesForTier(tier),
// // //         CreatedAt:         now,
// // //         UpdatedAt:         now,
// // //     }

// // //     if err := s.repo.Create(subscription); err != nil {
// // //         return nil, err
// // //     }

// // //     // Create invoice
// // //     invoice := &models.Invoice{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: subscriptionID,
// // //         SchoolID:       schoolID,
// // //         UserID:         userID,
// // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // //         Amount:         amount,
// // //         Tax:            decimal.NewFromFloat(0),
// // //         Discount:       decimal.NewFromFloat(0),
// // //         Total:          amount,
// // //         Currency:       models.CurrencyNGN,
// // //         Status:         models.InvoicePending,
// // //         DueDate:        now.AddDate(0, 0, 7),
// // //         Items: map[string]interface{}{
// // //             "tier":     string(tier),
// // //             "interval": string(interval),
// // //         },
// // //         CreatedAt: now,
// // //         UpdatedAt: now,
// // //     }

// // //     if err := s.repo.CreateInvoice(invoice); err != nil {
// // //         return nil, err
// // //     }

// // //     // Idempotency key for the gateway call
// // //     idempotencyKey := fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano())

// // //     // ============================================================
// // //     // Create recurring subscription on the payment gateway
// // //     // ============================================================
// // //     subReq := &payment.SubscriptionRequest{
// // //         SchoolID:     schoolID,
// // //         UserID:       userID,
// // //         Tier:         tier,
// // //         Interval:     interval,
// // //         Email:        req.Email,
// // //         CustomerName: "", // optionally fetch from user
// // //         SuccessURL:   req.SuccessURL,
// // //         CancelURL:    req.CancelURL,
// // //     }

// // //     gatewayResult, err := s.paymentService.CreateSubscription(ctx, models.PaymentGateway(req.Gateway), subReq)
// // //     if err != nil {
// // //         // mark subscription as failed – use SubStatusCancelled (or you could add SubStatusFailed to models)
// // //         subscription.Status = models.SubStatusCancelled
// // //         s.repo.Update(subscription)
// // //         return nil, fmt.Errorf("gateway subscription creation failed: %w", err)
// // //     }

// // //     // Store gateway subscription ID on local record
// // //     subscription.GatewaySubscriptionID = gatewayResult.GatewaySubscriptionID
// // //     subscription.GatewayCustomerID = gatewayResult.GatewayCustomerID
// // //     s.repo.Update(subscription)

// // //     // Build payment intent record from gateway result
// // //     dbPaymentIntent := &models.PaymentIntent{
// // //         ID:               uuid.New().String(),
// // //         SubscriptionID:   subscriptionID,
// // //         InvoiceID:        invoice.ID,
// // //         SchoolID:         schoolID,
// // //         UserID:           userID,
// // //         IdempotencyKey:   idempotencyKey,
// // //         Gateway:          models.PaymentGateway(req.Gateway),
// // //         Amount:           amount,
// // //         Currency:         models.CurrencyNGN,
// // //         Reference:        gatewayResult.SubscriptionID,
// // //         ClientSecret:     gatewayResult.ClientSecret,
// // //         AuthorizationURL: gatewayResult.AuthorizationURL,
// // //         Status:           models.IntentPending,
// // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // //         CreatedAt:        now,
// // //         UpdatedAt:        now,
// // //     }

// // //     if err := s.repo.CreatePaymentIntent(dbPaymentIntent); err != nil {
// // //         return nil, err
// // //     }

// // //     // Create history record
// // //     history := &models.SubscriptionHistory{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: subscriptionID,
// // //         SchoolID:       schoolID,
// // //         UserID:         userID,
// // //         NewTier:        tier,
// // //         NewStatus:      models.SubStatusTrial,
// // //         NewAmount:      amount,
// // //         ChangeReason:   "subscription_created",
// // //         ChangedBy:      userID,
// // //         Metadata: map[string]interface{}{
// // //             "gateway":           string(req.Gateway),
// // //             "payment_intent_id": dbPaymentIntent.ID,
// // //         },
// // //         CreatedAt: now,
// // //     }
// // //     s.repo.CreateHistory(history)

// // //     // Send email notification
// // //     go s.sendSubscriptionCreatedEmail(subscription, invoice, dbPaymentIntent, req.Email)

// // //     // Create reminder schedules
// // //     s.createReminderSchedules(subscriptionID)

// // //     return &dto.CreateSubscriptionResponse{
// // //         Subscription:  s.toSubscriptionResponse(subscription),
// // //         PaymentIntent: s.toPaymentIntentResponse(dbPaymentIntent),
// // //         Invoice:       s.toInvoiceResponse(invoice),
// // //     }, nil
// // // }

// // // func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
// // //     sub, err := s.repo.FindByID(id)
// // //     if err != nil {
// // //         return nil, errors.New("subscription not found")
// // //     }
// // //     return s.toSubscriptionResponse(sub), nil
// // // }

// // // func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
// // //     subs, err := s.repo.FindBySchool(schoolID)
// // //     if err != nil {
// // //         return nil, err
// // //     }

// // //     responses := make([]dto.SubscriptionResponse, len(subs))
// // //     for i, sub := range subs {
// // //         responses[i] = *s.toSubscriptionResponse(&sub)
// // //     }
// // //     return responses, nil
// // // }

// // // func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
// // //     sub, err := s.repo.FindCurrentBySchool(schoolID)
// // //     if err != nil {
// // //         return nil, errors.New("no active subscription found")
// // //     }
// // //     return s.toSubscriptionResponse(sub), nil
// // // }

// // // func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // //     sub, err := s.repo.FindByID(id)
// // //     if err != nil {
// // //         return nil, errors.New("subscription not found")
// // //     }

// // //     oldStatus := sub.Status
// // //     oldTier := sub.Tier
// // //     oldAmount := sub.Amount

// // //     if req.Tier != nil {
// // //         newTier := models.SubscriptionTier(*req.Tier)
// // //         newAmount := models.Pricing[newTier][sub.PaymentInterval]

// // //         sub.Tier = newTier
// // //         sub.Amount = newAmount
// // //         sub.MaxStudents = models.TierLimits[newTier].MaxStudents
// // //         sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
// // //         sub.MaxExams = models.TierLimits[newTier].MaxExams
// // //         sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
// // //         sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
// // //         sub.Features = getFeaturesForTier(newTier)
// // //     }

// // //     if req.AutoRenew != nil {
// // //         sub.AutoRenew = *req.AutoRenew
// // //     }

// // //     if req.CancelAtEndDate != nil {
// // //         sub.CancelAtEndDate = *req.CancelAtEndDate
// // //     }

// // //     if req.Status != nil {
// // //         sub.Status = models.SubscriptionStatus(*req.Status)
// // //     }

// // //     sub.UpdatedAt = time.Now()
// // //     sub.UpdatedBy = userID

// // //     if err := s.repo.Update(sub); err != nil {
// // //         return nil, err
// // //     }

// // //     // Create history record if important fields changed
// // //     if oldTier != sub.Tier || oldStatus != sub.Status || !oldAmount.Equal(sub.Amount) {
// // //         history := &models.SubscriptionHistory{
// // //             ID:             uuid.New().String(),
// // //             SubscriptionID: sub.ID,
// // //             SchoolID:       sub.SchoolID,
// // //             UserID:         userID,
// // //             OldTier:        oldTier,
// // //             NewTier:        sub.Tier,
// // //             OldStatus:      oldStatus,
// // //             NewStatus:      sub.Status,
// // //             OldAmount:      oldAmount,
// // //             NewAmount:      sub.Amount,
// // //             ChangeReason:   "subscription_updated",
// // //             ChangedBy:      userID,
// // //             CreatedAt:      time.Now(),
// // //         }
// // //         s.repo.CreateHistory(history)
// // //     }

// // //     return s.toSubscriptionResponse(sub), nil
// // // }

// // // func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
// // //     sub, err := s.repo.FindByID(id)
// // //     if err != nil {
// // //         return errors.New("subscription not found")
// // //     }

// // //     if cancelImmediately {
// // //         sub.Status = models.SubStatusCancelled

// // //         // Cancel at gateway
// // //         s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
// // //     } else {
// // //         sub.CancelAtEndDate = true
// // //     }

// // //     sub.UpdatedAt = time.Now()
// // //     sub.UpdatedBy = userID

// // //     if err := s.repo.Update(sub); err != nil {
// // //         return err
// // //     }

// // //     // Create history record
// // //     history := &models.SubscriptionHistory{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: sub.ID,
// // //         SchoolID:       sub.SchoolID,
// // //         UserID:         userID,
// // //         OldStatus:      sub.Status,
// // //         NewStatus:      models.SubStatusCancelled,
// // //         ChangeReason:   reason,
// // //         ChangedBy:      userID,
// // //         Metadata: map[string]interface{}{
// // //             "cancel_immediately": cancelImmediately,
// // //         },
// // //         CreatedAt: time.Now(),
// // //     }
// // //     s.repo.CreateHistory(history)

// // //     // Send cancellation email
// // //     go s.sendSubscriptionCancelledEmail(sub)

// // //     return nil
// // // }

// // // func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // //     sub, err := s.repo.FindByID(id)
// // //     if err != nil {
// // //         return nil, errors.New("subscription not found")
// // //     }

// // //     newInterval := models.PaymentInterval(req.Interval)
// // //     newAmount := models.Pricing[sub.Tier][newInterval]
// // //     newEndDate := calculateEndDate(time.Now(), newInterval)

// // //     sub.PaymentInterval = newInterval
// // //     sub.Amount = newAmount
// // //     sub.EndDate = newEndDate
// // //     sub.Status = models.SubStatusActive
// // //     sub.AutoRenew = true
// // //     sub.CancelAtEndDate = false
// // //     sub.UpdatedAt = time.Now()
// // //     sub.UpdatedBy = userID

// // //     if err := s.repo.Update(sub); err != nil {
// // //         return nil, err
// // //     }

// // //     // Create new invoice
// // //     invoice := &models.Invoice{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: sub.ID,
// // //         SchoolID:       sub.SchoolID,
// // //         UserID:         userID,
// // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // //         Amount:         newAmount,
// // //         Total:          newAmount,
// // //         Currency:       models.CurrencyNGN,
// // //         Status:         models.InvoicePending,
// // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // //         Items: map[string]interface{}{
// // //             "renewal":      true,
// // //             "old_interval": string(sub.PaymentInterval),
// // //             "new_interval": string(newInterval),
// // //         },
// // //         CreatedAt: time.Now(),
// // //         UpdatedAt: time.Now(),
// // //     }
// // //     s.repo.CreateInvoice(invoice)

// // //     // Create history record
// // //     history := &models.SubscriptionHistory{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: sub.ID,
// // //         SchoolID:       sub.SchoolID,
// // //         UserID:         userID,
// // //         NewAmount:      newAmount,
// // //         ChangeReason:   "subscription_renewed",
// // //         ChangedBy:      userID,
// // //         Metadata: map[string]interface{}{
// // //             "new_interval": string(newInterval),
// // //             "invoice_id":   invoice.ID,
// // //         },
// // //         CreatedAt: time.Now(),
// // //     }
// // //     s.repo.CreateHistory(history)

// // //     // Send renewal email
// // //     go s.sendSubscriptionRenewedEmail(sub, invoice)

// // //     return s.toSubscriptionResponse(sub), nil
// // // }

// // // // ============================================
// // // // PAYMENT INTENT MANAGEMENT
// // // // ============================================

// // // func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
// // //     sub, err := s.repo.FindByID(subscriptionID)
// // //     if err != nil {
// // //         return nil, errors.New("subscription not found")
// // //     }

// // //     // Create invoice for this payment
// // //     invoice := &models.Invoice{
// // //         ID:             uuid.New().String(),
// // //         SubscriptionID: subscriptionID,
// // //         SchoolID:       sub.SchoolID,
// // //         UserID:         userID,
// // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // //         Amount:         sub.Amount,
// // //         Total:          sub.Amount,
// // //         Currency:       sub.Currency,
// // //         Status:         models.InvoicePending,
// // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // //         CreatedAt:      time.Now(),
// // //         UpdatedAt:      time.Now(),
// // //     }
// // //     s.repo.CreateInvoice(invoice)

// // //     // Create payment intent
// // //     idempotencyKey := fmt.Sprintf("%s_%d_%d", subscriptionID, time.Now().UnixNano(), invoice.ID)
// // //     paymentReq := &payment.PaymentRequest{
// // //         SchoolID:    sub.SchoolID,
// // //         UserID:      userID,
// // //         Amount:      sub.Amount,
// // //         Currency:    sub.Currency,
// // //         Email:       req.SuccessURL,
// // //         SuccessURL:  req.SuccessURL,
// // //         CancelURL:   req.CancelURL,
// // //         Metadata: map[string]interface{}{
// // //             "subscription_id": subscriptionID,
// // //             "invoice_id":      invoice.ID,
// // //             "payment_type":    "renewal",
// // //         },
// // //     }

// // //     paymentIntent, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
// // //     if err != nil {
// // //         return nil, err
// // //     }

// // //     dbPaymentIntent := &models.PaymentIntent{
// // //         ID:               uuid.New().String(),
// // //         SubscriptionID:   subscriptionID,
// // //         InvoiceID:        invoice.ID,
// // //         SchoolID:         sub.SchoolID,
// // //         UserID:           userID,
// // //         IdempotencyKey:   idempotencyKey,
// // //         Gateway:          sub.Gateway,
// // //         Amount:           sub.Amount,
// // //         Currency:         sub.Currency,
// // //         Reference:        paymentIntent.Reference,
// // //         ClientSecret:     paymentIntent.ClientSecret,
// // //         AuthorizationURL: paymentIntent.AuthorizationURL,
// // //         Status:           models.IntentPending,
// // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // //         CreatedAt:        time.Now(),
// // //         UpdatedAt:        time.Now(),
// // //     }
// // //     s.repo.CreatePaymentIntent(dbPaymentIntent)

// // //     return s.toPaymentIntentResponse(dbPaymentIntent), nil
// // // }

// // // func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
// // //     pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
// // //     if err != nil {
// // //         return errors.New("payment intent not found")
// // //     }

// // //     if pi.IsFinalized {
// // //         return errors.New("payment intent already finalized")
// // //     }

// // //     // Verify with gateway
// // //     verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
// // //     if err != nil {
// // //         return err
// // //     }

// // //     pi.Status = models.PaymentIntentStatus(verification.Status)
// // //     pi.GatewayResponse = verification.GatewayData

// // //     if verification.Status == "succeeded" {
// // //         pi.PaidAt = timePtr(time.Now())
// // //         pi.IsFinalized = true

// // //         // Mark invoice as paid
// // //         s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())

// // //         // Update subscription
// // //         sub, _ := s.repo.FindByID(pi.SubscriptionID)
// // //         sub.Status = models.SubStatusActive
// // //         sub.LastPaymentDate = timePtr(time.Now())
// // //         sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// // //         s.repo.Update(sub)

// // //         // Create payment transaction
// // //         transaction := &models.PaymentTransaction{
// // //             ID:               uuid.New().String(),
// // //             SubscriptionID:   pi.SubscriptionID,
// // //             PaymentIntentID:  pi.ID,
// // //             InvoiceID:        pi.InvoiceID,
// // //             SchoolID:         pi.SchoolID,
// // //             UserID:           pi.UserID,
// // //             Amount:           pi.Amount,
// // //             Currency:         pi.Currency,
// // //             PaymentMethod:    models.MethodCard,
// // //             PaymentStatus:    models.PaymentPaid,
// // //             Gateway:          pi.Gateway,
// // //             Reference:        pi.Reference,
// // //             PaidAt:           timePtr(time.Now()),
// // //             CreatedAt:        time.Now(),
// // //             UpdatedAt:        time.Now(),
// // //         }
// // //         s.repo.CreatePaymentTransaction(transaction)

// // //         // Send payment success email
// // //         go s.sendPaymentSuccessEmail(sub, transaction)
// // //     }

// // //     s.repo.UpdatePaymentIntent(pi)

// // //     // Create event log
// // //     eventLog := &models.PaymentEventLog{
// // //         ID:              uuid.New().String(),
// // //         PaymentIntentID: pi.ID,
// // //         EventType:       "payment_intent_confirmed",
// // //         StatusBefore:    string(models.IntentPending),
// // //         StatusAfter:     string(pi.Status),
// // //         Payload: map[string]interface{}{
// // //             "verification": verification,
// // //         },
// // //         CreatedAt: time.Now(),
// // //     }
// // //     s.repo.CreatePaymentEventLog(eventLog)

// // //     return nil
// // // }

// // // // ============================================
// // // // HELPER FUNCTIONS
// // // // ============================================

// // // func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
// // //     switch interval {
// // //     case models.IntervalMonthly:
// // //         return start.AddDate(0, 1, 0)
// // //     case models.IntervalQuarterly:
// // //         return start.AddDate(0, 3, 0)
// // //     case models.IntervalYearly:
// // //         return start.AddDate(1, 0, 0)
// // //     default:
// // //         return start.AddDate(0, 1, 0)
// // //     }
// // // }

// // // func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
// // //     next := calculateEndDate(currentEnd, interval)
// // //     return &next
// // // }

// // // func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
// // //     features := make(models.JSONMap)
// // //     switch tier {
// // //     case models.TierBasic:
// // //         features["analytics"] = false
// // //         features["api_access"] = false
// // //         features["priority_support"] = false
// // //         features["bulk_import"] = false
// // //         features["white_label"] = false
// // //     case models.TierPremium:
// // //         features["analytics"] = true
// // //         features["api_access"] = true
// // //         features["priority_support"] = false
// // //         features["bulk_import"] = true
// // //         features["white_label"] = false
// // //         features["custom_reports"] = true
// // //     case models.TierEnterprise:
// // //         features["analytics"] = true
// // //         features["api_access"] = true
// // //         features["priority_support"] = true
// // //         features["bulk_import"] = true
// // //         features["white_label"] = true
// // //         features["custom_reports"] = true
// // //         features["dedicated_server"] = true
// // //         features["sla"] = true
// // //     }
// // //     return features
// // // }

// // // func (s *SubscriptionService) createReminderSchedules(subscriptionID string) {
// // //     reminderDays := []int{30, 14, 7, 3, 1}
// // //     for _, days := range reminderDays {
// // //         schedule := &models.ReminderSchedule{
// // //             ID:             uuid.New().String(),
// // //             SubscriptionID: subscriptionID,
// // //             ReminderType:   "expiry",
// // //             DaysBefore:     days,
// // //             Status:         "pending",
// // //             CreatedAt:      time.Now(),
// // //             UpdatedAt:      time.Now(),
// // //         }
// // //         s.repo.CreateReminderSchedule(schedule)
// // //     }
// // // }

// // // func timePtr(t time.Time) *time.Time {
// // //     return &t
// // // }

// // // // ============================================
// // // // RESPONSE MAPPERS
// // // // ============================================

// // // func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
// // //     return &dto.SubscriptionResponse{
// // //         ID:              sub.ID,
// // //         SchoolID:        sub.SchoolID,
// // //         Tier:            string(sub.Tier),
// // //         Status:          string(sub.Status),
// // //         Gateway:         string(sub.Gateway),
// // //         Amount:          sub.Amount,
// // //         Currency:        string(sub.Currency),
// // //         PaymentInterval: string(sub.PaymentInterval),
// // //         StartDate:       sub.StartDate,
// // //         EndDate:         sub.EndDate,
// // //         TrialEndsAt:     sub.TrialEndsAt,
// // //         AutoRenew:       sub.AutoRenew,
// // //         CancelAtEndDate: sub.CancelAtEndDate,
// // //         LastPaymentDate: sub.LastPaymentDate,
// // //         NextPaymentDate: sub.NextPaymentDate,
// // //         MaxStudents:     sub.MaxStudents,
// // //         MaxTeachers:     sub.MaxTeachers,
// // //         MaxExams:        sub.MaxExams,
// // //         MaxQuestions:    sub.MaxQuestions,
// // //         MaxStorageMB:    sub.MaxStorageMB,
// // //         Features:        sub.Features,
// // //         CreatedAt:       sub.CreatedAt,
// // //         UpdatedAt:       sub.UpdatedAt,
// // //     }
// // // }

// // // func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
// // //     return &dto.PaymentIntentResponse{
// // //         ID:               pi.ID,
// // //         SubscriptionID:   pi.SubscriptionID,
// // //         InvoiceID:        pi.InvoiceID,
// // //         Gateway:          string(pi.Gateway),
// // //         Amount:           pi.Amount,
// // //         Currency:         string(pi.Currency),
// // //         Reference:        pi.Reference,
// // //         ClientSecret:     pi.ClientSecret,
// // //         AuthorizationURL: pi.AuthorizationURL,
// // //         AccessCode:       pi.AccessCode,
// // //         Status:           string(pi.Status),
// // //         ExpiresAt:        pi.ExpiresAt,
// // //         PaymentMethod:    string(pi.PaymentMethod),
// // //     }
// // // }

// // // func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
// // //     return &dto.InvoiceResponse{
// // //         ID:            inv.ID,
// // //         InvoiceNumber: inv.InvoiceNumber,
// // //         Amount:        inv.Amount,
// // //         Tax:           inv.Tax,
// // //         Discount:      inv.Discount,
// // //         Total:         inv.Total,
// // //         Currency:      string(inv.Currency),
// // //         Status:        string(inv.Status),
// // //         DueDate:       inv.DueDate,
// // //         PaidAt:        inv.PaidAt,
// // //         PDFURL:        inv.PDFURL,
// // //         Items:         inv.Items,
// // //         CreatedAt:     inv.CreatedAt,
// // //     }
// // // }

// // // // ============================================
// // // // EMAIL NOTIFICATIONS (To be implemented)
// // // // ============================================

// // // func (s *SubscriptionService) sendSubscriptionCreatedEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
// // //     fmt.Printf("Subscription created email sent to %s\n", email)
// // // }

// // // func (s *SubscriptionService) sendSubscriptionRenewedEmail(sub *models.Subscription, inv *models.Invoice) {
// // //     fmt.Printf("Subscription renewed email sent\n")
// // // }

// // // func (s *SubscriptionService) sendSubscriptionCancelledEmail(sub *models.Subscription) {
// // //     fmt.Printf("Subscription cancelled email sent\n")
// // // }

// // // func (s *SubscriptionService) sendPaymentSuccessEmail(sub *models.Subscription, transaction *models.PaymentTransaction) {
// // //     fmt.Printf("Payment success email sent\n")
// // // }

// // // // ============================================
// // // // WEBHOOK PROCESSING (corrected)
// // // // ============================================

// // // func (s *SubscriptionService) ProcessWebhook(ctx context.Context, gateway models.PaymentGateway, payload []byte, signature string) error {
// // //     gw, err := s.paymentService.GetGateway(gateway)
// // //     if err != nil {
// // //         return err
// // //     }
// // //     event, err := gw.ParseWebhook(ctx, payload, signature)
// // //     if err != nil {
// // //         return err
// // //     }

// // //     // Log webhook event (store in DB)
// // //     webhookEvent := &models.WebhookEvent{
// // //         ID:             uuid.New().String(),
// // //         Gateway:        gateway,
// // //         EventType:      event.Type,
// // //         Payload:        event.RawData,
// // //         IdempotencyKey: event.Reference,
// // //         Status:         models.WebhookPending,   // corrected constant
// // //         CreatedAt:      time.Now(),
// // //         // Signature field removed (not in model)
// // //         // UpdatedAt field removed (not in model)
// // //     }
// // //     s.repo.CreateWebhookEvent(webhookEvent)

// // //     switch event.Type {
// // //     case "subscription_activated":
// // //         sub, err := s.repo.FindByGatewaySubscriptionID(event.GatewaySubscriptionID)
// // //         if err != nil {
// // //             return fmt.Errorf("subscription not found for gateway ID %s", event.GatewaySubscriptionID)
// // //         }
// // //         if sub.Status == models.SubStatusTrial || sub.Status == models.SubStatusPending {
// // //             now := time.Now()
// // //             sub.Status = models.SubStatusActive
// // //             sub.StartDate = now
// // //             sub.EndDate = calculateEndDate(now, sub.PaymentInterval)
// // //             sub.UpdatedAt = now
// // //             s.repo.Update(sub)

// // //             invoices, _ := s.repo.FindInvoicesBySubscription(sub.ID)
// // //             if len(invoices) > 0 {
// // //                 s.repo.MarkInvoiceAsPaid(invoices[0].ID, now)
// // //             }
// // //         }
// // //     case "payment_success":
// // //         pi, err := s.repo.FindPaymentIntentByReference(event.Reference)
// // //         if err == nil && pi != nil {
// // //             s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
// // //             sub, _ := s.repo.FindByID(pi.SubscriptionID)
// // //             if sub != nil {
// // //                 sub.LastPaymentDate = timePtr(time.Now())
// // //                 s.repo.Update(sub)
// // //             }
// // //         }
// // //     case "subscription_cancelled":
// // //         sub, err := s.repo.FindByGatewaySubscriptionID(event.GatewaySubscriptionID)
// // //         if err == nil {
// // //             sub.Status = models.SubStatusCancelled
// // //             s.repo.Update(sub)
// // //         }
// // //     }

// // //     // Update webhook event status
// // //     s.repo.UpdateWebhookEventStatus(webhookEvent.ID, models.WebhookProcessed, "")
// // //     return nil
// // // }

// // // // package service

// // // // import (
// // // //     "context"
// // // //     "errors"
// // // //     "fmt"
// // // //     "time"

// // // //     "github.com/google/uuid"
// // // //     "github.com/shopspring/decimal"

// // // //     "cbt-api/internal/models"
// // // //     "cbt-api/internal/subscription/dto"
// // // //     "cbt-api/internal/subscription/repository"
// // // //     "cbt-api/pkg/email"
// // // //     "cbt-api/pkg/payment"
// // // // )

// // // // type SubscriptionService struct {
// // // //     repo           *repository.SubscriptionRepository
// // // //     paymentService *payment.PaymentService
// // // //     emailService   *email.EmailService
// // // // }

// // // // func NewSubscriptionService(
// // // //     repo *repository.SubscriptionRepository,
// // // //     paymentService *payment.PaymentService,
// // // //     emailService *email.EmailService,
// // // // ) *SubscriptionService {
// // // //     return &SubscriptionService{
// // // //         repo:           repo,
// // // //         paymentService: paymentService,
// // // //         emailService:   emailService,
// // // //     }
// // // // }

// // // // // ============================================
// // // // // SUBSCRIPTION MANAGEMENT
// // // // // ============================================

// // // // // func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
// // // // //     // Check for existing active subscription
// // // // //     existing, _ := s.repo.FindCurrentBySchool(schoolID)
// // // // //     if existing != nil {
// // // // //         return nil, errors.New("school already has an active subscription")
// // // // //     }

// // // // //     // Get pricing
// // // // //     tier := models.SubscriptionTier(req.Tier)
// // // // //     interval := models.PaymentInterval(req.Interval)
// // // // //     amount, exists := models.Pricing[tier][interval]
// // // // //     if !exists {
// // // // //         return nil, errors.New("invalid pricing for selected tier and interval")
// // // // //     }

// // // // //     // Calculate dates
// // // // //     now := time.Now()
// // // // //     endDate := calculateEndDate(now, interval)
// // // // //     trialEndsAt := now.AddDate(0, 0, 7) // 7 days trial

// // // // //     // Create subscription record
// // // // //     subscriptionID := uuid.New().String()
// // // // //     subscription := &models.Subscription{
// // // // //         ID:                subscriptionID,
// // // // //         UserID:            userID,
// // // // //         SchoolID:          schoolID,
// // // // //         Tier:              tier,
// // // // //         Status:            models.SubStatusTrial,
// // // // //         Gateway:           models.PaymentGateway(req.Gateway),
// // // // //         Amount:            amount,
// // // // //         Currency:          models.CurrencyNGN,
// // // // //         PaymentInterval:   interval,
// // // // //         StartDate:         now,
// // // // //         EndDate:           endDate,
// // // // //         TrialEndsAt:       &trialEndsAt,
// // // // //         AutoRenew:         req.AutoRenew,
// // // // //         MaxStudents:       models.TierLimits[tier].MaxStudents,
// // // // //         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
// // // // //         MaxExams:          models.TierLimits[tier].MaxExams,
// // // // //         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
// // // // //         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
// // // // //         Features:          getFeaturesForTier(tier),
// // // // //         CreatedAt:         now,
// // // // //         UpdatedAt:         now,
// // // // //     }

// // // // //     if err := s.repo.Create(subscription); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create invoice
// // // // //     invoice := &models.Invoice{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: subscriptionID,
// // // // //         SchoolID:       schoolID,
// // // // //         UserID:         userID,
// // // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // // //         Amount:         amount,
// // // // //         Tax:            decimal.NewFromFloat(0),
// // // // //         Discount:       decimal.NewFromFloat(0),
// // // // //         Total:          amount,
// // // // //         Currency:       models.CurrencyNGN,
// // // // //         Status:         models.InvoicePending,
// // // // //         DueDate:        now.AddDate(0, 0, 7),
// // // // //         Items: map[string]interface{}{
// // // // //             "tier":     string(tier),
// // // // //             "interval": string(interval),
// // // // //         },
// // // // //         CreatedAt: now,
// // // // //         UpdatedAt: now,
// // // // //     }

// // // // //     if err := s.repo.CreateInvoice(invoice); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create payment intent
// // // // //     idempotencyKey := fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano())
// // // // //     paymentReq := &payment.PaymentRequest{
// // // // //         SchoolID:    schoolID,
// // // // //         UserID:      userID,
// // // // //         Amount:      amount,
// // // // //         Currency:    models.CurrencyNGN,
// // // // //         Email:       req.Email,
// // // // //         CallbackURL: req.CallbackURL,
// // // // //         SuccessURL:  req.SuccessURL,
// // // // //         CancelURL:   req.CancelURL,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "subscription_id": subscriptionID,
// // // // //             "invoice_id":      invoice.ID,
// // // // //             "school_id":       schoolID,
// // // // //             "user_id":         userID,
// // // // //             "tier":            string(tier),
// // // // //             "interval":        string(interval),
// // // // //         },
// // // // //     }

// // // // //     paymentIntent, err := s.paymentService.CreatePayment(ctx, models.PaymentGateway(req.Gateway), paymentReq)
// // // // //     if err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Store payment intent
// // // // //     dbPaymentIntent := &models.PaymentIntent{
// // // // //         ID:               uuid.New().String(),
// // // // //         SubscriptionID:   subscriptionID,
// // // // //         InvoiceID:        invoice.ID,
// // // // //         SchoolID:         schoolID,
// // // // //         UserID:           userID,
// // // // //         IdempotencyKey:   idempotencyKey,
// // // // //         Gateway:          models.PaymentGateway(req.Gateway),
// // // // //         Amount:           amount,
// // // // //         Currency:         models.CurrencyNGN,
// // // // //         Reference:        paymentIntent.Reference,
// // // // //         ClientSecret:     paymentIntent.ClientSecret,
// // // // //         AuthorizationURL: paymentIntent.AuthorizationURL,
// // // // //         AccessCode:       paymentIntent.AccessCode,
// // // // //         Status:           models.IntentPending,
// // // // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // // // //         CreatedAt:        now,
// // // // //         UpdatedAt:        now,
// // // // //     }

// // // // //     if err := s.repo.CreatePaymentIntent(dbPaymentIntent); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create history record
// // // // //     history := &models.SubscriptionHistory{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: subscriptionID,
// // // // //         SchoolID:       schoolID,
// // // // //         UserID:         userID,
// // // // //         NewTier:        tier,
// // // // //         NewStatus:      models.SubStatusTrial,
// // // // //         NewAmount:      amount,
// // // // //         ChangeReason:   "subscription_created",
// // // // //         ChangedBy:      userID,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "gateway":           string(req.Gateway),
// // // // //             "payment_intent_id": paymentIntent.ID,
// // // // //         },
// // // // //         CreatedAt: now,
// // // // //     }
// // // // //     s.repo.CreateHistory(history)

// // // // //     // Send email notification
// // // // //     go s.sendSubscriptionCreatedEmail(subscription, invoice, paymentIntent, req.Email)

// // // // //     // Create reminder schedules
// // // // //     s.createReminderSchedules(subscriptionID)

// // // // //     return &dto.CreateSubscriptionResponse{
// // // // //         Subscription:  s.toSubscriptionResponse(subscription),
// // // // //         PaymentIntent: s.toPaymentIntentResponse(dbPaymentIntent),
// // // // //         Invoice:       s.toInvoiceResponse(invoice),
// // // // //     }, nil
// // // // // }

// // // // // ========== REPLACE THIS BLOCK ==========
// // // // // Old: paymentReq := &payment.PaymentRequest{ ... }
// // // // //      paymentIntent, err := s.paymentService.CreatePayment(...)
// // // // // ========== WITH ==========

// // // // // // Create subscription on gateway (recurring)
// // // // // subReq := &payment.SubscriptionRequest{
// // // // //     SchoolID:     schoolID,
// // // // //     UserID:       userID,
// // // // //     Tier:         tier,
// // // // //     Interval:     interval,
// // // // //     Email:        req.Email,
// // // // //     CustomerName: "", // optionally fetch from user
// // // // //     SuccessURL:   req.SuccessURL,
// // // // //     CancelURL:    req.CancelURL,
// // // // // }

// // // // // gatewayResult, err := s.paymentService.CreateSubscription(ctx, models.PaymentGateway(req.Gateway), subReq)
// // // // // if err != nil {
// // // // //     // mark subscription as failed
// // // // //     subscription.Status = models.SubStatusFailed
// // // // //     s.repo.Update(subscription)
// // // // //     return nil, fmt.Errorf("gateway subscription creation failed: %w", err)
// // // // // }

// // // // // // Store gateway subscription ID on local record
// // // // // subscription.GatewaySubscriptionID = gatewayResult.GatewaySubscriptionID
// // // // // subscription.GatewayCustomerID = gatewayResult.GatewayCustomerID
// // // // // s.repo.Update(subscription)

// // // // // // Build payment intent record from gateway result
// // // // // dbPaymentIntent := &models.PaymentIntent{
// // // // //     ID:               uuid.New().String(),
// // // // //     SubscriptionID:   subscriptionID,
// // // // //     InvoiceID:        invoice.ID,
// // // // //     SchoolID:         schoolID,
// // // // //     UserID:           userID,
// // // // //     IdempotencyKey:   idempotencyKey,
// // // // //     Gateway:          models.PaymentGateway(req.Gateway),
// // // // //     Amount:           amount,
// // // // //     Currency:         models.CurrencyNGN,
// // // // //     Reference:        gatewayResult.SubscriptionID,
// // // // //     ClientSecret:     gatewayResult.ClientSecret,
// // // // //     AuthorizationURL: gatewayResult.AuthorizationURL,
// // // // //     Status:           models.IntentPending,
// // // // //     ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // // // //     CreatedAt:        now,
// // // // //     UpdatedAt:        now,
// // // // // }
// // // // // // Note: Flutterwave returns no authorization URL; user must be redirected to plan page? Actually Flutterwave subscription returns id, not link.
// // // // // // For Paystack, gatewayResult.AuthorizationURL is set. For Flutterwave, you may need to generate a payment link separately.
// // // // // // We'll handle that in the response: if AuthorizationURL is empty, the frontend might need to show a different message.



// // // // func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
// // // //     // Check for existing active subscription
// // // //     existing, _ := s.repo.FindCurrentBySchool(schoolID)
// // // //     if existing != nil {
// // // //         return nil, errors.New("school already has an active subscription")
// // // //     }

// // // //     // Get pricing
// // // //     tier := models.SubscriptionTier(req.Tier)
// // // //     interval := models.PaymentInterval(req.Interval)
// // // //     amount, exists := models.Pricing[tier][interval]
// // // //     if !exists {
// // // //         return nil, errors.New("invalid pricing for selected tier and interval")
// // // //     }

// // // //     // Calculate dates
// // // //     now := time.Now()
// // // //     endDate := calculateEndDate(now, interval)
// // // //     trialEndsAt := now.AddDate(0, 0, 7) // 7 days trial

// // // //     // Create subscription record
// // // //     subscriptionID := uuid.New().String()
// // // //     subscription := &models.Subscription{
// // // //         ID:                subscriptionID,
// // // //         UserID:            userID,
// // // //         SchoolID:          schoolID,
// // // //         Tier:              tier,
// // // //         Status:            models.SubStatusTrial,
// // // //         Gateway:           models.PaymentGateway(req.Gateway),
// // // //         Amount:            amount,
// // // //         Currency:          models.CurrencyNGN,
// // // //         PaymentInterval:   interval,
// // // //         StartDate:         now,
// // // //         EndDate:           endDate,
// // // //         TrialEndsAt:       &trialEndsAt,
// // // //         AutoRenew:         req.AutoRenew,
// // // //         MaxStudents:       models.TierLimits[tier].MaxStudents,
// // // //         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
// // // //         MaxExams:          models.TierLimits[tier].MaxExams,
// // // //         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
// // // //         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
// // // //         Features:          getFeaturesForTier(tier),
// // // //         CreatedAt:         now,
// // // //         UpdatedAt:         now,
// // // //     }

// // // //     if err := s.repo.Create(subscription); err != nil {
// // // //         return nil, err
// // // //     }

// // // //     // Create invoice
// // // //     invoice := &models.Invoice{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: subscriptionID,
// // // //         SchoolID:       schoolID,
// // // //         UserID:         userID,
// // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // //         Amount:         amount,
// // // //         Tax:            decimal.NewFromFloat(0),
// // // //         Discount:       decimal.NewFromFloat(0),
// // // //         Total:          amount,
// // // //         Currency:       models.CurrencyNGN,
// // // //         Status:         models.InvoicePending,
// // // //         DueDate:        now.AddDate(0, 0, 7),
// // // //         Items: map[string]interface{}{
// // // //             "tier":     string(tier),
// // // //             "interval": string(interval),
// // // //         },
// // // //         CreatedAt: now,
// // // //         UpdatedAt: now,
// // // //     }

// // // //     if err := s.repo.CreateInvoice(invoice); err != nil {
// // // //         return nil, err
// // // //     }

// // // //     // Idempotency key for the gateway call
// // // //     idempotencyKey := fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano())

// // // //     // ============================================================
// // // //     // Create recurring subscription on the payment gateway
// // // //     // ============================================================
// // // //     subReq := &payment.SubscriptionRequest{
// // // //         SchoolID:     schoolID,
// // // //         UserID:       userID,
// // // //         Tier:         tier,
// // // //         Interval:     interval,
// // // //         Email:        req.Email,
// // // //         CustomerName: "", // optionally fetch from user
// // // //         SuccessURL:   req.SuccessURL,
// // // //         CancelURL:    req.CancelURL,
// // // //     }

// // // //     gatewayResult, err := s.paymentService.CreateSubscription(ctx, models.PaymentGateway(req.Gateway), subReq)
// // // //     if err != nil {
// // // //         // mark subscription as failed
// // // //         subscription.Status = models.SubStatusFailed
// // // //         s.repo.Update(subscription)
// // // //         return nil, fmt.Errorf("gateway subscription creation failed: %w", err)
// // // //     }

// // // //     // Store gateway subscription ID on local record
// // // //     subscription.GatewaySubscriptionID = gatewayResult.GatewaySubscriptionID
// // // //     subscription.GatewayCustomerID = gatewayResult.GatewayCustomerID
// // // //     s.repo.Update(subscription)

// // // //     // Build payment intent record from gateway result
// // // //     dbPaymentIntent := &models.PaymentIntent{
// // // //         ID:               uuid.New().String(),
// // // //         SubscriptionID:   subscriptionID,
// // // //         InvoiceID:        invoice.ID,
// // // //         SchoolID:         schoolID,
// // // //         UserID:           userID,
// // // //         IdempotencyKey:   idempotencyKey,
// // // //         Gateway:          models.PaymentGateway(req.Gateway),
// // // //         Amount:           amount,
// // // //         Currency:         models.CurrencyNGN,
// // // //         Reference:        gatewayResult.SubscriptionID,
// // // //         ClientSecret:     gatewayResult.ClientSecret,
// // // //         AuthorizationURL: gatewayResult.AuthorizationURL,
// // // //         Status:           models.IntentPending,
// // // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // // //         CreatedAt:        now,
// // // //         UpdatedAt:        now,
// // // //     }

// // // //     if err := s.repo.CreatePaymentIntent(dbPaymentIntent); err != nil {
// // // //         return nil, err
// // // //     }

// // // //     // Create history record
// // // //     history := &models.SubscriptionHistory{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: subscriptionID,
// // // //         SchoolID:       schoolID,
// // // //         UserID:         userID,
// // // //         NewTier:        tier,
// // // //         NewStatus:      models.SubStatusTrial,
// // // //         NewAmount:      amount,
// // // //         ChangeReason:   "subscription_created",
// // // //         ChangedBy:      userID,
// // // //         Metadata: map[string]interface{}{
// // // //             "gateway":           string(req.Gateway),
// // // //             "payment_intent_id": dbPaymentIntent.ID,
// // // //         },
// // // //         CreatedAt: now,
// // // //     }
// // // //     s.repo.CreateHistory(history)

// // // //     // Send email notification
// // // //     go s.sendSubscriptionCreatedEmail(subscription, invoice, dbPaymentIntent, req.Email)

// // // //     // Create reminder schedules
// // // //     s.createReminderSchedules(subscriptionID)

// // // //     return &dto.CreateSubscriptionResponse{
// // // //         Subscription:  s.toSubscriptionResponse(subscription),
// // // //         PaymentIntent: s.toPaymentIntentResponse(dbPaymentIntent),
// // // //         Invoice:       s.toInvoiceResponse(invoice),
// // // //     }, nil
// // // // }


// // // // func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
// // // //     sub, err := s.repo.FindByID(id)
// // // //     if err != nil {
// // // //         return nil, errors.New("subscription not found")
// // // //     }
// // // //     return s.toSubscriptionResponse(sub), nil
// // // // }

// // // // func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
// // // //     subs, err := s.repo.FindBySchool(schoolID)
// // // //     if err != nil {
// // // //         return nil, err
// // // //     }

// // // //     responses := make([]dto.SubscriptionResponse, len(subs))
// // // //     for i, sub := range subs {
// // // //         responses[i] = *s.toSubscriptionResponse(&sub)
// // // //     }
// // // //     return responses, nil
// // // // }

// // // // func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
// // // //     sub, err := s.repo.FindCurrentBySchool(schoolID)
// // // //     if err != nil {
// // // //         return nil, errors.New("no active subscription found")
// // // //     }
// // // //     return s.toSubscriptionResponse(sub), nil
// // // // }

// // // // func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // // //     sub, err := s.repo.FindByID(id)
// // // //     if err != nil {
// // // //         return nil, errors.New("subscription not found")
// // // //     }

// // // //     oldStatus := sub.Status
// // // //     oldTier := sub.Tier
// // // //     oldAmount := sub.Amount

// // // //     if req.Tier != nil {
// // // //         newTier := models.SubscriptionTier(*req.Tier)
// // // //         newAmount := models.Pricing[newTier][sub.PaymentInterval]

// // // //         sub.Tier = newTier
// // // //         sub.Amount = newAmount
// // // //         sub.MaxStudents = models.TierLimits[newTier].MaxStudents
// // // //         sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
// // // //         sub.MaxExams = models.TierLimits[newTier].MaxExams
// // // //         sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
// // // //         sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
// // // //         sub.Features = getFeaturesForTier(newTier)
// // // //     }

// // // //     if req.AutoRenew != nil {
// // // //         sub.AutoRenew = *req.AutoRenew
// // // //     }

// // // //     if req.CancelAtEndDate != nil {
// // // //         sub.CancelAtEndDate = *req.CancelAtEndDate
// // // //     }

// // // //     if req.Status != nil {
// // // //         sub.Status = models.SubscriptionStatus(*req.Status)
// // // //     }

// // // //     sub.UpdatedAt = time.Now()
// // // //     sub.UpdatedBy = userID

// // // //     if err := s.repo.Update(sub); err != nil {
// // // //         return nil, err
// // // //     }

// // // //     // Create history record if important fields changed
// // // //     if oldTier != sub.Tier || oldStatus != sub.Status || !oldAmount.Equal(sub.Amount) {
// // // //         history := &models.SubscriptionHistory{
// // // //             ID:             uuid.New().String(),
// // // //             SubscriptionID: sub.ID,
// // // //             SchoolID:       sub.SchoolID,
// // // //             UserID:         userID,
// // // //             OldTier:        oldTier,
// // // //             NewTier:        sub.Tier,
// // // //             OldStatus:      oldStatus,
// // // //             NewStatus:      sub.Status,
// // // //             OldAmount:      oldAmount,
// // // //             NewAmount:      sub.Amount,
// // // //             ChangeReason:   "subscription_updated",
// // // //             ChangedBy:      userID,
// // // //             CreatedAt:      time.Now(),
// // // //         }
// // // //         s.repo.CreateHistory(history)
// // // //     }

// // // //     return s.toSubscriptionResponse(sub), nil
// // // // }

// // // // func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
// // // //     sub, err := s.repo.FindByID(id)
// // // //     if err != nil {
// // // //         return errors.New("subscription not found")
// // // //     }

// // // //     if cancelImmediately {
// // // //         sub.Status = models.SubStatusCancelled

// // // //         // Cancel at gateway
// // // //         s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
// // // //     } else {
// // // //         sub.CancelAtEndDate = true
// // // //     }

// // // //     sub.UpdatedAt = time.Now()
// // // //     sub.UpdatedBy = userID

// // // //     if err := s.repo.Update(sub); err != nil {
// // // //         return err
// // // //     }

// // // //     // Create history record
// // // //     history := &models.SubscriptionHistory{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: sub.ID,
// // // //         SchoolID:       sub.SchoolID,
// // // //         UserID:         userID,
// // // //         OldStatus:      sub.Status,
// // // //         NewStatus:      models.SubStatusCancelled,
// // // //         ChangeReason:   reason,
// // // //         ChangedBy:      userID,
// // // //         Metadata: map[string]interface{}{
// // // //             "cancel_immediately": cancelImmediately,
// // // //         },
// // // //         CreatedAt: time.Now(),
// // // //     }
// // // //     s.repo.CreateHistory(history)

// // // //     // Send cancellation email
// // // //     go s.sendSubscriptionCancelledEmail(sub)

// // // //     return nil
// // // // }

// // // // func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // // //     sub, err := s.repo.FindByID(id)
// // // //     if err != nil {
// // // //         return nil, errors.New("subscription not found")
// // // //     }

// // // //     newInterval := models.PaymentInterval(req.Interval)
// // // //     newAmount := models.Pricing[sub.Tier][newInterval]
// // // //     newEndDate := calculateEndDate(time.Now(), newInterval)

// // // //     sub.PaymentInterval = newInterval
// // // //     sub.Amount = newAmount
// // // //     sub.EndDate = newEndDate
// // // //     sub.Status = models.SubStatusActive
// // // //     sub.AutoRenew = true
// // // //     sub.CancelAtEndDate = false
// // // //     sub.UpdatedAt = time.Now()
// // // //     sub.UpdatedBy = userID

// // // //     if err := s.repo.Update(sub); err != nil {
// // // //         return nil, err
// // // //     }

// // // //     // Create new invoice
// // // //     invoice := &models.Invoice{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: sub.ID,
// // // //         SchoolID:       sub.SchoolID,
// // // //         UserID:         userID,
// // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // //         Amount:         newAmount,
// // // //         Total:          newAmount,
// // // //         Currency:       models.CurrencyNGN,
// // // //         Status:         models.InvoicePending,
// // // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // // //         Items: map[string]interface{}{
// // // //             "renewal":      true,
// // // //             "old_interval": string(sub.PaymentInterval),
// // // //             "new_interval": string(newInterval),
// // // //         },
// // // //         CreatedAt: time.Now(),
// // // //         UpdatedAt: time.Now(),
// // // //     }
// // // //     s.repo.CreateInvoice(invoice)

// // // //     // Create history record
// // // //     history := &models.SubscriptionHistory{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: sub.ID,
// // // //         SchoolID:       sub.SchoolID,
// // // //         UserID:         userID,
// // // //         NewAmount:      newAmount,
// // // //         ChangeReason:   "subscription_renewed",
// // // //         ChangedBy:      userID,
// // // //         Metadata: map[string]interface{}{
// // // //             "new_interval": string(newInterval),
// // // //             "invoice_id":   invoice.ID,
// // // //         },
// // // //         CreatedAt: time.Now(),
// // // //     }
// // // //     s.repo.CreateHistory(history)

// // // //     // Send renewal email
// // // //     go s.sendSubscriptionRenewedEmail(sub, invoice)

// // // //     return s.toSubscriptionResponse(sub), nil
// // // // }

// // // // // ============================================
// // // // // PAYMENT INTENT MANAGEMENT
// // // // // ============================================

// // // // func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
// // // //     sub, err := s.repo.FindByID(subscriptionID)
// // // //     if err != nil {
// // // //         return nil, errors.New("subscription not found")
// // // //     }

// // // //     // Create invoice for this payment
// // // //     invoice := &models.Invoice{
// // // //         ID:             uuid.New().String(),
// // // //         SubscriptionID: subscriptionID,
// // // //         SchoolID:       sub.SchoolID,
// // // //         UserID:         userID,
// // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // //         Amount:         sub.Amount,
// // // //         Total:          sub.Amount,
// // // //         Currency:       sub.Currency,
// // // //         Status:         models.InvoicePending,
// // // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // // //         CreatedAt:      time.Now(),
// // // //         UpdatedAt:      time.Now(),
// // // //     }
// // // //     s.repo.CreateInvoice(invoice)

// // // //     // Create payment intent
// // // //     idempotencyKey := fmt.Sprintf("%s_%d_%d", subscriptionID, time.Now().UnixNano(), invoice.ID)
// // // //     paymentReq := &payment.PaymentRequest{
// // // //         SchoolID:    sub.SchoolID,
// // // //         UserID:      userID,
// // // //         Amount:      sub.Amount,
// // // //         Currency:    sub.Currency,
// // // //         Email:       req.SuccessURL,
// // // //         SuccessURL:  req.SuccessURL,
// // // //         CancelURL:   req.CancelURL,
// // // //         Metadata: map[string]interface{}{
// // // //             "subscription_id": subscriptionID,
// // // //             "invoice_id":      invoice.ID,
// // // //             "payment_type":    "renewal",
// // // //         },
// // // //     }

// // // //     paymentIntent, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
// // // //     if err != nil {
// // // //         return nil, err
// // // //     }

// // // //     dbPaymentIntent := &models.PaymentIntent{
// // // //         ID:               uuid.New().String(),
// // // //         SubscriptionID:   subscriptionID,
// // // //         InvoiceID:        invoice.ID,
// // // //         SchoolID:         sub.SchoolID,
// // // //         UserID:           userID,
// // // //         IdempotencyKey:   idempotencyKey,
// // // //         Gateway:          sub.Gateway,
// // // //         Amount:           sub.Amount,
// // // //         Currency:         sub.Currency,
// // // //         Reference:        paymentIntent.Reference,
// // // //         ClientSecret:     paymentIntent.ClientSecret,
// // // //         AuthorizationURL: paymentIntent.AuthorizationURL,
// // // //         Status:           models.IntentPending,
// // // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // // //         CreatedAt:        time.Now(),
// // // //         UpdatedAt:        time.Now(),
// // // //     }
// // // //     s.repo.CreatePaymentIntent(dbPaymentIntent)

// // // //     return s.toPaymentIntentResponse(dbPaymentIntent), nil
// // // // }

// // // // func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
// // // //     pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
// // // //     if err != nil {
// // // //         return errors.New("payment intent not found")
// // // //     }

// // // //     if pi.IsFinalized {
// // // //         return errors.New("payment intent already finalized")
// // // //     }

// // // //     // Verify with gateway
// // // //     verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
// // // //     if err != nil {
// // // //         return err
// // // //     }

// // // //     pi.Status = models.PaymentIntentStatus(verification.Status)
// // // //     pi.GatewayResponse = verification.GatewayData

// // // //     if verification.Status == "succeeded" {
// // // //         pi.PaidAt = timePtr(time.Now())
// // // //         pi.IsFinalized = true

// // // //         // Mark invoice as paid
// // // //         s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())

// // // //         // Update subscription
// // // //         sub, _ := s.repo.FindByID(pi.SubscriptionID)
// // // //         sub.Status = models.SubStatusActive
// // // //         sub.LastPaymentDate = timePtr(time.Now())
// // // //         sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// // // //         // sub.PaymentStatus does not exist in Subscription model; removed this line
// // // //         s.repo.Update(sub)

// // // //         // Create payment transaction
// // // //         transaction := &models.PaymentTransaction{
// // // //             ID:               uuid.New().String(),
// // // //             SubscriptionID:   pi.SubscriptionID,
// // // //             PaymentIntentID:  pi.ID,
// // // //             InvoiceID:        pi.InvoiceID,
// // // //             SchoolID:         pi.SchoolID,
// // // //             UserID:           pi.UserID,
// // // //             Amount:           pi.Amount,
// // // //             Currency:         pi.Currency,
// // // //             PaymentMethod:    models.MethodCard,
// // // //             PaymentStatus:    models.PaymentPaid,
// // // //             Gateway:          pi.Gateway,
// // // //             Reference:        pi.Reference,
// // // //             PaidAt:           timePtr(time.Now()),
// // // //             CreatedAt:        time.Now(),
// // // //             UpdatedAt:        time.Now(),
// // // //         }
// // // //         s.repo.CreatePaymentTransaction(transaction)

// // // //         // Send payment success email
// // // //         go s.sendPaymentSuccessEmail(sub, transaction)
// // // //     }

// // // //     s.repo.UpdatePaymentIntent(pi)

// // // //     // Create event log
// // // //     eventLog := &models.PaymentEventLog{
// // // //         ID:             uuid.New().String(),
// // // //         PaymentIntentID: pi.ID,
// // // //         EventType:      "payment_intent_confirmed",
// // // //         StatusBefore:   string(models.IntentPending),
// // // //         StatusAfter:    string(pi.Status),
// // // //         Payload: map[string]interface{}{
// // // //             "verification": verification,
// // // //         },
// // // //         CreatedAt: time.Now(),
// // // //     }
// // // //     s.repo.CreatePaymentEventLog(eventLog)

// // // //     return nil
// // // // }

// // // // // ============================================
// // // // // HELPER FUNCTIONS
// // // // // ============================================

// // // // func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
// // // //     switch interval {
// // // //     case models.IntervalMonthly:
// // // //         return start.AddDate(0, 1, 0)
// // // //     case models.IntervalQuarterly:
// // // //         return start.AddDate(0, 3, 0)
// // // //     case models.IntervalYearly:
// // // //         return start.AddDate(1, 0, 0)
// // // //     default:
// // // //         return start.AddDate(0, 1, 0)
// // // //     }
// // // // }

// // // // func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
// // // //     next := calculateEndDate(currentEnd, interval)
// // // //     return &next
// // // // }

// // // // func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
// // // //     features := make(models.JSONMap)

// // // //     switch tier {
// // // //     case models.TierBasic:
// // // //         features["analytics"] = false
// // // //         features["api_access"] = false
// // // //         features["priority_support"] = false
// // // //         features["bulk_import"] = false
// // // //         features["white_label"] = false
// // // //     case models.TierPremium:
// // // //         features["analytics"] = true
// // // //         features["api_access"] = true
// // // //         features["priority_support"] = false
// // // //         features["bulk_import"] = true
// // // //         features["white_label"] = false
// // // //         features["custom_reports"] = true
// // // //     case models.TierEnterprise:
// // // //         features["analytics"] = true
// // // //         features["api_access"] = true
// // // //         features["priority_support"] = true
// // // //         features["bulk_import"] = true
// // // //         features["white_label"] = true
// // // //         features["custom_reports"] = true
// // // //         features["dedicated_server"] = true
// // // //         features["sla"] = true
// // // //     }

// // // //     return features
// // // // }

// // // // func (s *SubscriptionService) createReminderSchedules(subscriptionID string) {
// // // //     reminderDays := []int{30, 14, 7, 3, 1}
// // // //     for _, days := range reminderDays {
// // // //         schedule := &models.ReminderSchedule{
// // // //             ID:             uuid.New().String(),
// // // //             SubscriptionID: subscriptionID,
// // // //             ReminderType:   "expiry",
// // // //             DaysBefore:     days,
// // // //             Status:         "pending",
// // // //             CreatedAt:      time.Now(),
// // // //             UpdatedAt:      time.Now(),
// // // //         }
// // // //         s.repo.CreateReminderSchedule(schedule)
// // // //     }
// // // // }

// // // // func timePtr(t time.Time) *time.Time {
// // // //     return &t
// // // // }

// // // // // ============================================
// // // // // RESPONSE MAPPERS
// // // // // ============================================

// // // // func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
// // // //     return &dto.SubscriptionResponse{
// // // //         ID:              sub.ID,
// // // //         SchoolID:        sub.SchoolID,
// // // //         Tier:            string(sub.Tier),
// // // //         Status:          string(sub.Status),
// // // //         Gateway:         string(sub.Gateway),
// // // //         Amount:          sub.Amount,
// // // //         Currency:        string(sub.Currency),
// // // //         PaymentInterval: string(sub.PaymentInterval),
// // // //         StartDate:       sub.StartDate,
// // // //         EndDate:         sub.EndDate,
// // // //         TrialEndsAt:     sub.TrialEndsAt,
// // // //         AutoRenew:       sub.AutoRenew,
// // // //         CancelAtEndDate: sub.CancelAtEndDate,
// // // //         LastPaymentDate: sub.LastPaymentDate,
// // // //         NextPaymentDate: sub.NextPaymentDate,
// // // //         MaxStudents:     sub.MaxStudents,
// // // //         MaxTeachers:     sub.MaxTeachers,
// // // //         MaxExams:        sub.MaxExams,
// // // //         MaxQuestions:    sub.MaxQuestions,
// // // //         MaxStorageMB:    sub.MaxStorageMB,
// // // //         Features:        sub.Features,
// // // //         CreatedAt:       sub.CreatedAt,
// // // //         UpdatedAt:       sub.UpdatedAt,
// // // //     }
// // // // }

// // // // func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
// // // //     return &dto.PaymentIntentResponse{
// // // //         ID:               pi.ID,
// // // //         SubscriptionID:   pi.SubscriptionID,
// // // //         InvoiceID:        pi.InvoiceID,
// // // //         Gateway:          string(pi.Gateway),
// // // //         Amount:           pi.Amount,
// // // //         Currency:         string(pi.Currency),
// // // //         Reference:        pi.Reference,
// // // //         ClientSecret:     pi.ClientSecret,
// // // //         AuthorizationURL: pi.AuthorizationURL,
// // // //         AccessCode:       pi.AccessCode,
// // // //         Status:           string(pi.Status),
// // // //         ExpiresAt:        pi.ExpiresAt,
// // // //         PaymentMethod:    string(pi.PaymentMethod),
// // // //     }
// // // // }

// // // // func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
// // // //     return &dto.InvoiceResponse{
// // // //         ID:            inv.ID,
// // // //         InvoiceNumber: inv.InvoiceNumber,
// // // //         Amount:        inv.Amount,
// // // //         Tax:           inv.Tax,
// // // //         Discount:      inv.Discount,
// // // //         Total:         inv.Total,
// // // //         Currency:      string(inv.Currency),
// // // //         Status:        string(inv.Status),
// // // //         DueDate:       inv.DueDate,
// // // //         PaidAt:        inv.PaidAt,
// // // //         PDFURL:        inv.PDFURL,
// // // //         Items:         inv.Items,
// // // //         CreatedAt:     inv.CreatedAt,
// // // //     }
// // // // }

// // // // // ============================================
// // // // // EMAIL NOTIFICATIONS (To be implemented)
// // // // // ============================================

// // // // func (s *SubscriptionService) sendSubscriptionCreatedEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
// // // //     // TODO: Implement email sending
// // // //     fmt.Printf("Subscription created email sent to %s\n", email)
// // // // }

// // // // func (s *SubscriptionService) sendSubscriptionRenewedEmail(sub *models.Subscription, inv *models.Invoice) {
// // // //     // TODO: Implement email sending
// // // //     fmt.Printf("Subscription renewed email sent\n")
// // // // }

// // // // func (s *SubscriptionService) sendSubscriptionCancelledEmail(sub *models.Subscription) {
// // // //     // TODO: Implement email sending
// // // //     fmt.Printf("Subscription cancelled email sent\n")
// // // // }

// // // // func (s *SubscriptionService) sendPaymentSuccessEmail(sub *models.Subscription, transaction *models.PaymentTransaction) {
// // // //     // TODO: Implement email sending
// // // //     fmt.Printf("Payment success email sent\n")
// // // // }


// // // // // ProcessWebhook handles incoming webhooks from payment gateways
// // // // func (s *SubscriptionService) ProcessWebhook(ctx context.Context, gateway models.PaymentGateway, payload []byte, signature string) error {
// // // //     // Get the gateway implementation
// // // //     // Note: we need a method on PaymentService to get the gateway, or we can call directly.
// // // //     // For simplicity, we'll create a helper on PaymentService: GetGateway(gateway) Gateway
// // // //     gw, err := s.paymentService.GetGateway(gateway)
// // // //     if err != nil {
// // // //         return err
// // // //     }
// // // //     event, err := gw.ParseWebhook(ctx, payload, signature)
// // // //     if err != nil {
// // // //         return err
// // // //     }

// // // //     // Log webhook event (store in DB)
// // // //     webhookEvent := &models.WebhookEvent{
// // // //         ID:             uuid.New().String(),
// // // //         Gateway:        gateway,
// // // //         EventType:      event.Type,
// // // //         Payload:        event.RawData,
// // // //         Signature:      signature,
// // // //         IdempotencyKey: event.Reference, // use reference as idempotency key if available
// // // //         Status:         models.WebhookReceived,
// // // //         CreatedAt:      time.Now(),
// // // //         UpdatedAt:      time.Now(),
// // // //     }
// // // //     s.repo.CreateWebhookEvent(webhookEvent)

// // // //     switch event.Type {
// // // //     case "subscription_activated":
// // // //         // Find local subscription by gateway_subscription_id
// // // //         sub, err := s.repo.FindByGatewaySubscriptionID(event.GatewaySubscriptionID)
// // // //         if err != nil {
// // // //             return fmt.Errorf("subscription not found for gateway ID %s", event.GatewaySubscriptionID)
// // // //         }
// // // //         if sub.Status == models.SubStatusTrial || sub.Status == models.SubStatusPending {
// // // //             now := time.Now()
// // // //             sub.Status = models.SubStatusActive
// // // //             sub.StartDate = now
// // // //             sub.EndDate = calculateEndDate(now, sub.PaymentInterval)
// // // //             sub.UpdatedAt = now
// // // //             s.repo.Update(sub)

// // // //             // Mark the first invoice as paid
// // // //             invoices, _ := s.repo.FindInvoicesBySubscription(sub.ID)
// // // //             if len(invoices) > 0 {
// // // //                 s.repo.MarkInvoiceAsPaid(invoices[0].ID, now)
// // // //             }
// // // //         }
// // // //     case "payment_success":
// // // //         // For subsequent recurring payments, update invoice and create transaction
// // // //         // You may need to find subscription by reference or metadata
// // // //         // For simplicity, we'll implement a method to find payment intent by reference
// // // //         pi, err := s.repo.FindPaymentIntentByReference(event.Reference)
// // // //         if err == nil && pi != nil {
// // // //             s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
// // // //             // Update subscription last payment date
// // // //             sub, _ := s.repo.FindByID(pi.SubscriptionID)
// // // //             if sub != nil {
// // // //                 sub.LastPaymentDate = timePtr(time.Now())
// // // //                 s.repo.Update(sub)
// // // //             }
// // // //         }
// // // //     case "subscription_cancelled":
// // // //         sub, err := s.repo.FindByGatewaySubscriptionID(event.GatewaySubscriptionID)
// // // //         if err == nil {
// // // //             sub.Status = models.SubStatusCancelled
// // // //             s.repo.Update(sub)
// // // //         }
// // // //     }

// // // //     // Update webhook event status
// // // //     s.repo.UpdateWebhookEventStatus(webhookEvent.ID, models.WebhookProcessed, "")
// // // //     return nil
// // // // }


// // // // // package service

// // // // // import (
// // // // //     "context"
// // // // //     "errors"
// // // // //     "fmt"
// // // // //     "time"

// // // // //     "github.com/google/uuid"
// // // // //     "github.com/shopspring/decimal"

// // // // //     "cbt-api/internal/models"
// // // // //     "cbt-api/internal/subscription/dto"
// // // // //     "cbt-api/internal/subscription/repository"
// // // // //     "cbt-api/pkg/email"
// // // // //     "cbt-api/pkg/payment"
// // // // // )

// // // // // type SubscriptionService struct {
// // // // //     repo           *repository.SubscriptionRepository
// // // // //     paymentService *payment.PaymentService
// // // // //     emailService   *email.EmailService
// // // // // }

// // // // // func NewSubscriptionService(
// // // // //     repo *repository.SubscriptionRepository,
// // // // //     paymentService *payment.PaymentService,
// // // // //     emailService *email.EmailService,
// // // // // ) *SubscriptionService {
// // // // //     return &SubscriptionService{
// // // // //         repo:           repo,
// // // // //         paymentService: paymentService,
// // // // //         emailService:   emailService,
// // // // //     }
// // // // // }

// // // // // // ============================================
// // // // // // SUBSCRIPTION MANAGEMENT
// // // // // // ============================================

// // // // // func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest, userID, schoolID string) (*dto.CreateSubscriptionResponse, error) {
// // // // //     // Check for existing active subscription
// // // // //     existing, _ := s.repo.FindCurrentBySchool(schoolID)
// // // // //     if existing != nil {
// // // // //         return nil, errors.New("school already has an active subscription")
// // // // //     }

// // // // //     // Get pricing
// // // // //     tier := models.SubscriptionTier(req.Tier)
// // // // //     interval := models.PaymentInterval(req.Interval)
// // // // //     amount, exists := models.Pricing[tier][interval]
// // // // //     if !exists {
// // // // //         return nil, errors.New("invalid pricing for selected tier and interval")
// // // // //     }

// // // // //     // Calculate dates
// // // // //     now := time.Now()
// // // // //     endDate := calculateEndDate(now, interval)
// // // // //     trialEndsAt := now.AddDate(0, 0, 7) // 7 days trial

// // // // //     // Create subscription record
// // // // //     subscriptionID := uuid.New().String()
// // // // //     subscription := &models.Subscription{
// // // // //         ID:                subscriptionID,
// // // // //         UserID:            userID,
// // // // //         SchoolID:          schoolID,
// // // // //         Tier:              tier,
// // // // //         Status:            models.SubStatusTrial,
// // // // //         Gateway:           models.PaymentGateway(req.Gateway),
// // // // //         Amount:            amount,
// // // // //         Currency:          models.CurrencyNGN,
// // // // //         PaymentInterval:   interval,
// // // // //         StartDate:         now,
// // // // //         EndDate:           endDate,
// // // // //         TrialEndsAt:       &trialEndsAt,
// // // // //         AutoRenew:         req.AutoRenew,
// // // // //         MaxStudents:       models.TierLimits[tier].MaxStudents,
// // // // //         MaxTeachers:       models.TierLimits[tier].MaxTeachers,
// // // // //         MaxExams:          models.TierLimits[tier].MaxExams,
// // // // //         MaxQuestions:      models.TierLimits[tier].MaxQuestions,
// // // // //         MaxStorageMB:      models.TierLimits[tier].MaxStorageMB,
// // // // //         Features:          getFeaturesForTier(tier),
// // // // //         CreatedAt:         now,
// // // // //         UpdatedAt:         now,
// // // // //     }

// // // // //     if err := s.repo.Create(subscription); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create invoice
// // // // //     invoice := &models.Invoice{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: subscriptionID,
// // // // //         SchoolID:       schoolID,
// // // // //         UserID:         userID,
// // // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // // //         Amount:         amount,
// // // // //         Tax:            decimal.NewFromFloat(0),
// // // // //         Discount:       decimal.NewFromFloat(0),
// // // // //         Total:          amount,
// // // // //         Currency:       models.CurrencyNGN,
// // // // //         Status:         models.InvoicePending,
// // // // //         DueDate:        now.AddDate(0, 0, 7),
// // // // //         Items: map[string]interface{}{
// // // // //             "tier":     string(tier),
// // // // //             "interval": string(interval),
// // // // //         },
// // // // //         CreatedAt: now,
// // // // //         UpdatedAt: now,
// // // // //     }

// // // // //     if err := s.repo.CreateInvoice(invoice); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create payment intent
// // // // //     idempotencyKey := fmt.Sprintf("%s_%d", subscriptionID, time.Now().UnixNano())
// // // // //     paymentReq := &payment.PaymentRequest{
// // // // //         SchoolID:    schoolID,
// // // // //         UserID:      userID,
// // // // //         Amount:      amount,
// // // // //         Currency:    models.CurrencyNGN,
// // // // //         Email:       req.Email,
// // // // //         CallbackURL: req.CallbackURL,
// // // // //         SuccessURL:  req.SuccessURL,
// // // // //         CancelURL:   req.CancelURL,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "subscription_id": subscriptionID,
// // // // //             "invoice_id":      invoice.ID,
// // // // //             "school_id":       schoolID,
// // // // //             "user_id":         userID,
// // // // //             "tier":            string(tier),
// // // // //             "interval":        string(interval),
// // // // //         },
// // // // //     }

// // // // //     paymentIntent, err := s.paymentService.CreatePayment(ctx, models.PaymentGateway(req.Gateway), paymentReq)
// // // // //     if err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Store payment intent
// // // // //     dbPaymentIntent := &models.PaymentIntent{
// // // // //         ID:               uuid.New().String(),
// // // // //         SubscriptionID:   subscriptionID,
// // // // //         InvoiceID:        invoice.ID,
// // // // //         SchoolID:         schoolID,
// // // // //         UserID:           userID,
// // // // //         IdempotencyKey:   idempotencyKey,
// // // // //         Gateway:          models.PaymentGateway(req.Gateway),
// // // // //         Amount:           amount,
// // // // //         Currency:         models.CurrencyNGN,
// // // // //         Reference:        paymentIntent.Reference,
// // // // //         ClientSecret:     paymentIntent.ClientSecret,
// // // // //         AuthorizationURL: paymentIntent.AuthorizationURL,
// // // // //         AccessCode:       paymentIntent.AccessCode,
// // // // //         Status:           models.IntentPending,
// // // // //         ExpiresAt: timePtr(time.Now().Add(24 * time.Hour)),
// // // // //         CreatedAt:        now,
// // // // //         UpdatedAt:        now,
// // // // //     }

// // // // //     if err := s.repo.CreatePaymentIntent(dbPaymentIntent); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create history record
// // // // //     history := &models.SubscriptionHistory{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: subscriptionID,
// // // // //         SchoolID:       schoolID,
// // // // //         UserID:         userID,
// // // // //         NewTier:        tier,
// // // // //         NewStatus:      models.SubStatusTrial,
// // // // //         NewAmount:      amount,
// // // // //         ChangeReason:   "subscription_created",
// // // // //         ChangedBy:      userID,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "gateway":      string(req.Gateway),
// // // // //             "payment_intent_id": paymentIntent.ID,
// // // // //         },
// // // // //         CreatedAt: now,
// // // // //     }
// // // // //     s.repo.CreateHistory(history)

// // // // //     // Send email notification
// // // // //     go s.sendSubscriptionCreatedEmail(subscription, invoice, paymentIntent, req.Email)

// // // // //     // Create reminder schedules
// // // // //     s.createReminderSchedules(subscriptionID)

// // // // //     return &dto.CreateSubscriptionResponse{
// // // // //         Subscription:  s.toSubscriptionResponse(subscription),
// // // // //         PaymentIntent: s.toPaymentIntentResponse(dbPaymentIntent),
// // // // //         Invoice:       s.toInvoiceResponse(invoice),
// // // // //     }, nil
// // // // // }

// // // // // func (s *SubscriptionService) GetSubscription(id string) (*dto.SubscriptionResponse, error) {
// // // // //     sub, err := s.repo.FindByID(id)
// // // // //     if err != nil {
// // // // //         return nil, errors.New("subscription not found")
// // // // //     }
// // // // //     return s.toSubscriptionResponse(sub), nil
// // // // // }

// // // // // func (s *SubscriptionService) GetSubscriptionsBySchool(schoolID string) ([]dto.SubscriptionResponse, error) {
// // // // //     subs, err := s.repo.FindBySchool(schoolID)
// // // // //     if err != nil {
// // // // //         return nil, err
// // // // //     }
    
// // // // //     responses := make([]dto.SubscriptionResponse, len(subs))
// // // // //     for i, sub := range subs {
// // // // //         responses[i] = *s.toSubscriptionResponse(&sub)
// // // // //     }
// // // // //     return responses, nil
// // // // // }

// // // // // func (s *SubscriptionService) GetCurrentSubscription(schoolID string) (*dto.SubscriptionResponse, error) {
// // // // //     sub, err := s.repo.FindCurrentBySchool(schoolID)
// // // // //     if err != nil {
// // // // //         return nil, errors.New("no active subscription found")
// // // // //     }
// // // // //     return s.toSubscriptionResponse(sub), nil
// // // // // }

// // // // // func (s *SubscriptionService) UpdateSubscription(id string, req *dto.UpdateSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // // // //     sub, err := s.repo.FindByID(id)
// // // // //     if err != nil {
// // // // //         return nil, errors.New("subscription not found")
// // // // //     }

// // // // //     oldStatus := sub.Status
// // // // //     oldTier := sub.Tier
// // // // //     oldAmount := sub.Amount

// // // // //     if req.Tier != nil {
// // // // //         newTier := models.SubscriptionTier(*req.Tier)
// // // // //         newAmount := models.Pricing[newTier][sub.PaymentInterval]
        
// // // // //         sub.Tier = newTier
// // // // //         sub.Amount = newAmount
// // // // //         sub.MaxStudents = models.TierLimits[newTier].MaxStudents
// // // // //         sub.MaxTeachers = models.TierLimits[newTier].MaxTeachers
// // // // //         sub.MaxExams = models.TierLimits[newTier].MaxExams
// // // // //         sub.MaxQuestions = models.TierLimits[newTier].MaxQuestions
// // // // //         sub.MaxStorageMB = models.TierLimits[newTier].MaxStorageMB
// // // // //         sub.Features = getFeaturesForTier(newTier)
// // // // //     }
    
// // // // //     if req.AutoRenew != nil {
// // // // //         sub.AutoRenew = *req.AutoRenew
// // // // //     }
    
// // // // //     if req.CancelAtEndDate != nil {
// // // // //         sub.CancelAtEndDate = *req.CancelAtEndDate
// // // // //     }
    
// // // // //     if req.Status != nil {
// // // // //         sub.Status = models.SubscriptionStatus(*req.Status)
// // // // //     }
    
// // // // //     sub.UpdatedAt = time.Now()
// // // // //     sub.UpdatedBy = userID

// // // // //     if err := s.repo.Update(sub); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create history record if important fields changed
// // // // //     if oldTier != sub.Tier || oldStatus != sub.Status || !oldAmount.Equal(sub.Amount) {
// // // // //         history := &models.SubscriptionHistory{
// // // // //             ID:             uuid.New().String(),
// // // // //             SubscriptionID: sub.ID,
// // // // //             SchoolID:       sub.SchoolID,
// // // // //             UserID:         userID,
// // // // //             OldTier:        &oldTier,
// // // // //             NewTier:        &sub.Tier,
// // // // //             OldStatus:      &oldStatus,
// // // // //             NewStatus:      &sub.Status,
// // // // //             OldAmount:      &oldAmount,
// // // // //             NewAmount:      &sub.Amount,
// // // // //             ChangeReason:   "subscription_updated",
// // // // //             ChangedBy:      userID,
// // // // //             CreatedAt:      time.Now(),
// // // // //         }
// // // // //         s.repo.CreateHistory(history)
// // // // //     }

// // // // //     return s.toSubscriptionResponse(sub), nil
// // // // // }

// // // // // func (s *SubscriptionService) CancelSubscription(id string, cancelImmediately bool, reason string, userID string) error {
// // // // //     sub, err := s.repo.FindByID(id)
// // // // //     if err != nil {
// // // // //         return errors.New("subscription not found")
// // // // //     }

// // // // //     if cancelImmediately {
// // // // //         sub.Status = models.SubStatusCancelled
        
// // // // //         // Cancel at gateway
// // // // //         s.paymentService.CancelSubscription(context.Background(), sub.Gateway, sub.GatewaySubscriptionID)
// // // // //     } else {
// // // // //         sub.CancelAtEndDate = true
// // // // //     }
    
// // // // //     sub.UpdatedAt = time.Now()
// // // // //     sub.UpdatedBy = userID

// // // // //     if err := s.repo.Update(sub); err != nil {
// // // // //         return err
// // // // //     }

// // // // //     // Create history record
// // // // //     history := &models.SubscriptionHistory{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: sub.ID,
// // // // //         SchoolID:       sub.SchoolID,
// // // // //         UserID:         userID,
// // // // //         OldStatus:      &sub.Status,
// // // // //         NewStatus:      &models.SubStatusCancelled,
// // // // //         ChangeReason:   reason,
// // // // //         ChangedBy:      userID,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "cancel_immediately": cancelImmediately,
// // // // //         },
// // // // //         CreatedAt:      time.Now(),
// // // // //     }
// // // // //     s.repo.CreateHistory(history)

// // // // //     // Send cancellation email
// // // // //     go s.sendSubscriptionCancelledEmail(sub)

// // // // //     return nil
// // // // // }

// // // // // func (s *SubscriptionService) RenewSubscription(id string, req *dto.RenewSubscriptionRequest, userID string) (*dto.SubscriptionResponse, error) {
// // // // //     sub, err := s.repo.FindByID(id)
// // // // //     if err != nil {
// // // // //         return nil, errors.New("subscription not found")
// // // // //     }

// // // // //     newInterval := models.PaymentInterval(req.Interval)
// // // // //     newAmount := models.Pricing[sub.Tier][newInterval]
// // // // //     newEndDate := calculateEndDate(time.Now(), newInterval)

// // // // //     sub.PaymentInterval = newInterval
// // // // //     sub.Amount = newAmount
// // // // //     sub.EndDate = newEndDate
// // // // //     sub.Status = models.SubStatusActive
// // // // //     sub.AutoRenew = true
// // // // //     sub.CancelAtEndDate = false
// // // // //     sub.UpdatedAt = time.Now()
// // // // //     sub.UpdatedBy = userID

// // // // //     if err := s.repo.Update(sub); err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     // Create new invoice
// // // // //     invoice := &models.Invoice{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: sub.ID,
// // // // //         SchoolID:       sub.SchoolID,
// // // // //         UserID:         userID,
// // // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // // //         Amount:         newAmount,
// // // // //         Total:          newAmount,
// // // // //         Currency:       models.CurrencyNGN,
// // // // //         Status:         models.InvoicePending,
// // // // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // // // //         Items: map[string]interface{}{
// // // // //             "renewal":      true,
// // // // //             "old_interval": string(sub.PaymentInterval),
// // // // //             "new_interval": string(newInterval),
// // // // //         },
// // // // //         CreatedAt:      time.Now(),
// // // // //         UpdatedAt:      time.Now(),
// // // // //     }
// // // // //     s.repo.CreateInvoice(invoice)

// // // // //     // Create history record
// // // // //     history := &models.SubscriptionHistory{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: sub.ID,
// // // // //         SchoolID:       sub.SchoolID,
// // // // //         UserID:         userID,
// // // // //         NewAmount:      &newAmount,
// // // // //         ChangeReason:   "subscription_renewed",
// // // // //         ChangedBy:      userID,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "new_interval": string(newInterval),
// // // // //             "invoice_id":   invoice.ID,
// // // // //         },
// // // // //         CreatedAt:      time.Now(),
// // // // //     }
// // // // //     s.repo.CreateHistory(history)

// // // // //     // Send renewal email
// // // // //     go s.sendSubscriptionRenewedEmail(sub, invoice)

// // // // //     return s.toSubscriptionResponse(sub), nil
// // // // // }

// // // // // // ============================================
// // // // // // PAYMENT INTENT MANAGEMENT
// // // // // // ============================================

// // // // // func (s *SubscriptionService) CreatePaymentIntent(subscriptionID string, req *dto.CreatePaymentIntentRequest, userID string) (*dto.PaymentIntentResponse, error) {
// // // // //     sub, err := s.repo.FindByID(subscriptionID)
// // // // //     if err != nil {
// // // // //         return nil, errors.New("subscription not found")
// // // // //     }

// // // // //     // Create invoice for this payment
// // // // //     invoice := &models.Invoice{
// // // // //         ID:             uuid.New().String(),
// // // // //         SubscriptionID: subscriptionID,
// // // // //         SchoolID:       sub.SchoolID,
// // // // //         UserID:         userID,
// // // // //         InvoiceNumber:  dto.GenerateInvoiceNumber(),
// // // // //         Amount:         sub.Amount,
// // // // //         Total:          sub.Amount,
// // // // //         Currency:       sub.Currency,
// // // // //         Status:         models.InvoicePending,
// // // // //         DueDate:        time.Now().AddDate(0, 0, 7),
// // // // //         CreatedAt:      time.Now(),
// // // // //         UpdatedAt:      time.Now(),
// // // // //     }
// // // // //     s.repo.CreateInvoice(invoice)

// // // // //     // Create payment intent
// // // // //     idempotencyKey := fmt.Sprintf("%s_%d_%d", subscriptionID, time.Now().UnixNano(), invoice.ID)
// // // // //     paymentReq := &payment.PaymentRequest{
// // // // //         SchoolID:    sub.SchoolID,
// // // // //         UserID:      userID,
// // // // //         Amount:      sub.Amount,
// // // // //         Currency:    sub.Currency,
// // // // //         Email:       req.SuccessURL,
// // // // //         SuccessURL:  req.SuccessURL,
// // // // //         CancelURL:   req.CancelURL,
// // // // //         Metadata: map[string]interface{}{
// // // // //             "subscription_id": subscriptionID,
// // // // //             "invoice_id":      invoice.ID,
// // // // //             "payment_type":    "renewal",
// // // // //         },
// // // // //     }

// // // // //     paymentIntent, err := s.paymentService.CreatePayment(context.Background(), sub.Gateway, paymentReq)
// // // // //     if err != nil {
// // // // //         return nil, err
// // // // //     }

// // // // //     dbPaymentIntent := &models.PaymentIntent{
// // // // //         ID:               uuid.New().String(),
// // // // //         SubscriptionID:   subscriptionID,
// // // // //         InvoiceID:        invoice.ID,
// // // // //         SchoolID:         sub.SchoolID,
// // // // //         UserID:           userID,
// // // // //         IdempotencyKey:   idempotencyKey,
// // // // //         Gateway:          sub.Gateway,
// // // // //         Amount:           sub.Amount,
// // // // //         Currency:         sub.Currency,
// // // // //         Reference:        paymentIntent.Reference,
// // // // //         ClientSecret:     paymentIntent.ClientSecret,
// // // // //         AuthorizationURL: paymentIntent.AuthorizationURL,
// // // // //         Status:           models.IntentPending,
// // // // //         ExpiresAt:        timePtr(time.Now().Add(24 * time.Hour)),
// // // // //         CreatedAt:        time.Now(),
// // // // //         UpdatedAt:        time.Now(),
// // // // //     }
// // // // //     s.repo.CreatePaymentIntent(dbPaymentIntent)

// // // // //     return s.toPaymentIntentResponse(dbPaymentIntent), nil
// // // // // }

// // // // // func (s *SubscriptionService) ConfirmPaymentIntent(paymentIntentID string) error {
// // // // //     pi, err := s.repo.FindPaymentIntentByID(paymentIntentID)
// // // // //     if err != nil {
// // // // //         return errors.New("payment intent not found")
// // // // //     }

// // // // //     if pi.IsFinalized {
// // // // //         return errors.New("payment intent already finalized")
// // // // //     }

// // // // //     // Verify with gateway
// // // // //     verification, err := s.paymentService.VerifyPayment(context.Background(), pi.Gateway, pi.Reference)
// // // // //     if err != nil {
// // // // //         return err
// // // // //     }

// // // // //     pi.Status = models.PaymentIntentStatus(verification.Status)
// // // // //     pi.GatewayResponse = verification.GatewayData
    
// // // // //     if verification.Status == "succeeded" {
// // // // //         pi.PaidAt = timePtr(time.Now())
// // // // //         pi.IsFinalized = true
        
// // // // //         // Mark invoice as paid
// // // // //         s.repo.MarkInvoiceAsPaid(pi.InvoiceID, time.Now())
        
// // // // //         // Update subscription
// // // // //         sub, _ := s.repo.FindByID(pi.SubscriptionID)
// // // // //         sub.Status = models.SubStatusActive
// // // // //         sub.LastPaymentDate = timePtr(time.Now())
// // // // //         sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// // // // //         sub.PaymentStatus = models.PaymentPaid
// // // // //         s.repo.Update(sub)
        
// // // // //         // Create payment transaction
// // // // //         transaction := &models.PaymentTransaction{
// // // // //             ID:               uuid.New().String(),
// // // // //             SubscriptionID:   pi.SubscriptionID,
// // // // //             PaymentIntentID:  pi.ID,
// // // // //             InvoiceID:        pi.InvoiceID,
// // // // //             SchoolID:         pi.SchoolID,
// // // // //             UserID:           pi.UserID,
// // // // //             Amount:           pi.Amount,
// // // // //             Currency:         pi.Currency,
// // // // //             PaymentMethod:    models.MethodCard,
// // // // //             PaymentStatus:    models.PaymentPaid,
// // // // //             Gateway:          pi.Gateway,
// // // // //             Reference:        pi.Reference,
// // // // //             PaidAt:           timePtr(time.Now()),
// // // // //             CreatedAt:        time.Now(),
// // // // //             UpdatedAt:        time.Now(),
// // // // //         }
// // // // //         s.repo.CreatePaymentTransaction(transaction)
        
// // // // //         // Send payment success email
// // // // //         go s.sendPaymentSuccessEmail(sub, transaction)
// // // // //     }
    
// // // // //     s.repo.UpdatePaymentIntent(pi)
    
// // // // //     // Create event log
// // // // //     eventLog := &models.PaymentEventLog{
// // // // //         ID:             uuid.New().String(),
// // // // //         PaymentIntentID: pi.ID,
// // // // //         EventType:      "payment_intent_confirmed",
// // // // //         StatusBefore:   string(models.IntentPending),
// // // // //         StatusAfter:    string(pi.Status),
// // // // //         Payload: map[string]interface{}{
// // // // //             "verification": verification,
// // // // //         },
// // // // //         CreatedAt:      time.Now(),
// // // // //     }
// // // // //     s.repo.CreatePaymentEventLog(eventLog)
    
// // // // //     return nil
// // // // // }

// // // // // // ============================================
// // // // // // HELPER FUNCTIONS
// // // // // // ============================================

// // // // // func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
// // // // //     switch interval {
// // // // //     case models.IntervalMonthly:
// // // // //         return start.AddDate(0, 1, 0)
// // // // //     case models.IntervalQuarterly:
// // // // //         return start.AddDate(0, 3, 0)
// // // // //     case models.IntervalYearly:
// // // // //         return start.AddDate(1, 0, 0)
// // // // //     default:
// // // // //         return start.AddDate(0, 1, 0)
// // // // //     }
// // // // // }

// // // // // func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
// // // // //     next := calculateEndDate(currentEnd, interval)
// // // // //     return &next
// // // // // }

// // // // // func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
// // // // //     features := make(models.JSONMap)
    
// // // // //     switch tier {
// // // // //     case models.TierBasic:
// // // // //         features["analytics"] = false
// // // // //         features["api_access"] = false
// // // // //         features["priority_support"] = false
// // // // //         features["bulk_import"] = false
// // // // //         features["white_label"] = false
// // // // //     case models.TierPremium:
// // // // //         features["analytics"] = true
// // // // //         features["api_access"] = true
// // // // //         features["priority_support"] = false
// // // // //         features["bulk_import"] = true
// // // // //         features["white_label"] = false
// // // // //         features["custom_reports"] = true
// // // // //     case models.TierEnterprise:
// // // // //         features["analytics"] = true
// // // // //         features["api_access"] = true
// // // // //         features["priority_support"] = true
// // // // //         features["bulk_import"] = true
// // // // //         features["white_label"] = true
// // // // //         features["custom_reports"] = true
// // // // //         features["dedicated_server"] = true
// // // // //         features["sla"] = true
// // // // //     }
    
// // // // //     return features
// // // // // }

// // // // // func (s *SubscriptionService) createReminderSchedules(subscriptionID string) {
// // // // //     reminderDays := []int{30, 14, 7, 3, 1}
// // // // //     for _, days := range reminderDays {
// // // // //         schedule := &models.ReminderSchedule{
// // // // //             ID:             uuid.New().String(),
// // // // //             SubscriptionID: subscriptionID,
// // // // //             ReminderType:   "expiry",
// // // // //             DaysBefore:     days,
// // // // //             Status:         "pending",
// // // // //             CreatedAt:      time.Now(),
// // // // //             UpdatedAt:      time.Now(),
// // // // //         }
// // // // //         s.repo.CreateReminderSchedule(schedule)
// // // // //     }
// // // // // }

// // // // // func timePtr(t time.Time) *time.Time {
// // // // //     return &t
// // // // // }

// // // // // // ============================================
// // // // // // RESPONSE MAPPERS
// // // // // // ============================================

// // // // // func (s *SubscriptionService) toSubscriptionResponse(sub *models.Subscription) *dto.SubscriptionResponse {
// // // // //     return &dto.SubscriptionResponse{
// // // // //         ID:              sub.ID,
// // // // //         SchoolID:        sub.SchoolID,
// // // // //         Tier:            string(sub.Tier),
// // // // //         Status:          string(sub.Status),
// // // // //         Gateway:         string(sub.Gateway),
// // // // //         Amount:          sub.Amount,
// // // // //         Currency:        string(sub.Currency),
// // // // //         PaymentInterval: string(sub.PaymentInterval),
// // // // //         StartDate:       sub.StartDate,
// // // // //         EndDate:         sub.EndDate,
// // // // //         TrialEndsAt:     sub.TrialEndsAt,
// // // // //         AutoRenew:       sub.AutoRenew,
// // // // //         CancelAtEndDate: sub.CancelAtEndDate,
// // // // //         LastPaymentDate: sub.LastPaymentDate,
// // // // //         NextPaymentDate: sub.NextPaymentDate,
// // // // //         MaxStudents:     sub.MaxStudents,
// // // // //         MaxTeachers:     sub.MaxTeachers,
// // // // //         MaxExams:        sub.MaxExams,
// // // // //         MaxQuestions:    sub.MaxQuestions,
// // // // //         MaxStorageMB:    sub.MaxStorageMB,
// // // // //         Features:        sub.Features,
// // // // //         CreatedAt:       sub.CreatedAt,
// // // // //         UpdatedAt:       sub.UpdatedAt,
// // // // //     }
// // // // // }

// // // // // func (s *SubscriptionService) toPaymentIntentResponse(pi *models.PaymentIntent) *dto.PaymentIntentResponse {
// // // // //     return &dto.PaymentIntentResponse{
// // // // //         ID:               pi.ID,
// // // // //         SubscriptionID:   pi.SubscriptionID,
// // // // //         InvoiceID:        pi.InvoiceID,
// // // // //         Gateway:          string(pi.Gateway),
// // // // //         Amount:           pi.Amount,
// // // // //         Currency:         string(pi.Currency),
// // // // //         Reference:        pi.Reference,
// // // // //         ClientSecret:     pi.ClientSecret,
// // // // //         AuthorizationURL: pi.AuthorizationURL,
// // // // //         AccessCode:       pi.AccessCode,
// // // // //         Status:           string(pi.Status),
// // // // //         ExpiresAt:        pi.ExpiresAt,
// // // // //         PaymentMethod:    string(pi.PaymentMethod),
// // // // //     }
// // // // // }

// // // // // func (s *SubscriptionService) toInvoiceResponse(inv *models.Invoice) *dto.InvoiceResponse {
// // // // //     return &dto.InvoiceResponse{
// // // // //         ID:            inv.ID,
// // // // //         InvoiceNumber: inv.InvoiceNumber,
// // // // //         Amount:        inv.Amount,
// // // // //         Tax:           inv.Tax,
// // // // //         Discount:      inv.Discount,
// // // // //         Total:         inv.Total,
// // // // //         Currency:      string(inv.Currency),
// // // // //         Status:        string(inv.Status),
// // // // //         DueDate:       inv.DueDate,
// // // // //         PaidAt:        inv.PaidAt,
// // // // //         PDFURL:        inv.PDFURL,
// // // // //         Items:         inv.Items,
// // // // //         CreatedAt:     inv.CreatedAt,
// // // // //     }
// // // // // }

// // // // // // ============================================
// // // // // // EMAIL NOTIFICATIONS (To be implemented)
// // // // // // ============================================

// // // // // func (s *SubscriptionService) sendSubscriptionCreatedEmail(sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, email string) {
// // // // //     // TODO: Implement email sending
// // // // //     fmt.Printf("Subscription created email sent to %s\n", email)
// // // // // }

// // // // // func (s *SubscriptionService) sendSubscriptionRenewedEmail(sub *models.Subscription, inv *models.Invoice) {
// // // // //     // TODO: Implement email sending
// // // // //     fmt.Printf("Subscription renewed email sent\n")
// // // // // }

// // // // // func (s *SubscriptionService) sendSubscriptionCancelledEmail(sub *models.Subscription) {
// // // // //     // TODO: Implement email sending
// // // // //     fmt.Printf("Subscription cancelled email sent\n")
// // // // // }

// // // // // func (s *SubscriptionService) sendPaymentSuccessEmail(sub *models.Subscription, transaction *models.PaymentTransaction) {
// // // // //     // TODO: Implement email sending
// // // // //     fmt.Printf("Payment success email sent\n")
// // // // // }