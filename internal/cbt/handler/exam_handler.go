package handler

import (
	// "context" 
	"net/http"
	"strconv"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/service"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

type ExamHandler struct {
	examService *service.ExamService
}

func NewExamHandler(examService *service.ExamService) *ExamHandler {
	return &ExamHandler{
		examService: examService,
	}
}

// ============================================
// EXAM MANAGEMENT HANDLERS
// ============================================

// CreateExam godoc
// @Summary      Create a new exam
// @Tags         Exam Management
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateExamRequest true "Exam details"
// @Success      201 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams [post]
func (h *ExamHandler) CreateExam(c *gin.Context) {
	var req dto.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exam, err := h.examService.CreateExam(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Exam created successfully",
		"data":    exam,
	})
}

// CreateExamWithContext godoc
// @Summary      Create a new exam with full academic context
// @Tags         Exam Management
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateExamWithContextRequest true "Exam with context"
// @Success      201 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/with-context [post]
func (h *ExamHandler) CreateExamWithContext(c *gin.Context) {
	var req dto.CreateExamWithContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exam, err := h.examService.CreateExamWithContext(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Exam created successfully with context",
		"data":    exam,
	})
}

// GetExam godoc
// @Summary      Get exam by ID
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id} [get]
func (h *ExamHandler) GetExam(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	exam, err := h.examService.GetExam(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exam,
	})
}

// GetExamWithContext godoc
// @Summary      Get exam with full context by ID
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/with-context [get]
func (h *ExamHandler) GetExamWithContext(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	exam, err := h.examService.GetExamWithContext(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exam,
	})
}

// ListExams godoc
// @Summary      List all exams with pagination
// @Tags         Exam Management
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /admin/exams [get]
func (h *ExamHandler) ListExams(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	exams, err := h.examService.ListExams(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exams,
	})
}

// ListExamsBySubject godoc
// @Summary      List exams by subject
// @Tags         Exam Management
// @Produce      json
// @Param        subjectId path string true "Subject ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /admin/exams/subject/{subjectId} [get]
func (h *ExamHandler) ListExamsBySubject(c *gin.Context) {
	subjectID := c.Param("subjectId")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject id required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	exams, err := h.examService.ListExamsBySubject(c.Request.Context(), subjectID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exams,
	})
}

// ListExamsByTerm godoc
// @Summary      List exams by term
// @Tags         Exam Management
// @Produce      json
// @Param        termId path string true "Term ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /admin/exams/term/{termId} [get]
func (h *ExamHandler) ListExamsByTerm(c *gin.Context) {
	termID := c.Param("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	exams, err := h.examService.ListExamsByTerm(c.Request.Context(), termID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exams,
	})
}

// ListExamsBySession godoc
// @Summary      List exams by session
// @Tags         Exam Management
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /admin/exams/session/{sessionId} [get]
func (h *ExamHandler) ListExamsBySession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	exams, err := h.examService.ListExamsBySession(c.Request.Context(), sessionID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exams,
	})
}

// ListExamsByClass godoc
// @Summary      List exams by class
// @Tags         Exam Management
// @Produce      json
// @Param        classId path string true "Class ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /admin/exams/class/{classId} [get]
func (h *ExamHandler) ListExamsByClass(c *gin.Context) {
	classID := c.Param("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	exams, err := h.examService.ListExamsByClass(c.Request.Context(), classID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   exams,
	})
}

// UpdateExam godoc
// @Summary      Update an exam
// @Tags         Exam Management
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        request body dto.UpdateExamRequest true "Exam update details"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id} [put]
func (h *ExamHandler) UpdateExam(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	var req dto.UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exam, err := h.examService.UpdateExam(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Exam updated successfully",
		"data":    exam,
	})
}

// DeleteExam godoc
// @Summary      Delete an exam
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id} [delete]
func (h *ExamHandler) DeleteExam(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	if err := h.examService.DeleteExam(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Exam deleted successfully",
	})
}

