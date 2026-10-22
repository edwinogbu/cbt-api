package handler

import (
	"net/http"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func statusForAttendanceError(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "not found") || strings.Contains(msg, "required") ||
		strings.Contains(msg, "must be") || strings.Contains(msg, "cannot") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// MarkClassAttendance godoc
// @Summary      Record a class's attendance register for one date
// @Tags         Attendance
// @Accept       json
// @Produce      json
// @Param        request body dto.MarkClassAttendanceRequest true "Attendance register"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/attendance/mark [post]
func (h *ExamHandler) MarkClassAttendance(c *gin.Context) {
	var req dto.MarkClassAttendanceRequest
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
	resp, err := h.examService.MarkClassAttendance(c.Request.Context(), schoolID, userID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForAttendanceError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetClassAttendanceForDate godoc
// @Summary      Get a class's attendance register for one date
// @Tags         Attendance
// @Produce      json
// @Param        classId path string true "Class ID"
// @Param        date query string true "YYYY-MM-DD"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/attendance/class/{classId} [get]
func (h *ExamHandler) GetClassAttendanceForDate(c *gin.Context) {
	classID := c.Param("classId")
	date := c.Query("date")
	if classID == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classId and date are required"})
		return
	}
	resp, err := h.examService.GetClassAttendanceForDate(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), classID, date)
	if err != nil {
		c.JSON(statusForAttendanceError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetStudentAttendanceSummary godoc
// @Summary      Get one student's attendance summary for a term
// @Tags         Attendance
// @Produce      json
// @Param        studentId query string true "Student ID"
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/attendance/summary [get]
func (h *ExamHandler) GetStudentAttendanceSummary(c *gin.Context) {
	studentID := c.Query("studentId")
	termID := c.Query("termId")
	sessionID := c.Query("sessionId")
	if studentID == "" || termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "studentId, termId, and sessionId are required"})
		return
	}
	resp, err := h.examService.GetStudentAttendanceSummary(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), studentID, termID, sessionID)
	if err != nil {
		c.JSON(statusForAttendanceError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// GetMyAttendanceSummary godoc
// @Summary      Get the authenticated student's own attendance summary for a term
// @Tags         Student Results
// @Produce      json
// @Param        termId query string true "Term ID"
// @Param        sessionId query string true "Session ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /student/exams/attendance/summary [get]
func (h *ExamHandler) GetMyAttendanceSummary(c *gin.Context) {
	termID := c.Query("termId")
	sessionID := c.Query("sessionId")
	if termID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "termId and sessionId are required"})
		return
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.GetMyAttendanceSummary(c.Request.Context(), userID, termID, sessionID)
	if err != nil {
		c.JSON(statusForAttendanceError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
