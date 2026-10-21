package handler

import (
	"net/http"
	"strconv"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// statusForApprovalError maps a report-card approval/batch service
// error to an HTTP status: "not found"/"required"/"belong"/"must be"
// are caller mistakes (bad student/school/report-type), everything
// else is treated as an unexpected server-side failure.
func statusForApprovalError(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "not found") || strings.Contains(msg, "required") ||
		strings.Contains(msg, "belong") || strings.Contains(msg, "must be") ||
		strings.Contains(msg, "cannot be approved") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// ApproveReportCard godoc
// @Summary      Approve a student's report card, freezing a snapshot as the official copy
// @Tags         Report Card Approvals
// @Accept       json
// @Produce      json
// @Param        request body dto.ApproveReportCardRequest true "Approval"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/report-card/approve [post]
func (h *ExamHandler) ApproveReportCard(c *gin.Context) {
	var req dto.ApproveReportCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	schoolID := middleware.ResolveSchoolID(c, "")
	if userID == "" || schoolID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.ApproveReportCard(c.Request.Context(), schoolID, userID, &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// RejectReportCard godoc
// @Summary      Reject a student's report card, sending it back for correction
// @Tags         Report Card Approvals
// @Accept       json
// @Produce      json
// @Param        request body dto.RejectReportCardRequest true "Rejection"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/report-card/reject [post]
func (h *ExamHandler) RejectReportCard(c *gin.Context) {
	var req dto.RejectReportCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	schoolID := middleware.ResolveSchoolID(c, "")
	if userID == "" || schoolID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.RejectReportCard(c.Request.Context(), schoolID, userID, &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetReportCardApprovalStatus godoc
// @Summary      Get a student's report card's current approval status
// @Tags         Report Card Approvals
// @Produce      json
// @Param        studentId query string true "Student ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/report-card/approval-status [get]
func (h *ExamHandler) GetReportCardApprovalStatus(c *gin.Context) {
	scope := dto.ReportCardScope{
		StudentID:  c.Query("studentId"),
		ReportType: c.Query("reportType"),
		TermID:     c.Query("termId"),
		SessionID:  c.Query("sessionId"),
	}
	if scope.StudentID == "" || scope.ReportType == "" || scope.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "studentId, reportType, and sessionId are required"})
		return
	}
	schoolID := middleware.GetSchoolID(c)
	resp, err := h.examService.GetApprovalStatus(c.Request.Context(), schoolID, middleware.IsPlatformOwner(c), scope)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetReportCardApprovalHistory godoc
// @Summary      Get the report-card approval audit trail for this school
// @Tags         Report Card Approvals
// @Produce      json
// @Param        studentId query string false "Narrow to one student"
// @Param        reportType query string false "Narrow to termly, midterm, or promotion"
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /exams/report-card/approval-history [get]
func (h *ExamHandler) GetReportCardApprovalHistory(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	if schoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no school context"})
		return
	}
	entries, err := h.examService.GetApprovalHistory(c.Request.Context(), schoolID, c.Query("studentId"), c.Query("reportType"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": entries})
}

// BatchApproveReportCards godoc
// @Summary      Approve every (or every named) student's report card in a class
// @Tags         Report Card Approvals
// @Accept       json
// @Produce      json
// @Param        request body dto.BatchApproveRequest true "Batch approval"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/report-card/batch-approve [post]
func (h *ExamHandler) BatchApproveReportCards(c *gin.Context) {
	var req dto.BatchApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	schoolID := middleware.ResolveSchoolID(c, "")
	if userID == "" || schoolID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.BatchApproveReportCards(c.Request.Context(), schoolID, userID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetClassReportCardSummary godoc
// @Summary      Get a class's report-card roster summary (figures + approval status per student)
// @Tags         Report Card Batch
// @Produce      json
// @Param        classId path string true "Class ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/classes/{classId}/report-cards [get]
func (h *ExamHandler) GetClassReportCardSummary(c *gin.Context) {
	classID := c.Param("classId")
	reportType := c.Query("reportType")
	termID := c.Query("termId")
	sessionID := c.Query("sessionId")
	if classID == "" || reportType == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, reportType, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetClassReportCardSummary(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, reportType, termID, sessionID)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// DownloadClassReportCardsZIP godoc
// @Summary      Download every student in a class's report card PDF, packed into one ZIP
// @Tags         Report Card Batch
// @Produce      application/zip
// @Param        classId path string true "Class ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Success      200 {file} binary
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/classes/{classId}/report-cards/batch-pdf [get]
func (h *ExamHandler) DownloadClassReportCardsZIP(c *gin.Context) {
	classID := c.Param("classId")
	reportType := c.Query("reportType")
	termID := c.Query("termId")
	sessionID := c.Query("sessionId")
	if classID == "" || reportType == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, reportType, and sessionId are required"})
		return
	}
	data, filename, skipped, err := h.examService.GenerateClassReportCardsZIP(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, reportType, termID, sessionID)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	if len(skipped) > 0 {
		// A student's PDF failing to generate must never silently
		// shrink the ZIP with no sign anything was left out - named in
		// a response header the caller can surface, while the download
		// itself still proceeds for every student who did succeed.
		c.Header("X-Report-Cards-Skipped", strconv.Itoa(len(skipped)))
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/zip", data)
}

// ExportClassReportCards godoc
// @Summary      Export a class's report-card result sheet as CSV or Excel
// @Tags         Report Card Batch
// @Produce      application/octet-stream
// @Param        classId path string true "Class ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Param        format query string false "csv or excel" default(csv)
// @Success      200 {file} binary
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/classes/{classId}/report-cards/export [get]
func (h *ExamHandler) ExportClassReportCards(c *gin.Context) {
	classID := c.Param("classId")
	reportType := c.Query("reportType")
	termID := c.Query("termId")
	sessionID := c.Query("sessionId")
	format := c.DefaultQuery("format", "csv")
	if classID == "" || reportType == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, reportType, and sessionId are required"})
		return
	}
	data, contentType, filename, err := h.examService.ExportClassReportCards(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, reportType, termID, sessionID, format)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}
