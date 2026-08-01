package repository

import (
	"cbt-api/internal/models"
	"gorm.io/gorm"
)

type SubjectRepository struct {
	db *gorm.DB
}

func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
	return &SubjectRepository{db: db}
}

// Create
func (r *SubjectRepository) Create(subject *models.Subject) error {
	return r.db.Create(subject).Error
}

// FindByID - NOW ACCEPTS string
func (r *SubjectRepository) FindByID(id string) (*models.Subject, error) {
	var subject models.Subject
	err := r.db.Where("id = ?", id).First(&subject).Error
	return &subject, err
}

// FindByCode – only returns non‑deleted subjects
func (r *SubjectRepository) FindByCode(code string) (*models.Subject, error) {
	var subject models.Subject
	err := r.db.Where("code = ? AND deleted_at IS NULL", code).First(&subject).Error
	if err != nil {
		return nil, err
	}
	return &subject, nil
}

// Update
func (r *SubjectRepository) Update(subject *models.Subject) error {
	return r.db.Save(subject).Error
}

// Delete – soft delete (uses DeletedAt) - NOW ACCEPTS string
func (r *SubjectRepository) Delete(id string) error {
	return r.db.Delete(&models.Subject{}, "id = ?", id).Error
}

// List – paginated, excluding soft‑deleted
func (r *SubjectRepository) List(page, limit int) ([]models.Subject, int64, error) {
	var subjects []models.Subject
	offset := (page - 1) * limit
	var total int64
	query := r.db.Model(&models.Subject{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Offset(offset).Limit(limit).Find(&subjects).Error
	return subjects, total, err
}

// ListActive – only active subjects
func (r *SubjectRepository) ListActive() ([]models.Subject, error) {
	var subjects []models.Subject
	err := r.db.Where("is_active = ?", true).Find(&subjects).Error
	return subjects, err
}
