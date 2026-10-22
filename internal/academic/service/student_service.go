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

func (s *StudentService) Create(req *dto.CreateStudentRequest, schoolID string, isPlatformOwner bool) (*dto.StudentDetailResponse, error) {
    effectiveID := effectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
    if effectiveID == "" {
        return nil, ErrSchoolScopeRequired
    }

    user, err := s.userRepo.FindByID(req.UserID)
    if err != nil {
        return nil, errors.New("user not found")
    }
    existing, _ := s.studentRepo.FindByAdmissionNo(req.AdmissionNo)
    if existing != nil {
        return nil, errors.New("admission number already exists")
    }
    if req.ClassID != "" {
        class, err := s.classRepo.FindByID(req.ClassID)
        if err != nil {
            return nil, errors.New("class not found")
        }
        if class.SchoolID != effectiveID {
            return nil, errors.New("class not found")
        }
    }
    student := &models.Student{
        ID:            uuid.New().String(),
        UserID:        req.UserID,
        SchoolID:      effectiveID,
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

func (s *StudentService) GetByID(id, schoolID string, isPlatformOwner bool) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("student not found")
    }
    if err := authorizeSchoolOwnership(student.SchoolID, schoolID, isPlatformOwner); err != nil {
        return nil, errors.New("student not found")
    }
    user, err := s.userRepo.FindByID(student.UserID)
    if err != nil {
        return nil, err
    }
    return s.toDetailResponse(student, user), nil
}

// GetByUserID looks up the Student record for userID - reachable both
// as a student's own profile lookup (StudentHandler.GetStudentProfile,
// which passes the caller's own user_id from the verified JWT) AND as
// GET /user-student/:userId, which takes an arbitrary client-supplied
// userId. Without the tenant check, the latter would let any
// authenticated user enumerate another school's student records -
// including guardian contact info and address, direct PII - by guessing
// a user ID. The self-lookup case is unaffected: a student's own school
// is always their own record's school, so the check passes naturally.
func (s *StudentService) GetByUserID(userID, schoolID string, isPlatformOwner bool) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByUserID(userID)
    if err != nil {
        return nil, errors.New("student not found for this user")
    }
    if err := authorizeSchoolOwnership(student.SchoolID, schoolID, isPlatformOwner); err != nil {
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

// GetByClass returns the roster for classID, but only once the class
// itself is confirmed to belong to schoolID - without this check, any
// authenticated user who knew or enumerated another school's class ID
// could read that school's students (names, guardian contact info,
// addresses - direct PII), even though the URL names no school at all.
func (s *StudentService) GetByClass(classID, schoolID string, isPlatformOwner bool, page, limit int) ([]dto.StudentDetailResponse, int64, error) {
    class, err := s.classRepo.FindByID(classID)
    if err != nil {
        return nil, 0, errors.New("class not found")
    }
    if err := authorizeSchoolOwnership(class.SchoolID, schoolID, isPlatformOwner); err != nil {
        return nil, 0, errors.New("class not found")
    }

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

func (s *StudentService) Update(id string, req *dto.UpdateStudentRequest, schoolID string, isPlatformOwner bool) (*dto.StudentDetailResponse, error) {
    student, err := s.studentRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("student not found")
    }
    if err := authorizeSchoolOwnership(student.SchoolID, schoolID, isPlatformOwner); err != nil {
        return nil, errors.New("student not found")
    }
    if req.ClassID != nil {
        if *req.ClassID != "" {
            class, err := s.classRepo.FindByID(*req.ClassID)
            if err != nil {
                return nil, errors.New("class not found")
            }
            if class.SchoolID != student.SchoolID {
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

func (s *StudentService) Delete(id, schoolID string, isPlatformOwner bool) error {
    student, err := s.studentRepo.FindByID(id)
    if err != nil {
        return errors.New("student not found")
    }
    if err := authorizeSchoolOwnership(student.SchoolID, schoolID, isPlatformOwner); err != nil {
        return errors.New("student not found")
    }
    return s.studentRepo.Delete(id)
}

// TransferClass moves studentID to newClassID, requiring both to belong
// to schoolID - without the second check, a caller could transfer their
// own school's student into another school's class, and without the
// first, could transfer another school's student at all.
func (s *StudentService) TransferClass(studentID, newClassID, schoolID string, isPlatformOwner bool) error {
    student, err := s.studentRepo.FindByID(studentID)
    if err != nil {
        return errors.New("student not found")
    }
    if err := authorizeSchoolOwnership(student.SchoolID, schoolID, isPlatformOwner); err != nil {
        return errors.New("student not found")
    }
    class, err := s.classRepo.FindByID(newClassID)
    if err != nil {
        return errors.New("class not found")
    }
    if class.SchoolID != student.SchoolID {
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