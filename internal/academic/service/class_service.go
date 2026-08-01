package service

import (
     "context"  // ✅ ADD THIS
    "errors"
    "fmt"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type ClassService struct {
    classRepo      *repository.ClassRepository
    classLevelRepo *repository.ClassLevelRepository
    classArmRepo   *repository.ClassArmRepository
    sessionRepo    *repository.SessionRepository
}

func NewClassService(
    classRepo *repository.ClassRepository,
    classLevelRepo *repository.ClassLevelRepository,
    classArmRepo *repository.ClassArmRepository,
    sessionRepo *repository.SessionRepository,
) *ClassService {
    return &ClassService{
        classRepo:      classRepo,
        classLevelRepo: classLevelRepo,
        classArmRepo:   classArmRepo,
        sessionRepo:    sessionRepo,
    }
}

func (s *ClassService) Create(req *dto.CreateClassRequest) (*dto.ClassResponse, error) {
    // Verify class level exists
    classLevel, err := s.classLevelRepo.FindByID(req.ClassLevelID)
    if err != nil {
        return nil, errors.New("class level not found")
    }
    
    // Verify class arm exists
    classArm, err := s.classArmRepo.FindByID(req.ClassArmID)
    if err != nil {
        return nil, errors.New("class arm not found")
    }
    
    // Verify session exists
    _, err = s.sessionRepo.FindByID(req.SessionID)
    if err != nil {
        return nil, errors.New("session not found")
    }
    
    // Generate class code
    classCode := fmt.Sprintf("%s-%s-%s", 
        classLevel.Name, 
        classArm.Name,
        time.Now().Format("2006"))
    
    class := &models.Class{
        ID:           uuid.New().String(),
        SchoolID:     req.SchoolID,
        SessionID:    req.SessionID,
        ClassLevelID: req.ClassLevelID,
        ClassArmID:   req.ClassArmID,
        ClassCode:    classCode,
        RoomNumber:   req.RoomNumber,
        IsActive:     true,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    if req.TeacherID != nil && *req.TeacherID != "" {
        class.TeacherID = req.TeacherID
    }
    
    if err := s.classRepo.Create(class); err != nil {
        return nil, err
    }
    
    return s.toResponse(class), nil
}

func (s *ClassService) GetByID(id string) (*dto.ClassResponse, error) {
    class, err := s.classRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("class not found")
    }
    return s.toResponse(class), nil
}

func (s *ClassService) GetBySchool(schoolID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindBySchool(schoolID)
    if err != nil {
        return nil, err
    }
    
    return s.toResponseList(classes), nil
}

func (s *ClassService) GetBySession(sessionID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindBySession(sessionID)
    if err != nil {
        return nil, err
    }
    
    return s.toResponseList(classes), nil
}

func (s *ClassService) GetBySchoolAndSession(schoolID, sessionID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindBySchoolAndSession(schoolID, sessionID)
    if err != nil {
        return nil, err
    }
    
    return s.toResponseList(classes), nil
}

func (s *ClassService) Update(id string, req *dto.UpdateClassRequest) (*dto.ClassResponse, error) {
    class, err := s.classRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("class not found")
    }
    
    if req.TeacherID != nil {
        class.TeacherID = req.TeacherID
    }
    if req.RoomNumber != "" {
        class.RoomNumber = req.RoomNumber
    }
    if req.IsActive != nil {
        class.IsActive = *req.IsActive
    }
    class.UpdatedAt = time.Now()
    
    if err := s.classRepo.Update(class); err != nil {
        return nil, err
    }
    
    return s.toResponse(class), nil
}

func (s *ClassService) Delete(id string) error {
    return s.classRepo.Delete(id)
}

