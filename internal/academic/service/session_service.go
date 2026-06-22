package service

import (
    "errors"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type SessionService struct {
    repo *repository.SessionRepository
}

func NewSessionService(repo *repository.SessionRepository) *SessionService {
    return &SessionService{repo: repo}
}

func (s *SessionService) Create(req *dto.CreateSessionRequest) (*dto.SessionResponse, error) {
    session := &models.AcademicSession{
        ID:        uuid.New().String(),
        SchoolID:  req.SchoolID,
        Name:      req.Name,
        StartDate: req.StartDate,
        EndDate:   req.EndDate,
        IsCurrent: req.IsCurrent,
        IsActive:  true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    if err := s.repo.Create(session); err != nil {
        return nil, err
    }
    
    // If this session is set as current, update others
    if req.IsCurrent {
        s.repo.SetCurrent(req.SchoolID, session.ID)
    }
    
    return s.toResponse(session), nil
}

func (s *SessionService) GetByID(id string) (*dto.SessionResponse, error) {
    session, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("session not found")
    }
    return s.toResponse(session), nil
}

func (s *SessionService) GetBySchool(schoolID string) ([]dto.SessionResponse, error) {
    sessions, err := s.repo.FindBySchool(schoolID)
    if err != nil {
        return nil, err
    }
    
    var responses []dto.SessionResponse
    for _, session := range sessions {
        responses = append(responses, *s.toResponse(&session))
    }
    return responses, nil
}

func (s *SessionService) GetCurrent(schoolID string) (*dto.SessionResponse, error) {
    session, err := s.repo.FindCurrentBySchool(schoolID)
    if err != nil {
        return nil, errors.New("no current session found")
    }
    return s.toResponse(session), nil
}

func (s *SessionService) Update(id string, req *dto.UpdateSessionRequest) (*dto.SessionResponse, error) {
    session, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("session not found")
    }
    
    if req.Name != "" {
        session.Name = req.Name
    }
    if req.StartDate != nil {
        session.StartDate = *req.StartDate
    }
    if req.EndDate != nil {
        session.EndDate = *req.EndDate
    }
    if req.IsCurrent != nil {
        session.IsCurrent = *req.IsCurrent
    }
    if req.IsActive != nil {
        session.IsActive = *req.IsActive
    }
    session.UpdatedAt = time.Now()
    
    if err := s.repo.Update(session); err != nil {
        return nil, err
    }
    
    // If this session is set as current, update others
    if session.IsCurrent {
        s.repo.SetCurrent(session.SchoolID, session.ID)
    }
    
    return s.toResponse(session), nil
}

func (s *SessionService) Delete(id string) error {
    return s.repo.Delete(id)
}

func (s *SessionService) toResponse(session *models.AcademicSession) *dto.SessionResponse {
    return &dto.SessionResponse{
        ID:        session.ID,
        SchoolID:  session.SchoolID,
        Name:      session.Name,
        StartDate: session.StartDate,
        EndDate:   session.EndDate,
        IsCurrent: session.IsCurrent,
        IsActive:  session.IsActive,
        CreatedAt: session.CreatedAt,
        UpdatedAt: session.UpdatedAt,
    }
}