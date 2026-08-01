package dto

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ============================================
// CONSTANTS
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

	// Exam Types - Nigerian Academic Context
	ExamTypeWeeklyTest = "weekly_test"
	ExamTypeMidTerm    = "mid_term"
	ExamTypeMainExam   = "main_exam"
	ExamTypePractice   = "practice"

	// ============================================================
	// FILE UPLOAD FORMATS
	// ============================================================
	UploadFormatAuto  = "auto"
	UploadFormatCSV   = "csv"
	UploadFormatExcel = "excel"
	UploadFormatJSON  = "json"
	UploadFormatDOCX  = "docx"
	UploadFormatTXT   = "txt"

	// Excel sub-formats (for debugging)
	UploadFormatXLSX = "xlsx"
	UploadFormatXLS  = "xls"

	// ============================================================
	// TEXT MODES FOR TXT PARSING
	// ============================================================
	TextModePlain    = "plain"
	TextModeQA       = "qa"
	TextModeNumbered = "numbered"

	// ============================================================
	// VALIDATION LIMITS
	// ============================================================
	MaxMarks          = 1000
	MinMarks          = 1
	MaxQuestionLength = 10000
	MaxOptionsCount   = 10
	MaxTagsCount      = 10
	MaxRubricCriteria = 10
)

// ============================================
// REQUEST DTOs
// ============================================

// CreateQuestionRequest represents the request to create a new question
type CreateQuestionRequest struct {
	SchoolID          string            `json:"school_id" binding:"required,uuid"`
	SessionID         string            `json:"session_id" binding:"required,uuid"`
	TermID            string            `json:"term_id" binding:"required,uuid"`
	ClassLevelID      string            `json:"class_level_id" binding:"required,uuid"`
	ClassID           string            `json:"class_id" binding:"required,uuid"`
	SubjectID         string            `json:"subject_id" binding:"required,uuid"`
	ExamType          string            `json:"exam_type" binding:"required,oneof=weekly_test mid_term main_exam practice"`
	QuestionText      string            `json:"question_text" binding:"required"`
	QuestionType      string            `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
	Difficulty        string            `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel        string            `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	Marks             int               `json:"marks" binding:"required,min=1"`
	Topic             string            `json:"topic" binding:"required"`
	SubTopic          string            `json:"sub_topic"`
	CorrectAnswer     string            `json:"correct_answer"`
	OptionsArray      []QuestionOption  `json:"options_array,omitempty"`
	Options           map[string]string `json:"options,omitempty"`
	CorrectOptionKeys []string          `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria  `json:"rubric,omitempty"`
	Explanation       string            `json:"explanation"`
	TimeLimitSeconds  *int              `json:"time_limit_seconds"`
	Tags              []string          `json:"tags"`
	CurriculumType    string            `json:"curriculum_type"`
	SourceType        string            `json:"source_type"`
	ExternalID        string            `json:"external_id"`
	LearningObjective string            `json:"learning_objective"`
	NegativeMarks     float64           `json:"negative_marks"`
	Order             int               `json:"order"`
	IsRequired        bool              `json:"is_required"`
}

// UpdateQuestionRequest represents the request to update a question
type UpdateQuestionRequest struct {
	SchoolID          *string           `json:"school_id,omitempty" binding:"omitempty,uuid"`
	SessionID         *string           `json:"session_id,omitempty" binding:"omitempty,uuid"`
	TermID            *string           `json:"term_id,omitempty" binding:"omitempty,uuid"`
	ClassLevelID      *string           `json:"class_level_id,omitempty" binding:"omitempty,uuid"`
	ClassID           *string           `json:"class_id,omitempty" binding:"omitempty,uuid"`
	SubjectID         *string           `json:"subject_id,omitempty" binding:"omitempty,uuid"`
	ExamType          *string           `json:"exam_type,omitempty" binding:"omitempty,oneof=weekly_test mid_term main_exam practice"`
	QuestionText      *string           `json:"question_text"`
	Topic             *string           `json:"topic"`
	SubTopic          *string           `json:"sub_topic"`
	CorrectAnswer     *string           `json:"correct_answer"`
	Explanation       *string           `json:"explanation"`
	Marks             *int              `json:"marks"`
	Difficulty        *string           `json:"difficulty"`
	BloomLevel        *string           `json:"bloom_level"`
	TimeLimitSeconds  *int              `json:"time_limit_seconds"`
	Status            *string           `json:"status"`
	CurriculumType    *string           `json:"curriculum_type"`
	SourceType        *string           `json:"source_type"`
	LearningObjective *string           `json:"learning_objective"`
	NegativeMarks     *float64          `json:"negative_marks"`
	Order             *int              `json:"order"`
	IsRequired        *bool             `json:"is_required"`
	OptionsArray      []QuestionOption  `json:"options_array,omitempty"`
	Options           map[string]string `json:"options,omitempty"`
	CorrectOptionKeys []string          `json:"correct_option_keys,omitempty"`
	Rubric            []RubricCriteria  `json:"rubric,omitempty"`
}

// FilterQuestionsRequest represents advanced filter criteria
type FilterQuestionsRequest struct {
	SchoolID     string   `json:"school_id"`
	SessionID    string   `json:"session_id"`
	TermID       string   `json:"term_id"`
	ClassLevelID string   `json:"class_level_id"`
	ClassID      string   `json:"class_id"`
	SubjectID    string   `json:"subject_id"`
	ExamType     string   `json:"exam_type"`
	Topic        string   `json:"topic"`
	Difficulty   []string `json:"difficulty"`
	BloomLevel   []string `json:"bloom_level"`
	QuestionType []string `json:"question_type"`
	Tags         []string `json:"tags"`
	Status       string   `json:"status"`
	Search       string   `json:"search"`
	Page         int      `json:"page"`
	Limit        int      `json:"limit"`
}

// FilterQuestionsWithContextRequest - Extended filter with full context
type FilterQuestionsWithContextRequest struct {
	SchoolID         string   `json:"school_id" binding:"required,uuid"`
	SubjectID        string   `json:"subject_id" binding:"required,uuid"`
	SessionID        string   `json:"session_id,omitempty"`
	TermID           string   `json:"term_id,omitempty"`
	ClassLevelID     string   `json:"class_level_id,omitempty"`
	ClassID          string   `json:"class_id,omitempty"`
	ExamType         string   `json:"exam_type,omitempty"`
	IsCurrentTerm    *bool    `json:"is_current_term,omitempty"`
	IsCurrentSession *bool    `json:"is_current_session,omitempty"`
	Topic            string   `json:"topic"`
	Difficulty       []string `json:"difficulty"`
	BloomLevel       []string `json:"bloom_level"`
	QuestionType     []string `json:"question_type"`
	Status           string   `json:"status"`
	Search           string   `json:"search"`
	Page             int      `json:"page"`
	Limit            int      `json:"limit"`
}

