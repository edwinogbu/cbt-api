package dto

import (
	"mime/multipart"
	"time"
)

// ============================================
// REQUEST DTOs
// ============================================

// CreateQuestionRequest – extended with new fields (optional for backward compatibility)
type CreateQuestionRequest struct {
	// Existing fields
	SubjectID        string            `json:"subject_id" binding:"required,uuid"`
	Topic            string            `json:"topic"`
	SubTopic         string            `json:"sub_topic"`
	QuestionText     string            `json:"question_text" binding:"required"`
	QuestionType     string            `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
	Difficulty       string            `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel       string            `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	Options          map[string]string `json:"options,omitempty"` // legacy flat map – still accepted
	CorrectAnswer    string            `json:"correct_answer" binding:"required"`
	Explanation      string            `json:"explanation"`
	Marks            int               `json:"marks" binding:"required,min=1"`
	TimeLimitSeconds *int              `json:"time_limit_seconds"`
	Tags             []string          `json:"tags"`

	// New fields (optional for backward compatibility)
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
	// For new array format (if provided, overrides flat options)
	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria `json:"rubric,omitempty"`
}

// UpdateQuestionRequest – extended with new fields
type UpdateQuestionRequest struct {
	// Existing fields
	QuestionText     *string           `json:"question_text"`
	Options          map[string]string `json:"options"`
	CorrectAnswer    *string           `json:"correct_answer"`
	Explanation      *string           `json:"explanation"`
	Marks            *int              `json:"marks"`
	Difficulty       *string           `json:"difficulty"`
	BloomLevel       *string           `json:"bloom_level"`
	TimeLimitSeconds *int              `json:"time_limit_seconds"`
	Status           *string           `json:"status"`

	// New fields (optional)
	Topic             *string          `json:"topic"`
	SubTopic          *string          `json:"sub_topic"`
	CurriculumType    *string          `json:"curriculum_type"`
	SourceType        *string          `json:"source_type"`
	LearningObjective *string          `json:"learning_objective"`
	NegativeMarks     *float64         `json:"negative_marks"`
	Order             *int             `json:"order"`
	IsRequired        *bool            `json:"is_required"`
	OptionsArray      []QuestionOption `json:"options_array,omitempty"`
	CorrectOptionKeys []string         `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria `json:"rubric,omitempty"`
}

// BulkImportRequest – remains unchanged for file import
type BulkImportRequest struct {
	FileType string                `form:"file_type" binding:"required,oneof=csv json excel"`
	File     *multipart.FileHeader `form:"file" binding:"required"`
}

// AIGenerateQuestionsRequest – extended with school and class level
type AIGenerateQuestionsRequest struct {
	SchoolID         string   `json:"school_id" binding:"required,uuid"`
	ClassLevelID     string   `json:"class_level_id" binding:"required,uuid"`
	SubjectID        string   `json:"subject_id" binding:"required,uuid"`
	Topic            string   `json:"topic" binding:"required"`
	NumberOfQuestions int      `json:"number_of_questions" binding:"required,min=1,max=100"`
	Difficulty       string   `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel       string   `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	CurriculumType   string   `json:"curriculum_type,omitempty"`
	SourceText       string   `json:"source_text,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
}

type AIParaphraseRequest struct {
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	VariationCount int    `json:"variation_count" binding:"required,min=1,max=5"`
}

type ExtractTextQuestionsRequest struct {
	SchoolID     string `json:"school_id" binding:"required,uuid"`
	ClassLevelID string `json:"class_level_id" binding:"required,uuid"`
	SubjectID    string `json:"subject_id" binding:"required,uuid"`
	Text         string `json:"text" binding:"required"`
	Format       string `json:"format" binding:"required,oneof=plain markdown html"`
}

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

type CreateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type BulkDeleteRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required"`
}

type CloneQuestionRequest struct {
	QuestionID string `json:"question_id" binding:"required,uuid"`
}

// ============================================
// NEW DTOs FOR BULK IMPORT
// ============================================

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

