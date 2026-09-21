package handler

import (
	"net/http"
	"time"

	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/middleware"
	"cbt-api/internal/nodesync/service"

	"github.com/gin-gonic/gin"
)

type NodeSyncHandler struct {
	svc *service.NodeSyncService
}

func NewNodeSyncHandler(svc *service.NodeSyncService) *NodeSyncHandler {
	return &NodeSyncHandler{svc: svc}
}

// Push godoc
// @Summary      School Node pushes accumulated sessions/answers/results/events to Cloud
// @Tags         School Node Sync
// @Accept       json
// @Produce      json
// @Security     NodeAuth
// @Router       /nodesync/push [post]
func (h *NodeSyncHandler) Push(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	if schoolID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "node not authenticated"})
		return
	}

	var req dto.PushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Push(c.Request.Context(), schoolID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "push failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// Pull godoc
// @Summary      School Node pulls students/exams changed since a cursor from Cloud
// @Tags         School Node Sync
// @Produce      json
// @Param        since query string false "RFC3339 cursor; omit or empty for full sync"
// @Security     NodeAuth
// @Router       /nodesync/pull [get]
func (h *NodeSyncHandler) Pull(c *gin.Context) {
	schoolID := middleware.GetSchoolID(c)
	if schoolID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "node not authenticated"})
		return
	}

	since := time.Time{} // zero value = full sync
	if raw := c.Query("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "since must be RFC3339"})
			return
		}
		since = parsed
	}

	resp, err := h.svc.Pull(c.Request.Context(), schoolID, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pull failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": resp})
}

// CreateCredential godoc
// @Summary      Admin: issue a new node credential for a school (shown once)
// @Tags         School Node Sync
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Router       /admin/school-nodes/credentials [post]
func (h *NodeSyncHandler) CreateCredential(c *gin.Context) {
	var req struct {
		SchoolID string `json:"school_id" binding:"required,uuid"`
		Label    string `json:"label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.svc.CreateCredential(c.Request.Context(), req.SchoolID, req.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create credential"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data": gin.H{
			"api_key": key,
			"warning": "This key is shown only once. Store it in this node's .env.school-node as NODE_API_KEY and discard this response.",
		},
	})
}
