package dto

import "time"

// ============================================================
// PUSH: School Node -> Cloud
// (School->Cloud direction, per the offline-first plan's rule #8:
// sessions/submissions/results/audit only - never schools/teachers/
// students/exams/questions/schedules, which only ever flow Cloud->School.)
// ============================================================

type PushRequest struct {
	Sessions []SessionPush `json:"sessions"`
	Answers  []AnswerPush  `json:"answers"`
	Results  []ResultPush  `json:"results"`
	Events   []EventPush   `json:"events"`
}

// SessionPush mirrors models.ExamAttempt. ID is the node-generated UUID
// primary key and doubles as the idempotency key for upsert - a retried
// push of the same session is a no-op, not a duplicate.
type SessionPush struct {
	ID         string     `json:"id" binding:"required,uuid"`
	StudentID  string     `json:"student_id" binding:"required,uuid"`
	ExamID     string     `json:"exam_id" binding:"required,uuid"`
	StartTime  time.Time  `json:"start_time" binding:"required"`
	EndTime    *time.Time `json:"end_time"`
	Score      *int       `json:"score"`
	Percentage *float64   `json:"percentage"`
	Status     string     `json:"status" binding:"required"`
	IPAddress  string     `json:"ip_address"`
}

// AnswerPush mirrors models.StudentAnswer.
type AnswerPush struct {
	ID             string `json:"id" binding:"required,uuid"`
	AttemptID      string `json:"attempt_id" binding:"required,uuid"`
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	SelectedAnswer string `json:"selected_answer"`
	IsCorrect      bool   `json:"is_correct"`
	IsMarked       bool   `json:"is_marked"`
	TimeSpent      int    `json:"time_spent"`
}

// ResultPush mirrors models.Result.
type ResultPush struct {
	ID          string     `json:"id" binding:"required,uuid"`
	StudentID   string     `json:"student_id" binding:"required,uuid"`
	ExamID      string     `json:"exam_id" binding:"required,uuid"`
	AttemptID   string     `json:"attempt_id" binding:"required,uuid"`
	TotalScore  int        `json:"total_score"`
	Percentage  float64    `json:"percentage"`
	Grade       string     `json:"grade"`
	Remarks     string     `json:"remarks"`
	Passed      bool       `json:"passed"`
	PublishedAt *time.Time `json:"published_at"`
}

// EventPush mirrors models.ExamEventLog (the audit trail).
type EventPush struct {
	ID              string `json:"id" binding:"required,uuid"`
	AttemptID       string `json:"attempt_id" binding:"required,uuid"`
	StudentID       string `json:"student_id" binding:"required,uuid"`
	EventType       string `json:"event_type" binding:"required"`
	ClientSequence  int    `json:"client_sequence"`
	PayloadJSON     string `json:"payload_json"`
	ClientTimestamp string `json:"client_timestamp"`
}

// PushResponse reports what happened to every record in the batch -
// nothing is silently dropped. A record absent from Rejected and not
// counted in *Accepted did not happen; the handler guarantees every
// input ID appears in exactly one of the two.
type PushResponse struct {
	SessionsAccepted int             `json:"sessions_accepted"`
	AnswersAccepted  int             `json:"answers_accepted"`
	ResultsAccepted  int             `json:"results_accepted"`
	EventsAccepted   int             `json:"events_accepted"`
	Rejected         []RejectedPush  `json:"rejected,omitempty"`
	ServerTimestamp  time.Time       `json:"server_timestamp"`
}