type QuestionImportItem struct {
	ID                string           `json:"id"` // ignored – server generates
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
	Marks             int              `json:"marks" binding:"min=0"`
	NegativeMarks     float64          `json:"negative_marks"`
	TimeLimitSeconds  int              `json:"time_limit_seconds"`
	Tags              []string         `json:"tags"`
	Order             int              `json:"order"`
	IsRequired        bool             `json:"is_required"`
}

type QuestionOption struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type RubricCriteria struct {
	Criteria string `json:"criteria"`
	Marks    int    `json:"marks"`
}

// ============================================
// RESPONSE DTOs – updated
// ============================================

type QuestionBankResponse struct {
	// Existing fields
	ID               string                 `json:"id"`
	SubjectID        string                 `json:"subject_id"`
	SubjectName      string                 `json:"subject_name"`
	Topic            string                 `json:"topic"`
	SubTopic         string                 `json:"sub_topic"`
	QuestionText     string                 `json:"question_text"`
	QuestionType     string                 `json:"question_type"`
	Difficulty       string                 `json:"difficulty"`
	BloomLevel       string                 `json:"bloom_level"`
	Options          []QuestionOption       `json:"options,omitempty"` // changed to array
	CorrectAnswer    string                 `json:"correct_answer"`    // legacy
	Explanation      string                 `json:"explanation"`
	Marks            int                    `json:"marks"`
	TimeLimitSeconds *int                   `json:"time_limit_seconds,omitempty"`
	Tags             []string               `json:"tags"`
	Status           string                 `json:"status"`
	Version          int                    `json:"version"`
	UsageCount       int                    `json:"usage_count"`
	SuccessRate      *float64               `json:"success_rate,omitempty"`
	Attachments      []AttachmentResponse   `json:"attachments,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	CreatedBy        string                 `json:"created_by"`
	CreatedByName    string                 `json:"created_by_name"`

	// New fields
	SchoolID          string                 `json:"school_id"`
	ClassLevelID      string                 `json:"class_level_id"`
	ClassID           *string                `json:"class_id,omitempty"`
	SessionID         *string                `json:"session_id,omitempty"`
	TermID            *string                `json:"term_id,omitempty"`
	CurriculumType    string                 `json:"curriculum_type"`
	SourceType        string                 `json:"source_type"`
	ExternalID        string                 `json:"external_id,omitempty"`
	LearningObjective string                 `json:"learning_objective"`
	CorrectOptionKeys []string               `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria       `json:"rubric,omitempty"`
	NegativeMarks     float64                `json:"negative_marks"`
	Order             int                    `json:"order"`
	IsRequired        bool                   `json:"is_required"`
}

// Other existing response types (unchanged)
type AttachmentResponse struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	FileType  string    `json:"file_type"`
	FileURL   string    `json:"file_url"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

type TagResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	UsageCount  int       `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type BulkImportResponse struct {
	JobID            string   `json:"job_id"`
	Status           string   `json:"status"`
	TotalRecords     int      `json:"total_records"`
	ProcessedRecords int      `json:"processed_records"`
	FailedRecords    int      `json:"failed_records"`
	Errors           []string `json:"errors,omitempty"`
}

type AIQuestionGenerationResponse struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Questions []QuestionBankResponse `json:"questions,omitempty"`
	Message   string                 `json:"message,omitempty"`
}

