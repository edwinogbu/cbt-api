package handler

import (
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SubjectHandler struct {
	service *service.SubjectService
}

func NewSubjectHandler(s *service.SubjectService) *SubjectHandler {
	return &SubjectHandler{service: s}
}

// CreateSubject godoc
// @Summary      Create a new subject
// @Tags         Subjects
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateSubjectRequest true "Subject details"
// @Success      201 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /subjects [post]
func (h *SubjectHandler) CreateSubject(c *gin.Context) {
	var req dto.CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.CreateSubject(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// GetSubject godoc
// @Summary      Get a single subject
// @Tags         Subjects
// @Produce      json
// @Param        id path string true "Subject ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /subjects/{id} [get]
func (h *SubjectHandler) GetSubject(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.service.GetSubject(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// UpdateSubject godoc
// @Summary      Update a subject
// @Tags         Subjects
// @Accept       json
// @Produce      json
// @Param        id path string true "Subject ID"
// @Param        request body dto.UpdateSubjectRequest true "Fields to update"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /subjects/{id} [put]
func (h *SubjectHandler) UpdateSubject(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.UpdateSubject(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// DeleteSubject godoc
// @Summary      Delete a subject (soft delete)
// @Tags         Subjects
// @Produce      json
// @Param        id path string true "Subject ID"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /subjects/{id} [delete]
func (h *SubjectHandler) DeleteSubject(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteSubject(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subject deleted successfully"})
}

// ListSubjects godoc
// @Summary      List all subjects (paginated)
// @Tags         Subjects
// @Produce      json
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200 {object} map[string]interface{} "data, total, page, limit"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /subjects [get]
func (h *SubjectHandler) ListSubjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	subjects, total, err := h.service.ListSubjects(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  subjects,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ListActiveSubjects godoc
// @Summary      List only active subjects
// @Tags         Subjects
// @Produce      json
// @Success      200 {object} map[string]interface{} "data"
// @Security     BearerAuth
// @Router       /subjects/active [get]
func (h *SubjectHandler) ListActiveSubjects(c *gin.Context) {
	subjects, err := h.service.ListActiveSubjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": subjects})
}


