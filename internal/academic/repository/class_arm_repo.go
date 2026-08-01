// internal/academic/repository/class_arm_repo.go
package repository

import (
    "context"
    "fmt"

    "cbt-api/internal/academic/dto"
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

// ============================================
// CORE CRUD OPERATIONS
// ============================================

// Create inserts a new class arm into the database
func (r *ClassArmRepository) Create(arm *models.ClassArm) error {
    return r.db.Create(arm).Error
}

// FindByID retrieves a class arm by ID
func (r *ClassArmRepository) FindByID(id string) (*models.ClassArm, error) {
    var arm models.ClassArm
    uuid, err := uuid.Parse(id)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %w", err)
    }
    err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&arm).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("class arm not found")
        }
        return nil, err
    }
    return &arm, nil
}

// FindBySchool retrieves all class arms for a school
func (r *ClassArmRepository) FindBySchool(schoolID string) ([]models.ClassArm, error) {
    var arms []models.ClassArm
    uuid, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, fmt.Errorf("invalid school ID format: %w", err)
    }
    err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
        Order("sort_order ASC, name ASC").
        Find(&arms).Error
    if err != nil {
        return nil, err
    }
    return arms, nil
}

// Update updates an existing class arm
func (r *ClassArmRepository) Update(arm *models.ClassArm) error {
    return r.db.Save(arm).Error
}

// Delete soft-deletes a class arm by ID
func (r *ClassArmRepository) Delete(id string) error {
    uuid, err := uuid.Parse(id)
    if err != nil {
        return fmt.Errorf("invalid UUID format: %w", err)
    }
    result := r.db.Where("id = ?", uuid).Delete(&models.ClassArm{})
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("class arm not found")
    }
    return nil
}

// ============================================
// LIST & FILTER OPERATIONS
// ============================================

// ListClassArms retrieves class arms with pagination and filters
func (r *ClassArmRepository) ListClassArms(ctx context.Context, filters map[string]interface{}, page, limit int, sortBy, sortOrder string) ([]models.ClassArm, int64, error) {
    var arms []models.ClassArm
    var total int64

    // ✅ Build base query with context
    query := r.db.WithContext(ctx).Model(&models.ClassArm{}).Where("deleted_at IS NULL")

    // ✅ Apply filters (all optional)
    if schoolID, ok := filters["school_id"].(string); ok && schoolID != "" {
        query = query.Where("school_id = ?", schoolID)
    }

    if search, ok := filters["search"].(string); ok && search != "" {
        searchTerm := "%" + search + "%"
        query = query.Where("name ILIKE ? OR arm_code ILIKE ?", searchTerm, searchTerm)
    }

    if minCapacity, ok := filters["min_capacity"].(int); ok && minCapacity > 0 {
        query = query.Where("capacity >= ?", minCapacity)
    }

    if maxCapacity, ok := filters["max_capacity"].(int); ok && maxCapacity > 0 {
        query = query.Where("capacity <= ?", maxCapacity)
    }

    if isActive, ok := filters["is_active"].(*bool); ok && isActive != nil {
        query = query.Where("is_active = ?", *isActive)
    }

    // ✅ Count total records (before pagination)
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count class arms: %w", err)
    }

    // ✅ Apply sorting with security validation
    allowedSortFields := map[string]bool{
        "name": true, "arm_code": true, "capacity": true,
        "sort_order": true, "created_at": true,
    }
    if !allowedSortFields[sortBy] {
        sortBy = "created_at"
    }
    if sortOrder != "asc" && sortOrder != "desc" {
        sortOrder = "desc"
    }
    query = query.Order(sortBy + " " + sortOrder)

    // ✅ Apply pagination with validation
    if page < 1 {
        page = 1
    }
    if limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }
    offset := (page - 1) * limit

    // ✅ Execute query with pagination
    if err := query.Offset(offset).Limit(limit).Find(&arms).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to list class arms: %w", err)
    }

    return arms, total, nil
}

// ============================================
// STATISTICS OPERATIONS
// ============================================

// GetClassArmStats retrieves class arm statistics
func (r *ClassArmRepository) GetClassArmStats(ctx context.Context, schoolID string) (*dto.ClassArmStatsResponse, error) {
    var stats dto.ClassArmStatsResponse

    // ✅ Build base query with context
    query := r.db.WithContext(ctx).Model(&models.ClassArm{}).Where("deleted_at IS NULL")

    // ✅ Apply school filter if provided
    if schoolID != "" {
        query = query.Where("school_id = ?", schoolID)
    }

    // ✅ Use efficient single query with conditional aggregation
    // This is faster than loading all records and calculating in Go
    var result struct {
        Total         int64   `gorm:"column:total"`
        Active        int64   `gorm:"column:active"`
        Inactive      int64   `gorm:"column:inactive"`
        TotalCapacity int64   `gorm:"column:total_capacity"`
        AvgCapacity   float64 `gorm:"column:avg_capacity"`
    }

    err := query.Select(`
        COUNT(*) as total,
        COUNT(CASE WHEN is_active = true THEN 1 END) as active,
        COUNT(CASE WHEN is_active = false THEN 1 END) as inactive,
        COALESCE(SUM(capacity), 0) as total_capacity,
        COALESCE(AVG(capacity), 0) as avg_capacity
    `).Scan(&result).Error

    if err != nil {
        return nil, fmt.Errorf("failed to get class arm statistics: %w", err)
    }

    stats.Total = result.Total
    stats.Active = result.Active
    stats.Inactive = result.Inactive
    stats.TotalCapacity = result.TotalCapacity
    stats.AvgCapacity = result.AvgCapacity

    return &stats, nil
}

