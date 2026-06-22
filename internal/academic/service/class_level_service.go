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

func (s *ClassLevelService) Delete(id string) error {
    return s.repo.Delete(id)
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