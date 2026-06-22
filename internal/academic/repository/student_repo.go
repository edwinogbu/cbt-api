package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type StudentRepository struct {
    db *gorm.DB
}

func NewStudentRepository(db *gorm.DB) *StudentRepository {
    return &StudentRepository{db: db}
}

func (r *StudentRepository) Create(student *models.Student) error {
    return r.db.Create(student).Error
}

func (r *StudentRepository) FindByID(id string) (*models.Student, error) {
    var student models.Student
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&student).Error
    if err != nil {
        return nil, err
    }
    return &student, nil
}

func (r *StudentRepository) FindByUserID(userID string) (*models.Student, error) {
    var student models.Student
    err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).First(&student).Error
    if err != nil {
        return nil, err
    }
    return &student, nil
}

// func (r *StudentRepository) FindByAdmissionNo(admissionNo string) (*models.Student, error) {
//     var student models.Student
//     err := r.db.Where("admission_no = ? AND deleted_at IS NULL", admissionNo).First(&student).Error
//     if err != nil {
//         return nil, err
//     }
//     return &student, nil
// }

func (r *StudentRepository) FindBySchool(schoolID string, page, limit int) ([]models.Student, int64, error) {
    var students []models.Student
    var total int64
    
    offset := (page - 1) * limit
    
    query := r.db.Model(&models.Student{}).Where("school_id = ? AND deleted_at IS NULL", schoolID)
    query.Count(&total)
    
    err := query.Offset(offset).Limit(limit).
        Order("created_at DESC").
        Find(&students).Error
    
    return students, total, err
}

func (r *StudentRepository) FindByClass(classID string, page, limit int) ([]models.Student, int64, error) {
    var students []models.Student
    var total int64
    
    offset := (page - 1) * limit
    
    query := r.db.Model(&models.Student{}).Where("class_id = ? AND deleted_at IS NULL", classID)
    query.Count(&total)
    
    err := query.Offset(offset).Limit(limit).
        Order("created_at DESC").
        Find(&students).Error
    
    return students, total, err
}

func (r *StudentRepository) Update(student *models.Student) error {
    return r.db.Save(student).Error
}

func (r *StudentRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.Student{}).Error
}

func (r *StudentRepository) TransferClass(studentID, newClassID string) error {
    return r.db.Model(&models.Student{}).
        Where("id = ?", studentID).
        Update("class_id", newClassID).Error
}

func (r *StudentRepository) GetStudentCountByClass(classID string) (int64, error) {
    var count int64
    err := r.db.Model(&models.Student{}).
        Where("class_id = ? AND deleted_at IS NULL", classID).
        Count(&count).Error
    return count, err
}

func (r *StudentRepository) FindByAdmissionNo(admissionNo string) (*models.Student, error) {
    var student models.Student
    err := r.db.Where("admission_no = ? AND deleted_at IS NULL", admissionNo).First(&student).Error
    if err != nil {
        return nil, err
    }
    return &student, nil
}
// func (r *StudentRepository) FindByAdmissionNo(admissionNo string) (*models.Student, error) {
//     var student models.Student
//     err := r.db.Where("admission_no = ? AND deleted_at IS NULL", admissionNo).First(&student).Error
//     if err != nil {
//         return nil, err
//     }
//     return &student, nil
// }