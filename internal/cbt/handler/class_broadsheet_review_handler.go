package handler

import (
	"net/http"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// This file is the Class Teacher Broad Sheet's own actionable workflow
// (template B, page 4) - save/submit/approve/return on the class
// teacher's written summary. The read path is folded into
// GetClassBroadSheet's own response (ClassBroadSheetResponse.Review),
// not a separate GET endpoint, since a caller who loads the broad
// sheet always wants its current review state in the same call.

func actorOrUnauthorized(c *gin.Context) (string, bool) {
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	return actorID, true
}

// SaveClassBroadSheetReview godoc
// @Summary      Save (without submitting) the class teacher's broad-sheet summary
// @Tags         Broad Sheets
// @Accept       json
// @Produce      json
// @Param        request body dto.SaveClassBroadSheetReviewRequest true "Summary to save"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class/review/save [post]
func (h *ExamHandler) SaveClassBroadSheetReview(c *gin.Context) {
	var req dto.SaveClassBroadSheetReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := actorOrUnauthorized(c)
	if !ok {
		return
	}
	resp, err := h.examService.SaveClassBroadSheetReviewSummary(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// SubmitClassBroadSheetReview godoc
// @Summary      Submit the class teacher's broad-sheet summary for review
// @Tags         Broad Sheets
// @Accept       json
// @Produce      json
// @Param        request body dto.ClassBroadSheetReviewActionRequest true "Scope"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class/review/submit [post]
func (h *ExamHandler) SubmitClassBroadSheetReview(c *gin.Context) {
	var req dto.ClassBroadSheetReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := actorOrUnauthorized(c)
	if !ok {
		return
	}
	resp, err := h.examService.SubmitClassBroadSheetReview(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ApproveClassBroadSheetReview godoc
// @Summary      Approve a submitted class broad-sheet review
// @Tags         Broad Sheets
// @Accept       json
// @Produce      json
// @Param        request body dto.ClassBroadSheetReviewActionRequest true "Scope"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class/review/approve [post]
func (h *ExamHandler) ApproveClassBroadSheetReview(c *gin.Context) {
	var req dto.ClassBroadSheetReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := actorOrUnauthorized(c)
	if !ok {
		return
	}
	resp, err := h.examService.ApproveClassBroadSheetReview(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// ReturnClassBroadSheetReview godoc
// @Summary      Return a submitted class broad-sheet review to the class teacher
// @Tags         Broad Sheets
// @Accept       json
// @Produce      json
// @Param        request body dto.ClassBroadSheetReviewActionRequest true "Scope + return reason in notes"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/broadsheet/class/review/return [post]
func (h *ExamHandler) ReturnClassBroadSheetReview(c *gin.Context) {
	var req dto.ClassBroadSheetReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := actorOrUnauthorized(c)
	if !ok {
		return
	}
	resp, err := h.examService.ReturnClassBroadSheetReview(c.Request.Context(), middleware.GetSchoolID(c), actorID, middleware.IsPlatformOwner(c), &req)
	if err != nil {
		c.JSON(statusForApprovalError(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}
