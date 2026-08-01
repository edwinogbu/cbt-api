package dto

import "time"

// CreateStudentByTeacherRequest – admission_no and username are now optional
type CreateStudentByTeacherRequest struct {
	SchoolID      string     `json:"school_id" binding:"required,uuid"`
	ClassID       string     `json:"class_id" binding:"required,uuid"`
	FirstName     string     `json:"first_name" binding:"required"`
	LastName      string     `json:"last_name" binding:"required"`
	Username      string     `json:"username"`      // optional – auto from admission_no
	AdmissionNo   string     `json:"admission_no"`  // optional – auto generated
	DateOfBirth   *time.Time `json:"date_of_birth"`
	Gender        string     `json:"gender" binding:"omitempty,oneof=Male Female Other"`
	Address       string     `json:"address"`
	GuardianName  string     `json:"guardian_name"`
	GuardianPhone string     `json:"guardian_phone"`
	GuardianEmail string     `json:"guardian_email" binding:"omitempty,email"`
}

// CompleteStudentResponse – returned after creation
type CompleteStudentResponse struct {
	StudentID         string    `json:"student_id"`
	UserID            string    `json:"user_id"`
	AdmissionNo       string    `json:"admission_no"`
	Username          string    `json:"username"`
	GeneratedPassword string    `json:"generated_password"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	DateOfBirth       *time.Time `json:"date_of_birth,omitempty"`
	Gender            string    `json:"gender,omitempty"`
	Address           string    `json:"address,omitempty"`
	GuardianName      string    `json:"guardian_name,omitempty"`
	GuardianPhone     string    `json:"guardian_phone,omitempty"`
	GuardianEmail     string    `json:"guardian_email,omitempty"`

	Class struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ClassLevel  string `json:"class_level"`
		ClassArm    string `json:"class_arm"`
		TeacherName string `json:"teacher_name"`
	} `json:"class"`

	SchoolID               string    `json:"school_id"`
	SchoolName             string    `json:"school_name"`
	ExpectedGraduationYear int       `json:"expected_graduation_year"`
	TermsSpentSoFar        int       `json:"terms_spent_so_far"`
	IsActive               bool      `json:"is_active"`
	Status                 string    `json:"status"`
	CreatedAt              time.Time `json:"created_at"`
}

// ResetStudentPasswordResponse
type ResetStudentPasswordResponse struct {
	Username string `json:"username"`
	Password string `json:"new_password"`
}

// DeactivateStudentRequest
type DeactivateStudentRequest struct {
	Reason string `json:"reason" binding:"required,oneof=transferred expelled graduated deceased inactive"`
}

// UpdateStudentByTeacherRequest
type UpdateStudentByTeacherRequest struct {
	FirstName     *string    `json:"first_name"`
	LastName      *string    `json:"last_name"`
	DateOfBirth   *time.Time `json:"date_of_birth"`
	Gender        *string    `json:"gender" binding:"omitempty,oneof=Male Female Other"`
	Address       *string    `json:"address"`
	GuardianName  *string    `json:"guardian_name"`
	GuardianPhone *string    `json:"guardian_phone"`
	GuardianEmail *string    `json:"guardian_email" binding:"omitempty,email"`
}

// StudentListResponse - With Password
type StudentListResponse struct {
	ID          string    `json:"id"`
	AdmissionNo string    `json:"admission_no"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`  // Plain text password
	Gender      string    `json:"gender"`
	IsActive    bool      `json:"is_active"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// StudentCredentialResponse - Full student credentials with password
type StudentCredentialResponse struct {
	ID            string `json:"id"`
	AdmissionNo   string `json:"admission_no"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	PhoneNumber   string `json:"phone_number"`
	Gender        string `json:"gender"`
	Class         string `json:"class"`
	ClassLevel    string `json:"class_level"`
	ClassArm      string `json:"class_arm"`
	SchoolName    string `json:"school_name"`
	GuardianName  string `json:"guardian_name"`
	GuardianPhone string `json:"guardian_phone"`
	GuardianEmail string `json:"guardian_email"`
	IsActive      bool   `json:"is_active"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// GetAllStudentsWithCredentialsResponse - Teacher's class students with credentials
type GetAllStudentsWithCredentialsResponse struct {
	Data []StudentCredentialResponse `json:"data"`
	Meta struct {
		ClassName     string `json:"class_name"`
		ClassLevel    string `json:"class_level"`
		ClassArm      string `json:"class_arm"`
		TotalStudents int    `json:"total_students"`
		SchoolName    string `json:"school_name"`
		TeacherName   string `json:"teacher_name"`
	} `json:"meta"`
}


// package dto

// import "time"

// // CreateStudentByTeacherRequest – admission_no and username are now optional
// type CreateStudentByTeacherRequest struct {
// 	SchoolID      string     `json:"school_id" binding:"required,uuid"`
// 	ClassID       string     `json:"class_id" binding:"required,uuid"`
// 	FirstName     string     `json:"first_name" binding:"required"`
// 	LastName      string     `json:"last_name" binding:"required"`
// 	Username      string     `json:"username"`      // optional – auto from admission_no
// 	AdmissionNo   string     `json:"admission_no"`  // optional – auto generated
// 	DateOfBirth   *time.Time `json:"date_of_birth"`
// 	Gender        string     `json:"gender" binding:"omitempty,oneof=Male Female Other"`
// 	Address       string     `json:"address"`
// 	GuardianName  string     `json:"guardian_name"`
// 	GuardianPhone string     `json:"guardian_phone"`
// 	GuardianEmail string     `json:"guardian_email" binding:"omitempty,email"`
// }

// // CompleteStudentResponse – returned after creation (contains every detail)
// type CompleteStudentResponse struct {
// 	StudentID         string    `json:"student_id"`
// 	UserID            string    `json:"user_id"`
// 	AdmissionNo       string    `json:"admission_no"`
// 	Username          string    `json:"username"`
// 	GeneratedPassword string    `json:"generated_password"` // only on creation
// 	FirstName         string    `json:"first_name"`
// 	LastName          string    `json:"last_name"`
// 	DateOfBirth       *time.Time `json:"date_of_birth,omitempty"`
// 	Gender            string    `json:"gender,omitempty"`
// 	Address           string    `json:"address,omitempty"`
// 	GuardianName      string    `json:"guardian_name,omitempty"`
// 	GuardianPhone     string    `json:"guardian_phone,omitempty"`
// 	GuardianEmail     string    `json:"guardian_email,omitempty"`

// 	Class struct {
// 		ID          string `json:"id"`
// 		Name        string `json:"name"`
// 		ClassLevel  string `json:"class_level"`
// 		ClassArm    string `json:"class_arm"`
// 		TeacherName string `json:"teacher_name"`
// 	} `json:"class"`

// 	SchoolID               string    `json:"school_id"`
// 	SchoolName             string    `json:"school_name"`
// 	ExpectedGraduationYear int       `json:"expected_graduation_year"`
// 	TermsSpentSoFar        int       `json:"terms_spent_so_far"`
// 	IsActive               bool      `json:"is_active"`
// 	Status                 string    `json:"status"`
// 	CreatedAt              time.Time `json:"created_at"`
// }

// // ResetStudentPasswordResponse (unchanged)
// type ResetStudentPasswordResponse struct {
// 	Username string `json:"username"`
// 	Password string `json:"new_password"`
// }

// // DeactivateStudentRequest (unchanged)
// type DeactivateStudentRequest struct {
// 	Reason string `json:"reason" binding:"required,oneof=transferred expelled graduated deceased inactive"`
// }

// // UpdateStudentByTeacherRequest (unchanged)
// type UpdateStudentByTeacherRequest struct {
// 	FirstName     *string    `json:"first_name"`
// 	LastName      *string    `json:"last_name"`
// 	DateOfBirth   *time.Time `json:"date_of_birth"`
// 	Gender        *string    `json:"gender" binding:"omitempty,oneof=Male Female Other"`
// 	Address       *string    `json:"address"`
// 	GuardianName  *string    `json:"guardian_name"`
// 	GuardianPhone *string    `json:"guardian_phone"`
// 	GuardianEmail *string    `json:"guardian_email" binding:"omitempty,email"`
// }

// // StudentListResponse (unchanged)
// type StudentListResponse struct {
// 	ID          string    `json:"id"`
// 	AdmissionNo string    `json:"admission_no"`
// 	FirstName   string    `json:"first_name"`
// 	LastName    string    `json:"last_name"`
// 	Username    string    `json:"username"`
// 	Gender      string    `json:"gender"`
// 	IsActive    bool      `json:"is_active"`
// 	Status      string    `json:"status"`
// 	CreatedAt   time.Time `json:"created_at"`
// }

// // internal/teacher/dto/student_dto.go

// // StudentListResponse - Add Password field
// type StudentListResponse struct {
//     ID          string    `json:"id"`
//     AdmissionNo string    `json:"admission_no"`
//     FirstName   string    `json:"first_name"`
//     LastName    string    `json:"last_name"`
//     Username    string    `json:"username"`
//     Password    string    `json:"password"`  // ⭐ NEW - Plain text password
//     Gender      string    `json:"gender"`
//     IsActive    bool      `json:"is_active"`
//     Status      string    `json:"status"`
//     CreatedAt   time.Time `json:"created_at"`
// }

// // StudentCredentialResponse - New DTO for password exports
// type StudentCredentialResponse struct {
//     ID          string `json:"id"`
//     AdmissionNo string `json:"admission_no"`
//     FirstName   string `json:"first_name"`
//     LastName    string `json:"last_name"`
//     Username    string `json:"username"`
//     Password    string `json:"password"`
//     Email       string `json:"email"`
//     PhoneNumber string `json:"phone_number"`
//     Gender      string `json:"gender"`
//     Class       string `json:"class"`
//     IsActive    bool   `json:"is_active"`
//     Status      string `json:"status"`
//     CreatedAt   string `json:"created_at"`
// }