// ============================================
// EXAM QUESTIONS HANDLERS
// ============================================

// AddQuestionsToExam godoc
// @Summary      Add questions to an exam
// @Tags         Exam Questions
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        request body dto.AddQuestionsToExamRequest true "Question IDs"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/questions [post]
func (h *ExamHandler) AddQuestionsToExam(c *gin.Context) {
	examID := c.Param("id")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	var req dto.AddQuestionsToExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.examService.AddQuestionsToExam(c.Request.Context(), examID, req.QuestionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Questions added to exam successfully",
	})
}

// BulkAddQuestionsToExam godoc
// @Summary      Bulk add questions to an exam with filters
// @Tags         Exam Questions
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        request body dto.BulkAddQuestionsToExamRequest true "Bulk add request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/questions/bulk [post]
func (h *ExamHandler) BulkAddQuestionsToExam(c *gin.Context) {
	examID := c.Param("id")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	var req dto.BulkAddQuestionsToExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.examService.BulkAddQuestionsToExam(c.Request.Context(), examID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Questions added to exam successfully",
	})
}

// RemoveQuestionFromExam godoc
// @Summary      Remove a question from an exam
// @Tags         Exam Questions
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Param        questionId path string true "Question ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{examId}/questions/{questionId} [delete]
func (h *ExamHandler) RemoveQuestionFromExam(c *gin.Context) {
	examID := c.Param("examId")
	questionID := c.Param("questionId")

	if examID == "" || questionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id and question id required"})
		return
	}

	if err := h.examService.RemoveQuestionFromExam(c.Request.Context(), examID, questionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Question removed from exam successfully",
	})
}

// PreviewExam godoc
// @Summary      Preview an exam with all questions and statistics
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/preview [get]
func (h *ExamHandler) PreviewExam(c *gin.Context) {
	examID := c.Param("id")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	preview, err := h.examService.PreviewExam(c.Request.Context(), examID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   preview,
	})
}

// ============================================
// EXAM ASSIGNMENT HANDLERS
// ============================================

// AssignExam godoc
// @Summary      Assign an exam to students or a class
// @Tags         Exam Management
// @Accept       json
// @Produce      json
// @Param        id path string true "Exam ID"
// @Param        request body dto.AssignExamRequest true "Assignment details"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/assign [post]
func (h *ExamHandler) AssignExam(c *gin.Context) {
	examID := c.Param("id")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	var req dto.AssignExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.examService.AssignExam(c.Request.Context(), examID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Exam assigned successfully",
	})
}

// ============================================
// STUDENT EXAM TAKING HANDLERS
// ============================================

// StartExam godoc
// @Summary      Start an exam for a student
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.StartExamRequest true "Start exam request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/start [post]
func (h *ExamHandler) StartExam(c *gin.Context) {
	var req dto.StartExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get student ID from context (authenticated user)
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	req.StudentID = userID

	resp, err := h.examService.StartExam(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "exam has not started yet" || err.Error() == "exam has already ended" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}


// Add these methods to ExamHandler

// GetStudentDashboardForUser godoc
// @Summary      Get student dashboard for authenticated user
// @Tags         Student Dashboard
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/dashboard [get]
func (h *ExamHandler) GetStudentDashboardForUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.examService.GetStudentDashboardForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   dashboard,
	})
}


// // ============================================
// // StartExamForUser - Updated to call StartExamForStudent
// // ============================================
// // StartExamForUser godoc
// // @Summary      Start an exam for authenticated student
// // @Tags         Student Exam
// // @Accept       json
// // @Produce      json
// // @Param        examId path string true "Exam ID"
// // @Param        request body dto.StartExamRequest true "Start exam request"
// // @Success      200 {object} map[string]interface{} "data"
// // @Failure      400 {object} map[string]interface{}
// // @Failure      403 {object} map[string]interface{}
// // @Failure      404 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /student/exams/start/{examId} [post]
// func (h *ExamHandler) StartExamForUser(c *gin.Context) {
//     examID := c.Param("examId")
//     if examID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
//         return
//     }

