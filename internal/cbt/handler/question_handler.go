package handler

import (
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/service"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================
// EXPORTED ERROR VARIABLES
// ============================================

var (
	ErrQuestionNotFound  = errors.New("question not found")
	ErrInvalidQuestionID = errors.New("invalid question ID")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrTagAlreadyExists  = errors.New("tag already exists")
	ErrValidationFailed  = errors.New("validation failed")
)

// QuestionHandler handles all question HTTP requests
type QuestionHandler struct {
	questionService *service.QuestionService
}

// NewQuestionHandler creates a new QuestionHandler instance
func NewQuestionHandler(s *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: s}
}

// ============================================
// CRUD OPERATIONS
// ============================================

// CreateQuestion godoc
// @Summary      Create a new question
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateQuestionRequest true "Question details"
// @Success      201  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions [post]
func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	var req dto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.CreateQuestion(ctx, &req, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Question created successfully",
		"data":    resp,
	})
}

// GetQuestion godoc
// @Summary      Get a single question
// @Tags         Questions
// @Produce      json
// @Param        id path string true "Question UUID"
// @Success      200  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/{id} [get]
func (h *QuestionHandler) GetQuestion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
		return
	}

	if err := h.validateUUID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetQuestion(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// UpdateQuestion godoc
// @Summary      Update a question
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        id path string true "Question UUID"
// @Param        request body dto.UpdateQuestionRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "data contains updated QuestionBankResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/{id} [put]
func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
		return
	}

	if err := h.validateUUID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req dto.UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.UpdateQuestion(ctx, id, &req, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Question updated successfully",
		"data":    resp,
	})
}

// DeleteQuestion godoc
// @Summary      Delete a question
// @Description  Soft‑delete a question (it will not be returned in lists)
// @Tags         Questions
// @Produce      json
// @Param        id path string true "Question UUID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/{id} [delete]
func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
		return
	}

	if err := h.validateUUID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()
	if err := h.questionService.DeleteQuestion(ctx, id, userID.(string)); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Question deleted successfully",
	})
}

// ============================================
// LISTING & FILTERING OPERATIONS
// ============================================

// ListQuestions godoc
// @Summary      List questions by subject
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions [get]
func (h *QuestionHandler) ListQuestions(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, err := h.parsePage(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := h.parseLimit(c.DefaultQuery("limit", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, total, err := h.questionService.ListQuestions(ctx, subjectID, page, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// FilterQuestions godoc
// @Summary      Advanced filter for questions
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.FilterQuestionsRequest true "Filter criteria"
// @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/filter [post]
func (h *QuestionHandler) FilterQuestions(c *gin.Context) {
	var req dto.FilterQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	ctx := c.Request.Context()
	resp, total, err := h.questionService.FilterQuestions(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
		"total":  total,
		"page":   req.Page,
		"limit":  req.Limit,
	})
}

// GetStatistics godoc
// @Summary      Get question statistics
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string false "Subject UUID (optional)"
// @Success      200  {object}  map[string]interface{}  "data (statistics)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/statistics [get]
func (h *QuestionHandler) GetStatistics(c *gin.Context) {
	subjectID := c.Query("subject_id")

	if subjectID != "" {
		if err := h.validateUUID(subjectID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	ctx := c.Request.Context()
	stats, err := h.questionService.GetStatistics(ctx, subjectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ============================================
// BULK OPERATIONS
// ============================================

// BulkCreateQuestions godoc
// @Summary      Bulk create questions from JSON array
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkCreateQuestionRequest true "Array of questions"
// @Success      201 {object} map[string]interface{} "data, count"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/bulk [post]
func (h *QuestionHandler) BulkCreateQuestions(c *gin.Context) {
	var req dto.BulkCreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(req.Questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no questions provided"})
		return
	}

	if len(req.Questions) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "too many questions",
			"details": "Maximum 1000 questions per bulk request",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.BulkCreateQuestionsFromJSON(ctx, &req, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Bulk questions created successfully",
		"data":    resp,
		"count":   len(resp),
	})
}

// BulkDelete godoc
// @Summary      Delete multiple questions
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkDeleteRequest true "List of question IDs"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/bulk [delete]
func (h *QuestionHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(req.QuestionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question_ids cannot be empty"})
		return
	}

	for _, id := range req.QuestionIDs {
		if err := h.validateUUID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid question ID",
				"details": err.Error(),
				"id":      id,
			})
			return
		}
	}

	ctx := c.Request.Context()
	if err := h.questionService.BulkDelete(ctx, &req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Bulk delete successful",
	})
}

// BulkUpdateStatus godoc
// @Summary      Bulk update question status
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkUpdateStatusRequest true "Question IDs and status"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/bulk/status [put]
func (h *QuestionHandler) BulkUpdateStatus(c *gin.Context) {
	var req dto.BulkUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(req.QuestionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question_ids cannot be empty"})
		return
	}

	if req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	for _, id := range req.QuestionIDs {
		if err := h.validateUUID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid question ID",
				"details": err.Error(),
				"id":      id,
			})
			return
		}
	}

	ctx := c.Request.Context()
	if err := h.questionService.BulkUpdateStatus(ctx, req.QuestionIDs, req.Status); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Bulk status update successful",
	})
}