// BulkCreateQuestionRequest represents bulk question creation
type BulkCreateQuestionRequest struct {
	Questions []CreateQuestionRequest `json:"questions" binding:"required"`
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

// BulkUpdateStatusRequest represents bulk status update request
type BulkUpdateStatusRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required"`
	Status      string   `json:"status" binding:"required"`
}

// ============================================================
// BULK UPLOAD REQUEST - SUPPORTS ALL FORMATS
// ============================================================

// BulkUploadRequest represents file upload request with FULL academic context
type BulkUploadRequest struct {
	SchoolID           string                `form:"school_id" binding:"required,uuid"`
	SessionID          string                `form:"session_id" binding:"required,uuid"`
	TermID             string                `form:"term_id" binding:"required,uuid"`
	ClassLevelID       string                `form:"class_level_id" binding:"required,uuid"`
	ClassID            string                `form:"class_id" binding:"required,uuid"`
	SubjectID          string                `form:"subject_id" binding:"required,uuid"`
	ExamType           string                `form:"exam_type" binding:"required,oneof=weekly_test mid_term main_exam practice"`
	File               *multipart.FileHeader `form:"file" binding:"required"`
	Format             string                `form:"format" binding:"omitempty,oneof=auto csv excel json docx txt"`
	HasHeader          bool                  `form:"has_header"`
	SheetName          string                `form:"sheet_name"`
	TextMode           string                `form:"text_mode" binding:"omitempty,oneof=plain qa numbered"`
	PreserveFormatting bool                  `form:"preserve_formatting"`
	CurriculumType     string                `form:"curriculum_type"`
}

// ============================================================
// SIMPLIFIED CSV ROWS (Teacher-Friendly)
// ============================================================

// SimplifiedCSVQuestionRow - Teacher fills only content
type SimplifiedCSVQuestionRow struct {
	QuestionText  string `csv:"question_text"`
	Marks         int    `csv:"marks"`
	Topic         string `csv:"topic"`
	SubTopic      string `csv:"sub_topic"`
	OptionA       string `csv:"option_a"`
	OptionB       string `csv:"option_b"`
	OptionC       string `csv:"option_c"`
	OptionD       string `csv:"option_d"`
	CorrectAnswer string `csv:"correct_answer"`
	Explanation   string `csv:"explanation"`
	Tags          string `csv:"tags"`
}

// FullCSVQuestionRow - Full CSV with all fields
type FullCSVQuestionRow struct {
	SchoolID      string `csv:"school_id"`
	SessionID     string `csv:"session_id"`
	TermID        string `csv:"term_id"`
	ClassLevelID  string `csv:"class_level_id"`
	ClassID       string `csv:"class_id"`
	SubjectID     string `csv:"subject_id"`
	ExamType      string `csv:"exam_type"`
	QuestionText  string `csv:"question_text"`
	QuestionType  string `csv:"question_type"`
	Difficulty    string `csv:"difficulty"`
	BloomLevel    string `csv:"bloom_level"`
	Marks         int    `csv:"marks"`
	Topic         string `csv:"topic"`
	SubTopic      string `csv:"sub_topic"`
	OptionA       string `csv:"option_a"`
	OptionB       string `csv:"option_b"`
	OptionC       string `csv:"option_c"`
	OptionD       string `csv:"option_d"`
	CorrectAnswer string `csv:"correct_answer"`
	Explanation   string `csv:"explanation"`
	Tags          string `csv:"tags"`
}

// ============================================================
// FILE TEMPLATE DTOS
// ============================================================

// CSVTemplateResponse represents CSV template structure
type CSVTemplateResponse struct {
	Headers           []string          `json:"headers"`
	ExampleRow        map[string]string `json:"example_row"`
	RequiredFields    []string          `json:"required_fields"`
	FieldDescriptions map[string]string `json:"field_descriptions"`
}

// DOCXTemplateResponse represents DOCX template structure
type DOCXTemplateResponse struct {
	TemplateContent string   `json:"template_content"`
	Instructions    []string `json:"instructions"`
	Example         string   `json:"example"`
}

// TXTTemplateResponse represents TXT template structure
type TXTTemplateResponse struct {
	TemplateContent string   `json:"template_content"`
	Instructions    []string `json:"instructions"`
	Example         string   `json:"example"`
}

// ============================================================
// FORMAT DETECTION HELPERS
// ============================================================

// DetectFormatFromExtension detects the format from file extension
func DetectFormatFromExtension(filename string) string {
	if filename == "" {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return UploadFormatCSV
	case ".xlsx", ".xls":
		return UploadFormatExcel
	case ".json":
		return UploadFormatJSON
	case ".docx":
		return UploadFormatDOCX
	case ".txt":
		return UploadFormatTXT
	default:
		return ""
	}
}

// ShouldAutoDetect returns true if format should be auto-detected
func ShouldAutoDetect(format string) bool {
	return format == UploadFormatAuto || format == ""
}

// IsValidFormat returns true if format is valid
func IsValidFormat(format string) bool {
	validFormats := map[string]bool{
		UploadFormatAuto:  true,
		UploadFormatCSV:   true,
		UploadFormatExcel: true,
		UploadFormatJSON:  true,
		UploadFormatDOCX:  true,
		UploadFormatTXT:   true,
	}
	return validFormats[format]
}

// GetFormatFromRequest determines the actual format to use
func GetFormatFromRequest(format string, filename string) (string, error) {
	if ShouldAutoDetect(format) {
		detected := DetectFormatFromExtension(filename)
		if detected == "" {
			return "", fmt.Errorf("unable to detect format from file '%s'. Please specify format explicitly (csv, excel, json, docx, txt)", filename)
		}
		return detected, nil
	}
	if !IsValidFormat(format) {
		return "", fmt.Errorf("invalid format '%s'. Supported formats: csv, excel, json, docx, txt", format)
	}
	return format, nil
}

// ============================================================
// PARSER INTERFACE
// ============================================================

// QuestionParser defines the interface for parsing question files
type QuestionParser interface {
	Parse(file io.Reader) ([]QuestionImportItem, error)
	SupportedFormat() string
}

// ============================================================
// PARSER IMPLEMENTATIONS
// ============================================================

// ============================================================
// CSV PARSER
// ============================================================

// CSVParser parses CSV files
type CSVParser struct {
	hasHeader bool
}

// NewCSVParser creates a new CSV parser
func NewCSVParser() *CSVParser {
	return &CSVParser{hasHeader: true}
}

func (p *CSVParser) SupportedFormat() string {
	return UploadFormatCSV
}

