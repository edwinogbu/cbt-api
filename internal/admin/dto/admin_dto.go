package dto

// AssignTeacherRequest
type AssignTeacherRequest struct {
    ClassID   string `json:"class_id" binding:"required,uuid"`
    TeacherID string `json:"teacher_id" binding:"required,uuid"`
}

// UnassignTeacherRequest
type UnassignTeacherRequest struct {
    ClassID string `json:"class_id" binding:"required,uuid"`
}

// ListUsersQuery
type ListUsersQuery struct {
    Role   string `form:"role" binding:"omitempty,oneof=admin teacher student parent"`
    Page   int    `form:"page" default:"1"`
    Limit  int    `form:"limit" default:"20"`
    Search string `form:"search"`
}

// UserListResponse
type UserListResponse struct {
    ID          string `json:"id"`
    Username    string `json:"username"`
    Email       string `json:"email"`
    FirstName   string `json:"first_name"`
    LastName    string `json:"last_name"`
    PhoneNumber string `json:"phone_number"`
    Role        string `json:"role"`
    IsActive    bool   `json:"is_active"`
    CreatedAt   string `json:"created_at"`
}

// StudentListResponse
type StudentListResponse struct {
    ID           string `json:"id"`
    AdmissionNo  string `json:"admission_no"`
    UserID       string `json:"user_id"`
    FirstName    string `json:"first_name"`
    LastName     string `json:"last_name"`
    ClassName    string `json:"class_name"`
    IsActive     bool   `json:"is_active"`
    Status       string `json:"status"`
    CreatedAt    string `json:"created_at"`
}

// ClassListResponse – needed for ListAllClasses
type ClassListResponse struct {
    ID           string  `json:"id"`
    SchoolID     string  `json:"school_id"`
    SchoolName   string  `json:"school_name"`
    ClassLevelID string  `json:"class_level_id"`
    ClassLevel   string  `json:"class_level"`
    ClassArmID   string  `json:"class_arm_id"`
    ClassArm     string  `json:"class_arm"`
    ClassCode    string  `json:"class_code"`
    TeacherID    *string `json:"teacher_id"`
    TeacherName  string  `json:"teacher_name"`
    RoomNumber   string  `json:"room_number"`
    IsActive     bool    `json:"is_active"`
    CreatedAt    string  `json:"created_at"`
}

// TeacherClassInfo
type TeacherClassInfo struct {
    ClassID    string `json:"class_id"`
    ClassCode  string `json:"class_code"`
    ClassLevel string `json:"class_level"`
    ClassArm   string `json:"class_arm"`
    RoomNumber string `json:"room_number"`
    IsActive   bool   `json:"is_active"`
}

// TeacherListResponse
type TeacherListResponse struct {
    ID              string             `json:"id"`
    Username        string             `json:"username"`
    Email           string             `json:"email"`
    FirstName       string             `json:"first_name"`
    LastName        string             `json:"last_name"`
    PhoneNumber     string             `json:"phone_number"`
    IsActive        bool               `json:"is_active"`
    Status          string             `json:"status"`
    CreatedAt       string             `json:"created_at"`
    AssignedClasses []TeacherClassInfo `json:"assigned_classes"`
}


// package dto

// // AssignTeacherRequest represents the payload to assign a teacher to a class
// type AssignTeacherRequest struct {
// 	ClassID   string `json:"class_id" binding:"required,uuid"`
// 	TeacherID string `json:"teacher_id" binding:"required,uuid"`
// }

// // UnassignTeacherRequest (optional – we can also use path param)
// type UnassignTeacherRequest struct {
// 	ClassID string `json:"class_id" binding:"required,uuid"`
// }

// // ListUsersQuery for filtering
// type ListUsersQuery struct {
// 	Role   string `form:"role" binding:"omitempty,oneof=admin teacher student parent"`
// 	Page   int    `form:"page" default:"1"`
// 	Limit  int    `form:"limit" default:"20"`
// 	Search string `form:"search"`
// }

// // UserListResponse
// type UserListResponse struct {
// 	ID          string `json:"id"`
// 	Username    string `json:"username"`
// 	Email       string `json:"email"`
// 	FirstName   string `json:"first_name"`
// 	LastName    string `json:"last_name"`
// 	PhoneNumber string `json:"phone_number"`
// 	Role        string `json:"role"`
// 	IsActive    bool   `json:"is_active"`
// 	CreatedAt   string `json:"created_at"`
// }

// // StudentListResponse (for admin view)
// type StudentListResponse struct {
// 	ID           string `json:"id"`
// 	AdmissionNo  string `json:"admission_no"`
// 	UserID       string `json:"user_id"`
// 	FirstName    string `json:"first_name"`
// 	LastName     string `json:"last_name"`
// 	ClassName    string `json:"class_name"`
// 	IsActive     bool   `json:"is_active"`
// 	Status       string `json:"status"`
// 	CreatedAt    string `json:"created_at"`
// }

// // TeacherClassInfo represents a class assigned to a teacher
// type TeacherClassInfo struct {
//     ClassID     string `json:"class_id"`
//     ClassCode   string `json:"class_code"`
//     ClassLevel  string `json:"class_level"`
//     ClassArm    string `json:"class_arm"`
//     RoomNumber  string `json:"room_number"`
//     IsActive    bool   `json:"is_active"`
// }

// // TeacherListResponse for admin view
// type TeacherListResponse struct {
//     ID           string              `json:"id"`
//     Username     string              `json:"username"`
//     Email        string              `json:"email"`
//     FirstName    string              `json:"first_name"`
//     LastName     string              `json:"last_name"`
//     PhoneNumber  string              `json:"phone_number"`
//     IsActive     bool                `json:"is_active"`
//     Status       string              `json:"status"`
//     CreatedAt    string              `json:"created_at"`
//     AssignedClasses []TeacherClassInfo `json:"assigned_classes"`
// }

// // // ClassListResponse represents a class for admin listing
// // type ClassListResponse struct {
// //     ID           string  `json:"id"`
// //     SchoolID     string  `json:"school_id"`
// //     SchoolName   string  `json:"school_name"`
// //     ClassLevelID string  `json:"class_level_id"`
// //     ClassLevel   string  `json:"class_level"`
// //     ClassArmID   string  `json:"class_arm_id"`
// //     ClassArm     string  `json:"class_arm"`
// //     ClassCode    string  `json:"class_code"`
// //     TeacherID    *string `json:"teacher_id"`
// //     TeacherName  string  `json:"teacher_name"`
// //     RoomNumber   string  `json:"room_number"`
// //     IsActive     bool    `json:"is_active"`
// //     CreatedAt    string  `json:"created_at"`
// // }

// // type ClassListResponse struct {
// //     ID           string `json:"id"`
// //     SchoolID     string `json:"school_id"`
// //     SchoolName   string `json:"school_name"`
// //     ClassLevelID string `json:"class_level_id"`
// //     ClassLevel   string `json:"class_level"`
// //     ClassArmID   string `json:"class_arm_id"`
// //     ClassArm     string `json:"class_arm"`
// //     ClassCode    string `json:"class_code"`
// //     TeacherID    *string `json:"teacher_id"`
// //     TeacherName  string `json:"teacher_name"`
// //     RoomNumber   string `json:"room_number"`
// //     IsActive     bool   `json:"is_active"`
// //     CreatedAt    string `json:"created_at"`
// // }