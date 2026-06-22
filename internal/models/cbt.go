package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================
// STRUCTURAL & LOOKUP TYPES
// ============================================

type DifficultyLevel string
type BloomTaxonomy string
type QuestionType string
type QuestionStatus string

const (
	DifficultyEasy   DifficultyLevel = "easy"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHard   DifficultyLevel = "hard"
	DifficultyExpert DifficultyLevel = "expert"

	BloomRemember   BloomTaxonomy = "remember"
	BloomUnderstand BloomTaxonomy = "understand"
	BloomApply      BloomTaxonomy = "apply"
	BloomAnalyse    BloomTaxonomy = "analyse"
	BloomEvaluate   BloomTaxonomy = "evaluate"
	BloomCreate     BloomTaxonomy = "create"

	QuestionTypeSingle    QuestionType = "single_choice"
	QuestionTypeMultiple  QuestionType = "multiple_choice"
	QuestionTypeTrueFalse QuestionType = "true_false"
	QuestionTypeEssay     QuestionType = "essay"
	QuestionTypeFillBlank QuestionType = "fill_blank"

	QuestionStatusDraft     QuestionStatus = "draft"
	QuestionStatusPublished QuestionStatus = "published"
	QuestionStatusArchived  QuestionStatus = "archived"
)

// ============================================
// JSON STORAGE TYPES - FIXED
// ============================================

// JSONMap for general object storage
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil || len(j) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONMap: expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*j = JSONMap{}
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// JSONArray for proper array storage - FIXED
type JSONArray []interface{}

func (j JSONArray) Value() (driver.Value, error) {
	if j == nil || len(j) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(j)
}

func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = JSONArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONArray: expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*j = JSONArray{}
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// OptionStorage for question options - FIXED
type OptionStorage []OptionItem

type OptionItem struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

func (o OptionStorage) Value() (driver.Value, error) {
	if o == nil || len(o) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(o)
}