func (p *CSVParser) Parse(file io.Reader) ([]QuestionImportItem, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have header + at least one data row")
	}

	header := records[0]
	colMap := p.buildColumnMap(header)

	var questions []QuestionImportItem
	for i := 1; i < len(records); i++ {
		q, err := p.parseRow(records[i], colMap)
		if err != nil {
			continue
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (p *CSVParser) buildColumnMap(header []string) map[string]int {
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return colMap
}

func (p *CSVParser) parseRow(row []string, colMap map[string]int) (QuestionImportItem, error) {
	q := QuestionImportItem{}
	q.QuestionText = p.getColumn(row, colMap, "question_text")
	q.QuestionType = p.getColumn(row, colMap, "question_type")
	q.Difficulty = p.getColumn(row, colMap, "difficulty")
	q.BloomLevel = p.getColumn(row, colMap, "bloom_level")
	q.Topic = p.getColumn(row, colMap, "topic")
	q.SubTopic = p.getColumn(row, colMap, "sub_topic")
	q.Explanation = p.getColumn(row, colMap, "explanation")
	q.Marks = p.parseInt(p.getColumn(row, colMap, "marks"))

	options := []QuestionOption{
		{Key: "A", Text: p.getColumn(row, colMap, "option_a")},
		{Key: "B", Text: p.getColumn(row, colMap, "option_b")},
		{Key: "C", Text: p.getColumn(row, colMap, "option_c")},
		{Key: "D", Text: p.getColumn(row, colMap, "option_d")},
	}
	for _, opt := range options {
		if opt.Text != "" {
			q.Options = append(q.Options, opt)
		}
	}

	correctAnswer := p.getColumn(row, colMap, "correct_answer")
	if correctAnswer != "" {
		q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
	}

	tags := p.getColumn(row, colMap, "tags")
	if tags != "" {
		q.Tags = strings.Split(tags, ",")
		for i, tag := range q.Tags {
			q.Tags[i] = strings.TrimSpace(tag)
		}
	}

	p.setDefaults(&q)
	return q, nil
}

func (p *CSVParser) getColumn(row []string, colMap map[string]int, colName string) string {
	if idx, ok := colMap[colName]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func (p *CSVParser) parseInt(str string) int {
	if str == "" {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(str))
	if err != nil {
		return 0
	}
	return val
}

func (p *CSVParser) setDefaults(q *QuestionImportItem) {
	if q.QuestionType == "" {
		q.QuestionType = QuestionTypeSingle
	}
	if q.Difficulty == "" {
		q.Difficulty = DifficultyMedium
	}
	if q.BloomLevel == "" {
		q.BloomLevel = BloomRemember
	}
	if q.Marks == 0 {
		if q.QuestionType == QuestionTypeTrueFalse || q.QuestionType == QuestionTypeFillBlank {
			q.Marks = 1
		} else {
			q.Marks = 2
		}
	}
}

// ============================================================
// EXCEL PARSER
// ============================================================

// ExcelParser parses Excel files (.xlsx, .xls)
type ExcelParser struct {
	sheetName string
	hasHeader bool
}

// NewExcelParser creates a new Excel parser
func NewExcelParser() *ExcelParser {
	return &ExcelParser{hasHeader: true}
}

func (p *ExcelParser) SupportedFormat() string {
	return UploadFormatExcel
}

func (p *ExcelParser) Parse(file io.Reader) ([]QuestionImportItem, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel file has no sheets")
	}

	sheetName := p.sheetName
	if sheetName == "" {
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet '%s': %w", sheetName, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel sheet must have header + at least one data row")
	}

	header := rows[0]
	colMap := p.buildColumnMap(header)

	var questions []QuestionImportItem
	for i := 1; i < len(rows); i++ {
		q, err := p.parseRow(rows[i], colMap)
		if err != nil {
			continue
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (p *ExcelParser) buildColumnMap(header []string) map[string]int {
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return colMap
}

func (p *ExcelParser) parseRow(row []string, colMap map[string]int) (QuestionImportItem, error) {
	q := QuestionImportItem{}
	q.QuestionText = p.getColumn(row, colMap, "question_text")
	q.QuestionType = p.getColumn(row, colMap, "question_type")
	q.Difficulty = p.getColumn(row, colMap, "difficulty")
	q.BloomLevel = p.getColumn(row, colMap, "bloom_level")
	q.Topic = p.getColumn(row, colMap, "topic")
	q.SubTopic = p.getColumn(row, colMap, "sub_topic")
	q.Explanation = p.getColumn(row, colMap, "explanation")
	q.Marks = p.parseInt(p.getColumn(row, colMap, "marks"))

	options := []QuestionOption{
		{Key: "A", Text: p.getColumn(row, colMap, "option_a")},
		{Key: "B", Text: p.getColumn(row, colMap, "option_b")},
		{Key: "C", Text: p.getColumn(row, colMap, "option_c")},
		{Key: "D", Text: p.getColumn(row, colMap, "option_d")},
	}
	for _, opt := range options {
		if opt.Text != "" {
			q.Options = append(q.Options, opt)
		}
	}

	correctAnswer := p.getColumn(row, colMap, "correct_answer")
	if correctAnswer != "" {
		q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
	}

	tags := p.getColumn(row, colMap, "tags")
	if tags != "" {
		q.Tags = strings.Split(tags, ",")
		for i, tag := range q.Tags {
			q.Tags[i] = strings.TrimSpace(tag)
		}
	}

	p.setDefaults(&q)
	return q, nil
}

func (p *ExcelParser) getColumn(row []string, colMap map[string]int, colName string) string {
	if idx, ok := colMap[colName]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func (p *ExcelParser) parseInt(str string) int {
	if str == "" {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(str))
	if err != nil {
		return 0
	}
	return val
}

func (p *ExcelParser) setDefaults(q *QuestionImportItem) {
	if q.QuestionType == "" {
		q.QuestionType = QuestionTypeSingle
	}
	if q.Difficulty == "" {
		q.Difficulty = DifficultyMedium
	}
	if q.BloomLevel == "" {
		q.BloomLevel = BloomRemember
	}
	if q.Marks == 0 {
		if q.QuestionType == QuestionTypeTrueFalse || q.QuestionType == QuestionTypeFillBlank {
			q.Marks = 1
		} else {
			q.Marks = 2
		}
	}
}

// ============================================================
// JSON PARSER
// ============================================================

// JSONParser parses JSON files
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (p *JSONParser) SupportedFormat() string {
	return UploadFormatJSON
}

func (p *JSONParser) Parse(file io.Reader) ([]QuestionImportItem, error) {
	var importData JSONQuestionImport
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&importData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(importData.Questions) == 0 {
		return nil, fmt.Errorf("no questions found in JSON")
	}

	var questions []QuestionImportItem
	for _, q := range importData.Questions {
		item := QuestionImportItem{
			Topic:        q.Topic,
			SubTopic:     q.SubTopic,
			QuestionText: q.QuestionText,
			QuestionType: q.QuestionType,
			Difficulty:   q.Difficulty,
			BloomLevel:   q.BloomLevel,
			Marks:        q.Marks,
			Explanation:  q.Explanation,
		}

		options := []QuestionOption{}
		if q.OptionA != "" {
			options = append(options, QuestionOption{Key: "A", Text: q.OptionA})
		}
		if q.OptionB != "" {
			options = append(options, QuestionOption{Key: "B", Text: q.OptionB})
		}
		if q.OptionC != "" {
			options = append(options, QuestionOption{Key: "C", Text: q.OptionC})
		}
		if q.OptionD != "" {
			options = append(options, QuestionOption{Key: "D", Text: q.OptionD})
		}
		item.Options = options

		if q.CorrectAnswer != "" {
			item.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(q.CorrectAnswer))}
		}

		if q.Tags != "" {
			item.Tags = strings.Split(q.Tags, ",")
			for i, tag := range item.Tags {
				item.Tags[i] = strings.TrimSpace(tag)
			}
		}

		p.setDefaults(&item)
		questions = append(questions, item)
	}

	return questions, nil
}

func (p *JSONParser) setDefaults(q *QuestionImportItem) {
	if q.QuestionType == "" {
		q.QuestionType = QuestionTypeSingle
	}
	if q.Difficulty == "" {
		q.Difficulty = DifficultyMedium
	}
	if q.BloomLevel == "" {
		q.BloomLevel = BloomRemember
	}
	if q.Marks == 0 {
		if q.QuestionType == QuestionTypeTrueFalse || q.QuestionType == QuestionTypeFillBlank {
			q.Marks = 1
		} else {
			q.Marks = 2
		}
	}
}

// ============================================================
// DOCX PARSER - COMPLETE FIXED VERSION
// ============================================================

// DOCXParser parses DOCX files
type DOCXParser struct {
	preserveFormatting bool
}

// NewDOCXParser creates a new DOCX parser
func NewDOCXParser() *DOCXParser {
	return &DOCXParser{}
}

func (p *DOCXParser) SupportedFormat() string {
	return UploadFormatDOCX
}

func (p *DOCXParser) Parse(file io.Reader) ([]QuestionImportItem, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX file: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX as zip: %w", err)
	}

	var documentXML []byte
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			documentXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if documentXML == nil {
		return nil, fmt.Errorf("no document.xml found in DOCX file")
	}

	text := p.extractTextFromDOCX(documentXML)

	if len(text) > 0 {
		log.Printf("DOCX extracted text (first 500 chars): %s...", text[:min(500, len(text))])
	} else {
		log.Println("DOCX extracted text is EMPTY!")
	}

	if text == "" {
		return nil, fmt.Errorf("no text content extracted from DOCX")
	}

	return p.parseQuestionsFromText(text)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *DOCXParser) extractTextFromDOCX(xmlData []byte) string {
	text := string(xmlData)
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (p *DOCXParser) parseQuestionsFromText(text string) ([]QuestionImportItem, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text content")
	}

	var questions []QuestionImportItem

	parts := regexp.MustCompile(`(?i)(?:Q(?:uestion)?\.?\s*\d+\.?\s*|\d+\.\s*)`).Split(text, -1)

	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}

	log.Printf("Found %d question parts in DOCX", len(parts))

	for idx, qText := range parts {
		qText = strings.TrimSpace(qText)
		if qText == "" || len(qText) < 5 {
			continue
		}

		log.Printf("Processing question %d: %s...", idx+1, qText[:min(50, len(qText))])

		q := QuestionImportItem{
			QuestionType: QuestionTypeSingle,
			Difficulty:   DifficultyMedium,
			BloomLevel:   BloomRemember,
			Marks:        2,
			Topic:        "General",
		}

		lines := strings.Split(qText, "\n")
		var questionLines []string
		var options []QuestionOption
		var correctAnswer string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			lowerLine := strings.ToLower(line)

			// Check for answer patterns
			if strings.Contains(lowerLine, "answer:") ||
				strings.Contains(lowerLine, "ans:") ||
				strings.Contains(lowerLine, "correct answer:") ||
				strings.Contains(lowerLine, "correct:") {

				log.Printf("Found answer line: %s", line)

				if strings.Contains(line, ":") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) > 1 {
						answer := strings.TrimSpace(parts[1])
						log.Printf("Extracted answer from colon: '%s'", answer)

						if len(answer) == 1 && answer >= "A" && answer <= "D" {
							correctAnswer = answer
							log.Printf("✅ Found single letter answer: %s", correctAnswer)
						} else {
							letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(answer)
							if len(letterMatch) > 1 {
								correctAnswer = strings.ToUpper(letterMatch[1])
								log.Printf("✅ Found letter in answer text: %s", correctAnswer)
							} else {
								correctAnswer = answer
								log.Printf("Using full answer text: %s", correctAnswer)
							}
						}
					}
				} else {
					lastChar := string(line[len(line)-1])
					if lastChar >= "A" && lastChar <= "D" {
						correctAnswer = lastChar
						log.Printf("✅ Found answer from last char: %s", correctAnswer)
					} else {
						letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(line)
						if len(letterMatch) > 1 {
							correctAnswer = strings.ToUpper(letterMatch[1])
							log.Printf("✅ Found letter from regex: %s", correctAnswer)
						}
					}
				}
				continue
			}

			// Option patterns
			if match := regexp.MustCompile(`^([A-Da-d])[\)\.]\s*(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			if match := regexp.MustCompile(`^([A-Da-d])\.\s*(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			if match := regexp.MustCompile(`^([A-Da-d])\s+(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			if match := regexp.MustCompile(`^\(([A-Da-d])\)\s*(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			questionLines = append(questionLines, line)
		}

		q.QuestionText = strings.Join(questionLines, " ")
		q.QuestionText = strings.TrimSpace(q.QuestionText)

		if q.QuestionText == "" {
			q.QuestionText = qText
		}

		log.Printf("Question text: %s", q.QuestionText[:min(50, len(q.QuestionText))])
		log.Printf("Found %d options, correctAnswer: '%s'", len(options), correctAnswer)

		// Set options and correct answer from regular parsing
		if len(options) > 0 {
			q.Options = options
			q.QuestionType = QuestionTypeSingle

			if correctAnswer != "" {
				if len(correctAnswer) == 1 && correctAnswer >= "A" && correctAnswer <= "D" {
					q.CorrectOptionKeys = []string{strings.ToUpper(correctAnswer)}
					log.Printf("✅ Set CorrectOptionKeys to: %v", q.CorrectOptionKeys)
				} else {
					letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(correctAnswer)
					if len(letterMatch) > 1 {
						q.CorrectOptionKeys = []string{strings.ToUpper(letterMatch[1])}
						log.Printf("✅ Set CorrectOptionKeys from letter match: %v", q.CorrectOptionKeys)
					} else {
						q.CorrectAnswer = correctAnswer
						log.Printf("✅ Set CorrectAnswer to: %s", q.CorrectAnswer)
					}
				}
			}
		}

		// Try to extract topic from question text
		if q.Topic == "General" {
			topicPatterns := []string{
				`[Tt]opic:\s*([^\n]+)`,
				`\[([^\]]+)\]`,
				`^([A-Za-z\s]+):`,
			}
			for _, pattern := range topicPatterns {
				re := regexp.MustCompile(pattern)
				if matches := re.FindStringSubmatch(q.QuestionText); len(matches) > 1 {
					potentialTopic := strings.TrimSpace(matches[1])
					if len(potentialTopic) > 0 && len(potentialTopic) < 50 {
						q.Topic = potentialTopic
						log.Printf("Extracted topic: %s", q.Topic)
						break
					}
				}
			}
		}

		// FALLBACK: try to extract options from question text
		if len(q.Options) == 0 && q.QuestionType == QuestionTypeSingle {
			optionPattern := regexp.MustCompile(`([A-Da-d])[\)\.]\s*([^A-Da-d]+)`)
			matches := optionPattern.FindAllStringSubmatch(q.QuestionText, -1)
			if len(matches) > 0 {
				var extractedOptions []QuestionOption
				for _, match := range matches {
					if len(match) > 2 {
						extractedOptions = append(extractedOptions, QuestionOption{
							Key:  strings.ToUpper(match[1]),
							Text: strings.TrimSpace(match[2]),
						})
					}
				}
				if len(extractedOptions) > 0 {
					q.Options = extractedOptions
					q.QuestionType = QuestionTypeSingle

					// ✅ FIX: Set correct option keys from the extracted answer
					if correctAnswer != "" {
						if len(correctAnswer) == 1 && correctAnswer >= "A" && correctAnswer <= "D" {
							q.CorrectOptionKeys = []string{strings.ToUpper(correctAnswer)}
							log.Printf("✅ Set CorrectOptionKeys from fallback: %v", q.CorrectOptionKeys)
						} else {
							letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(correctAnswer)
							if len(letterMatch) > 1 {
								q.CorrectOptionKeys = []string{strings.ToUpper(letterMatch[1])}
								log.Printf("✅ Set CorrectOptionKeys from fallback letter match: %v", q.CorrectOptionKeys)
							}
						}
					}
					log.Printf("Extracted %d options from question text", len(extractedOptions))
				}
			}
		}

		p.determineQuestionType(&q, options, correctAnswer)
		p.setDefaults(&q)

		log.Printf("Final question %d: options=%d, correctOptionKeys=%v", idx+1, len(q.Options), q.CorrectOptionKeys)

		questions = append(questions, q)
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no valid questions found in DOCX")
	}

	log.Printf("Successfully parsed %d questions from DOCX", len(questions))
	return questions, nil
}

func (p *DOCXParser) determineQuestionType(q *QuestionImportItem, options []QuestionOption, correctAnswer string) {
	if len(q.Options) > 0 {
		return
	}

	if len(options) == 2 {
		lowerA := strings.ToLower(options[0].Text)
		lowerB := strings.ToLower(options[1].Text)
		if (lowerA == "true" || lowerA == "t") && (lowerB == "false" || lowerB == "f") {
			q.QuestionType = QuestionTypeTrueFalse
			if strings.ToUpper(correctAnswer) == "TRUE" || strings.ToUpper(correctAnswer) == "T" || strings.ToUpper(correctAnswer) == "A" {
				q.CorrectOptionKeys = []string{"TRUE"}
			} else {
				q.CorrectOptionKeys = []string{"FALSE"}
			}
			return
		}
	}

	if len(options) > 0 {
		q.QuestionType = QuestionTypeSingle
		q.Options = options
		if correctAnswer != "" {
			q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
		}
		return
	}

	if strings.Contains(q.QuestionText, "_______") || strings.Contains(q.QuestionText, "___") {
		q.QuestionType = QuestionTypeFillBlank
		if correctAnswer != "" {
			q.CorrectAnswer = correctAnswer
		}
		return
	}

	q.QuestionType = QuestionTypeSingle
	if correctAnswer != "" {
		q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
	}
}

func (p *DOCXParser) setDefaults(q *QuestionImportItem) {
	if q.QuestionType == "" {
		q.QuestionType = QuestionTypeSingle
	}
	if q.Difficulty == "" {
		q.Difficulty = DifficultyMedium
	}
	if q.BloomLevel == "" {
		q.BloomLevel = BloomRemember
	}
	if q.Marks == 0 {
		if q.QuestionType == QuestionTypeTrueFalse || q.QuestionType == QuestionTypeFillBlank {
			q.Marks = 1
		} else {
			q.Marks = 2
		}
	}
	if q.Topic == "" {
		q.Topic = "General"
	}
}

// ============================================================
// TXT PARSER
// ============================================================

// TXTParser parses text files
type TXTParser struct {
	textMode string
}

// NewTXTParser creates a new TXT parser
func NewTXTParser() *TXTParser {
	return &TXTParser{textMode: TextModeQA}
}

func (p *TXTParser) SupportedFormat() string {
	return UploadFormatTXT
}

func (p *TXTParser) Parse(file io.Reader) ([]QuestionImportItem, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read TXT file: %w", err)
	}
	text := string(data)
	return p.parseQuestionsFromText(text)
}

func (p *TXTParser) parseQuestionsFromText(text string) ([]QuestionImportItem, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text content")
	}

	var questions []QuestionImportItem
	var parts []string

	switch p.textMode {
	case TextModeNumbered:
		parts = regexp.MustCompile(`(?i)(?:Q|Question)\s*\d+\.\s*`).Split(text, -1)
	case TextModePlain:
		parts = regexp.MustCompile(`\n\s*\n+`).Split(text, -1)
	default:
		parts = regexp.MustCompile(`(?i)(?:Q(?:uestion)?\.?\s*\d+\.?\s*|\d+\.\s*)`).Split(text, -1)
	}

	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}

	for _, qText := range parts {
		qText = strings.TrimSpace(qText)
		if qText == "" || len(qText) < 5 {
			continue
		}

		q := QuestionImportItem{
			QuestionType: QuestionTypeSingle,
			Difficulty:   DifficultyMedium,
			BloomLevel:   BloomRemember,
			Marks:        2,
			Topic:        "General",
		}

		lines := strings.Split(qText, "\n")
		var questionLines []string
		var options []QuestionOption
		var correctAnswer string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			lowerLine := strings.ToLower(line)

			if strings.Contains(lowerLine, "answer:") || strings.Contains(lowerLine, "ans:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) > 1 {
					answer := strings.TrimSpace(parts[1])
					if len(answer) == 1 && answer >= "A" && answer <= "D" {
						correctAnswer = answer
					} else {
						letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(answer)
						if len(letterMatch) > 1 {
							correctAnswer = strings.ToUpper(letterMatch[1])
						} else {
							correctAnswer = answer
						}
					}
				}
				continue
			}

			if match := regexp.MustCompile(`^([A-Da-d])[\)\.]\s*(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			if match := regexp.MustCompile(`^([A-Da-d])\s+(.*)`).FindStringSubmatch(line); len(match) > 2 {
				options = append(options, QuestionOption{
					Key:  strings.ToUpper(match[1]),
					Text: match[2],
				})
				continue
			}

			questionLines = append(questionLines, line)
		}

		q.QuestionText = strings.Join(questionLines, " ")
		q.QuestionText = strings.TrimSpace(q.QuestionText)

		if q.QuestionText == "" {
			q.QuestionText = qText
		}

		if len(options) > 0 {
			q.Options = options
			q.QuestionType = QuestionTypeSingle
			if correctAnswer != "" {
				if len(correctAnswer) == 1 && correctAnswer >= "A" && correctAnswer <= "D" {
					q.CorrectOptionKeys = []string{strings.ToUpper(correctAnswer)}
				} else {
					letterMatch := regexp.MustCompile(`([A-Da-d])`).FindStringSubmatch(correctAnswer)
					if len(letterMatch) > 1 {
						q.CorrectOptionKeys = []string{strings.ToUpper(letterMatch[1])}
					} else {
						q.CorrectAnswer = correctAnswer
					}
				}
			}
		}

		// Try to extract topic from question text
		if q.Topic == "General" {
			topicPatterns := []string{
				`[Tt]opic:\s*([^\n]+)`,
				`\[([^\]]+)\]`,
				`^([A-Za-z\s]+):`,
			}
			for _, pattern := range topicPatterns {
				re := regexp.MustCompile(pattern)
				if matches := re.FindStringSubmatch(q.QuestionText); len(matches) > 1 {
					potentialTopic := strings.TrimSpace(matches[1])
					if len(potentialTopic) > 0 && len(potentialTopic) < 50 {
						q.Topic = potentialTopic
						break
					}
				}
			}
		}

		p.determineQuestionType(&q, options, correctAnswer)
		p.setDefaults(&q)
		questions = append(questions, q)
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no valid questions found in text")
	}

	return questions, nil
}

func (p *TXTParser) determineQuestionType(q *QuestionImportItem, options []QuestionOption, correctAnswer string) {
	if len(options) == 2 {
		lowerA := strings.ToLower(options[0].Text)
		lowerB := strings.ToLower(options[1].Text)
		if (lowerA == "true" || lowerA == "t") && (lowerB == "false" || lowerB == "f") {
			q.QuestionType = QuestionTypeTrueFalse
			if strings.ToUpper(correctAnswer) == "TRUE" || strings.ToUpper(correctAnswer) == "T" || strings.ToUpper(correctAnswer) == "A" {
				q.CorrectOptionKeys = []string{"TRUE"}
			} else {
				q.CorrectOptionKeys = []string{"FALSE"}
			}
			return
		}
	}

	if len(options) > 0 {
		q.QuestionType = QuestionTypeSingle
		q.Options = options
		if correctAnswer != "" {
			q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
		}
		return
	}

	if strings.Contains(q.QuestionText, "_______") || strings.Contains(q.QuestionText, "___") {
		q.QuestionType = QuestionTypeFillBlank
		if correctAnswer != "" {
			q.CorrectAnswer = correctAnswer
		}
		return
	}

	q.QuestionType = QuestionTypeSingle
	if correctAnswer != "" {
		q.CorrectOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
	}
}

func (p *TXTParser) setDefaults(q *QuestionImportItem) {
	if q.QuestionType == "" {
		q.QuestionType = QuestionTypeSingle
	}
	if q.Difficulty == "" {
		q.Difficulty = DifficultyMedium
	}
	if q.BloomLevel == "" {
		q.BloomLevel = BloomRemember
	}
	if q.Marks == 0 {
		if q.QuestionType == QuestionTypeTrueFalse || q.QuestionType == QuestionTypeFillBlank {
			q.Marks = 1
		} else {
			q.Marks = 2
		}
	}
	if q.Topic == "" {
		q.Topic = "General"
	}
}

// ============================================================
// PARSER FACTORY
// ============================================================

// ParserFactory creates the appropriate parser based on format
type ParserFactory struct{}

// NewParserFactory creates a new parser factory
func NewParserFactory() *ParserFactory {
	return &ParserFactory{}
}

// GetParser returns the appropriate parser for the given format
func (f *ParserFactory) GetParser(format string) (QuestionParser, error) {
	switch format {
	case UploadFormatCSV:
		return NewCSVParser(), nil
	case UploadFormatExcel:
		return NewExcelParser(), nil
	case UploadFormatJSON:
		return NewJSONParser(), nil
	case UploadFormatDOCX:
		return NewDOCXParser(), nil
	case UploadFormatTXT:
		return NewTXTParser(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ============================================================
// AI & EXTRACT DTOs
// ============================================================

// AIGenerateQuestionsRequest represents AI question generation request
type AIGenerateQuestionsRequest struct {
	SchoolID          string   `json:"school_id" binding:"required,uuid"`
	SessionID         string   `json:"session_id" binding:"required,uuid"`
	TermID            string   `json:"term_id" binding:"required,uuid"`
	ClassLevelID      string   `json:"class_level_id" binding:"required,uuid"`
	ClassID           string   `json:"class_id" binding:"required,uuid"`
	SubjectID         string   `json:"subject_id" binding:"required,uuid"`
	ExamType          string   `json:"exam_type" binding:"required,oneof=weekly_test mid_term main_exam practice"`
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
	SessionID    string `json:"session_id" binding:"required,uuid"`
	TermID       string `json:"term_id" binding:"required,uuid"`
	ClassLevelID string `json:"class_level_id" binding:"required,uuid"`
	ClassID      string `json:"class_id" binding:"required,uuid"`
	SubjectID    string `json:"subject_id" binding:"required,uuid"`
	ExamType     string `json:"exam_type" binding:"required,oneof=weekly_test mid_term main_exam practice"`
	Text         string `json:"text" binding:"required"`
	Format       string `json:"format" binding:"required,oneof=plain markdown html"`
}

// BulkQuestionImportRequest represents bulk import from structured data
type BulkQuestionImportRequest struct {
	SchoolID       string               `json:"school_id" binding:"required,uuid"`
	SessionID      string               `json:"session_id" binding:"required,uuid"`
	TermID         string               `json:"term_id" binding:"required,uuid"`
	ClassLevelID   string               `json:"class_level_id" binding:"required,uuid"`
	ClassID        string               `json:"class_id" binding:"required,uuid"`
	SubjectID      string               `json:"subject_id" binding:"required,uuid"`
	ExamType       string               `json:"exam_type" binding:"required,oneof=weekly_test mid_term main_exam practice"`
	CurriculumType string               `json:"curriculum_type"`
	SourceType     string               `json:"source_type"`
	Status         string               `json:"status"`
	CreatedBy      string               `json:"created_by" binding:"required,uuid"`
	Questions      []QuestionImportItem `json:"questions" binding:"required,dive"`
}

// QuestionImportItem represents a single question in bulk import
type QuestionImportItem struct {
	ExternalID        string           `json:"external_id"`
	Topic             string           `json:"topic"`
	SubTopic          string           `json:"sub_topic"`
	LearningObjective string           `json:"learning_objective"`
	QuestionText      string           `json:"question_text" binding:"required"`
	QuestionType      string           `json:"question_type" binding:"required,oneof=single_choice multiple_choice true_false essay fill_blank"`
	Difficulty        string           `json:"difficulty" binding:"required,oneof=easy medium hard expert"`
	BloomLevel        string           `json:"bloom_level" binding:"required,oneof=remember understand apply analyse evaluate create"`
	Options           []QuestionOption `json:"options"`
	CorrectAnswer     string           `json:"correct_answer"`
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

// CloneQuestionRequest represents question cloning request
type CloneQuestionRequest struct {
	QuestionID string `json:"question_id" binding:"required,uuid"`
}

// AIParaphraseRequest represents AI paraphrasing request
type AIParaphraseRequest struct {
	QuestionID     string `json:"question_id" binding:"required,uuid"`
	VariationCount int    `json:"variation_count" binding:"required,min=1,max=5"`
}

// ============================================================
// RESPONSE DTOs
// ============================================================

// QuestionBankResponse represents the response for a question
type QuestionBankResponse struct {
	ID                string               `json:"id"`
	QuestionText      string               `json:"question_text"`
	QuestionType      string               `json:"question_type"`
	Difficulty        string               `json:"difficulty"`
	BloomLevel        string               `json:"bloom_level"`
	Marks             int                  `json:"marks"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	SchoolID          string               `json:"school_id"`
	SchoolName        string               `json:"school_name"`
	SessionID         string               `json:"session_id"`
	SessionName       string               `json:"session_name"`
	TermID            string               `json:"term_id"`
	TermName          string               `json:"term_name"`
	TermNumber        int                  `json:"term_number"`
	ClassLevelID      string               `json:"class_level_id"`
	ClassLevel        string               `json:"class_level"`
	ClassID           string               `json:"class_id"`
	ClassName         string               `json:"class_name"`
	SubjectID         string               `json:"subject_id"`
	SubjectName       string               `json:"subject_name"`
	ExamType          string               `json:"exam_type"`
	Topic             string               `json:"topic"`
	SubTopic          string               `json:"sub_topic"`
	LearningObjective string               `json:"learning_objective"`
	Options           []QuestionOption     `json:"options"`
	CorrectAnswer     string               `json:"correct_answer"`
	CorrectOptionKeys []string             `json:"correct_option_keys,omitempty"`
	Explanation       string               `json:"explanation"`
	Rubric            []RubricCriteria     `json:"rubric,omitempty"`
	Tags              []string             `json:"tags"`
	Status            string               `json:"status"`
	Version           int                  `json:"version"`
	UsageCount        int                  `json:"usage_count"`
	SuccessRate       *float64             `json:"success_rate,omitempty"`
	NegativeMarks     float64              `json:"negative_marks"`
	TimeLimitSeconds  *int                 `json:"time_limit_seconds,omitempty"`
	Order             int                  `json:"order"`
	IsRequired        bool                 `json:"is_required"`
	CurriculumType    string               `json:"curriculum_type"`
	SourceType        string               `json:"source_type"`
	ExternalID        string               `json:"external_id"`
	CreatedBy         string               `json:"created_by"`
	CreatedByName     string               `json:"created_by_name"`
	Attachments       []AttachmentResponse `json:"attachments,omitempty"`
}