// ============================================
// BULK UPLOAD FROM FILE
// ============================================

// BulkUploadFile godoc
// @Summary      Bulk upload questions from file
// @Description  Upload questions from CSV, Excel, JSON, DOCX, or TXT files with full academic context
// @Tags         Questions
// @Accept       multipart/form-data
// @Produce      json
// @Param        school_id formData string true "School UUID"
// @Param        session_id formData string true "Academic Session UUID"
// @Param        term_id formData string true "Term UUID"
// @Param        class_level_id formData string true "Class Level UUID"
// @Param        class_id formData string true "Class UUID"
// @Param        subject_id formData string true "Subject UUID"
// @Param        exam_type formData string true "Exam Type: weekly_test, mid_term, main_exam, practice"
// @Param        file formData file true "File (CSV, Excel, JSON, DOCX, TXT)"
// @Param        format formData string false "File format: auto, csv, excel, json, docx, txt (default auto)"
// @Param        has_header formData bool false "CSV has header (default true)"
// @Param        sheet_name formData string false "Excel sheet name (default first sheet)"
// @Param        text_mode formData string false "Text parsing mode: plain, qa, numbered (default qa)"
// @Param        preserve_formatting formData bool false "Preserve formatting for DOCX/TXT"
// @Param        curriculum_type formData string false "Curriculum type"
// @Success      200 {object} dto.BulkUploadResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      413 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/bulk-upload [post]
func (h *QuestionHandler) BulkUploadFile(c *gin.Context) {
	// Parse ALL required fields
	schoolID := c.PostForm("school_id")
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
		return
	}

	sessionID := c.PostForm("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	termID := c.PostForm("term_id")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	classLevelID := c.PostForm("class_level_id")
	if classLevelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_level_id is required"})
		return
	}

	classID := c.PostForm("class_id")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	subjectID := c.PostForm("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	examType := c.PostForm("exam_type")
	if examType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam_type is required"})
		return
	}

	// Validate all UUIDs
	if err := h.validateUUID(schoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
		return
	}
	if err := h.validateUUID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}
	if err := h.validateUUID(termID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}
	if err := h.validateUUID(classLevelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id"})
		return
	}
	if err := h.validateUUID(classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id"})
		return
	}
	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}

	// Validate exam type
	validExamTypes := map[string]bool{
		"weekly_test": true,
		"mid_term":    true,
		"main_exam":   true,
		"practice":    true,
	}
	if !validExamTypes[examType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid exam_type",
			"details": "exam_type must be one of: weekly_test, mid_term, main_exam, practice",
		})
		return
	}

	// Parse optional fields
	format := c.PostForm("format")
	if format == "" {
		format = dto.UploadFormatAuto
	}

	hasHeader := c.PostForm("has_header") == "true" || c.PostForm("has_header") == "1"
	sheetName := c.PostForm("sheet_name")
	textMode := c.PostForm("text_mode")
	if textMode == "" {
		textMode = dto.TextModeQA
	}
	preserveFormatting := c.PostForm("preserve_formatting") == "true" || c.PostForm("preserve_formatting") == "1"
	curriculumType := c.PostForm("curriculum_type")

	// Get file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":   "file too large",
			"details": "Maximum file size is 10MB",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Build request with FULL context
	req := &dto.BulkUploadRequest{
		SchoolID:           schoolID,
		SessionID:          sessionID,
		TermID:             termID,
		ClassLevelID:       classLevelID,
		ClassID:            classID,
		SubjectID:          subjectID,
		ExamType:           examType,
		File:               file,
		Format:             format,
		HasHeader:          hasHeader,
		SheetName:          sheetName,
		TextMode:           textMode,
		PreserveFormatting: preserveFormatting,
		CurriculumType:     curriculumType,
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
		return
	}
	defer src.Close()

	ctx := c.Request.Context()
	resp, err := h.questionService.BulkUploadFromFile(ctx, src, format, req, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    resp,
		"message": fmt.Sprintf("Uploaded with Session: %s, Term: %s, Exam Type: %s", sessionID, termID, examType),
	})
}

