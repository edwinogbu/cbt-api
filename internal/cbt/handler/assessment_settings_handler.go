package handler

import (
	"net/http"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// statusForSettingsError maps a validation/not-found error from the
// assessment-settings or teacher-subject-assignment service to an HTTP
// status, mirroring statusForApprovalError's own convention.
func statusForSettingsError(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "not found") || strings.Contains(msg, "required") ||
		strings.Contains(msg, "must") || strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "claimed by") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// UpsertAssessmentScheme godoc
// @Summary      Create or replace an assessment scheme (weighted CA/Test/Exam components) at a scope
// @Tags         Assessment Settings
// @Accept       json
// @Produce      json
// @Param        request body dto.AssessmentSchemeRequest true "Scheme + components"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/assessment-schemes [post]
func (h *ExamHandler) UpsertAssessmentScheme(c *gin.Context) {
	var req dto.AssessmentSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	resp, err := h.examService.UpsertAssessmentScheme(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"), userID, &req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ListAssessmentSchemes godoc
// @Summary      List this school's assessment schemes
// @Tags         Assessment Settings
// @Produce      json
// @Param        classLevelId query string false "Narrow to one class level (and school-wide schemes)"
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /academic/assessment-schemes [get]
func (h *ExamHandler) ListAssessmentSchemes(c *gin.Context) {
	resp, err := h.examService.ListAssessmentSchemes(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"), c.Query("classLevelId"))
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// DeactivateAssessmentScheme godoc
// @Summary      Deactivate an assessment scheme
// @Tags         Assessment Settings
// @Produce      json
// @Param        id path string true "Scheme ID"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/assessment-schemes/{id} [delete]
func (h *ExamHandler) DeactivateAssessmentScheme(c *gin.Context) {
	if err := h.examService.DeactivateAssessmentScheme(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Param("id")); err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// UpsertGradingScale godoc
// @Summary      Create or replace a grading scale (grade boundaries) at a scope
// @Tags         Assessment Settings
// @Accept       json
// @Produce      json
// @Param        request body dto.GradingScaleRequest true "Scale + bands"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/grading-scales [post]
func (h *ExamHandler) UpsertGradingScale(c *gin.Context) {
	var req dto.GradingScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	resp, err := h.examService.UpsertGradingScale(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"), userID, &req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ListGradingScales godoc
// @Summary      List this school's grading scales
// @Tags         Assessment Settings
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /academic/grading-scales [get]
func (h *ExamHandler) ListGradingScales(c *gin.Context) {
	resp, err := h.examService.ListGradingScales(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("schoolId"))
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// DeactivateGradingScale godoc
// @Summary      Deactivate a grading scale
// @Tags         Assessment Settings
// @Produce      json
// @Param        id path string true "Scale ID"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/grading-scales/{id} [delete]
func (h *ExamHandler) DeactivateGradingScale(c *gin.Context) {
	if err := h.examService.DeactivateGradingScale(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Param("id")); err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// CreateTeacherSubjectAssignment godoc
// @Summary      Assign a teacher to a subject within a class for a session
// @Tags         Teacher Subject Assignments
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateTeacherSubjectAssignmentRequest true "Assignment"
// @Success      201 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/teacher-subject-assignments [post]
func (h *ExamHandler) CreateTeacherSubjectAssignment(c *gin.Context) {
	var req dto.CreateTeacherSubjectAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.examService.CreateTeacherSubjectAssignment(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": resp})
}

// ListTeacherSubjectAssignments godoc
// @Summary      List teacher-subject assignments for this school
// @Tags         Teacher Subject Assignments
// @Produce      json
// @Param        classId query string false "Narrow to one class"
// @Param        teacherId query string false "Narrow to one teacher"
// @Param        sessionId query string false "Narrow to one session"
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /academic/teacher-subject-assignments [get]
func (h *ExamHandler) ListTeacherSubjectAssignments(c *gin.Context) {
	resp, err := h.examService.ListTeacherSubjectAssignments(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Query("classId"), c.Query("teacherId"), c.Query("sessionId"))
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// DeleteTeacherSubjectAssignment godoc
// @Summary      Remove a teacher-subject assignment
// @Tags         Teacher Subject Assignments
// @Produce      json
// @Param        id path string true "Assignment ID"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /academic/teacher-subject-assignments/{id} [delete]
func (h *ExamHandler) DeleteTeacherSubjectAssignment(c *gin.Context) {
	if err := h.examService.DeleteTeacherSubjectAssignment(c.Request.Context(), middleware.GetSchoolID(c), middleware.IsPlatformOwner(c), c.Param("id")); err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetMySubjectAssignments godoc
// @Summary      Get the authenticated teacher's own subject/class assignments
// @Tags         Teacher Subject Assignments
// @Produce      json
// @Param        sessionId query string false "Narrow to one session"
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /teacher/my-subjects [get]
func (h *ExamHandler) GetMySubjectAssignments(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.examService.GetMySubjectAssignments(c.Request.Context(), userID, c.Query("sessionId"))
	if err != nil {
		c.JSON(statusForSettingsError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