// QuestionContextResponse - For displaying question with full academic context
type QuestionContextResponse struct {
	QuestionID       string    `json:"question_id"`
	QuestionText     string    `json:"question_text"`
	QuestionType     string    `json:"question_type"`
	Difficulty       string    `json:"difficulty"`
	BloomLevel       string    `json:"bloom_level"`
	Marks            int       `json:"marks"`
	CreatedAt        time.Time `json:"created_at"`
	SchoolID         string    `json:"school_id"`
	SchoolName       string    `json:"school_name"`
	SessionID        string    `json:"session_id"`
	SessionName      string    `json:"session_name"`
	TermID           string    `json:"term_id"`
	TermName         string    `json:"term_name"`
	TermNumber       int       `json:"term_number"`
	ClassLevelID     string    `json:"class_level_id"`
	ClassLevelName   string    `json:"class_level_name"`
	ClassID          string    `json:"class_id"`
	ClassName        string    `json:"class_name"`
	SubjectID        string    `json:"subject_id"`
	SubjectName      string    `json:"subject_name"`
	ExamType         string    `json:"exam_type"`
	IsCurrentTerm    bool      `json:"is_current_term"`
	IsCurrentSession bool      `json:"is_current_session"`
	Topic            string    `json:"topic"`
}

// QuestionTermGroupResponse - Questions grouped by term
type QuestionTermGroupResponse struct {
	TermID          string                    `json:"term_id"`
	TermName        string                    `json:"term_name"`
	TermNumber      int                       `json:"term_number"`
	SessionName     string                    `json:"session_name"`
	TotalCount      int                       `json:"total_count"`
	WeeklyTestCount int                       `json:"weekly_test_count"`
	MidTermCount    int                       `json:"mid_term_count"`
	MainExamCount   int                       `json:"main_exam_count"`
	PracticeCount   int                       `json:"practice_count"`
	Questions       []QuestionContextResponse `json:"questions"`
}

