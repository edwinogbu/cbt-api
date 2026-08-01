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
// UNIVERSAL ID HELPER
// ============================================

func GenerateID() string {
	return uuid.New().String()
}

func GenerateUUID() uuid.UUID {
	return uuid.New()
}

func ToUUID(id interface{}) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	switch v := id.(type) {
	case string:
		if v == "" {
			return uuid.Nil
		}
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil
		}
		return parsed
	case uuid.UUID:
		return v
	case *uuid.UUID:
		if v == nil {
			return uuid.Nil
		}
		return *v
	default:
		return uuid.Nil
	}
}

func ToString(id interface{}) string {
	if id == nil {
		return ""
	}
	switch v := id.(type) {
	case string:
		return v
	case uuid.UUID:
		return v.String()
	case *uuid.UUID:
		if v == nil {
			return ""
		}
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func IsValidUUID(id string) bool {
	if id == "" {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}

// ============================================
// STRUCTURAL & LOOKUP TYPES
// ============================================

type DifficultyLevel string
type BloomTaxonomy string
type QuestionType string
type QuestionStatus string
type ExamType string
type ExamStatus string
type AttemptStatus string
type ViolationType string

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

	ExamTypeWeeklyTest ExamType = "weekly_test"
	ExamTypeMidTerm    ExamType = "mid_term"
	ExamTypeMainExam   ExamType = "main_exam"
	ExamTypePractice   ExamType = "practice"
	ExamTypeQuiz       ExamType = "quiz"

	ExamStatusDraft     ExamStatus = "draft"
	ExamStatusPublished ExamStatus = "published"
	ExamStatusActive    ExamStatus = "active"
	ExamStatusCompleted ExamStatus = "completed"
	ExamStatusArchived  ExamStatus = "archived"

	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusCompleted  AttemptStatus = "completed"
	AttemptStatusSubmitted  AttemptStatus = "submitted"
	AttemptStatusGraded     AttemptStatus = "graded"

	ViolationTypeTabSwitch   ViolationType = "tab_switch"
	ViolationTypeCopyPaste   ViolationType = "copy_paste"
	ViolationTypeScreenshot  ViolationType = "screenshot"
	ViolationTypeFaceMissing ViolationType = "face_missing"
)

// ============================================
// JSON STORAGE TYPES
// ============================================

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

type ReviewPolicyStorage struct {
	ReleaseMode        string `json:"release_mode"`
	ShowScore          bool   `json:"show_score"`
	ShowGrade          bool   `json:"show_grade"`
	ShowCorrectAnswers bool   `json:"show_correct_answers"`
	ShowExplanations   bool   `json:"show_explanations"`
	ShowQuestionReview bool   `json:"show_question_review"`
	AllowRetake        bool   `json:"allow_retake"`
}

func (r ReviewPolicyStorage) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *ReviewPolicyStorage) Scan(value interface{}) error {
	if value == nil {
		*r = ReviewPolicyStorage{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ReviewPolicyStorage: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, r)
}

// ============================================
// CORE MODELS
// ============================================

type Subject struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Code        string         `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Description string         `gorm:"type:text" json:"description"`
	IsActive    bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Exam struct {
	ID               string              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title            string              `gorm:"type:varchar(255);not null" json:"title"`
	SubjectID        string              `gorm:"type:uuid;not null;index" json:"subject_id"`
	ClassID          string              `gorm:"type:uuid;not null;index" json:"class_id"`
	SessionID        string              `gorm:"type:uuid;not null;index" json:"session_id"`
	TermID           string              `gorm:"type:uuid;not null;index" json:"term_id"`
	ExamType         string              `gorm:"type:varchar(50);not null;default:'main_exam'" json:"exam_type"`
	Status           string              `gorm:"type:varchar(50);default:'draft'" json:"status"`
	DurationMinutes  int                 `gorm:"type:integer;not null" json:"duration_minutes"`
	TotalMarks       int                 `gorm:"type:integer;not null" json:"total_marks"`
	PassMark         int                 `gorm:"type:integer" json:"pass_mark"`
	Instructions     string              `gorm:"type:text" json:"instructions"`
	StartTime        *time.Time          `json:"start_time"`
	EndTime          *time.Time          `json:"end_time"`
	ShuffleQuestions bool                `gorm:"type:boolean;default:false" json:"shuffle_questions"`
	ShuffleOptions   bool                `gorm:"type:boolean;default:false" json:"shuffle_options"`
	IsActive         bool                `gorm:"type:boolean;default:true" json:"is_active"`
	ReviewPolicy     ReviewPolicyStorage `gorm:"type:jsonb" json:"review_policy"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	DeletedAt        gorm.DeletedAt      `gorm:"index" json:"-"`
}

type QuestionBank struct {
	ID                string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"-"`

	SchoolID          string          `gorm:"type:uuid;not null;index" json:"school_id"`
	SessionID         string          `gorm:"type:uuid;not null;index" json:"session_id"`
	TermID            string          `gorm:"type:uuid;not null;index" json:"term_id"`
	ClassLevelID      string          `gorm:"type:uuid;not null;index" json:"class_level_id"`
	ClassID           string          `gorm:"type:uuid;not null;index" json:"class_id"`
	SubjectID         string          `gorm:"type:uuid;not null;index" json:"subject_id"`
	ExamType          string          `gorm:"type:varchar(50);not null;index" json:"exam_type"`

	QuestionText      string          `gorm:"type:text;not null" json:"question_text"`
	QuestionType      QuestionType    `gorm:"type:varchar(50);default:'single_choice'" json:"question_type"`
	Difficulty        DifficultyLevel `gorm:"type:varchar(50);default:'medium'" json:"difficulty"`
	BloomLevel        BloomTaxonomy   `gorm:"type:varchar(50)" json:"bloom_level"`
	Topic             string          `gorm:"type:varchar(255);not null;index" json:"topic"`
	SubTopic          string          `gorm:"type:varchar(255)" json:"sub_topic"`
	LearningObjective string          `gorm:"type:text" json:"learning_objective"`

	// Options           OptionStorage   `gorm:"type:jsonb;default:'[]'" json:"options"`
	// CorrectOptionKeys []string        `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`
	// CorrectAnswer     string          `gorm:"type:text" json:"correct_answer"`
	// Rubric            RubricStorage   `gorm:"type:jsonb;default:'[]'" json:"rubric"`
	// Tags              TagStorage      `gorm:"type:jsonb;default:'[]'" json:"tags"`

	 Options           OptionStorage      `gorm:"type:jsonb;default:'[]'" json:"options"`
    CorrectOptionKeys CorrectOptionKeys  `gorm:"type:jsonb;default:'[]'" json:"correct_option_keys"`  // ✅ Changed from []string
    CorrectAnswer     string             `gorm:"type:text" json:"correct_answer"`
    Rubric            RubricStorage      `gorm:"type:jsonb;default:'[]'" json:"rubric"`
    Tags              TagStorage         `gorm:"type:jsonb;default:'[]'" json:"tags"`
    
	Explanation       string          `gorm:"type:text" json:"explanation"`

	Marks             int             `gorm:"type:integer;default:1" json:"marks"`
	NegativeMarks     float64         `gorm:"type:numeric(5,2);default:0" json:"negative_marks"`
	TimeLimitSeconds  *int            `gorm:"type:integer" json:"time_limit_seconds"`
	Order             int             `gorm:"type:integer;default:0" json:"order"`
	IsRequired        bool            `gorm:"type:boolean;default:false" json:"is_required"`

	Status            QuestionStatus  `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Version           int             `gorm:"type:integer;default:1" json:"version"`
	ParentID          *string         `gorm:"type:uuid" json:"parent_id,omitempty"`
	CreatedBy         string          `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy         string          `gorm:"type:uuid" json:"updated_by"`
	UsageCount        int             `gorm:"type:integer;default:0" json:"usage_count"`
	SuccessRate       *float64        `gorm:"type:numeric(5,2)" json:"success_rate,omitempty"`

	CurriculumType    string          `gorm:"type:varchar(50)" json:"curriculum_type"`
	SourceType        string          `gorm:"type:varchar(50);default:'manual'" json:"source_type"`
	ExternalID        string          `gorm:"type:varchar(255);index" json:"external_id,omitempty"`
	Metadata          JSONMap         `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

type ExamQuestion struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExamID     string    `gorm:"type:uuid;not null;index" json:"exam_id"`
	QuestionID string    `gorm:"type:uuid;not null;index" json:"question_id"`
	SortOrder  int       `gorm:"type:integer;default:0" json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

type ExamAttempt struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID  string         `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID     string         `gorm:"type:uuid;not null;index" json:"exam_id"`
	StartTime  time.Time      `json:"start_time"`
	EndTime    *time.Time     `json:"end_time"`
	Score      *int           `gorm:"type:integer" json:"score"`
	Percentage *float64       `gorm:"type:numeric(5,2)" json:"percentage"`
	Status     string         `gorm:"type:varchar(50);default:'in_progress'" json:"status"`
	DeviceInfo JSONMap        `gorm:"type:jsonb;default:'{}'" json:"device_info"`
	IPAddress  string         `gorm:"type:varchar(50)" json:"ip_address"`
	Signature  string         `gorm:"type:text" json:"signature"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}


type StringArray []string

func (s *StringArray) Scan(value interface{}) error {
    if value == nil {
        *s = StringArray{}
        return nil
    }
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("failed to scan StringArray: expected []byte, got %T", value)
    }
    if len(bytes) == 0 || string(bytes) == "null" {
        *s = StringArray{}
        return nil
    }
    return json.Unmarshal(bytes, s)
}

func (s StringArray) Value() (driver.Value, error) {
    if s == nil || len(s) == 0 {
        return []byte("[]"), nil
    }
    return json.Marshal(s)
}


type StudentAnswer struct {
	ID                 string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID          string         `gorm:"type:uuid;not null;index" json:"attempt_id"`
	QuestionID         string         `gorm:"type:uuid;not null;index" json:"question_id"`
	SelectedAnswer     string         `gorm:"type:text" json:"selected_answer"`
	// SelectedOptionKeys []string       `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
	SelectedOptionKeys StringArray `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
	EssayResponse      string         `gorm:"type:text" json:"essay_response,omitempty"`
	IsCorrect          bool           `gorm:"type:boolean;default:false" json:"is_correct"`
	IsMarked           bool           `gorm:"type:boolean;default:false" json:"is_marked"`
	TimeSpent          int            `gorm:"type:integer" json:"time_spent"`
	SyncedAt           *time.Time     `json:"synced_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

type Result struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID   string         `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID      string         `gorm:"type:uuid;not null;index" json:"exam_id"`
	AttemptID   string         `gorm:"type:uuid;not null;index" json:"attempt_id"`
	TotalScore  int            `gorm:"type:integer" json:"total_score"`
	Percentage  float64        `gorm:"type:numeric(5,2)" json:"percentage"`
	Grade       string         `gorm:"type:varchar(10)" json:"grade"`
	Remarks     string         `gorm:"type:text" json:"remarks"`
	Passed      bool       `gorm:"index"` // ✅ Make sure this field exists
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type PracticeSession struct {
	ID             string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID      string     `gorm:"type:uuid;not null;index" json:"student_id"`
	SubjectID      string     `gorm:"type:uuid;not null;index" json:"subject_id"`
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

type OfflineAnswer struct {
	ID                 string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID          string     `gorm:"type:uuid;not null;index" json:"student_id"`
	ExamID             string     `gorm:"type:uuid;not null;index" json:"exam_id"`
	AttemptID          string     `gorm:"type:uuid;not null;index" json:"attempt_id"`
	QuestionID         string     `gorm:"type:uuid;not null;index" json:"question_id"`
	SelectedAnswer     string     `gorm:"type:text" json:"selected_answer"`
	SelectedOptionKeys []string   `gorm:"type:jsonb;default:'[]'" json:"selected_option_keys"`
	IsMarked           bool       `gorm:"type:boolean;default:false" json:"is_marked"`
	TimeSpent          int        `gorm:"type:integer" json:"time_spent"`
	DeviceID           string     `gorm:"type:varchar(255)" json:"device_id"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ExamAssignment struct {
	ID              string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExamID          string     `gorm:"type:uuid;not null;index" json:"exam_id"`
	ClassID         *string    `gorm:"type:uuid;index" json:"class_id"`
	StudentID       *string    `gorm:"type:uuid;index" json:"student_id"`
	AssignedBy      string     `gorm:"type:uuid;not null" json:"assigned_by"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	AttemptsAllowed int        `gorm:"type:integer;default:1" json:"attempts_allowed"`
	Status          string     `gorm:"type:varchar(50);default:'active'" json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ProctoringSession struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID string     `gorm:"type:uuid;not null;index" json:"attempt_id"`
	StudentID string     `gorm:"type:uuid;not null;index" json:"student_id"`
	Status    string     `gorm:"type:varchar(50);default:'active'" json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ProctoringViolation struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProctoringID  string    `gorm:"type:uuid;not null;index" json:"proctoring_id"`
	AttemptID     string    `gorm:"type:uuid;not null;index" json:"attempt_id"`
	ViolationType string    `gorm:"type:varchar(100)" json:"violation_type"`
	Severity      string    `gorm:"type:varchar(50);default:'warning'" json:"severity"`
	Details       string    `gorm:"type:text" json:"details"`
	Timestamp     time.Time `json:"timestamp"`
	CreatedAt     time.Time `json:"created_at"`
}

// type Tag struct {
// 	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
// 	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
// 	Slug        string         `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
// 	Description string         `gorm:"type:text" json:"description"`
// 	UsageCount  int            `gorm:"type:integer;default:0" json:"usage_count"`
// 	CreatedAt   time.Time      `json:"created_at"`
// 	UpdatedAt   time.Time      `json:"updated_at"`
// 	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
// }
type Tag struct {
    ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Name        string         `gorm:"type:varchar(100);unique;not null" json:"name"`  // Use unique instead of uniqueIndex
    Slug        string         `gorm:"type:varchar(100);unique" json:"slug"`            // Use unique instead of uniqueIndex
    Description string         `gorm:"type:text" json:"description"`
    UsageCount  int            `gorm:"type:integer;default:0" json:"usage_count"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type QuestionTagMapping struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID string `gorm:"type:uuid;not null;index" json:"question_id"`
	TagID      string `gorm:"type:uuid;not null;index" json:"tag_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type QuestionBankAttachment struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID string `gorm:"type:uuid;not null;index" json:"question_id"`
	FileName   string `gorm:"type:varchar(255);not null" json:"file_name"`
	FileType   string `gorm:"type:varchar(100)" json:"file_type"`
	FileURL    string `gorm:"type:text" json:"file_url"`
	FileSize   int64  `gorm:"type:bigint" json:"file_size"`
	CreatedAt  time.Time `json:"created_at"`
}

type BulkImportJob struct {
	ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID           string         `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName         string         `gorm:"type:varchar(255)" json:"file_name"`
	FileType         string         `gorm:"type:varchar(50)" json:"file_type"`
	TotalRecords     int            `gorm:"type:integer" json:"total_records"`
	ProcessedRecords int            `gorm:"type:integer" json:"processed_records"`
	FailedRecords    int            `gorm:"type:integer" json:"failed_records"`
	Status           string         `gorm:"type:varchar(50);default:'pending'" json:"status"`
	Errors           JSONMap        `gorm:"type:jsonb;default:'{}'" json:"errors"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type AIQuestionGenerationJob struct {
	ID                 string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID             string          `gorm:"type:uuid;not null;index" json:"user_id"`
	SubjectID          string          `gorm:"type:uuid;not null;index" json:"subject_id"`
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
// CORRECT OPTION KEYS - JSON STORAGE TYPE
// ============================================

type CorrectOptionKeys []string

func (c CorrectOptionKeys) Value() (driver.Value, error) {
    if c == nil || len(c) == 0 {
        return []byte("[]"), nil
    }
    return json.Marshal(c)
}

func (c *CorrectOptionKeys) Scan(value interface{}) error {
    if value == nil {
        *c = CorrectOptionKeys{}
        return nil
    }
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("failed to scan CorrectOptionKeys: expected []byte, got %T", value)
    }
    if len(bytes) == 0 || string(bytes) == "null" {
        *c = CorrectOptionKeys{}
        return nil
    }
    return json.Unmarshal(bytes, c)
}

// ============================================
// TABLE MAPPINGS
// ============================================

func (Subject) TableName() string                    { return "subjects" }
func (Exam) TableName() string                      { return "exams" }
func (QuestionBank) TableName() string              { return "question_bank" }
func (ExamQuestion) TableName() string              { return "exam_questions" }
func (ExamAttempt) TableName() string               { return "exam_attempts" }
func (StudentAnswer) TableName() string             { return "student_answers" }
func (Result) TableName() string                    { return "results" }
func (PracticeSession) TableName() string           { return "practice_sessions" }
func (OfflineAnswer) TableName() string             { return "offline_answers" }
func (ExamAssignment) TableName() string            { return "exam_assignments" }
func (ProctoringSession) TableName() string         { return "proctoring_sessions" }
func (ProctoringViolation) TableName() string       { return "proctoring_violations" }
func (Tag) TableName() string                       { return "tags" }
func (QuestionTagMapping) TableName() string        { return "question_tag_mappings" }
func (QuestionBankAttachment) TableName() string    { return "question_bank_attachments" }
func (BulkImportJob) TableName() string             { return "bulk_import_jobs" }
func (AIQuestionGenerationJob) TableName() string   { return "ai_question_generation_jobs" }
