package service

import (
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