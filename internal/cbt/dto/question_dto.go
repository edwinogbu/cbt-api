package dto

import (
	"mime/multipart"
	"time"
)

// ============================================
// REQUEST DTOs - FIXED
// ============================================

// CreateQuestionRequest represents the request to create a new question
type CreateQuestionRequest struct {
	// Required fields
	SubjectID    string `json:"subject_id" binding:"required,uuid"`
	QuestionText string `json:"question_text" binding:"required"`
	QuestionType string `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
	Difficulty   string `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel   string `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	Marks        int    `json:"marks" binding:"required,min=1"`

	// Optional fields
	Topic            string `json:"topic"`
	SubTopic         string `json:"sub_topic"`
	CorrectAnswer    string `json:"correct_answer"`
	Explanation      string `json:"explanation"`
	TimeLimitSeconds *int   `json:"time_limit_seconds"`
	Tags             []string `json:"tags"`

	// Contextual fields
	SchoolID          string `json:"school_id,omitempty" binding:"omitempty,uuid"`
	ClassLevelID      string `json:"class_level_id,omitempty" binding:"omitempty,uuid"`
	ClassID           string `json:"class_id,omitempty" binding:"omitempty,uuid"`
	SessionID         string `json:"session_id,omitempty" binding:"omitempty,uuid"`
	TermID            string `json:"term_id,omitempty" binding:"omitempty,uuid"`
	CurriculumType    string `json:"curriculum_type,omitempty"`
	SourceType        string `json:"source_type,omitempty"`
	ExternalID        string `json:"external_id,omitempty"`
	LearningObjective string `json:"learning_objective,omitempty"`
	NegativeMarks     float64 `json:"negative_marks,omitempty"`
	Order             int    `json:"order,omitempty"`
	IsRequired        bool   `json:"is_required,omitempty"`

	// Question type specific fields
	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
	Options           map[string]string `json:"options,omitempty"` // Legacy flat map
	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria `json:"rubric,omitempty"`
}

// UpdateQuestionRequest represents the request to update a question
type UpdateQuestionRequest struct {
	// Optional fields (all pointers to allow partial updates)
	QuestionText     *string           `json:"question_text"`
	Topic            *string           `json:"topic"`
	SubTopic         *string           `json:"sub_topic"`
	CorrectAnswer    *string           `json:"correct_answer"`
	Explanation      *string           `json:"explanation"`
	Marks            *int              `json:"marks"`
	Difficulty       *string           `json:"difficulty"`
	BloomLevel       *string           `json:"bloom_level"`
	TimeLimitSeconds *int              `json:"time_limit_seconds"`
	Status           *string           `json:"status"`
	CurriculumType   *string           `json:"curriculum_type"`
	SourceType       *string           `json:"source_type"`
	LearningObjective *string          `json:"learning_objective"`
	NegativeMarks    *float64          `json:"negative_marks"`
	Order            *int              `json:"order"`
	IsRequired       *bool             `json:"is_required"`

	// Question type specific fields
	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
	Options           map[string]string `json:"options,omitempty"`
	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria `json:"rubric,omitempty"`
}

// FilterQuestionsRequest represents advanced filter criteria
type FilterQuestionsRequest struct {
	SubjectID      string   `json:"subject_id"`
	SchoolID       string   `json:"school_id"`
	ClassLevelID   string   `json:"class_level_id"`
	SessionID      string   `json:"session_id"`
	TermID         string   `json:"term_id"`
	Topic          string   `json:"topic"`
	Difficulty     []string `json:"difficulty"`
	BloomLevel     []string `json:"bloom_level"`
	QuestionType   []string `json:"question_type"`
	Tags           []string `json:"tags"`
	Status         string   `json:"status"`
	Search         string   `json:"search"`
	Page           int      `json:"page"`
	Limit          int      `json:"limit"`
}

// BulkCreateQuestionRequest represents bulk question creation
type BulkCreateQuestionRequest struct {
	Questions []CreateQuestionRequest `json:"questions" binding:"required,dive"`
}

// BulkDeleteRequest represents bulk question deletion
type BulkDeleteRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required"`
}