//     var req dto.StartExamRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     // Set IP address from request
//     req.IPAddress = c.ClientIP()
    
//     // Device info from user agent
//     req.DeviceInfo = c.GetHeader("User-Agent")

//     // Get user ID from context
//     userID := middleware.GetUserID(c)
//     if userID == "" {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     // ✅ FIX: Use the correct method name and pass user_id in context
//     ctx := context.WithValue(c.Request.Context(), "user_id", userID)
    
//     resp, err := h.examService.StartExamForStudent(ctx, examID, &req)
//     if err != nil {
//         status := http.StatusInternalServerError
//         switch err.Error() {
//         case "exam has not started yet", "exam has already ended":
//             status = http.StatusForbidden
//         case "exam not found", "student not found":
//             status = http.StatusNotFound
//         case "exam not assigned to your class":
//             status = http.StatusForbidden
//         }
//         c.JSON(status, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   resp,
//     })
// }
// handler/exam_handler.go

// StartExamForUser godoc
// @Summary      Start an exam for authenticated student
// @Description  Starts an exam for the authenticated student. 
// @Description  Works with or without request body. Exam ID from URL path is always used.
// @Description  IP address and device info are automatically captured.
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Param        request body dto.StartExamRequest false "Optional request body (exam_id is ignored, URL param takes precedence)"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/start/{examId} [post]
func (h *ExamHandler) StartExamForUser(c *gin.Context) {
    // 1. Get exam ID from URL path (required)
    examID := c.Param("examId")
    if examID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "exam id is required in the URL path"})
        return
    }

    // 2. Build request with automatic values
    req := &dto.StartExamRequest{
        ExamID:     examID,                           // From URL path
        IPAddress:  c.ClientIP(),                     // Automatically from request
        DeviceInfo: c.GetHeader("User-Agent"),        // Automatically from request
        StudentID:  "",                               // Will be filled from context
    }

    // 3. Try to bind JSON body if present (optional)
    //    This allows the client to send additional data, but doesn't require it
    var bodyReq dto.StartExamRequest
    if err := c.ShouldBindJSON(&bodyReq); err == nil {
        // Body was valid JSON - use any non-empty fields from body
        // But URL param always takes precedence for ExamID
        if bodyReq.IPAddress != "" {
            req.IPAddress = bodyReq.IPAddress
        }
        if bodyReq.DeviceInfo != "" {
            req.DeviceInfo = bodyReq.DeviceInfo
        }
        // Note: ExamID from body is ignored, URL param is authoritative
        // StudentID from body is ignored for security
    }
    // If body is empty or invalid JSON, we just continue with defaults

    // 4. Get student ID from Gin context (set by middleware)
    studentID := middleware.GetStudentID(c)
    if studentID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "student context not found"})
        return
    }
    req.StudentID = studentID

    // 5. Call service with complete request
    resp, err := h.examService.StartExamForStudent(c.Request.Context(), examID, studentID, req)
    if err != nil {
        status := http.StatusInternalServerError
        switch err.Error() {
        case "exam has not started yet", "exam has already ended":
            status = http.StatusForbidden
        case "exam not found", "student not found":
            status = http.StatusNotFound
        case "exam not assigned to your class":
            status = http.StatusForbidden
        }
        c.JSON(status, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// GetExamPackageForUser godoc
// @Summary      Download an exam package for offline use (no attempt created)
// @Tags         Student Exams
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/package/{examId} [get]
func (h *ExamHandler) GetExamPackageForUser(c *gin.Context) {
    examID := c.Param("examId")
    if examID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "exam id is required in the URL path"})
        return
    }

    studentID := middleware.GetStudentID(c)
    if studentID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "student context not found"})
        return
    }

    resp, err := h.examService.GetExamPackageForStudent(c.Request.Context(), examID, studentID)
    if err != nil {
        status := http.StatusInternalServerError
        switch err.Error() {
        case "exam has already ended":
            status = http.StatusForbidden
        case "exam not found", "student not found":
            status = http.StatusNotFound
        case "exam not assigned to your class":
            status = http.StatusForbidden
        }
        c.JSON(status, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// GetStudentExamResultForUser godoc
// @Summary      Get exam result for authenticated student
// @Tags         Student Results
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/result/{examId} [get]
func (h *ExamHandler) GetStudentExamResultForUser(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.examService.GetStudentExamResultForUser(c.Request.Context(), examID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetStudentExamReviewForUser godoc
// @Summary      Get exam review for authenticated student
// @Tags         Student Results
// @Produce      json
// @Param        attemptId path string true "Attempt ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/review/{attemptId} [get]
func (h *ExamHandler) GetStudentExamReviewForUser(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if attemptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attempt id required"})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	review, err := h.examService.GetExamReviewForUser(c.Request.Context(), attemptID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "unauthorized", "review not allowed for this exam":
			status = http.StatusForbidden
		case "attempt not found":
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   review,
	})
}

// GetStudentPerformanceForUser godoc
// @Summary      Get performance overview for authenticated student
// @Tags         Student Results
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/performance [get]
func (h *ExamHandler) GetStudentPerformanceForUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	performance, err := h.examService.GetStudentPerformanceForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   performance,
	})
}

