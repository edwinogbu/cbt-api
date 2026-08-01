package service

import (
    "context"
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
        // Get session name for each term
        sessionName := s.getSessionName(term.SessionID)
        responses = append(responses, *s.toResponseWithSession(&term, sessionName))
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

// ListAllTerms returns all terms with pagination
func (s *TermService) ListAllTerms(ctx context.Context, page, limit int) (*dto.TermListResponse, error) {
    terms, total, err := s.repo.ListAllTerms(ctx, page, limit)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TermResponse, len(terms))
    for i, term := range terms {
        // Get session name for each term
        sessionName := s.getSessionName(term.SessionID)
        responses[i] = s.buildTermResponse(&term, sessionName)
    }

    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.TermListResponse{
        Terms:      responses,
        Total:      total,
        Page:       page,
        Limit:      limit,
        TotalPages: totalPages,
    }, nil
}

// ListAllTermsBySchool returns all terms for a specific school
func (s *TermService) ListAllTermsBySchool(ctx context.Context, schoolID string, page, limit int) (*dto.TermListResponse, error) {
    terms, total, err := s.repo.ListAllTermsBySchool(ctx, schoolID, page, limit)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TermResponse, len(terms))
    for i, term := range terms {
        // Get session name for each term
        sessionName := s.getSessionName(term.SessionID)
        responses[i] = s.buildTermResponse(&term, sessionName)
    }

    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.TermListResponse{
        Terms:      responses,
        Total:      total,
        Page:       page,
        Limit:      limit,
        TotalPages: totalPages,
    }, nil
}

// GetAllTerms returns all terms without pagination (for dropdowns)
func (s *TermService) GetAllTerms(ctx context.Context) ([]dto.TermResponse, error) {
    terms, err := s.repo.GetAllTerms(ctx)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TermResponse, len(terms))
    for i, term := range terms {
        // Get session name for each term
        sessionName := s.getSessionName(term.SessionID)
        responses[i] = s.buildTermResponse(&term, sessionName)
    }

    return responses, nil
}

// Helper: Get session name by session ID
func (s *TermService) getSessionName(sessionID string) string {
    var session models.AcademicSession
    err := s.repo.GetDB().Where("id = ? AND deleted_at IS NULL", sessionID).First(&session).Error
    if err != nil {
        return ""
    }
    return session.Name
}

// Helper: Build TermResponse with session name
func (s *TermService) buildTermResponse(term *models.Term, sessionName string) dto.TermResponse {
    return dto.TermResponse{
        ID:          term.ID,
        SessionID:   term.SessionID,
        SessionName: sessionName,
        TermNumber:  term.TermNumber,
        Name:        term.Name,
        StartDate:   term.StartDate,
        EndDate:     term.EndDate,
        IsCurrent:   term.IsCurrent,
        IsActive:    term.IsActive,
        CreatedAt:   term.CreatedAt,
        UpdatedAt:   term.UpdatedAt,
    }
}

// toResponse returns a TermResponse without session name (for backward compatibility)
func (s *TermService) toResponse(term *models.Term) *dto.TermResponse {
    sessionName := s.getSessionName(term.SessionID)
    resp := s.buildTermResponse(term, sessionName)
    return &resp
}

// toResponseWithSession returns a TermResponse with session name
func (s *TermService) toResponseWithSession(term *models.Term, sessionName string) *dto.TermResponse {
    resp := s.buildTermResponse(term, sessionName)
    return &resp
}

// ============================================================
// NEW SERVICE METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListTerms - ALL PARAMETERS OPTIONAL
func (s *TermService) ListTerms(ctx context.Context, req *dto.ListTermsRequest) (*dto.TermListResponse, error) {
    terms, total, err := s.repo.ListWithFilters(ctx, req)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TermResponse, len(terms))
    for i, term := range terms {
        sessionName := s.getSessionName(term.SessionID)
        responses[i] = s.buildTermResponse(&term, sessionName)
    }

    totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.TermListResponse{
        Terms:      responses,
        Total:      total,
        Page:       req.Page,
        Limit:      req.Limit,
        TotalPages: totalPages,
    }, nil
}

// BulkDeleteTerms - bulk delete
func (s *TermService) BulkDeleteTerms(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }
    return s.repo.BulkDeleteTerms(ctx, ids)
}

// GetTermStats - statistics
func (s *TermService) GetTermStats(ctx context.Context, schoolID string) (*dto.TermStatsResponse, error) {
    return s.repo.GetTermStats(ctx, schoolID)
}

