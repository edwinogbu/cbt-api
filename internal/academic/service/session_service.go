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

// internal/academic/service/session_service.go

// Add these new methods

// ListSessions returns paginated sessions with filters
func (s *SessionService) ListSessions(req *dto.ListSessionsRequest) (*dto.SessionListResponse, error) {
    // Build filters map
    filters := make(map[string]interface{})

    if req.SchoolID != "" {
        filters["school_id"] = req.SchoolID
    }
    if req.Search != "" {
        filters["search"] = req.Search
    }
    if req.Status != "" && req.Status != "all" {
        filters["status"] = req.Status
    }
    if req.IsActive != nil {
        filters["is_active"] = req.IsActive
    }
    if req.StartDate != "" {
        filters["start_date"] = req.StartDate
    }
    if req.EndDate != "" {
        filters["end_date"] = req.EndDate
    }

    // Set defaults
    page := req.Page
    if page < 1 {
        page = 1
    }
    limit := req.Limit
    if limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }

    sortBy := req.SortBy
    if sortBy == "" {
        sortBy = "created_at"
    }
    sortOrder := req.SortOrder
    if sortOrder == "" {
        sortOrder = "desc"
    }

    // Get sessions
    sessions, total, err := s.repo.ListSessions(filters, page, limit, sortBy, sortOrder)
    if err != nil {
        return nil, err
    }

    // Convert to response DTOs
    items := make([]dto.SessionResponse, len(sessions))
    for i, session := range sessions {
        items[i] = *s.toResponse(&session)
    }

    // Calculate total pages
    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    // Build response
    response := &dto.SessionListResponse{
        Items: items,
        Pagination: dto.PaginationMeta{
            Page:       page,
            Limit:      limit,
            Total:      total,
            TotalPages: totalPages,
        },
    }

    return response, nil
}

// GetSessionStats returns session statistics
func (s *SessionService) GetSessionStats(schoolID string) (*dto.SessionStatsResponse, error) {
    return s.repo.GetSessionStats(schoolID)
}

// BulkDeleteSessions deletes multiple sessions
func (s *SessionService) BulkDeleteSessions(ids []string) (int64, error) {
    return s.repo.BulkDeleteSessions(ids)
}

// GetSessionTimeline returns sessions ordered by start_date
func (s *SessionService) GetSessionTimeline(schoolID string) ([]dto.SessionResponse, error) {
    sessions, err := s.repo.GetSessionTimeline(schoolID)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.SessionResponse, len(sessions))
    for i, session := range sessions {
        responses[i] = *s.toResponse(&session)
    }
    return responses, nil
}

// GetSessionSummary returns a summary for dashboard
func (s *SessionService) GetSessionSummary(schoolID string) (*dto.SessionSummaryResponse, error) {
    stats, err := s.repo.GetSessionStats(schoolID)
    if err != nil {
        return nil, err
    }

    var current *dto.SessionResponse
    currentSession, err := s.repo.FindCurrentBySchool(schoolID)
    if err == nil && currentSession != nil {
        current = s.toResponse(currentSession)
    }

    timeline, err := s.repo.GetSessionTimeline(schoolID)
    if err != nil {
        return nil, err
    }

    timelineResponses := make([]dto.SessionResponse, len(timeline))
    for i, session := range timeline {
        timelineResponses[i] = *s.toResponse(&session)
    }

    return &dto.SessionSummaryResponse{
        Current:  current,
        Stats:    *stats,
        Timeline: timelineResponses,
    }, nil
}