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

type StudentService struct {
    studentRepo *repository.StudentRepository
    userRepo    *repository.UserRepository
    classRepo   *repository.ClassRepository
}

func NewStudentService(
    studentRepo *repository.StudentRepository,
    userRepo *repository.UserRepository,
    classRepo *repository.ClassRepository,
) *StudentService {
    return &StudentService{
        studentRepo: studentRepo,
        userRepo:    userRepo,
        classRepo:   classRepo,
    }
}

func (s *StudentService) Create(req *dto.CreateStudentRequest) (*dto.StudentDetailResponse, error) {
    user, err := s.userRepo.FindByID(req.UserID)
    if err != nil {
        return nil, errors.New("user not found")
    }
    existing, _ := s.studentRepo.FindByAdmissionNo(req.AdmissionNo)
    if existing != nil {
        return nil, errors.New("admission number already exists")
    }
    if req.ClassID != "" {
        _, err := s.classRepo.FindByID(req.ClassID)
        if err != nil {
            return nil, errors.New("class not found")
        }
    }
    student := &models.Student{
        ID:            uuid.New().String(),
        UserID:        req.UserID,
        SchoolID:      req.SchoolID,
        ClassID:       req.ClassID,
        AdmissionNo:   req.AdmissionNo,
        DateOfBirth:   req.DateOfBirth,
        Gender:        req.Gender,
        Address:       req.Address,
        GuardianName:  req.GuardianName,
        GuardianPhone: req.GuardianPhone,
        GuardianEmail: req.GuardianEmail,
        IsActive:      true,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }
    if err := s.studentRepo.Create(student); err != nil {
        return nil, err
    }
    return s.toDetailResponse(student, user), nil
}

func (s *StudentService) GetByID(id string) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("student not found")
    }
    user, err := s.userRepo.FindByID(student.UserID)
    if err != nil {
        return nil, err
    }
    return s.toDetailResponse(student, user), nil
}

func (s *StudentService) GetByUserID(userID string) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByUserID(userID)
    if err != nil {
        return nil, errors.New("student not found for this user")
    }
    user, err := s.userRepo.FindByID(student.UserID)
    if err != nil {
        return nil, err
    }
    return s.toDetailResponse(student, user), nil
}

func (s *StudentService) GetBySchool(schoolID string, page, limit int) ([]dto.StudentDetailResponse, int64, error) {
    students, total, err := s.studentRepo.FindBySchool(schoolID, page, limit)
    if err != nil {
        return nil, 0, err
    }
    var responses []dto.StudentDetailResponse
    for _, student := range students {
        user, _ := s.userRepo.FindByID(student.UserID)
        responses = append(responses, *s.toDetailResponse(&student, user))
    }
    return responses, total, nil
}

func (s *StudentService) GetByClass(classID string, page, limit int) ([]dto.StudentDetailResponse, int64, error) {
    students, total, err := s.studentRepo.FindByClass(classID, page, limit)
    if err != nil {
        return nil, 0, err
    }
    var responses []dto.StudentDetailResponse
    for _, student := range students {
        user, _ := s.userRepo.FindByID(student.UserID)
        responses = append(responses, *s.toDetailResponse(&student, user))
    }
    return responses, total, nil
}

func (s *StudentService) Update(id string, req *dto.UpdateStudentRequest) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("student not found")
    }
    if req.ClassID != nil {
        if *req.ClassID != "" {
            _, err := s.classRepo.FindByID(*req.ClassID)
            if err != nil {
                return nil, errors.New("class not found")
            }
        }
        student.ClassID = *req.ClassID
    }
    if req.DateOfBirth != nil {
        student.DateOfBirth = req.DateOfBirth
    }
    if req.Gender != "" {
        student.Gender = req.Gender
    }
    if req.Address != "" {
        student.Address = req.Address
    }
    if req.GuardianName != "" {
        student.GuardianName = req.GuardianName
    }
    if req.GuardianPhone != "" {
        student.GuardianPhone = req.GuardianPhone
    }
    if req.GuardianEmail != "" {
        student.GuardianEmail = req.GuardianEmail
    }
    if req.IsActive != nil {
        student.IsActive = *req.IsActive
    }
    student.UpdatedAt = time.Now()
    if err := s.studentRepo.Update(student); err != nil {
        return nil, err
    }
    user, _ := s.userRepo.FindByID(student.UserID)
    return s.toDetailResponse(student, user), nil
}

func (s *StudentService) Delete(id string) error {
    return s.studentRepo.Delete(id)
}

