package service

import (
    "errors"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type ClassArmService struct {
    repo *repository.ClassArmRepository
}

func NewClassArmService(repo *repository.ClassArmRepository) *ClassArmService {
    return &ClassArmService{repo: repo}
}

func (s *ClassArmService) Create(req *dto.CreateClassArmRequest) (*dto.ClassArmResponse, error) {
    arm := &models.ClassArm{
        ID:         uuid.New().String(),
        SchoolID:   req.SchoolID,
        Name:       req.Name,
        ArmCode:    req.ArmCode,
        Capacity:   req.Capacity,
        RoomNumber: req.RoomNumber,
        SortOrder:  req.SortOrder,
        IsActive:   true,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }
    
    if err := s.repo.Create(arm); err != nil {
        return nil, err
    }
    
    return s.toResponse(arm), nil
}

func (s *ClassArmService) GetByID(id string) (*dto.ClassArmResponse, error) {
    arm, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("class arm not found")
    }
    return s.toResponse(arm), nil
}

func (s *ClassArmService) GetBySchool(schoolID string) ([]dto.ClassArmResponse, error) {
    arms, err := s.repo.FindBySchool(schoolID)
    if err != nil {
        return nil, err
    }
    
    var responses []dto.ClassArmResponse
    for _, arm := range arms {
        responses = append(responses, *s.toResponse(&arm))
    }
    return responses, nil
}

func (s *ClassArmService) Update(id string, req *dto.UpdateClassArmRequest) (*dto.ClassArmResponse, error) {
    arm, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("class arm not found")
    }
    
    if req.Name != "" {
        arm.Name = req.Name
    }
    if req.ArmCode != "" {
        arm.ArmCode = req.ArmCode
    }
    if req.Capacity != nil {
        arm.Capacity = *req.Capacity
    }
    if req.RoomNumber != "" {
        arm.RoomNumber = req.RoomNumber
    }
    if req.SortOrder != nil {
        arm.SortOrder = *req.SortOrder
    }
    if req.IsActive != nil {
        arm.IsActive = *req.IsActive
    }
    arm.UpdatedAt = time.Now()
    
    if err := s.repo.Update(arm); err != nil {
        return nil, err
    }
    
    return s.toResponse(arm), nil
}

func (s *ClassArmService) Delete(id string) error {
    return s.repo.Delete(id)
}

func (s *ClassArmService) toResponse(arm *models.ClassArm) *dto.ClassArmResponse {
    return &dto.ClassArmResponse{
        ID:         arm.ID,
        SchoolID:   arm.SchoolID,
        Name:       arm.Name,
        ArmCode:    arm.ArmCode,
        Capacity:   arm.Capacity,
        RoomNumber: arm.RoomNumber,
        SortOrder:  arm.SortOrder,
        IsActive:   arm.IsActive,
        CreatedAt:  arm.CreatedAt,
        UpdatedAt:  arm.UpdatedAt,
    }
}