// CreateTagRequest represents tag creation
type CreateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// AIGenerateQuestionsRequest represents AI question generation request
type AIGenerateQuestionsRequest struct {
	SchoolID          string   `json:"school_id" binding:"required,uuid"`
	ClassLevelID      string   `json:"class_level_id" binding:"required,uuid"`
	SubjectID         string   `json:"subject_id" binding:"required,uuid"`
	Topic             string   `json:"topic" binding:"required"`
	NumberOfQuestions int      `json:"number_of_questions" binding:"required,min=1,max=100"`
	Difficulty        string   `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel        string   `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	CurriculumType    string   `json:"curriculum_type,omitempty"`
	SourceText        string   `json:"source_text,omitempty"`
	Keywords          []string `json:"keywords,omitempty"`
}

// ExtractTextQuestionsRequest represents text extraction request
type ExtractTextQuestionsRequest struct {
	SchoolID     string `json:"school_id" binding:"required,uuid"`
	ClassLevelID string `json:"class_level_id" binding:"required,uuid"`
	SubjectID    string `json:"subject_id" binding:"required,uuid"`
	Text         string `json:"text" binding:"required"`
	Format       string `json:"format" binding:"required,oneof=plain markdown html"`
}

// BulkQuestionImportRequest represents bulk import from structured data
type BulkQuestionImportRequest struct {
	SchoolID       string               `json:"school_id" binding:"required,uuid"`
	ClassLevelID   string               `json:"class_level_id" binding:"required,uuid"`
	ClassID        string               `json:"class_id,omitempty"`
	SessionID      string               `json:"session_id,omitempty"`
	TermID         string               `json:"term_id,omitempty"`
	CurriculumType string               `json:"curriculum_type"`
	SourceType     string               `json:"source_type"`
	Status         string               `json:"status"`
	CreatedBy      string               `json:"created_by" binding:"required,uuid"`
	Questions      []QuestionImportItem `json:"questions" binding:"required,dive"`
}

// QuestionImportItem represents a single question in bulk import
type QuestionImportItem struct {
	ExternalID        string           `json:"external_id"`
	SubjectID         string           `json:"subject_id" binding:"required,uuid"`
	Topic             string           `json:"topic"`
	SubTopic          string           `json:"sub_topic"`
	LearningObjective string           `json:"learning_objective"`
	QuestionText      string           `json:"question_text" binding:"required"`
	QuestionType      string           `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
	Difficulty        string           `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel        string           `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	Options           []QuestionOption `json:"options"`
	CorrectOptionKeys []string         `json:"correct_option_keys"`
	Rubric            []RubricCriteria `json:"rubric"`
	Explanation       string           `json:"explanation"`
	Marks             int              `json:"marks" binding:"min=1"`
	NegativeMarks     float64          `json:"negative_marks"`
	TimeLimitSeconds  int              `json:"time_limit_seconds"`
	Tags              []string         `json:"tags"`
	Order             int              `json:"order"`
	IsRequired        bool             `json:"is_required"`
}

// BulkUploadRequest represents file upload request
type BulkUploadRequest struct {
	ExamID    string                `form:"exam_id" binding:"required,uuid"`
	File      *multipart.FileHeader `form:"file" binding:"required"`
	Format    string                `form:"format" binding:"required,oneof=csv json excel"`
	HasHeader bool                  `form:"has_header"`
}

// ============================================
// RESPONSE DTOs - FIXED
// ============================================

// QuestionBankResponse represents the response for a question
type QuestionBankResponse struct {
	// Core fields
	ID           string    `json:"id"`
	QuestionText string    `json:"question_text"`
	QuestionType string    `json:"question_type"`
	Difficulty   string    `json:"difficulty"`
	BloomLevel   string    `json:"bloom_level"`
	Marks        int       `json:"marks"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Contextual fields
	SubjectID        string  `json:"subject_id"`
	SubjectName      string  `json:"subject_name"`
	SchoolID         string  `json:"school_id"`
	ClassLevelID     string  `json:"class_level_id"`
	ClassID          *string `json:"class_id,omitempty"`
	SessionID        *string `json:"session_id,omitempty"`
	TermID           *string `json:"term_id,omitempty"`
	CurriculumType   string  `json:"curriculum_type"`
	SourceType       string  `json:"source_type"`
	ExternalID       string  `json:"external_id,omitempty"`

	// Content fields
	Topic             string          `json:"topic"`
	SubTopic          string          `json:"sub_topic"`
	LearningObjective string          `json:"learning_objective"`
	Options           []QuestionOption `json:"options,omitempty"`
	CorrectAnswer     string          `json:"correct_answer"`
	CorrectOptionKeys []string        `json:"correct_option_keys,omitempty"`
	Explanation       string          `json:"explanation"`
	Rubric            []RubricCriteria `json:"rubric,omitempty"`
	Tags              []string        `json:"tags"`

	// Metadata fields
	Status           string   `json:"status"`
	Version          int      `json:"version"`
	UsageCount       int      `json:"usage_count"`
	SuccessRate      *float64 `json:"success_rate,omitempty"`
	NegativeMarks    float64  `json:"negative_marks"`
	TimeLimitSeconds *int     `json:"time_limit_seconds,omitempty"`
	Order            int      `json:"order"`
	IsRequired       bool     `json:"is_required"`

	// User fields
	CreatedBy     string    `json:"created_by"`
	CreatedByName string    `json:"created_by_name"`
	Attachments   []AttachmentResponse `json:"attachments,omitempty"`
}

