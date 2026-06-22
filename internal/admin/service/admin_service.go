package service

import (
	"errors"
	"cbt-api/internal/academic/repository"
	"cbt-api/internal/admin/dto"
	"cbt-api/internal/models"
	// "github.com/google/uuid" // removed (unused)
	"gorm.io/gorm"
)

type AdminService struct {
	userRepo    *repository.UserRepository
	classRepo   *repository.ClassRepository
	studentRepo *repository.StudentRepository
	db          *gorm.DB
}

func NewAdminService(
	userRepo *repository.UserRepository,
	classRepo *repository.ClassRepository,
	studentRepo *repository.StudentRepository,
	db *gorm.DB,
) *AdminService {
	return &AdminService{
		userRepo:    userRepo,
		classRepo:   classRepo,
		studentRepo: studentRepo,
		db:          db,
	}
}

func (s *AdminService) AssignTeacher(req *dto.AssignTeacherRequest) error {
	class, err := s.classRepo.FindByID(req.ClassID)
	if err != nil {
		return errors.New("class not found")
	}
	teacher, err := s.userRepo.FindByID(req.TeacherID)
	if err != nil {
		return errors.New("teacher not found")
	}
	if teacher.Role != models.RoleTeacher {
		return errors.New("user is not a teacher")
	}
	class.TeacherID = &teacher.ID
	return s.classRepo.Update(class)
}

func (s *AdminService) UnassignTeacher(classID string) error {
	class, err := s.classRepo.FindByID(classID)
	if err != nil {
		return errors.New("class not found")
	}
	class.TeacherID = nil
	return s.classRepo.Update(class)
}