func (s *StudentService) TransferClass(studentID, newClassID string) error {
    _, err := s.studentRepo.FindByID(studentID)
    if err != nil {
        return errors.New("student not found")
    }
    _, err = s.classRepo.FindByID(newClassID)
    if err != nil {
        return errors.New("class not found")
    }
    return s.studentRepo.TransferClass(studentID, newClassID)
}

func (s *StudentService) GetStudentCountByClass(classID string) (int64, error) {
    return s.studentRepo.GetStudentCountByClass(classID)
}

func (s *StudentService) GenerateAdmissionNumber(schoolCode string) string {
    return fmt.Sprintf("%s/%d", schoolCode, time.Now().UnixNano())
}

// toDetailResponse handles the conversion, including the email pointer to string.
func (s *StudentService) toDetailResponse(student *models.Student, user *models.User) *dto.StudentDetailResponse {
    // Convert *string email to plain string (empty if nil)
    emailStr := ""
    if user.Email != nil {
        emailStr = *user.Email
    }
    return &dto.StudentDetailResponse{
        StudentResponse: dto.StudentResponse{
            ID:            student.ID,
            UserID:        student.UserID,
            SchoolID:      student.SchoolID,
            ClassID:       student.ClassID,
            AdmissionNo:   student.AdmissionNo,
            DateOfBirth:   student.DateOfBirth,
            Gender:        student.Gender,
            Address:       student.Address,
            GuardianName:  student.GuardianName,
            GuardianPhone: student.GuardianPhone,
            GuardianEmail: student.GuardianEmail,
            IsActive:      student.IsActive,
            CreatedAt:     student.CreatedAt,
            UpdatedAt:     student.UpdatedAt,
        },
        User: dto.UserBriefDTO{
            ID:          user.ID,
            Email:       emailStr,      // ✅ fixed: now a plain string
            FirstName:   user.FirstName,
            LastName:    user.LastName,
            PhoneNumber: user.PhoneNumber,
        },
    }
}



// package service

// import (
//     "errors"
//     "fmt"
//     "time"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
// )

// type StudentService struct {
//     studentRepo *repository.StudentRepository
//     userRepo    *repository.UserRepository
//     classRepo   *repository.ClassRepository
// }

// func NewStudentService(
//     studentRepo *repository.StudentRepository,
//     userRepo *repository.UserRepository,
//     classRepo *repository.ClassRepository,
// ) *StudentService {
//     return &StudentService{
//         studentRepo: studentRepo,
//         userRepo:    userRepo,
//         classRepo:   classRepo,
//     }
// }

// func (s *StudentService) Create(req *dto.CreateStudentRequest) (*dto.StudentDetailResponse, error) {
//     // Check if user exists
//     user, err := s.userRepo.FindByID(req.UserID)
//     if err != nil {
//         return nil, errors.New("user not found")
//     }
    
//     // Check if admission number already exists
//     existing, _ := s.studentRepo.FindByAdmissionNo(req.AdmissionNo)
//     if existing != nil {
//         return nil, errors.New("admission number already exists")
//     }
    
//     // Check if class exists (if provided)
//     if req.ClassID != "" {
//         _, err := s.classRepo.FindByID(req.ClassID)
//         if err != nil {
//             return nil, errors.New("class not found")
//         }
//     }
    
//     student := &models.Student{
//         ID:            uuid.New().String(),
//         UserID:        req.UserID,
//         SchoolID:      req.SchoolID,
//         ClassID:       req.ClassID,
//         AdmissionNo:   req.AdmissionNo,
//         DateOfBirth:   req.DateOfBirth,
//         Gender:        req.Gender,
//         Address:       req.Address,
//         GuardianName:  req.GuardianName,
//         GuardianPhone: req.GuardianPhone,
//         GuardianEmail: req.GuardianEmail,
//         IsActive:      true,
//         CreatedAt:     time.Now(),
//         UpdatedAt:     time.Now(),
//     }
    
//     if err := s.studentRepo.Create(student); err != nil {
//         return nil, err
//     }
    
//     return s.toDetailResponse(student, user), nil
// }

// func (s *StudentService) GetByID(id string) (*dto.StudentDetailResponse, error) {
//     student, err := s.studentRepo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("student not found")
//     }
    
//     user, err := s.userRepo.FindByID(student.UserID)
//     if err != nil {
//         return nil, err
//     }
    
//     return s.toDetailResponse(student, user), nil
// }

// func (s *StudentService) GetByUserID(userID string) (*dto.StudentDetailResponse, error) {
//     student, err := s.studentRepo.FindByUserID(userID)
//     if err != nil {
//         return nil, errors.New("student not found for this user")
//     }
    
