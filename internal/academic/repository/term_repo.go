package repository

import (
    "context"  // ✅ ADDED
    "errors"  // ✅ ADD THIS - fixes undefined: errors
    "fmt"
        
    "cbt-api/internal/academic/dto"  // ✅ ADD THIS - fixes undefined: dto
    "cbt-api/internal/models"
    "gorm.io/gorm"
)

type TermRepository struct {
    db *gorm.DB
}

func NewTermRepository(db *gorm.DB) *TermRepository {
    return &TermRepository{db: db}
}

func (r *TermRepository) Create(term *models.Term) error {
    return r.db.Create(term).Error
}

func (r *TermRepository) FindByID(id string) (*models.Term, error) {
    var term models.Term
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&term).Error
    if err != nil {
        return nil, err
    }
    return &term, nil
}

func (r *TermRepository) FindBySession(sessionID string) ([]models.Term, error) {
    var terms []models.Term
    err := r.db.Where("session_id = ? AND deleted_at IS NULL", sessionID).
        Order("term_number ASC").Find(&terms).Error
    return terms, err
}

func (r *TermRepository) FindCurrentBySession(sessionID string) (*models.Term, error) {
    var term models.Term
    err := r.db.Where("session_id = ? AND is_current = true AND deleted_at IS NULL", sessionID).
        First(&term).Error
    if err != nil {
        return nil, err
    }
    return &term, nil
}

func (r *TermRepository) Update(term *models.Term) error {
    return r.db.Save(term).Error
}

func (r *TermRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.Term{}).Error
}

func (r *TermRepository) SetCurrent(sessionID, termID string) error {
    // First, unset current for all terms in this session
    err := r.db.Model(&models.Term{}).
        Where("session_id = ?", sessionID).
        Update("is_current", false).Error
    if err != nil {
        return err
    }
    
    // Then set the new current term
    return r.db.Model(&models.Term{}).
        Where("id = ?", termID).
        Update("is_current", true).Error
}

// ListAllTerms returns all terms with optional pagination
func (r *TermRepository) ListAllTerms(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var terms []models.Term
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Term{}).Where("deleted_at IS NULL")

    // Count total records
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count terms: %w", err)
    }

    offset := (page - 1) * limit
    err := query.
        Offset(offset).
        Limit(limit).
        Order("created_at DESC").
        Find(&terms).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list terms: %w", err)
    }

    return terms, total, nil
}

// ListAllTermsWithSession returns all terms with session details
func (r *TermRepository) ListAllTermsWithSession(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var terms []models.Term
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Term{}).
        Where("terms.deleted_at IS NULL").
        Preload("Session") // Assuming Term has Session relationship

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count terms: %w", err)
    }

    offset := (page - 1) * limit
    err := query.
        Offset(offset).
        Limit(limit).
        Order("terms.created_at DESC").
        Find(&terms).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list terms with session: %w", err)
    }

    return terms, total, nil
}

// ListAllTermsBySchool returns all terms for a specific school
func (r *TermRepository) ListAllTermsBySchool(ctx context.Context, schoolID string, page, limit int) ([]models.Term, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var terms []models.Term
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Term{}).
        Where("deleted_at IS NULL").
        Joins("JOIN academic_sessions ON academic_sessions.id = terms.session_id").
        Where("academic_sessions.school_id = ?", schoolID)

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count terms: %w", err)
    }

    offset := (page - 1) * limit
    err := query.
        Offset(offset).
        Limit(limit).
        Order("terms.created_at DESC").
        Find(&terms).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list terms: %w", err)
    }

    return terms, total, nil
}

// ListActiveTerms returns only active terms
func (r *TermRepository) ListActiveTerms(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var terms []models.Term
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Term{}).
        Where("deleted_at IS NULL AND is_active = ?", true)

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count active terms: %w", err)
    }

    offset := (page - 1) * limit
    err := query.
        Offset(offset).
        Limit(limit).
        Order("created_at DESC").
        Find(&terms).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list active terms: %w", err)
    }

    return terms, total, nil
}

// GetAllTerms returns all terms without pagination (for dropdown/selection)
func (r *TermRepository) GetAllTerms(ctx context.Context) ([]models.Term, error) {
    var terms []models.Term
    err := r.db.WithContext(ctx).
        Where("deleted_at IS NULL").
        Order("created_at DESC").
        Find(&terms).Error
    if err != nil {
        return nil, fmt.Errorf("failed to get all terms: %w", err)
    }
    return terms, nil
}

// GetAllTermsBySession returns all terms for a session without pagination
func (r *TermRepository) GetAllTermsBySession(ctx context.Context, sessionID string) ([]models.Term, error) {
    var terms []models.Term
    err := r.db.WithContext(ctx).
        Where("session_id = ? AND deleted_at IS NULL", sessionID).
        Order("term_number ASC").
        Find(&terms).Error
    if err != nil {
        return nil, fmt.Errorf("failed to get terms by session: %w", err)
    }
    return terms, nil
}

// GetDB returns the database instance (for service to query related data)
func (r *TermRepository) GetDB() *gorm.DB {
    return r.db
}

