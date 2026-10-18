package handler

import (
	"net/http"

	"cbt-api/internal/middleware"
	syncDto "cbt-api/internal/sync/dto"
	"cbt-api/internal/sync/service"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	service *service.SyncService
}

func NewSyncHandler(service *service.SyncService) *SyncHandler {
	return &SyncHandler{service: service}
}

// Sync godoc
// @Summary      Sync an offline-queued record (answer, submission, violation, or event)
// @Tags         Offline Sync
// @Accept       json
// @Produce      json
// @Param        type path string true "Sync type" Enums(answer, submission, violation, event)
// @Param        request body dto.SyncRequest true "Sync request"
// @Success      200 {object} dto.SyncSuccessResponse
// @Failure      400 {object} dto.SyncErrorResponse
// @Failure      403 {object} dto.SyncErrorResponse
// @Failure      404 {object} dto.SyncErrorResponse
// @Failure      409 {object} dto.SyncConflictResponse
// @Security     BearerAuth
// @Router       /sync/{type} [post]
func (h *SyncHandler) Sync(c *gin.Context) {
	pathType := c.Param("type")

	var req syncDto.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, syncDto.SyncErrorResponse{
			ErrorCode: "VALIDATION_ERROR",
			Message:   err.Error(),
			Retryable: false,
		})
		return
	}

	// The URL path segment and the body's own `type` field must agree -
	// this is a deliberate double-check against a client bug or a stale
	// queue item sending its payload to the wrong route.
	if pathType != "" && pathType != req.Type {
		c.JSON(http.StatusBadRequest, syncDto.SyncErrorResponse{
			ErrorCode: "TYPE_MISMATCH",
			Message:   "URL type does not match request body type",
			Retryable: false,
		})
		return
	}

	studentID := middleware.GetStudentID(c)
	if studentID == "" {
		c.JSON(http.StatusUnauthorized, syncDto.SyncErrorResponse{
			ErrorCode: "UNAUTHORIZED",
			Message:   "student context not found",
			Retryable: false,
		})
		return
	}

	result := h.service.Process(c.Request.Context(), studentID, &req)

	switch result.Outcome {
	case syncDto.OutcomeAccepted, syncDto.OutcomeAlreadyProcessed:
		c.JSON(http.StatusOK, syncDto.SyncSuccessResponse{
			AlreadyProcessed: result.Outcome == syncDto.OutcomeAlreadyProcessed,
			ServerVersion:    result.ServerVersion,
			ServerSequence:   result.ServerSequence,
			ServerTimestamp:  result.ServerTimestamp,
			Result:           result.Result,
		})
	case syncDto.OutcomeConflict:
		c.JSON(http.StatusConflict, syncDto.SyncConflictResponse{
			ServerVersion:   result.ServerVersion,
			ServerRecord:    result.ServerRecord,
			ServerTimestamp: result.ServerTimestamp,
		})
	default: // REJECTED
		status := result.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}
		c.JSON(status, syncDto.SyncErrorResponse{
			ErrorCode: result.ErrorCode,
			Message:   result.ErrorMessage,
			Retryable: result.Retryable,
		})
	}
}
