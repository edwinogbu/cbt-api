package handler

import (
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/service"
	"errors"
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

type QuestionHandler struct {
	questionService *service.QuestionService
}

func NewQuestionHandler(s *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: s}
}

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
// @Router       /questions/create [post]
func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	var req dto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context (set by AuthMiddleware)
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
// @Router       /questions/update/{id} [put]
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

	// Get user ID from context
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
// @Router       /questions/delete/{id} [delete]
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

	// Get user ID from context
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
// @Router       /questions/list [get]
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

	// Set default pagination
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
// @Router       /questions/bulk-delete [post]
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

	// Validate all IDs
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

	// Validate subject_id if provided
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
// @Router       /questions/tags/create [post]
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
// @Router       /questions/tags/list [get]
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

// BulkCreateQuestions handles JSON array of questions
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

	// Get user ID from context
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

// BulkUploadFile handles file upload (CSV, JSON, Excel)
// @Summary      Bulk upload questions from file
// @Tags         Questions
// @Accept       multipart/form-data
// @Produce      json
// @Param        subject_id formData string true "Subject UUID"
// @Param        file formData file true "File (CSV, JSON, Excel)"
// @Param        format formData string true "File format: csv, json, excel"
// @Param        has_header formData bool false "CSV has header (default true)"
// @Success      200 {object} dto.BulkUploadResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      413 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /questions/bulk-upload [post]
func (h *QuestionHandler) BulkUploadFile(c *gin.Context) {
	// Parse form data
	subjectID := c.PostForm("subject_id")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
		return
	}

	if err := h.validateUUID(subjectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	format := c.PostForm("format")
	if format == "" {
		format = "csv"
	}

	// Validate format
	validFormats := map[string]bool{"csv": true, "json": true, "excel": true}
	if !validFormats[format] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid format",
			"details": "format must be csv, json, or excel",
		})
		return
	}

	hasHeader := c.PostForm("has_header") == "true" || c.PostForm("has_header") == "1"

	// Get file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Validate file size (max 10MB)
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":   "file too large",
			"details": "Maximum file size is 10MB",
		})
		return
	}

	// Validate file extension
	if !h.validateFileExtension(file.Filename, format) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid file type",
			"details": "File extension does not match format",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
		return
	}
	defer src.Close()

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.questionService.BulkUploadFromFile(ctx, src, format, subjectID, hasHeader, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

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

	// Validate UUIDs
	if err := h.validateUUID(req.SchoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassLevelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
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

	// Validate UUIDs
	if err := h.validateUUID(req.SchoolID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
		return
	}
	if err := h.validateUUID(req.ClassLevelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
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
		// Log the error here with your logger
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

func (h *QuestionHandler) validateFileExtension(filename, format string) bool {
	extensions := map[string][]string{
		"csv":   {".csv"},
		"json":  {".json"},
		"excel": {".xlsx", ".xls"},
	}

	allowedExts, ok := extensions[format]
	if !ok {
		return false
	}

	for _, ext := range allowedExts {
		if len(filename) >= len(ext) && filename[len(filename)-len(ext):] == ext {
			return true
		}
	}
	return false
}

// package handler

// import (
// 	"cbt-api/internal/cbt/dto"
// 	"cbt-api/internal/cbt/service"
// 	"errors"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// )

// type QuestionHandler struct {
// 	questionService *service.QuestionService
// }

// func NewQuestionHandler(s *service.QuestionService) *QuestionHandler {
// 	return &QuestionHandler{questionService: s}
// }

// // CreateQuestion godoc
// // @Summary      Create a new question
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateQuestionRequest true "Question details"
// // @Success      201  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/create [post]
// func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
// 	var req dto.CreateQuestionRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	// Get user ID from context (set by AuthMiddleware)
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.CreateQuestion(ctx, &req, userID.(string))
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"status":  "success",
// 		"message": "Question created successfully",
// 		"data":    resp,
// 	})
// }

// // GetQuestion godoc
// // @Summary      Get a single question
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Success      200  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/{id} [get]
// func (h *QuestionHandler) GetQuestion(c *gin.Context) {
// 	id := c.Param("id")
// 	if id == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
// 		return
// 	}

// 	if err := h.validateUUID(id); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.GetQuestion(ctx, id)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   resp,
// 	})
// }

// // UpdateQuestion godoc
// // @Summary      Update a question
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Param        request body dto.UpdateQuestionRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "data contains updated QuestionBankResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      403  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/update/{id} [put]
// func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
// 	id := c.Param("id")
// 	if id == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
// 		return
// 	}

// 	if err := h.validateUUID(id); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	var req dto.UpdateQuestionRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	// Get user ID from context
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.UpdateQuestion(ctx, id, &req, userID.(string))
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  "success",
// 		"message": "Question updated successfully",
// 		"data":    resp,
// 	})
// }

// // DeleteQuestion godoc
// // @Summary      Delete a question
// // @Description  Soft‑delete a question (it will not be returned in lists)
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      403  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/delete/{id} [delete]
// func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
// 	id := c.Param("id")
// 	if id == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "question ID is required"})
// 		return
// 	}

// 	if err := h.validateUUID(id); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Get user ID from context
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	if err := h.questionService.DeleteQuestion(ctx, id, userID.(string)); err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  "success",
// 		"message": "Question deleted successfully",
// 	})
// }

// // ListQuestions godoc
// // @Summary      List questions by subject
// // @Tags         Questions
// // @Produce      json
// // @Param        subject_id query string true "Subject UUID"
// // @Param        page query int false "Page number (default 1)"
// // @Param        limit query int false "Items per page (default 20, max 100)"
// // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/list [get]
// func (h *QuestionHandler) ListQuestions(c *gin.Context) {
// 	subjectID := c.Query("subject_id")
// 	if subjectID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
// 		return
// 	}

// 	if err := h.validateUUID(subjectID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	page, err := h.parsePage(c.DefaultQuery("page", "1"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	limit, err := h.parseLimit(c.DefaultQuery("limit", "20"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, total, err := h.questionService.ListQuestions(ctx, subjectID, page, limit)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   resp,
// 		"total":  total,
// 		"page":   page,
// 		"limit":  limit,
// 	})
// }

// // FilterQuestions godoc
// // @Summary      Advanced filter for questions
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.FilterQuestionsRequest true "Filter criteria"
// // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/filter [post]
// func (h *QuestionHandler) FilterQuestions(c *gin.Context) {
// 	var req dto.FilterQuestionsRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	// Set default pagination
// 	if req.Page < 1 {
// 		req.Page = 1
// 	}
// 	if req.Limit < 1 || req.Limit > 100 {
// 		req.Limit = 20
// 	}

// 	ctx := c.Request.Context()
// 	resp, total, err := h.questionService.FilterQuestions(ctx, &req)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   resp,
// 		"total":  total,
// 		"page":   req.Page,
// 		"limit":  req.Limit,
// 	})
// }

// // BulkDelete godoc
// // @Summary      Delete multiple questions
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkDeleteRequest true "List of question IDs"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk-delete [post]
// func (h *QuestionHandler) BulkDelete(c *gin.Context) {
// 	var req dto.BulkDeleteRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	if len(req.QuestionIDs) == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "question_ids cannot be empty"})
// 		return
// 	}

// 	// Validate all IDs
// 	for _, id := range req.QuestionIDs {
// 		if err := h.validateUUID(id); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error":   "invalid question ID",
// 				"details": err.Error(),
// 				"id":      id,
// 			})
// 			return
// 		}
// 	}

// 	ctx := c.Request.Context()
// 	if err := h.questionService.BulkDelete(ctx, &req); err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  "success",
// 		"message": "Bulk delete successful",
// 	})
// }

// // GetStatistics godoc
// // @Summary      Get question statistics
// // @Tags         Questions
// // @Produce      json
// // @Param        subject_id query string false "Subject UUID (optional)"
// // @Success      200  {object}  map[string]interface{}  "data (statistics)"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/statistics [get]
// func (h *QuestionHandler) GetStatistics(c *gin.Context) {
// 	subjectID := c.Query("subject_id")

// 	// Validate subject_id if provided
// 	if subjectID != "" {
// 		if err := h.validateUUID(subjectID); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}
// 	}

// 	ctx := c.Request.Context()
// 	stats, err := h.questionService.GetStatistics(ctx, subjectID)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   stats,
// 	})
// }

// // CreateTag godoc
// // @Summary      Create a new tag
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateTagRequest true "Tag name and description"
// // @Success      201  {object}  map[string]interface{}  "data contains TagResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      409  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/tags/create [post]
// func (h *QuestionHandler) CreateTag(c *gin.Context) {
// 	var req dto.CreateTagRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	if req.Name == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "tag name is required"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.CreateTag(ctx, &req)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"status":  "success",
// 		"message": "Tag created successfully",
// 		"data":    resp,
// 	})
// }

// // ListTags godoc
// // @Summary      List all tags
// // @Tags         Questions
// // @Produce      json
// // @Param        page query int false "Page number (default 1)"
// // @Param        limit query int false "Items per page (default 20, max 100)"
// // @Success      200  {object}  map[string]interface{}  "data (list of tags)"
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/tags/list [get]
// func (h *QuestionHandler) ListTags(c *gin.Context) {
// 	page, err := h.parsePage(c.DefaultQuery("page", "1"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	limit, err := h.parseLimit(c.DefaultQuery("limit", "20"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	tags, total, err := h.questionService.ListTags(ctx, page, limit)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   tags,
// 		"total":  total,
// 		"page":   page,
// 		"limit":  limit,
// 	})
// }

// // BulkCreateQuestions handles JSON array of questions
// // @Summary      Bulk create questions from JSON array
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkCreateQuestionRequest true "Array of questions"
// // @Success      201 {object} map[string]interface{} "data, count"
// // @Failure      400 {object} map[string]interface{}
// // @Failure      401 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk [post]
// func (h *QuestionHandler) BulkCreateQuestions(c *gin.Context) {
// 	var req dto.BulkCreateQuestionRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	if len(req.Questions) == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "no questions provided"})
// 		return
// 	}

// 	if len(req.Questions) > 1000 {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "too many questions",
// 			"details": "Maximum 1000 questions per bulk request",
// 		})
// 		return
// 	}

// 	// Get user ID from context
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.BulkCreateQuestionsFromJSON(ctx, &req, userID.(string))
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"status":  "success",
// 		"message": "Bulk questions created successfully",
// 		"data":    resp,
// 		"count":   len(resp),
// 	})
// }

// // BulkUploadFile handles file upload (CSV, JSON, Excel)
// // @Summary      Bulk upload questions from file
// // @Tags         Questions
// // @Accept       multipart/form-data
// // @Produce      json
// // @Param        subject_id formData string true "Subject UUID"
// // @Param        file formData file true "File (CSV, JSON, Excel)"
// // @Param        format formData string true "File format: csv, json, excel"
// // @Param        has_header formData bool false "CSV has header (default true)"
// // @Success      200 {object} dto.BulkUploadResponse
// // @Failure      400 {object} map[string]interface{}
// // @Failure      401 {object} map[string]interface{}
// // @Failure      413 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk-upload [post]
// func (h *QuestionHandler) BulkUploadFile(c *gin.Context) {
// 	// Parse form data
// 	subjectID := c.PostForm("subject_id")
// 	if subjectID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
// 		return
// 	}

// 	if err := h.validateUUID(subjectID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	format := c.PostForm("format")
// 	if format == "" {
// 		format = "csv"
// 	}

// 	// Validate format
// 	validFormats := map[string]bool{"csv": true, "json": true, "excel": true}
// 	if !validFormats[format] {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid format",
// 			"details": "format must be csv, json, or excel",
// 		})
// 		return
// 	}

// 	hasHeader := c.PostForm("has_header") == "true" || c.PostForm("has_header") == "1"

// 	// Get file
// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
// 		return
// 	}

// 	// Validate file size (max 10MB)
// 	const maxFileSize = 10 * 1024 * 1024 // 10MB
// 	if file.Size > maxFileSize {
// 		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
// 			"error":   "file too large",
// 			"details": "Maximum file size is 10MB",
// 		})
// 		return
// 	}

// 	// Validate file extension
// 	if !h.validateFileExtension(file.Filename, format) {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid file type",
// 			"details": "File extension does not match format",
// 		})
// 		return
// 	}

// 	src, err := file.Open()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
// 		return
// 	}
// 	defer src.Close()

// 	// Get user ID from context
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.BulkUploadFromFile(ctx, src, format, subjectID, hasHeader, userID.(string))
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   resp,
// 	})
// }

// // GenerateQuestionsWithAI godoc
// // @Summary      Generate questions using AI (async)
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.AIGenerateQuestionsRequest true "Generation parameters"
// // @Success      200 {object} dto.AIQuestionGenerationResponse
// // @Failure      400 {object} map[string]interface{}
// // @Failure      401 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/ai/generate [post]
// func (h *QuestionHandler) GenerateQuestionsWithAI(c *gin.Context) {
// 	var req dto.AIGenerateQuestionsRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	// Validate UUIDs
// 	if err := h.validateUUID(req.SchoolID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
// 		return
// 	}
// 	if err := h.validateUUID(req.ClassLevelID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
// 		return
// 	}
// 	if err := h.validateUUID(req.SubjectID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id: " + err.Error()})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.GenerateQuestionsWithAI(ctx, &req)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// // ExtractQuestionsFromText godoc
// // @Summary      Extract questions from raw text (async)
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.ExtractTextQuestionsRequest true "Text extraction parameters"
// // @Success      200 {object} dto.AIQuestionGenerationResponse
// // @Failure      400 {object} map[string]interface{}
// // @Failure      401 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/extract [post]
// func (h *QuestionHandler) ExtractQuestionsFromText(c *gin.Context) {
// 	var req dto.ExtractTextQuestionsRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "invalid request body",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	// Validate UUIDs
// 	if err := h.validateUUID(req.SchoolID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school_id: " + err.Error()})
// 		return
// 	}
// 	if err := h.validateUUID(req.ClassLevelID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_level_id: " + err.Error()})
// 		return
// 	}
// 	if err := h.validateUUID(req.SubjectID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id: " + err.Error()})
// 		return
// 	}

// 	if req.Text == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.ExtractQuestionsFromText(ctx, &req)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// // GetJobStatus godoc
// // @Summary      Get status of an AI job
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Job ID"
// // @Success      200 {object} dto.AIJobStatusResponse
// // @Failure      400 {object} map[string]interface{}
// // @Failure      404 {object} map[string]interface{}
// // @Failure      500 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/jobs/{id} [get]
// func (h *QuestionHandler) GetJobStatus(c *gin.Context) {
// 	id := c.Param("id")
// 	if id == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "job ID is required"})
// 		return
// 	}

// 	if err := h.validateUUID(id); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx := c.Request.Context()
// 	resp, err := h.questionService.GetJobStatus(ctx, id)
// 	if err != nil {
// 		h.handleError(c, err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// // ============================================
// // HELPER FUNCTIONS
// // ============================================

// func (h *QuestionHandler) handleError(c *gin.Context, err error) {
// 	switch {
// 	case err == nil:
// 		return
// 	case errors.Is(err, service.ErrQuestionNotFound):
// 		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
// 	case errors.Is(err, service.ErrInvalidQuestionID):
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question ID"})
// 	case errors.Is(err, service.ErrPermissionDenied):
// 		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
// 	case errors.Is(err, service.ErrTagAlreadyExists):
// 		c.JSON(http.StatusConflict, gin.H{"error": "tag already exists"})
// 	case errors.Is(err, service.ErrValidationFailed):
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 	default:
// 		// Log the error here with your logger
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error":   "internal server error",
// 			"details": err.Error(),
// 		})
// 	}
// }

// func (h *QuestionHandler) validateUUID(id string) error {
// 	if id == "" {
// 		return errors.New("UUID cannot be empty")
// 	}
// 	_, err := uuid.Parse(id)
// 	if err != nil {
// 		return errors.New("invalid UUID format: " + err.Error())
// 	}
// 	return nil
// }

// func (h *QuestionHandler) parsePage(pageStr string) (int, error) {
// 	page, err := strconv.Atoi(pageStr)
// 	if err != nil || page < 1 {
// 		return 1, nil
// 	}
// 	return page, nil
// }

// func (h *QuestionHandler) parseLimit(limitStr string) (int, error) {
// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil || limit < 1 {
// 		return 20, nil
// 	}
// 	if limit > 100 {
// 		return 100, nil
// 	}
// 	return limit, nil
// }

// func (h *QuestionHandler) validateFileExtension(filename, format string) bool {
// 	extensions := map[string][]string{
// 		"csv":   {".csv"},
// 		"json":  {".json"},
// 		"excel": {".xlsx", ".xls"},
// 	}

// 	allowedExts, ok := extensions[format]
// 	if !ok {
// 		return false
// 	}

// 	for _, ext := range allowedExts {
// 		if len(filename) >= len(ext) && filename[len(filename)-len(ext):] == ext {
// 			return true
// 		}
// 	}
// 	return false
// }

// // ============================================
// // EXPORTED ERROR VARIABLES FOR SERVICE
// // ============================================

// // These are exported so service can use them for error handling
// var (
// 	ErrQuestionNotFound  = errors.New("question not found")
// 	ErrInvalidQuestionID = errors.New("invalid question ID")
// 	ErrPermissionDenied  = errors.New("permission denied")
// 	ErrTagAlreadyExists  = errors.New("tag already exists")
// 	ErrValidationFailed  = errors.New("validation failed")
// )




// package handler

// import (
//     "cbt-api/internal/cbt/dto"
//     "cbt-api/internal/cbt/service"
//     "net/http"
//     "strconv"

//     "github.com/gin-gonic/gin"
// )

// type QuestionHandler struct {
//     questionService *service.QuestionService
// }

// func NewQuestionHandler(s *service.QuestionService) *QuestionHandler {
//     return &QuestionHandler{questionService: s}
// }

// // CreateQuestion godoc
// // @Summary      Create a new question
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateQuestionRequest true "Question details"
// // @Success      201  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/create [post]
// func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
//     var req dto.CreateQuestionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     // Get user ID from context (set by AuthMiddleware)
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
//         return
//     }
    
//     resp, err := h.questionService.CreateQuestion(&req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusCreated, gin.H{"data": resp})
// }

// // GetQuestion godoc
// // @Summary      Get a single question
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Success      200  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // @Failure      404  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/{id} [get]
// func (h *QuestionHandler) GetQuestion(c *gin.Context) {
//     id := c.Param("id")
//     resp, err := h.questionService.GetQuestion(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }

// // UpdateQuestion godoc
// // @Summary      Update a question
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Param        request body dto.UpdateQuestionRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "data contains updated QuestionBankResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/update/{id} [put]
// func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
//     id := c.Param("id")
//     var req dto.UpdateQuestionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     // Get user ID from context
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
//         return
//     }
    
//     resp, err := h.questionService.UpdateQuestion(id, &req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }

// // DeleteQuestion godoc
// // @Summary      Delete a question
// // @Description  Soft‑delete a question (it will not be returned in lists)
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Question UUID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/delete/{id} [delete]
// func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
//     id := c.Param("id")
//     if err := h.questionService.DeleteQuestion(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"message": "question deleted"})
// }

// // ListQuestions godoc
// // @Summary      List questions by subject
// // @Tags         Questions
// // @Produce      json
// // @Param        subject_id query string true "Subject UUID"
// // @Param        page query int false "Page number (default 1)"
// // @Param        limit query int false "Items per page (default 20, max 100)"
// // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/list [get]
// func (h *QuestionHandler) ListQuestions(c *gin.Context) {
//     subjectID := c.Query("subject_id")
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
//     resp, total, err := h.questionService.ListQuestions(subjectID, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{
//         "data":  resp,
//         "total": total,
//         "page":  page,
//         "limit": limit,
//     })
// }

// // FilterQuestions godoc
// // @Summary      Advanced filter for questions
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.FilterQuestionsRequest true "Filter criteria"
// // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/filter [post]
// func (h *QuestionHandler) FilterQuestions(c *gin.Context) {
//     var req dto.FilterQuestionsRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, total, err := h.questionService.FilterQuestions(&req)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{
//         "data":  resp,
//         "total": total,
//         "page":  req.Page,
//         "limit": req.Limit,
//     })
// }

// // BulkDelete godoc
// // @Summary      Delete multiple questions
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkDeleteRequest true "List of question IDs"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk-delete [post]
// func (h *QuestionHandler) BulkDelete(c *gin.Context) {
//     var req dto.BulkDeleteRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     if err := h.questionService.BulkDelete(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"message": "bulk delete successful"})
// }

// // GetStatistics godoc
// // @Summary      Get question statistics
// // @Tags         Questions
// // @Produce      json
// // @Param        subject_id query string false "Subject UUID (optional)"
// // @Success      200  {object}  map[string]interface{}  "data (statistics)"
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/statistics [get]
// func (h *QuestionHandler) GetStatistics(c *gin.Context) {
//     subjectID := c.Query("subject_id")
//     stats, err := h.questionService.GetStatistics(subjectID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": stats})
// }

// // CreateTag godoc
// // @Summary      Create a new tag
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateTagRequest true "Tag name and description"
// // @Success      201  {object}  map[string]interface{}  "data contains TagResponse"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/tags/create [post]
// func (h *QuestionHandler) CreateTag(c *gin.Context) {
//     var req dto.CreateTagRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, err := h.questionService.CreateTag(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusCreated, gin.H{"data": resp})
// }

// // ListTags godoc
// // @Summary      List all tags
// // @Tags         Questions
// // @Produce      json
// // @Success      200  {object}  map[string]interface{}  "data (list of tags)"
// // @Failure      401  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/tags/list [get]
// func (h *QuestionHandler) ListTags(c *gin.Context) {
//     tags, err := h.questionService.ListTags()
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": tags})
// }

// // BulkCreateQuestions handles JSON array of questions
// // @Summary      Bulk create questions from JSON array
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkCreateQuestionRequest true "Array of questions"
// // @Success      201 {object} map[string]interface{}
// // @Failure      400 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk [post]
// func (h *QuestionHandler) BulkCreateQuestions(c *gin.Context) {
//     var req dto.BulkCreateQuestionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     // Get user ID from context
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
//         return
//     }
    
//     resp, err := h.questionService.BulkCreateQuestionsFromJSON(&req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusCreated, gin.H{"data": resp, "count": len(resp)})
// }

// // BulkUploadFile handles file upload (CSV, JSON, Excel)
// // @Summary      Bulk upload questions from file
// // @Tags         Questions
// // @Accept       multipart/form-data
// // @Produce      json
// // @Param        subject_id formData string true "Subject UUID"
// // @Param        file formData file true "File (CSV, JSON, Excel)"
// // @Param        format formData string true "File format: csv, json, excel"
// // @Param        has_header formData bool false "CSV has header (default true)"
// // @Success      200 {object} dto.BulkUploadResponse
// // @Failure      400 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/bulk-upload [post]
// func (h *QuestionHandler) BulkUploadFile(c *gin.Context) {
//     subjectID := c.PostForm("subject_id")
//     if subjectID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
//         return
//     }
//     format := c.PostForm("format")
//     if format == "" {
//         format = "csv"
//     }
//     hasHeader := c.PostForm("has_header") == "true" || c.PostForm("has_header") == "1"
    
//     file, err := c.FormFile("file")
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
//         return
//     }
//     src, err := file.Open()
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
//         return
//     }
//     defer src.Close()
    
//     // Get user ID from context
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
//         return
//     }
    
//     resp, err := h.questionService.BulkUploadFromFile(src, format, subjectID, hasHeader, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, resp)
// }


// // GenerateQuestionsWithAI godoc
// // @Summary      Generate questions using AI (async)
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.AIGenerateQuestionsRequest true "Generation parameters"
// // @Success      200 {object} dto.AIQuestionGenerationResponse
// // @Failure      400 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/ai/generate [post]
// func (h *QuestionHandler) GenerateQuestionsWithAI(c *gin.Context) {
//     var req dto.AIGenerateQuestionsRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, err := h.questionService.GenerateQuestionsWithAI(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, resp)
// }

// // ExtractQuestionsFromText godoc
// // @Summary      Extract questions from raw text (async)
// // @Tags         Questions
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.ExtractTextQuestionsRequest true "Text extraction parameters"
// // @Success      200 {object} dto.AIQuestionGenerationResponse
// // @Failure      400 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/extract [post]
// func (h *QuestionHandler) ExtractQuestionsFromText(c *gin.Context) {
//     var req dto.ExtractTextQuestionsRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, err := h.questionService.ExtractQuestionsFromText(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, resp)
// }

// // GetJobStatus godoc
// // @Summary      Get status of an AI job
// // @Tags         Questions
// // @Produce      json
// // @Param        id path string true "Job ID"
// // @Success      200 {object} dto.AIJobStatusResponse
// // @Failure      404 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /questions/jobs/{id} [get]
// func (h *QuestionHandler) GetJobStatus(c *gin.Context) {
//     id := c.Param("id")
//     resp, err := h.questionService.GetJobStatus(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, resp)
// }


// // package handler

// // import (
// //     "cbt-api/internal/cbt/dto"
// //     "cbt-api/internal/cbt/service"
// //     "net/http"
// //     "strconv"

// //     "github.com/gin-gonic/gin"
// //     // "github.com/xuri/excelize/v2"
// // )

// // type QuestionHandler struct {
// //     questionService *service.QuestionService
// // }

// // func NewQuestionHandler(s *service.QuestionService) *QuestionHandler {
// //     return &QuestionHandler{questionService: s}
// // }

// // // // CreateQuestion godoc
// // // // @Summary      Create a new question
// // // // @Tags         Questions
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.CreateQuestionRequest true "Question details"
// // // // @Success      201  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // // // @Failure      400  {object}  map[string]interface{}
// // // // @Failure      401  {object}  map[string]interface{}
// // // // @Security     BearerAuth
// // // // @Router       /questions/create [post]
// // // func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
// // //     var req dto.CreateQuestionRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }
// // //     resp, err := h.questionService.CreateQuestion(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }
// // //     c.JSON(http.StatusCreated, gin.H{"data": resp})
// // // }

// // // CreateQuestion godoc
// // // @Summary      Create a new question
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.CreateQuestionRequest true "Question details"
// // // @Success      201  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/create [post]
// // func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
// //     var req dto.CreateQuestionRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     // Get user ID from context (set by auth middleware)
// //     userID, exists := c.Get("user_id")
// //     if !exists {
// //         c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// //         return
// //     }
    
// //     // Convert to string
// //     userIDStr, ok := userID.(string)
// //     if !ok {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
// //         return
// //     }
    
// //     // Pass user ID to service
// //     resp, err := h.questionService.CreateQuestion(&req, userIDStr)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusCreated, gin.H{"data": resp})
// // }

// // // GetQuestion godoc
// // // @Summary      Get a single question
// // // @Tags         Questions
// // // @Produce      json
// // // @Param        id path string true "Question UUID"
// // // @Success      200  {object}  map[string]interface{}  "data contains QuestionBankResponse"
// // // @Failure      404  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/{id} [get]
// // func (h *QuestionHandler) GetQuestion(c *gin.Context) {
// //     id := c.Param("id")
// //     resp, err := h.questionService.GetQuestion(id)
// //     if err != nil {
// //         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"data": resp})
// // }

// // // UpdateQuestion godoc
// // // @Summary      Update a question
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        id path string true "Question UUID"
// // // @Param        request body dto.UpdateQuestionRequest true "Fields to update"
// // // @Success      200  {object}  map[string]interface{}  "data contains updated QuestionBankResponse"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/update/{id} [put]
// // func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
// //     id := c.Param("id")
// //     var req dto.UpdateQuestionRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     resp, err := h.questionService.UpdateQuestion(id, &req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"data": resp})
// // }

// // // DeleteQuestion godoc
// // // @Summary      Delete a question
// // // @Description  Soft‑delete a question (it will not be returned in lists)
// // // @Tags         Questions
// // // @Produce      json
// // // @Param        id path string true "Question UUID"
// // // @Success      200  {object}  map[string]interface{}  "message"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/delete/{id} [delete]
// // func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
// //     id := c.Param("id")
// //     if err := h.questionService.DeleteQuestion(id); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"message": "question deleted"})
// // }

// // // ListQuestions godoc
// // // @Summary      List questions by subject
// // // @Tags         Questions
// // // @Produce      json
// // // @Param        subject_id query string true "Subject UUID"
// // // @Param        page query int false "Page number (default 1)"
// // // @Param        limit query int false "Items per page (default 20, max 100)"
// // // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/list [get]
// // func (h *QuestionHandler) ListQuestions(c *gin.Context) {
// //     subjectID := c.Query("subject_id")
// //     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
// //     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
// //     resp, total, err := h.questionService.ListQuestions(subjectID, page, limit)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{
// //         "data":  resp,
// //         "total": total,
// //         "page":  page,
// //         "limit": limit,
// //     })
// // }

// // // FilterQuestions godoc
// // // @Summary      Advanced filter for questions
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.FilterQuestionsRequest true "Filter criteria"
// // // @Success      200  {object}  map[string]interface{}  "data, total, page, limit"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/filter [post]
// // func (h *QuestionHandler) FilterQuestions(c *gin.Context) {
// //     var req dto.FilterQuestionsRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     resp, total, err := h.questionService.FilterQuestions(&req)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{
// //         "data":  resp,
// //         "total": total,
// //         "page":  req.Page,
// //         "limit": req.Limit,
// //     })
// // }

// // // BulkDelete godoc
// // // @Summary      Delete multiple questions
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.BulkDeleteRequest true "List of question IDs"
// // // @Success      200  {object}  map[string]interface{}  "message"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/bulk-delete [post]
// // func (h *QuestionHandler) BulkDelete(c *gin.Context) {
// //     var req dto.BulkDeleteRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     if err := h.questionService.BulkDelete(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"message": "bulk delete successful"})
// // }

// // // GetStatistics godoc
// // // @Summary      Get question statistics
// // // @Tags         Questions
// // // @Produce      json
// // // @Param        subject_id query string false "Subject UUID (optional)"
// // // @Success      200  {object}  map[string]interface{}  "data (statistics)"
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/statistics [get]
// // func (h *QuestionHandler) GetStatistics(c *gin.Context) {
// //     subjectID := c.Query("subject_id")
// //     stats, err := h.questionService.GetStatistics(subjectID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"data": stats})
// // }

// // // CreateTag godoc
// // // @Summary      Create a new tag
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.CreateTagRequest true "Tag name and description"
// // // @Success      201  {object}  map[string]interface{}  "data contains TagResponse"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/tags/create [post]
// // func (h *QuestionHandler) CreateTag(c *gin.Context) {
// //     var req dto.CreateTagRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     resp, err := h.questionService.CreateTag(&req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusCreated, gin.H{"data": resp})
// // }

// // // ListTags godoc
// // // @Summary      List all tags
// // // @Tags         Questions
// // // @Produce      json
// // // @Success      200  {object}  map[string]interface{}  "data (list of tags)"
// // // @Failure      401  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/tags/list [get]
// // func (h *QuestionHandler) ListTags(c *gin.Context) {
// //     tags, err := h.questionService.ListTags()
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, gin.H{"data": tags})
// // }

// // // BulkCreateQuestions handles JSON array of questions
// // // @Summary      Bulk create questions from JSON array
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.BulkCreateQuestionRequest true "Array of questions"
// // // @Success      201 {object} map[string]interface{}
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/bulk [post]
// // func (h *QuestionHandler) BulkCreateQuestions(c *gin.Context) {
// //     var req dto.BulkCreateQuestionRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
// //     resp, err := h.questionService.BulkCreateQuestionsFromJSON(&req)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusCreated, gin.H{"data": resp, "count": len(resp)})
// // }

// // // BulkUploadFile handles file upload (CSV, JSON, Excel)
// // // @Summary      Bulk upload questions from file
// // // @Tags         Questions
// // // @Accept       multipart/form-data
// // // @Produce      json
// // // @Param        subject_id formData string true "Subject UUID"
// // // @Param        file formData file true "File (CSV, JSON, Excel)"
// // // @Param        format formData string true "File format: csv, json, excel"
// // // @Param        has_header formData bool false "CSV has header (default true)"
// // // @Success      200 {object} dto.BulkUploadResponse
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/bulk-upload [post]
// // func (h *QuestionHandler) BulkUploadFile(c *gin.Context) {
// //     subjectID := c.PostForm("subject_id")
// //     if subjectID == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": "subject_id is required"})
// //         return
// //     }
// //     format := c.PostForm("format")
// //     if format == "" {
// //         format = "csv"
// //     }
// //     hasHeader := c.PostForm("has_header") == "true" || c.PostForm("has_header") == "1"
    
// //     file, err := c.FormFile("file")
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
// //         return
// //     }
// //     src, err := file.Open()
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
// //         return
// //     }
// //     defer src.Close()
    
// //     resp, err := h.questionService.BulkUploadFromFile(src, format, subjectID, hasHeader)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
// //     c.JSON(http.StatusOK, resp)
// // }


// // // GenerateQuestionsWithAI godoc
// // // @Summary      Generate questions using AI (async)
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.AIGenerateQuestionsRequest true "Generation parameters"
// // // @Success      200 {object} dto.AIQuestionGenerationResponse
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/ai/generate [post]
// // func (h *QuestionHandler) GenerateQuestionsWithAI(c *gin.Context) {
// // 	var req dto.AIGenerateQuestionsRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	resp, err := h.questionService.GenerateQuestionsWithAI(&req)
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, resp)
// // }

// // // ExtractQuestionsFromText godoc
// // // @Summary      Extract questions from raw text (async)
// // // @Tags         Questions
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.ExtractTextQuestionsRequest true "Text extraction parameters"
// // // @Success      200 {object} dto.AIQuestionGenerationResponse
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/extract [post]
// // func (h *QuestionHandler) ExtractQuestionsFromText(c *gin.Context) {
// // 	var req dto.ExtractTextQuestionsRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	resp, err := h.questionService.ExtractQuestionsFromText(&req)
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, resp)
// // }

// // // GetJobStatus godoc
// // // @Summary      Get status of an AI job
// // // @Tags         Questions
// // // @Produce      json
// // // @Param        id path string true "Job ID"
// // // @Success      200 {object} dto.AIJobStatusResponse
// // // @Failure      404 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /questions/jobs/{id} [get]
// // func (h *QuestionHandler) GetJobStatus(c *gin.Context) {
// // 	id := c.Param("id")
// // 	resp, err := h.questionService.GetJobStatus(id)
// // 	if err != nil {
// // 		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, resp)
// // }

