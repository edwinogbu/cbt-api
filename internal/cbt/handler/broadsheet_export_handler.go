package handler

import (
	"net/http"
	"strconv"

	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// ExportClassBroadSheet godoc
// @Summary      Export the Class Teacher Broad Sheet as PDF, CSV, or Excel
// @Tags         Broad Sheets
// @Produce      application/octet-stream
// @Param        classId query string true "Class ID"
// @Param        reportType query string true "termly, midterm, or promotion"
// @Param        termId query string false "Term ID (required for termly/midterm)"
// @Param        sessionId query string true "Session ID"
// @Param        format query string false "pdf, csv, or excel" default(pdf)
// @Success      200 {file} binary
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class/export [get]
func (h *ExamHandler) ExportClassBroadSheet(c *gin.Context) {
	classID, reportType, termID, sessionID := c.Query("classId"), c.Query("reportType"), c.Query("termId"), c.Query("sessionId")
	if classID == "" || reportType == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId, reportType, and sessionId are required"})
		return
	}
	format := c.DefaultQuery("format", "pdf")
	data, contentType, filename, err := h.examService.ExportClassBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, reportType, termID, sessionID, format)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	writeBroadSheetExport(c, data, contentType, filename)
}

// ExportSubjectBroadSheet godoc
// @Summary      Export the Subject Broad Sheet as PDF, CSV, or Excel
// @Tags         Broad Sheets
// @Produce      application/octet-stream
// @Param        subjectId query string true "Subject ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Param        format query string false "pdf, csv, or excel" default(pdf)
// @Success      200 {file} binary
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/subject/export [get]
func (h *ExamHandler) ExportSubjectBroadSheet(c *gin.Context) {
	subjectID, termID, sessionID := c.Query("subjectId"), c.Query("termId"), c.Query("sessionId")
	if subjectID == "" || termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectId, termId, and sessionId are required"})
		return
	}
	format := c.DefaultQuery("format", "pdf")
	data, contentType, filename, err := h.examService.ExportSubjectBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), subjectID, termID, sessionID, format)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	writeBroadSheetExport(c, data, contentType, filename)
}

// ExportSchoolBroadSheet godoc
// @Summary      Export the VP Academics / School General Broad Sheet as PDF, CSV, or Excel
// @Tags         Broad Sheets
// @Produce      application/octet-stream
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Param        reportType query string false "termly, midterm, or promotion" default(termly)
// @Param        schoolId query string false "Platform-owner only: target school"
// @Param        format query string false "pdf, csv, or excel" default(pdf)
// @Success      200 {file} binary
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/school/export [get]
func (h *ExamHandler) ExportSchoolBroadSheet(c *gin.Context) {
	termID, sessionID := c.Query("termId"), c.Query("sessionId")
	if termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "termId and sessionId are required"})
		return
	}
	reportType := c.DefaultQuery("reportType", "termly")
	format := c.DefaultQuery("format", "pdf")
	data, contentType, filename, err := h.examService.ExportSchoolBroadSheet(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"), termID, sessionID, reportType, format)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	writeBroadSheetExport(c, data, contentType, filename)
}

// writeBroadSheetExport streams a generated export with the same
// attachment headers ExportClassReportCards uses, so every binary
// export in this API behaves identically to a browser/axios client.
func writeBroadSheetExport(c *gin.Context, data []byte, contentType, filename string) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}
