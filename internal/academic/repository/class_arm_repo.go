package repository

import (
    "cbt-api/internal/models"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ClassArmRepository struct {
    db *gorm.DB
}

func NewClassArmRepository(db *gorm.DB) *ClassArmRepository {
    return &ClassArmRepository{db: db}
}

func (r *ClassArmRepository) Create(arm *models.ClassArm) error {
    return r.db.Create(arm).Error
}

func (r *ClassArmRepository) FindByID(id string) (*models.ClassArm, error) {
    var arm models.ClassArm
    uuid, err := uuid.Parse(id)
    if err != nil {
        return nil, err
    }
    err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&arm).Error
    if err != nil {
        return nil, err
    }
    return &arm, nil
}

func (r *ClassArmRepository) FindBySchool(schoolID string) ([]models.ClassArm, error) {
    var arms []models.ClassArm
    uuid, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, err
    }
    err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
        Order("sort_order ASC, name ASC").Find(&arms).Error
    return arms, err
}

func (r *ClassArmRepository) Update(arm *models.ClassArm) error {
    return r.db.Save(arm).Error
}

func (r *ClassArmRepository) Delete(id string) error {
    uuid, err := uuid.Parse(id)
    if err != nil {
        return err
    }
    return r.db.Where("id = ?", uuid).Delete(&models.ClassArm{}).Error
}

// package repository

// import (
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
// )

// type ClassArmRepository struct {
//     db *gorm.DB
// }

// func NewClassArmRepository(db *gorm.DB) *ClassArmRepository {
//     return &ClassArmRepository{db: db}
// }

// func (r *ClassArmRepository) Create(arm *models.ClassArm) error {
//     return r.db.Create(arm).Error
// }

// func (r *ClassArmRepository) FindByID(id string) (*models.ClassArm, error) {
//     var arm models.ClassArm
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&arm).Error
//     if err != nil {
//         return nil, err
//     }
//     return &arm, nil
// }

// func (r *ClassArmRepository) FindBySchool(schoolID string) ([]models.ClassArm, error) {
//     var arms []models.ClassArm
//     err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
//         Order("sort_order ASC, name ASC").Find(&arms).Error
//     return arms, err
// }

// func (r *ClassArmRepository) Update(arm *models.ClassArm) error {
//     return r.db.Save(arm).Error
// }

// func (r *ClassArmRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.ClassArm{}).Error
// }