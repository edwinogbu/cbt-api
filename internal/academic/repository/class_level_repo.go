package repository

import (
    "cbt-api/internal/models"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ClassLevelRepository struct {
    db *gorm.DB
}

func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
    return &ClassLevelRepository{db: db}
}

// Create inserts a new class level
func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
    if level == nil {
        return errors.New("class level cannot be nil")
    }
    
    if level.SchoolID == "" {
        return errors.New("school_id is required")
    }
    if level.Name == "" {
        return errors.New("name is required")
    }
    if level.LevelNumber <= 0 {
        return errors.New("level_number must be greater than 0")
    }
    if level.Category == "" {
        return errors.New("category is required")
    }
    
    return r.db.Create(level).Error
}

// FindByID retrieves a class level by ID
func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
    if id == "" {
        return nil, errors.New("id is required")
    }
    
    var level models.ClassLevel
    parsedUUID, err := uuid.Parse(id)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %w", err)
    }
    
    err = r.db.Where("id = ? AND deleted_at IS NULL", parsedUUID).First(&level).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, fmt.Errorf("class level with ID %s not found", id)
        }
        return nil, err
    }
    return &level, nil
}

// FindBySchool retrieves all class levels for a school
func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
    if schoolID == "" {
        return nil, errors.New("school_id is required")
    }
    
    var levels []models.ClassLevel
    parsedUUID, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %w", err)
    }
    
    err = r.db.Where("school_id = ? AND deleted_at IS NULL", parsedUUID).
        Order("level_number ASC, sort_order ASC").Find(&levels).Error
    if err != nil {
        return nil, err
    }
    return levels, nil
}

// Update updates an existing class level
func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
    if level == nil {
        return errors.New("class level cannot be nil")
    }
    if level.ID == "" {
        return errors.New("id is required")
    }
    
    level.UpdatedAt = time.Now()
    return r.db.Save(level).Error
}

// Delete soft-deletes a single class level by ID
func (r *ClassLevelRepository) Delete(id string) error {
    if id == "" {
        return errors.New("id is required")
    }
    
    parsedUUID, err := uuid.Parse(id)
    if err != nil {
        return fmt.Errorf("invalid UUID format: %w", err)
    }
    
    result := r.db.Where("id = ?", parsedUUID).Delete(&models.ClassLevel{})
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("class level with ID %s not found", id)
    }
    return nil
}

// BulkDelete deletes multiple class levels by IDs
func (r *ClassLevelRepository) BulkDelete(ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided for bulk delete")
    }

    var uuids []uuid.UUID
    invalidIDs := []string{}
    
    for _, id := range ids {
        if id == "" {
            continue
        }
        parsedUUID, err := uuid.Parse(id)
        if err != nil {
            invalidIDs = append(invalidIDs, id)
            continue
        }
        uuids = append(uuids, parsedUUID)
    }
    
    if len(invalidIDs) > 0 {
        return 0, fmt.Errorf("invalid UUID(s) provided: %v", invalidIDs)
    }
    
    if len(uuids) == 0 {
        return 0, errors.New("no valid IDs provided")
    }
    
    result := r.db.Where("id IN ?", uuids).Delete(&models.ClassLevel{})
    if result.Error != nil {
        return 0, fmt.Errorf("bulk delete failed: %w", result.Error)
    }
    
    return result.RowsAffected, nil
}

// FindByCategory retrieves class levels by category
func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
    if schoolID == "" {
        return nil, errors.New("school_id is required")
    }
    if category == "" {
        return nil, errors.New("category is required")
    }
    
    var levels []models.ClassLevel
    parsedUUID, err := uuid.Parse(schoolID)
    if err != nil {
        return nil, fmt.Errorf("invalid UUID format: %w", err)
    }
    
    err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", parsedUUID, category).
        Order("level_number ASC").Find(&levels).Error
    if err != nil {
        return nil, err
    }
    return levels, nil
}

// ============================================================
// HIGH-PERFORMANCE LIST WITH ALL OPTIONAL PARAMETERS
// ============================================================

func (r *ClassLevelRepository) List(
    schoolID, category, search, sortBy, sortDir string,
    page, limit int,
    isActive *bool,
) ([]models.ClassLevel, int64, error) {
    var levels []models.ClassLevel
    var total int64

    query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")

    if schoolID != "" {
        parsedUUID, err := uuid.Parse(schoolID)
        if err != nil {
            return nil, 0, fmt.Errorf("invalid school_id format: %w", err)
        }
        query = query.Where("school_id = ?", parsedUUID)
    }

    if category != "" {
        validCategories := map[string]bool{"JSS": true, "SSS": true, "PRIMARY": true}
        if !validCategories[category] {
            return nil, 0, fmt.Errorf("invalid category: %s", category)
        }
        query = query.Where("category = ?", category)
    }

    if isActive != nil {
        query = query.Where("is_active = ?", *isActive)
    }

    if search != "" {
        searchTerm := "%" + strings.ToLower(search) + "%"
        query = query.Where("LOWER(name) LIKE ? OR LOWER(category) LIKE ?", searchTerm, searchTerm)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count records: %w", err)
    }

    if total == 0 {
        return []models.ClassLevel{}, 0, nil
    }

    validSortFields := map[string]bool{
        "name": true, "level_number": true, "category": true, 
        "sort_order": true, "created_at": true, "updated_at": true,
    }
    
    if sortBy == "" {
        sortBy = "level_number"
    }
    if !validSortFields[sortBy] {
        return nil, 0, fmt.Errorf("invalid sort_by field: %s", sortBy)
    }
    
    if sortDir == "" {
        sortDir = "asc"
    }
    sortDir = strings.ToLower(sortDir)
    if sortDir != "asc" && sortDir != "desc" {
        return nil, 0, fmt.Errorf("invalid sort_order: %s", sortDir)
    }

    query = query.Order(fmt.Sprintf("%s %s, sort_order ASC", sortBy, sortDir))

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
    if err := query.Offset(offset).Limit(limit).Find(&levels).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to fetch records: %w", err)
    }

    return levels, total, nil
}

