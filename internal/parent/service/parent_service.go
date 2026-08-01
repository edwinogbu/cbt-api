package service

import (
	"context"  // ✅ ADD THIS IMPORT
	"errors"
	"fmt"

	"cbt-api/internal/academic/repository"
	"cbt-api/internal/models"
	"cbt-api/internal/parent/dto"
	cbtExamRepo "cbt-api/internal/cbt/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

type ParentService struct {
	parentRepo  *repository.ParentRepository
	studentRepo *repository.StudentRepository
	classRepo   *repository.ClassRepository
	examRepo    *cbtExamRepo.ExamRepository
	userRepo    *repository.UserRepository
	db          *gorm.DB
	logger      *zap.Logger
}

func NewParentService(
	parentRepo *repository.ParentRepository,
	studentRepo *repository.StudentRepository,
	classRepo *repository.ClassRepository,
	examRepo *cbtExamRepo.ExamRepository,
	userRepo *repository.UserRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *ParentService {
	return &ParentService{
		parentRepo:  parentRepo,
		studentRepo: studentRepo,
		classRepo:   classRepo,
		examRepo:    examRepo,
		userRepo:    userRepo,
		db:          db,
		logger:      logger,
	}
}

func (s *ParentService) AutoLinkParentOnRegister(parentID string, admissionNumber string) error {
	students, err := s.parentRepo.FindStudentsByAdmissionNumber(admissionNumber)
	if err != nil {
		return fmt.Errorf("failed to find students: %w", err)
	}
	if len(students) == 0 {
		return errors.New("no student found with that admission number")
	}
	for _, student := range students {
		linked, _ := s.parentRepo.IsLinked(parentID, student.ID)
		if linked {
			continue
		}
		link := &models.ParentStudent{
			ID:        generateID(),
			ParentID:  parentID,
			StudentID: student.ID,
		}
		if err := s.parentRepo.CreateLink(link); err != nil {
			s.logger.Error("failed to create parent-student link",
				zap.String("parent_id", parentID),
				zap.String("student_id", student.ID),
				zap.Error(err))
		}
	}
	return nil
}

func (s *ParentService) GetLinkedChildren(parentID string) ([]dto.ChildResponse, error) {
	links, err := s.parentRepo.FindLinksByParentID(parentID)
	if err != nil {
		return nil, err
	}
	var children []dto.ChildResponse
	for _, link := range links {
		student := link.Student
		if student.ID == "" {
			continue
		}
		user, err := s.userRepo.FindByID(student.UserID)
		if err != nil {
			s.logger.Warn("user not found for student", zap.String("student_id", student.ID))
			continue
		}
		className := ""
		teacherName := ""
		if student.Class != nil {
			if student.Class.ClassLevel != nil {
				className = student.Class.ClassLevel.Name
			}
			if student.Class.ClassArm != nil {
				if className != "" {
					className = className + " " + student.Class.ClassArm.Name
				} else {
					className = student.Class.ClassArm.Name
				}
			}
			if student.Class.TeacherID != nil {
				teacher, _ := s.userRepo.FindByID(*student.Class.TeacherID)
				if teacher != nil {
					teacherName = fmt.Sprintf("%s %s", teacher.FirstName, teacher.LastName)
				}
			}
		}
		children = append(children, dto.ChildResponse{
			StudentID:    student.ID,
			FullName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			AdmissionNo:  student.AdmissionNo,
			ClassName:    className,
			ClassTeacher: teacherName,
		})
	}
	return children, nil
}

// GetChildResults returns exam results for a specific child (only if linked)
// ADDED ctx context.Context parameter
func (s *ParentService) GetChildResults(ctx context.Context, parentID, studentID string) (*dto.ChildResultResponse, error) {
	linked, err := s.parentRepo.IsLinked(parentID, studentID)
	if err != nil {
		return nil, err
	}
	if !linked {
		return nil, errors.New("you are not linked to this student")
	}
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	user, err := s.userRepo.FindByID(student.UserID)
	if err != nil {
		return nil, err
	}
	className := ""
	teacherName := ""
	if student.Class != nil {
		if student.Class.ClassLevel != nil {
			className = student.Class.ClassLevel.Name
		}
		if student.Class.ClassArm != nil {
			if className != "" {
				className = className + " " + student.Class.ClassArm.Name
			} else {
				className = student.Class.ClassArm.Name
			}
		}
		if student.Class.TeacherID != nil {
			teacher, _ := s.userRepo.FindByID(*student.Class.TeacherID)
			if teacher != nil {
				teacherName = fmt.Sprintf("%s %s", teacher.FirstName, teacher.LastName)
			}
		}
	}
	childInfo := dto.ChildResponse{
		StudentID:    student.ID,
		FullName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		AdmissionNo:  student.AdmissionNo,
		ClassName:    className,
		ClassTeacher: teacherName,
	}

	// Fetch exam attempts for this student
	// ✅ ctx is now available
	examAttempts, err := s.examRepo.GetAttemptsByStudent(ctx, studentID)
	if err != nil {
		// No attempts – return empty results
		return &dto.ChildResultResponse{
			Child:   childInfo,
			Results: []dto.ExamResultResponse{},
		}, nil
	}

	var results []dto.ExamResultResponse
	for _, attempt := range examAttempts {
		// Fetch the exam details using attempt.ExamID
		exam, err := s.examRepo.FindExamByID(attempt.ExamID)
		if err != nil {
			s.logger.Warn("exam not found for attempt",
				zap.String("attempt_id", attempt.ID),
				zap.String("exam_id", attempt.ExamID),
				zap.Error(err))
			continue
		}

		// Convert Score (*int) to float64
		var score float64
		if attempt.Score != nil {
			score = float64(*attempt.Score)
		}

		// Total marks from Exam
		totalMarks := float64(exam.TotalMarks)

		// Calculate percentage
		percentage := 0.0
		if totalMarks > 0 {
			percentage = (score / totalMarks) * 100
		}

		grade := calculateGrade(percentage)

		results = append(results, dto.ExamResultResponse{
			ExamID:     attempt.ExamID,
			ExamTitle:  exam.Title,
			Subject:    "", // Subject name would require joining Subject table – optional
			Score:      score,
			TotalScore: totalMarks,
			Percentage: percentage,
			Grade:      grade,
			TakenAt:    attempt.CreatedAt,
		})
	}

	return &dto.ChildResultResponse{
		Child:   childInfo,
		Results: results,
	}, nil
}

func calculateGrade(percentage float64) string {
	switch {
	case percentage >= 80:
		return "A"
	case percentage >= 70:
		return "B"
	case percentage >= 60:
		return "C"
	case percentage >= 50:
		return "D"
	case percentage >= 40:
		return "E"
	default:
		return "F"
	}
}

func generateID() string {
	return uuid.New().String()
}


// package service

// import (
// 	"errors"
// 	"fmt"

// 	"cbt-api/internal/academic/repository"
// 	"cbt-api/internal/models"
// 	"cbt-api/internal/parent/dto"
// 	cbtExamRepo "cbt-api/internal/cbt/repository"
// 	"go.uber.org/zap"
// 	"gorm.io/gorm"
// 	"github.com/google/uuid"
// )

// type ParentService struct {
// 	parentRepo  *repository.ParentRepository
// 	studentRepo *repository.StudentRepository
// 	classRepo   *repository.ClassRepository
// 	examRepo    *cbtExamRepo.ExamRepository
// 	userRepo    *repository.UserRepository
// 	db          *gorm.DB
// 	logger      *zap.Logger
// }

// func NewParentService(
// 	parentRepo *repository.ParentRepository,
// 	studentRepo *repository.StudentRepository,
// 	classRepo *repository.ClassRepository,
// 	examRepo *cbtExamRepo.ExamRepository,
// 	userRepo *repository.UserRepository,
// 	db *gorm.DB,
// 	logger *zap.Logger,
// ) *ParentService {
// 	return &ParentService{
// 		parentRepo:  parentRepo,
// 		studentRepo: studentRepo,
// 		classRepo:   classRepo,
// 		examRepo:    examRepo,
// 		userRepo:    userRepo,
// 		db:          db,
// 		logger:      logger,
// 	}
// }

// func (s *ParentService) AutoLinkParentOnRegister(parentID string, admissionNumber string) error {
// 	students, err := s.parentRepo.FindStudentsByAdmissionNumber(admissionNumber)
// 	if err != nil {
// 		return fmt.Errorf("failed to find students: %w", err)
// 	}
// 	if len(students) == 0 {
// 		return errors.New("no student found with that admission number")
// 	}
// 	for _, student := range students {
// 		linked, _ := s.parentRepo.IsLinked(parentID, student.ID)
// 		if linked {
// 			continue
// 		}
// 		link := &models.ParentStudent{
// 			ID:        generateID(),
// 			ParentID:  parentID,
// 			StudentID: student.ID,
// 		}
// 		if err := s.parentRepo.CreateLink(link); err != nil {
// 			s.logger.Error("failed to create parent-student link",
// 				zap.String("parent_id", parentID),
// 				zap.String("student_id", student.ID),
// 				zap.Error(err))
// 		}
// 	}
// 	return nil
// }

// func (s *ParentService) GetLinkedChildren(parentID string) ([]dto.ChildResponse, error) {
// 	links, err := s.parentRepo.FindLinksByParentID(parentID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var children []dto.ChildResponse
// 	for _, link := range links {
// 		student := link.Student
// 		if student.ID == "" {
// 			continue
// 		}
// 		user, err := s.userRepo.FindByID(student.UserID)
// 		if err != nil {
// 			s.logger.Warn("user not found for student", zap.String("student_id", student.ID))
// 			continue
// 		}
// 		className := ""
// 		teacherName := ""
// 		if student.Class != nil {
// 			if student.Class.ClassLevel != nil {
// 				className = student.Class.ClassLevel.Name
// 			}
// 			if student.Class.ClassArm != nil {
// 				if className != "" {
// 					className = className + " " + student.Class.ClassArm.Name
// 				} else {
// 					className = student.Class.ClassArm.Name
// 				}
// 			}
// 			if student.Class.TeacherID != nil {
// 				teacher, _ := s.userRepo.FindByID(*student.Class.TeacherID)
// 				if teacher != nil {
// 					teacherName = fmt.Sprintf("%s %s", teacher.FirstName, teacher.LastName)
// 				}
// 			}
// 		}
// 		children = append(children, dto.ChildResponse{
// 			StudentID:    student.ID,
// 			FullName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
// 			AdmissionNo:  student.AdmissionNo,
// 			ClassName:    className,
// 			ClassTeacher: teacherName,
// 		})
// 	}
// 	return children, nil
// }

// // GetChildResults returns exam results for a specific child (only if linked)
// func (s *ParentService) GetChildResults(parentID, studentID string) (*dto.ChildResultResponse, error) {
// 	linked, err := s.parentRepo.IsLinked(parentID, studentID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !linked {
// 		return nil, errors.New("you are not linked to this student")
// 	}
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return nil, errors.New("student not found")
// 	}
// 	user, err := s.userRepo.FindByID(student.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	className := ""
// 	teacherName := ""
// 	if student.Class != nil {
// 		if student.Class.ClassLevel != nil {
// 			className = student.Class.ClassLevel.Name
// 		}
// 		if student.Class.ClassArm != nil {
// 			if className != "" {
// 				className = className + " " + student.Class.ClassArm.Name
// 			} else {
// 				className = student.Class.ClassArm.Name
// 			}
// 		}
// 		if student.Class.TeacherID != nil {
// 			teacher, _ := s.userRepo.FindByID(*student.Class.TeacherID)
// 			if teacher != nil {
// 				teacherName = fmt.Sprintf("%s %s", teacher.FirstName, teacher.LastName)
// 			}
// 		}
// 	}
// 	childInfo := dto.ChildResponse{
// 		StudentID:    student.ID,
// 		FullName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
// 		AdmissionNo:  student.AdmissionNo,
// 		ClassName:    className,
// 		ClassTeacher: teacherName,
// 	}

// 	// Fetch exam attempts for this student
// 	examAttempts, err := s.examRepo.GetAttemptsByStudent(ctx, studentID)
// 	if err != nil {
// 		// No attempts – return empty results
// 		return &dto.ChildResultResponse{
// 			Child:   childInfo,
// 			Results: []dto.ExamResultResponse{},
// 		}, nil
// 	}

// 	var results []dto.ExamResultResponse
// 	for _, attempt := range examAttempts {
// 		// Fetch the exam details using attempt.ExamID
// 		exam, err := s.examRepo.FindExamByID(attempt.ExamID)
// 		if err != nil {
// 			s.logger.Warn("exam not found for attempt",
// 				zap.String("attempt_id", attempt.ID),
// 				zap.String("exam_id", attempt.ExamID),
// 				zap.Error(err))
// 			continue
// 		}

// 		// Convert Score (*int) to float64
// 		var score float64
// 		if attempt.Score != nil {
// 			score = float64(*attempt.Score)
// 		}

// 		// Total marks from Exam
// 		totalMarks := float64(exam.TotalMarks)

// 		// Calculate percentage
// 		percentage := 0.0
// 		if totalMarks > 0 {
// 			percentage = (score / totalMarks) * 100
// 		}

// 		grade := calculateGrade(percentage)

// 		results = append(results, dto.ExamResultResponse{
// 			ExamID:     attempt.ExamID,
// 			ExamTitle:  exam.Title,
// 			Subject:    "", // Subject name would require joining Subject table – optional
// 			Score:      score,
// 			TotalScore: totalMarks,
// 			Percentage: percentage,
// 			Grade:      grade,
// 			TakenAt:    attempt.CreatedAt,
// 		})
// 	}

// 	return &dto.ChildResultResponse{
// 		Child:   childInfo,
// 		Results: results,
// 	}, nil
// }

// func calculateGrade(percentage float64) string {
// 	switch {
// 	case percentage >= 80:
// 		return "A"
// 	case percentage >= 70:
// 		return "B"
// 	case percentage >= 60:
// 		return "C"
// 	case percentage >= 50:
// 		return "D"
// 	case percentage >= 40:
// 		return "E"
// 	default:
// 		return "F"
// 	}
// }

// func generateID() string {
// 	return uuid.New().String()
// }
