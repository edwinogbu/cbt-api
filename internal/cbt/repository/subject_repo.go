package repository

import (
	"cbt-api/internal/models"
	"github.com/google/uuid"
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

// FindByID
func (r *SubjectRepository) FindByID(id uuid.UUID) (*models.Subject, error) {
	var subject models.Subject
	err := r.db.Where("id = ?", id).First(&subject).Error
	return &subject, err
}

// // FindByCode – for uniqueness check (global, not per school)
// func (r *SubjectRepository) FindByCode(code string) (*models.Subject, error) {
// 	var subject models.Subject
// 	err := r.db.Where("code = ?", code).First(&subject).Error
// 	return &subject, err
// }


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

// Delete – soft delete (uses DeletedAt)
func (r *SubjectRepository) Delete(id uuid.UUID) error {
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


// package repository

// import (
// 	"cbt-api/internal/models"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// type SubjectRepository struct {
// 	db *gorm.DB
// }

// func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
// 	return &SubjectRepository{db: db}
// }

// // Create
// func (r *SubjectRepository) Create(subject *models.Subject) error {
// 	return r.db.Create(subject).Error
// }

// // FindByID – returns soft‑deleted records as well (or use Unscoped? we'll use normal query)
// func (r *SubjectRepository) FindByID(id uuid.UUID) (*models.Subject, error) {
// 	var subject models.Subject
// 	err := r.db.Where("id = ?", id).First(&subject).Error
// 	return &subject, err
// }

// // FindByCode – for uniqueness check
// func (r *SubjectRepository) FindByCode(code string) (*models.Subject, error) {
// 	var subject models.Subject
// 	err := r.db.Where("code = ?", code).First(&subject).Error
// 	return &subject, err
// }

// // Update
// func (r *SubjectRepository) Update(subject *models.Subject) error {
// 	return r.db.Save(subject).Error
// }

// // Delete – soft delete (uses DeletedAt)
// func (r *SubjectRepository) Delete(id uuid.UUID) error {
// 	return r.db.Delete(&models.Subject{}, "id = ?", id).Error
// }

// // List – paginated, excluding soft‑deleted
// func (r *SubjectRepository) List(page, limit int) ([]models.Subject, int64, error) {
// 	var subjects []models.Subject
// 	offset := (page - 1) * limit
// 	var total int64
// 	query := r.db.Model(&models.Subject{})
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	err := query.Offset(offset).Limit(limit).Find(&subjects).Error
// 	return subjects, total, err
// }

// // ListActive – only active subjects
// func (r *SubjectRepository) ListActive() ([]models.Subject, error) {
// 	var subjects []models.Subject
// 	err := r.db.Where("is_active = ?", true).Find(&subjects).Error
// 	return subjects, err
// }


// // package repository

// // import (
// // 	"cbt-api/internal/models"
// // 	"github.com/google/uuid"
// // 	"gorm.io/gorm"
// // )

// // type SubjectRepository struct {
// // 	db *gorm.DB
// // }

// // func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
// // 	return &SubjectRepository{db: db}
// // }

// // // Create
// // func (r *SubjectRepository) Create(subject *models.Subject) error {
// // 	return r.db.Create(subject).Error
// // }

// // // FindByID
// // func (r *SubjectRepository) FindByID(id uuid.UUID) (*models.Subject, error) {
// // 	var subject models.Subject
// // 	err := r.db.Where("id = ?", id).First(&subject).Error
// // 	return &subject, err
// // }

// // // Update
// // func (r *SubjectRepository) Update(subject *models.Subject) error {
// // 	return r.db.Save(subject).Error
// // }

// // // Delete
// // func (r *SubjectRepository) Delete(id uuid.UUID) error {
// // 	return r.db.Delete(&models.Subject{}, "id = ?", id).Error
// // }

// // // ListBySchool – with pagination
// // func (r *SubjectRepository) ListBySchool(schoolID uuid.UUID, page, limit int) ([]models.Subject, int64, error) {
// // 	var subjects []models.Subject
// // 	offset := (page - 1) * limit
// // 	query := r.db.Model(&models.Subject{}).Where("school_id = ?", schoolID)
// // 	var total int64
// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, err
// // 	}
// // 	err := query.Offset(offset).Limit(limit).Find(&subjects).Error
// // 	return subjects, total, err
// // }

// // // FindByCode – for uniqueness check
// // func (r *SubjectRepository) FindByCode(code string, schoolID uuid.UUID) (*models.Subject, error) {
// // 	var subject models.Subject
// // 	err := r.db.Where("code = ? AND school_id = ?", code, schoolID).First(&subject).Error
// // 	return &subject, err
// // }