// QuestionContextSummaryResponse - Full summary by term
type QuestionContextSummaryResponse struct {
	SubjectID       string                      `json:"subject_id"`
	SubjectName     string                      `json:"subject_name"`
	SchoolID        string                      `json:"school_id"`
	SchoolName      string                      `json:"school_name"`
	TotalQuestions  int                         `json:"total_questions"`
	WeeklyTestCount int                         `json:"weekly_test_count"`
	MidTermCount    int                         `json:"mid_term_count"`
	MainExamCount   int                         `json:"main_exam_count"`
	PracticeCount   int                         `json:"practice_count"`
	TermGroups      []QuestionTermGroupResponse `json:"term_groups"`
	Topics          []TopicSummary              `json:"topics"`
}

// TopicSummary - Breakdown by topic
type TopicSummary struct {
	Topic      string `json:"topic"`
	TotalCount int    `json:"total_count"`
	WeeklyTest int    `json:"weekly_test"`
	MidTerm    int    `json:"mid_term"`
	MainExam   int    `json:"main_exam"`
	Practice   int    `json:"practice"`
}

// TermContext - Current term information
type TermContext struct {
	TermID      string `json:"term_id"`
	TermName    string `json:"term_name"`
	TermNumber  int    `json:"term_number"`
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
}