// ============================================================
// STATISTICS WITH OPTIONAL FILTERS
// ============================================================

// ClassLevelStats - define stats struct here
type ClassLevelStats struct {
    Total          int64            `json:"total"`
    Active         int64            `json:"active"`
    Inactive       int64            `json:"inactive"`
    ByCategory     map[string]int64 `json:"by_category"`
    TotalLevels    int64            `json:"total_levels"`
    AvgLevelNumber float64          `json:"avg_level_number"`
}

func (r *ClassLevelRepository) GetStats(schoolID string) (*ClassLevelStats, error) {
    stats := &ClassLevelStats{
        ByCategory: make(map[string]int64),
    }

    query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")
    
    if schoolID != "" {
        parsedUUID, err := uuid.Parse(schoolID)
        if err != nil {
            return nil, fmt.Errorf("invalid school_id format: %w", err)
        }
        query = query.Where("school_id = ?", parsedUUID)
    }

    if err := query.Count(&stats.Total).Error; err != nil {
        return nil, fmt.Errorf("failed to get total count: %w", err)
    }

    activeQuery := query
    if err := activeQuery.Where("is_active = ?", true).Count(&stats.Active).Error; err != nil {
        return nil, fmt.Errorf("failed to get active count: %w", err)
    }

    stats.Inactive = stats.Total - stats.Active

    var categoryResults []struct {
        Category string
        Count    int64
    }
    if err := query.Select("category, COUNT(*) as count").
        Group("category").
        Scan(&categoryResults).Error; err != nil {
        return nil, fmt.Errorf("failed to get category counts: %w", err)
    }
    
    for _, result := range categoryResults {
        stats.ByCategory[result.Category] = result.Count
    }

    var totalLevels int64
    if err := query.Select("COUNT(DISTINCT level_number)").Scan(&totalLevels).Error; err != nil {
        return nil, fmt.Errorf("failed to get total levels: %w", err)
    }
    stats.TotalLevels = totalLevels

    var avgLevelNumber float64
    if err := query.Select("COALESCE(AVG(level_number), 0)").Scan(&avgLevelNumber).Error; err != nil {
        return nil, fmt.Errorf("failed to get average level number: %w", err)
    }
    stats.AvgLevelNumber = avgLevelNumber

    return stats, nil
}

// ============================================================
// CHECK EXISTENCE
// ============================================================