// ============================================
// TAG OPERATIONS
// ============================================

// CreateTag godoc
// @Summary      Create a new tag
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateTagRequest true "Tag name and description"
// @Success      201  {object}  map[string]interface{}  "data contains TagResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /tags [post]
func (h *QuestionHandler) CreateTag(c *gin.Context) {
	var req dto.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag name is required"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.CreateTag(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Tag created successfully",
		"data":    resp,
	})
}

// ListTags godoc
// @Summary      List all tags
// @Tags         Questions
// @Produce      json
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200  {object}  map[string]interface{}  "data (list of tags)"
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /tags [get]
func (h *QuestionHandler) ListTags(c *gin.Context) {
	page, err := h.parsePage(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := h.parseLimit(c.DefaultQuery("limit", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	tags, total, err := h.questionService.ListTags(ctx, page, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   tags,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// ============================================
// AI OPERATIONS
// ============================================

// GenerateQuestionsWithAI godoc
// @Summary      Generate questions using AI (async)
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.AIGenerateQuestionsRequest true "Generation parameters"
// @Success      200 {object} dto.AIQuestionGenerationResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/ai/generate [post]
func (h *QuestionHandler) GenerateQuestionsWithAI(c *gin.Context) {
	var req dto.AIGenerateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required UUIDs
	if err := h.validateUUID(req.SchoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.SessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.TermID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassLevelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.SubjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GenerateQuestionsWithAI(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ExtractQuestionsFromText godoc
// @Summary      Extract questions from raw text (async)
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.ExtractTextQuestionsRequest true "Text extraction parameters"
// @Success      200 {object} dto.AIQuestionGenerationResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/extract [post]
func (h *QuestionHandler) ExtractQuestionsFromText(c *gin.Context) {
	var req dto.ExtractTextQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required UUIDs
	if err := h.validateUUID(req.SchoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.SessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.TermID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassLevelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.SubjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id: " + err.Error()})
		return
	}

	if req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.ExtractQuestionsFromText(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetJobStatus godoc
// @Summary      Get status of an AI job
// @Tags         Questions
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} dto.AIJobStatusResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/jobs/{id} [get]
func (h *QuestionHandler) GetJobStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job ID is required"})
		return
	}

	if err := h.validateUUID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetJobStatus(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// CONTEXT-AWARE QUERIES
// ============================================

// GetCurrentAcademicContext godoc
// @Summary      Get current academic context
// @Description  Get current session and term for a school
// @Tags         Academic
// @Produce      json
// @Param        school_id query string true "School UUID"
// @Success      200 {object} dto.CurrentAcademicContextResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/current [get]
func (h *QuestionHandler) GetCurrentAcademicContext(c *gin.Context) {
	schoolID := c.Query("school_id")
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
		return
	}

	if err := h.validateUUID(schoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
		return
	}

	ctx := c.Request.Context()
	context, err := h.questionService.GetCurrentAcademicContext(ctx, schoolID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   context,
	})
}

// GetQuestionsByTerm godoc
// @Summary      Get questions by term
// @Description  Get all questions for a subject in a specific term
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        term_id query string true "Term UUID"
// @Success      200 {object} map[string]interface{} "data (list of questions), count"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-term [get]
func (h *QuestionHandler) GetQuestionsByTerm(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	termID := c.Query("term_id")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(termID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}

	ctx := c.Request.Context()
	questions, err := h.questionService.GetQuestionsByTerm(ctx, subjectID, termID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   questions,
		"count":  len(questions),
	})
}

// GetQuestionsBySession godoc
// @Summary      Get questions by session
// @Description  Get all questions for a subject in a specific session
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        session_id query string true "Session UUID"
// @Success      200 {object} map[string]interface{} "data (list of questions), count"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-session [get]
func (h *QuestionHandler) GetQuestionsBySession(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	ctx := c.Request.Context()
	questions, err := h.questionService.GetQuestionsBySession(ctx, subjectID, sessionID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   questions,
		"count":  len(questions),
	})
}

// GetQuestionsByClass godoc
// @Summary      Get questions by class
// @Description  Get all questions for a specific class
// @Tags         Questions
// @Produce      json
// @Param        class_id query string true "Class UUID"
// @Success      200 {object} map[string]interface{} "data (list of questions), count"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-class [get]
func (h *QuestionHandler) GetQuestionsByClass(c *gin.Context) {
	classID := c.Query("class_id")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	if err := h.validateUUID(classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id"})
		return
	}

	ctx := c.Request.Context()
	questions, err := h.questionService.GetQuestionsByClass(ctx, classID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   questions,
		"count":  len(questions),
	})
}

// GetQuestionContextSummary godoc
// @Summary      Get question context summary
// @Description  Get summary of questions grouped by term for a subject
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Success      200 {object} dto.QuestionContextSummaryResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/context/summary [get]
func (h *QuestionHandler) GetQuestionContextSummary(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}

	ctx := c.Request.Context()
	summary, err := h.questionService.GetQuestionContextSummary(ctx, subjectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}

// GetQuestionsWithContext godoc
// @Summary      Get questions with full context
// @Description  Get questions filtered with term/session context
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.FilterQuestionsWithContextRequest true "Filter criteria"
// @Success      200 {object} dto.QuestionListWithContextResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/with-context [post]
func (h *QuestionHandler) GetQuestionsWithContext(c *gin.Context) {
	var req dto.FilterQuestionsWithContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := h.validateUUID(req.SubjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(req.SchoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetQuestionsWithContext(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// GetQuestionsForExam godoc
// @Summary      Get questions for exam creation
// @Description  Get all questions for a subject in a specific term and session
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        term_id query string true "Term UUID"
// @Param        session_id query string true "Session UUID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/for-exam [get]
func (h *QuestionHandler) GetQuestionsForExam(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	termID := c.Query("term_id")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(termID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}
	if err := h.validateUUID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	ctx := c.Request.Context()
	questions, err := h.questionService.GetQuestionsForExam(ctx, subjectID, termID, sessionID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   questions,
		"count":  len(questions),
	})
}


// ============================================
// HELPER FUNCTIONS
// ============================================

func (h *QuestionHandler) handleError(c *gin.Context, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, ErrQuestionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
	case errors.Is(err, ErrInvalidQuestionID):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question ID"})
	case errors.Is(err, ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	case errors.Is(err, ErrTagAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "tag already exists"})
	case errors.Is(err, ErrValidationFailed):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal server error",
			"details": err.Error(),
		})
	}
}

func (h *QuestionHandler) validateUUID(id string) error {
	if id == "" {
		return errors.New("UUID cannot be empty")
	}
	_, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid UUID format: " + err.Error())
	}
	return nil
}

func (h *QuestionHandler) parsePage(pageStr string) (int, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1, nil
	}
	return page, nil
}

func (h *QuestionHandler) parseLimit(limitStr string) (int, error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return 20, nil
	}
	if limit > 100 {
		return 100, nil
	}
	return limit, nil
}

// ============================================================
// GROUPED QUESTION HANDLERS - ADD TO END OF question_handler.go
// ============================================================

// // GetQuestionsGrouped godoc
// // @Summary      Get questions grouped by exam type and question type
// // @Description  Returns questions organized by exam_type -> question_type with full context
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.FilterQuestionsGroupedRequest true "Filter criteria with context"
// // @Success      200 {object} dto.QuestionGroupsResponse
// // @Failure      400 {object} map[string]interface{}
// // @Failure      401 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/grouped [post]
// func (h *QuestionHandler) GetQuestionsGrouped(c *gin.Context) {
// 	var req dto.FilterQuestionsGroupedRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	if err := h.validateUUID(req.SchoolID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
// 		return
// 	}
// 	if err := h.validateUUID(req.SubjectID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
// 		return
// 	}

// 	if req.ExamType != "" {
// 		validExamTypes := map[string]bool{
// 			"weekly_test": true,
// 			"mid_term":    true,
// 			"main_exam":   true,
// 			"practice":    true,
// 		}
// 		if !validExamTypes[req.ExamType] {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error":   "invalid exam_type",
// 				"details": "exam_type must be one of: weekly_test, mid_term, main_exam, practice",
// 			})
// 			return
// 		}
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.GetQuestionsGrouped(ctx, &req)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// GetQuestionsGrouped godoc
// @Summary      Get questions grouped by exam type and question type
// @Description  Returns questions organized by exam_type -> question_type with full context. If no parameters provided, returns ALL questions.
// @Tags         Questions
// @Accept       json
// @Produce      json
// @Param        request body dto.FilterQuestionsGroupedRequest false "Filter criteria with context (optional)"
// @Success      200 {object} dto.QuestionGroupsResponse
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/grouped [post]
func (h *QuestionHandler) GetQuestionsGrouped(c *gin.Context) {
	var req dto.FilterQuestionsGroupedRequest
	
	// If request body is empty or invalid, use empty struct (returns all)
	if err := c.ShouldBindJSON(&req); err != nil {
		// If body is empty or invalid, just use empty req (no filters)
		req = dto.FilterQuestionsGroupedRequest{}
	}

	// If school_id and subject_id are provided, validate them
	if req.SchoolID != "" {
		if err := h.validateUUID(req.SchoolID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
			return
		}
	}
	if req.SubjectID != "" {
		if err := h.validateUUID(req.SubjectID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
			return
		}
	}

	if req.ExamType != "" {
		validExamTypes := map[string]bool{
			"weekly_test": true,
			"mid_term":    true,
			"main_exam":   true,
			"practice":    true,
		}
		if !validExamTypes[req.ExamType] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid exam_type",
				"details": "exam_type must be one of: weekly_test, mid_term, main_exam, practice",
			})
			return
		}
	}

	ctx := c.Request.Context()
	
	// If no filters provided, use GetAllQuestionsGrouped
	if req.SchoolID == "" && req.SubjectID == "" {
		resp, err := h.questionService.GetAllQuestionsGrouped(ctx)
		if err != nil {
			h.handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	resp, err := h.questionService.GetQuestionsGrouped(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// // GetAllQuestionsGrouped godoc
// // @Summary      Get ALL questions grouped by exam type and question type
// // @Description  Returns ALL questions organized by exam_type -> question_type with full context. No parameters required.
// // @Tags         Questions
// // @Produce      json
// // @Success      200 {object} dto.QuestionGroupsResponse
// // @Failure      401 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/all-grouped [get]
// func (h *QuestionHandler) GetAllQuestionsGrouped(c *gin.Context) {
// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.GetAllQuestionsGrouped(ctx)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// GetAllQuestionsGrouped godoc
// @Summary      Get ALL questions grouped by exam type and question type
// @Description  Returns ALL questions organized by exam_type -> question_type with full context. No parameters required.
// @Tags         Questions
// @Produce      json
// @Success      200 {object} dto.QuestionGroupsResponse
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/all-grouped [get]
func (h *QuestionHandler) GetAllQuestionsGrouped(c *gin.Context) {
	ctx := c.Request.Context()
	resp, err := h.questionService.GetAllQuestionsGrouped(ctx)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetQuestionsByExamType godoc
// @Summary      Get questions grouped by question type for a specific exam
// @Description  Returns all questions for an exam_type, grouped by question_type
// @Tags         Questions
// @Produce      json
// @Param        school_id query string true "School UUID"
// @Param        subject_id query string true "Subject UUID"
// @Param        exam_type query string true "Exam Type: weekly_test, mid_term, main_exam, practice"
// @Param        session_id query string false "Session UUID (optional)"
// @Param        term_id query string false "Term UUID (optional)"
// @Param        class_id query string false "Class UUID (optional)"
// @Success      200 {object} dto.QuestionGroupsResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-exam-type [get]
func (h *QuestionHandler) GetQuestionsByExamType(c *gin.Context) {
	schoolID := c.Query("school_id")
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
		return
	}

	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	examType := c.Query("exam_type")
	if examType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam_type is required"})
		return
	}

	validExamTypes := map[string]bool{
		"weekly_test": true,
		"mid_term":    true,
		"main_exam":   true,
		"practice":    true,
	}
	if !validExamTypes[examType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid exam_type",
			"details": "exam_type must be one of: weekly_test, mid_term, main_exam, practice",
		})
		return
	}

	if err := h.validateUUID(schoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id"})
		return
	}
	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}

	sessionID := c.Query("session_id")
	termID := c.Query("term_id")
	classID := c.Query("class_id")

	req := dto.FilterQuestionsGroupedRequest{
		SchoolID:  schoolID,
		SubjectID: subjectID,
		ExamType:  examType,
		SessionID: sessionID,
		TermID:    termID,
		ClassID:   classID,
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetQuestionsGrouped(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetQuestionsByTermGrouped godoc
// @Summary      Get questions by term grouped by exam type
// @Description  Returns grouped questions for a specific term
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        term_id query string true "Term UUID"
// @Success      200 {object} dto.QuestionGroupsResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-term-grouped [get]
func (h *QuestionHandler) GetQuestionsByTermGrouped(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	termID := c.Query("term_id")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(termID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetQuestionsByTermWithGrouping(ctx, subjectID, termID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetQuestionsBySessionGrouped godoc
// @Summary      Get questions by session grouped by exam type
// @Description  Returns grouped questions for a specific session
// @Tags         Questions
// @Produce      json
// @Param        subject_id query string true "Subject UUID"
// @Param        session_id query string true "Session UUID"
// @Success      200 {object} dto.QuestionGroupsResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/by-session-grouped [get]
func (h *QuestionHandler) GetQuestionsBySessionGrouped(c *gin.Context) {
	subjectID := c.Query("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}
	if err := h.validateUUID(sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.GetQuestionsBySessionWithGrouping(ctx, subjectID, sessionID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}