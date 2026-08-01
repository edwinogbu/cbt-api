package handler

import (
    "net/http"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type SessionHandler struct {
    service *service.SessionService
}

func NewSessionHandler(service *service.SessionService) *SessionHandler {
    return &SessionHandler{service: service}
}

// CreateSession godoc
// @Summary      Create a new academic session
// @Description  Add an academic session (e.g., "2025/2026") for a school.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateSessionRequest true "Session details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions [post]
func (h *SessionHandler) CreateSession(c *gin.Context) {
    var req dto.CreateSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    session, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Session created successfully",
        "data":    session,
    })
}

// GetSession godoc
// @Summary      Get session by ID
// @Description  Retrieve a single academic session.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/{id} [get]
func (h *SessionHandler) GetSession(c *gin.Context) {
    id := c.Param("id")
    
    session, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": session})
}

// GetSessions godoc
// @Summary      Get all sessions for a school
// @Description  List all academic sessions belonging to a school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-sessions/{schoolId} [get]
func (h *SessionHandler) GetSessions(c *gin.Context) {
    schoolID := c.Param("schoolId")
    
    sessions, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": sessions})
}

// GetCurrentSession godoc
// @Summary      Get current session for a school
// @Description  Retrieve the currently active academic session.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-sessions/{schoolId}/current [get]
func (h *SessionHandler) GetCurrentSession(c *gin.Context) {
    schoolID := c.Param("schoolId")
    
    session, err := h.service.GetCurrent(schoolID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": session})
}

// UpdateSession godoc
// @Summary      Update a session
// @Description  Modify an existing academic session.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Session ID"
// @Param        request body dto.UpdateSessionRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/{id} [put]
func (h *SessionHandler) UpdateSession(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    session, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Session updated successfully",
        "data":    session,
    })
}

// DeleteSession godoc
// @Summary      Delete a session
// @Description  Soft‑delete an academic session.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/{id} [delete]
func (h *SessionHandler) DeleteSession(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

// internal/academic/handler/session_handler.go

// Add these new handler methods

// ListSessions godoc
// @Summary      List all sessions with pagination and filters
// @Description  Get a paginated list of sessions with filtering and sorting
// @Tags         Academic
// @Produce      json
// @Param        page query int false "Page number (default: 1)"
// @Param        limit query int false "Items per page (default: 20, max: 100)"
// @Param        search query string false "Search by name"
// @Param        school_id query string false "Filter by school ID"
// @Param        status query string false "Filter by status (active, upcoming, ended, inactive, all)" Enums(active, upcoming, ended, inactive, all)
// @Param        sort_by query string false "Sort by field (name, start_date, end_date, created_at)"
// @Param        sort_order query string false "Sort order (asc, desc)"
// @Param        start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param        end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Param        is_active query boolean false "Filter by active status"
// @Success      200  {object}  dto.SessionListResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions [get]
func (h *SessionHandler) ListSessions(c *gin.Context) {
    var req dto.ListSessionsRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    response, err := h.service.ListSessions(&req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   response,
    })
}

// GetSessionStats godoc
// @Summary      Get session statistics
// @Description  Get statistics for sessions (total, active, upcoming, ended, inactive)
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "School ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/stats [get]
func (h *SessionHandler) GetSessionStats(c *gin.Context) {
    schoolID := c.Query("school_id")

    stats, err := h.service.GetSessionStats(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   stats,
    })
}

// BulkDeleteSessions godoc
// @Summary      Bulk delete sessions
// @Description  Delete multiple sessions by IDs
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkDeleteRequest true "Session IDs to delete"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/bulk [delete]
func (h *SessionHandler) BulkDeleteSessions(c *gin.Context) {
    var req dto.BulkDeleteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if len(req.IDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "no session IDs provided"})
        return
    }

    count, err := h.service.BulkDeleteSessions(req.IDs)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Sessions deleted successfully",
        "data": gin.H{
            "deleted_count": count,
        },
    })
}

// GetSessionTimeline godoc
// @Summary      Get session timeline
// @Description  Get all sessions ordered by start date for timeline visualization
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "School ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/timeline [get]
func (h *SessionHandler) GetSessionTimeline(c *gin.Context) {
    schoolID := c.Query("school_id")

    timeline, err := h.service.GetSessionTimeline(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   timeline,
    })
}

// GetSessionSummary godoc
// @Summary      Get session summary for dashboard
// @Description  Get a summary of sessions including current, stats, and timeline
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "School ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/summary [get]
func (h *SessionHandler) GetSessionSummary(c *gin.Context) {
    schoolID := c.Query("school_id")

    summary, err := h.service.GetSessionSummary(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   summary,
    })
}

// SearchSessions godoc
// @Summary      Search sessions
// @Description  Search sessions by name with pagination
// @Tags         Academic
// @Produce      json
// @Param        q query string true "Search query"
// @Param        page query int false "Page number (default: 1)"
// @Param        limit query int false "Items per page (default: 20)"
// @Param        school_id query string false "School ID"
// @Success      200  {object}  dto.SessionListResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /sessions/search [get]
func (h *SessionHandler) SearchSessions(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
        return
    }

    req := &dto.ListSessionsRequest{
        Page:     1,
        Limit:    20,
        Search:   query,
        SchoolID: c.Query("school_id"),
    }

    if page := c.Query("page"); page != "" {
        // Parse page
    }
    if limit := c.Query("limit"); limit != "" {
        // Parse limit
    }

    response, err := h.service.ListSessions(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   response,
    })
}


// package handler

// import (
//     "net/http"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type SessionHandler struct {
//     service *service.SessionService
// }

// func NewSessionHandler(service *service.SessionService) *SessionHandler {
//     return &SessionHandler{service: service}
// }

// // CreateSession creates a new academic session
// func (h *SessionHandler) CreateSession(c *gin.Context) {
//     var req dto.CreateSessionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     session, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Session created successfully",
//         "data":    session,
//     })
// }

// // GetSession gets a session by ID
// func (h *SessionHandler) GetSession(c *gin.Context) {
//     id := c.Param("id")
    
//     session, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": session})
// }

// // GetSessions gets all sessions for a school
// func (h *SessionHandler) GetSessions(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     sessions, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": sessions})
// }

// // GetCurrentSession gets the current session for a school
// func (h *SessionHandler) GetCurrentSession(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     session, err := h.service.GetCurrent(schoolID)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": session})
// }

// // UpdateSession updates a session
// func (h *SessionHandler) UpdateSession(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateSessionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     session, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Session updated successfully",
//         "data":    session,
//     })
// }

// // DeleteSession deletes a session
// func (h *SessionHandler) DeleteSession(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
// }