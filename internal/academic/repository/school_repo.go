package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type SchoolRepository struct {
    db *gorm.DB
}

func NewSchoolRepository(db *gorm.DB) *SchoolRepository {
    return &SchoolRepository{db: db}
}

func (r *SchoolRepository) Create(school *models.School) error {
    return r.db.Create(school).Error
}

func (r *SchoolRepository) FindByID(id string) (*models.School, error) {
    var school models.School
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&school).Error
    if err != nil {
        return nil, err
    }
    return &school, nil
}

func (r *SchoolRepository) FindByCode(code string) (*models.School, error) {
    var school models.School
    err := r.db.Where("code = ? AND deleted_at IS NULL", code).First(&school).Error
    if err != nil {
        return nil, err
    }
    return &school, nil
}

func (r *SchoolRepository) FindAll(page, limit int) ([]models.School, int64, error) {
    var schools []models.School
    var total int64
    
    offset := (page - 1) * limit
    
    query := r.db.Model(&models.School{}).Where("deleted_at IS NULL")
    query.Count(&total)
    
    err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&schools).Error
    return schools, total, err
}

func (r *SchoolRepository) Update(school *models.School) error {
    return r.db.Save(school).Error
}

func (r *SchoolRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.School{}).Error
}

func (r *SchoolRepository) CodeExists(code string) (bool, error) {
    var count int64
    err := r.db.Model(&models.School{}).Where("code = ? AND deleted_at IS NULL", code).Count(&count).Error
    return count > 0, err
}