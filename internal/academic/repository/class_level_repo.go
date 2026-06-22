package repository

import (
    "cbt-api/internal/models"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ClassLevelRepository struct {
    db *gorm.DB
}

func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
    return &ClassLevelRepository{db: db}
}

func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
    return r.db.Create(level).Error
}

func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
    var level models.ClassLevel
    uuid, err := uuid.Parse(id)
    if err != nil {
        return nil, err
    }
    err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&level).Error
    if err != nil {
        return nil, err
    }
    return &level, nil
}

func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
    var levels []models.ClassLevel
    uuid, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, err
    }
    err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
        Order("level_number ASC, sort_order ASC").Find(&levels).Error
    return levels, err
}

func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
    return r.db.Save(level).Error
}

func (r *ClassLevelRepository) Delete(id string) error {
    uuid, err := uuid.Parse(id)
    if err != nil {
        return err
    }
    return r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{}).Error
}


func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
    var levels []models.ClassLevel
    uuid, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, err
    }
    err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", uuid, category).
        Order("level_number ASC").Find(&levels).Error
    return levels, err
}

// package repository

// import (
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
// )

// type ClassLevelRepository struct {
//     db *gorm.DB
// }

// func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
//     return &ClassLevelRepository{db: db}
// }

// func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
//     return r.db.Create(level).Error
// }

// func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
//     var level models.ClassLevel
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&level).Error
//     if err != nil {
//         return nil, err
//     }
//     return &level, nil
// }

// func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
//     var levels []models.ClassLevel
//     err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
//         Order("level_number ASC, sort_order ASC").Find(&levels).Error
//     return levels, err
// }

// func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
//     var levels []models.ClassLevel
//     err := r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", schoolID, category).
//         Order("level_number ASC").Find(&levels).Error
//     return levels, err
// }

// func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
//     return r.db.Save(level).Error
// }

// func (r *ClassLevelRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.ClassLevel{}).Error
// }