//     user, err := s.userRepo.FindByID(student.UserID)
//     if err != nil {
//         return nil, err
//     }
    
//     return s.toDetailResponse(student, user), nil
// }

// func (s *StudentService) GetBySchool(schoolID string, page, limit int) ([]dto.StudentDetailResponse, int64, error) {
//     students, total, err := s.studentRepo.FindBySchool(schoolID, page, limit)
//     if err != nil {
//         return nil, 0, err
//     }
    
//     var responses []dto.StudentDetailResponse
//     for _, student := range students {
//         user, _ := s.userRepo.FindByID(student.UserID)
//         responses = append(responses, *s.toDetailResponse(&student, user))
//     }
    
//     return responses, total, nil
// }

// func (s *StudentService) GetByClass(classID string, page, limit int) ([]dto.StudentDetailResponse, int64, error) {
//     students, total, err := s.studentRepo.FindByClass(classID, page, limit)
//     if err != nil {
//         return nil, 0, err
//     }
    
//     var responses []dto.StudentDetailResponse
//     for _, student := range students {
//         user, _ := s.userRepo.FindByID(student.UserID)
//         responses = append(responses, *s.toDetailResponse(&student, user))
//     }
    
//     return responses, total, nil
// }

// func (s *StudentService) Update(id string, req *dto.UpdateStudentRequest) (*dto.StudentDetailResponse, error) {
//     student, err := s.studentRepo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("student not found")
//     }
    
//     if req.ClassID != nil {
//         // Verify new class exists
//         if *req.ClassID != "" {
//             _, err := s.classRepo.FindByID(*req.ClassID)
//             if err != nil {
//                 return nil, errors.New("class not found")
//             }
//         }
//         student.ClassID = *req.ClassID
//     }
//     if req.DateOfBirth != nil {
//         student.DateOfBirth = req.DateOfBirth
//     }
//     if req.Gender != "" {
//         student.Gender = req.Gender
//     }
//     if req.Address != "" {
//         student.Address = req.Address
//     }
//     if req.GuardianName != "" {
//         student.GuardianName = req.GuardianName
//     }
//     if req.GuardianPhone != "" {
//         student.GuardianPhone = req.GuardianPhone
//     }
//     if req.GuardianEmail != "" {
//         student.GuardianEmail = req.GuardianEmail
//     }
//     if req.IsActive != nil {
//         student.IsActive = *req.IsActive
//     }
//     student.UpdatedAt = time.Now()
    
//     if err := s.studentRepo.Update(student); err != nil {
//         return nil, err
//     }
    
//     user, _ := s.userRepo.FindByID(student.UserID)
//     return s.toDetailResponse(student, user), nil
// }

// func (s *StudentService) Delete(id string) error {
//     return s.studentRepo.Delete(id)
// }

// func (s *StudentService) TransferClass(studentID, newClassID string) error {
//     // Verify student exists
//     _, err := s.studentRepo.FindByID(studentID)
//     if err != nil {
//         return errors.New("student not found")
//     }
    
//     // Verify new class exists
//     _, err = s.classRepo.FindByID(newClassID)
//     if err != nil {
//         return errors.New("class not found")
//     }
    
//     return s.studentRepo.TransferClass(studentID, newClassID)
// }

// func (s *StudentService) GetStudentCountByClass(classID string) (int64, error) {
//     return s.studentRepo.GetStudentCountByClass(classID)
// }

// func (s *StudentService) GenerateAdmissionNumber(schoolCode string) string {
//     return fmt.Sprintf("%s/%d", schoolCode, time.Now().UnixNano())
// }

// func (s *StudentService) toDetailResponse(student *models.Student, user *models.User) *dto.StudentDetailResponse {
//     return &dto.StudentDetailResponse{
//         StudentResponse: dto.StudentResponse{
//             ID:            student.ID,
//             UserID:        student.UserID,
//             SchoolID:      student.SchoolID,
//             ClassID:       student.ClassID,
//             AdmissionNo:   student.AdmissionNo,
//             DateOfBirth:   student.DateOfBirth,
//             Gender:        student.Gender,
//             Address:       student.Address,
//             GuardianName:  student.GuardianName,
//             GuardianPhone: student.GuardianPhone,
//             GuardianEmail: student.GuardianEmail,
//             IsActive:      student.IsActive,
//             CreatedAt:     student.CreatedAt,
//             UpdatedAt:     student.UpdatedAt,
//         },
//         User: dto.UserBriefDTO{
//             ID:          user.ID,
//             Email:       user.Email,
//             FirstName:   user.FirstName,
//             LastName:    user.LastName,
//             PhoneNumber: user.PhoneNumber,
//         },
//     }
// }