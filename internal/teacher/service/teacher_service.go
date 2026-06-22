package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"cbt-api/internal/academic/repository"
	"cbt-api/internal/models"
	"cbt-api/internal/teacher/dto"
	"cbt-api/pkg/excel"
	"cbt-api/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TeacherService struct {
	userRepo    *repository.UserRepository
	studentRepo *repository.StudentRepository
	classRepo   *repository.ClassRepository
	schoolRepo  *repository.SchoolRepository
	// sessionRepo temporarily removed – graduation year will use simple calculation
	db     *gorm.DB
	logger *zap.Logger
}

// NewTeacherService – sessionRepo is no longer required
func NewTeacherService(
	userRepo *repository.UserRepository,
	studentRepo *repository.StudentRepository,
	classRepo *repository.ClassRepository,
	schoolRepo *repository.SchoolRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *TeacherService {
	return &TeacherService{
		userRepo:    userRepo,
		studentRepo: studentRepo,
		classRepo:   classRepo,
		schoolRepo:  schoolRepo,
		db:          db,
		logger:      logger,
	}
}

func generateRandomPassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	length := 8
	password := make([]byte, length)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}
	return string(password), nil
}

// generateAdmissionNo creates a unique admission number (ADM20250001, ...)
func (s *TeacherService) generateAdmissionNo(schoolID string) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("ADM%d", year)

	var lastSeq int
	var lastAdmission string
	err := s.db.Model(&models.Student{}).
		Where("admission_no LIKE ?", prefix+"%").
		Order("admission_no DESC").
		Limit(1).
		Pluck("admission_no", &lastAdmission).Error
	if err == nil && lastAdmission != "" {
		fmt.Sscanf(lastAdmission, prefix+"%d", &lastSeq)
	}
	nextSeq := lastSeq + 1
	return fmt.Sprintf("%s%04d", prefix, nextSeq), nil
}

// calculateExpectedGraduationYear – simplified fallback (current year + 3)
func (s *TeacherService) calculateExpectedGraduationYear(class *models.Class, schoolID string) int {
	// For now, just add 3 years to current year
	// Later you can enhance by fetching class level duration and session start year
	return time.Now().Year() + 3
}

// getClassWithTeacher fetches class, its level, arm, and teacher name (no class.Name)
func (s *TeacherService) getClassWithTeacher(classID string) (classInfo struct {
	ID          string
	ClassName   string // using ClassLevel + ClassArm to build a name
	ClassLevel  string
	ClassArm    string
	TeacherName string
}, err error) {
	var class models.Class
	err = s.db.Preload("ClassLevel").Preload("ClassArm").Preload("Teacher").First(&class, "id = ?", classID).Error
	if err != nil {
		return
	}
	classInfo.ID = class.ID
	// Build a class name from level and arm (e.g., "SS3 A")
	if class.ClassLevel != nil {
		classInfo.ClassLevel = class.ClassLevel.Name
		classInfo.ClassName = class.ClassLevel.Name
	}
	if class.ClassArm != nil {
		classInfo.ClassArm = class.ClassArm.Name
		classInfo.ClassName = classInfo.ClassLevel + " " + class.ClassArm.Name
	}
	if class.Teacher != nil {
		classInfo.TeacherName = class.Teacher.FirstName + " " + class.Teacher.LastName
	} else if class.TeacherID != nil {
		if user, err := s.userRepo.FindByID(*class.TeacherID); err == nil {
			classInfo.TeacherName = user.FirstName + " " + user.LastName
		}
	}
	return
}

func (s *TeacherService) getSchoolName(schoolID string) string {
	school, err := s.schoolRepo.FindByID(schoolID)
	if err != nil {
		return ""
	}
	return school.Name
}

func (s *TeacherService) getTermsSpent(studentID string) int {
	// Placeholder – implement when StudentTerm model exists
	return 0
}

// createStudentInternal – core logic (returns CompleteStudentResponse)
func (s *TeacherService) createStudentInternal(
	userRepo *repository.UserRepository,
	studentRepo *repository.StudentRepository,
	req *dto.CreateStudentByTeacherRequest,
	teacherID string,
) (*dto.CompleteStudentResponse, error) {
	class, err := s.classRepo.FindByID(req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("class not found: %w", err)
	}
	if class.TeacherID == nil || *class.TeacherID != teacherID {
		return nil, errors.New("you are not authorized to add students to this class")
	}

	admissionNo := req.AdmissionNo
	if admissionNo == "" {
		admissionNo, err = s.generateAdmissionNo(req.SchoolID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate admission number: %w", err)
		}
	} else {
		existing, _ := studentRepo.FindByAdmissionNo(admissionNo)
		if existing != nil {
			return nil, errors.New("admission number already exists")
		}
	}

	username := req.Username
	if username == "" {
		username = admissionNo
		for {
			_, err := userRepo.FindByUsername(username)
			if err != nil {
				break
			}
			username = fmt.Sprintf("%s%d", admissionNo, time.Now().UnixNano())
		}
	} else {
		if _, err := userRepo.FindByUsername(username); err == nil {
			return nil, errors.New("username already taken")
		}
	}

	rawPassword, err := generateRandomPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}
	hashedPassword, err := utils.HashPassword(rawPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	var emailPtr *string
	if req.GuardianEmail != "" {
		emailPtr = &req.GuardianEmail
	}
	user := &models.User{
		ID:            uuid.New().String(),
		Username:      username,
		Email:         emailPtr,
		Password:      hashedPassword,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		PhoneNumber:   req.GuardianPhone,
		Role:          models.RoleStudent,
		Status:        models.StatusActive,
		IsActive:      true,
		EmailVerified: false,
	}
	if err := userRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	student := &models.Student{
		ID:            uuid.New().String(),
		UserID:        user.ID,
		SchoolID:      req.SchoolID,
		ClassID:       req.ClassID,
		AdmissionNo:   admissionNo,
		DateOfBirth:   req.DateOfBirth,
		Gender:        req.Gender,
		Address:       req.Address,
		GuardianName:  req.GuardianName,
		GuardianPhone: req.GuardianPhone,
		GuardianEmail: req.GuardianEmail,
		IsActive:      true,
		Status:        "active",
	}
	if err := studentRepo.Create(student); err != nil {
		_ = userRepo.SoftDeleteUser(user.ID)
		return nil, fmt.Errorf("failed to create student profile: %w", err)
	}

	classInfo, _ := s.getClassWithTeacher(req.ClassID)
	schoolName := s.getSchoolName(req.SchoolID)
	expectedGradYear := s.calculateExpectedGraduationYear(class, req.SchoolID)
	termsSpent := s.getTermsSpent(student.ID)

	return &dto.CompleteStudentResponse{
		StudentID:         student.ID,
		UserID:            user.ID,
		AdmissionNo:       student.AdmissionNo,
		Username:          user.Username,
		GeneratedPassword: rawPassword,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		DateOfBirth:       student.DateOfBirth,
		Gender:            student.Gender,
		Address:           student.Address,
		GuardianName:      student.GuardianName,
		GuardianPhone:     student.GuardianPhone,
		GuardianEmail:     student.GuardianEmail,
		Class: struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			ClassLevel  string `json:"class_level"`
			ClassArm    string `json:"class_arm"`
			TeacherName string `json:"teacher_name"`
		}{
			ID:          classInfo.ID,
			Name:        classInfo.ClassName,
			ClassLevel:  classInfo.ClassLevel,
			ClassArm:    classInfo.ClassArm,
			TeacherName: classInfo.TeacherName,
		},
		SchoolID:               student.SchoolID,
		SchoolName:             schoolName,
		ExpectedGraduationYear: expectedGradYear,
		TermsSpentSoFar:        termsSpent,
		IsActive:               student.IsActive,
		Status:                 student.Status,
		CreatedAt:              student.CreatedAt,
	}, nil
}

