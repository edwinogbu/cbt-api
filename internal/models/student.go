// internal/models/student.go
package models

import (
    "time"
    "gorm.io/gorm"
)

// Student – enhanced with Status field and relationships
type Student struct {
    ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    UserID        string         `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
    SchoolID      string         `gorm:"type:uuid;not null;index" json:"school_id"`
    ClassID       string         `gorm:"type:uuid;index" json:"class_id"`
    AdmissionNo   string         `gorm:"uniqueIndex;not null" json:"admission_no"`
    DateOfBirth   *time.Time     `json:"date_of_birth"`
    Gender        string         `gorm:"type:varchar(10)" json:"gender"`
    Address       string         `json:"address"`
    GuardianName  string         `json:"guardian_name"`
    GuardianPhone string         `json:"guardian_phone"`
    GuardianEmail string         `json:"guardian_email"`
    IsActive      bool           `gorm:"default:true" json:"is_active"`
    Status        string         `gorm:"type:varchar(20);default:'active'" json:"status"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

    // Relationships
    User  User  `gorm:"foreignKey:UserID"`
    Class *Class `gorm:"foreignKey:ClassID"`
}

// ParentStudent – new model for parent-child linking
type ParentStudent struct {
    ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    ParentID  string    `gorm:"type:uuid;not null;index" json:"parent_id"`
    StudentID string    `gorm:"type:uuid;not null;index" json:"student_id"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

    Parent  User    `gorm:"foreignKey:ParentID"`
    Student Student `gorm:"foreignKey:StudentID"`
}

func (Student) TableName() string { return "students" }
func (ParentStudent) TableName() string { return "parent_students" }


// package models

// import (
//     "time"
    
//     "gorm.io/gorm"
// )

// type Student struct {
//     ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     UserID        string         `gorm:"type:uuid;not null;index" json:"user_id"`
//     SchoolID      string         `gorm:"type:uuid;not null;index" json:"school_id"`
//     ClassID       string         `gorm:"type:uuid;index" json:"class_id"`
//     AdmissionNo   string         `gorm:"uniqueIndex;not null" json:"admission_no"`
//     DateOfBirth   *time.Time     `json:"date_of_birth"`
//     Gender        string         `json:"gender"`
//     Address       string         `json:"address"`
//     GuardianName  string         `json:"guardian_name"`
//     GuardianPhone string         `json:"guardian_phone"`
//     GuardianEmail string         `json:"guardian_email"`
//     IsActive      bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt     time.Time      `json:"created_at"`
//     UpdatedAt     time.Time      `json:"updated_at"`
//     DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
// }

// func (Student) TableName() string { return "students" }

// // package models  // CHANGE FROM "package dto" TO "package models"

// // import (
// //     "time"
    
// //     // "github.com/google/uuid"
// //     "gorm.io/gorm"
// // )

// // type Student struct {
// //     ID            string         `gorm:"type:uuid;primaryKey" json:"id"`
// //     UserID        string         `gorm:"type:uuid;not null;index" json:"user_id"`
// //     SchoolID      string         `gorm:"type:uuid;not null;index" json:"school_id"`
// //     ClassID       string         `gorm:"type:uuid;index" json:"class_id"`
// //     AdmissionNo   string         `gorm:"uniqueIndex;not null" json:"admission_no"`
// //     DateOfBirth   *time.Time     `json:"date_of_birth"`
// //     Gender        string         `json:"gender"`
// //     Address       string         `json:"address"`
// //     GuardianName  string         `json:"guardian_name"`
// //     GuardianPhone string         `json:"guardian_phone"`
// //     GuardianEmail string         `json:"guardian_email"`
// //     IsActive      bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt     time.Time      `json:"created_at"`
// //     UpdatedAt     time.Time      `json:"updated_at"`
// //     DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // func (Student) TableName() string { return "students" }


// // package dto

// // import "time"

// // // Request DTOs
// // type CreateStudentRequest struct {
// //     UserID        string     `json:"user_id" binding:"required"`
// //     SchoolID      string     `json:"school_id" binding:"required"`
// //     ClassID       string     `json:"class_id"`
// //     AdmissionNo   string     `json:"admission_no" binding:"required"`
// //     DateOfBirth   *time.Time `json:"date_of_birth"`
// //     Gender        string     `json:"gender" binding:"omitempty,oneof=Male Female Other"`
// //     Address       string     `json:"address"`
// //     GuardianName  string     `json:"guardian_name"`
// //     GuardianPhone string     `json:"guardian_phone"`
// //     GuardianEmail string     `json:"guardian_email" binding:"omitempty,email"`
// // }

// // type UpdateStudentRequest struct {
// //     ClassID       *string    `json:"class_id"`
// //     DateOfBirth   *time.Time `json:"date_of_birth"`
// //     Gender        string     `json:"gender"`
// //     Address       string     `json:"address"`
// //     GuardianName  string     `json:"guardian_name"`
// //     GuardianPhone string     `json:"guardian_phone"`
// //     GuardianEmail string     `json:"guardian_email"`
// //     IsActive      *bool      `json:"is_active"`
// // }

// // // Response DTOs
// // type StudentResponse struct {
// //     ID            string     `json:"id"`
// //     UserID        string     `json:"user_id"`
// //     SchoolID      string     `json:"school_id"`
// //     ClassID       string     `json:"class_id"`
// //     AdmissionNo   string     `json:"admission_no"`
// //     DateOfBirth   *time.Time `json:"date_of_birth"`
// //     Gender        string     `json:"gender"`
// //     Address       string     `json:"address"`
// //     GuardianName  string     `json:"guardian_name"`
// //     GuardianPhone string     `json:"guardian_phone"`
// //     GuardianEmail string     `json:"guardian_email"`
// //     IsActive      bool       `json:"is_active"`
// //     CreatedAt     time.Time  `json:"created_at"`
// //     UpdatedAt     time.Time  `json:"updated_at"`
// // }

// // // Student with User Details
// // type StudentDetailResponse struct {
// //     StudentResponse
// //     User UserBriefDTO `json:"user"`
// // }

// // type UserBriefDTO struct {
// //     ID        string `json:"id"`
// //     Email     string `json:"email"`
// //     FirstName string `json:"first_name"`
// //     LastName  string `json:"last_name"`
// //     PhoneNumber string `json:"phone_number"`
// // }

// // // package models

// // // import (
// // //     "time"
    
// // //     "github.com/google/uuid"
// // //     "gorm.io/gorm"
// // // )

// // // type Student struct {
// // //     ID            string         `gorm:"type:uuid;primaryKey" json:"id"`
// // //     UserID        string         `gorm:"type:uuid;not null;index" json:"user_id"`
// // //     SchoolID      string         `gorm:"type:uuid;not null;index" json:"school_id"`
// // //     ClassID       string         `gorm:"type:uuid;index" json:"class_id"`
// // //     AdmissionNo   string         `gorm:"uniqueIndex;not null" json:"admission_no"`
// // //     DateOfBirth   *time.Time     `json:"date_of_birth"`
// // //     Gender        string         `json:"gender"`
// // //     Address       string         `json:"address"`
// // //     GuardianName  string         `json:"guardian_name"`
// // //     GuardianPhone string         `json:"guardian_phone"`
// // //     GuardianEmail string         `json:"guardian_email"`
// // //     IsActive      bool           `gorm:"default:true" json:"is_active"`
// // //     CreatedAt     time.Time      `json:"created_at"`
// // //     UpdatedAt     time.Time      `json:"updated_at"`
// // //     DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
// // // }

// // // func (Student) TableName() string { return "students" }