type QuestionListResponse struct {
	Questions  []QuestionBankResponse `json:"questions"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

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

type ImportErrorDetail struct {
	Row    int    `json:"row"`
	Column string `json:"column"`
	Error  string `json:"error"`
	Value  string `json:"value,omitempty"`
}

type CSVTemplateResponse struct {
	Headers           []string          `json:"headers"`
	ExampleRow        map[string]string `json:"example_row"`
	RequiredFields    []string          `json:"required_fields"`
	FieldDescriptions map[string]string `json:"field_descriptions"`
}

type QuestionStatistics struct {
	TotalQuestions      int               `json:"total_questions"`
	PublishedCount      int               `json:"published_count"`
	DraftCount          int               `json:"draft_count"`
	ArchivedCount       int               `json:"archived_count"`
	ByDifficulty        map[string]int    `json:"by_difficulty"`
	ByBloomLevel        map[string]int    `json:"by_bloom_level"`
	ByType              map[string]int    `json:"by_type"`
	AverageMarks        float64           `json:"average_marks"`
	TotalUsage          int               `json:"total_usage"`
	AverageSuccessRate  float64           `json:"average_success_rate"`
}

type BulkCreateQuestionRequest struct {
	Questions []CreateQuestionRequest `json:"questions" binding:"required,dive"`
}

// ============================================
// BULK UPLOAD TYPES (CSV, JSON) – unchanged
// ============================================

type BulkUploadRequest struct {
	ExamID    string                `form:"exam_id" binding:"required,uuid"`
	File      *multipart.FileHeader `form:"file" binding:"required"`
	Format    string                `form:"format" binding:"required,oneof=csv json excel"`
	HasHeader bool                  `form:"has_header"`
}

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

type JSONQuestionImport struct {
	Questions []JSONQuestion `json:"questions"`
}

type BulkUploadResponse struct {
	TotalProcessed int      `json:"total_processed"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors,omitempty"`
}

// AIJobStatusResponse holds the status of an AI job.
type AIJobStatusResponse struct {
	JobID        string     `json:"job_id"`
	Status       string     `json:"status"` // queued, processing, completed, failed
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}



// package dto

// import (
//     "mime/multipart"
//     "time"
// )

// // ============================================
// // REQUEST DTOs
// // ============================================

// type CreateQuestionRequest struct {
//     SubjectID        string            `json:"subject_id" binding:"required,uuid"`
//     Topic            string            `json:"topic"`
//     SubTopic         string            `json:"sub_topic"`
//     QuestionText     string            `json:"question_text" binding:"required"`
//     QuestionType     string            `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
//     Difficulty       string            `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
//     BloomLevel       string            `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
//     Options          map[string]string `json:"options,omitempty"`
//     CorrectAnswer    string            `json:"correct_answer" binding:"required"`
//     Explanation      string            `json:"explanation"`
//     Marks            int               `json:"marks" binding:"required,min=1"`
//     TimeLimitSeconds *int              `json:"time_limit_seconds"`
//     Tags             []string          `json:"tags"`
// }

// type UpdateQuestionRequest struct {
//     QuestionText     *string           `json:"question_text"`
//     Options          map[string]string `json:"options"`
//     CorrectAnswer    *string           `json:"correct_answer"`
//     Explanation      *string           `json:"explanation"`
//     Marks            *int              `json:"marks"`
//     Difficulty       *string           `json:"difficulty"`
//     BloomLevel       *string           `json:"bloom_level"`
//     TimeLimitSeconds *int              `json:"time_limit_seconds"`
//     Status           *string           `json:"status"`
// }

// type BulkImportRequest struct {
//     FileType string                `form:"file_type" binding:"required,oneof=csv json excel"`
//     File     *multipart.FileHeader `form:"file" binding:"required"`
// }

// type AIGenerateQuestionsRequest struct {
//     SubjectID         string   `json:"subject_id" binding:"required,uuid"`
//     Topic             string   `json:"topic" binding:"required"`
//     NumberOfQuestions int      `json:"number_of_questions" binding:"required,min=1,max=100"`
//     Difficulty        string   `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
//     BloomLevel        string   `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
//     SourceText        string   `json:"source_text,omitempty"`
//     Keywords          []string `json:"keywords,omitempty"`
// }

// type AIParaphraseRequest struct {
//     QuestionID     string `json:"question_id" binding:"required,uuid"`
//     VariationCount int    `json:"variation_count" binding:"required,min=1,max=5"`
// }

// type ExtractTextQuestionsRequest struct {
//     Text   string `json:"text" binding:"required"`
//     Format string `json:"format" binding:"required,oneof=plain markdown html"`
// }

