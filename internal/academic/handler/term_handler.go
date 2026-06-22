package handler

import (
    "net/http"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type TermHandler struct {
    service *service.TermService
}

func NewTermHandler(service *service.TermService) *TermHandler {
    return &TermHandler{service: service}
}

// CreateTerm godoc
// @Summary      Create a new term
// @Description  Add a term (e.g., "First Term") for an academic session.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateTermRequest true "Term details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms [post]
func (h *TermHandler) CreateTerm(c *gin.Context) {
    var req dto.CreateTermRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    term, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Term created successfully",
        "data":    term,
    })
}

// GetTerm godoc
// @Summary      Get term by ID
// @Description  Retrieve a single term.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Term ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/{id} [get]
func (h *TermHandler) GetTerm(c *gin.Context) {
    id := c.Param("id")
    
    term, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": term})
}

// GetTerms godoc
// @Summary      Get all terms for a session
// @Description  List terms belonging to an academic session.
// @Tags         Academic
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /session-terms/{sessionId} [get]
func (h *TermHandler) GetTerms(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    terms, err := h.service.GetBySession(sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": terms})
}

// GetCurrentTerm godoc
// @Summary      Get current term for a session
// @Description  Retrieve the currently active term within a session.
// @Tags         Academic
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /session-terms/{sessionId}/current [get]
func (h *TermHandler) GetCurrentTerm(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    term, err := h.service.GetCurrent(sessionID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": term})
}

// UpdateTerm godoc
// @Summary      Update a term
// @Description  Modify an existing term.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Term ID"
// @Param        request body dto.UpdateTermRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/{id} [put]
func (h *TermHandler) UpdateTerm(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateTermRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    term, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Term updated successfully",
        "data":    term,
    })
}

// DeleteTerm godoc
// @Summary      Delete a term
// @Description  Soft‑delete a term.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Term ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/{id} [delete]
func (h *TermHandler) DeleteTerm(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
}




// package handler

// import (
//     "net/http"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type TermHandler struct {
//     service *service.TermService
// }

// func NewTermHandler(service *service.TermService) *TermHandler {
//     return &TermHandler{service: service}
// }

// // CreateTerm creates a new term
// func (h *TermHandler) CreateTerm(c *gin.Context) {
//     var req dto.CreateTermRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     term, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Term created successfully",
//         "data":    term,
//     })
// }

// // GetTerm gets a term by ID
// func (h *TermHandler) GetTerm(c *gin.Context) {
//     id := c.Param("id")
    
//     term, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": term})
// }

// // GetTerms gets all terms for a session
// func (h *TermHandler) GetTerms(c *gin.Context) {
//     sessionID := c.Param("sessionId")
    
//     terms, err := h.service.GetBySession(sessionID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": terms})
// }

// // GetCurrentTerm gets the current term for a session
// func (h *TermHandler) GetCurrentTerm(c *gin.Context) {
//     sessionID := c.Param("sessionId")
    
//     term, err := h.service.GetCurrent(sessionID)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": term})
// }

// // UpdateTerm updates a term
// func (h *TermHandler) UpdateTerm(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateTermRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     term, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Term updated successfully",
//         "data":    term,
//     })
// }

// // DeleteTerm deletes a term
// func (h *TermHandler) DeleteTerm(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
// }