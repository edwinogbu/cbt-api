package handler

import (
	"net/http"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// GetScoreRegister godoc
// @Summary      Get the Student Score Register for one class/subject/term/session
// @Tags         Score Register
// @Produce      json
// @Param        classId query string true "Class ID"
// @Param        subjectId query string true "Subject ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register [get]
func (h *ExamHandler) GetScoreRegister(c *gin.Context) {
	classID, subjectID, termID, sessionID := c.Query("classId"), c.Query("subjectId"), c.Query("termId"), c.Query("sessionId")
	if classID == "" || subjectID == "" || termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, subjectId, termId, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetScoreRegister(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, subjectID, termID, sessionID)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// SaveScoreRegister godoc
// @Summary      Batch-save one component (column) of the Student Score Register
// @Tags         Score Register
// @Accept       json
// @Produce      json
// @Param        request body dto.SaveScoreRegisterRequest true "Scores to save"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/score-register/save [post]
func (h *ExamHandler) SaveScoreRegister(c *gin.Context) {
	var req dto.SaveScoreRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.SaveScoreRegister(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