// type FilterQuestionsRequest struct {
//     SubjectID    string   `json:"subject_id"`
//     Topic        string   `json:"topic"`
//     Difficulty   []string `json:"difficulty"`
//     BloomLevel   []string `json:"bloom_level"`
//     QuestionType []string `json:"question_type"`
//     Tags         []string `json:"tags"`
//     Status       string   `json:"status"`
//     Search       string   `json:"search"`
//     Page         int      `json:"page"`
//     Limit        int      `json:"limit"`
// }

// type CreateTagRequest struct {
//     Name        string `json:"name" binding:"required"`
//     Description string `json:"description"`
// }

// type BulkDeleteRequest struct {
//     QuestionIDs []string `json:"question_ids" binding:"required"`
// }

// type CloneQuestionRequest struct {
//     QuestionID string `json:"question_id" binding:"required,uuid"`
// }

// // ============================================
// // RESPONSE DTOs
// // ============================================

// type QuestionBankResponse struct {
//     ID               string                 `json:"id"`
//     SubjectID        string                 `json:"subject_id"`
//     SubjectName      string                 `json:"subject_name"`
//     Topic            string                 `json:"topic"`
//     SubTopic         string                 `json:"sub_topic"`
//     QuestionText     string                 `json:"question_text"`
//     QuestionType     string                 `json:"question_type"`
//     Difficulty       string                 `json:"difficulty"`
//     BloomLevel       string                 `json:"bloom_level"`
//     Options          map[string]string      `json:"options,omitempty"`
//     CorrectAnswer    string                 `json:"correct_answer"`
//     Explanation      string                 `json:"explanation"`
//     Marks            int                    `json:"marks"`
//     TimeLimitSeconds *int                   `json:"time_limit_seconds,omitempty"`
//     Tags             []string               `json:"tags"`
//     Status           string                 `json:"status"`
//     Version          int                    `json:"version"`
//     UsageCount       int                    `json:"usage_count"`
//     SuccessRate      *float64               `json:"success_rate,omitempty"`
//     Attachments      []AttachmentResponse   `json:"attachments,omitempty"`
//     CreatedAt        time.Time              `json:"created_at"`
//     UpdatedAt        time.Time              `json:"updated_at"`
//     CreatedBy        string                 `json:"created_by"`
//     CreatedByName    string                 `json:"created_by_name"`
// }

// type AttachmentResponse struct {
//     ID        string    `json:"id"`
//     FileName  string    `json:"file_name"`
//     FileType  string    `json:"file_type"`
//     FileURL   string    `json:"file_url"`
//     FileSize  int64     `json:"file_size"`
//     CreatedAt time.Time `json:"created_at"`
// }

// type TagResponse struct {
//     ID          string    `json:"id"`
//     Name        string    `json:"name"`
//     Slug        string    `json:"slug"`
//     Description string    `json:"description"`
//     UsageCount  int       `json:"usage_count"`
//     CreatedAt   time.Time `json:"created_at"`
// }

// type BulkImportResponse struct {
//     JobID            string   `json:"job_id"`
//     Status           string   `json:"status"`
//     TotalRecords     int      `json:"total_records"`
//     ProcessedRecords int      `json:"processed_records"`
//     FailedRecords    int      `json:"failed_records"`
//     Errors           []string `json:"errors,omitempty"`
// }

// type AIQuestionGenerationResponse struct {
//     JobID     string                 `json:"job_id"`
//     Status    string                 `json:"status"`
//     Questions []QuestionBankResponse `json:"questions,omitempty"`
//     Message   string                 `json:"message,omitempty"`
// }

// type QuestionListResponse struct {
//     Questions  []QuestionBankResponse `json:"questions"`
//     Total      int64                  `json:"total"`
//     Page       int                    `json:"page"`
//     Limit      int                    `json:"limit"`
//     TotalPages int                    `json:"total_pages"`
// }

