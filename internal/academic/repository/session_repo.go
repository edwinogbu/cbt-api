package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type SessionRepository struct {
    db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
    return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(session *models.AcademicSession) error {
    return r.db.Create(session).Error
}

func (r *SessionRepository) FindByID(id string) (*models.AcademicSession, error) {
    var session models.AcademicSession
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&session).Error
    if err != nil {
        return nil, err
    }
    return &session, nil
}

func (r *SessionRepository) FindBySchool(schoolID string) ([]models.AcademicSession, error) {
    var sessions []models.AcademicSession
    err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
        Order("start_date DESC").Find(&sessions).Error
    return sessions, err
}

func (r *SessionRepository) FindCurrentBySchool(schoolID string) (*models.AcademicSession, error) {
    var session models.AcademicSession
    err := r.db.Where("school_id = ? AND is_current = true AND deleted_at IS NULL", schoolID).
        First(&session).Error
    if err != nil {
        return nil, err
    }
    return &session, nil
}

func (r *SessionRepository) Update(session *models.AcademicSession) error {
    return r.db.Save(session).Error
}

func (r *SessionRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.AcademicSession{}).Error
}

func (r *SessionRepository) SetCurrent(schoolID, sessionID string) error {
    // First, unset current for all sessions in this school
    err := r.db.Model(&models.AcademicSession{}).
        Where("school_id = ?", schoolID).
        Update("is_current", false).Error
    if err != nil {
        return err
    }
    
    // Then set the new current session
    return r.db.Model(&models.AcademicSession{}).
        Where("id = ?", sessionID).
        Update("is_current", true).Error
}