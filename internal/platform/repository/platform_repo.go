package repository

import (
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

type PlatformRepository struct {
	db *gorm.DB
}

func NewPlatformRepository(db *gorm.DB) *PlatformRepository {
	return &PlatformRepository{db: db}
}

// ============================================
// TENANT REGISTRY
// ============================================

func (r *PlatformRepository) ListSchools(status, search string, page, limit int) ([]models.School, int64, error) {
	var schools []models.School
	var total int64

	query := r.db.Model(&models.School{}).Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR email ILIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&schools).Error
	return schools, total, err
}

func (r *PlatformRepository) FindSchoolByID(id string) (*models.School, error) {
	var school models.School
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

func (r *PlatformRepository) UpdateSchool(school *models.School) error {
	return r.db.Save(school).Error
}

func (r *PlatformRepository) FindCurrentSubscription(schoolID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
		Order("created_at DESC").First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// ============================================
// AUDIT LOG
// ============================================

func (r *PlatformRepository) CreateAuditLog(entry *models.TenantAuditLog) error {
	return r.db.Create(entry).Error
}

func (r *PlatformRepository) ListAuditLogs(action string, page, limit int) ([]models.TenantAuditLog, int64, error) {
	var logs []models.TenantAuditLog
	var total int64

	query := r.db.Model(&models.TenantAuditLog{})
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}

// ============================================
// LAN SERVER CONFIG
// ============================================

func (r *PlatformRepository) FindLANConfigBySchool(schoolID string) (*models.LANServerConfig, error) {
	var cfg models.LANServerConfig
	err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *PlatformRepository) UpsertLANConfig(cfg *models.LANServerConfig) error {
	if cfg.ID == "" {
		return r.db.Create(cfg).Error
	}
	return r.db.Save(cfg).Error
}

func (r *PlatformRepository) CreateLANConfigHistory(entry *models.LANServerConfigHistory) error {
	return r.db.Create(entry).Error
}

func (r *PlatformRepository) ListLANConfigHistory(schoolID string, limit int) ([]models.LANServerConfigHistory, error) {
	var history []models.LANServerConfigHistory
	err := r.db.Where("school_id = ?", schoolID).
		Order("created_at DESC").Limit(limit).Find(&history).Error
	return history, err
}