// SearchTerms - search
func (s *TermService) SearchTerms(ctx context.Context, query string, schoolID string, page, limit int) (*dto.TermListResponse, error) {
    terms, total, err := s.repo.SearchTerms(ctx, query, schoolID, page, limit)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TermResponse, len(terms))
    for i, term := range terms {
        sessionName := s.getSessionName(term.SessionID)
        responses[i] = s.buildTermResponse(&term, sessionName)
    }

    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.TermListResponse{
        Terms:      responses,
        Total:      total,
        Page:       page,
        Limit:      limit,
        TotalPages: totalPages,
    }, nil
}


// package service

// import (
//     "context"  // ✅ ADD THIS
//     "errors"
//     "time"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
// )

// type TermService struct {
//     repo *repository.TermRepository  // ✅ Field name is 'repo'
// }

// func NewTermService(repo *repository.TermRepository) *TermService {
//     return &TermService{repo: repo}
// }

// func (s *TermService) Create(req *dto.CreateTermRequest) (*dto.TermResponse, error) {
//     // Generate name if not provided
//     name := req.Name
//     if name == "" {
//         termNames := map[int]string{
//             1: "First Term",
//             2: "Second Term",
//             3: "Third Term",
//         }
//         name = termNames[req.TermNumber]
//     }
    
//     term := &models.Term{
//         ID:         uuid.New().String(),
//         SessionID:  req.SessionID,
//         TermNumber: req.TermNumber,
//         Name:       name,
//         StartDate:  req.StartDate,
//         EndDate:    req.EndDate,
//         IsCurrent:  req.IsCurrent,
//         IsActive:   true,
//         CreatedAt:  time.Now(),
//         UpdatedAt:  time.Now(),
//     }
    
//     if err := s.repo.Create(term); err != nil {  // ✅ Use 'repo', not 'termRepo'
//         return nil, err
//     }
    
//     // If this term is set as current, update others
//     if req.IsCurrent {
//         s.repo.SetCurrent(req.SessionID, term.ID)  // ✅ Use 'repo'
//     }
    
//     return s.toResponse(term), nil
// }

// func (s *TermService) GetByID(id string) (*dto.TermResponse, error) {
//     term, err := s.repo.FindByID(id)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, errors.New("term not found")
//     }
//     return s.toResponse(term), nil
// }

// func (s *TermService) GetBySession(sessionID string) ([]dto.TermResponse, error) {
//     terms, err := s.repo.FindBySession(sessionID)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, err
//     }
    
//     var responses []dto.TermResponse
//     for _, term := range terms {
//         responses = append(responses, *s.toResponse(&term))
//     }
//     return responses, nil
// }

// func (s *TermService) GetCurrent(sessionID string) (*dto.TermResponse, error) {
//     term, err := s.repo.FindCurrentBySession(sessionID)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, errors.New("no current term found")
//     }
//     return s.toResponse(term), nil
// }

// func (s *TermService) Update(id string, req *dto.UpdateTermRequest) (*dto.TermResponse, error) {
//     term, err := s.repo.FindByID(id)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, errors.New("term not found")
//     }
    
//     if req.Name != "" {
//         term.Name = req.Name
//     }
//     if req.StartDate != nil {
//         term.StartDate = *req.StartDate
//     }
//     if req.EndDate != nil {
//         term.EndDate = *req.EndDate
//     }
//     if req.IsCurrent != nil {
//         term.IsCurrent = *req.IsCurrent
//     }
//     if req.IsActive != nil {
//         term.IsActive = *req.IsActive
//     }
//     term.UpdatedAt = time.Now()
    
//     if err := s.repo.Update(term); err != nil {  // ✅ Use 'repo'
//         return nil, err
//     }
    
//     // If this term is set as current, update others
//     if term.IsCurrent {
//         s.repo.SetCurrent(term.SessionID, term.ID)  // ✅ Use 'repo'
//     }
    
//     return s.toResponse(term), nil
// }

// func (s *TermService) Delete(id string) error {
//     return s.repo.Delete(id)  // ✅ Use 'repo'
// }

// // ListAllTerms returns all terms with pagination
// func (s *TermService) ListAllTerms(ctx context.Context, page, limit int) (*dto.TermListResponse, error) {
//     terms, total, err := s.repo.ListAllTerms(ctx, page, limit)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.TermResponse, len(terms))
//     for i, term := range terms {
//         responses[i] = dto.TermResponse{
//             ID:          term.ID,
//             SessionID:   term.SessionID,
//             TermNumber:  term.TermNumber,
//             Name:        term.Name,
//             StartDate:   term.StartDate,
//             EndDate:     term.EndDate,
//             IsCurrent:   term.IsCurrent,
//             IsActive:    term.IsActive,
//             CreatedAt:   term.CreatedAt,
//             UpdatedAt:   term.UpdatedAt,
//         }
//     }

