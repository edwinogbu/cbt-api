package handler

import (
	"net/http"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func bindSubjectResultScope(c *gin.Context) (*dto.SubjectResultScopeRequest, bool) {
	var req dto.SubjectResultScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	return &req, true
}

// GetSubjectResultSubmissionStatus godoc
// @Summary      Get a class/subject/term/session's approval-chain status
// @Tags         Subject Result Submissions
// @Produce      json
// @Param        classId query string true "Class ID"
// @Param        subjectId query string true "Subject ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /exams/score-register/status [get]
func (h *ExamHandler) GetSubjectResultSubmissionStatus(c *gin.Context) {
	classID, subjectID, termID, sessionID := c.Query("classId"), c.Query("subjectId"), c.Query("termId"), c.Query("sessionId")
	if classID == "" || subjectID == "" || termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, subjectId, termId, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetSubjectResultSubmissionStatus(c.Request.Context(), middleware.GetSchoolID(c), classID, subjectID, termID, sessionID)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// SubmitSubjectResults godoc
// @Summary      Teacher: submit a class/subject's results for review/approval
// @Tags         Subject Result Submissions
// @Accept       json
// @Produce      json
// @Param        request body dto.SubjectResultScopeRequest true "Scope"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register/submit [post]
func (h *ExamHandler) SubmitSubjectResults(c *gin.Context) {
	req, ok := bindSubjectResultScope(c)
	if !ok {
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.examService.SubmitSubjectResults(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ReviewSubjectResults godoc
// @Summary      HOD/Examination Officer: mark a submitted result set reviewed
// @Tags         Subject Result Submissions
// @Accept       json
// @Produce      json
// @Param        request body dto.SubjectResultScopeRequest true "Scope"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register/review [post]
func (h *ExamHandler) ReviewSubjectResults(c *gin.Context) {
	req, ok := bindSubjectResultScope(c)
	if !ok {
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.examService.ReviewSubjectResults(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ReturnSubjectResults godoc
// @Summary      Send a result set back to the subject teacher for correction
// @Tags         Subject Result Submissions
// @Accept       json
// @Produce      json
// @Param        request body dto.SubjectResultScopeRequest true "Scope (notes required)"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register/return [post]
func (h *ExamHandler) ReturnSubjectResults(c *gin.Context) {
	req, ok := bindSubjectResultScope(c)
	if !ok {
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.examService.ReturnSubjectResults(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ApproveSubjectResults godoc
// @Summary      Academic administrator: approve and lock a result set
// @Tags         Subject Result Submissions
// @Accept       json
// @Produce      json
// @Param        request body dto.SubjectResultScopeRequest true "Scope"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register/approve [post]
func (h *ExamHandler) ApproveSubjectResults(c *gin.Context) {
	req, ok := bindSubjectResultScope(c)
	if !ok {
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.examService.ApproveSubjectResults(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