func (s *ClassService) toResponse(class *models.Class) *dto.ClassResponse {
    // Get class level details
    classLevel, _ := s.classLevelRepo.FindByID(class.ClassLevelID)
    classArm, _ := s.classArmRepo.FindByID(class.ClassArmID)
    
    studentCount, _ := s.classRepo.GetStudentCount(class.ID)
    
    var classLevelDTO dto.ClassLevelBriefDTO
    var classArmDTO dto.ClassArmBriefDTO
    
    if classLevel != nil {
        classLevelDTO = dto.ClassLevelBriefDTO{
            ID:          classLevel.ID,
            Name:        classLevel.Name,
            LevelNumber: classLevel.LevelNumber,
            Category:    classLevel.Category,
        }
    }
    
    if classArm != nil {
        classArmDTO = dto.ClassArmBriefDTO{
            ID:      classArm.ID,
            Name:    classArm.Name,
            ArmCode: classArm.ArmCode,
        }
    }
    
    return &dto.ClassResponse{
        ID:           class.ID,
        SchoolID:     class.SchoolID,
        SessionID:    class.SessionID,
        ClassLevel:   classLevelDTO,
        ClassArm:     classArmDTO,
        ClassCode:    class.ClassCode,
        TeacherID:    class.TeacherID,
        RoomNumber:   class.RoomNumber,
        StudentCount: int(studentCount),
        IsActive:     class.IsActive,
        CreatedAt:    class.CreatedAt,
        UpdatedAt:    class.UpdatedAt,
    }
}

func (s *ClassService) toResponseList(classes []models.Class) []dto.ClassResponse {
    var responses []dto.ClassResponse
    for _, class := range classes {
        responses = append(responses, *s.toResponse(&class))
    }
    return responses
}

// ============================================================
// NEW SERVICE METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListClasses - ALL PARAMETERS OPTIONAL
func (s *ClassService) ListClasses(ctx context.Context, req *dto.ListClassesRequest) (*dto.ClassListResponse, error) {
    classes, total, err := s.classRepo.ListWithFilters(ctx, req)
    if err != nil {
        return nil, err
    }

    responses := s.toResponseList(classes)

    totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.ClassListResponse{
        Items:      responses,
        Total:      total,
        Page:       req.Page,
        Limit:      req.Limit,
        TotalPages: totalPages,
    }, nil
}

// GetByTeacher - get classes by teacher ID
func (s *ClassService) GetByTeacher(ctx context.Context, teacherID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindByTeacher(teacherID)
    if err != nil {
        return nil, err
    }
    return s.toResponseList(classes), nil
}

// GetByArm - get classes by class arm ID
func (s *ClassService) GetByArm(ctx context.Context, classArmID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindByArm(ctx, classArmID)
    if err != nil {
        return nil, err
    }
    return s.toResponseList(classes), nil
}

// GetByLevel - get classes by class level ID
func (s *ClassService) GetByLevel(ctx context.Context, classLevelID string) ([]dto.ClassResponse, error) {
    classes, err := s.classRepo.FindByLevel(ctx, classLevelID)
    if err != nil {
        return nil, err
    }
    return s.toResponseList(classes), nil
}

// BulkDeleteClasses - bulk delete
func (s *ClassService) BulkDeleteClasses(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }
    return s.classRepo.BulkDeleteClasses(ctx, ids)
}

// GetClassStats - statistics
func (s *ClassService) GetClassStats(ctx context.Context, schoolID string) (*dto.ClassStatsResponse, error) {
    return s.classRepo.GetClassStats(ctx, schoolID)
}

// SearchClasses - search
func (s *ClassService) SearchClasses(ctx context.Context, query string, schoolID string, page, limit int) (*dto.ClassListResponse, error) {
    classes, total, err := s.classRepo.SearchClasses(ctx, query, schoolID, page, limit)
    if err != nil {
        return nil, err
    }

    responses := s.toResponseList(classes)

    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    return &dto.ClassListResponse{
        Items:      responses,
        Total:      total,
        Page:       page,
        Limit:      limit,
        TotalPages: totalPages,
    }, nil
}