//     totalPages := int((total + int64(limit) - 1) / int64(limit))
//     if totalPages < 1 {
//         totalPages = 1
//     }

//     return &dto.TermListResponse{
//         Terms:      responses,
//         Total:      total,
//         Page:       page,
//         Limit:      limit,
//         TotalPages: totalPages,
//     }, nil
// }

// // ListAllTermsBySchool returns all terms for a specific school
// func (s *TermService) ListAllTermsBySchool(ctx context.Context, schoolID string, page, limit int) (*dto.TermListResponse, error) {
//     terms, total, err := s.repo.ListAllTermsBySchool(ctx, schoolID, page, limit)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.TermResponse, len(terms))
//     for i, term := range terms {
//         responses[i] = dto.TermResponse{
//             ID:          term.ID,
//             SessionID:   term.SessionID,
//             TermNumber:  term.TermNumber,
//             Name:        term.Name,
//             StartDate:   term.StartDate,
//             EndDate:     term.EndDate,
//             IsCurrent:   term.IsCurrent,
//             IsActive:    term.IsActive,
//             CreatedAt:   term.CreatedAt,
//             UpdatedAt:   term.UpdatedAt,
//         }
//     }

//     totalPages := int((total + int64(limit) - 1) / int64(limit))
//     if totalPages < 1 {
//         totalPages = 1
//     }

//     return &dto.TermListResponse{
//         Terms:      responses,
//         Total:      total,
//         Page:       page,
//         Limit:      limit,
//         TotalPages: totalPages,
//     }, nil
// }

// // GetAllTerms returns all terms without pagination (for dropdowns)
// func (s *TermService) GetAllTerms(ctx context.Context) ([]dto.TermResponse, error) {
//     terms, err := s.repo.GetAllTerms(ctx)  // ✅ Use 'repo'
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.TermResponse, len(terms))
//     for i, term := range terms {
//         responses[i] = dto.TermResponse{
//             ID:          term.ID,
//             SessionID:   term.SessionID,
//             TermNumber:  term.TermNumber,
//             Name:        term.Name,
//             StartDate:   term.StartDate,
//             EndDate:     term.EndDate,
//             IsCurrent:   term.IsCurrent,
//             IsActive:    term.IsActive,
//             CreatedAt:   term.CreatedAt,
//             UpdatedAt:   term.UpdatedAt,
//         }
//     }

//     return responses, nil
// }

// func (s *TermService) toResponse(term *models.Term) *dto.TermResponse {
//     return &dto.TermResponse{
//         ID:         term.ID,
//         SessionID:  term.SessionID,
//         TermNumber: term.TermNumber,
//         Name:       term.Name,
//         StartDate:  term.StartDate,
//         EndDate:    term.EndDate,
//         IsCurrent:  term.IsCurrent,
//         IsActive:   term.IsActive,
//         CreatedAt:  term.CreatedAt,
//         UpdatedAt:  term.UpdatedAt,
//     }
// }


// package service

// import (
//     "errors"
//     "time"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
// )

// type TermService struct {
//     repo *repository.TermRepository
// }

// func NewTermService(repo *repository.TermRepository) *TermService {
//     return &TermService{repo: repo}
// }

// func (s *TermService) Create(req *dto.CreateTermRequest) (*dto.TermResponse, error) {
//     // Generate name if not provided
//     name := req.Name
//     if name == "" {
//         termNames := map[int]string{
//             1: "First Term",
//             2: "Second Term",
//             3: "Third Term",
//         }
//         name = termNames[req.TermNumber]
//     }
    
//     term := &models.Term{
//         ID:         uuid.New().String(),
//         SessionID:  req.SessionID,
//         TermNumber: req.TermNumber,
//         Name:       name,
//         StartDate:  req.StartDate,
//         EndDate:    req.EndDate,
//         IsCurrent:  req.IsCurrent,
//         IsActive:   true,
//         CreatedAt:  time.Now(),
//         UpdatedAt:  time.Now(),
//     }
    
//     if err := s.repo.Create(term); err != nil {
//         return nil, err
//     }
    
//     // If this term is set as current, update others
//     if req.IsCurrent {
//         s.repo.SetCurrent(req.SessionID, term.ID)
//     }
    
//     return s.toResponse(term), nil
// }

// func (s *TermService) GetByID(id string) (*dto.TermResponse, error) {
//     term, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("term not found")
//     }
//     return s.toResponse(term), nil
// }

