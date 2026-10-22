package repository

import (
	"cbt-api/internal/models"
	"gorm.io/gorm"
)

type ParentRepository struct {
	db *gorm.DB
}

func NewParentRepository(db *gorm.DB) *ParentRepository {
	return &ParentRepository{db: db}
}

// CreateLink creates a parent-student association
func (r *ParentRepository) CreateLink(link *models.ParentStudent) error {
	return r.db.Create(link).Error
}

// FindLinksByParentID returns all parent-student links for a given parent
func (r *ParentRepository) FindLinksByParentID(parentID string) ([]models.ParentStudent, error) {
	var links []models.ParentStudent
	err := r.db.Where("parent_id = ?", parentID).Preload("Student").Preload("Student.Class").Find(&links).Error
	return links, err
}

// FindStudentsByAdmissionNumberInSchool returns students with that
// admission number WITHIN one school. schoolID must come from the
// registering parent's own, server-validated school (their school_code),
// never from client input directly - admission numbers are often
// short/sequential, not secrets, and without this scope any registrant
// could auto-link themselves as "parent" to a student at a completely
// different school just by guessing a number, crossing the tenant
// boundary this platform otherwise enforces everywhere else.
func (r *ParentRepository) FindStudentsByAdmissionNumberInSchool(admissionNo, schoolID string) ([]models.Student, error) {
	var students []models.Student
	err := r.db.Where("admission_no = ? AND school_id = ? AND deleted_at IS NULL", admissionNo, schoolID).Find(&students).Error
	return students, err
}

// CheckIfLinked verifies if a parent is already linked to a student
func (r *ParentRepository) IsLinked(parentID, studentID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.ParentStudent{}).Where("parent_id = ? AND student_id = ?", parentID, studentID).Count(&count).Error
	return count > 0, err
}

// ============================================
// ADDED: FindStudentsByParentID
// ============================================

// FindStudentsByParentID returns all students linked to a parent
func (r *ParentRepository) FindStudentsByParentID(parentID string) ([]models.Student, error) {
	var students []models.Student
	err := r.db.Table("students").
		Joins("JOIN parent_students ON parent_students.student_id = students.id").
		Where("parent_students.parent_id = ? AND students.deleted_at IS NULL", parentID).
		Find(&students).Error
	return students, err
}



// package repository

// import (
// 	"cbt-api/internal/models"
// 	"gorm.io/gorm"
// )

// type ParentRepository struct {
// 	db *gorm.DB
// }

// func NewParentRepository(db *gorm.DB) *ParentRepository {
// 	return &ParentRepository{db: db}
// }

// // CreateLink creates a parent-student association
// func (r *ParentRepository) CreateLink(link *models.ParentStudent) error {
// 	return r.db.Create(link).Error
// }

// // FindLinksByParentID returns all parent-student links for a given parent
// func (r *ParentRepository) FindLinksByParentID(parentID string) ([]models.ParentStudent, error) {
// 	var links []models.ParentStudent
// 	err := r.db.Where("parent_id = ?", parentID).Preload("Student").Preload("Student.Class").Find(&links).Error
// 	return links, err
// }

// // FindStudentByAdmissionNumber returns all students with that admission number (supports multiple)
// func (r *ParentRepository) FindStudentsByAdmissionNumber(admissionNo string) ([]models.Student, error) {
// 	var students []models.Student
// 	err := r.db.Where("admission_no = ? AND deleted_at IS NULL", admissionNo).Find(&students).Error
// 	return students, err
// }

// // CheckIfLinked verifies if a parent is already linked to a student
// func (r *ParentRepository) IsLinked(parentID, studentID string) (bool, error) {
// 	var count int64
// 	err := r.db.Model(&models.ParentStudent{}).Where("parent_id = ? AND student_id = ?", parentID, studentID).Count(&count).Error
// 	return count > 0, err
// }