// ============================================
// BULK OPERATIONS
// ============================================

// BulkDeleteClassArms deletes multiple class arms by IDs
func (r *ClassArmRepository) BulkDeleteClassArms(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, fmt.Errorf("no IDs provided")
    }

    // ✅ Validate IDs are valid UUIDs
    var validIDs []string
    for _, id := range ids {
        if _, err := uuid.Parse(id); err == nil {
            validIDs = append(validIDs, id)
        }
    }

    if len(validIDs) == 0 {
        return 0, fmt.Errorf("no valid UUIDs provided")
    }

    // ✅ Use transaction for bulk delete
    var deletedCount int64
    err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // ✅ First, get count of existing records
        var count int64
        if err := tx.Model(&models.ClassArm{}).
            Where("id IN ? AND deleted_at IS NULL", validIDs).
            Count(&count).Error; err != nil {
            return fmt.Errorf("failed to count records: %w", err)
        }

        if count == 0 {
            return fmt.Errorf("no matching class arms found")
        }

        // ✅ Perform soft delete
        result := tx.Where("id IN ?", validIDs).Delete(&models.ClassArm{})
        if result.Error != nil {
            return fmt.Errorf("failed to delete class arms: %w", result.Error)
        }

        deletedCount = result.RowsAffected
        return nil
    })

    if err != nil {
        return 0, err
    }

    return deletedCount, nil
}

// ============================================
// SEARCH OPERATIONS
// ============================================

// SearchClassArms searches class arms by name or code
func (r *ClassArmRepository) SearchClassArms(ctx context.Context, query string, limit int) ([]models.ClassArm, error) {
    var arms []models.ClassArm

    // ✅ Validate search query
    if query == "" {
        return nil, fmt.Errorf("search query is required")
    }

    // ✅ Validate and sanitize limit
    if limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }

    // ✅ Build search query with pagination
    searchTerm := "%" + query + "%"
    err := r.db.WithContext(ctx).
        Where("deleted_at IS NULL").
        Where("name ILIKE ? OR arm_code ILIKE ?", searchTerm, searchTerm).
        Limit(limit).
        Order("name ASC").
        Find(&arms).Error

    if err != nil {
        return nil, fmt.Errorf("failed to search class arms: %w", err)
    }

    return arms, nil
}

// ============================================
// UTILITY OPERATIONS
// ============================================

// ExistsByCode checks if an arm with the given code exists
func (r *ClassArmRepository) ExistsByCode(schoolID, code string) (bool, error) {
    var count int64
    err := r.db.Model(&models.ClassArm{}).
        Where("school_id = ? AND arm_code = ? AND deleted_at IS NULL", schoolID, code).
        Count(&count).Error
    if err != nil {
        return false, fmt.Errorf("failed to check arm code existence: %w", err)
    }
    return count > 0, nil
}

// ExistsByName checks if an arm with the given name exists in the school
func (r *ClassArmRepository) ExistsByName(schoolID, name string) (bool, error) {
    var count int64
    err := r.db.Model(&models.ClassArm{}).
        Where("school_id = ? AND name = ? AND deleted_at IS NULL", schoolID, name).
        Count(&count).Error
    if err != nil {
        return false, fmt.Errorf("failed to check arm name existence: %w", err)
    }
    return count > 0, nil
}

// GetBySchoolWithStats retrieves class arms for a school with statistics
func (r *ClassArmRepository) GetBySchoolWithStats(ctx context.Context, schoolID string) ([]models.ClassArm, error) {
    var arms []models.ClassArm
    uuid, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, fmt.Errorf("invalid school ID format: %w", err)
    }

    // ✅ Use eager loading and preload any related data if needed
    err = r.db.WithContext(ctx).
        Where("school_id = ? AND deleted_at IS NULL", uuid).
        Order("sort_order ASC, name ASC").
        Find(&arms).Error
    if err != nil {
        return nil, fmt.Errorf("failed to get class arms: %w", err)
    }

    return arms, nil
}



// package repository