func (o *OptionStorage) Scan(value interface{}) error {
	if value == nil {
		*o = OptionStorage{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan OptionStorage: expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*o = OptionStorage{}
		return nil
	}
	return json.Unmarshal(bytes, o)
}

// RubricStorage for rubric criteria - FIXED
type RubricStorage []RubricItem

type RubricItem struct {
	Criteria string `json:"criteria"`
	Marks    int    `json:"marks"`
}

func (r RubricStorage) Value() (driver.Value, error) {
	if r == nil || len(r) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(r)
}

func (r *RubricStorage) Scan(value interface{}) error {
	if value == nil {
		*r = RubricStorage{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan RubricStorage: expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*r = RubricStorage{}
		return nil
	}
	return json.Unmarshal(bytes, r)
}

// TagStorage for tags - FIXED
type TagStorage []string

func (t TagStorage) Value() (driver.Value, error) {
	if t == nil || len(t) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(t)
}

func (t *TagStorage) Scan(value interface{}) error {
	if value == nil {
		*t = TagStorage{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan TagStorage: expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*t = TagStorage{}
		return nil
	}
	return json.Unmarshal(bytes, t)
}

// ============================================
// CORE ACADEMIC & ENGINE CONFIG MODELS
// ============================================

type Subject struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Code        string         `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Description string         `gorm:"type:text" json:"description"`
	IsActive    bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Exam struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title            string         `gorm:"type:varchar(255);not null" json:"title"`
	SubjectID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"subject_id"`
	ClassID          *uuid.UUID     `gorm:"type:uuid;index" json:"class_id"`
	DurationMinutes  int            `gorm:"type:integer;not null" json:"duration_minutes"`
	TotalMarks       int            `gorm:"type:integer;not null" json:"total_marks"`
	PassMark         int            `gorm:"type:integer" json:"pass_mark"`
	Instructions     string         `gorm:"type:text" json:"instructions"`
	StartTime        *time.Time     `json:"start_time"`
	EndTime          *time.Time     `json:"end_time"`
	ShuffleQuestions bool           `gorm:"type:boolean;default:false" json:"shuffle_questions"`
	ShuffleOptions   bool           `gorm:"type:boolean;default:false" json:"shuffle_options"`
	IsActive         bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// ============================================
// LIVE COMPUTER BASED TESTING (CBT) QUESTION INSTANCE
// ============================================

type Question struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExamID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
	QuestionText  string         `gorm:"type:text;not null" json:"question_text"`
	QuestionType  QuestionType   `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
	Explanation   string         `gorm:"type:text" json:"explanation"`
	Marks         int            `gorm:"type:integer;default:1" json:"marks"`
	NegativeMarks float64        `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
	SortOrder     int            `gorm:"type:integer;default:0" json:"sort_order"`
	IsRequired    bool           `gorm:"type:boolean;default:false" json:"is_required"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Legacy Flat Fallbacks (kept strictly for full backward compatibility)
	OptionA       string `gorm:"type:text" json:"option_a,omitempty"`
	OptionB       string `gorm:"type:text" json:"option_b,omitempty"`
	OptionC       string `gorm:"type:text" json:"option_c,omitempty"`
	OptionD       string `gorm:"type:text" json:"option_d,omitempty"`
	CorrectAnswer string `gorm:"type:varchar(50)" json:"correct_answer,omitempty"`

	// New Modern Array Formats - FIXED
	Options           OptionStorage  `gorm:"type:jsonb;default:'[]'" json:"options"`
	CorrectOptionKeys []string       `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`
	Rubric            RubricStorage  `gorm:"type:jsonb;default:'[]'" json:"rubric"`
}

// ============================================
// STUDENT ASSESSMENT SESSION TRACKING
// ============================================

type ExamAttempt struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
	StartTime  time.Time      `json:"start_time"`
	EndTime    *time.Time     `json:"end_time"`
	Score      *int           `gorm:"type:integer" json:"score"`
	Percentage *float64       `gorm:"type:numeric(5,2)" json:"percentage"`
	Status     string         `gorm:"type:varchar(50);default:'in_progress'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type StudentAnswer struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"attempt_id"`
	QuestionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"question_id"`
	IsCorrect  bool           `gorm:"type:boolean;default:false" json:"is_correct"`
	TimeSpent  int            `gorm:"type:integer" json:"time_spent"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	SelectedAnswer     string   `gorm:"type:text" json:"selected_answer"`
	SelectedOptionKeys []string `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
	EssayResponse      string   `gorm:"type:text" json:"essay_response,omitempty"`
}

type Result struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
	TotalScore  int            `gorm:"type:integer" json:"total_score"`
	Percentage  float64        `gorm:"type:numeric(5,2)" json:"percentage"`
	Grade       string         `gorm:"type:varchar(10)" json:"grade"`
	Remarks     string         `gorm:"type:text" json:"remarks"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ============================================
// SELF-PACED PRACTICE & PREPARATION SESSIONS
// ============================================

type PracticeSession struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
	SubjectID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"subject_id"`
	QuestionIDs    JSONArray  `gorm:"type:jsonb;default:'[]'" json:"question_ids"`
	TotalQuestions int        `gorm:"type:integer;default:0" json:"total_questions"`
	Answered       int        `gorm:"type:integer;default:0" json:"answered"`
	Score          int        `gorm:"type:integer;default:0" json:"score"`
	Status         string     `gorm:"type:varchar(50);default:'in_progress'" json:"status"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ============================================
// AI & OFFLINE-FIRST SYNCHRONIZATION ENGINES
// ============================================

type OfflineAnswer struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID             uuid.UUID  `gorm:"type:uuid;not null;index" json:"exam_id"`
	QuestionID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"question_id"`
	SelectedAnswer     string     `gorm:"type:text" json:"selected_answer"`
	SelectedOptionKeys []string   `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
	DeviceID           string     `gorm:"type:varchar(255)" json:"device_id"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type BulkImportJob struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName         string     `gorm:"type:varchar(255)" json:"file_name"`
	FileType         string     `gorm:"type:varchar(50)" json:"file_type"`
	TotalRecords     int        `gorm:"type:integer" json:"total_records"`
	ProcessedRecords int        `gorm:"type:integer" json:"processed_records"`
	FailedRecords    int        `gorm:"type:integer" json:"failed_records"`
	Status           string     `gorm:"type:varchar(50);default:'pending'" json:"status"`
	Errors           JSONMap    `gorm:"type:jsonb;default:'{}'" json:"errors"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type AIQuestionGenerationJob struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID             uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	SubjectID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"subject_id"`
	Topic              string          `gorm:"type:varchar(255)" json:"topic"`
	NumberOfQuestions  int             `gorm:"type:integer" json:"number_of_questions"`
	Difficulty         DifficultyLevel `gorm:"type:varchar(50)" json:"difficulty"`
	BloomLevel         BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
	SourceText         string          `gorm:"type:text" json:"source_text,omitempty"`
	GeneratedQuestions JSONArray       `gorm:"type:jsonb;default:'[]'" json:"generated_questions"`
	Status             string          `gorm:"type:varchar(50);default:'pending'" json:"status"`
	ErrorMessage       string          `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
}

// ============================================
// PROCTORING & SECURITY VERIFICATION
// ============================================

type ProctoringSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID uuid.UUID  `gorm:"type:uuid;not null;index" json:"attempt_id"`
	StudentID uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
	Status    string     `gorm:"type:varchar(50);default:'active'" json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ProctoringViolation struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProctoringID  uuid.UUID `gorm:"type:uuid;not null;index" json:"proctoring_id"`
	AttemptID     uuid.UUID `gorm:"type:uuid;not null;index" json:"attempt_id"`
	ViolationType string    `gorm:"type:varchar(100)" json:"violation_type"`
	Severity      string    `gorm:"type:varchar(50);default:'warning'" json:"severity"`
	Details       string    `gorm:"type:text" json:"details"`
	Timestamp     time.Time `json:"timestamp"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================
// ASSIGNMENT, STRUCTURAL BANKS & TAG MAPPINGS
// ============================================

type ExamAssignment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExamID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"exam_id"`
	ClassID         *uuid.UUID `gorm:"type:uuid;index" json:"class_id"`
	StudentID       *uuid.UUID `gorm:"type:uuid;index" json:"student_id"`
	AssignedBy      uuid.UUID  `gorm:"type:uuid;not null" json:"assigned_by"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	AttemptsAllowed int        `gorm:"type:integer;default:1" json:"attempts_allowed"`
	Status          string     `gorm:"type:varchar(50);default:'active'" json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ============================================
// QUESTION BANK - FIXED
// ============================================

type QuestionBank struct {
	// ---------- PRIMARY IDENTIFIERS ----------
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// ---------- CONTEXTUAL IDENTIFIERS ----------
	SchoolID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_school_session_external,unique" json:"school_id"`
	ClassLevelID uuid.UUID  `gorm:"type:uuid;not null;index" json:"class_level_id"`
	ClassID      *uuid.UUID `gorm:"type:uuid;index" json:"class_id,omitempty"`
	SessionID    *uuid.UUID `gorm:"type:uuid;index:idx_school_session_external,unique" json:"session_id,omitempty"`
	TermID       *uuid.UUID `gorm:"type:uuid;index" json:"term_id,omitempty"`
	SubjectID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"subject_id"`

	// ---------- QUESTION CONTENT ----------
	QuestionText      string          `gorm:"type:text;not null" json:"question_text"`
	QuestionType      QuestionType    `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
	Difficulty        DifficultyLevel `gorm:"type:varchar(50);default:'medium'" json:"difficulty"`
	BloomLevel        BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
	Topic             string          `gorm:"type:varchar(255);index" json:"topic"`
	SubTopic          string          `gorm:"type:varchar(255)" json:"sub_topic"`
	LearningObjective string          `gorm:"type:text" json:"learning_objective"`

	// ---------- FIXED JSON FIELDS ----------
	Options           OptionStorage  `gorm:"type:jsonb;default:'[]'" json:"options"`
	// CorrectOptionKeys []string       `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`
	CorrectOptionKeys []string `gorm:"type:jsonb;serializer:json;default:'[]'" json:"correct_option_keys"`
	CorrectAnswer     string         `gorm:"type:text" json:"correct_answer"` // Legacy field
	Rubric            RubricStorage  `gorm:"type:jsonb;default:'[]'" json:"rubric"`
	Tags              TagStorage     `gorm:"type:jsonb;default:'[]'" json:"tags"`
	Explanation       string         `gorm:"type:text" json:"explanation"`

	// ---------- SCORING & METADATA ----------
	Marks            int     `gorm:"type:integer;default:1" json:"marks"`
	NegativeMarks    float64 `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
	TimeLimitSeconds *int    `gorm:"type:integer" json:"time_limit_seconds"`
	Order            int     `gorm:"type:integer;default:0" json:"order"`
	IsRequired       bool    `gorm:"type:boolean;default:false" json:"is_required"`

	// ---------- VERSIONING & STATUS ----------
	Status     QuestionStatus `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Version    int            `gorm:"type:integer;default:1" json:"version"`
	ParentID   *uuid.UUID     `gorm:"type:uuid" json:"parent_id,omitempty"`
	CreatedBy  uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy  uuid.UUID      `gorm:"type:uuid" json:"updated_by"`
	UsageCount int            `gorm:"type:integer;default:0" json:"usage_count"`
	SuccessRate *float64      `gorm:"type:numeric(5,2)" json:"success_rate,omitempty"`

	// ---------- IMPORT TRACKING ----------
	CurriculumType string `gorm:"type:varchar(50)" json:"curriculum_type"`
	SourceType     string `gorm:"type:varchar(50);default:'manual'" json:"source_type"`
	ExternalID     string `gorm:"type:varchar(255);index:idx_school_session_external,unique" json:"external_id,omitempty"`
	Metadata       JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

// ============================================
// TAG MODELS
// ============================================

type Tag struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Slug        string         `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	UsageCount  int            `gorm:"type:integer;default:0" json:"usage_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// QuestionTag – kept for backward compatibility
type QuestionTag struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	UsageCount  int       `gorm:"type:integer;default:0" json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type QuestionBankAttachment struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID uuid.UUID `gorm:"type:uuid;not null;index" json:"question_id"`
	FileName   string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FileType   string    `gorm:"type:varchar(100)" json:"file_type"`
	FileURL    string    `gorm:"type:text" json:"file_url"`
	FileSize   int64     `gorm:"type:bigint" json:"file_size"`
	CreatedAt  time.Time `json:"created_at"`
}

type QuestionTagMapping struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID uuid.UUID `gorm:"type:uuid;not null;index:idx_question_tag" json:"question_id"`
	TagID      uuid.UUID `gorm:"type:uuid;not null;index:idx_question_tag" json:"tag_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type ExamQuestion struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExamID     uuid.UUID `gorm:"type:uuid;not null;index:idx_exam_question" json:"exam_id"`
	QuestionID uuid.UUID `gorm:"type:uuid;not null;index:idx_exam_question" json:"question_id"`
	SortOrder  int       `gorm:"type:integer;default:0" json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

// ============================================
// EXPLICIT GORM TABLE MAPPINGS
// ============================================

func (QuestionBank) TableName() string            { return "question_bank" }
func (QuestionTag) TableName() string             { return "question_tags" }
func (QuestionBankAttachment) TableName() string  { return "question_bank_attachments" }
func (BulkImportJob) TableName() string           { return "bulk_import_jobs" }
func (AIQuestionGenerationJob) TableName() string { return "ai_question_generation_jobs" }
func (Tag) TableName() string                     { return "tags" }
func (QuestionTagMapping) TableName() string      { return "question_tag_mappings" }
func (ExamQuestion) TableName() string            { return "exam_questions" }
func (Subject) TableName() string                 { return "subjects" }
func (Exam) TableName() string                    { return "exams" }
func (Question) TableName() string                { return "questions" }
func (ExamAttempt) TableName() string             { return "exam_attempts" }
func (StudentAnswer) TableName() string           { return "student_answers" }
func (Result) TableName() string                  { return "results" }
func (PracticeSession) TableName() string         { return "practice_sessions" }
func (ProctoringSession) TableName() string       { return "proctoring_sessions" }
func (ProctoringViolation) TableName() string     { return "proctoring_violations" }
func (OfflineAnswer) TableName() string           { return "offline_answers" }
func (ExamAssignment) TableName() string          { return "exam_assignments" }


// package models

// import (
// 	"database/sql/driver"
// 	"encoding/json"
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // ============================================
// // STRUCTURAL & LOOKUP TYPES
// // ============================================

// type DifficultyLevel string
// type BloomTaxonomy string
// type QuestionType string
// type QuestionStatus string

// const (
// 	DifficultyEasy   DifficultyLevel = "easy"
// 	DifficultyMedium DifficultyLevel = "medium"
// 	DifficultyHard   DifficultyLevel = "hard"
// 	DifficultyExpert DifficultyLevel = "expert"

// 	BloomRemember   BloomTaxonomy = "remember"
// 	BloomUnderstand BloomTaxonomy = "understand"
// 	BloomApply      BloomTaxonomy = "apply"
// 	BloomAnalyse    BloomTaxonomy = "analyse"
// 	BloomEvaluate   BloomTaxonomy = "evaluate"
// 	BloomCreate     BloomTaxonomy = "create"

// 	QuestionTypeSingle    QuestionType = "single_choice"
// 	QuestionTypeMultiple  QuestionType = "multiple_choice"
// 	QuestionTypeTrueFalse QuestionType = "true_false"
// 	QuestionTypeEssay     QuestionType = "essay"
// 	QuestionTypeFillBlank QuestionType = "fill_blank"

// 	QuestionStatusDraft     QuestionStatus = "draft"
// 	QuestionStatusPublished QuestionStatus = "published"
// 	QuestionStatusArchived  QuestionStatus = "archived"
// )

// // JSONMap handles high-performance JSONB database interactions cleanly.
// type JSONMap map[string]interface{}

// func (j JSONMap) Value() (driver.Value, error) {
// 	if j == nil {
// 		return nil, nil
// 	}
// 	return json.Marshal(j)
// }

// func (j *JSONMap) Scan(value interface{}) error {
// 	if value == nil {
// 		*j = make(JSONMap)
// 		return nil
// 	}
// 	bytes, ok := value.([]byte)
// 	if !ok {
// 		return nil
// 	}
// 	return json.Unmarshal(bytes, j)
// }

// // ============================================
// // CORE ACADEMIC & ENGINE CONFIG MODELS
// // ============================================

// type Subject struct {
// 	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	Name        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
// 	Code        string         `gorm:"type:varchar(50);uniqueIndex" json:"code"`
// 	Description string         `gorm:"type:text" json:"description"`
// 	IsActive    bool           `gorm:"type:boolean;default:true" json:"is_active"`
// 	CreatedAt   time.Time      `json:"created_at"`
// 	UpdatedAt   time.Time      `json:"updated_at"`
// 	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type Exam struct {
// 	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	Title            string         `gorm:"type:varchar(255);not null" json:"title"`
// 	SubjectID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"subject_id"`
// 	ClassID          *uuid.UUID     `gorm:"type:uuid;index" json:"class_id"`
// 	DurationMinutes  int            `gorm:"type:integer;not null" json:"duration_minutes"`
// 	TotalMarks       int            `gorm:"type:integer;not null" json:"total_marks"`
// 	PassMark         int            `gorm:"type:integer" json:"pass_mark"`
// 	Instructions     string         `gorm:"type:text" json:"instructions"`
// 	StartTime        *time.Time     `json:"start_time"`
// 	EndTime          *time.Time     `json:"end_time"`
// 	ShuffleQuestions bool           `gorm:"type:boolean;default:false" json:"shuffle_questions"`
// 	ShuffleOptions   bool           `gorm:"type:boolean;default:false" json:"shuffle_options"`
// 	IsActive         bool           `gorm:"type:boolean;default:true" json:"is_active"`
// 	CreatedAt        time.Time      `json:"created_at"`
// 	UpdatedAt        time.Time      `json:"updated_at"`
// 	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // ============================================
// // LIVE COMPUTER BASED TESTING (CBT) QUESTION INSTANCE
// // ============================================

// type Question struct {
// 	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	ExamID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
// 	QuestionText  string         `gorm:"type:text;not null" json:"question_text"`
// 	QuestionType  QuestionType   `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
// 	Explanation   string         `gorm:"type:text" json:"explanation"`
// 	Marks         int            `gorm:"type:integer;default:1" json:"marks"`
// 	NegativeMarks float64        `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
// 	SortOrder     int            `gorm:"type:integer;default:0" json:"sort_order"`
// 	IsRequired    bool           `gorm:"type:boolean;default:false" json:"is_required"`
// 	CreatedAt     time.Time      `json:"created_at"`
// 	UpdatedAt     time.Time      `json:"updated_at"`
// 	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

// 	// Legacy Flat Fallbacks (kept strictly for full backward compatibility)
// 	OptionA       string `gorm:"type:text" json:"option_a,omitempty"`
// 	OptionB       string `gorm:"type:text" json:"option_b,omitempty"`
// 	OptionC       string `gorm:"type:text" json:"option_c,omitempty"`
// 	OptionD       string `gorm:"type:text" json:"option_d,omitempty"`
// 	CorrectAnswer string `gorm:"type:varchar(50)" json:"correct_answer,omitempty"`

// 	// New Modern Array Formats
// 	Options           JSONMap  `gorm:"type:jsonb;default:'[]'" json:"options"` // []{key, text}
// 	CorrectOptionKeys []string `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`
// 	Rubric            JSONMap  `gorm:"type:jsonb;default:'[]'" json:"rubric"` // []{criteria, marks}
// }

// // ============================================
// // STUDENT ASSESSMENT SESSION TRACKING
// // ============================================

// type ExamAttempt struct {
// 	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	StudentID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"student_id"`
// 	ExamID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
// 	StartTime  time.Time      `json:"start_time"`
// 	EndTime    *time.Time     `json:"end_time"`
// 	Score      *int           `gorm:"type:integer" json:"score"`
// 	Percentage *float64       `gorm:"type:numeric(5,2)" json:"percentage"`
// 	Status     string         `gorm:"type:varchar(50);default:'in_progress'" json:"status"`
// 	CreatedAt  time.Time      `json:"created_at"`
// 	UpdatedAt  time.Time      `json:"updated_at"`
// 	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
// }

// type StudentAnswer struct {
// 	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	AttemptID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"attempt_id"`
// 	QuestionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"question_id"`
// 	IsCorrect  bool           `gorm:"type:boolean;default:false" json:"is_correct"`
// 	TimeSpent  int            `gorm:"type:integer" json:"time_spent"`
// 	CreatedAt  time.Time      `json:"created_at"`
// 	UpdatedAt  time.Time      `json:"updated_at"`
// 	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

// 	// Legacy + Array formats for flexible response tracking
// 	SelectedAnswer     string   `gorm:"type:text" json:"selected_answer"`
// 	SelectedOptionKeys []string `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
// 	EssayResponse      string   `gorm:"type:text" json:"essay_response,omitempty"`
// }

// type Result struct {
// 	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	StudentID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"student_id"`
// 	ExamID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"exam_id"`
// 	TotalScore  int            `gorm:"type:integer" json:"total_score"`
// 	Percentage  float64        `gorm:"type:numeric(5,2)" json:"percentage"`
// 	Grade       string         `gorm:"type:varchar(10)" json:"grade"`
// 	Remarks     string         `gorm:"type:text" json:"remarks"`
// 	PublishedAt *time.Time     `json:"published_at"`
// 	CreatedAt   time.Time      `json:"created_at"`
// 	UpdatedAt   time.Time      `json:"updated_at"`
// 	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // ============================================
// // SELF-PACED PRACTICE & PREPARATION SESSIONS
// // ============================================

// type PracticeSession struct {
// 	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	StudentID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
// 	SubjectID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"subject_id"`
// 	QuestionIDs    JSONMap    `gorm:"type:jsonb;default:'[]'" json:"question_ids"`
// 	TotalQuestions int        `gorm:"type:integer;default:0" json:"total_questions"`
// 	Answered       int        `gorm:"type:integer;default:0" json:"answered"`
// 	Score          int        `gorm:"type:integer;default:0" json:"score"`
// 	Status         string     `gorm:"type:varchar(50);default:'in_progress'" json:"status"`
// 	StartedAt      time.Time  `json:"started_at"`
// 	CompletedAt    *time.Time `json:"completed_at,omitempty"`
// 	CreatedAt      time.Time  `json:"created_at"`
// 	UpdatedAt      time.Time  `json:"updated_at"`
// }

// // ============================================
// // AI & OFFLINE-FIRST SYNCHRONIZATION ENGINES
// // ============================================

// type OfflineAnswer struct {
// 	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	StudentID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
// 	ExamID             uuid.UUID  `gorm:"type:uuid;not null;index" json:"exam_id"`
// 	QuestionID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"question_id"`
// 	SelectedAnswer     string     `gorm:"type:text" json:"selected_answer"`
// 	SelectedOptionKeys []string   `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
// 	DeviceID           string     `gorm:"type:varchar(255)" json:"device_id"`
// 	SyncedAt           *time.Time `json:"synced_at,omitempty"`
// 	CreatedAt          time.Time  `json:"created_at"`
// }

// type BulkImportJob struct {
// 	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	UserID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
// 	FileName         string     `gorm:"type:varchar(255)" json:"file_name"`
// 	FileType         string     `gorm:"type:varchar(50)" json:"file_type"`
// 	TotalRecords     int        `gorm:"type:integer" json:"total_records"`
// 	ProcessedRecords int        `gorm:"type:integer" json:"processed_records"`
// 	FailedRecords    int        `gorm:"type:integer" json:"failed_records"`
// 	Status           string     `gorm:"type:varchar(50);default:'pending'" json:"status"`
// 	Errors           JSONMap    `gorm:"type:jsonb;default:'{}'" json:"errors"`
// 	StartedAt        *time.Time `json:"started_at,omitempty"`
// 	CompletedAt      *time.Time `json:"completed_at,omitempty"`
// 	CreatedAt        time.Time  `json:"created_at"`
// 	UpdatedAt        time.Time  `json:"updated_at"`
// }

// type AIQuestionGenerationJob struct {
// 	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	UserID             uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
// 	SubjectID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"subject_id"`
// 	Topic              string          `gorm:"type:varchar(255)" json:"topic"`
// 	NumberOfQuestions  int             `gorm:"type:integer" json:"number_of_questions"`
// 	Difficulty         DifficultyLevel `gorm:"type:varchar(50)" json:"difficulty"`
// 	BloomLevel         BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
// 	SourceText         string          `gorm:"type:text" json:"source_text,omitempty"`
// 	GeneratedQuestions JSONMap         `gorm:"type:jsonb;default:'[]'" json:"generated_questions"`
// 	Status             string          `gorm:"type:varchar(50);default:'pending'" json:"status"`
// 	ErrorMessage       string          `gorm:"type:text" json:"error_message,omitempty"`
// 	CreatedAt          time.Time       `json:"created_at"`
// 	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
// }

// // ============================================
// // PROCTORING & SECURITY VERIFICATION
// // ============================================

// type ProctoringSession struct {
// 	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	AttemptID uuid.UUID  `gorm:"type:uuid;not null;index" json:"attempt_id"`
// 	StudentID uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
// 	Status    string     `gorm:"type:varchar(50);default:'active'" json:"status"`
// 	StartedAt time.Time  `json:"started_at"`
// 	EndedAt   *time.Time `json:"ended_at,omitempty"`
// 	CreatedAt time.Time  `json:"created_at"`
// 	UpdatedAt time.Time  `json:"updated_at"`
// }

// type ProctoringViolation struct {
// 	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	ProctoringID  uuid.UUID `gorm:"type:uuid;not null;index" json:"proctoring_id"`
// 	AttemptID     uuid.UUID `gorm:"type:uuid;not null;index" json:"attempt_id"`
// 	ViolationType string    `gorm:"type:varchar(100)" json:"violation_type"`
// 	Severity      string    `gorm:"type:varchar(50);default:'warning'" json:"severity"`
// 	Details       string    `gorm:"type:text" json:"details"`
// 	Timestamp     time.Time `json:"timestamp"`
// 	CreatedAt     time.Time `json:"created_at"`
// }

// // ============================================
// // ASSIGNMENT, STRUCTURAL BANKS & TAG MAPPINGS
// // ============================================

// type ExamAssignment struct {
// 	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	ExamID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"exam_id"`
// 	ClassID         *uuid.UUID `gorm:"type:uuid;index" json:"class_id"`
// 	StudentID       *uuid.UUID `gorm:"type:uuid;index" json:"student_id"`
// 	AssignedBy      uuid.UUID  `gorm:"type:uuid;not null" json:"assigned_by"`
// 	StartTime       *time.Time `json:"start_time,omitempty"`
// 	EndTime         *time.Time `json:"end_time,omitempty"`
// 	AttemptsAllowed int        `gorm:"type:integer;default:1" json:"attempts_allowed"`
// 	Status          string     `gorm:"type:varchar(50);default:'active'" json:"status"`
// 	CreatedAt       time.Time  `json:"created_at"`
// 	UpdatedAt       time.Time  `json:"updated_at"`
// }

// type QuestionBank struct {
// 	// ---------- EXISTING FIELDS (fully preserved) ----------
// 	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	SubjectID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"subject_id"`
// 	Topic            string          `gorm:"type:varchar(255);index" json:"topic"`
// 	SubTopic         string          `gorm:"type:varchar(255)" json:"sub_topic"`
// 	QuestionText     string          `gorm:"type:text;not null" json:"question_text"`
// 	QuestionType     QuestionType    `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
// 	Difficulty       DifficultyLevel `gorm:"type:varchar(50);default:'medium'" json:"difficulty"`
// 	BloomLevel       BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
// 	Options          JSONMap         `gorm:"type:jsonb;default:'[]'" json:"options"`
// 	CorrectAnswer    string          `gorm:"type:text" json:"correct_answer"`
// 	Explanation      string          `gorm:"type:text" json:"explanation"`
// 	Marks            int             `gorm:"type:integer;default:1" json:"marks"`
// 	TimeLimitSeconds *int            `gorm:"type:integer" json:"time_limit_seconds"`
// 	Tags             JSONMap         `gorm:"type:jsonb;default:'[]'" json:"tags"`
// 	Metadata         JSONMap         `gorm:"type:jsonb;default:'{}'" json:"metadata"`
// 	Status           QuestionStatus  `gorm:"type:varchar(50);default:'draft'" json:"status"`
// 	Version          int             `gorm:"type:integer;default:1" json:"version"`
// 	ParentID         *uuid.UUID      `gorm:"type:uuid" json:"parent_id,omitempty"`
// 	CreatedBy        uuid.UUID       `gorm:"type:uuid;not null" json:"created_by"`
// 	UpdatedBy        uuid.UUID       `gorm:"type:uuid" json:"updated_by"`
// 	UsageCount       int             `gorm:"type:integer;default:0" json:"usage_count"`
// 	SuccessRate      *float64        `gorm:"type:numeric(5,2)" json:"success_rate,omitempty"`
// 	CreatedAt        time.Time       `json:"created_at"`
// 	UpdatedAt        time.Time       `json:"updated_at"`
// 	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`

// 	// ---------- NEW FIELDS (Production Intake Architecture) ----------
// 	// COMPOSITE UNIQUE INDEX for idempotency (critical for bulk imports)
// 	SchoolID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_school_session_external,unique" json:"school_id"`
// 	ClassLevelID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"class_level_id"`
// 	ClassID           *uuid.UUID `gorm:"type:uuid;index" json:"class_id,omitempty"`
// 	SessionID         *uuid.UUID `gorm:"type:uuid;index:idx_school_session_external,unique" json:"session_id,omitempty"`
// 	TermID            *uuid.UUID `gorm:"type:uuid;index" json:"term_id,omitempty"`
// 	CurriculumType    string     `gorm:"type:varchar(50)" json:"curriculum_type"`
// 	SourceType        string     `gorm:"type:varchar(50);default:'manual'" json:"source_type"`
// 	ExternalID        string     `gorm:"type:varchar(255);index:idx_school_session_external,unique" json:"external_id,omitempty"`
// 	LearningObjective string     `gorm:"type:text" json:"learning_objective"`
// 	CorrectOptionKeys []string   `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`
// 	Rubric            JSONMap    `gorm:"type:jsonb;default:'[]'" json:"rubric"`
// 	NegativeMarks     float64    `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
// 	Order             int        `gorm:"type:integer;default:0" json:"order"`
// 	IsRequired        bool       `gorm:"type:boolean;default:false" json:"is_required"`
// }

// // type QuestionBank struct {
// // 	// ---------- EXISTING FIELDS (fully preserved) ----------
// // 	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// // 	SubjectID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"subject_id"`
// // 	Topic            string          `gorm:"type:varchar(255);index" json:"topic"`
// // 	SubTopic         string          `gorm:"type:varchar(255)" json:"sub_topic"`
// // 	QuestionText     string          `gorm:"type:text;not null" json:"question_text"`
// // 	QuestionType     QuestionType    `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
// // 	Difficulty       DifficultyLevel `gorm:"type:varchar(50);default:'medium'" json:"difficulty"`
// // 	BloomLevel       BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
// // 	Options          JSONMap         `gorm:"type:jsonb;default:'[]'" json:"options"`
// // 	CorrectAnswer    string          `gorm:"type:text" json:"correct_answer"`
// // 	Explanation      string          `gorm:"type:text" json:"explanation"`
// // 	Marks            int             `gorm:"type:integer;default:1" json:"marks"`
// // 	TimeLimitSeconds *int            `gorm:"type:integer" json:"time_limit_seconds"`
// // 	Tags             JSONMap         `gorm:"type:jsonb;default:'[]'" json:"tags"`
// // 	Metadata         JSONMap         `gorm:"type:jsonb;default:'{}'" json:"metadata"`
// // 	Status           QuestionStatus  `gorm:"type:varchar(50);default:'draft'" json:"status"`
// // 	Version          int             `gorm:"type:integer;default:1" json:"version"`
// // 	ParentID         *uuid.UUID      `gorm:"type:uuid" json:"parent_id,omitempty"`
// // 	CreatedBy        uuid.UUID       `gorm:"type:uuid;not null" json:"created_by"`
// // 	UpdatedBy        uuid.UUID       `gorm:"type:uuid" json:"updated_by"`
// // 	UsageCount       int             `gorm:"type:integer;default:0" json:"usage_count"`
// // 	SuccessRate      *float64        `gorm:"type:numeric(5,2)" json:"success_rate,omitempty"`
// // 	CreatedAt        time.Time       `json:"created_at"`
// // 	UpdatedAt        time.Time       `json:"updated_at"`
// // 	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`

// // 	// ---------- NEW FIELDS ----------
// // 	SchoolID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_school_session_external,unique" json:"school_id"`
// // 	ClassLevelID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"class_level_id"`
// // 	ClassID           *uuid.UUID `gorm:"type:uuid;index" json:"class_id,omitempty"`
// // 	SessionID         *uuid.UUID `gorm:"type:uuid;index:idx_school_session_external,unique" json:"session_id,omitempty"`
// // 	TermID            *uuid.UUID `gorm:"type:uuid;index" json:"term_id,omitempty"`
// // 	CurriculumType    string     `gorm:"type:varchar(50)" json:"curriculum_type"`
// // 	SourceType        string     `gorm:"type:varchar(50);default:'manual'" json:"source_type"`
// // 	ExternalID        string     `gorm:"type:varchar(255);index:idx_school_session_external,unique" json:"external_id,omitempty"`
// // 	LearningObjective string     `gorm:"type:text" json:"learning_objective"`
	
// // 	// ✅ FIXED: Added serializer:json tag
// // 	CorrectOptionKeys []string   `gorm:"type:jsonb;serializer:json;default:'[]'" json:"correct_option_keys"`
	
// // 	Rubric            JSONMap    `gorm:"type:jsonb;default:'[]'" json:"rubric"`
// // 	NegativeMarks     float64    `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
// // 	Order             int        `gorm:"type:integer;default:0" json:"order"`
// // 	IsRequired        bool       `gorm:"type:boolean;default:false" json:"is_required"`
// // }

// type Tag struct {
// 	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
// 	Slug        string         `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
// 	Description string         `gorm:"type:text" json:"description"`
// 	UsageCount  int            `gorm:"type:integer;default:0" json:"usage_count"`
// 	CreatedAt   time.Time      `json:"created_at"`
// 	UpdatedAt   time.Time      `json:"updated_at"`
// 	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// }

// // QuestionTag – kept for backward compatibility (used by some queries)
// type QuestionTag struct {
// 	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
// 	Slug        string    `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
// 	Description string    `gorm:"type:text" json:"description"`
// 	UsageCount  int       `gorm:"type:integer;default:0" json:"usage_count"`
// 	CreatedAt   time.Time `json:"created_at"`
// 	UpdatedAt   time.Time `json:"updated_at"`
// }

// type QuestionBankAttachment struct {
// 	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	QuestionID uuid.UUID `gorm:"type:uuid;not null;index" json:"question_id"`
// 	FileName   string    `gorm:"type:varchar(255);not null" json:"file_name"`
// 	FileType   string    `gorm:"type:varchar(100)" json:"file_type"`
// 	FileURL    string    `gorm:"type:text" json:"file_url"`
// 	FileSize   int64     `gorm:"type:bigint" json:"file_size"`
// 	CreatedAt  time.Time `json:"created_at"`
// }

// type QuestionTagMapping struct {
// 	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	QuestionID uuid.UUID `gorm:"type:uuid;not null;index:idx_question_tag" json:"question_id"`
// 	TagID      uuid.UUID `gorm:"type:uuid;not null;index:idx_question_tag" json:"tag_id"`
// 	CreatedAt  time.Time `json:"created_at"`
// }

// type ExamQuestion struct {
// 	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	ExamID     uuid.UUID `gorm:"type:uuid;not null;index:idx_exam_question" json:"exam_id"`
// 	QuestionID uuid.UUID `gorm:"type:uuid;not null;index:idx_exam_question" json:"question_id"`
// 	SortOrder  int       `gorm:"type:integer;default:0" json:"sort_order"`
// 	CreatedAt  time.Time `json:"created_at"`
// }

// // ============================================
// // EXPLICIT GORM TABLE MAPPINGS
// // ============================================

// func (QuestionBank) TableName() string            { return "question_bank" }
// func (QuestionTag) TableName() string             { return "question_tags" }
// func (QuestionBankAttachment) TableName() string  { return "question_bank_attachments" }
// func (BulkImportJob) TableName() string           { return "bulk_import_jobs" }
// func (AIQuestionGenerationJob) TableName() string { return "ai_question_generation_jobs" }
// func (Tag) TableName() string                     { return "tags" }
// func (QuestionTagMapping) TableName() string      { return "question_tag_mappings" }
// func (ExamQuestion) TableName() string            { return "exam_questions" }
// func (Subject) TableName() string                 { return "subjects" }
// func (Exam) TableName() string                    { return "exams" }
// func (Question) TableName() string                { return "questions" }
// func (ExamAttempt) TableName() string             { return "exam_attempts" }
// func (StudentAnswer) TableName() string           { return "student_answers" }
// func (Result) TableName() string                  { return "results" }
// func (PracticeSession) TableName() string         { return "practice_sessions" }
// func (ProctoringSession) TableName() string       { return "proctoring_sessions" }
// func (ProctoringViolation) TableName() string     { return "proctoring_violations" }
// func (OfflineAnswer) TableName() string           { return "offline_answers" }
// func (ExamAssignment) TableName() string          { return "exam_assignments" }

