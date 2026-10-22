package handler

import (
	"net/http"

	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// GetClassBroadSheet godoc
// @Summary      Class Teacher Broad Sheet: every student x every subject for a class
// @Tags         Broad Sheets
// @Produce      json
// @Param        classId query string true "Class ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class [get]
func (h *ExamHandler) GetClassBroadSheet(c *gin.Context) {
	classID, reportType, termID, sessionID := c.Query("classId"), c.Query("reportType"), c.Query("termId"), c.Query("sessionId")
	if classID == "" || reportType == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, reportType, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetClassBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, reportType, termID, sessionID)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetSubjectBroadSheet godoc
// @Summary      HOD/Examination Officer Broad Sheet: one subject across every class
// @Tags         Broad Sheets
// @Produce      json
// @Param        subjectId query string true "Subject ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/subject [get]
func (h *ExamHandler) GetSubjectBroadSheet(c *gin.Context) {
	subjectID, termID, sessionID := c.Query("subjectId"), c.Query("termId"), c.Query("sessionId")
	if subjectID == "" || termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectId, termId, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetSubjectBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), subjectID, termID, sessionID)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetSchoolBroadSheet godoc
// @Summary      VP Academics / School General Broad Sheet: school-wide class x subject monitor
// @Tags         Broad Sheets
// @Produce      json
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Param        reportType query string false "termly, midterm, or promotion" default(termly)
// @Param        schoolId query string false "Platform-owner only: target school"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/school [get]
func (h *ExamHandler) GetSchoolBroadSheet(c *gin.Context) {
	termID, sessionID := c.Query("termId"), c.Query("sessionId")
	if termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "termId and sessionId are required"})
		return
	}
	reportType := c.DefaultQuery("reportType", "termly")
	resp, err := h.examService.GetSchoolBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"), termID, sessionID, reportType)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
