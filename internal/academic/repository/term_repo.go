package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type TermRepository struct {
    db *gorm.DB
}

func NewTermRepository(db *gorm.DB) *TermRepository {
    return &TermRepository{db: db}
}

func (r *TermRepository) Create(term *models.Term) error {
    return r.db.Create(term).Error
}

func (r *TermRepository) FindByID(id string) (*models.Term, error) {
    var term models.Term
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&term).Error
    if err != nil {
        return nil, err
    }
    return &term, nil
}

func (r *TermRepository) FindBySession(sessionID string) ([]models.Term, error) {
    var terms []models.Term
    err := r.db.Where("session_id = ? AND deleted_at IS NULL", sessionID).
        Order("term_number ASC").Find(&terms).Error
    return terms, err
}

func (r *TermRepository) FindCurrentBySession(sessionID string) (*models.Term, error) {
    var term models.Term
    err := r.db.Where("session_id = ? AND is_current = true AND deleted_at IS NULL", sessionID).
        First(&term).Error
    if err != nil {
        return nil, err
    }
    return &term, nil
}

func (r *TermRepository) Update(term *models.Term) error {
    return r.db.Save(term).Error
}

func (r *TermRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.Term{}).Error
}

func (r *TermRepository) SetCurrent(sessionID, termID string) error {
    // First, unset current for all terms in this session
    err := r.db.Model(&models.Term{}).
        Where("session_id = ?", sessionID).
        Update("is_current", false).Error
    if err != nil {
        return err
    }
    
    // Then set the new current term
    return r.db.Model(&models.Term{}).
        Where("id = ?", termID).
        Update("is_current", true).Error
}