package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"

)

type SubscriptionRepository struct {
    db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
    return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(sub *models.Subscription) error {
    return r.db.Create(sub).Error
}

func (r *SubscriptionRepository) FindByID(id string) (*models.Subscription, error) {
    var sub models.Subscription
    err := r.db.Where("id = ?", id).First(&sub).Error
    if err != nil {
        return nil, err
    }
    return &sub, nil
}

func (r *SubscriptionRepository) FindCurrentBySchool(schoolID string) (*models.Subscription, error) {
    var sub models.Subscription
    err := r.db.Where("school_id = ? AND status = ? AND end_date > NOW()", schoolID, models.StatusActive).
        Order("created_at DESC").First(&sub).Error
    if err != nil {
        return nil, err
    }
    return &sub, nil
}

func (r *SubscriptionRepository) FindBySchool(schoolID string) ([]models.Subscription, error) {
    var subs []models.Subscription
    err := r.db.Where("school_id = ?", schoolID).Order("created_at DESC").Find(&subs).Error
    return subs, err
}

func (r *SubscriptionRepository) Update(sub *models.Subscription) error {
    return r.db.Save(sub).Error
}

func (r *SubscriptionRepository) Cancel(id string) error {
    return r.db.Model(&models.Subscription{}).Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":          models.SubStatusCancelled,
            "auto_renew":      false,
            "cancel_at_end_date": true,
        }).Error
}

func (r *SubscriptionRepository) CreateHistory(history *models.SubscriptionHistory) error {
    return r.db.Create(history).Error
}

func (r *SubscriptionRepository) CreatePayment(payment *models.PaymentTransaction) error {
    return r.db.Create(payment).Error
}