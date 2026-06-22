package models

import (
    "time"
    
    "gorm.io/gorm"
)

// AcademicSession remains unchanged
type AcademicSession struct {
    ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SchoolID  string         `gorm:"type:uuid;not null;index" json:"school_id"`
    Name      string         `gorm:"not null" json:"name"`
    StartDate time.Time      `json:"start_date"`
    EndDate   time.Time      `json:"end_date"`
    IsCurrent bool           `gorm:"default:false" json:"is_current"`
    IsActive  bool           `gorm:"default:true" json:"is_active"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Term unchanged
type Term struct {
    ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SessionID  string         `gorm:"type:uuid;not null;index" json:"session_id"`
    TermNumber int            `gorm:"not null" json:"term_number"`
    Name       string         `json:"name"`
    StartDate  time.Time      `json:"start_date"`
    EndDate    time.Time      `json:"end_date"`
    IsCurrent  bool           `gorm:"default:false" json:"is_current"`
    IsActive   bool           `gorm:"default:true" json:"is_active"`
    CreatedAt  time.Time      `json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// ClassLevel unchanged
type ClassLevel struct {
    ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SchoolID    string         `gorm:"type:uuid;not null;index" json:"school_id"`
    Name        string         `json:"name"`
    LevelNumber int            `json:"level_number"`
    Category    string         `json:"category"`
    PromotionTo *string        `gorm:"type:uuid" json:"promotion_to"`
    SortOrder   int            `gorm:"default:0" json:"sort_order"`
    IsActive    bool           `gorm:"default:true" json:"is_active"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ClassArm unchanged
type ClassArm struct {
    ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SchoolID   string         `gorm:"type:uuid;not null;index" json:"school_id"`
    Name       string         `json:"name"`
    ArmCode    string         `json:"arm_code"`
    Capacity   int            `gorm:"default:40" json:"capacity"`
    RoomNumber string         `json:"room_number"`
    SortOrder  int            `gorm:"default:0" json:"sort_order"`
    IsActive   bool           `gorm:"default:true" json:"is_active"`
    CreatedAt  time.Time      `json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// Class unchanged
type Class struct {
    ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SchoolID     string         `gorm:"type:uuid;not null;index" json:"school_id"`
    SessionID    string         `gorm:"type:uuid;not null;index" json:"session_id"`
    ClassLevelID string         `gorm:"type:uuid;not null;index" json:"class_level_id"`
    ClassArmID   string         `gorm:"type:uuid;not null;index" json:"class_arm_id"`
    ClassCode    string         `gorm:"uniqueIndex" json:"class_code"`
    TeacherID    *string        `gorm:"type:uuid" json:"teacher_id"`
    RoomNumber   string         `json:"room_number"`
    IsActive     bool           `gorm:"default:true" json:"is_active"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

     // Add these relationships
    ClassLevel *ClassLevel `gorm:"foreignKey:ClassLevelID"`
    ClassArm   *ClassArm   `gorm:"foreignKey:ClassArmID"`
    Teacher *User `gorm:"foreignKey:TeacherID"` // add this line

}


// Table names
func (AcademicSession) TableName() string { return "academic_sessions" }
func (Term) TableName() string { return "terms" }
func (ClassLevel) TableName() string { return "class_levels" }
func (ClassArm) TableName() string { return "class_arms" }
func (Class) TableName() string { return "classes" }


// func (Student) TableName() string { return "students" }
// func (ParentStudent) TableName() string { return "parent_students" }




// package models

// import (
//     "time"
    
//     "gorm.io/gorm"
// )

// type AcademicSession struct {
//     ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     SchoolID  string         `gorm:"type:uuid;not null;index" json:"school_id"`
//     Name      string         `gorm:"not null" json:"name"`
//     StartDate time.Time      `json:"start_date"`
//     EndDate   time.Time      `json:"end_date"`
//     IsCurrent bool           `gorm:"default:false" json:"is_current"`
//     IsActive  bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt time.Time      `json:"created_at"`
//     UpdatedAt time.Time      `json:"updated_at"`
//     DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type Term struct {
//     ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     SessionID  string         `gorm:"type:uuid;not null;index" json:"session_id"`
//     TermNumber int            `gorm:"not null" json:"term_number"`
//     Name       string         `json:"name"`
//     StartDate  time.Time      `json:"start_date"`
//     EndDate    time.Time      `json:"end_date"`
//     IsCurrent  bool           `gorm:"default:false" json:"is_current"`
//     IsActive   bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt  time.Time      `json:"created_at"`
//     UpdatedAt  time.Time      `json:"updated_at"`
//     DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type ClassLevel struct {
//     ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     SchoolID    string         `gorm:"type:uuid;not null;index" json:"school_id"`
//     Name        string         `json:"name"`
//     LevelNumber int            `json:"level_number"`
//     Category    string         `json:"category"`
//     PromotionTo *string        `gorm:"type:uuid" json:"promotion_to"`
//     SortOrder   int            `gorm:"default:0" json:"sort_order"`
//     IsActive    bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt   time.Time      `json:"created_at"`
//     UpdatedAt   time.Time      `json:"updated_at"`
//     DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type ClassArm struct {
//     ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     SchoolID   string         `gorm:"type:uuid;not null;index" json:"school_id"`
//     Name       string         `json:"name"`
//     ArmCode    string         `json:"arm_code"`
//     Capacity   int            `gorm:"default:40" json:"capacity"`
//     RoomNumber string         `json:"room_number"`
//     SortOrder  int            `gorm:"default:0" json:"sort_order"`
//     IsActive   bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt  time.Time      `json:"created_at"`
//     UpdatedAt  time.Time      `json:"updated_at"`
//     DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type Class struct {
//     ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
//     SchoolID     string         `gorm:"type:uuid;not null;index" json:"school_id"`
//     SessionID    string         `gorm:"type:uuid;not null;index" json:"session_id"`
//     ClassLevelID string         `gorm:"type:uuid;not null;index" json:"class_level_id"`
//     ClassArmID   string         `gorm:"type:uuid;not null;index" json:"class_arm_id"`
//     ClassCode    string         `gorm:"uniqueIndex" json:"class_code"`
//     TeacherID    *string        `gorm:"type:uuid" json:"teacher_id"`
//     RoomNumber   string         `json:"room_number"`
//     IsActive     bool           `gorm:"default:true" json:"is_active"`
//     CreatedAt    time.Time      `json:"created_at"`
//     UpdatedAt    time.Time      `json:"updated_at"`
//     DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
// }

// func (AcademicSession) TableName() string { return "academic_sessions" }
// func (Term) TableName() string { return "terms" }
// func (ClassLevel) TableName() string { return "class_levels" }
// func (ClassArm) TableName() string { return "class_arms" }
// func (Class) TableName() string { return "classes" }


// // package models

// // import (
// //     "time"
    
// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// // )

// // type AcademicSession struct {
// //     ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     SchoolID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"school_id"`
// //     Name      string         `gorm:"not null" json:"name"`
// //     StartDate time.Time      `json:"start_date"`
// //     EndDate   time.Time      `json:"end_date"`
// //     IsCurrent bool           `gorm:"default:false" json:"is_current"`
// //     IsActive  bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt time.Time      `json:"created_at"`
// //     UpdatedAt time.Time      `json:"updated_at"`
// //     DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // type Term struct {
// //     ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     SessionID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"session_id"`
// //     TermNumber int            `gorm:"not null" json:"term_number"`
// //     Name       string         `json:"name"`
// //     StartDate  time.Time      `json:"start_date"`
// //     EndDate    time.Time      `json:"end_date"`
// //     IsCurrent  bool           `gorm:"default:false" json:"is_current"`
// //     IsActive   bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt  time.Time      `json:"created_at"`
// //     UpdatedAt  time.Time      `json:"updated_at"`
// //     DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // type ClassLevel struct {
// //     ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     SchoolID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"school_id"`
// //     Name        string         `json:"name"`
// //     LevelNumber int            `json:"level_number"`
// //     Category    string         `json:"category"`
// //     PromotionTo *uuid.UUID     `json:"promotion_to"`
// //     SortOrder   int            `gorm:"default:0" json:"sort_order"`
// //     IsActive    bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt   time.Time      `json:"created_at"`
// //     UpdatedAt   time.Time      `json:"updated_at"`
// //     DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // type ClassArm struct {
// //     ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     SchoolID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"school_id"`
// //     Name       string         `json:"name"`
// //     ArmCode    string         `json:"arm_code"`
// //     Capacity   int            `gorm:"default:40" json:"capacity"`
// //     RoomNumber string         `json:"room_number"`
// //     SortOrder  int            `gorm:"default:0" json:"sort_order"`
// //     IsActive   bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt  time.Time      `json:"created_at"`
// //     UpdatedAt  time.Time      `json:"updated_at"`
// //     DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // type Class struct {
// //     ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// //     SchoolID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"school_id"`
// //     SessionID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"session_id"`
// //     ClassLevelID uuid.UUID      `gorm:"type:uuid;not null;index" json:"class_level_id"`
// //     ClassArmID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"class_arm_id"`
// //     ClassCode    string         `gorm:"uniqueIndex" json:"class_code"`
// //     TeacherID    *uuid.UUID     `json:"teacher_id"`
// //     RoomNumber   string         `json:"room_number"`
// //     IsActive     bool           `gorm:"default:true" json:"is_active"`
// //     CreatedAt    time.Time      `json:"created_at"`
// //     UpdatedAt    time.Time      `json:"updated_at"`
// //     DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
// // }

// // func (AcademicSession) TableName() string { return "academic_sessions" }
// // func (Term) TableName() string { return "terms" }
// // func (ClassLevel) TableName() string { return "class_levels" }
// // func (ClassArm) TableName() string { return "class_arms" }
// // func (Class) TableName() string { return "classes" }