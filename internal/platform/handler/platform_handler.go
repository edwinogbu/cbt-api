package handler

import (
	"net/http"

	"cbt-api/internal/middleware"
	"cbt-api/internal/platform/dto"
	"cbt-api/internal/platform/service"

	"github.com/gin-gonic/gin"
)

// PlatformHandler exposes the platform_owner-only tenant registry,
// branding, audit log, and LAN Setup Dashboard endpoints (Master
// Implementation Plan Phases 3-4, 10, 11). Every route this handler
// backs is gated at the route layer with middleware.AuthMiddleware()
// followed by middleware.PlatformOwnerOnly() - see
// api/routes/routes.go's setupPlatformRoutes - so no handler method
// here re-checks the role; it only reads the already-verified actor
// user ID off the JWT context for audit attribution.
type PlatformHandler struct {
	service *service.PlatformService
}

func NewPlatformHandler(service *service.PlatformService) *PlatformHandler {
	return &PlatformHandler{service: service}
}

// ============================================
// TENANT REGISTRY
// ============================================

func (h *PlatformHandler) ListTenants(c *gin.Context) {
	var q dto.TenantListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.ListTenants(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tenants"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PlatformHandler) GetTenant(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.service.GetTenant(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *PlatformHandler) UpdateBranding(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.service.UpdateBranding(actorID, id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "branding updated"})
}

func (h *PlatformHandler) ProvisionTenant(c *gin.Context) {
	id := c.Param("id")
	var req dto.ProvisionTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.service.ProvisionTenant(actorID, id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant provisioned"})
}

func (h *PlatformHandler) SuspendTenant(c *gin.Context) {
	id := c.Param("id")
	var req dto.SuspendTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.service.SuspendTenant(actorID, id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant suspended"})
}

// ============================================
// AUDIT LOG
// ============================================

func (h *PlatformHandler) ListAuditLogs(c *gin.Context) {
	var q dto.AuditLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.ListAuditLogs(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// LAN SERVER CONFIG
// ============================================

func (h *PlatformHandler) GetLANConfig(c *gin.Context) {
	schoolID := c.Param("schoolId")
	resp, err := h.service.GetLANConfig(schoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PlatformHandler) UpdateLANConfig(c *gin.Context) {
	schoolID := c.Param("schoolId")
	var req dto.UpdateLANConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.service.UpdateLANConfig(actorID, schoolID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PlatformHandler) SetLANConfigActive(c *gin.Context) {
	schoolID := c.Param("schoolId")
	var body struct {
		Active bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.service.SetLANConfigActive(actorID, schoolID, body.Active)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PlatformHandler) TestLANConnectivity(c *gin.Context) {
	schoolID := c.Param("schoolId")
	actorID := middleware.GetUserID(c)
	resp, err := h.service.TestLANConnectivity(actorID, schoolID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PlatformHandler) ListLANConfigHistory(c *gin.Context) {
	schoolID := c.Param("schoolId")
	history, err := h.service.ListLANConfigHistory(schoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}

func (h *PlatformHandler) RollbackLANConfig(c *gin.Context) {
	schoolID := c.Param("schoolId")
	var body struct {
		HistoryID string `json:"history_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.GetUserID(c)
	resp, err := h.service.RollbackLANConfig(actorID, schoolID, body.HistoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
