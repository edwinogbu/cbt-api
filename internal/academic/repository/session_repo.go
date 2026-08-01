package repository

import (
     "cbt-api/internal/academic/dto"  // ✅ ADD THIS IMPORT
    "cbt-api/internal/models"
    "time"                            // ✅ ADD THIS IMPORT
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

// Add these new methods

// ListSessions retrieves sessions with pagination and filters
func (r *SessionRepository) ListSessions(filters map[string]interface{}, page, limit int, sortBy, sortOrder string) ([]models.AcademicSession, int64, error) {
    var sessions []models.AcademicSession
    var total int64

    query := r.db.Model(&models.AcademicSession{}).Where("deleted_at IS NULL")

    // Apply filters
    if schoolID, ok := filters["school_id"].(string); ok && schoolID != "" {
        query = query.Where("school_id = ?", schoolID)
    }

    if search, ok := filters["search"].(string); ok && search != "" {
        query = query.Where("name ILIKE ?", "%"+search+"%")
    }

    if status, ok := filters["status"].(string); ok && status != "" && status != "all" {
        now := time.Now()
        switch status {
        case "active":
            query = query.Where("is_current = ?", true)
        case "upcoming":
            query = query.Where("is_current = ? AND start_date > ?", false, now)
        case "ended":
            query = query.Where("is_current = ? AND end_date < ?", false, now)
        case "inactive":
            query = query.Where("is_current = ? AND start_date <= ? AND end_date >= ?", false, now, now)
        }
    }

    if isActive, ok := filters["is_active"].(*bool); ok && isActive != nil {
        query = query.Where("is_active = ?", *isActive)
    }

    if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
        query = query.Where("start_date >= ?", startDate)
    }

    if endDate, ok := filters["end_date"].(string); ok && endDate != "" {
        query = query.Where("end_date <= ?", endDate)
    }

    // Count total
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // Apply sorting
    if sortBy == "" {
        sortBy = "created_at"
    }
    if sortOrder == "" {
        sortOrder = "desc"
    }
    query = query.Order(sortBy + " " + sortOrder)

    // Apply pagination
    offset := (page - 1) * limit
    if err := query.Offset(offset).Limit(limit).Find(&sessions).Error; err != nil {
        return nil, 0, err
    }

    return sessions, total, nil
}

func (r *SessionRepository) GetSessionStats(schoolID string) (*dto.SessionStatsResponse, error) {
    var stats dto.SessionStatsResponse
    now := time.Now().UTC()

    query := r.db.Model(&models.AcademicSession{}).
        Select(`
            COUNT(*) as total,
            COUNT(CASE WHEN is_current = true THEN 1 END) as active,
            COUNT(CASE WHEN is_current = false AND start_date > ? THEN 1 END) as upcoming,
            COUNT(CASE WHEN is_current = false AND end_date < ? THEN 1 END) as ended,
            COUNT(CASE WHEN is_current = false AND start_date <= ? AND end_date >= ? THEN 1 END) as inactive
        `, now, now, now, now).
        Where("deleted_at IS NULL")

    if schoolID != "" {
        query = query.Where("school_id = ?", schoolID)
    }

    if err := query.Scan(&stats).Error; err != nil {
        return nil, err
    }

    return &stats, nil
}

// BulkDeleteSessions deletes multiple sessions by IDs
func (r *SessionRepository) BulkDeleteSessions(ids []string) (int64, error) {
    result := r.db.Where("id IN ?", ids).Delete(&models.AcademicSession{})
    return result.RowsAffected, result.Error
}

// GetSessionTimeline retrieves sessions ordered by start_date for timeline view
func (r *SessionRepository) GetSessionTimeline(schoolID string) ([]models.AcademicSession, error) {
    var sessions []models.AcademicSession
    query := r.db.Where("deleted_at IS NULL")

    if schoolID != "" {
        query = query.Where("school_id = ?", schoolID)
    }

    err := query.Order("start_date ASC").Find(&sessions).Error
    return sessions, err
}