// func (s *TermService) GetBySession(sessionID string) ([]dto.TermResponse, error) {
//     terms, err := s.repo.FindBySession(sessionID)
//     if err != nil {
//         return nil, err
//     }
    
//     var responses []dto.TermResponse
//     for _, term := range terms {
//         responses = append(responses, *s.toResponse(&term))
//     }
//     return responses, nil
// }

// func (s *TermService) GetCurrent(sessionID string) (*dto.TermResponse, error) {
//     term, err := s.repo.FindCurrentBySession(sessionID)
//     if err != nil {
//         return nil, errors.New("no current term found")
//     }
//     return s.toResponse(term), nil
// }

// func (s *TermService) Update(id string, req *dto.UpdateTermRequest) (*dto.TermResponse, error) {
//     term, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("term not found")
//     }
    
//     if req.Name != "" {
//         term.Name = req.Name
//     }
//     if req.StartDate != nil {
//         term.StartDate = *req.StartDate
//     }
//     if req.EndDate != nil {
//         term.EndDate = *req.EndDate
//     }
//     if req.IsCurrent != nil {
//         term.IsCurrent = *req.IsCurrent
//     }
//     if req.IsActive != nil {
//         term.IsActive = *req.IsActive
//     }
//     term.UpdatedAt = time.Now()
    
//     if err := s.repo.Update(term); err != nil {
//         return nil, err
//     }
    
//     // If this term is set as current, update others
//     if term.IsCurrent {
//         s.repo.SetCurrent(term.SessionID, term.ID)
//     }
    
//     return s.toResponse(term), nil
// }

// func (s *TermService) Delete(id string) error {
//     return s.repo.Delete(id)
// }

// // ListAllTerms returns all terms with pagination
// func (s *TermService) ListAllTerms(ctx context.Context, page, limit int) (*dto.TermListResponse, error) {
//     terms, total, err := s.termRepo.ListAllTerms(ctx, page, limit)
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.TermResponse, len(terms))
//     for i, term := range terms {
//         responses[i] = dto.TermResponse{
//             ID:          term.ID,
//             SessionID:   term.SessionID,
//             TermNumber:  term.TermNumber,
//             Name:        term.Name,
//             StartDate:   term.StartDate,
//             EndDate:     term.EndDate,
//             IsCurrent:   term.IsCurrent,
//             IsActive:    term.IsActive,
//             CreatedAt:   term.CreatedAt,
//             UpdatedAt:   term.UpdatedAt,
//         }
//     }

//     totalPages := int((total + int64(limit) - 1) / int64(limit))
//     if totalPages < 1 {
//         totalPages = 1
//     }

//     return &dto.TermListResponse{
//         Terms:      responses,
//         Total:      total,
//         Page:       page,
//         Limit:      limit,
//         TotalPages: totalPages,
//     }, nil
// }

// // ListAllTermsBySchool returns all terms for a specific school
// func (s *TermService) ListAllTermsBySchool(ctx context.Context, schoolID string, page, limit int) (*dto.TermListResponse, error) {
//     terms, total, err := s.termRepo.ListAllTermsBySchool(ctx, schoolID, page, limit)
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.TermResponse, len(terms))
//     for i, term := range terms {
//         responses[i] = dto.TermResponse{
//             ID:          term.ID,
//             SessionID:   term.SessionID,
//             TermNumber:  term.TermNumber,
//             Name:        term.Name,
//             StartDate:   term.StartDate,
//             EndDate:     term.EndDate,
//             IsCurrent:   term.IsCurrent,
//             IsActive:    term.IsActive,
//             CreatedAt:   term.CreatedAt,
//             UpdatedAt:   term.UpdatedAt,
//         }
//     }

//     totalPages := int((total + int64(limit) - 1) / int64(limit))
//     if totalPages < 1 {
//         totalPages = 1
//     }

//     return &dto.TermListResponse{
//         Terms:      responses,
//         Total:      total,
//         Page:       page,
//         Limit:      limit,
//         TotalPages: totalPages,
//     }, nil
// }


// func (s *TermService) toResponse(term *models.Term) *dto.TermResponse {
//     return &dto.TermResponse{
//         ID:         term.ID,
//         SessionID:  term.SessionID,
//         TermNumber: term.TermNumber,
//         Name:       term.Name,
//         StartDate:  term.StartDate,
//         EndDate:    term.EndDate,
//         IsCurrent:  term.IsCurrent,
//         IsActive:   term.IsActive,
//         CreatedAt:  term.CreatedAt,
//         UpdatedAt:  term.UpdatedAt,
//     }
// }