// import (
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
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
//     uuid, err := uuid.Parse(id)
//     if err != nil {
//         return nil, err
//     }
//     err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&arm).Error
//     if err != nil {
//         return nil, err
//     }
//     return &arm, nil
// }

// func (r *ClassArmRepository) FindBySchool(schoolID string) ([]models.ClassArm, error) {
//     var arms []models.ClassArm
//     uuid, err := uuid.Parse(schoolID)
//     if err != nil {
//         return nil, err
//     }
//     err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
//         Order("sort_order ASC, name ASC").Find(&arms).Error
//     return arms, err
// }

// func (r *ClassArmRepository) Update(arm *models.ClassArm) error {
//     return r.db.Save(arm).Error
// }

// func (r *ClassArmRepository) Delete(id string) error {
//     uuid, err := uuid.Parse(id)
//     if err != nil {
//         return err
//     }
//     return r.db.Where("id = ?", uuid).Delete(&models.ClassArm{}).Error
// }


// // internal/academic/repository/class_arm_repository.go
// // Add these new methods to the existing file

// import (
//     "context"
//     "fmt"
//     // ... existing imports
// )

// // ListClassArms retrieves class arms with pagination and filters
// func (r *ClassArmRepository) ListClassArms(ctx context.Context, filters map[string]interface{}, page, limit int, sortBy, sortOrder string) ([]models.ClassArm, int64, error) {
//     var arms []models.ClassArm
//     var total int64

//     query := r.db.WithContext(ctx).Model(&models.ClassArm{}).Where("deleted_at IS NULL")

//     // Apply filters
//     if schoolID, ok := filters["school_id"].(string); ok && schoolID != "" {
//         query = query.Where("school_id = ?", schoolID)
//     }

//     if search, ok := filters["search"].(string); ok && search != "" {
//         query = query.Where("name ILIKE ? OR arm_code ILIKE ?", "%"+search+"%", "%"+search+"%")
//     }

//     if minCapacity, ok := filters["min_capacity"].(int); ok && minCapacity > 0 {
//         query = query.Where("capacity >= ?", minCapacity)
//     }

//     if maxCapacity, ok := filters["max_capacity"].(int); ok && maxCapacity > 0 {
//         query = query.Where("capacity <= ?", maxCapacity)
//     }

//     if isActive, ok := filters["is_active"].(*bool); ok && isActive != nil {
//         query = query.Where("is_active = ?", *isActive)
//     }

//     // Count total
//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, err
//     }

//     // Apply sorting
//     if sortBy == "" {
//         sortBy = "created_at"
//     }
//     if sortOrder == "" {
//         sortOrder = "desc"
//     }
//     query = query.Order(sortBy + " " + sortOrder)

//     // Apply pagination
//     offset := (page - 1) * limit
//     if err := query.Offset(offset).Limit(limit).Find(&arms).Error; err != nil {
//         return nil, 0, err
//     }

//     return arms, total, nil
// }

// // GetClassArmStats retrieves class arm statistics
// func (r *ClassArmRepository) GetClassArmStats(ctx context.Context, schoolID string) (*dto.ClassArmStatsResponse, error) {
//     var stats dto.ClassArmStatsResponse

//     query := r.db.WithContext(ctx).Model(&models.ClassArm{}).Where("deleted_at IS NULL")

//     if schoolID != "" {
//         query = query.Where("school_id = ?", schoolID)
//     }

//     // Get all arms to calculate stats
//     var arms []models.ClassArm
//     if err := query.Find(&arms).Error; err != nil {
//         return nil, err
//     }

//     stats.Total = int64(len(arms))
//     var totalCapacity int64
//     var activeCount int64

//     for _, arm := range arms {
//         totalCapacity += int64(arm.Capacity)
//         if arm.IsActive {
//             activeCount++
//         }
//     }

//     stats.Active = activeCount
//     stats.Inactive = stats.Total - activeCount
//     stats.TotalCapacity = totalCapacity
//     if stats.Total > 0 {
//         stats.AvgCapacity = float64(totalCapacity) / float64(stats.Total)
//     }

//     return &stats, nil
// }

// // BulkDeleteClassArms deletes multiple class arms by IDs
// func (r *ClassArmRepository) BulkDeleteClassArms(ctx context.Context, ids []string) (int64, error) {
//     result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.ClassArm{})
//     return result.RowsAffected, result.Error
// }

// // SearchClassArms searches class arms by name or code
// func (r *ClassArmRepository) SearchClassArms(ctx context.Context, query string, limit int) ([]models.ClassArm, error) {
//     var arms []models.ClassArm
//     err := r.db.WithContext(ctx).
//         Where("deleted_at IS NULL").
//         Where("name ILIKE ? OR arm_code ILIKE ?", "%"+query+"%", "%"+query+"%").
//         Limit(limit).
//         Order("name ASC").
//         Find(&arms).Error
//     return arms, err
// }