// ============================================================
// NEW REPOSITORY METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListWithFilters - ALL PARAMETERS OPTIONAL
func (r *TermRepository) ListWithFilters(ctx context.Context, req *dto.ListTermsRequest) ([]models.Term, int64, error) {
    // Set defaults
    if req.Page < 1 {
        req.Page = 1
    }
    if req.Limit < 1 || req.Limit > 100 {
        req.Limit = 20
    }

    var terms []models.Term
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Term{}).Where("terms.deleted_at IS NULL")

    // ALL FILTERS ARE OPTIONAL
    if req.SessionID != "" {
        query = query.Where("terms.session_id = ?", req.SessionID)
    }

    if req.SchoolID != "" {
        query = query.Joins("JOIN academic_sessions ON academic_sessions.id = terms.session_id").
            Where("academic_sessions.school_id = ?", req.SchoolID)
    }

    if req.IsActive != nil {
        query = query.Where("terms.is_active = ?", *req.IsActive)
    }

    if req.IsCurrent != nil {
        query = query.Where("terms.is_current = ?", *req.IsCurrent)
    }

    if req.Search != "" {
        searchTerm := "%" + req.Search + "%"
        query = query.Where("terms.name ILIKE ? OR terms.term_number::text ILIKE ?", searchTerm, searchTerm)
    }

    // Count total
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count terms: %w", err)
    }

    // Sorting (with defaults)
    sortBy := req.SortBy
    if sortBy == "" {
        sortBy = "terms.created_at"
    }
    sortOrder := req.SortOrder
    if sortOrder == "" {
        sortOrder = "DESC"
    }
    query = query.Order(sortBy + " " + sortOrder)

    // Pagination
    offset := (req.Page - 1) * req.Limit
    if err := query.Offset(offset).Limit(req.Limit).Find(&terms).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to list terms: %w", err)
    }

    return terms, total, nil
}

// BulkDeleteTerms - bulk delete
func (r *TermRepository) BulkDeleteTerms(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }

    result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Term{})
    if result.Error != nil {
        return 0, fmt.Errorf("bulk delete failed: %w", result.Error)
    }
    return result.RowsAffected, nil
}

// GetTermStats - statistics
func (r *TermRepository) GetTermStats(ctx context.Context, schoolID string) (*dto.TermStatsResponse, error) {
    stats := &dto.TermStatsResponse{
        BySession: make(map[string]int64),
    }

    query := r.db.WithContext(ctx).Model(&models.Term{}).Where("terms.deleted_at IS NULL")

    if schoolID != "" {
        query = query.Joins("JOIN academic_sessions ON academic_sessions.id = terms.session_id").
            Where("academic_sessions.school_id = ?", schoolID)
    }

    // Total
    if err := query.Count(&stats.Total).Error; err != nil {
        return nil, fmt.Errorf("failed to get total: %w", err)
    }

    // Active
    activeQuery := query
    if err := activeQuery.Where("terms.is_active = ?", true).Count(&stats.Active).Error; err != nil {
        return nil, fmt.Errorf("failed to get active: %w", err)
    }

    // Inactive
    stats.Inactive = stats.Total - stats.Active

    // Current
    currentQuery := query
    if err := currentQuery.Where("terms.is_current = ?", true).Count(&stats.Current).Error; err != nil {
        return nil, fmt.Errorf("failed to get current: %w", err)
    }

    // By Session
    var sessionResults []struct {
        SessionID string
        Count     int64
    }
    if err := query.Select("session_id, COUNT(*) as count").
        Group("session_id").
        Scan(&sessionResults).Error; err != nil {
        return nil, fmt.Errorf("failed to get by session: %w", err)
    }

    for _, result := range sessionResults {
        stats.BySession[result.SessionID] = result.Count
    }

    stats.TotalTerms = stats.Total

    return stats, nil
}

// SearchTerms - search
func (r *TermRepository) SearchTerms(ctx context.Context, query string, schoolID string, page, limit int) ([]models.Term, int64, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    var terms []models.Term
    var total int64

    dbQuery := r.db.WithContext(ctx).Model(&models.Term{}).Where("terms.deleted_at IS NULL")

    if schoolID != "" {
        dbQuery = dbQuery.Joins("JOIN academic_sessions ON academic_sessions.id = terms.session_id").
            Where("academic_sessions.school_id = ?", schoolID)
    }

    searchTerm := "%" + query + "%"
    dbQuery = dbQuery.Where("terms.name ILIKE ? OR terms.term_number::text ILIKE ?", searchTerm, searchTerm)

    if err := dbQuery.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count search results: %w", err)
    }

    offset := (page - 1) * limit
    if err := dbQuery.Offset(offset).Limit(limit).Order("terms.created_at DESC").Find(&terms).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to search terms: %w", err)
    }

    return terms, total, nil
}


// package repository

// import (
//      "context"  
//     "fmt"     
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
// )

// type TermRepository struct {
//     db *gorm.DB
// }

// func NewTermRepository(db *gorm.DB) *TermRepository {
//     return &TermRepository{db: db}
// }

// func (r *TermRepository) Create(term *models.Term) error {
//     return r.db.Create(term).Error
// }