func (r *ClassLevelRepository) Exists(id string) (bool, error) {
    if id == "" {
        return false, errors.New("id is required")
    }
    
    parsedUUID, err := uuid.Parse(id)
    if err != nil {
        return false, fmt.Errorf("invalid UUID format: %w", err)
    }
    
    var count int64
    err = r.db.Model(&models.ClassLevel{}).
        Where("id = ? AND deleted_at IS NULL", parsedUUID).
        Count(&count).Error
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

func (r *ClassLevelRepository) CheckNameExists(schoolID, name string, excludeID string) (bool, error) {
    if schoolID == "" || name == "" {
        return false, errors.New("school_id and name are required")
    }
    
    parsedUUID, err := uuid.Parse(schoolID)
    if err != nil {
        return false, fmt.Errorf("invalid UUID format: %w", err)
    }
    
    query := r.db.Model(&models.ClassLevel{}).
        Where("school_id = ? AND LOWER(name) = LOWER(?) AND deleted_at IS NULL", parsedUUID, name)
    
    if excludeID != "" {
        excludeUUID, err := uuid.Parse(excludeID)
        if err != nil {
            return false, fmt.Errorf("invalid exclude UUID format: %w", err)
        }
        query = query.Where("id != ?", excludeUUID)
    }
    
    var count int64
    err = query.Count(&count).Error
    if err != nil {
        return false, err
    }
    return count > 0, nil
}



// package repository

// import (
//     "cbt-api/internal/models"
//     "errors"
//     "fmt"
//     "strings"
//     "time"

//     "github.com/google/uuid"
//     "gorm.io/gorm"
// )

// type ClassLevelRepository struct {
//     db *gorm.DB
// }

// func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
//     return &ClassLevelRepository{db: db}
// }

// // Create inserts a new class level
// func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
//     if level == nil {
//         return errors.New("class level cannot be nil")
//     }
    
//     // Validate required fields
//     if level.SchoolID == "" {
//         return errors.New("school_id is required")
//     }
//     if level.Name == "" {
//         return errors.New("name is required")
//     }
//     if level.LevelNumber <= 0 {
//         return errors.New("level_number must be greater than 0")
//     }
//     if level.Category == "" {
//         return errors.New("category is required")
//     }
    
//     return r.db.Create(level).Error
// }

// // FindByID retrieves a class level by ID
// func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
//     if id == "" {
//         return nil, errors.New("id is required")
//     }
    
//     var level models.ClassLevel
//     uuid, err := uuid.Parse(id)
//     if err != nil {
//         return nil, fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&level).Error
//     if err != nil {
//         if errors.Is(err, gorm.ErrRecordNotFound) {
//             return nil, fmt.Errorf("class level with ID %s not found", id)
//         }
//         return nil, err
//     }
//     return &level, nil
// }

// // FindBySchool retrieves all class levels for a school
// func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
//     if schoolID == "" {
//         return nil, errors.New("school_id is required")
//     }
    
//     var levels []models.ClassLevel
//     uuid, err := uuid.Parse(schoolID)
//     if err != nil {
//         return nil, fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
//         Order("level_number ASC, sort_order ASC").Find(&levels).Error
//     if err != nil {
//         return nil, err
//     }
//     return levels, nil
// }

// // Update updates an existing class level
// func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
//     if level == nil {
//         return errors.New("class level cannot be nil")
//     }
//     if level.ID == "" {
//         return errors.New("id is required")
//     }
    
//     level.UpdatedAt = time.Now()
//     return r.db.Save(level).Error
// }

// // Delete soft-deletes a class level
// // func (r *ClassLevelRepository) Delete(id string) error {
// //     if id == "" {
// //         return errors.New("id is required")
// //     }
    
// //     uuid, err := uuid.Parse(id)
// //     if err != nil {
// //         return fmt.Errorf("invalid UUID format: %w", err)
// //     }
    
// //     result := r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{})
// //     if result.Error != nil {
// //         return result.Error
// //     }
// //     if result.RowsAffected == 0 {
// //         return fmt.Errorf("class level with ID %s not found", id)
// //     }
// //     return nil
// // }

// // Delete soft-deletes a single class level by ID
// func (r *ClassLevelRepository) Delete(id string) error {
//     if id == "" {
//         return errors.New("id is required")
//     }
    
//     uuid, err := uuid.Parse(id)
//     if err != nil {
//         return fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     result := r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{})
//     if result.Error != nil {
//         return result.Error
//     }
//     if result.RowsAffected == 0 {
//         return fmt.Errorf("class level with ID %s not found", id)
//     }
//     return nil
// }

// // FindByCategory retrieves class levels by category
// func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
//     if schoolID == "" {
//         return nil, errors.New("school_id is required")
//     }
//     if category == "" {
//         return nil, errors.New("category is required")
//     }
    
//     var levels []models.ClassLevel
//     uuid, err := uuid.Parse(schoolID)
//     if err != nil {
//         return nil, fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", uuid, category).
//         Order("level_number ASC").Find(&levels).Error
//     if err != nil {
//         return nil, err
//     }
//     return levels, nil
// }

// // ============================================================
// // HIGH-PERFORMANCE LIST WITH ALL OPTIONAL PARAMETERS
// // ============================================================

// // List performs a high-performance paginated query with all optional parameters
// // Uses indexed queries and efficient filtering
// func (r *ClassLevelRepository) List(
//     schoolID, category, search, sortBy, sortDir string,
//     page, limit int,
//     isActive *bool,
// ) ([]models.ClassLevel, int64, error) {
//     var levels []models.ClassLevel
//     var total int64

//     // Start with base query (only non-deleted records)
//     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")

//     // ============================================================
//     // OPTIONAL FILTERS (ALL OPTIONAL)
//     // ============================================================

//     // Filter by school_id (if provided)
//     if schoolID != "" {
//         uuid, err := uuid.Parse(schoolID)
//         if err != nil {
//             return nil, 0, fmt.Errorf("invalid school_id format: %w", err)
//         }
//         query = query.Where("school_id = ?", uuid)
//     }

//     // Filter by category (if provided)
//     if category != "" {
//         // Validate category
//         validCategories := map[string]bool{"JSS": true, "SSS": true, "PRIMARY": true}
//         if !validCategories[category] {
//             return nil, 0, fmt.Errorf("invalid category: %s (must be JSS, SSS, or PRIMARY)", category)
//         }
//         query = query.Where("category = ?", category)
//     }

//     // Filter by is_active (if provided)
//     if isActive != nil {
//         query = query.Where("is_active = ?", *isActive)
//     }

//     // Search by name or category (if provided)
//     if search != "" {
//         searchTerm := "%" + strings.ToLower(search) + "%"
//         query = query.Where("LOWER(name) LIKE ? OR LOWER(category) LIKE ?", searchTerm, searchTerm)
//     }

//     // ============================================================
//     // GET TOTAL COUNT (BEFORE PAGINATION)
//     // ============================================================
//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to count records: %w", err)
//     }

//     // If no records, return early
//     if total == 0 {
//         return []models.ClassLevel{}, 0, nil
//     }

//     // ============================================================
//     // SORTING (WITH VALIDATION AND DEFAULTS)
//     // ============================================================
//     validSortFields := map[string]bool{
//         "name": true, "level_number": true, "category": true, 
//         "sort_order": true, "created_at": true, "updated_at": true,
//     }
    
//     if sortBy == "" {
//         sortBy = "level_number" // Default sort
//     }
//     if !validSortFields[sortBy] {
//         return nil, 0, fmt.Errorf("invalid sort_by field: %s", sortBy)
//     }
    
//     if sortDir == "" {
//         sortDir = "asc"
//     }
//     sortDir = strings.ToLower(sortDir)
//     if sortDir != "asc" && sortDir != "desc" {
//         return nil, 0, fmt.Errorf("invalid sort_order: %s (must be asc or desc)", sortDir)
//     }

//     // Apply sorting with secondary sort for consistency
//     query = query.Order(fmt.Sprintf("%s %s, sort_order ASC", sortBy, sortDir))

//     // ============================================================
//     // PAGINATION (WITH VALIDATION AND DEFAULTS)
//     // ============================================================
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 {
//         limit = 20
//     }
//     if limit > 100 {
//         limit = 100 // Max limit to prevent performance issues
//     }

//     offset := (page - 1) * limit

//     // ============================================================
//     // EXECUTE QUERY WITH PAGINATION
//     // ============================================================
//     if err := query.Offset(offset).Limit(limit).Find(&levels).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to fetch records: %w", err)
//     }

//     return levels, total, nil
// }

// // ============================================================
// // BULK DELETE WITH VALIDATION
// // ============================================================

// // BulkDelete deletes multiple class levels by IDs
// func (r *ClassLevelRepository) BulkDelete(ids []string) (int64, error) {
//     if len(ids) == 0 {
//         return 0, errors.New("no IDs provided for bulk delete")
//     }

//     var uuids []uuid.UUID
//     invalidIDs := []string{}
    
//     for _, id := range ids {
//         if id == "" {
//             continue
//         }
//         uuid, err := uuid.Parse(id)
//         if err != nil {
//             invalidIDs = append(invalidIDs, id)
//             continue
//         }
//         uuids = append(uuids, uuid)
//     }
    
//     if len(invalidIDs) > 0 {
//         return 0, fmt.Errorf("invalid UUID(s) provided: %v", invalidIDs)
//     }
    
//     if len(uuids) == 0 {
//         return 0, errors.New("no valid IDs provided")
//     }
    
//     result := r.db.Where("id IN ?", uuids).Delete(&models.ClassLevel{})
//     if result.Error != nil {
//         return 0, fmt.Errorf("bulk delete failed: %w", result.Error)
//     }
    
//     return result.RowsAffected, nil
// }

// // ============================================================
// // STATISTICS WITH OPTIONAL FILTERS
// // ============================================================

// // GetStats retrieves statistics for class levels with optional school filter
// func (r *ClassLevelRepository) GetStats(schoolID string) (*models.ClassLevelStats, error) {
//     stats := &models.ClassLevelStats{
//         ByCategory: make(map[string]int64),
//     }

//     // Base query
//     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")
    
//     // Apply school filter if provided
//     if schoolID != "" {
//         uuid, err := uuid.Parse(schoolID)
//         if err != nil {
//             return nil, fmt.Errorf("invalid school_id format: %w", err)
//         }
//         query = query.Where("school_id = ?", uuid)
//     }

//     // ============================================================
//     // TOTAL COUNT
//     // ============================================================
//     if err := query.Count(&stats.Total).Error; err != nil {
//         return nil, fmt.Errorf("failed to get total count: %w", err)
//     }

//     // ============================================================
//     // ACTIVE COUNT
//     // ============================================================
//     activeQuery := query
//     if err := activeQuery.Where("is_active = ?", true).Count(&stats.Active).Error; err != nil {
//         return nil, fmt.Errorf("failed to get active count: %w", err)
//     }

//     // ============================================================
//     // INACTIVE COUNT
//     // ============================================================
//     stats.Inactive = stats.Total - stats.Active

//     // ============================================================
//     // BY CATEGORY COUNT
//     // ============================================================
//     var categoryResults []struct {
//         Category string
//         Count    int64
//     }
//     if err := query.Select("category, COUNT(*) as count").
//         Group("category").
//         Scan(&categoryResults).Error; err != nil {
//         return nil, fmt.Errorf("failed to get category counts: %w", err)
//     }
    
//     for _, result := range categoryResults {
//         stats.ByCategory[result.Category] = result.Count
//     }

//     // ============================================================
//     // TOTAL LEVELS (Distinct level numbers)
//     // ============================================================
//     var totalLevels int64
//     if err := query.Select("COUNT(DISTINCT level_number)").Scan(&totalLevels).Error; err != nil {
//         return nil, fmt.Errorf("failed to get total levels: %w", err)
//     }
//     stats.TotalLevels = totalLevels

//     // ============================================================
//     // AVERAGE LEVEL NUMBER
//     // ============================================================
//     var avgLevelNumber float64
//     if err := query.Select("COALESCE(AVG(level_number), 0)").Scan(&avgLevelNumber).Error; err != nil {
//         return nil, fmt.Errorf("failed to get average level number: %w", err)
//     }
//     stats.AvgLevelNumber = avgLevelNumber

//     return stats, nil
// }

// // ============================================================
// // CHECK EXISTENCE
// // ============================================================

// // Exists checks if a class level exists by ID
// func (r *ClassLevelRepository) Exists(id string) (bool, error) {
//     if id == "" {
//         return false, errors.New("id is required")
//     }
    
//     uuid, err := uuid.Parse(id)
//     if err != nil {
//         return false, fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     var count int64
//     err = r.db.Model(&models.ClassLevel{}).
//         Where("id = ? AND deleted_at IS NULL", uuid).
//         Count(&count).Error
//     if err != nil {
//         return false, err
//     }
//     return count > 0, nil
// }

// // CheckNameExists checks if a class level with the same name exists
// func (r *ClassLevelRepository) CheckNameExists(schoolID, name string, excludeID string) (bool, error) {
//     if schoolID == "" || name == "" {
//         return false, errors.New("school_id and name are required")
//     }
    
//     uuid, err := uuid.Parse(schoolID)
//     if err != nil {
//         return false, fmt.Errorf("invalid UUID format: %w", err)
//     }
    
//     query := r.db.Model(&models.ClassLevel{}).
//         Where("school_id = ? AND LOWER(name) = LOWER(?) AND deleted_at IS NULL", uuid, name)
    
//     if excludeID != "" {
//         excludeUUID, err := uuid.Parse(excludeID)
//         if err != nil {
//             return false, fmt.Errorf("invalid exclude UUID format: %w", err)
//         }
//         query = query.Where("id != ?", excludeUUID)
//     }
    
//     var count int64
//     err = query.Count(&count).Error
//     if err != nil {
//         return false, err
//     }
//     return count > 0, nil
// }


// // package repository

// // import (
// //     "cbt-api/internal/models"
// //     "errors"
// //     "fmt"
// //     "strings"
// //     "time"

// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// // )

// // type ClassLevelRepository struct {
// //     db *gorm.DB
// // }

// // func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
// //     return &ClassLevelRepository{db: db}
// // }

// // func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
// //     if level == nil {
// //         return errors.New("class level cannot be nil")
// //     }
// //     return r.db.Create(level).Error
// // }

// // func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
// //     if id == "" {
// //         return nil, errors.New("ID cannot be empty")
// //     }

// //     uuid, err := uuid.Parse(id)
// //     if err != nil {
// //         return nil, fmt.Errorf("invalid UUID format: %w", err)
// //     }

// //     var level models.ClassLevel
// //     err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&level).Error
// //     if err != nil {
// //         if errors.Is(err, gorm.ErrRecordNotFound) {
// //             return nil, errors.New("class level not found")
// //         }
// //         return nil, fmt.Errorf("failed to find class level: %w", err)
// //     }
// //     return &level, nil
// // }

// // func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
// //     if schoolID == "" {
// //         return nil, errors.New("school ID cannot be empty")
// //     }

// //     uuid, err := uuid.Parse(schoolID)
// //     if err != nil {
// //         return nil, fmt.Errorf("invalid UUID format: %w", err)
// //     }

// //     var levels []models.ClassLevel
// //     err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
// //         Order("level_number ASC, sort_order ASC").Find(&levels).Error
// //     if err != nil {
// //         return nil, fmt.Errorf("failed to find class levels: %w", err)
// //     }
// //     return levels, nil
// // }

// // func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
// //     if level == nil {
// //         return errors.New("class level cannot be nil")
// //     }
// //     if level.ID == "" {
// //         return errors.New("ID cannot be empty")
// //     }
// //     return r.db.Save(level).Error
// // }

// // func (r *ClassLevelRepository) Delete(id string) error {
// //     if id == "" {
// //         return errors.New("ID cannot be empty")
// //     }

// //     uuid, err := uuid.Parse(id)
// //     if err != nil {
// //         return fmt.Errorf("invalid UUID format: %w", err)
// //     }

// //     result := r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{})
// //     if result.Error != nil {
// //         return fmt.Errorf("failed to delete class level: %w", result.Error)
// //     }
// //     if result.RowsAffected == 0 {
// //         return errors.New("class level not found")
// //     }
// //     return nil
// // }

// // func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
// //     if schoolID == "" {
// //         return nil, errors.New("school ID cannot be empty")
// //     }
// //     if category == "" {
// //         return nil, errors.New("category cannot be empty")
// //     }

// //     uuid, err := uuid.Parse(schoolID)
// //     if err != nil {
// //         return nil, fmt.Errorf("invalid UUID format: %w", err)
// //     }

// //     var levels []models.ClassLevel
// //     err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", uuid, category).
// //         Order("level_number ASC").Find(&levels).Error
// //     if err != nil {
// //         return nil, fmt.Errorf("failed to find class levels by category: %w", err)
// //     }
// //     return levels, nil
// // }

// // // ============================================================
// // // HIGH-PERFORMANCE LIST WITH ALL OPTIONAL PARAMETERS
// // // ============================================================

// // // List performs a high-performance paginated query with all optional parameters.
// // // All parameters are optional - if not provided, returns all class levels.
// // func (r *ClassLevelRepository) List(
// //     schoolID string,     // OPTIONAL - filter by school
// //     category string,     // OPTIONAL - filter by category (JSS, SSS, PRIMARY)
// //     search string,       // OPTIONAL - search by name
// //     sortBy string,       // OPTIONAL - sort field (default: level_number)
// //     sortDir string,      // OPTIONAL - sort direction (default: asc)
// //     page int,            // OPTIONAL - page number (default: 1)
// //     limit int,           // OPTIONAL - items per page (default: 20, max: 100)
// //     isActive *bool,      // OPTIONAL - filter by active status
// // ) ([]models.ClassLevel, int64, error) {
// //     var levels []models.ClassLevel
// //     var total int64

// //     // Build query with soft delete filter
// //     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")

// //     // Apply filters (ALL OPTIONAL)
// //     if schoolID != "" {
// //         uuid, err := uuid.Parse(schoolID)
// //         if err == nil {
// //             query = query.Where("school_id = ?", uuid)
// //         }
// //     }

// //     if category != "" {
// //         query = query.Where("category = ?", category)
// //     }

// //     if search != "" {
// //         search = "%" + strings.ToLower(search) + "%"
// //         query = query.Where("LOWER(name) LIKE ? OR LOWER(category) LIKE ?", search, search)
// //     }

// //     if isActive != nil {
// //         query = query.Where("is_active = ?", *isActive)
// //     }

// //     // Get total count (for pagination)
// //     if err := query.Count(&total).Error; err != nil {
// //         return nil, 0, fmt.Errorf("failed to count class levels: %w", err)
// //     }

// //     // Apply sorting (with defaults if not provided)
// //     if sortBy == "" {
// //         sortBy = "level_number"
// //     }
// //     if sortDir == "" {
// //         sortDir = "asc"
// //     }
// //     // Validate sort direction
// //     if sortDir != "asc" && sortDir != "desc" {
// //         sortDir = "asc"
// //     }
// //     query = query.Order(sortBy + " " + sortDir + ", sort_order ASC")

// //     // Apply pagination (with defaults if not provided)
// //     if page < 1 {
// //         page = 1
// //     }
// //     if limit < 1 {
// //         limit = 20
// //     }
// //     if limit > 100 {
// //         limit = 100
// //     }

// //     offset := (page - 1) * limit
// //     if err := query.Offset(offset).Limit(limit).Find(&levels).Error; err != nil {
// //         return nil, 0, fmt.Errorf("failed to fetch class levels: %w", err)
// //     }

// //     return levels, total, nil
// // }

// // // ============================================================
// // // BULK OPERATIONS
// // // ============================================================

// // // BulkDelete performs high-performance bulk delete
// // func (r *ClassLevelRepository) BulkDelete(ids []string) (int64, error) {
// //     if len(ids) == 0 {
// //         return 0, errors.New("no IDs provided")
// //     }

// //     var uuids []uuid.UUID
// //     for _, id := range ids {
// //         if id == "" {
// //             continue
// //         }
// //         uuid, err := uuid.Parse(id)
// //         if err != nil {
// //             continue
// //         }
// //         uuids = append(uuids, uuid)
// //     }

// //     if len(uuids) == 0 {
// //         return 0, errors.New("no valid UUIDs provided")
// //     }

// //     // Use transaction for bulk delete
// //     var deletedCount int64
// //     err := r.db.Transaction(func(tx *gorm.DB) error {
// //         result := tx.Where("id IN ?", uuids).Delete(&models.ClassLevel{})
// //         if result.Error != nil {
// //             return fmt.Errorf("failed to bulk delete class levels: %w", result.Error)
// //         }
// //         deletedCount = result.RowsAffected
// //         return nil
// //     })

// //     if err != nil {
// //         return 0, err
// //     }

// //     if deletedCount == 0 {
// //         return 0, errors.New("no class levels found to delete")
// //     }

// //     return deletedCount, nil
// // }

// // // ============================================================
// // // STATISTICS
// // // ============================================================

// // // GetStats returns high-performance statistics with optional school filter
// // func (r *ClassLevelRepository) GetStats(schoolID string) (*models.ClassLevelStats, error) {
// //     stats := &models.ClassLevelStats{
// //         ByCategory: make(map[string]int64),
// //     }

// //     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")

// //     if schoolID != "" {
// //         uuid, err := uuid.Parse(schoolID)
// //         if err == nil {
// //             query = query.Where("school_id = ?", uuid)
// //         }
// //     }

// //     // Use single query for total count
// //     if err := query.Count(&stats.Total).Error; err != nil {
// //         return nil, fmt.Errorf("failed to get total count: %w", err)
// //     }

// //     // Get active count
// //     activeQuery := query
// //     if err := activeQuery.Where("is_active = ?", true).Count(&stats.Active).Error; err != nil {
// //         return nil, fmt.Errorf("failed to get active count: %w", err)
// //     }

// //     // Inactive is total - active
// //     stats.Inactive = stats.Total - stats.Active

// //     // Get category breakdown using GROUP BY
// //     var categoryResults []struct {
// //         Category string
// //         Count    int64
// //     }
// //     if err := query.Select("category, COUNT(*) as count").Group("category").Scan(&categoryResults).Error; err != nil {
// //         return nil, fmt.Errorf("failed to get category breakdown: %w", err)
// //     }
// //     for _, result := range categoryResults {
// //         stats.ByCategory[result.Category] = result.Count
// //     }

// //     // Get total levels (distinct level numbers)
// //     var totalLevels int64
// //     if err := query.Select("COUNT(DISTINCT level_number)").Scan(&totalLevels).Error; err != nil {
// //         return nil, fmt.Errorf("failed to get total levels: %w", err)
// //     }
// //     stats.TotalLevels = totalLevels

// //     // Get average level number
// //     var avgLevelNumber float64
// //     if err := query.Select("COALESCE(AVG(level_number), 0)").Scan(&avgLevelNumber).Error; err != nil {
// //         return nil, fmt.Errorf("failed to get average level number: %w", err)
// //     }
// //     stats.AvgLevelNumber = avgLevelNumber

// //     return stats, nil
// // }

// // // ============================================================
// // // SEARCH
// // // ============================================================

// // // Search performs a quick search with optional filters
// // func (r *ClassLevelRepository) Search(
// //     schoolID string,
// //     query string,
// //     page int,
// //     limit int,
// // ) ([]models.ClassLevel, int64, error) {
// //     if query == "" {
// //         return nil, 0, errors.New("search query cannot be empty")
// //     }

// //     return r.List(schoolID, "", query, "name", "asc", page, limit, nil)
// // }



// // package repository

// // import (
// //     "cbt-api/internal/models"
// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// // )

// // type ClassLevelRepository struct {
// //     db *gorm.DB
// // }

// // func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
// //     return &ClassLevelRepository{db: db}
// // }

// // func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
// //     return r.db.Create(level).Error
// // }

// // func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
// //     var level models.ClassLevel
// //     uuid, err := uuid.Parse(id)
// //     if err != nil {
// //         return nil, err
// //     }
// //     err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&level).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &level, nil
// // }

// // func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
// //     var levels []models.ClassLevel
// //     uuid, err := uuid.Parse(schoolID)
// //     if err != nil {
// //         return nil, err
// //     }
// //     err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
// //         Order("level_number ASC, sort_order ASC").Find(&levels).Error
// //     return levels, err
// // }

// // func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
// //     return r.db.Save(level).Error
// // }

// // func (r *ClassLevelRepository) Delete(id string) error {
// //     uuid, err := uuid.Parse(id)
// //     if err != nil {
// //         return err
// //     }
// //     return r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{}).Error
// // }

// // func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
// //     var levels []models.ClassLevel
// //     uuid, err := uuid.Parse(schoolID)
// //     if err != nil {
// //         return nil, err
// //     }
// //     err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", uuid, category).
// //         Order("level_number ASC").Find(&levels).Error
// //     return levels, err
// // }

// // // NEW: List with pagination, filtering, sorting
// // func (r *ClassLevelRepository) List(schoolID, category, search, sortBy, sortDir string, page, limit int, isActive *bool) ([]models.ClassLevel, int64, error) {
// //     var levels []models.ClassLevel
// //     var total int64
    
// //     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")
    
// //     // Apply filters
// //     if schoolID != "" {
// //         uuid, err := uuid.Parse(schoolID)
// //         if err != nil {
// //             return nil, 0, err
// //         }
// //         query = query.Where("school_id = ?", uuid)
// //     }
    
// //     if category != "" {
// //         query = query.Where("category = ?", category)
// //     }
    
// //     if isActive != nil {
// //         query = query.Where("is_active = ?", *isActive)
// //     }
    
// //     if search != "" {
// //         query = query.Where("name ILIKE ? OR level_number::text ILIKE ?", "%"+search+"%", "%"+search+"%")
// //     }
    
// //     // Get total count
// //     if err := query.Count(&total).Error; err != nil {
// //         return nil, 0, err
// //     }
    
// //     // Apply sorting
// //     if sortBy == "" {
// //         sortBy = "level_number"
// //     }
// //     if sortDir == "" {
// //         sortDir = "asc"
// //     }
// //     query = query.Order(sortBy + " " + sortDir + ", sort_order asc")
    
// //     // Apply pagination
// //     offset := (page - 1) * limit
// //     if err := query.Limit(limit).Offset(offset).Find(&levels).Error; err != nil {
// //         return nil, 0, err
// //     }
    
// //     return levels, total, nil
// // }

// // // NEW: Bulk delete
// // func (r *ClassLevelRepository) BulkDelete(ids []string) (int64, error) {
// //     var uuids []uuid.UUID
// //     for _, id := range ids {
// //         uuid, err := uuid.Parse(id)
// //         if err != nil {
// //             return 0, err
// //         }
// //         uuids = append(uuids, uuid)
// //     }
    
// //     result := r.db.Where("id IN ?", uuids).Delete(&models.ClassLevel{})
// //     return result.RowsAffected, result.Error
// // }

// // // NEW: Get stats
// // func (r *ClassLevelRepository) GetStats(schoolID string) (*ClassLevelStats, error) {
// //     var stats ClassLevelStats
    
// //     query := r.db.Model(&models.ClassLevel{}).Where("deleted_at IS NULL")
// //     if schoolID != "" {
// //         uuid, err := uuid.Parse(schoolID)
// //         if err != nil {
// //             return nil, err
// //         }
// //         query = query.Where("school_id = ?", uuid)
// //     }
    
// //     // Total
// //     if err := query.Count(&stats.Total).Error; err != nil {
// //         return nil, err
// //     }
    
// //     // Active/Inactive
// //     if err := query.Where("is_active = ?", true).Count(&stats.Active).Error; err != nil {
// //         return nil, err
// //     }
// //     stats.Inactive = stats.Total - stats.Active
    
// //     // By Category
// //     var categoryResults []struct {
// //         Category string
// //         Count    int64
// //     }
// //     if err := query.Select("category, count(*) as count").Group("category").Scan(&categoryResults).Error; err != nil {
// //         return nil, err
// //     }
    
// //     stats.ByCategory = make(map[string]int64)
// //     for _, result := range categoryResults {
// //         stats.ByCategory[result.Category] = result.Count
// //     }
    
// //     // Avg Level Number
// //     var avgResult struct {
// //         Avg float64
// //     }
// //     if err := query.Select("COALESCE(AVG(level_number), 0) as avg").Scan(&avgResult).Error; err != nil {
// //         return nil, err
// //     }
// //     stats.AvgLevelNumber = avgResult.Avg
// //     stats.TotalLevels = stats.Total
    
// //     return &stats, nil
// // }

// // type ClassLevelStats struct {
// //     Total          int64
// //     Active         int64
// //     Inactive       int64
// //     ByCategory     map[string]int64
// //     TotalLevels    int64
// //     AvgLevelNumber float64
// // }


// // // package repository

// // // import (
// // //     "cbt-api/internal/models"
// // //     "github.com/google/uuid"
// // //     "gorm.io/gorm"
// // // )

// // // type ClassLevelRepository struct {
// // //     db *gorm.DB
// // // }

// // // func NewClassLevelRepository(db *gorm.DB) *ClassLevelRepository {
// // //     return &ClassLevelRepository{db: db}
// // // }

// // // func (r *ClassLevelRepository) Create(level *models.ClassLevel) error {
// // //     return r.db.Create(level).Error
// // // }

// // // func (r *ClassLevelRepository) FindByID(id string) (*models.ClassLevel, error) {
// // //     var level models.ClassLevel
// // //     uuid, err := uuid.Parse(id)
// // //     if err != nil {
// // //         return nil, err
// // //     }
// // //     err = r.db.Where("id = ? AND deleted_at IS NULL", uuid).First(&level).Error
// // //     if err != nil {
// // //         return nil, err
// // //     }
// // //     return &level, nil
// // // }

// // // func (r *ClassLevelRepository) FindBySchool(schoolID string) ([]models.ClassLevel, error) {
// // //     var levels []models.ClassLevel
// // //     uuid, err := uuid.Parse(schoolID)
// // //     if err != nil {
// // //         return nil, err
// // //     }
// // //     err = r.db.Where("school_id = ? AND deleted_at IS NULL", uuid).
// // //         Order("level_number ASC, sort_order ASC").Find(&levels).Error
// // //     return levels, err
// // // }

// // // func (r *ClassLevelRepository) Update(level *models.ClassLevel) error {
// // //     return r.db.Save(level).Error
// // // }

// // // func (r *ClassLevelRepository) Delete(id string) error {
// // //     uuid, err := uuid.Parse(id)
// // //     if err != nil {
// // //         return err
// // //     }
// // //     return r.db.Where("id = ?", uuid).Delete(&models.ClassLevel{}).Error
// // // }


// // // func (r *ClassLevelRepository) FindByCategory(schoolID, category string) ([]models.ClassLevel, error) {
// // //     var levels []models.ClassLevel
// // //     uuid, err := uuid.Parse(schoolID)
// // //     if err != nil {
// // //         return nil, err
// // //     }
// // //     err = r.db.Where("school_id = ? AND category = ? AND deleted_at IS NULL", uuid, category).
// // //         Order("level_number ASC").Find(&levels).Error
// // //     return levels, err
// // // }