// TagResponse represents the response for a tag
type TagResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	UsageCount  int       `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// AttachmentResponse represents a question attachment
type AttachmentResponse struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	FileType  string    `json:"file_type"`
	FileURL   string    `json:"file_url"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// BulkImportResponse represents bulk import response
type BulkImportResponse struct {
	JobID            string   `json:"job_id"`
	Status           string   `json:"status"`
	TotalRecords     int      `json:"total_records"`
	ProcessedRecords int      `json:"processed_records"`
	FailedRecords    int      `json:"failed_records"`
	Errors           []string `json:"errors,omitempty"`
}

// AIQuestionGenerationResponse represents AI generation response
type AIQuestionGenerationResponse struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Questions []QuestionBankResponse `json:"questions,omitempty"`
	Message   string                 `json:"message,omitempty"`
}

// AIJobStatusResponse represents AI job status
type AIJobStatusResponse struct {
	JobID        string     `json:"job_id"`
	Status       string     `json:"status"` // queued, processing, completed, failed
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// BulkUploadResponse represents bulk upload response
type BulkUploadResponse struct {
	TotalProcessed int      `json:"total_processed"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors,omitempty"`
}

// QuestionStatistics represents question statistics
type QuestionStatistics struct {
	TotalQuestions     int               `json:"total_questions"`
	PublishedCount     int               `json:"published_count"`
	DraftCount         int               `json:"draft_count"`
	ArchivedCount      int               `json:"archived_count"`
	ByDifficulty       map[string]int    `json:"by_difficulty"`
	ByBloomLevel       map[string]int    `json:"by_bloom_level"`
	ByType             map[string]int    `json:"by_type"`
	AverageMarks       float64           `json:"average_marks"`
	TotalUsage         int               `json:"total_usage"`
	AverageSuccessRate float64           `json:"average_success_rate"`
}

// BulkImportStatusResponse represents bulk import status
type BulkImportStatusResponse struct {
	JobID            string              `json:"job_id"`
	Status           string              `json:"status"`
	TotalRecords     int                 `json:"total_records"`
	ProcessedRecords int                 `json:"processed_records"`
	FailedRecords    int                 `json:"failed_records"`
	Progress         float64             `json:"progress"`
	Errors           []ImportErrorDetail `json:"errors,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	CompletedAt      *time.Time          `json:"completed_at,omitempty"`
}

// ImportErrorDetail represents a detailed import error
type ImportErrorDetail struct {
	Row    int    `json:"row"`
	Column string `json:"column"`
	Error  string `json:"error"`
	Value  string `json:"value,omitempty"`
}

// QuestionListResponse represents paginated question list
type QuestionListResponse struct {
	Questions  []QuestionBankResponse `json:"questions"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// CSVTemplateResponse represents CSV template structure
type CSVTemplateResponse struct {
	Headers           []string          `json:"headers"`
	ExampleRow        map[string]string `json:"example_row"`
	RequiredFields    []string          `json:"required_fields"`
	FieldDescriptions map[string]string `json:"field_descriptions"`
}

// CloneQuestionRequest represents question cloning request
type CloneQuestionRequest struct {
	QuestionID string `json:"question_id" binding:"required,uuid"`
}

// AIParaphraseRequest represents AI paraphrasing request
type AIParaphraseRequest struct {
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	VariationCount int    `json:"variation_count" binding:"required,min=1,max=5"`
}

// ============================================
// SUB-STRUCTURES - FIXED
// ============================================

// QuestionOption represents a single option in a question
type QuestionOption struct {
	Key  string `json:"key" binding:"required"`
	Text string `json:"text" binding:"required"`
}

// RubricCriteria represents a single rubric criteria
type RubricCriteria struct {
	Criteria string `json:"criteria" binding:"required"`
	Marks    int    `json:"marks" binding:"required,min=1"`
}

// CSVQuestionRow represents a row in CSV import
type CSVQuestionRow struct {
	QuestionText  string `csv:"question_text"`
	OptionA       string `csv:"option_a"`
	OptionB       string `csv:"option_b"`
	OptionC       string `csv:"option_c"`
	OptionD       string `csv:"option_d"`
	CorrectAnswer string `csv:"correct_answer"`
	Explanation   string `csv:"explanation"`
	Marks         int    `csv:"marks"`
	Topic         string `csv:"topic"`
	SubTopic      string `csv:"sub_topic"`
	Difficulty    string `csv:"difficulty"`
	BloomLevel    string `csv:"bloom_level"`
	QuestionType  string `csv:"question_type"`
	SubjectID     string `csv:"subject_id"`
}

// JSONQuestion represents a question in JSON import
type JSONQuestion struct {
	QuestionText  string `json:"question_text"`
	OptionA       string `json:"option_a"`
	OptionB       string `json:"option_b"`
	OptionC       string `json:"option_c"`
	OptionD       string `json:"option_d"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation"`
	Marks         int    `json:"marks"`
	Topic         string `json:"topic"`
	SubTopic      string `json:"sub_topic"`
	Difficulty    string `json:"difficulty"`
	BloomLevel    string `json:"bloom_level"`
	QuestionType  string `json:"question_type"`
	SubjectID     string `json:"subject_id"`
}

// JSONQuestionImport represents the structure for JSON import
type JSONQuestionImport struct {
	Questions []JSONQuestion `json:"questions"`
}

// ============================================
// VALIDATION METHODS - FIXED
// ============================================

// Validate performs validation on CreateQuestionRequest
func (req *CreateQuestionRequest) Validate() error {
	if req.QuestionText == "" {
		return &ValidationError{Field: "question_text", Message: "question text is required"}
	}

	if req.SubjectID == "" {
		return &ValidationError{Field: "subject_id", Message: "subject ID is required"}
	}

	if req.Marks < 1 {
		return &ValidationError{Field: "marks", Message: "marks must be at least 1"}
	}

	// Validate based on question type
	switch req.QuestionType {
	case "single_choice", "multiple_choice", "true_false":
		if len(req.OptionsArray) == 0 && len(req.Options) == 0 {
			return &ValidationError{Field: "options", Message: "options are required for MCQ questions"}
		}
		if len(req.CorrectOptionKeys) == 0 && req.CorrectAnswer == "" {
			return &ValidationError{Field: "correct_option_keys", Message: "correct answer is required for MCQ questions"}
		}
	case "essay":
		if len(req.Rubric) == 0 {
			return &ValidationError{Field: "rubric", Message: "rubric is required for essay questions"}
		}
	}

	return nil
}

// Validate performs validation on UpdateQuestionRequest
func (req *UpdateQuestionRequest) Validate() error {
	if req.Marks != nil && *req.Marks < 1 {
		return &ValidationError{Field: "marks", Message: "marks must be at least 1"}
	}

	if req.Status != nil {
		validStatuses := []string{"draft", "published", "archived"}
		valid := false
		for _, s := range validStatuses {
			if s == *req.Status {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{Field: "status", Message: "invalid status value"}
		}
	}

	return nil
}

// ============================================
// ERROR TYPES - FIXED
// ============================================

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ============================================
// CONSTANTS - FIXED
// ============================================

const (
	// Question types
	QuestionTypeSingle    = "single_choice"
	QuestionTypeMultiple  = "multiple_choice"
	QuestionTypeTrueFalse = "true_false"
	QuestionTypeEssay     = "essay"
	QuestionTypeFillBlank = "fill_blank"

	// Difficulty levels
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
	DifficultyExpert = "expert"

	// Bloom's Taxonomy levels
	BloomRemember   = "remember"
	BloomUnderstand = "understand"
	BloomApply      = "apply"
	BloomAnalyse    = "analyse"
	BloomEvaluate   = "evaluate"
	BloomCreate     = "create"

	// Question statuses
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	// Validation limits
	MaxMarks          = 1000
	MinMarks          = 1
	MaxQuestionLength = 10000
	MaxOptionsCount   = 10
	MaxTagsCount      = 10
	MaxRubricCriteria = 10
)




// package dto

// import (
// 	"mime/multipart"
// 	"time"
// )

// // ============================================
// // REQUEST DTOs
// // ============================================

// // CreateQuestionRequest – extended with new fields (optional for backward compatibility)
// type CreateQuestionRequest struct {
// 	// Existing fields
// 	SubjectID        string            `json:"subject_id" binding:"required,uuid"`
// 	Topic            string            `json:"topic"`
// 	SubTopic         string            `json:"sub_topic"`
// 	QuestionText     string            `json:"question_text" binding:"required"`
// 	QuestionType     string            `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
// 	Difficulty       string            `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
// 	BloomLevel       string            `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
// 	Options          map[string]string `json:"options,omitempty"` // legacy flat map – still accepted
// 	CorrectAnswer    string            `json:"correct_answer" binding:"required"`
// 	Explanation      string            `json:"explanation"`
// 	Marks            int               `json:"marks" binding:"required,min=1"`
// 	TimeLimitSeconds *int              `json:"time_limit_seconds"`
// 	Tags             []string          `json:"tags"`

// 	// New fields (optional for backward compatibility)
// 	SchoolID          string `json:"school_id,omitempty" binding:"omitempty,uuid"`
// 	ClassLevelID      string `json:"class_level_id,omitempty" binding:"omitempty,uuid"`
// 	ClassID           string `json:"class_id,omitempty" binding:"omitempty,uuid"`
// 	SessionID         string `json:"session_id,omitempty" binding:"omitempty,uuid"`
// 	TermID            string `json:"term_id,omitempty" binding:"omitempty,uuid"`
// 	CurriculumType    string `json:"curriculum_type,omitempty"`
// 	SourceType        string `json:"source_type,omitempty"`
// 	ExternalID        string `json:"external_id,omitempty"`
// 	LearningObjective string `json:"learning_objective,omitempty"`
// 	NegativeMarks     float64 `json:"negative_marks,omitempty"`
// 	Order             int    `json:"order,omitempty"`
// 	IsRequired        bool   `json:"is_required,omitempty"`
// 	// For new array format (if provided, overrides flat options)
// 	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
// 	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
// 	Rubric            []RubricCriteria `json:"rubric,omitempty"`
// }

// // UpdateQuestionRequest – extended with new fields
// type UpdateQuestionRequest struct {
// 	// Existing fields
// 	QuestionText     *string           `json:"question_text"`
// 	Options          map[string]string `json:"options"`
// 	CorrectAnswer    *string           `json:"correct_answer"`
// 	Explanation      *string           `json:"explanation"`
// 	Marks            *int              `json:"marks"`
// 	Difficulty       *string           `json:"difficulty"`
// 	BloomLevel       *string           `json:"bloom_level"`
// 	TimeLimitSeconds *int              `json:"time_limit_seconds"`
// 	Status           *string           `json:"status"`

// 	// New fields (optional)
// 	Topic             *string          `json:"topic"`
// 	SubTopic          *string          `json:"sub_topic"`
// 	CurriculumType    *string          `json:"curriculum_type"`
// 	SourceType        *string          `json:"source_type"`
// 	LearningObjective *string          `json:"learning_objective"`
// 	NegativeMarks     *float64         `json:"negative_marks"`
// 	Order             *int             `json:"order"`
// 	IsRequired        *bool            `json:"is_required"`
// 	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
// 	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
// 	Rubric            []RubricCriteria `json:"rubric,omitempty"`
// }

// // BulkImportRequest – remains unchanged for file import
// type BulkImportRequest struct {
// 	FileType string                `form:"file_type" binding:"required,oneof=csv json excel"`
// 	File     *multipart.FileHeader `form:"file" binding:"required"`
// }

// // AIGenerateQuestionsRequest – extended with school and class level
// type AIGenerateQuestionsRequest struct {
// 	SchoolID         string   `json:"school_id" binding:"required,uuid"`
// 	ClassLevelID     string   `json:"class_level_id" binding:"required,uuid"`
// 	SubjectID        string   `json:"subject_id" binding:"required,uuid"`
// 	Topic            string   `json:"topic" binding:"required"`
// 	NumberOfQuestions int      `json:"number_of_questions" binding:"required,min=1,max=100"`
// 	Difficulty       string   `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
// 	BloomLevel       string   `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
// 	CurriculumType   string   `json:"curriculum_type,omitempty"`
// 	SourceText       string   `json:"source_text,omitempty"`
// 	Keywords         []string `json:"keywords,omitempty"`
// }

// type AIParaphraseRequest struct {
// 	QuestionID     string `json:"question_id" binding:"required,uuid"`
// 	VariationCount int    `json:"variation_count" binding:"required,min=1,max=5"`
// }

// type ExtractTextQuestionsRequest struct {
// 	SchoolID     string `json:"school_id" binding:"required,uuid"`
// 	ClassLevelID string `json:"class_level_id" binding:"required,uuid"`
// 	SubjectID    string `json:"subject_id" binding:"required,uuid"`
// 	Text         string `json:"text" binding:"required"`
// 	Format       string `json:"format" binding:"required,oneof=plain markdown html"`
// }

// type FilterQuestionsRequest struct {
// 	SubjectID      string   `json:"subject_id"`
// 	SchoolID       string   `json:"school_id"`
// 	ClassLevelID   string   `json:"class_level_id"`
// 	SessionID      string   `json:"session_id"`
// 	TermID         string   `json:"term_id"`
// 	Topic          string   `json:"topic"`
// 	Difficulty     []string `json:"difficulty"`
// 	BloomLevel     []string `json:"bloom_level"`
// 	QuestionType   []string `json:"question_type"`
// 	Tags           []string `json:"tags"`
// 	Status         string   `json:"status"`
// 	Search         string   `json:"search"`
// 	Page           int      `json:"page"`
// 	Limit          int      `json:"limit"`
// }

// type CreateTagRequest struct {
// 	Name        string `json:"name" binding:"required"`
// 	Description string `json:"description"`
// }

// type BulkDeleteRequest struct {
// 	QuestionIDs []string `json:"question_ids" binding:"required"`
// }

// type CloneQuestionRequest struct {
// 	QuestionID string `json:"question_id" binding:"required,uuid"`
// }

// // ============================================
// // NEW DTOs FOR BULK IMPORT
// // ============================================

// type BulkQuestionImportRequest struct {
// 	SchoolID       string               `json:"school_id" binding:"required,uuid"`
// 	ClassLevelID   string               `json:"class_level_id" binding:"required,uuid"`
// 	ClassID        string               `json:"class_id,omitempty"`
// 	SessionID      string               `json:"session_id,omitempty"`
// 	TermID         string               `json:"term_id,omitempty"`
// 	CurriculumType string               `json:"curriculum_type"`
// 	SourceType     string               `json:"source_type"`
// 	Status         string               `json:"status"`
// 	CreatedBy      string               `json:"created_by" binding:"required,uuid"`
// 	Questions      []QuestionImportItem `json:"questions" binding:"required,dive"`
// }

// type QuestionImportItem struct {
// 	ID                string           `json:"id"` // ignored – server generates
// 	ExternalID        string           `json:"external_id"`
// 	SubjectID         string           `json:"subject_id" binding:"required,uuid"`
// 	Topic             string           `json:"topic"`
// 	SubTopic          string           `json:"sub_topic"`
// 	LearningObjective string           `json:"learning_objective"`
// 	QuestionText      string           `json:"question_text" binding:"required"`
// 	QuestionType      string           `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
// 	Difficulty        string           `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
// 	BloomLevel        string           `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
// 	Options           []QuestionOption `json:"options"`
// 	CorrectOptionKeys []string         `json:"correct_option_keys"`
// 	Rubric            []RubricCriteria `json:"rubric"`
// 	Explanation       string           `json:"explanation"`
// 	Marks             int              `json:"marks" binding:"min=0"`
// 	NegativeMarks     float64          `json:"negative_marks"`
// 	TimeLimitSeconds  int              `json:"time_limit_seconds"`
// 	Tags              []string         `json:"tags"`
// 	Order             int              `json:"order"`
// 	IsRequired        bool             `json:"is_required"`
// }

// type QuestionOption struct {
// 	Key  string `json:"key"`
// 	Text string `json:"text"`
// }

// type RubricCriteria struct {
// 	Criteria string `json:"criteria"`
// 	Marks    int    `json:"marks"`
// }

// // ============================================
// // RESPONSE DTOs – updated
// // ============================================

// type QuestionBankResponse struct {
// 	// Existing fields
// 	ID               string                 `json:"id"`
// 	SubjectID        string                 `json:"subject_id"`
// 	SubjectName      string                 `json:"subject_name"`
// 	Topic            string                 `json:"topic"`
// 	SubTopic         string                 `json:"sub_topic"`
// 	QuestionText     string                 `json:"question_text"`
// 	QuestionType     string                 `json:"question_type"`
// 	Difficulty       string                 `json:"difficulty"`
// 	BloomLevel       string                 `json:"bloom_level"`
// 	Options          []QuestionOption       `json:"options,omitempty"` // changed to array
// 	CorrectAnswer    string                 `json:"correct_answer"`    // legacy
// 	Explanation      string                 `json:"explanation"`
// 	Marks            int                    `json:"marks"`
// 	TimeLimitSeconds *int                   `json:"time_limit_seconds,omitempty"`
// 	Tags             []string               `json:"tags"`
// 	Status           string                 `json:"status"`
// 	Version          int                    `json:"version"`
// 	UsageCount       int                    `json:"usage_count"`
// 	SuccessRate      *float64               `json:"success_rate,omitempty"`
// 	Attachments      []AttachmentResponse   `json:"attachments,omitempty"`
// 	CreatedAt        time.Time              `json:"created_at"`
// 	UpdatedAt        time.Time              `json:"updated_at"`
// 	CreatedBy        string                 `json:"created_by"`
// 	CreatedByName    string                 `json:"created_by_name"`

// 	// New fields
// 	SchoolID          string                 `json:"school_id"`
// 	ClassLevelID      string                 `json:"class_level_id"`
// 	ClassID           *string                `json:"class_id,omitempty"`
// 	SessionID         *string                `json:"session_id,omitempty"`
// 	TermID            *string                `json:"term_id,omitempty"`
// 	CurriculumType    string                 `json:"curriculum_type"`
// 	SourceType        string                 `json:"source_type"`
// 	ExternalID        string                 `json:"external_id,omitempty"`
// 	LearningObjective string                 `json:"learning_objective"`
// 	CorrectOptionKeys []string               `json:"correct_option_keys,omitempty"`
// 	Rubric            []RubricCriteria       `json:"rubric,omitempty"`
// 	NegativeMarks     float64                `json:"negative_marks"`
// 	Order             int                    `json:"order"`
// 	IsRequired        bool                   `json:"is_required"`
// }

// // Other existing response types (unchanged)
// type AttachmentResponse struct {
// 	ID        string    `json:"id"`
// 	FileName  string    `json:"file_name"`
// 	FileType  string    `json:"file_type"`
// 	FileURL   string    `json:"file_url"`
// 	FileSize  int64     `json:"file_size"`
// 	CreatedAt time.Time `json:"created_at"`
// }

// type TagResponse struct {
// 	ID          string    `json:"id"`
// 	Name        string    `json:"name"`
// 	Slug        string    `json:"slug"`
// 	Description string    `json:"description"`
// 	UsageCount  int       `json:"usage_count"`
// 	CreatedAt   time.Time `json:"created_at"`
// }

// type BulkImportResponse struct {
// 	JobID            string   `json:"job_id"`
// 	Status           string   `json:"status"`
// 	TotalRecords     int      `json:"total_records"`
// 	ProcessedRecords int      `json:"processed_records"`
// 	FailedRecords    int      `json:"failed_records"`
// 	Errors           []string `json:"errors,omitempty"`
// }

// type AIQuestionGenerationResponse struct {
// 	JobID     string                 `json:"job_id"`
// 	Status    string                 `json:"status"`
// 	Questions []QuestionBankResponse `json:"questions,omitempty"`
// 	Message   string                 `json:"message,omitempty"`
// }

// type QuestionListResponse struct {
// 	Questions  []QuestionBankResponse `json:"questions"`
// 	Total      int64                  `json:"total"`
// 	Page       int                    `json:"page"`
// 	Limit      int                    `json:"limit"`
// 	TotalPages int                    `json:"total_pages"`
// }

// type BulkImportStatusResponse struct {
// 	JobID            string              `json:"job_id"`
// 	Status           string              `json:"status"`
// 	TotalRecords     int                 `json:"total_records"`
// 	ProcessedRecords int                 `json:"processed_records"`
// 	FailedRecords    int                 `json:"failed_records"`
// 	Progress         float64             `json:"progress"`
// 	Errors           []ImportErrorDetail `json:"errors,omitempty"`
// 	CreatedAt        time.Time           `json:"created_at"`
// 	CompletedAt      *time.Time          `json:"completed_at,omitempty"`
// }

// type ImportErrorDetail struct {
// 	Row    int    `json:"row"`
// 	Column string `json:"column"`
// 	Error  string `json:"error"`
// 	Value  string `json:"value,omitempty"`
// }

// type CSVTemplateResponse struct {
// 	Headers           []string          `json:"headers"`
// 	ExampleRow        map[string]string `json:"example_row"`
// 	RequiredFields    []string          `json:"required_fields"`
// 	FieldDescriptions map[string]string `json:"field_descriptions"`
// }

// type QuestionStatistics struct {
// 	TotalQuestions      int               `json:"total_questions"`
// 	PublishedCount      int               `json:"published_count"`
// 	DraftCount          int               `json:"draft_count"`
// 	ArchivedCount       int               `json:"archived_count"`
// 	ByDifficulty        map[string]int    `json:"by_difficulty"`
// 	ByBloomLevel        map[string]int    `json:"by_bloom_level"`
// 	ByType              map[string]int    `json:"by_type"`
// 	AverageMarks        float64           `json:"average_marks"`
// 	TotalUsage          int               `json:"total_usage"`
// 	AverageSuccessRate  float64           `json:"average_success_rate"`
// }

// type BulkCreateQuestionRequest struct {
// 	Questions []CreateQuestionRequest `json:"questions" binding:"required,dive"`
// }

// // ============================================
// // BULK UPLOAD TYPES (CSV, JSON) – unchanged
// // ============================================

// type BulkUploadRequest struct {
// 	ExamID    string                `form:"exam_id" binding:"required,uuid"`
// 	File      *multipart.FileHeader `form:"file" binding:"required"`
// 	Format    string                `form:"format" binding:"required,oneof=csv json excel"`
// 	HasHeader bool                  `form:"has_header"`
// }

// type CSVQuestionRow struct {
// 	QuestionText  string `csv:"question_text"`
// 	OptionA       string `csv:"option_a"`
// 	OptionB       string `csv:"option_b"`
// 	OptionC       string `csv:"option_c"`
// 	OptionD       string `csv:"option_d"`
// 	CorrectAnswer string `csv:"correct_answer"`
// 	Explanation   string `csv:"explanation"`
// 	Marks         int    `csv:"marks"`
// 	Topic         string `csv:"topic"`
// 	SubTopic      string `csv:"sub_topic"`
// 	Difficulty    string `csv:"difficulty"`
// 	BloomLevel    string `csv:"bloom_level"`
// 	QuestionType  string `csv:"question_type"`
// 	SubjectID     string `csv:"subject_id"`
// }

// type JSONQuestion struct {
// 	QuestionText  string `json:"question_text"`
// 	OptionA       string `json:"option_a"`
// 	OptionB       string `json:"option_b"`
// 	OptionC       string `json:"option_c"`
// 	OptionD       string `json:"option_d"`
// 	CorrectAnswer string `json:"correct_answer"`
// 	Explanation   string `json:"explanation"`
// 	Marks         int    `json:"marks"`
// 	Topic         string `json:"topic"`
// 	SubTopic      string `json:"sub_topic"`
// 	Difficulty    string `json:"difficulty"`
// 	BloomLevel    string `json:"bloom_level"`
// 	QuestionType  string `json:"question_type"`
// 	SubjectID     string `json:"subject_id"`
// }

// type JSONQuestionImport struct {
// 	Questions []JSONQuestion `json:"questions"`
// }

// type BulkUploadResponse struct {
// 	TotalProcessed int      `json:"total_processed"`
// 	SuccessCount   int      `json:"success_count"`
// 	FailedCount    int      `json:"failed_count"`
// 	Errors         []string `json:"errors,omitempty"`
// }

// // AIJobStatusResponse holds the status of an AI job.
// type AIJobStatusResponse struct {
// 	JobID        string     `json:"job_id"`
// 	Status       string     `json:"status"` // queued, processing, completed, failed
// 	ErrorMessage string     `json:"error_message,omitempty"`
// 	CreatedAt    time.Time  `json:"created_at"`
// 	CompletedAt  *time.Time `json:"completed_at,omitempty"`
// }