// type BulkImportStatusResponse struct {
//     JobID            string              `json:"job_id"`
//     Status           string              `json:"status"`
//     TotalRecords     int                 `json:"total_records"`
//     ProcessedRecords int                 `json:"processed_records"`
//     FailedRecords    int                 `json:"failed_records"`
//     Progress         float64             `json:"progress"`
//     Errors           []ImportErrorDetail `json:"errors,omitempty"`
//     CreatedAt        time.Time           `json:"created_at"`
//     CompletedAt      *time.Time          `json:"completed_at,omitempty"`
// }

// type ImportErrorDetail struct {
//     Row    int    `json:"row"`
//     Column string `json:"column"`
//     Error  string `json:"error"`
//     Value  string `json:"value,omitempty"`
// }

// type CSVTemplateResponse struct {
//     Headers           []string          `json:"headers"`
//     ExampleRow        map[string]string `json:"example_row"`
//     RequiredFields    []string          `json:"required_fields"`
//     FieldDescriptions map[string]string `json:"field_descriptions"`
// }

// type QuestionStatistics struct {
//     TotalQuestions      int               `json:"total_questions"`
//     PublishedCount      int               `json:"published_count"`
//     DraftCount          int               `json:"draft_count"`
//     ArchivedCount       int               `json:"archived_count"`
//     ByDifficulty        map[string]int    `json:"by_difficulty"`
//     ByBloomLevel        map[string]int    `json:"by_bloom_level"`
//     ByType              map[string]int    `json:"by_type"`
//     AverageMarks        float64           `json:"average_marks"`
//     TotalUsage          int               `json:"total_usage"`
//     AverageSuccessRate  float64           `json:"average_success_rate"`
// }

// type BulkCreateQuestionRequest struct {
//     Questions []CreateQuestionRequest `json:"questions" binding:"required,dive"`
// }

// // ============================================
// // BULK UPLOAD TYPES (CSV, JSON)
// // ============================================

// type BulkUploadRequest struct {
//     ExamID    string                `form:"exam_id" binding:"required,uuid"`
//     File      *multipart.FileHeader `form:"file" binding:"required"`
//     Format    string                `form:"format" binding:"required,oneof=csv json excel"`
//     HasHeader bool                  `form:"has_header"`
// }

// // CSVQuestionRow with all fields needed for bulk import
// type CSVQuestionRow struct {
//     QuestionText  string `csv:"question_text"`
//     OptionA       string `csv:"option_a"`
//     OptionB       string `csv:"option_b"`
//     OptionC       string `csv:"option_c"`
//     OptionD       string `csv:"option_d"`
//     CorrectAnswer string `csv:"correct_answer"`
//     Explanation   string `csv:"explanation"`
//     Marks         int    `csv:"marks"`
//     Topic         string `csv:"topic"`
//     SubTopic      string `csv:"sub_topic"`
//     Difficulty    string `csv:"difficulty"`
//     BloomLevel    string `csv:"bloom_level"`
//     QuestionType  string `csv:"question_type"`
//     SubjectID     string `csv:"subject_id"`
// }

// // JSONQuestion for bulk JSON imports
// type JSONQuestion struct {
//     QuestionText  string `json:"question_text"`
//     OptionA       string `json:"option_a"`
//     OptionB       string `json:"option_b"`
//     OptionC       string `json:"option_c"`
//     OptionD       string `json:"option_d"`
//     CorrectAnswer string `json:"correct_answer"`
//     Explanation   string `json:"explanation"`
//     Marks         int    `json:"marks"`
//     Topic         string `json:"topic"`
//     SubTopic      string `json:"sub_topic"`
//     Difficulty    string `json:"difficulty"`
//     BloomLevel    string `json:"bloom_level"`
//     QuestionType  string `json:"question_type"`
//     SubjectID     string `json:"subject_id"`
// }

// type JSONQuestionImport struct {
//     Questions []JSONQuestion `json:"questions"`
// }

// type BulkUploadResponse struct {
//     TotalProcessed int      `json:"total_processed"`
//     SuccessCount   int      `json:"success_count"`
//     FailedCount    int      `json:"failed_count"`
//     Errors         []string `json:"errors,omitempty"`
// }








