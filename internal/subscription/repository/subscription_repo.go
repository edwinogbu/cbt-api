package repository

import (
    "time"

    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type SubscriptionRepository struct {
    db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
    return &SubscriptionRepository{db: db}
}

// ============================================
// SUBSCRIPTION OPERATIONS
// ============================================

func (r *SubscriptionRepository) Create(sub *models.Subscription) error {
    return r.db.Create(sub).Error
}

func (r *SubscriptionRepository) FindByID(id string) (*models.Subscription, error) {
    var sub models.Subscription
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).
        Preload("Invoices").
        Preload("PaymentIntents").
        Preload("PaymentTransactions").
        First(&sub).Error
    if err != nil {
        return nil, err
    }
    return &sub, nil
}

func (r *SubscriptionRepository) FindBySchool(schoolID string) ([]models.Subscription, error) {
    var subs []models.Subscription
    err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
        Order("created_at DESC").
        Find(&subs).Error
    return subs, err
}

func (r *SubscriptionRepository) FindCurrentBySchool(schoolID string) (*models.Subscription, error) {
    var sub models.Subscription
    err := r.db.Where("school_id = ? AND status IN (?, ?, ?) AND end_date > NOW() AND deleted_at IS NULL", 
        schoolID, models.SubStatusActive, models.SubStatusTrial, models.SubStatusPending).
        Order("created_at DESC").
        First(&sub).Error
    if err != nil {
        return nil, err
    }
    return &sub, nil
}

func (r *SubscriptionRepository) FindActive() ([]models.Subscription, error) {
    var subs []models.Subscription
    err := r.db.Where("status = ? AND end_date > NOW() AND deleted_at IS NULL", models.SubStatusActive).
        Find(&subs).Error
    return subs, err
}

func (r *SubscriptionRepository) FindExpiringSoon(days int) ([]models.Subscription, error) {
    threshold := time.Now().AddDate(0, 0, days)
    var subs []models.Subscription
    err := r.db.Where("status = ? AND end_date <= ? AND end_date > NOW() AND deleted_at IS NULL", 
        models.SubStatusActive, threshold).
        Find(&subs).Error
    return subs, err
}

func (r *SubscriptionRepository) Update(sub *models.Subscription) error {
    return r.db.Save(sub).Error
}

func (r *SubscriptionRepository) UpdateStatus(id string, status models.SubscriptionStatus) error {
    return r.db.Model(&models.Subscription{}).Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":     status,
            "updated_at": time.Now(),
        }).Error
}

func (r *SubscriptionRepository) Cancel(id string, cancelImmediately bool) error {
    updates := map[string]interface{}{
        "cancel_at_end_date": true,
        "updated_at":         time.Now(),
    }
    if cancelImmediately {
        updates["status"] = models.SubStatusPending
    }
    return r.db.Model(&models.Subscription{}).Where("id = ?", id).
        Updates(updates).Error
}

func (r *SubscriptionRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.Subscription{}).Error
}

// ============================================
// INVOICE OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreateInvoice(invoice *models.Invoice) error {
    return r.db.Create(invoice).Error
}

func (r *SubscriptionRepository) FindInvoiceByID(id string) (*models.Invoice, error) {
    var invoice models.Invoice
    err := r.db.Where("id = ?", id).First(&invoice).Error
    if err != nil {
        return nil, err
    }
    return &invoice, nil
}

func (r *SubscriptionRepository) FindInvoiceByNumber(number string) (*models.Invoice, error) {
    var invoice models.Invoice
    err := r.db.Where("invoice_number = ?", number).First(&invoice).Error
    return &invoice, err
}

func (r *SubscriptionRepository) FindInvoicesBySubscription(subscriptionID string) ([]models.Invoice, error) {
    var invoices []models.Invoice
    err := r.db.Where("subscription_id = ?", subscriptionID).
        Order("created_at DESC").
        Find(&invoices).Error
    return invoices, err
}

func (r *SubscriptionRepository) UpdateInvoice(invoice *models.Invoice) error {
    return r.db.Save(invoice).Error
}

func (r *SubscriptionRepository) MarkInvoiceAsPaid(id string, paidAt time.Time) error {
    return r.db.Model(&models.Invoice{}).Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":   models.InvoicePaid,
            "paid_at":  paidAt,
            "updated_at": time.Now(),
        }).Error
}

// ============================================
// PAYMENT INTENT OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreatePaymentIntent(pi *models.PaymentIntent) error {
    return r.db.Create(pi).Error
}

func (r *SubscriptionRepository) FindPaymentIntentByID(id string) (*models.PaymentIntent, error) {
    var pi models.PaymentIntent
    err := r.db.Where("id = ?", id).First(&pi).Error
    if err != nil {
        return nil, err
    }
    return &pi, nil
}

func (r *SubscriptionRepository) FindPaymentIntentByReference(reference string) (*models.PaymentIntent, error) {
    var pi models.PaymentIntent
    err := r.db.Where("reference = ?", reference).First(&pi).Error
    return &pi, err
}

func (r *SubscriptionRepository) FindPaymentIntentByIdempotencyKey(key string) (*models.PaymentIntent, error) {
    var pi models.PaymentIntent
    err := r.db.Where("idempotency_key = ?", key).First(&pi).Error
    if err != nil {
        return nil, err
    }
    return &pi, nil
}

