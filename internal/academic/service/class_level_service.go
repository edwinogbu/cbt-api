package service

import (
    "errors"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type ClassLevelService struct {
    repo *repository.ClassLevelRepository
}

func NewClassLevelService(repo *repository.ClassLevelRepository) *ClassLevelService {
    return &ClassLevelService{repo: repo}
}

func (s *ClassLevelService) Create(req *dto.CreateClassLevelRequest) (*dto.ClassLevelResponse, error) {
    level := &models.ClassLevel{
        ID:          uuid.New().String(),
        SchoolID:    req.SchoolID,
        Name:        req.Name,
        LevelNumber: req.LevelNumber,
        Category:    req.Category,
        SortOrder:   req.SortOrder,
        IsActive:    true,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    if req.PromotionTo != nil && *req.PromotionTo != "" {
        level.PromotionTo = req.PromotionTo
    }
    
    if err := s.repo.Create(level); err != nil {
        return nil, err
    }
    
    return s.toResponse(level), nil
}

func (s *ClassLevelService) GetByID(id string) (*dto.ClassLevelResponse, error) {
    level, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("class level not found")
    }
    return s.toResponse(level), nil
}

func (s *ClassLevelService) GetBySchool(schoolID string) ([]dto.ClassLevelResponse, error) {
    levels, err := s.repo.FindBySchool(schoolID)
    if err != nil {
        return nil, err
    }
    
    var responses []dto.ClassLevelResponse
    for _, level := range levels {
        responses = append(responses, *s.toResponse(&level))
    }
    return responses, nil
}

func (s *ClassLevelService) GetByCategory(schoolID, category string) ([]dto.ClassLevelResponse, error) {
    levels, err := s.repo.FindByCategory(schoolID, category)
    if err != nil {
        return nil, err
    }
    
    var responses []dto.ClassLevelResponse
    for _, level := range levels {
        responses = append(responses, *s.toResponse(&level))
    }
    return responses, nil
}

func (s *ClassLevelService) Update(id string, req *dto.UpdateClassLevelRequest) (*dto.ClassLevelResponse, error) {
    level, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("class level not found")
    }
    
    if req.Name != "" {
        level.Name = req.Name
    }
    if req.LevelNumber != nil {
        level.LevelNumber = *req.LevelNumber
    }
    if req.Category != "" {
        level.Category = req.Category
    }
    if req.PromotionTo != nil {
        level.PromotionTo = req.PromotionTo
    }
    if req.SortOrder != nil {
        level.SortOrder = *req.SortOrder
    }
    if req.IsActive != nil {
        level.IsActive = *req.IsActive
    }
    level.UpdatedAt = time.Now()
    
    if err := s.repo.Update(level); err != nil {
        return nil, err
    }
    
    return s.toResponse(level), nil
}

// func (s *ClassLevelService) Delete(id string) error {
//     return s.repo.Delete(id)
// }
// Delete soft-deletes a single class level by ID
func (s *ClassLevelService) Delete(id string) error {
    if id == "" {
        return errors.New("id is required")
    }
    return s.repo.Delete(id)
}

// BulkDeleteClassLevels deletes multiple class levels by IDs
func (s *ClassLevelService) BulkDeleteClassLevels(ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided for bulk delete")
    }
    return s.repo.BulkDelete(ids)
}


// NEW: List with pagination, filtering, sorting
func (s *ClassLevelService) List(req *dto.ListClassLevelsRequest) (*dto.SearchClassLevelsResponse, error) {
    levels, total, err := s.repo.List(
        req.SchoolID,
        req.Category,
        req.Search,
        req.SortBy,
        req.SortDir,
        req.Page,
        req.Limit,
        req.IsActive,
    )
    if err != nil {
        return nil, err
    }
    
    var responses []dto.ClassLevelResponse
    for _, level := range levels {
        responses = append(responses, *s.toResponse(&level))
    }
    
    totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
    
    return &dto.SearchClassLevelsResponse{
        Items:      responses,
        Total:      total,
        Page:       req.Page,
        Limit:      req.Limit,
        TotalPages: totalPages,
    }, nil
}

// NEW: Bulk delete
func (s *ClassLevelService) BulkDelete(ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }
    return s.repo.BulkDelete(ids)
}

// NEW: Get stats
func (s *ClassLevelService) GetStats(schoolID string) (*dto.ClassLevelStatsResponse, error) {
    stats, err := s.repo.GetStats(schoolID)
    if err != nil {
        return nil, err
    }
    
    return &dto.ClassLevelStatsResponse{
        Total:          stats.Total,
        Active:         stats.Active,
        Inactive:       stats.Inactive,
        ByCategory:     stats.ByCategory,
        TotalLevels:    stats.TotalLevels,
        AvgLevelNumber: stats.AvgLevelNumber,
    }, nil
}