// func (r *TermRepository) FindByID(id string) (*models.Term, error) {
//     var term models.Term
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&term).Error
//     if err != nil {
//         return nil, err
//     }
//     return &term, nil
// }

// func (r *TermRepository) FindBySession(sessionID string) ([]models.Term, error) {
//     var terms []models.Term
//     err := r.db.Where("session_id = ? AND deleted_at IS NULL", sessionID).
//         Order("term_number ASC").Find(&terms).Error
//     return terms, err
// }

// func (r *TermRepository) FindCurrentBySession(sessionID string) (*models.Term, error) {
//     var term models.Term
//     err := r.db.Where("session_id = ? AND is_current = true AND deleted_at IS NULL", sessionID).
//         First(&term).Error
//     if err != nil {
//         return nil, err
//     }
//     return &term, nil
// }

// func (r *TermRepository) Update(term *models.Term) error {
//     return r.db.Save(term).Error
// }

// func (r *TermRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.Term{}).Error
// }

// func (r *TermRepository) SetCurrent(sessionID, termID string) error {
//     // First, unset current for all terms in this session
//     err := r.db.Model(&models.Term{}).
//         Where("session_id = ?", sessionID).
//         Update("is_current", false).Error
//     if err != nil {
//         return err
//     }
    
//     // Then set the new current term
//     return r.db.Model(&models.Term{}).
//         Where("id = ?", termID).
//         Update("is_current", true).Error
// }

// // ListAllTerms returns all terms with optional pagination
// func (r *TermRepository) ListAllTerms(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     var terms []models.Term
//     var total int64

//     query := r.db.WithContext(ctx).Model(&models.Term{}).Where("deleted_at IS NULL")

//     // Count total records
//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to count terms: %w", err)
//     }

//     offset := (page - 1) * limit
//     err := query.
//         Offset(offset).
//         Limit(limit).
//         Order("created_at DESC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, 0, fmt.Errorf("failed to list terms: %w", err)
//     }

//     return terms, total, nil
// }

// // ListAllTermsWithSession returns all terms with session details
// func (r *TermRepository) ListAllTermsWithSession(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     var terms []models.Term
//     var total int64

//     query := r.db.WithContext(ctx).Model(&models.Term{}).
//         Where("terms.deleted_at IS NULL").
//         Preload("Session") // Assuming Term has Session relationship

//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to count terms: %w", err)
//     }

//     offset := (page - 1) * limit
//     err := query.
//         Offset(offset).
//         Limit(limit).
//         Order("terms.created_at DESC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, 0, fmt.Errorf("failed to list terms with session: %w", err)
//     }

//     return terms, total, nil
// }

// // ListAllTermsBySchool returns all terms for a specific school
// func (r *TermRepository) ListAllTermsBySchool(ctx context.Context, schoolID string, page, limit int) ([]models.Term, int64, error) {
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     var terms []models.Term
//     var total int64

//     query := r.db.WithContext(ctx).Model(&models.Term{}).
//         Where("deleted_at IS NULL").
//         Joins("JOIN academic_sessions ON academic_sessions.id = terms.session_id").
//         Where("academic_sessions.school_id = ?", schoolID)

//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to count terms: %w", err)
//     }

//     offset := (page - 1) * limit
//     err := query.
//         Offset(offset).
//         Limit(limit).
//         Order("terms.created_at DESC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, 0, fmt.Errorf("failed to list terms: %w", err)
//     }

//     return terms, total, nil
// }

// // ListActiveTerms returns only active terms
// func (r *TermRepository) ListActiveTerms(ctx context.Context, page, limit int) ([]models.Term, int64, error) {
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     var terms []models.Term
//     var total int64

//     query := r.db.WithContext(ctx).Model(&models.Term{}).
//         Where("deleted_at IS NULL AND is_active = ?", true)

//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, fmt.Errorf("failed to count active terms: %w", err)
//     }

//     offset := (page - 1) * limit
//     err := query.
//         Offset(offset).
//         Limit(limit).
//         Order("created_at DESC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, 0, fmt.Errorf("failed to list active terms: %w", err)
//     }

//     return terms, total, nil
// }

// // GetAllTerms returns all terms without pagination (for dropdown/selection)
// func (r *TermRepository) GetAllTerms(ctx context.Context) ([]models.Term, error) {
//     var terms []models.Term
//     err := r.db.WithContext(ctx).
//         Where("deleted_at IS NULL").
//         Order("created_at DESC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, fmt.Errorf("failed to get all terms: %w", err)
//     }
//     return terms, nil
// }

// // GetAllTermsBySession returns all terms for a session without pagination
// func (r *TermRepository) GetAllTermsBySession(ctx context.Context, sessionID string) ([]models.Term, error) {
//     var terms []models.Term
//     err := r.db.WithContext(ctx).
//         Where("session_id = ? AND deleted_at IS NULL", sessionID).
//         Order("term_number ASC").
//         Find(&terms).Error
//     if err != nil {
//         return nil, fmt.Errorf("failed to get terms by session: %w", err)
//     }
//     return terms, nil
// }