// CreateStudent – public method
func (s *TeacherService) CreateStudent(req *dto.CreateStudentByTeacherRequest, teacherID string) (*dto.CompleteStudentResponse, error) {
	return s.createStudentInternal(s.userRepo, s.studentRepo, req, teacherID)
}

// GetMyStudents – unchanged
func (s *TeacherService) GetMyStudents(teacherID string, page, limit int) ([]dto.StudentListResponse, int64, error) {
	classes, err := s.classRepo.FindByTeacher(teacherID)
	if err != nil || len(classes) == 0 {
		return nil, 0, errors.New("no class assigned to you")
	}
	classID := classes[0].ID

	students, total, err := s.studentRepo.FindByClass(classID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var result []dto.StudentListResponse
	for _, stu := range students {
		user, err := s.userRepo.FindByID(stu.UserID)
		if err != nil {
			s.logger.Warn("user not found for student", zap.String("student_id", stu.ID), zap.Error(err))
			continue
		}
		result = append(result, dto.StudentListResponse{
			ID:          stu.ID,
			AdmissionNo: stu.AdmissionNo,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Username:    user.Username,
			Gender:      stu.Gender,
			IsActive:    stu.IsActive,
			Status:      stu.Status,
			CreatedAt:   stu.CreatedAt,
		})
	}
	return result, total, nil
}

// GetStudentByID – unchanged
func (s *TeacherService) GetStudentByID(studentID, teacherID string) (*dto.StudentListResponse, error) {
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	class, err := s.classRepo.FindByID(student.ClassID)
	if err != nil {
		return nil, errors.New("class not found")
	}
	if class.TeacherID == nil || *class.TeacherID != teacherID {
		return nil, errors.New("unauthorized to view this student")
	}
	user, err := s.userRepo.FindByID(student.UserID)
	if err != nil {
		return nil, err
	}
	return &dto.StudentListResponse{
		ID:          student.ID,
		AdmissionNo: student.AdmissionNo,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Username:    user.Username,
		Gender:      student.Gender,
		IsActive:    student.IsActive,
		Status:      student.Status,
		CreatedAt:   student.CreatedAt,
	}, nil
}

// UpdateStudent – unchanged
func (s *TeacherService) UpdateStudent(studentID string, req *dto.UpdateStudentByTeacherRequest, teacherID string) error {
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return errors.New("student not found")
	}
	class, err := s.classRepo.FindByID(student.ClassID)
	if err != nil {
		return errors.New("class not found")
	}
	if class.TeacherID == nil || *class.TeacherID != teacherID {
		return errors.New("unauthorized to update this student")
	}
	user, err := s.userRepo.FindByID(student.UserID)
	if err != nil {
		return err
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.GuardianPhone != nil {
		user.PhoneNumber = *req.GuardianPhone
	}
	if err := s.userRepo.UpdateUser(user); err != nil {
		return err
	}
	if req.DateOfBirth != nil {
		student.DateOfBirth = req.DateOfBirth
	}
	if req.Gender != nil {
		student.Gender = *req.Gender
	}
	if req.Address != nil {
		student.Address = *req.Address
	}
	if req.GuardianName != nil {
		student.GuardianName = *req.GuardianName
	}
	if req.GuardianEmail != nil {
		student.GuardianEmail = *req.GuardianEmail
	}
	student.UpdatedAt = time.Now()
	return s.studentRepo.Update(student)
}

// ResetStudentPassword – unchanged
func (s *TeacherService) ResetStudentPassword(studentID, teacherID string) (*dto.ResetStudentPasswordResponse, error) {
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	class, err := s.classRepo.FindByID(student.ClassID)
	if err != nil {
		return nil, errors.New("class not found")
	}
	if class.TeacherID == nil || *class.TeacherID != teacherID {
		return nil, errors.New("unauthorized to reset password for this student")
	}
	user, err := s.userRepo.FindByID(student.UserID)
	if err != nil {
		return nil, err
	}
	newPassword, err := generateRandomPassword()
	if err != nil {
		return nil, errors.New("failed to generate new password")
	}
	hashed, err := utils.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	user.Password = hashed
	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, err
	}
	return &dto.ResetStudentPasswordResponse{
		Username: user.Username,
		Password: newPassword,
	}, nil
}

// DeactivateStudent – unchanged
func (s *TeacherService) DeactivateStudent(studentID, teacherID string, reason string) error {
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return errors.New("student not found")
	}
	class, err := s.classRepo.FindByID(student.ClassID)
	if err != nil {
		return errors.New("class not found")
	}
	if class.TeacherID == nil || *class.TeacherID != teacherID {
		return errors.New("unauthorized to deactivate this student")
	}
	student.IsActive = false
	student.Status = reason
	user, err := s.userRepo.FindByID(student.UserID)
	if err == nil {
		user.IsActive = false
		_ = s.userRepo.UpdateUser(user)
	}
	return s.studentRepo.Update(student)
}