// GetStudentTermlyResultForUser godoc
// @Summary      Get termly result for authenticated student
// @Tags         Termly Results
// @Produce      json
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/termly-result [get]
func (h *ExamHandler) GetStudentTermlyResultForUser(c *gin.Context) {
	termID := c.Query("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.examService.GetTermlyResultForUser(c.Request.Context(), termID, sessionID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetTermlyReportCardForUser godoc
// @Summary      Get report card for authenticated student
// @Tags         Termly Results
// @Produce      json
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/report-card [get]
func (h *ExamHandler) GetTermlyReportCardForUser(c *gin.Context) {
	termID := c.Query("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportCard, err := h.examService.GetReportCardForUser(c.Request.Context(), termID, sessionID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   reportCard,
	})
}

// StartPracticeForUser godoc
// @Summary      Start practice session for authenticated student
// @Tags         Practice
// @Accept       json
// @Produce      json
// @Param        subjectId query string true "Subject ID"
// @Param        questionCount query int false "Number of questions" default(10)
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/practice [post]
func (h *ExamHandler) StartPracticeForUser(c *gin.Context) {
	subjectID := c.Query("subjectId")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject id required"})
		return
	}

	questionCount, _ := strconv.Atoi(c.DefaultQuery("questionCount", "10"))

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	session, err := h.examService.StartPracticeForUser(c.Request.Context(), subjectID, questionCount, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   session,
	})
}

// GetAttemptState godoc
// @Summary      Get current state of an exam attempt
// @Tags         Student Exam
// @Produce      json
// @Param        attemptId path string true "Attempt ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/attempt/{attemptId} [get]
func (h *ExamHandler) GetAttemptState(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if attemptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attempt id required"})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	state, err := h.examService.GetAttemptState(c.Request.Context(), attemptID, studentID)
	if err != nil {
		status := http.StatusNotFound
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   state,
	})
}