// NEW: Search (uses List with search parameter)
func (s *ClassLevelService) Search(schoolID, query string, page, limit int) (*dto.SearchClassLevelsResponse, error) {
    req := &dto.ListClassLevelsRequest{
        SchoolID: schoolID,
        Search:   query,
        Page:     page,
        Limit:    limit,
        SortBy:   "name",
        SortDir:  "asc",
    }
    return s.List(req)
}

func (s *ClassLevelService) toResponse(level *models.ClassLevel) *dto.ClassLevelResponse {
    var promotionTo *string
    if level.PromotionTo != nil && *level.PromotionTo != "" {
        promotionTo = level.PromotionTo
    }
    
    return &dto.ClassLevelResponse{
        ID:          level.ID,
        SchoolID:    level.SchoolID,
        Name:        level.Name,
        LevelNumber: level.LevelNumber,
        Category:    level.Category,
        PromotionTo: promotionTo,
        SortOrder:   level.SortOrder,
        IsActive:    level.IsActive,
        CreatedAt:   level.CreatedAt,
        UpdatedAt:   level.UpdatedAt,
    }
}




// package service

// import (
//     "errors"
//     "time"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
// )

// type ClassLevelService struct {
//     repo *repository.ClassLevelRepository
// }

// func NewClassLevelService(repo *repository.ClassLevelRepository) *ClassLevelService {
//     return &ClassLevelService{repo: repo}
// }

// func (s *ClassLevelService) Create(req *dto.CreateClassLevelRequest) (*dto.ClassLevelResponse, error) {
//     level := &models.ClassLevel{
//         ID:          uuid.New().String(),
//         SchoolID:    req.SchoolID,
//         Name:        req.Name,
//         LevelNumber: req.LevelNumber,
//         Category:    req.Category,
//         SortOrder:   req.SortOrder,
//         IsActive:    true,
//         CreatedAt:   time.Now(),
//         UpdatedAt:   time.Now(),
//     }
    
//     if req.PromotionTo != nil && *req.PromotionTo != "" {
//         level.PromotionTo = req.PromotionTo
//     }
    
//     if err := s.repo.Create(level); err != nil {
//         return nil, err
//     }
    
//     return s.toResponse(level), nil
// }

// func (s *ClassLevelService) GetByID(id string) (*dto.ClassLevelResponse, error) {
//     level, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("class level not found")
//     }
//     return s.toResponse(level), nil
// }

// func (s *ClassLevelService) GetBySchool(schoolID string) ([]dto.ClassLevelResponse, error) {
//     levels, err := s.repo.FindBySchool(schoolID)
//     if err != nil {
//         return nil, err
//     }
    
//     var responses []dto.ClassLevelResponse
//     for _, level := range levels {
//         responses = append(responses, *s.toResponse(&level))
//     }
//     return responses, nil
// }

// func (s *ClassLevelService) GetByCategory(schoolID, category string) ([]dto.ClassLevelResponse, error) {
//     levels, err := s.repo.FindByCategory(schoolID, category)
//     if err != nil {
//         return nil, err
//     }
    
//     var responses []dto.ClassLevelResponse
//     for _, level := range levels {
//         responses = append(responses, *s.toResponse(&level))
//     }
//     return responses, nil
// }

// func (s *ClassLevelService) Update(id string, req *dto.UpdateClassLevelRequest) (*dto.ClassLevelResponse, error) {
//     level, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("class level not found")
//     }
    
//     if req.Name != "" {
//         level.Name = req.Name
//     }
//     if req.LevelNumber != nil {
//         level.LevelNumber = *req.LevelNumber
//     }
//     if req.Category != "" {
//         level.Category = req.Category
//     }
//     if req.PromotionTo != nil {
//         level.PromotionTo = req.PromotionTo
//     }
//     if req.SortOrder != nil {
//         level.SortOrder = *req.SortOrder
//     }
//     if req.IsActive != nil {
//         level.IsActive = *req.IsActive
//     }
//     level.UpdatedAt = time.Now()
    
//     if err := s.repo.Update(level); err != nil {
//         return nil, err
//     }
    
//     return s.toResponse(level), nil
// }

// func (s *ClassLevelService) Delete(id string) error {
//     return s.repo.Delete(id)
// }

// func (s *ClassLevelService) toResponse(level *models.ClassLevel) *dto.ClassLevelResponse {
//     var promotionTo *string
//     if level.PromotionTo != nil && *level.PromotionTo != "" {
//         promotionTo = level.PromotionTo
//     }
    
//     return &dto.ClassLevelResponse{
//         ID:          level.ID,
//         SchoolID:    level.SchoolID,
//         Name:        level.Name,
//         LevelNumber: level.LevelNumber,
//         Category:    level.Category,
//         PromotionTo: promotionTo,
//         SortOrder:   level.SortOrder,
//         IsActive:    level.IsActive,
//         CreatedAt:   level.CreatedAt,
//         UpdatedAt:   level.UpdatedAt,
//     }
// }