// BulkCreateStudents – now returns []dto.CompleteStudentResponse
func (s *TeacherService) BulkCreateStudents(fileReader io.Reader, teacherID string) ([]dto.CompleteStudentResponse, []string, error) {
	classes, err := s.classRepo.FindByTeacher(teacherID)
	if err != nil || len(classes) == 0 {
		return nil, nil, errors.New("no class assigned to you")
	}
	classID := classes[0].ID
	schoolID := classes[0].SchoolID

	rows, err := excel.ReadStudentsFromExcel(fileReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse Excel: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil, errors.New("no valid student rows found")
	}

	var created []dto.CompleteStudentResponse
	var errorsList []string

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := repository.NewUserRepository(tx)
		txStudentRepo := repository.NewStudentRepository(tx)

		for _, row := range rows {
			req := &dto.CreateStudentByTeacherRequest{
				SchoolID:      schoolID,
				ClassID:       classID,
				FirstName:     row.FirstName,
				LastName:      row.LastName,
				Username:      row.Username,
				AdmissionNo:   row.AdmissionNo,
				DateOfBirth:   row.DateOfBirth,
				Gender:        row.Gender,
				Address:       row.Address,
				GuardianName:  row.GuardianName,
				GuardianPhone: row.GuardianPhone,
				GuardianEmail: row.GuardianEmail,
			}
			resp, err := s.createStudentInternal(txUserRepo, txStudentRepo, req, teacherID)
			if err != nil {
				errMsg := fmt.Sprintf("Row %d: %v", row.RowNumber, err)
				errorsList = append(errorsList, errMsg)
				s.logger.Warn("bulk student creation failed", zap.Int("row", row.RowNumber), zap.Error(err))
				continue
			}
			created = append(created, *resp)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return created, errorsList, nil
}


// package service

// import (
// 	"crypto/rand"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"math/big"
// 	"time"

// 	"cbt-api/internal/academic/repository"
// 	"cbt-api/internal/models"
// 	"cbt-api/internal/teacher/dto"
// 	"cbt-api/pkg/excel"
// 	"cbt-api/pkg/utils"
// 	"github.com/google/uuid"
// 	"go.uber.org/zap"
// 	"gorm.io/gorm"
// )

// type TeacherService struct {
// 	userRepo    *repository.UserRepository
// 	studentRepo *repository.StudentRepository
// 	classRepo   *repository.ClassRepository
// 	schoolRepo  *repository.SchoolRepository
// 	sessionRepo *repository.SessionRepository // NEW – for graduation year
// 	db          *gorm.DB
// 	logger      *zap.Logger
// }

// // Updated constructor – now accepts sessionRepo
// func NewTeacherService(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	classRepo *repository.ClassRepository,
// 	schoolRepo *repository.SchoolRepository,
// 	sessionRepo *repository.SessionRepository,
// 	db *gorm.DB,
// 	logger *zap.Logger,
// ) *TeacherService {
// 	return &TeacherService{
// 		userRepo:    userRepo,
// 		studentRepo: studentRepo,
// 		classRepo:   classRepo,
// 		schoolRepo:  schoolRepo,
// 		sessionRepo: sessionRepo,
// 		db:          db,
// 		logger:      logger,
// 	}
// }

// // generateRandomPassword creates a secure random password (8 chars)
// func generateRandomPassword() (string, error) {
// 	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
// 	length := 8
// 	password := make([]byte, length)
// 	for i := range password {
// 		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
// 		if err != nil {
// 			return "", err
// 		}
// 		password[i] = charset[num.Int64()]
// 	}
// 	return string(password), nil
// }

// // generateAdmissionNo creates a unique admission number.
// // Format: ADM<year><4-digit sequence> e.g., ADM20250001
// func (s *TeacherService) generateAdmissionNo(schoolID string) (string, error) {
// 	year := time.Now().Year()
// 	prefix := fmt.Sprintf("ADM%d", year)

// 	var lastSeq int
// 	var lastAdmission string
// 	err := s.db.Model(&models.Student{}).
// 		Where("admission_no LIKE ?", prefix+"%").
// 		Order("admission_no DESC").
// 		Limit(1).
// 		Pluck("admission_no", &lastAdmission).Error
// 	if err == nil && lastAdmission != "" {
// 		fmt.Sscanf(lastAdmission, prefix+"%d", &lastSeq)
// 	}
// 	nextSeq := lastSeq + 1
// 	return fmt.Sprintf("%s%04d", prefix, nextSeq), nil
// }

// // calculateExpectedGraduationYear uses class duration and current session start year.
// func (s *TeacherService) calculateExpectedGraduationYear(class *models.Class, schoolID string) int {
// 	currentSession, err := s.sessionRepo.FindCurrentSessionBySchool(schoolID)
// 	if err != nil {
// 		return time.Now().Year() + 3 // fallback
// 	}
// 	duration := 1
// 	if class.ClassLevel != nil && class.ClassLevel.DurationYears > 0 {
// 		duration = class.ClassLevel.DurationYears
// 	}
// 	return currentSession.StartYear + duration
// }

// // getClassWithTeacher fetches class, its level, arm, and teacher name.
// func (s *TeacherService) getClassWithTeacher(classID string) (classInfo struct {
// 	ID          string
// 	Name        string
// 	ClassLevel  string
// 	ClassArm    string
// 	TeacherName string
// }, err error) {
// 	var class models.Class
// 	err = s.db.Preload("ClassLevel").Preload("ClassArm").Preload("Teacher").First(&class, "id = ?", classID).Error
// 	if err != nil {
// 		return
// 	}
// 	classInfo.ID = class.ID
// 	classInfo.Name = class.Name
// 	if class.ClassLevel != nil {
// 		classInfo.ClassLevel = class.ClassLevel.Name
// 	}
// 	if class.ClassArm != nil {
// 		classInfo.ClassArm = class.ClassArm.Name
// 	}
// 	if class.Teacher != nil {
// 		classInfo.TeacherName = class.Teacher.FirstName + " " + class.Teacher.LastName
// 	} else if class.TeacherID != nil {
// 		if user, err := s.userRepo.FindByID(*class.TeacherID); err == nil {
// 			classInfo.TeacherName = user.FirstName + " " + user.LastName
// 		}
// 	}
// 	return
// }

// // getSchoolName returns the school name by ID.
// func (s *TeacherService) getSchoolName(schoolID string) string {
// 	school, err := s.schoolRepo.FindByID(schoolID)
// 	if err != nil {
// 		return ""
// 	}
// 	return school.Name
// }

// // getTermsSpent returns the number of terms the student has been enrolled in (placeholder)
// func (s *TeacherService) getTermsSpent(studentID string) int {
// 	// Implement when StudentTerm model exists
// 	return 0
// }

// // createStudentInternal – core logic (now returns CompleteStudentResponse)
// func (s *TeacherService) createStudentInternal(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	req *dto.CreateStudentByTeacherRequest,
// 	teacherID string,
// ) (*dto.CompleteStudentResponse, error) {
// 	// 1. Verify teacher owns the class
// 	class, err := s.classRepo.FindByID(req.ClassID)
// 	if err != nil {
// 		return nil, fmt.Errorf("class not found: %w", err)
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("you are not authorized to add students to this class")
// 	}

// 	// 2. Admission number: use provided or auto‑generate
// 	admissionNo := req.AdmissionNo
// 	if admissionNo == "" {
// 		admissionNo, err = s.generateAdmissionNo(req.SchoolID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to generate admission number: %w", err)
// 		}
// 	} else {
// 		existing, _ := studentRepo.FindByAdmissionNo(admissionNo)
// 		if existing != nil {
// 			return nil, errors.New("admission number already exists")
// 		}
// 	}

// 	// 3. Username: use provided or fallback to admission number
// 	username := req.Username
// 	if username == "" {
// 		username = admissionNo
// 		for {
// 			_, err := userRepo.FindByUsername(username)
// 			if err != nil {
// 				break
// 			}
// 			username = fmt.Sprintf("%s%d", admissionNo, time.Now().UnixNano())
// 		}
// 	} else {
// 		if _, err := userRepo.FindByUsername(username); err == nil {
// 			return nil, errors.New("username already taken")
// 		}
// 	}

// 	// 4. Generate random password
// 	rawPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate password: %w", err)
// 	}
// 	hashedPassword, err := utils.HashPassword(rawPassword)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to hash password: %w", err)
// 	}

// 	// 5. Create user (student role)
// 	var emailPtr *string
// 	if req.GuardianEmail != "" {
// 		emailPtr = &req.GuardianEmail
// 	}
// 	user := &models.User{
// 		ID:            uuid.New().String(),
// 		Username:      username,
// 		Email:         emailPtr,
// 		Password:      hashedPassword,
// 		FirstName:     req.FirstName,
// 		LastName:      req.LastName,
// 		PhoneNumber:   req.GuardianPhone,
// 		Role:          models.RoleStudent,
// 		Status:        models.StatusActive,
// 		IsActive:      true,
// 		EmailVerified: false,
// 	}
// 	if err := userRepo.CreateUser(user); err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	// 6. Create student profile
// 	student := &models.Student{
// 		ID:            uuid.New().String(),
// 		UserID:        user.ID,
// 		SchoolID:      req.SchoolID,
// 		ClassID:       req.ClassID,
// 		AdmissionNo:   admissionNo,
// 		DateOfBirth:   req.DateOfBirth,
// 		Gender:        req.Gender,
// 		Address:       req.Address,
// 		GuardianName:  req.GuardianName,
// 		GuardianPhone: req.GuardianPhone,
// 		GuardianEmail: req.GuardianEmail,
// 		IsActive:      true,
// 		Status:        "active",
// 	}
// 	if err := studentRepo.Create(student); err != nil {
// 		_ = userRepo.SoftDeleteUser(user.ID)
// 		return nil, fmt.Errorf("failed to create student profile: %w", err)
// 	}

// 	// 7. Build rich response
// 	classInfo, _ := s.getClassWithTeacher(req.ClassID)
// 	schoolName := s.getSchoolName(req.SchoolID)
// 	expectedGradYear := s.calculateExpectedGraduationYear(class, req.SchoolID)
// 	termsSpent := s.getTermsSpent(student.ID)

// 	return &dto.CompleteStudentResponse{
// 		StudentID:         student.ID,
// 		UserID:            user.ID,
// 		AdmissionNo:       student.AdmissionNo,
// 		Username:          user.Username,
// 		GeneratedPassword: rawPassword,
// 		FirstName:         user.FirstName,
// 		LastName:          user.LastName,
// 		DateOfBirth:       student.DateOfBirth,
// 		Gender:            student.Gender,
// 		Address:           student.Address,
// 		GuardianName:      student.GuardianName,
// 		GuardianPhone:     student.GuardianPhone,
// 		GuardianEmail:     student.GuardianEmail,
// 		Class: struct {
// 			ID          string `json:"id"`
// 			Name        string `json:"name"`
// 			ClassLevel  string `json:"class_level"`
// 			ClassArm    string `json:"class_arm"`
// 			TeacherName string `json:"teacher_name"`
// 		}{
// 			ID:          classInfo.ID,
// 			Name:        classInfo.Name,
// 			ClassLevel:  classInfo.ClassLevel,
// 			ClassArm:    classInfo.ClassArm,
// 			TeacherName: classInfo.TeacherName,
// 		},
// 		SchoolID:               student.SchoolID,
// 		SchoolName:             schoolName,
// 		ExpectedGraduationYear: expectedGradYear,
// 		TermsSpentSoFar:        termsSpent,
// 		IsActive:               student.IsActive,
// 		Status:                 student.Status,
// 		CreatedAt:              student.CreatedAt,
// 	}, nil
// }

// // CreateStudent – public method (now returns CompleteStudentResponse)
// func (s *TeacherService) CreateStudent(req *dto.CreateStudentByTeacherRequest, teacherID string) (*dto.CompleteStudentResponse, error) {
// 	return s.createStudentInternal(s.userRepo, s.studentRepo, req, teacherID)
// }

// // GetMyStudents (unchanged)
// func (s *TeacherService) GetMyStudents(teacherID string, page, limit int) ([]dto.StudentListResponse, int64, error) {
// 	classes, err := s.classRepo.FindByTeacher(teacherID)
// 	if err != nil || len(classes) == 0 {
// 		return nil, 0, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID

// 	students, total, err := s.studentRepo.FindByClass(classID, page, limit)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	var result []dto.StudentListResponse
// 	for _, stu := range students {
// 		user, err := s.userRepo.FindByID(stu.UserID)
// 		if err != nil {
// 			s.logger.Warn("user not found for student", zap.String("student_id", stu.ID), zap.Error(err))
// 			continue
// 		}
// 		result = append(result, dto.StudentListResponse{
// 			ID:          stu.ID,
// 			AdmissionNo: stu.AdmissionNo,
// 			FirstName:   user.FirstName,
// 			LastName:    user.LastName,
// 			Username:    user.Username,
// 			Gender:      stu.Gender,
// 			IsActive:    stu.IsActive,
// 			Status:      stu.Status,
// 			CreatedAt:   stu.CreatedAt,
// 		})
// 	}
// 	return result, total, nil
// }

// // GetStudentByID (unchanged)
// func (s *TeacherService) GetStudentByID(studentID, teacherID string) (*dto.StudentListResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to view this student")
// 	}
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &dto.StudentListResponse{
// 		ID:          student.ID,
// 		AdmissionNo: student.AdmissionNo,
// 		FirstName:   user.FirstName,
// 		LastName:    user.LastName,
// 		Username:    user.Username,
// 		Gender:      student.Gender,
// 		IsActive:    student.IsActive,
// 		Status:      student.Status,
// 		CreatedAt:   student.CreatedAt,
// 	}, nil
// }

// // UpdateStudent (unchanged)
// func (s *TeacherService) UpdateStudent(studentID string, req *dto.UpdateStudentByTeacherRequest, teacherID string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to update this student")
// 	}
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return err
// 	}
// 	if req.FirstName != nil {
// 		user.FirstName = *req.FirstName
// 	}
// 	if req.LastName != nil {
// 		user.LastName = *req.LastName
// 	}
// 	if req.GuardianPhone != nil {
// 		user.PhoneNumber = *req.GuardianPhone
// 	}
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return err
// 	}
// 	if req.DateOfBirth != nil {
// 		student.DateOfBirth = req.DateOfBirth
// 	}
// 	if req.Gender != nil {
// 		student.Gender = *req.Gender
// 	}
// 	if req.Address != nil {
// 		student.Address = *req.Address
// 	}
// 	if req.GuardianName != nil {
// 		student.GuardianName = *req.GuardianName
// 	}
// 	if req.GuardianEmail != nil {
// 		student.GuardianEmail = *req.GuardianEmail
// 	}
// 	student.UpdatedAt = time.Now()
// 	return s.studentRepo.Update(student)
// }

// // ResetStudentPassword (unchanged)
// func (s *TeacherService) ResetStudentPassword(studentID, teacherID string) (*dto.ResetStudentPasswordResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to reset password for this student")
// 	}
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	newPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, errors.New("failed to generate new password")
// 	}
// 	hashed, err := utils.HashPassword(newPassword)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Password = hashed
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return nil, err
// 	}
// 	return &dto.ResetStudentPasswordResponse{
// 		Username: user.Username,
// 		Password: newPassword,
// 	}, nil
// }

// // DeactivateStudent (unchanged)
// func (s *TeacherService) DeactivateStudent(studentID, teacherID string, reason string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to deactivate this student")
// 	}
// 	student.IsActive = false
// 	student.Status = reason
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err == nil {
// 		user.IsActive = false
// 		_ = s.userRepo.UpdateUser(user)
// 	}
// 	return s.studentRepo.Update(student)
// }

// // BulkCreateStudents – now returns []dto.CompleteStudentResponse
// func (s *TeacherService) BulkCreateStudents(fileReader io.Reader, teacherID string) ([]dto.CompleteStudentResponse, []string, error) {
// 	classes, err := s.classRepo.FindByTeacher(teacherID)
// 	if err != nil || len(classes) == 0 {
// 		return nil, nil, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID
// 	schoolID := classes[0].SchoolID

// 	rows, err := excel.ReadStudentsFromExcel(fileReader)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to parse Excel: %w", err)
// 	}
// 	if len(rows) == 0 {
// 		return nil, nil, errors.New("no valid student rows found")
// 	}

// 	var created []dto.CompleteStudentResponse
// 	var errorsList []string

// 	err = s.db.Transaction(func(tx *gorm.DB) error {
// 		txUserRepo := repository.NewUserRepository(tx)
// 		txStudentRepo := repository.NewStudentRepository(tx)

// 		for _, row := range rows {
// 			req := &dto.CreateStudentByTeacherRequest{
// 				SchoolID:      schoolID,
// 				ClassID:       classID,
// 				FirstName:     row.FirstName,
// 				LastName:      row.LastName,
// 				Username:      row.Username,
// 				AdmissionNo:   row.AdmissionNo,
// 				DateOfBirth:   row.DateOfBirth,
// 				Gender:        row.Gender,
// 				Address:       row.Address,
// 				GuardianName:  row.GuardianName,
// 				GuardianPhone: row.GuardianPhone,
// 				GuardianEmail: row.GuardianEmail,
// 			}
// 			resp, err := s.createStudentInternal(txUserRepo, txStudentRepo, req, teacherID)
// 			if err != nil {
// 				errMsg := fmt.Sprintf("Row %d: %v", row.RowNumber, err)
// 				errorsList = append(errorsList, errMsg)
// 				s.logger.Warn("bulk student creation failed", zap.Int("row", row.RowNumber), zap.Error(err))
// 				continue
// 			}
// 			created = append(created, *resp)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return created, errorsList, nil
// }


// package service

// import (
// 	"crypto/rand"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"math/big"
// 	"time"

// 	"cbt-api/internal/academic/repository"
// 	"cbt-api/internal/models"
// 	"cbt-api/internal/teacher/dto"
// 	"cbt-api/pkg/excel"
// 	"cbt-api/pkg/utils"
// 	"github.com/google/uuid"
// 	"go.uber.org/zap"
// 	"gorm.io/gorm"
// )

// type TeacherService struct {
// 	userRepo    *repository.UserRepository
// 	studentRepo *repository.StudentRepository
// 	classRepo   *repository.ClassRepository
// 	schoolRepo  *repository.SchoolRepository
// 	db          *gorm.DB
// 	logger      *zap.Logger
// }

// func NewTeacherService(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	classRepo *repository.ClassRepository,
// 	schoolRepo *repository.SchoolRepository,
// 	db *gorm.DB,
// 	logger *zap.Logger,
// ) *TeacherService {
// 	return &TeacherService{
// 		userRepo:    userRepo,
// 		studentRepo: studentRepo,
// 		classRepo:   classRepo,
// 		schoolRepo:  schoolRepo,
// 		db:          db,
// 		logger:      logger,
// 	}
// }

// func generateRandomPassword() (string, error) {
// 	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
// 	length := 8
// 	password := make([]byte, length)
// 	for i := range password {
// 		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
// 		if err != nil {
// 			return "", err
// 		}
// 		password[i] = charset[num.Int64()]
// 	}
// 	return string(password), nil
// }

// func (s *TeacherService) createStudentInternal(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	req *dto.CreateStudentByTeacherRequest,
// 	teacherID string,
// ) (*dto.CreateStudentByTeacherResponse, error) {
// 	class, err := s.classRepo.FindByID(req.ClassID)
// 	if err != nil {
// 		return nil, fmt.Errorf("class not found: %w", err)
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("you are not authorized to add students to this class")
// 	}

// 	existing, _ := studentRepo.FindByAdmissionNo(req.AdmissionNo)
// 	if existing != nil {
// 		return nil, errors.New("admission number already exists")
// 	}

// 	username := req.Username
// 	if username == "" {
// 		username = req.AdmissionNo
// 		for {
// 			_, err := userRepo.FindByUsername(username)
// 			if err != nil {
// 				break
// 			}
// 			username = fmt.Sprintf("%s%d", req.AdmissionNo, time.Now().UnixNano())
// 		}
// 	} else {
// 		_, err := userRepo.FindByUsername(username)
// 		if err == nil {
// 			return nil, errors.New("username already taken")
// 		}
// 	}

// 	rawPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate password: %w", err)
// 	}
// 	hashedPassword, err := utils.HashPassword(rawPassword)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to hash password: %w", err)
// 	}

// 	// Convert GuardianEmail to *string
// 	var emailPtr *string
// 	if req.GuardianEmail != "" {
// 		emailPtr = &req.GuardianEmail
// 	}
// 	user := &models.User{
// 		ID:            uuid.New().String(),
// 		Username:      username,
// 		Email:         emailPtr,
// 		Password:      hashedPassword,
// 		FirstName:     req.FirstName,
// 		LastName:      req.LastName,
// 		PhoneNumber:   req.GuardianPhone,
// 		Role:          models.RoleStudent,
// 		Status:        models.StatusActive,
// 		IsActive:      true,
// 		EmailVerified: false,
// 	}
// 	if err := userRepo.CreateUser(user); err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	student := &models.Student{
// 		ID:            uuid.New().String(),
// 		UserID:        user.ID,
// 		SchoolID:      req.SchoolID,
// 		ClassID:       req.ClassID,
// 		AdmissionNo:   req.AdmissionNo,
// 		DateOfBirth:   req.DateOfBirth,
// 		Gender:        req.Gender,
// 		Address:       req.Address,
// 		GuardianName:  req.GuardianName,
// 		GuardianPhone: req.GuardianPhone,
// 		GuardianEmail: req.GuardianEmail,
// 		IsActive:      true,
// 		Status:        "active",
// 	}
// 	if err := studentRepo.Create(student); err != nil {
// 		_ = userRepo.SoftDeleteUser(user.ID)
// 		return nil, fmt.Errorf("failed to create student profile: %w", err)
// 	}

// 	return &dto.CreateStudentByTeacherResponse{
// 		StudentID:         student.ID,
// 		UserID:            user.ID,
// 		Username:          user.Username,
// 		GeneratedPassword: rawPassword,
// 		AdmissionNo:       student.AdmissionNo,
// 		FirstName:         user.FirstName,
// 		LastName:          user.LastName,
// 	}, nil
// }

// func (s *TeacherService) CreateStudent(req *dto.CreateStudentByTeacherRequest, teacherID string) (*dto.CreateStudentByTeacherResponse, error) {
// 	return s.createStudentInternal(s.userRepo, s.studentRepo, req, teacherID)
// }

// func (s *TeacherService) GetMyStudents(teacherID string, page, limit int) ([]dto.StudentListResponse, int64, error) {
// 	classes, err := s.classRepo.FindByTeacher(teacherID) // ✅ changed from FindByTeacherID
// 	if err != nil || len(classes) == 0 {
// 		return nil, 0, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID

// 	students, total, err := s.studentRepo.FindByClass(classID, page, limit)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	var result []dto.StudentListResponse
// 	for _, stu := range students {
// 		user, err := s.userRepo.FindByID(stu.UserID)
// 		if err != nil {
// 			s.logger.Warn("user not found for student", zap.String("student_id", stu.ID), zap.Error(err))
// 			continue
// 		}
// 		result = append(result, dto.StudentListResponse{
// 			ID:          stu.ID,
// 			AdmissionNo: stu.AdmissionNo,
// 			FirstName:   user.FirstName,
// 			LastName:    user.LastName,
// 			Username:    user.Username,
// 			Gender:      stu.Gender,
// 			IsActive:    stu.IsActive,
// 			Status:      stu.Status,
// 			CreatedAt:   stu.CreatedAt,
// 		})
// 	}
// 	return result, total, nil
// }

// func (s *TeacherService) GetStudentByID(studentID, teacherID string) (*dto.StudentListResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to view this student")
// 	}

// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &dto.StudentListResponse{
// 		ID:          student.ID,
// 		AdmissionNo: student.AdmissionNo,
// 		FirstName:   user.FirstName,
// 		LastName:    user.LastName,
// 		Username:    user.Username,
// 		Gender:      student.Gender,
// 		IsActive:    student.IsActive,
// 		Status:      student.Status,
// 		CreatedAt:   student.CreatedAt,
// 	}, nil
// }

// func (s *TeacherService) UpdateStudent(studentID string, req *dto.UpdateStudentByTeacherRequest, teacherID string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to update this student")
// 	}

// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return err
// 	}
// 	if req.FirstName != nil {
// 		user.FirstName = *req.FirstName
// 	}
// 	if req.LastName != nil {
// 		user.LastName = *req.LastName
// 	}
// 	if req.GuardianPhone != nil {
// 		user.PhoneNumber = *req.GuardianPhone
// 	}
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return err
// 	}

// 	if req.DateOfBirth != nil {
// 		student.DateOfBirth = req.DateOfBirth
// 	}
// 	if req.Gender != nil {
// 		student.Gender = *req.Gender
// 	}
// 	if req.Address != nil {
// 		student.Address = *req.Address
// 	}
// 	if req.GuardianName != nil {
// 		student.GuardianName = *req.GuardianName
// 	}
// 	if req.GuardianEmail != nil {
// 		student.GuardianEmail = *req.GuardianEmail
// 	}
// 	student.UpdatedAt = time.Now()
// 	return s.studentRepo.Update(student)
// }

// func (s *TeacherService) ResetStudentPassword(studentID, teacherID string) (*dto.ResetStudentPasswordResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to reset password for this student")
// 	}

// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	newPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, errors.New("failed to generate new password")
// 	}
// 	hashed, err := utils.HashPassword(newPassword)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Password = hashed
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return nil, err
// 	}
// 	return &dto.ResetStudentPasswordResponse{
// 		Username: user.Username,
// 		Password: newPassword,
// 	}, nil
// }

// func (s *TeacherService) DeactivateStudent(studentID, teacherID string, reason string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to deactivate this student")
// 	}
// 	student.IsActive = false
// 	student.Status = reason
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err == nil {
// 		user.IsActive = false
// 		_ = s.userRepo.UpdateUser(user)
// 	}
// 	return s.studentRepo.Update(student)
// }

// func (s *TeacherService) BulkCreateStudents(fileReader io.Reader, teacherID string) ([]dto.CreateStudentByTeacherResponse, []string, error) {
// 	classes, err := s.classRepo.FindByTeacher(teacherID) // ✅ changed from FindByTeacherID
// 	if err != nil || len(classes) == 0 {
// 		return nil, nil, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID
// 	schoolID := classes[0].SchoolID

// 	rows, err := excel.ReadStudentsFromExcel(fileReader)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to parse Excel: %w", err)
// 	}
// 	if len(rows) == 0 {
// 		return nil, nil, errors.New("no valid student rows found")
// 	}

// 	var created []dto.CreateStudentByTeacherResponse
// 	var errorsList []string

// 	err = s.db.Transaction(func(tx *gorm.DB) error {
// 		txUserRepo := repository.NewUserRepository(tx)
// 		txStudentRepo := repository.NewStudentRepository(tx)

// 		for _, row := range rows {
// 			req := &dto.CreateStudentByTeacherRequest{
// 				SchoolID:      schoolID,
// 				ClassID:       classID,
// 				FirstName:     row.FirstName,
// 				LastName:      row.LastName,
// 				Username:      row.Username,
// 				AdmissionNo:   row.AdmissionNo,
// 				DateOfBirth:   row.DateOfBirth,
// 				Gender:        row.Gender,
// 				Address:       row.Address,
// 				GuardianName:  row.GuardianName,
// 				GuardianPhone: row.GuardianPhone,
// 				GuardianEmail: row.GuardianEmail,
// 			}
// 			resp, err := s.createStudentInternal(txUserRepo, txStudentRepo, req, teacherID)
// 			if err != nil {
// 				errMsg := fmt.Sprintf("Row %d: %v", row.RowNumber, err)
// 				errorsList = append(errorsList, errMsg)
// 				s.logger.Warn("bulk student creation failed for row", zap.Int("row", row.RowNumber), zap.Error(err))
// 				continue
// 			}
// 			created = append(created, *resp)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return created, errorsList, nil
// }



// package service

// import (
// 	"crypto/rand"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"math/big"
// 	"time"

// 	"cbt-api/internal/academic/repository"
// 	"cbt-api/internal/models"
// 	"cbt-api/internal/teacher/dto"
// 	"cbt-api/pkg/excel"
// 	"cbt-api/pkg/utils"
// 	"github.com/google/uuid"
// 	"go.uber.org/zap"
// 	"gorm.io/gorm"
// )

// type TeacherService struct {
// 	userRepo    *repository.UserRepository
// 	studentRepo *repository.StudentRepository
// 	classRepo   *repository.ClassRepository
// 	schoolRepo  *repository.SchoolRepository
// 	db          *gorm.DB
// 	logger      *zap.Logger
// }

// func NewTeacherService(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	classRepo *repository.ClassRepository,
// 	schoolRepo *repository.SchoolRepository,
// 	db *gorm.DB,
// 	logger *zap.Logger,
// ) *TeacherService {
// 	return &TeacherService{
// 		userRepo:    userRepo,
// 		studentRepo: studentRepo,
// 		classRepo:   classRepo,
// 		schoolRepo:  schoolRepo,
// 		db:          db,
// 		logger:      logger,
// 	}
// }

// // generateRandomPassword creates a secure random password of length 8
// func generateRandomPassword() (string, error) {
// 	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
// 	length := 8
// 	password := make([]byte, length)
// 	for i := range password {
// 		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
// 		if err != nil {
// 			return "", err
// 		}
// 		password[i] = charset[num.Int64()]
// 	}
// 	return string(password), nil
// }

// // createStudentInternal is the core logic used by both single and bulk creation.
// func (s *TeacherService) createStudentInternal(
// 	userRepo *repository.UserRepository,
// 	studentRepo *repository.StudentRepository,
// 	req *dto.CreateStudentByTeacherRequest,
// 	teacherID string,
// ) (*dto.CreateStudentByTeacherResponse, error) {
// 	// 1. Verify teacher owns the class
// 	class, err := s.classRepo.FindByID(req.ClassID)
// 	if err != nil {
// 		return nil, fmt.Errorf("class not found: %w", err)
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("you are not authorized to add students to this class")
// 	}

// 	// 2. Check admission number uniqueness
// 	existing, _ := studentRepo.FindByAdmissionNo(req.AdmissionNo)
// 	if existing != nil {
// 		return nil, errors.New("admission number already exists")
// 	}

// 	// 3. Generate username if not provided
// 	username := req.Username
// 	if username == "" {
// 		username = req.AdmissionNo
// 		// ensure uniqueness
// 		for {
// 			_, err := userRepo.FindByUsername(username)
// 			if err != nil {
// 				break
// 			}
// 			username = fmt.Sprintf("%s%d", req.AdmissionNo, time.Now().UnixNano())
// 		}
// 	} else {
// 		// verify username uniqueness
// 		_, err := userRepo.FindByUsername(username)
// 		if err == nil {
// 			return nil, errors.New("username already taken")
// 		}
// 	}

// 	// 4. Generate random password
// 	rawPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate password: %w", err)
// 	}
// 	hashedPassword, err := utils.HashPassword(rawPassword)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to hash password: %w", err)
// 	}

// 	// 5. Create user (student role)
// 	user := &models.User{
// 		ID:            uuid.New().String(),
// 		Username:      username,
// 		Email:         req.GuardianEmail,
// 		Password:      hashedPassword,
// 		FirstName:     req.FirstName,
// 		LastName:      req.LastName,
// 		PhoneNumber:   req.GuardianPhone,
// 		Role:          models.RoleStudent,
// 		Status:        models.StatusActive,
// 		IsActive:      true,
// 		EmailVerified: false,
// 	}
// 	if err := userRepo.CreateUser(user); err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	// 6. Create student profile
// 	student := &models.Student{
// 		ID:            uuid.New().String(),
// 		UserID:        user.ID,
// 		SchoolID:      req.SchoolID,
// 		ClassID:       req.ClassID,
// 		AdmissionNo:   req.AdmissionNo,
// 		DateOfBirth:   req.DateOfBirth,
// 		Gender:        req.Gender,
// 		Address:       req.Address,
// 		GuardianName:  req.GuardianName,
// 		GuardianPhone: req.GuardianPhone,
// 		GuardianEmail: req.GuardianEmail,
// 		IsActive:      true,
// 		Status:        "active",
// 	}
// 	if err := studentRepo.Create(student); err != nil {
// 		// rollback user creation
// 		_ = userRepo.SoftDeleteUser(user.ID)
// 		return nil, fmt.Errorf("failed to create student profile: %w", err)
// 	}

// 	return &dto.CreateStudentByTeacherResponse{
// 		StudentID:         student.ID,
// 		UserID:            user.ID,
// 		Username:          user.Username,
// 		GeneratedPassword: rawPassword,
// 		AdmissionNo:       student.AdmissionNo,
// 		FirstName:         user.FirstName,
// 		LastName:          user.LastName,
// 	}, nil
// }

// // CreateStudent – single student creation (uses default repos)
// func (s *TeacherService) CreateStudent(req *dto.CreateStudentByTeacherRequest, teacherID string) (*dto.CreateStudentByTeacherResponse, error) {
// 	return s.createStudentInternal(s.userRepo, s.studentRepo, req, teacherID)
// }

// // GetMyStudents returns all students in the teacher's class
// func (s *TeacherService) GetMyStudents(teacherID string, page, limit int) ([]dto.StudentListResponse, int64, error) {
// 	classes, err := s.classRepo.FindByTeacherID(teacherID)
// 	if err != nil || len(classes) == 0 {
// 		return nil, 0, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID

// 	students, total, err := s.studentRepo.FindByClass(classID, page, limit)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	var result []dto.StudentListResponse
// 	for _, stu := range students {
// 		user, err := s.userRepo.FindByID(stu.UserID)
// 		if err != nil {
// 			s.logger.Warn("user not found for student", zap.String("student_id", stu.ID), zap.Error(err))
// 			continue
// 		}
// 		result = append(result, dto.StudentListResponse{
// 			ID:          stu.ID,
// 			AdmissionNo: stu.AdmissionNo,
// 			FirstName:   user.FirstName,
// 			LastName:    user.LastName,
// 			Username:    user.Username,
// 			Gender:      stu.Gender,
// 			IsActive:    stu.IsActive,
// 			Status:      stu.Status,
// 			CreatedAt:   stu.CreatedAt,
// 		})
// 	}
// 	return result, total, nil
// }

// // GetStudentByID returns a single student if belongs to teacher's class
// func (s *TeacherService) GetStudentByID(studentID, teacherID string) (*dto.StudentListResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to view this student")
// 	}

// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &dto.StudentListResponse{
// 		ID:          student.ID,
// 		AdmissionNo: student.AdmissionNo,
// 		FirstName:   user.FirstName,
// 		LastName:    user.LastName,
// 		Username:    user.Username,
// 		Gender:      student.Gender,
// 		IsActive:    student.IsActive,
// 		Status:      student.Status,
// 		CreatedAt:   student.CreatedAt,
// 	}, nil
// }

// // UpdateStudent updates a student's details (teacher only)
// func (s *TeacherService) UpdateStudent(studentID string, req *dto.UpdateStudentByTeacherRequest, teacherID string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to update this student")
// 	}

// 	// Update user fields
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return err
// 	}
// 	if req.FirstName != nil {
// 		user.FirstName = *req.FirstName
// 	}
// 	if req.LastName != nil {
// 		user.LastName = *req.LastName
// 	}
// 	if req.GuardianPhone != nil {
// 		user.PhoneNumber = *req.GuardianPhone
// 	}
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return err
// 	}

// 	// Update student fields
// 	if req.DateOfBirth != nil {
// 		student.DateOfBirth = req.DateOfBirth
// 	}
// 	if req.Gender != nil {
// 		student.Gender = *req.Gender
// 	}
// 	if req.Address != nil {
// 		student.Address = *req.Address
// 	}
// 	if req.GuardianName != nil {
// 		student.GuardianName = *req.GuardianName
// 	}
// 	if req.GuardianEmail != nil {
// 		student.GuardianEmail = *req.GuardianEmail
// 	}
// 	student.UpdatedAt = time.Now()
// 	return s.studentRepo.Update(student)
// }

// // ResetStudentPassword generates a new password and returns it
// func (s *TeacherService) ResetStudentPassword(studentID, teacherID string) (*dto.ResetStudentPasswordResponse, error) {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return nil, errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return nil, errors.New("unauthorized to reset password for this student")
// 	}

// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	newPassword, err := generateRandomPassword()
// 	if err != nil {
// 		return nil, errors.New("failed to generate new password")
// 	}
// 	hashed, err := utils.HashPassword(newPassword)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Password = hashed
// 	if err := s.userRepo.UpdateUser(user); err != nil {
// 		return nil, err
// 	}
// 	return &dto.ResetStudentPasswordResponse{
// 		Username: user.Username,
// 		Password: newPassword,
// 	}, nil
// }

// // DeactivateStudent sets IsActive=false and updates status
// func (s *TeacherService) DeactivateStudent(studentID, teacherID string, reason string) error {
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	class, err := s.classRepo.FindByID(student.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	if class.TeacherID == nil || *class.TeacherID != teacherID {
// 		return errors.New("unauthorized to deactivate this student")
// 	}
// 	student.IsActive = false
// 	student.Status = reason
// 	// Also deactivate the user account
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err == nil {
// 		user.IsActive = false
// 		_ = s.userRepo.UpdateUser(user)
// 	}
// 	return s.studentRepo.Update(student)
// }

// // BulkCreateStudents processes an Excel file and creates multiple students in a transaction.
// func (s *TeacherService) BulkCreateStudents(fileReader io.Reader, teacherID string) ([]dto.CreateStudentByTeacherResponse, []string, error) {
// 	// 1. Verify teacher has a class
// 	classes, err := s.classRepo.FindByTeacherID(teacherID)
// 	if err != nil || len(classes) == 0 {
// 		return nil, nil, errors.New("no class assigned to you")
// 	}
// 	classID := classes[0].ID
// 	schoolID := classes[0].SchoolID

// 	// 2. Parse Excel
// 	rows, err := excel.ReadStudentsFromExcel(fileReader)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to parse Excel: %w", err)
// 	}
// 	if len(rows) == 0 {
// 		return nil, nil, errors.New("no valid student rows found")
// 	}

// 	var created []dto.CreateStudentByTeacherResponse
// 	var errorsList []string

// 	// 3. Start a database transaction
// 	err = s.db.Transaction(func(tx *gorm.DB) error {
// 		// Create transaction-aware repositories
// 		txUserRepo := repository.NewUserRepository(tx)
// 		txStudentRepo := repository.NewStudentRepository(tx)

// 		for _, row := range rows {
// 			// Prepare the request DTO for this row
// 			req := &dto.CreateStudentByTeacherRequest{
// 				SchoolID:      schoolID,
// 				ClassID:       classID,
// 				FirstName:     row.FirstName,
// 				LastName:      row.LastName,
// 				Username:      row.Username,
// 				AdmissionNo:   row.AdmissionNo,
// 				DateOfBirth:   row.DateOfBirth,
// 				Gender:        row.Gender,
// 				Address:       row.Address,
// 				GuardianName:  row.GuardianName,
// 				GuardianPhone: row.GuardianPhone,
// 				GuardianEmail: row.GuardianEmail,
// 			}

// 			// Attempt to create the student using transaction repos
// 			resp, err := s.createStudentInternal(txUserRepo, txStudentRepo, req, teacherID)
// 			if err != nil {
// 				errMsg := fmt.Sprintf("Row %d: %v", row.RowNumber, err)
// 				errorsList = append(errorsList, errMsg)
// 				s.logger.Warn("bulk student creation failed for row", zap.Int("row", row.RowNumber), zap.Error(err))
// 				// Continue with next rows (partial success). To abort on any error, return err here.
// 				continue
// 			}
// 			created = append(created, *resp)
// 		}
// 		// If we want all-or-nothing, uncomment the next line:
// 		// if len(errorsList) > 0 { return fmt.Errorf("some rows failed") }
// 		return nil
// 	})

// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return created, errorsList, nil
// }