func (s *AdminService) ListUsersByRole(query dto.ListUsersQuery) ([]dto.UserListResponse, int64, error) {
	var users []models.User
	var total int64
	db := s.db.Model(&models.User{}).Where("deleted_at IS NULL")
	if query.Role != "" {
		db = db.Where("role = ?", query.Role)
	}
	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR username ILIKE ?",
			search, search, search, search)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (query.Page - 1) * query.Limit
	if err := db.Offset(offset).Limit(query.Limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	var result []dto.UserListResponse
	for _, u := range users {
		// ✅ convert *string email to string
		emailStr := ""
		if u.Email != nil {
			emailStr = *u.Email
		}
		result = append(result, dto.UserListResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       emailStr,
			FirstName:   u.FirstName,
			LastName:    u.LastName,
			PhoneNumber: u.PhoneNumber,
			Role:        string(u.Role),
			IsActive:    u.IsActive,
			CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, total, nil
}

// func (s *AdminService) ListAllStudents(page, limit int, search string) ([]dto.StudentListResponse, int64, error) {
// 	var students []models.Student
// 	var total int64
// 	// ✅ preload class level and arm
// 	query := s.db.Model(&models.Student{}).
// 		Preload("User").
// 		Preload("Class.ClassLevel").
// 		Preload("Class.ClassArm").
// 		Where("students.deleted_at IS NULL")
// 	if search != "" {
// 		searchTerm := "%" + search + "%"
// 		query = query.Where("students.admission_no ILIKE ? OR users.first_name ILIKE ? OR users.last_name ILIKE ?",
// 			searchTerm, searchTerm, searchTerm)
// 	}
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	offset := (page - 1) * limit
// 	if err := query.Offset(offset).Limit(limit).
// 		Order("students.created_at DESC").
// 		Find(&students).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	var result []dto.StudentListResponse
// 	for _, s := range students {
// 		// ✅ build class name from Level and Arm
// 		className := ""
// 		if s.Class != nil {
// 			if s.Class.ClassLevel != nil {
// 				className = s.Class.ClassLevel.Name
// 			}
// 			if s.Class.ClassArm != nil {
// 				if className != "" {
// 					className = className + " " + s.Class.ClassArm.Name
// 				} else {
// 					className = s.Class.ClassArm.Name
// 				}
// 			}
// 		}
// 		emailStr := ""
// 		if s.User.Email != nil {
// 			emailStr = *s.User.Email
// 		}
// 		result = append(result, dto.StudentListResponse{
// 			ID:          s.ID,
// 			AdmissionNo: s.AdmissionNo,
// 			UserID:      s.UserID,
// 			FirstName:   s.User.FirstName,
// 			LastName:    s.User.LastName,
// 			ClassName:   className,
// 			IsActive:    s.IsActive,
// 			Status:      s.Status,
// 			CreatedAt:   s.CreatedAt.Format("2006-01-02 15:04:05"),
// 		})
// 	}
// 	return result, total, nil
// }

// ListAllStudents returns all students with class name and user details
func (s *AdminService) ListAllStudents(page, limit int, search string) ([]dto.StudentListResponse, int64, error) {
    var students []models.Student
    var total int64

    // Preload necessary relationships
    query := s.db.Model(&models.Student{}).
        Preload("User").
        Preload("Class.ClassLevel").
        Preload("Class.ClassArm").
        Where("students.deleted_at IS NULL")

    if search != "" {
        searchTerm := "%" + search + "%"
        query = query.Where("students.admission_no ILIKE ? OR users.first_name ILIKE ? OR users.last_name ILIKE ?",
            searchTerm, searchTerm, searchTerm)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (page - 1) * limit
    if err := query.Offset(offset).Limit(limit).
        Order("students.created_at DESC").
        Find(&students).Error; err != nil {
        return nil, 0, err
    }

    var result []dto.StudentListResponse
    for _, s := range students {
        // Build class name from Level and Arm (if they exist)
        className := ""
        if s.Class != nil {
            if s.Class.ClassLevel != nil {
                className = s.Class.ClassLevel.Name
            }
            if s.Class.ClassArm != nil {
                if className != "" {
                    className = className + " " + s.Class.ClassArm.Name
                } else {
                    className = s.Class.ClassArm.Name
                }
            }
        }

        result = append(result, dto.StudentListResponse{
            ID:          s.ID,
            AdmissionNo: s.AdmissionNo,
            UserID:      s.UserID,
            FirstName:   s.User.FirstName,
            LastName:    s.User.LastName,
            ClassName:   className,
            IsActive:    s.IsActive,
            Status:      s.Status,
            CreatedAt:   s.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return result, total, nil
}

func (s *AdminService) HardDeleteStudent(studentID string) error {
	student, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		return errors.New("student not found")
	}
	return s.db.Unscoped().Delete(student).Error
}

// func (s *AdminService) ListAllClasses(page, limit int, search string) ([]dto.ClassListResponse, int64, error) {
//     var classes []models.Class
//     var total int64

//     query := s.db.Model(&models.Class{}).
//         Preload("School").
//         Preload("ClassLevel").
//         Preload("ClassArm").
//         Preload("Teacher").
//         Where("classes.deleted_at IS NULL")

//     if search != "" {
//         searchTerm := "%" + search + "%"
//         query = query.Where("class_code ILIKE ? OR class_level.name ILIKE ? OR class_arm.name ILIKE ?",
//             searchTerm, searchTerm, searchTerm)
//     }

//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, err
//     }

//     offset := (page - 1) * limit
//     if err := query.Offset(offset).Limit(limit).Order("classes.created_at DESC").Find(&classes).Error; err != nil {
//         return nil, 0, err
//     }

//     var result []dto.ClassListResponse
//     for _, c := range classes {
//         schoolName := ""
//         if c.School != nil {
//             schoolName = c.School.Name
//         }
//         className := ""
//         if c.ClassLevel != nil {
//             className = c.ClassLevel.Name
//         }
//         armName := ""
//         if c.ClassArm != nil {
//             armName = c.ClassArm.Name
//         }
//         teacherName := ""
//         if c.Teacher != nil {
//             teacherName = fmt.Sprintf("%s %s", c.Teacher.FirstName, c.Teacher.LastName)
//         }
//         result = append(result, dto.ClassListResponse{
//             ID:           c.ID,
//             SchoolID:     c.SchoolID,
//             SchoolName:   schoolName,
//             ClassLevelID: c.ClassLevelID,
//             ClassLevel:   className,
//             ClassArmID:   c.ClassArmID,
//             ClassArm:     armName,
//             ClassCode:    c.ClassCode,
//             TeacherID:    c.TeacherID,
//             TeacherName:  teacherName,
//             RoomNumber:   c.RoomNumber,
//             IsActive:     c.IsActive,
//             CreatedAt:    c.CreatedAt.Format("2006-01-02 15:04:05"),
//         })
//     }
//     return result, total, nil
// }

// ListAllClasses returns all classes with school, level, arm, and teacher details
func (s *AdminService) ListAllClasses(page, limit int, search string) ([]dto.ClassListResponse, int64, error) {
    var classes []models.Class
    var total int64

    query := s.db.Model(&models.Class{}).
        Preload("ClassLevel").
        Preload("ClassArm").
        Where("classes.deleted_at IS NULL")

    if search != "" {
        searchTerm := "%" + search + "%"
        query = query.Where("class_code ILIKE ? OR class_level.name ILIKE ? OR class_arm.name ILIKE ?",
            searchTerm, searchTerm, searchTerm)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (page - 1) * limit
    if err := query.Offset(offset).Limit(limit).
        Order("classes.created_at DESC").
        Find(&classes).Error; err != nil {
        return nil, 0, err
    }

    var result []dto.ClassListResponse
    for _, c := range classes {
        // Fetch school name
        schoolName := ""
        var school models.School
        if err := s.db.Where("id = ?", c.SchoolID).First(&school).Error; err == nil {
            schoolName = school.Name
        }

        className := ""
        if c.ClassLevel != nil {
            className = c.ClassLevel.Name
        }
        armName := ""
        if c.ClassArm != nil {
            armName = c.ClassArm.Name
        }

        // Fetch teacher name using UserRepository
        teacherName := ""
        if c.TeacherID != nil {
            teacher, err := s.userRepo.FindByID(*c.TeacherID)
            if err == nil && teacher != nil {
                teacherName = teacher.FirstName + " " + teacher.LastName
            }
        }

        result = append(result, dto.ClassListResponse{
            ID:           c.ID,
            SchoolID:     c.SchoolID,
            SchoolName:   schoolName,
            ClassLevelID: c.ClassLevelID,
            ClassLevel:   className,
            ClassArmID:   c.ClassArmID,
            ClassArm:     armName,
            ClassCode:    c.ClassCode,
            TeacherID:    c.TeacherID,
            TeacherName:  teacherName,
            RoomNumber:   c.RoomNumber,
            IsActive:     c.IsActive,
            CreatedAt:    c.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return result, total, nil
}

// ListAllTeachers returns all teachers with their personal details and assigned classes
func (s *AdminService) ListAllTeachers(page, limit int, search string) ([]dto.TeacherListResponse, int64, error) {
    var teachers []models.User
    var total int64

    // Query users with role 'teacher' and not soft-deleted
    query := s.db.Model(&models.User{}).
        Where("role = ? AND deleted_at IS NULL", models.RoleTeacher)

    if search != "" {
        searchTerm := "%" + search + "%"
        query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR username ILIKE ?",
            searchTerm, searchTerm, searchTerm, searchTerm)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (page - 1) * limit
    if err := query.Offset(offset).Limit(limit).
        Order("created_at DESC").
        Find(&teachers).Error; err != nil {
        return nil, 0, err
    }

    var result []dto.TeacherListResponse
    for _, teacher := range teachers {
        // Fetch classes where this teacher is assigned
        var classes []models.Class
        if err := s.db.Model(&models.Class{}).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("teacher_id = ? AND deleted_at IS NULL", teacher.ID).
            Find(&classes).Error; err != nil {
            // Log error but continue (return empty classes list for this teacher)
            classes = []models.Class{}
        }

        assignedClasses := make([]dto.TeacherClassInfo, 0, len(classes))
        for _, cls := range classes {
            className := ""
            if cls.ClassLevel != nil {
                className = cls.ClassLevel.Name
            }
            armName := ""
            if cls.ClassArm != nil {
                armName = cls.ClassArm.Name
            }
            assignedClasses = append(assignedClasses, dto.TeacherClassInfo{
                ClassID:    cls.ID,
                ClassCode:  cls.ClassCode,
                ClassLevel: className,
                ClassArm:   armName,
                RoomNumber: cls.RoomNumber,
                IsActive:   cls.IsActive,
            })
        }

        emailStr := ""
        if teacher.Email != nil {
            emailStr = *teacher.Email
        }

        result = append(result, dto.TeacherListResponse{
            ID:           teacher.ID,
            Username:     teacher.Username,
            Email:        emailStr,
            FirstName:    teacher.FirstName,
            LastName:     teacher.LastName,
            PhoneNumber:  teacher.PhoneNumber,
            IsActive:     teacher.IsActive,
            Status:       string(teacher.Status),
            CreatedAt:    teacher.CreatedAt.Format("2006-01-02 15:04:05"),
            AssignedClasses: assignedClasses,
        })
    }
    return result, total, nil
}



// package service

// import (
// 	"errors"
// 	"cbt-api/internal/academic/repository"
// 	"cbt-api/internal/admin/dto"
// 	"cbt-api/internal/models"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// type AdminService struct {
// 	userRepo    *repository.UserRepository
// 	classRepo   *repository.ClassRepository
// 	studentRepo *repository.StudentRepository
// 	db          *gorm.DB
// }

// func NewAdminService(
// 	userRepo *repository.UserRepository,
// 	classRepo *repository.ClassRepository,
// 	studentRepo *repository.StudentRepository,
// 	db *gorm.DB,
// ) *AdminService {
// 	return &AdminService{
// 		userRepo:    userRepo,
// 		classRepo:   classRepo,
// 		studentRepo: studentRepo,
// 		db:          db,
// 	}
// }

// // AssignTeacher assigns a teacher to a class
// func (s *AdminService) AssignTeacher(req *dto.AssignTeacherRequest) error {
// 	// 1. Find class
// 	class, err := s.classRepo.FindByID(req.ClassID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}

// 	// 2. Find teacher (must exist and have role "teacher")
// 	teacher, err := s.userRepo.FindByID(req.TeacherID)
// 	if err != nil {
// 		return errors.New("teacher not found")
// 	}
// 	if teacher.Role != models.RoleTeacher {
// 		return errors.New("user is not a teacher")
// 	}

// 	// 3. Assign teacher
// 	class.TeacherID = &teacher.ID
// 	return s.classRepo.Update(class)
// }

// // UnassignTeacher removes teacher from a class
// func (s *AdminService) UnassignTeacher(classID string) error {
// 	class, err := s.classRepo.FindByID(classID)
// 	if err != nil {
// 		return errors.New("class not found")
// 	}
// 	class.TeacherID = nil
// 	return s.classRepo.Update(class)
// }

// // ListUsersByRole returns all users filtered by role (with pagination and search)
// func (s *AdminService) ListUsersByRole(query dto.ListUsersQuery) ([]dto.UserListResponse, int64, error) {
// 	var users []models.User
// 	var total int64

// 	db := s.db.Model(&models.User{}).Where("deleted_at IS NULL")

// 	if query.Role != "" {
// 		db = db.Where("role = ?", query.Role)
// 	}
// 	if query.Search != "" {
// 		search := "%" + query.Search + "%"
// 		db = db.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR username ILIKE ?",
// 			search, search, search, search)
// 	}

// 	// Count total
// 	if err := db.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	offset := (query.Page - 1) * query.Limit
// 	if err := db.Offset(offset).Limit(query.Limit).Order("created_at DESC").Find(&users).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	var result []dto.UserListResponse
// 	for _, u := range users {
// 		result = append(result, dto.UserListResponse{
// 			ID:          u.ID,
// 			Username:    u.Username,
// 			Email:       u.Email,
// 			FirstName:   u.FirstName,
// 			LastName:    u.LastName,
// 			PhoneNumber: u.PhoneNumber,
// 			Role:        string(u.Role),
// 			IsActive:    u.IsActive,
// 			CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
// 		})
// 	}
// 	return result, total, nil
// }

// // ListAllStudents returns all students with class name and user details
// func (s *AdminService) ListAllStudents(page, limit int, search string) ([]dto.StudentListResponse, int64, error) {
// 	var students []models.Student
// 	var total int64

// 	query := s.db.Model(&models.Student{}).
// 		Preload("User").
// 		Preload("Class").
// 		Where("students.deleted_at IS NULL")

// 	if search != "" {
// 		searchTerm := "%" + search + "%"
// 		query = query.Where("students.admission_no ILIKE ? OR users.first_name ILIKE ? OR users.last_name ILIKE ?",
// 			searchTerm, searchTerm, searchTerm)
// 	}

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	offset := (page - 1) * limit
// 	if err := query.Offset(offset).Limit(limit).
// 		Order("students.created_at DESC").
// 		Find(&students).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	var result []dto.StudentListResponse
// 	for _, s := range students {
// 		className := ""
// 		if s.Class != nil {
// 			className = s.Class.Name
// 		}
// 		result = append(result, dto.StudentListResponse{
// 			ID:          s.ID,
// 			AdmissionNo: s.AdmissionNo,
// 			UserID:      s.UserID,
// 			FirstName:   s.User.FirstName,
// 			LastName:    s.User.LastName,
// 			ClassName:   className,
// 			IsActive:    s.IsActive,
// 			Status:      s.Status,
// 			CreatedAt:   s.CreatedAt.Format("2006-01-02 15:04:05"),
// 		})
// 	}
// 	return result, total, nil
// }

// // HardDeleteStudent permanently deletes a student (soft delete is default, this is hard)
// func (s *AdminService) HardDeleteStudent(studentID string) error {
// 	// First, check if student exists
// 	student, err := s.studentRepo.FindByID(studentID)
// 	if err != nil {
// 		return errors.New("student not found")
// 	}
// 	// Perform hard delete (remove from DB)
// 	return s.db.Unscoped().Delete(student).Error
// }