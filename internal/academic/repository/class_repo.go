package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type ClassRepository struct {
    db *gorm.DB
}

func NewClassRepository(db *gorm.DB) *ClassRepository {
    return &ClassRepository{db: db}
}

func (r *ClassRepository) Create(class *models.Class) error {
    return r.db.Create(class).Error
}

func (r *ClassRepository) FindByID(id string) (*models.Class, error) {
    var class models.Class
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&class).Error
    if err != nil {
        return nil, err
    }
    return &class, nil
}

func (r *ClassRepository) FindBySchool(schoolID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Order("created_at DESC").Find(&classes).Error
    return classes, err
}

func (r *ClassRepository) FindBySession(sessionID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.Where("session_id = ? AND deleted_at IS NULL", sessionID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Order("class_level_id ASC, class_arm_id ASC").Find(&classes).Error
    return classes, err
}

func (r *ClassRepository) FindBySchoolAndSession(schoolID, sessionID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Order("class_level_id ASC, class_arm_id ASC").Find(&classes).Error
    return classes, err
}

func (r *ClassRepository) FindByTeacher(teacherID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.Where("teacher_id = ? AND deleted_at IS NULL", teacherID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Find(&classes).Error
    return classes, err
}

func (r *ClassRepository) Update(class *models.Class) error {
    return r.db.Save(class).Error
}

func (r *ClassRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.Class{}).Error
}

// func (r *ClassRepository) GetStudentCount(classID string) (int64, error) {
//     var count int64
//     err := r.db.Model(&models.Student{}).
//         Where("class_id = ? AND deleted_at IS NULL", classID).
//         Count(&count).Error
//     return count, err
// }

// GetStudentCount returns the number of students in a class
func (r *ClassRepository) GetStudentCount(classID string) (int64, error) {
    var count int64
    err := r.db.Model(&models.Student{}).
        Where("class_id = ? AND deleted_at IS NULL", classID).
        Count(&count).Error
    return count, err
}