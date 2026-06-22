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