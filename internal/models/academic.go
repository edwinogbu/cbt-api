package models

import (
    "strings"
    "time"
    
    "gorm.io/gorm"
)

// // ============================================
// // SHARED HELPER - CLASS DISPLAY NAME
// // ============================================

// // GetClassDisplayName returns the display name for a class
// // Combines ClassLevel.Name + ClassArm.Name if both exist
// // This is a model-level helper that can be used by any service
// func GetClassDisplayName(class *Class) string {
//     if class == nil {
//         return ""
//     }
//     name := ""
//     if class.ClassLevel != nil {
//         name = class.ClassLevel.Name
//     }
//     if class.ClassArm != nil {
//         if name != "" {
//             name += " " + class.ClassArm.Name
//         } else {
//             name = class.ClassArm.Name
//         }
//     }
//     return name
// }

// ============================================
// SHARED HELPER - CLASS DISPLAY NAME
// ============================================

// GetClassDisplayName returns the display name for a class
// Combines ClassLevel.Name + ClassArm.Name if both exist
// Prevents duplication like "JSS 1 JSS 1 progress"
func GetClassDisplayName(class *Class) string {
    if class == nil {
        return ""
    }
    
    levelName := ""
    if class.ClassLevel != nil {
        levelName = class.ClassLevel.Name
    }
    
    armName := ""
    if class.ClassArm != nil {
        armName = class.ClassArm.Name
    }
    
    // If both exist
    if levelName != "" && armName != "" {
        // ✅ FIX: Check if arm name is already contained in level name
        // This prevents "JSS 1 JSS 1 progress"
        if strings.Contains(levelName, armName) {
            return levelName
        }
        if strings.Contains(armName, levelName) {
            return armName
        }
        return levelName + " " + armName
    }
    
    if levelName != "" {
        return levelName
    }
    
    if armName != "" {
        return armName
    }
    
    return ""
}

// ============================================
// ACADEMIC MODELS
// ============================================

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

    ClassLevel *ClassLevel `gorm:"foreignKey:ClassLevelID"`
    ClassArm   *ClassArm   `gorm:"foreignKey:ClassArmID"`
    Teacher    *User       `gorm:"foreignKey:TeacherID"`
}

func (AcademicSession) TableName() string { return "academic_sessions" }
func (Term) TableName() string { return "terms" }
func (ClassLevel) TableName() string { return "class_levels" }
func (ClassArm) TableName() string { return "class_arms" }
func (Class) TableName() string { return "classes" }
