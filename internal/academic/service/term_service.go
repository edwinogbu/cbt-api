package service

import (
    "errors"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type TermService struct {
    repo *repository.TermRepository
}

func NewTermService(repo *repository.TermRepository) *TermService {
    return &TermService{repo: repo}
}

func (s *TermService) Create(req *dto.CreateTermRequest) (*dto.TermResponse, error) {
    // Generate name if not provided
    name := req.Name
    if name == "" {
        termNames := map[int]string{
            1: "First Term",
            2: "Second Term",
            3: "Third Term",
        }
        name = termNames[req.TermNumber]
    }
    
    term := &models.Term{
        ID:         uuid.New().String(),
        SessionID:  req.SessionID,
        TermNumber: req.TermNumber,
        Name:       name,
        StartDate:  req.StartDate,
        EndDate:    req.EndDate,
        IsCurrent:  req.IsCurrent,
        IsActive:   true,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }
    
    if err := s.repo.Create(term); err != nil {
        return nil, err
    }
    
    // If this term is set as current, update others
    if req.IsCurrent {
        s.repo.SetCurrent(req.SessionID, term.ID)
    }
    
    return s.toResponse(term), nil
}

func (s *TermService) GetByID(id string) (*dto.TermResponse, error) {
    term, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("term not found")
    }
    return s.toResponse(term), nil
}

func (s *TermService) GetBySession(sessionID string) ([]dto.TermResponse, error) {
    terms, err := s.repo.FindBySession(sessionID)
    if err != nil {
        return nil, err
    }
    
    var responses []dto.TermResponse
    for _, term := range terms {
        responses = append(responses, *s.toResponse(&term))
    }
    return responses, nil
}

func (s *TermService) GetCurrent(sessionID string) (*dto.TermResponse, error) {
    term, err := s.repo.FindCurrentBySession(sessionID)
    if err != nil {
        return nil, errors.New("no current term found")
    }
    return s.toResponse(term), nil
}

func (s *TermService) Update(id string, req *dto.UpdateTermRequest) (*dto.TermResponse, error) {
    term, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("term not found")
    }
    
    if req.Name != "" {
        term.Name = req.Name
    }
    if req.StartDate != nil {
        term.StartDate = *req.StartDate
    }
    if req.EndDate != nil {
        term.EndDate = *req.EndDate
    }
    if req.IsCurrent != nil {
        term.IsCurrent = *req.IsCurrent
    }
    if req.IsActive != nil {
        term.IsActive = *req.IsActive
    }
    term.UpdatedAt = time.Now()
    
    if err := s.repo.Update(term); err != nil {
        return nil, err
    }
    
    // If this term is set as current, update others
    if term.IsCurrent {
        s.repo.SetCurrent(term.SessionID, term.ID)
    }
    
    return s.toResponse(term), nil
}

func (s *TermService) Delete(id string) error {
    return s.repo.Delete(id)
}

func (s *TermService) toResponse(term *models.Term) *dto.TermResponse {
    return &dto.TermResponse{
        ID:         term.ID,
        SessionID:  term.SessionID,
        TermNumber: term.TermNumber,
        Name:       term.Name,
        StartDate:  term.StartDate,
        EndDate:    term.EndDate,
        IsCurrent:  term.IsCurrent,
        IsActive:   term.IsActive,
        CreatedAt:  term.CreatedAt,
        UpdatedAt:  term.UpdatedAt,
    }
}