type RejectedPush struct {
	Type   string `json:"type"` // "session" | "answer" | "result" | "event"
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ============================================================
// PULL: Cloud -> School Node
// (Cloud->School direction per rule #8. Scoped in this pass to students
// and exams+questions - the two entity types a node needs to actually
// run exams offline. Teachers/schedules-as-their-own-concern are a
// follow-up using this same cursor pattern, not implemented here.)
// ============================================================

type PullResponse struct {
	// Apply order matters - see client.pull(): ClassLevels/ClassArms before
	// Classes (classes.class_level_id/class_arm_id FKs), Classes and Users
	// before Students (students.class_id/user_id FKs). Only the rows
	// actually referenced by the pulled students are included (not a
	// general academic-structure/staff-directory sync - out of scope for
	// this pass; a node's first-ever provisioning still needs a full
	// migrate+seed, per README-SCHOOL-NODE.md).
	ClassLevels     []ClassLevelPull `json:"class_levels"`
	ClassArms       []ClassArmPull   `json:"class_arms"`
	Classes         []ClassPull      `json:"classes"`
	Users           []UserPull       `json:"users"`
	Students        []StudentPull    `json:"students"`
	Exams           []ExamPull       `json:"exams"`
	Cursor          time.Time        `json:"cursor"`
	ServerTimestamp time.Time        `json:"server_timestamp"`
}

type ClassLevelPull struct {
	ID          string `json:"id"`
	SchoolID    string `json:"school_id"`
	Name        string `json:"name"`
	LevelNumber int    `json:"level_number"`
	Category    string `json:"category"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

type ClassArmPull struct {
	ID         string `json:"id"`
	SchoolID   string `json:"school_id"`
	Name       string `json:"name"`
	ArmCode    string `json:"arm_code"`
	Capacity   int    `json:"capacity"`
	SortOrder  int    `json:"sort_order"`
	IsActive   bool   `json:"is_active"`
}

type ClassPull struct {
	ID           string `json:"id"`
	SchoolID     string `json:"school_id"`
	SessionID    string `json:"session_id"`
	ClassLevelID string `json:"class_level_id"`
	ClassArmID   string `json:"class_arm_id"`
	ClassCode    string `json:"class_code"`
	IsActive     bool   `json:"is_active"`
}

type UserPull struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Email         *string `json:"email"`
	PasswordHash  string  `json:"password_hash"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	PhoneNumber   string  `json:"phone_number"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	EmailVerified bool    `json:"email_verified"`
	IsActive      bool    `json:"is_active"`
	SchoolID      *string `json:"school_id,omitempty"`
}

type StudentPull struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	SchoolID    string  `json:"school_id"`
	ClassID     string  `json:"class_id"`
	AdmissionNo string  `json:"admission_no"`
	FullName    string  `json:"full_name"`
	Email       string  `json:"email"`
	IsActive    bool    `json:"is_active"`
	Status      string  `json:"status"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

type ExamPull struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	SubjectID        string           `json:"subject_id"`
	ClassID          string           `json:"class_id"`
	// SessionID/TermID: same reasoning as QuestionPull above - Exam's
	// session_id/term_id are NOT NULL uuid with no DB default.
	SessionID        string           `json:"session_id"`
	TermID           string           `json:"term_id"`
	ExamType         string           `json:"exam_type"`
	Status           string           `json:"status"`
	DurationMinutes  int              `json:"duration_minutes"`
	TotalMarks       int              `json:"total_marks"`
	PassMark         int              `json:"pass_mark"`
	Instructions     string           `json:"instructions"`
	StartTime        *time.Time       `json:"start_time"`
	EndTime          *time.Time       `json:"end_time"`
	ShuffleQuestions bool             `json:"shuffle_questions"`
	ShuffleOptions   bool             `json:"shuffle_options"`
	IsActive         bool             `json:"is_active"`
	Questions        []QuestionPull   `json:"questions"`
	DeletedAt        *string          `json:"deleted_at,omitempty"`
}

type QuestionPull struct {
	ID           string `json:"id"`
	// SchoolID/SessionID/TermID/ClassLevelID/SubjectID/ExamType/Topic are
	// carried straight through from the source question_bank row - all
	// NOT NULL uuid/varchar columns on that table with no DB-level
	// default, so a node applying this as a brand new local row needs
	// real values for every one of them, not just the fields a question
	// display actually needs.
	SchoolID      string `json:"school_id"`
	SessionID     string `json:"session_id"`
	TermID        string `json:"term_id"`
	ClassLevelID  string `json:"class_level_id"`
	ClassID       string `json:"class_id"`
	SubjectID     string `json:"subject_id"`
	ExamType      string `json:"exam_type"`
	Topic         string `json:"topic"`
	// CreatedBy: question_bank.created_by is NOT NULL uuid with no
	// default - carried straight through from source for the same reason
	// as the fields above.
	CreatedBy     string `json:"created_by"`
	QuestionText string `json:"question_text"`
	OptionA      string `json:"option_a"`
	OptionB      string `json:"option_b"`
	OptionC      string `json:"option_c"`
	OptionD      string `json:"option_d"`
	CorrectOption string `json:"correct_option"`
	Marks        int    `json:"marks"`
	SortOrder    int    `json:"sort_order"`
}
