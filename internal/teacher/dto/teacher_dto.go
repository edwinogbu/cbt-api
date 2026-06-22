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

// CompleteStudentResponse – returned after creation (contains every detail)
type CompleteStudentResponse struct {
	StudentID         string    `json:"student_id"`
	UserID            string    `json:"user_id"`
	AdmissionNo       string    `json:"admission_no"`
	Username          string    `json:"username"`
	GeneratedPassword string    `json:"generated_password"` // only on creation
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

// ResetStudentPasswordResponse (unchanged)
type ResetStudentPasswordResponse struct {
	Username string `json:"username"`
	Password string `json:"new_password"`
}

// DeactivateStudentRequest (unchanged)
type DeactivateStudentRequest struct {
	Reason string `json:"reason" binding:"required,oneof=transferred expelled graduated deceased inactive"`
}

// UpdateStudentByTeacherRequest (unchanged)
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

// StudentListResponse (unchanged)
type StudentListResponse struct {
	ID          string    `json:"id"`
	AdmissionNo string    `json:"admission_no"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Username    string    `json:"username"`
	Gender      string    `json:"gender"`
	IsActive    bool      `json:"is_active"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}



// package dto

// import "time"

// // CreateStudentByTeacherRequest is used when a teacher creates a single student
// type CreateStudentByTeacherRequest struct {
// 	SchoolID      string     `json:"school_id" binding:"required,uuid"`
// 	ClassID       string     `json:"class_id" binding:"required,uuid"`
// 	FirstName     string     `json:"first_name" binding:"required"`
// 	LastName      string     `json:"last_name" binding:"required"`
// 	Username      string     `json:"username"` // optional – auto from admission_no or first+last
// 	AdmissionNo   string     `json:"admission_no" binding:"required"`
// 	DateOfBirth   *time.Time `json:"date_of_birth"`
// 	Gender        string     `json:"gender" binding:"omitempty,oneof=Male Female Other"`
// 	Address       string     `json:"address"`
// 	GuardianName  string     `json:"guardian_name"`
// 	GuardianPhone string     `json:"guardian_phone"`
// 	GuardianEmail string     `json:"guardian_email" binding:"omitempty,email"`
// }

// // CreateStudentByTeacherResponse returns the created student and generated password
// type CreateStudentByTeacherResponse struct {
// 	StudentID      string    `json:"student_id"`
// 	UserID         string    `json:"user_id"`
// 	Username       string    `json:"username"`
// 	GeneratedPassword string `json:"generated_password"`
// 	AdmissionNo    string    `json:"admission_no"`
// 	FirstName      string    `json:"first_name"`
// 	LastName       string    `json:"last_name"`
// }

// // ResetStudentPasswordResponse returns new credentials
// type ResetStudentPasswordResponse struct {
// 	Username string `json:"username"`
// 	Password string `json:"new_password"`
// }

// // DeactivateStudentRequest for changing student status
// type DeactivateStudentRequest struct {
// 	Reason string `json:"reason" binding:"required,oneof=transferred expelled graduated deceased inactive"`
// }

// // UpdateStudentByTeacherRequest (teacher can update limited fields)
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

// // StudentListResponse for teacher's class
// type StudentListResponse struct {
// 	ID           string     `json:"id"`
// 	AdmissionNo  string     `json:"admission_no"`
// 	FirstName    string     `json:"first_name"`
// 	LastName     string     `json:"last_name"`
// 	Username     string     `json:"username"`
// 	Gender       string     `json:"gender"`
// 	IsActive     bool       `json:"is_active"`
// 	Status       string     `json:"status"`
// 	CreatedAt    time.Time  `json:"created_at"`
// }