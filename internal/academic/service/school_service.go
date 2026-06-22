package service

import (
    "errors"
    "fmt"
    "math/rand"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type SchoolService struct {
    repo *repository.SchoolRepository
}

func NewSchoolService(repo *repository.SchoolRepository) *SchoolService {
    return &SchoolService{repo: repo}
}

// func (s *SchoolService) Create(req *dto.CreateSchoolRequest) (*dto.SchoolResponse, error) {
//     // Check if code already exists
//     existing, _ := s.repo.FindByCode(req.Code)
//     if existing != nil {
//         return nil, errors.New("school code already exists")
//     }
    
//     school := &models.School{
//         ID:        uuid.New().String(),
//         Name:      req.Name,
//         Code:      req.Code,
//         Address:   req.Address,
//         Phone:     req.Phone,
//         Email:     req.Email,
//         Logo:      req.Logo,
//         Website:   req.Website,
//         IsActive:  true,
//         CreatedAt: time.Now(),
//         UpdatedAt: time.Now(),
//     }
    
//     if err := s.repo.Create(school); err != nil {
//         return nil, err
//     }
    
//     return s.toResponse(school), nil
// }


// generateUniqueSchoolCode creates a unique code (e.g., SCH-ABC123)
func (s *SchoolService) generateUniqueSchoolCode() (string, error) {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const prefix = "SCH-"
    const randomLength = 6
    rand.Seed(time.Now().UnixNano())

    for attempts := 0; attempts < 10; attempts++ {
        code := prefix
        for i := 0; i < randomLength; i++ {
            code += string(charset[rand.Intn(len(charset))])
        }
        exists, err := s.repo.CodeExists(code)
        if err != nil {
            return "", err
        }
        if !exists {
            return code, nil
        }
    }
    return "", errors.New("failed to generate unique school code after 10 attempts")
}

// Create a new school
func (s *SchoolService) Create(req *dto.CreateSchoolRequest) (*dto.SchoolResponse, error) {
    // Auto‑generate code if not provided
    if req.Code == "" {
        code, err := s.generateUniqueSchoolCode()
        if err != nil {
            return nil, err
        }
        req.Code = code
    } else {
        // Ensure provided code is unique
        exists, err := s.repo.CodeExists(req.Code)
        if err != nil {
            return nil, err
        }
        if exists {
            return nil, fmt.Errorf("school code %s already exists", req.Code)
        }
    }

    // Original school creation logic
    school := &models.School{
        ID:        uuid.New().String(),
        Name:      req.Name,
        Code:      req.Code,
        Address:   req.Address,
        Phone:     req.Phone,
        Email:     req.Email,
        Logo:      req.Logo,
        Website:   req.Website,
        IsActive:  true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := s.repo.Create(school); err != nil {
        return nil, err
    }

    return s.toResponse(school), nil
}


func (s *SchoolService) GetByID(id string) (*dto.SchoolResponse, error) {
    school, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("school not found")
    }
    return s.toResponse(school), nil
}

func (s *SchoolService) GetAll(page, limit int) ([]dto.SchoolResponse, int64, error) {
    schools, total, err := s.repo.FindAll(page, limit)
    if err != nil {
        return nil, 0, err
    }
    
    var responses []dto.SchoolResponse
    for _, school := range schools {
        responses = append(responses, *s.toResponse(&school))
    }
    
    return responses, total, nil
}

func (s *SchoolService) Update(id string, req *dto.UpdateSchoolRequest) (*dto.SchoolResponse, error) {
    school, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("school not found")
    }
    
    if req.Name != "" {
        school.Name = req.Name
    }
    if req.Address != "" {
        school.Address = req.Address
    }
    if req.Phone != "" {
        school.Phone = req.Phone
    }
    if req.Email != "" {
        school.Email = req.Email
    }
    if req.Logo != "" {
        school.Logo = req.Logo
    }
    if req.Website != "" {
        school.Website = req.Website
    }
    if req.IsActive != nil {
        school.IsActive = *req.IsActive
    }
    school.UpdatedAt = time.Now()
    
    if err := s.repo.Update(school); err != nil {
        return nil, err
    }
    
    return s.toResponse(school), nil
}

func (s *SchoolService) Delete(id string) error {
    return s.repo.Delete(id)
}

func (s *SchoolService) toResponse(school *models.School) *dto.SchoolResponse {
    return &dto.SchoolResponse{
        ID:        school.ID,
        Name:      school.Name,
        Code:      school.Code,
        Address:   school.Address,
        Phone:     school.Phone,
        Email:     school.Email,
        Logo:      school.Logo,
        Website:   school.Website,
        IsActive:  school.IsActive,
        CreatedAt: school.CreatedAt,
        UpdatedAt: school.UpdatedAt,
    }
}