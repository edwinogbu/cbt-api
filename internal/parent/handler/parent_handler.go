package handler

import (
	"net/http"

	"cbt-api/internal/middleware"
	"cbt-api/internal/parent/service"
	"github.com/gin-gonic/gin"
)

type ParentHandler struct {
	service *service.ParentService
}

func NewParentHandler(svc *service.ParentService) *ParentHandler {
	return &ParentHandler{service: svc}
}

// GetChildren godoc
// @Summary      Get all children linked to the logged‑in parent
// @Tags         Parent
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /parent/children [get]
func (h *ParentHandler) GetChildren(c *gin.Context) {
	parentID := middleware.GetUserID(c)
	children, err := h.service.GetLinkedChildren(parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": children})
}

// GetChildResults godoc
// @Summary      Get exam results for a specific child (must be linked to parent)
// @Tags         Parent
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /parent/child/{studentId}/results [get]
func (h *ParentHandler) GetChildResults(c *gin.Context) {
	parentID := middleware.GetUserID(c)
	studentID := c.Param("studentId")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}
	results, err := h.service.GetChildResults(parentID, studentID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "you are not linked to this student" {
			status = http.StatusForbidden
		} else if err.Error() == "student not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}