// SaveAnswer godoc
// @Summary      Save a student's answer
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.SaveAnswerRequest true "Save answer request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/answer [post]
func (h *ExamHandler) SaveAnswer(c *gin.Context) {
	var req dto.SaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.SaveAnswer(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "time limit exceeded", "unauthorized":
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// BulkSaveAnswers godoc
// @Summary      Save multiple answers (for offline sync)
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkSaveAnswerRequest true "Bulk save request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/bulk-answer [post]
func (h *ExamHandler) BulkSaveAnswers(c *gin.Context) {
	var req dto.BulkSaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.BulkSaveAnswers(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// MarkReview godoc
// @Summary      Mark a question for review
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.MarkReviewRequest true "Mark review request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/mark-review [post]
func (h *ExamHandler) MarkReview(c *gin.Context) {
	var req dto.MarkReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.MarkReview(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// AutoSave godoc
// @Summary      Auto-save answers (for offline support)
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.AutoSaveRequest true "Auto-save request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/auto-save [post]
func (h *ExamHandler) AutoSave(c *gin.Context) {
	var req dto.AutoSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.AutoSave(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// SubmitExam godoc
// @Summary      Submit an exam for grading
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.SubmitExamRequest true "Submit exam request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/submit [post]
func (h *ExamHandler) SubmitExam(c *gin.Context) {
	var req dto.SubmitExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.SubmitExam(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "exam already submitted", "unauthorized":
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// OfflineSubmit godoc
// @Summary      Submit an exam offline (queued for later grading)
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.OfflineSubmitRequest true "Offline submit request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/offline-submit [post]
func (h *ExamHandler) OfflineSubmit(c *gin.Context) {
	var req dto.OfflineSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.OfflineSubmit(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// SyncOfflineAnswers godoc
// @Summary      Sync offline answers when online
// @Tags         Student Exam
// @Accept       json
// @Produce      json
// @Param        request body dto.SyncAnswersRequest true "Sync answers request"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/sync [post]
func (h *ExamHandler) SyncOfflineAnswers(c *gin.Context) {
	var req dto.SyncAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student context not found"})
		return
	}

	resp, err := h.examService.SyncOfflineAnswers(c.Request.Context(), &req, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// ============================================
// STUDENT RESULTS HANDLERS
// ============================================

// GetStudentExamResult godoc
// @Summary      Get a student's exam result
// @Tags         Student Results
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/result/{examId} [get]
func (h *ExamHandler) GetStudentExamResult(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	studentID := middleware.GetUserID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.examService.GetStudentExamResult(c.Request.Context(), studentID, examID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetStudentExamReview godoc
// @Summary      Get detailed exam review with correct answers
// @Tags         Student Results
// @Produce      json
// @Param        attemptId path string true "Attempt ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/review/{attemptId} [get]
func (h *ExamHandler) GetStudentExamReview(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if attemptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attempt id required"})
		return
	}

	studentID := middleware.GetUserID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	review, err := h.examService.GetExamReview(c.Request.Context(), attemptID, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" || err.Error() == "review not allowed for this exam" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   review,
	})
}

// GetStudentDashboard godoc
// @Summary      Get student dashboard with all exams
// @Tags         Student Dashboard
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/dashboard [get]
func (h *ExamHandler) GetStudentDashboard(c *gin.Context) {
	studentID := middleware.GetUserID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.examService.GetStudentDashboard(c.Request.Context(), studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   dashboard,
	})
}

// ============================================
// TEACHER RESULTS HANDLERS
// ============================================

// GetClassResults godoc
// @Summary      Get class results for an exam
// @Tags         Teacher Results
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Param        classId query string true "Class ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/exams/{examId}/results/class [get]
func (h *ExamHandler) GetClassResults(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	results, err := h.examService.GetClassResults(c.Request.Context(), examID, classID, page, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   results,
	})
}

// GetExamRankings godoc
// @Summary      Get exam rankings
// @Tags         Teacher Results
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Param        limit query int false "Number of rankings" default(10)
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/exams/{examId}/rankings [get]
func (h *ExamHandler) GetExamRankings(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	rankings, err := h.examService.GetExamRankings(c.Request.Context(), examID, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   rankings,
	})
}

// GetExamStatistics godoc
// @Summary      Get exam statistics
// @Tags         Teacher Results
// @Produce      json
// @Param        examId path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/exams/{examId}/statistics [get]
func (h *ExamHandler) GetExamStatistics(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	stats, err := h.examService.GetExamStatistics(c.Request.Context(), examID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ExportExamResults godoc
// @Summary      Export exam results to CSV or Excel
// @Tags         Teacher Results
// @Produce      application/octet-stream
// @Param        examId path string true "Exam ID"
// @Param        classId query string true "Class ID"
// @Param        format query string true "Export format (csv or excel)"
// @Success      200 {file} file
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/exams/{examId}/export [get]
func (h *ExamHandler) ExportExamResults(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "csv"
	}

	data, contentType, err := h.examService.ExportExamResults(c.Request.Context(), examID, classID, format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filename := "exam_results." + format
	if format == "csv" {
		filename = "exam_results.csv"
	} else if format == "excel" {
		filename = "exam_results.xlsx"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}

// GetTeacherDashboard godoc
// @Summary      Get teacher dashboard
// @Tags         Teacher Dashboard
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/dashboard [get]
func (h *ExamHandler) GetTeacherDashboard(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	if teacherID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.examService.GetTeacherDashboard(c.Request.Context(), teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   dashboard,
	})
}

// ============================================
// TERMLY RESULTS HANDLERS (WAEC/NECO Style)
// ============================================

// GetStudentTermlyResult godoc
// @Summary      Get termly result for a student (WAEC/NECO style)
// @Tags         Termly Results
// @Produce      json
// @Param        studentId query string true "Student ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/termly-result [get]
func (h *ExamHandler) GetStudentTermlyResult(c *gin.Context) {
	studentID := c.Query("studentId")
	if studentID == "" {
		studentID = middleware.GetUserID(c)
	}
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}

	termID := c.Query("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	result, err := h.examService.GetStudentTermlyResult(c.Request.Context(), studentID, termID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetTermlyReportCard godoc
// @Summary      Get termly report card (WAEC/NECO style)
// @Tags         Termly Results
// @Produce      json
// @Param        studentId query string true "Student ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/report-card [get]
func (h *ExamHandler) GetTermlyReportCard(c *gin.Context) {
	studentID := c.Query("studentId")
	if studentID == "" {
		studentID = middleware.GetUserID(c)
	}
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}

	termID := c.Query("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	reportCard, err := h.examService.GetTermlyReportCard(c.Request.Context(), studentID, termID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   reportCard,
	})
}

// ============================================
// PRACTICE SESSIONS HANDLERS
// ============================================

// StartPractice godoc
// @Summary      Start a practice session
// @Tags         Practice
// @Accept       json
// @Produce      json
// @Param        subjectId query string true "Subject ID"
// @Param        questionCount query int false "Number of questions" default(10)
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/practice/start [post]
func (h *ExamHandler) StartPractice(c *gin.Context) {
	studentID := middleware.GetUserID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subjectID := c.Query("subjectId")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject id required"})
		return
	}

	questionCount, _ := strconv.Atoi(c.DefaultQuery("questionCount", "10"))

	session, err := h.examService.StartPractice(c.Request.Context(), studentID, subjectID, questionCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   session,
	})
}

// ============================================
// ADD MISSING HANDLER METHODS
// ============================================

// // GetExamAssignments godoc
// // @Summary      Get assignments for an exam
// // @Tags         Exam Management
// // @Produce      json
// // @Param        examId path string true "Exam ID"
// // @Success      200 {object} map[string]interface{} "data"
// // @Failure      404 {object} map[string]interface{}
// // @Security     BearerAuth
// // @Router       /admin/exams/{examId}/assignments [get]
// func (h *ExamHandler) GetExamAssignments(c *gin.Context) {
// 	examID := c.Param("examId")
// 	if examID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
// 		return
// 	}

// 	assignments, err := h.examService.GetExamAssignments(c.Request.Context(), examID)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   assignments,
// 	})
// }

// GetExamAssignments godoc
// @Summary      Get assignments for an exam
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/assignments [get]
func (h *ExamHandler) GetExamAssignments(c *gin.Context) {
    // FIX: Use "id" to match route parameter
    examID := c.Param("id")
    if examID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
        return
    }

    assignments, err := h.examService.GetExamAssignments(c.Request.Context(), examID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   assignments,
    })
}

// ExportResults godoc
// @Summary      Export exam results
// @Tags         Teacher Results
// @Produce      application/octet-stream
// @Param        examId path string true "Exam ID"
// @Param        classId query string true "Class ID"
// @Param        format query string false "Export format (csv or excel)" default(csv)
// @Success      200 {file} file
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/exams/{examId}/export [get]
func (h *ExamHandler) ExportResults(c *gin.Context) {
	examID := c.Param("examId")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	format := c.DefaultQuery("format", "csv")

	data, contentType, err := h.examService.ExportExamResults(c.Request.Context(), examID, classID, format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filename := "exam_results." + format
	if format == "csv" {
		filename = "exam_results.csv"
	} else if format == "excel" {
		filename = "exam_results.xlsx"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}

// GetStudentPerformance godoc
// @Summary      Get student performance overview
// @Tags         Student Results
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/{studentId}/performance [get]
func (h *ExamHandler) GetStudentPerformance(c *gin.Context) {
	studentID := c.Param("studentId")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}

	performance, err := h.examService.GetStudentPerformance(c.Request.Context(), studentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   performance,
	})
}

// GetTeacherSubjectResults godoc
// @Summary      Get subject results for a teacher
// @Tags         Teacher Results
// @Produce      json
// @Param        subjectId query string true "Subject ID"
// @Param        classId query string true "Class ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/results/subject [get]
func (h *ExamHandler) GetTeacherSubjectResults(c *gin.Context) {
	subjectID := c.Query("subjectId")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject id required"})
		return
	}

	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	results, err := h.examService.GetTeacherSubjectResults(c.Request.Context(), subjectID, classID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   results,
	})
}

// GetTeacherClassResults godoc
// @Summary      Get class results for a teacher
// @Tags         Teacher Results
// @Produce      json
// @Param        classId query string true "Class ID"
// @Param        subjectId query string false "Subject ID (optional)"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/results/class [get]
func (h *ExamHandler) GetTeacherClassResults(c *gin.Context) {
	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	subjectID := c.Query("subjectId")

	results, err := h.examService.GetTeacherClassResults(c.Request.Context(), classID, subjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   results,
	})
}

// GetSchoolPerformanceOverview godoc
// @Summary      Get school performance overview
// @Tags         School Performance
// @Produce      json
// @Param        schoolId query string true "School ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/school/{schoolId}/performance [get]
func (h *ExamHandler) GetSchoolPerformanceOverview(c *gin.Context) {
	schoolID := c.Query("schoolId")
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "school id required"})
		return
	}

	overview, err := h.examService.GetSchoolPerformanceOverview(c.Request.Context(), schoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   overview,
	})
}

// GetSubjectQuestionsForExam godoc
// @Summary      Get subject questions for exam creation
// @Tags         Exam Management
// @Produce      json
// @Param        subjectId query string true "Subject ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Param        classId query string true "Class ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/questions/subject [get]
func (h *ExamHandler) GetSubjectQuestionsForExam(c *gin.Context) {
	subjectID := c.Query("subjectId")
	if subjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject id required"})
		return
	}

	termID := c.Query("termId")
	if termID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term id required"})
		return
	}

	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id required"})
		return
	}

	classID := c.Query("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class id required"})
		return
	}

	questions, err := h.examService.GetSubjectQuestionsForExam(c.Request.Context(), subjectID, termID, sessionID, classID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   questions,
	})
}

// PublishExam godoc
// @Summary      Publish an exam
// @Tags         Exam Management
// @Produce      json
// @Param        id path string true "Exam ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/exams/{id}/publish [post]
func (h *ExamHandler) PublishExam(c *gin.Context) {
	examID := c.Param("id")
	if examID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam id required"})
		return
	}

	if err := h.examService.PublishExam(c.Request.Context(), examID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Exam published successfully",
	})
}