func (r *SubscriptionRepository) UpdatePaymentIntent(pi *models.PaymentIntent) error {
    return r.db.Save(pi).Error
}

func (r *SubscriptionRepository) UpdatePaymentIntentStatus(id string, status models.PaymentIntentStatus) error {
    return r.db.Model(&models.PaymentIntent{}).Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":     status,
            "updated_at": time.Now(),
        }).Error
}

// ============================================
// PAYMENT TRANSACTION OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreatePaymentTransaction(pt *models.PaymentTransaction) error {
    return r.db.Create(pt).Error
}

func (r *SubscriptionRepository) FindPaymentTransactionByReference(reference string) (*models.PaymentTransaction, error) {
    var pt models.PaymentTransaction
    err := r.db.Where("reference = ?", reference).First(&pt).Error
    return &pt, err
}

func (r *SubscriptionRepository) FindPaymentTransactionsBySubscription(subscriptionID string) ([]models.PaymentTransaction, error) {
    var pts []models.PaymentTransaction
    err := r.db.Where("subscription_id = ?", subscriptionID).
        Order("created_at DESC").
        Find(&pts).Error
    return pts, err
}

// ============================================
// WEBHOOK EVENT OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreateWebhookEvent(event *models.WebhookEvent) error {
    return r.db.Create(event).Error
}

func (r *SubscriptionRepository) FindWebhookEventByIdempotencyKey(key string) (*models.WebhookEvent, error) {
    var event models.WebhookEvent
    err := r.db.Where("idempotency_key = ?", key).First(&event).Error
    if err != nil {
        return nil, err
    }
    return &event, nil
}

func (r *SubscriptionRepository) UpdateWebhookEventStatus(id string, status models.WebhookStatus, errMsg string) error {
    updates := map[string]interface{}{
        "status":       status,
        "processed_at": time.Now(),
        "updated_at":   time.Now(),
    }
    if errMsg != "" {
        updates["error_message"] = errMsg
    }
    return r.db.Model(&models.WebhookEvent{}).Where("id = ?", id).
        Updates(updates).Error
}

// ============================================
// SUBSCRIPTION HISTORY OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreateHistory(history *models.SubscriptionHistory) error {
    return r.db.Create(history).Error
}

func (r *SubscriptionRepository) FindHistoryBySubscription(subscriptionID string) ([]models.SubscriptionHistory, error) {
    var histories []models.SubscriptionHistory
    err := r.db.Where("subscription_id = ?", subscriptionID).
        Order("created_at DESC").
        Find(&histories).Error
    return histories, err
}

// ============================================
// EMAIL NOTIFICATION OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreateEmailNotification(notification *models.EmailNotification) error {
    return r.db.Create(notification).Error
}

func (r *SubscriptionRepository) HasReminderBeenSent(subscriptionID string, daysBefore int) (bool, error) {
    var count int64
    err := r.db.Model(&models.ReminderSchedule{}).
        Where("subscription_id = ? AND days_before = ? AND status = 'sent'", 
            subscriptionID, daysBefore).
        Count(&count).Error
    return count > 0, err
}

// ============================================
// REMINDER SCHEDULE OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreateReminderSchedule(schedule *models.ReminderSchedule) error {
    return r.db.Create(schedule).Error
}

func (r *SubscriptionRepository) FindPendingReminders() ([]models.ReminderSchedule, error) {
    var schedules []models.ReminderSchedule
    err := r.db.Where("status = 'pending' AND sent_at IS NULL").
        Find(&schedules).Error
    return schedules, err
}

func (r *SubscriptionRepository) UpdateReminderScheduleStatus(id string, status string) error {
    return r.db.Model(&models.ReminderSchedule{}).Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":   status,
            "sent_at":  time.Now(),
            "updated_at": time.Now(),
        }).Error
}

// ============================================
// PAYMENT EVENT LOG OPERATIONS
// ============================================

func (r *SubscriptionRepository) CreatePaymentEventLog(log *models.PaymentEventLog) error {
    return r.db.Create(log).Error
}

func (r *SubscriptionRepository) FindPaymentEventLogsByPaymentIntent(paymentIntentID string) ([]models.PaymentEventLog, error) {
    var logs []models.PaymentEventLog
    err := r.db.Where("payment_intent_id = ?", paymentIntentID).
        Order("created_at ASC").
        Find(&logs).Error
    return logs, err
}


func (r *SubscriptionRepository) FindByGatewaySubscriptionID(gatewaySubID string) (*models.Subscription, error) {
    var sub models.Subscription
    err := r.db.Where("gateway_subscription_id = ? AND deleted_at IS NULL", gatewaySubID).First(&sub).Error
    if err != nil {
        return nil, err
    }
    return &sub, nil
}

// func (r *SubscriptionRepository) FindByGatewaySubscriptionID(gatewaySubID string) (*models.Subscription, error) {
//     var sub models.Subscription
//     err := r.db.Where("gateway_subscription_id = ? AND deleted_at IS NULL", gatewaySubID).First(&sub).Error
//     if err != nil {
//         return nil, err
//     }
//     return &sub, nil
// }