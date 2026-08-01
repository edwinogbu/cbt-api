package repository

import (
     "context"  // ✅ ADD THIS
    "errors"   // ✅ ADD THIS
    "fmt"      // ✅ ADD THIS
    
    "cbt-api/internal/academic/dto"  // ✅ ADD THIS
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

// ============================================================
// NEW REPOSITORY METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListWithFilters - ALL PARAMETERS OPTIONAL
func (r *ClassRepository) ListWithFilters(ctx context.Context, req *dto.ListClassesRequest) ([]models.Class, int64, error) {
    if req.Page < 1 {
        req.Page = 1
    }
    if req.Limit < 1 || req.Limit > 100 {
        req.Limit = 20
    }

    var classes []models.Class
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Class{}).
        Where("classes.deleted_at IS NULL").
        Preload("ClassLevel").
        Preload("ClassArm")

    // ALL FILTERS ARE OPTIONAL
    if req.SchoolID != "" {
        query = query.Where("classes.school_id = ?", req.SchoolID)
    }
    if req.SessionID != "" {
        query = query.Where("classes.session_id = ?", req.SessionID)
    }
    if req.ClassLevelID != "" {
        query = query.Where("classes.class_level_id = ?", req.ClassLevelID)
    }
    if req.ClassArmID != "" {
        query = query.Where("classes.class_arm_id = ?", req.ClassArmID)
    }
    if req.TeacherID != "" {
        query = query.Where("classes.teacher_id = ?", req.TeacherID)
    }
    if req.IsActive != nil {
        query = query.Where("classes.is_active = ?", *req.IsActive)
    }

    // Search by class code or room number
    if req.Search != "" {
        searchTerm := "%" + req.Search + "%"
        query = query.Where("classes.class_code ILIKE ? OR classes.room_number ILIKE ?", searchTerm, searchTerm)
    }

    // Count total
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count classes: %w", err)
    }

    // Sorting (with defaults)
    sortBy := req.SortBy
    if sortBy == "" {
        sortBy = "classes.created_at"
    }
    sortOrder := req.SortOrder
    if sortOrder == "" {
        sortOrder = "DESC"
    }
    query = query.Order(sortBy + " " + sortOrder)

    // Pagination
    offset := (req.Page - 1) * req.Limit
    if err := query.Offset(offset).Limit(req.Limit).Find(&classes).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to list classes: %w", err)
    }

    return classes, total, nil
}

// FindByArm - get classes by class arm ID
func (r *ClassRepository) FindByArm(ctx context.Context, classArmID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.WithContext(ctx).
        Where("class_arm_id = ? AND deleted_at IS NULL", classArmID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Find(&classes).Error
    return classes, err
}

// FindByLevel - get classes by class level ID
func (r *ClassRepository) FindByLevel(ctx context.Context, classLevelID string) ([]models.Class, error) {
    var classes []models.Class
    err := r.db.WithContext(ctx).
        Where("class_level_id = ? AND deleted_at IS NULL", classLevelID).
        Preload("ClassLevel").
        Preload("ClassArm").
        Find(&classes).Error
    return classes, err
}

// BulkDeleteClasses - bulk delete
func (r *ClassRepository) BulkDeleteClasses(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }
    result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Class{})
    if result.Error != nil {
        return 0, fmt.Errorf("bulk delete failed: %w", result.Error)
    }
    return result.RowsAffected, nil
}

// GetClassStats - statistics
func (r *ClassRepository) GetClassStats(ctx context.Context, schoolID string) (*dto.ClassStatsResponse, error) {
    stats := &dto.ClassStatsResponse{
        BySchool: make(map[string]int64),
        ByLevel:  make(map[string]int64),
        ByArm:    make(map[string]int64),
    }

    query := r.db.WithContext(ctx).Model(&models.Class{}).Where("classes.deleted_at IS NULL")

    if schoolID != "" {
        query = query.Where("classes.school_id = ?", schoolID)
    }

    // Total
    if err := query.Count(&stats.Total).Error; err != nil {
        return nil, fmt.Errorf("failed to get total: %w", err)
    }

    // Active
    activeQuery := query
    if err := activeQuery.Where("classes.is_active = ?", true).Count(&stats.Active).Error; err != nil {
        return nil, fmt.Errorf("failed to get active: %w", err)
    }

    // Inactive
    stats.Inactive = stats.Total - stats.Active

    // By School
    var schoolResults []struct {
        SchoolID string
        Count    int64
    }
    if err := query.Select("school_id, COUNT(*) as count").
        Group("school_id").
        Scan(&schoolResults).Error; err != nil {
        return nil, fmt.Errorf("failed to get by school: %w", err)
    }
    for _, result := range schoolResults {
        stats.BySchool[result.SchoolID] = result.Count
    }

    // By Level
    var levelResults []struct {
        ClassLevelID string
        Count        int64
    }
    if err := query.Select("class_level_id, COUNT(*) as count").
        Group("class_level_id").
        Scan(&levelResults).Error; err != nil {
        return nil, fmt.Errorf("failed to get by level: %w", err)
    }
    for _, result := range levelResults {
        stats.ByLevel[result.ClassLevelID] = result.Count
    }

    // By Arm
    var armResults []struct {
        ClassArmID string
        Count      int64
    }
    if err := query.Select("class_arm_id, COUNT(*) as count").
        Group("class_arm_id").
        Scan(&armResults).Error; err != nil {
        return nil, fmt.Errorf("failed to get by arm: %w", err)
    }
    for _, result := range armResults {
        stats.ByArm[result.ClassArmID] = result.Count
    }

    // Total Students
    var totalStudents int64
    subQuery := r.db.WithContext(ctx).Model(&models.Student{}).
        Where("deleted_at IS NULL").
        Select("COUNT(*)")
    if schoolID != "" {
        subQuery = subQuery.Where("school_id = ?", schoolID)
    }
    if err := subQuery.Scan(&totalStudents).Error; err != nil {
        return nil, fmt.Errorf("failed to get total students: %w", err)
    }
    stats.TotalStudents = totalStudents

    // Average per class
    if stats.Total > 0 {
        stats.AvgPerClass = float64(stats.TotalStudents) / float64(stats.Total)
    }

    return stats, nil
}

// SearchClasses - search
func (r *ClassRepository) SearchClasses(ctx context.Context, query string, schoolID string, page, limit int) ([]models.Class, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var classes []models.Class
    var total int64

    dbQuery := r.db.WithContext(ctx).Model(&models.Class{}).
        Where("classes.deleted_at IS NULL").
        Preload("ClassLevel").
        Preload("ClassArm")

    if schoolID != "" {
        dbQuery = dbQuery.Where("classes.school_id = ?", schoolID)
    }

    searchTerm := "%" + query + "%"
    dbQuery = dbQuery.Where("classes.class_code ILIKE ? OR classes.room_number ILIKE ?", searchTerm, searchTerm)

    if err := dbQuery.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count search results: %w", err)
    }

    offset := (page - 1) * limit
    if err := dbQuery.Offset(offset).Limit(limit).Order("classes.created_at DESC").Find(&classes).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to search classes: %w", err)
    }

    return classes, total, nil
}