// QuestionListWithContextResponse - Response with term/session context
type QuestionListWithContextResponse struct {
	Questions    []QuestionBankResponse `json:"questions"`
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	TotalPages   int                    `json:"total_pages"`
	TermNames    map[string]string      `json:"term_names"`
	SessionNames map[string]string      `json:"session_names"`
	CurrentTerm  *TermContext           `json:"current_term,omitempty"`
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
	Status       string     `json:"status"`
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

// CurrentAcademicContextResponse - Current session and term for a school
type CurrentAcademicContextResponse struct {
	SchoolID    string `json:"school_id"`
	SchoolName  string `json:"school_name"`
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	SessionYear string `json:"session_year"`
	TermID      string `json:"term_id"`
	TermName    string `json:"term_name"`
	TermNumber  int    `json:"term_number"`
	IsCurrent   bool   `json:"is_current"`
	IsActive    bool   `json:"is_active"`
}

// ============================================================
// SUB-STRUCTURES
// ============================================================

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

// JSONQuestion represents a question in JSON import with FULL academic context
type JSONQuestion struct {
	SchoolID      string `json:"school_id"`
	SessionID     string `json:"session_id"`
	TermID        string `json:"term_id"`
	ClassLevelID  string `json:"class_level_id"`
	ClassID       string `json:"class_id"`
	SubjectID     string `json:"subject_id"`
	ExamType      string `json:"exam_type"`
	QuestionText  string `json:"question_text"`
	QuestionType  string `json:"question_type"`
	Difficulty    string `json:"difficulty"`
	BloomLevel    string `json:"bloom_level"`
	Marks         int    `json:"marks"`
	Topic         string `json:"topic"`
	SubTopic      string `json:"sub_topic"`
	OptionA       string `json:"option_a"`
	OptionB       string `json:"option_b"`
	OptionC       string `json:"option_c"`
	OptionD       string `json:"option_d"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation"`
	Tags          string `json:"tags"`
}

// JSONQuestionImport represents the structure for JSON import
type JSONQuestionImport struct {
	Questions []JSONQuestion `json:"questions"`
}

// ============================================================
// VALIDATION METHODS
// ============================================================

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Validate performs validation on CreateQuestionRequest
func (req *CreateQuestionRequest) Validate() error {
	if req.SchoolID == "" {
		return &ValidationError{Field: "school_id", Message: "school ID is required"}
	}
	if req.SessionID == "" {
		return &ValidationError{Field: "session_id", Message: "session ID is required"}
	}
	if req.TermID == "" {
		return &ValidationError{Field: "term_id", Message: "term ID is required"}
	}
	if req.ClassLevelID == "" {
		return &ValidationError{Field: "class_level_id", Message: "class level ID is required"}
	}
	if req.ClassID == "" {
		return &ValidationError{Field: "class_id", Message: "class ID is required"}
	}
	if req.SubjectID == "" {
		return &ValidationError{Field: "subject_id", Message: "subject ID is required"}
	}
	if req.ExamType == "" {
		return &ValidationError{Field: "exam_type", Message: "exam type is required"}
	}
	if req.QuestionText == "" {
		return &ValidationError{Field: "question_text", Message: "question text is required"}
	}
	if req.Topic == "" {
		return &ValidationError{Field: "topic", Message: "topic is required"}
	}
	if req.Marks < 1 {
		return &ValidationError{Field: "marks", Message: "marks must be at least 1"}
	}

	switch req.QuestionType {
	case "single_choice", "multiple_choice", "true_false":
		if len(req.OptionsArray) == 0 && len(req.Options) == 0 {
			return &ValidationError{Field: "options", Message: "options are required for MCQ questions"}
		}
		if len(req.CorrectOptionKeys) == 0 && req.CorrectAnswer == "" {
			return &ValidationError{Field: "correct_option_keys", Message: "correct answer is required for MCQ questions"}
		}
		if req.Rubric != nil && len(req.Rubric) > 0 {
			return &ValidationError{Field: "rubric", Message: "MCQ questions cannot have rubric"}
		}
	case "essay":
		if req.Rubric == nil || len(req.Rubric) == 0 {
			return &ValidationError{Field: "rubric", Message: "rubric is required for essay questions"}
		}
		if len(req.OptionsArray) > 0 || len(req.Options) > 0 {
			return &ValidationError{Field: "options", Message: "essay questions cannot have options"}
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
	if req.ExamType != nil {
		validExamTypes := []string{"weekly_test", "mid_term", "main_exam", "practice"}
		valid := false
		for _, e := range validExamTypes {
			if e == *req.ExamType {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{Field: "exam_type", Message: "invalid exam type"}
		}
	}
	return nil
}

// ============================================================
// ENTERPRISE QUESTION GROUPS RESPONSE DTO
// ADD THIS ENTIRE BLOCK TO THE END OF question_dto.go
// ============================================================

// QuestionGroupsResponse - Root response structure for grouped questions
type QuestionGroupsResponse struct {
	Status         bool                   `json:"status"`
	Message        string                 `json:"message"`
	School         SchoolContext          `json:"school"`
	Session        SessionContext         `json:"session"`
	Term           TermContext            `json:"term"` // Uses existing TermContext
	Class          ClassContext           `json:"class"`
	Subject        SubjectContext         `json:"subject"`
	QuestionGroups map[string]ExamGroup   `json:"question_groups"`
}

// SchoolContext - School information
type SchoolContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SessionContext - Session information
type SessionContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ClassContext - Class information
type ClassContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubjectContext - Subject information
type SubjectContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ExamGroup - Questions grouped by exam type
type ExamGroup struct {
	TotalQuestions int                       `json:"total_questions"`
	QuestionTypes  map[string][]QuestionItem `json:"question_types"`
}

// QuestionItem - Individual question with structured options
type QuestionItem struct {
	QuestionID        string            `json:"question_id"`
	QuestionText      string            `json:"question_text"`
	Options           map[string]string `json:"options,omitempty"`
	CorrectAnswer     string            `json:"correct_answer,omitempty"`
	CorrectOptionKeys []string          `json:"correct_option_keys,omitempty"`
	Difficulty        string            `json:"difficulty"`
	BloomLevel        string            `json:"bloom_level"`
	Marks             int               `json:"marks"`
	Topic             string            `json:"topic"`
	SubTopic          string            `json:"sub_topic,omitempty"`
	Explanation       string            `json:"explanation,omitempty"`
	Rubric            []RubricCriteria  `json:"rubric,omitempty"` // Uses existing RubricCriteria
}

// FilterQuestionsGroupedRequest - Extended filter for grouped response
type FilterQuestionsGroupedRequest struct {
	SchoolID         string   `json:"school_id" binding:"required,uuid"`
	SubjectID        string   `json:"subject_id" binding:"required,uuid"`
	SessionID        string   `json:"session_id,omitempty"`
	TermID           string   `json:"term_id,omitempty"`
	ClassLevelID     string   `json:"class_level_id,omitempty"`
	ClassID          string   `json:"class_id,omitempty"`
	ExamType         string   `json:"exam_type,omitempty"`
	IsCurrentTerm    *bool    `json:"is_current_term,omitempty"`
	IsCurrentSession *bool    `json:"is_current_session,omitempty"`
	Difficulty       []string `json:"difficulty"`
	BloomLevel       []string `json:"bloom_level"`
	QuestionType     []string `json:"question_type"`
	Topic            string   `json:"topic"`
	Status           string   `json:"status"`
	Search           string   `json:"search"`
}