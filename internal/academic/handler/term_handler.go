package handler

import (
    "net/http"
    "strconv"  // ✅ ADD THIS - fixes undefined: strconv
    
    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type TermHandler struct {
    service *service.TermService  // ✅ Field is 'service'
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
    
    term, err := h.service.Create(&req)  // ✅ Use 'service'
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
    
    term, err := h.service.GetByID(id)  // ✅ Use 'service'
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
    
    terms, err := h.service.GetBySession(sessionID)  // ✅ Use 'service'
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
    
    term, err := h.service.GetCurrent(sessionID)  // ✅ Use 'service'
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
    
    term, err := h.service.Update(id, &req)  // ✅ Use 'service'
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
    
    if err := h.service.Delete(id); err != nil {  // ✅ Use 'service'
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
}

// ============================================
// NEW LISTING ENDPOINTS (3 NEW)
// ============================================

// ListAllTerms godoc
// @Summary      List all terms with pagination
// @Tags         Terms
// @Produce      json
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200  {object}  dto.TermListResponse
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/list [get]
func (h *TermHandler) ListAllTerms(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))    // ✅ strconv now works
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))  // ✅ strconv now works

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    ctx := c.Request.Context()
    resp, err := h.service.ListAllTerms(ctx, page, limit)  // ✅ Use 'service'
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to list terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// ListAllTermsBySchool godoc
// @Summary      List all terms for a school
// @Tags         Terms
// @Produce      json
// @Param        school_id query string true "School UUID"
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200  {object}  dto.TermListResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/school [get]
func (h *TermHandler) ListAllTermsBySchool(c *gin.Context) {
    schoolID := c.Query("school_id")
    if schoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
        return
    }

    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))    // ✅ strconv now works
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))  // ✅ strconv now works

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    ctx := c.Request.Context()
    resp, err := h.service.ListAllTermsBySchool(ctx, schoolID, page, limit)  // ✅ Use 'service'
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to list terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// GetAllTerms godoc
// @Summary      Get all terms without pagination (for dropdowns)
// @Tags         Terms
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/all [get]
func (h *TermHandler) GetAllTerms(c *gin.Context) {
    ctx := c.Request.Context()
    terms, err := h.service.GetAllTerms(ctx)  // ✅ Use 'service'
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to get all terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   terms,
        "count":  len(terms),
    })
}

// ============================================================
// NEW HANDLER METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListTerms godoc
// @Summary      List all terms with pagination, filtering, sorting
// @Description  Get paginated list of terms with optional filters.
// @Description  ALL parameters are optional - returns all terms if no params.
// @Tags         Academic
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20) max(100)
// @Param        search query string false "Search by name or term number"
// @Param        session_id query string false "Filter by session ID"
// @Param        school_id query string false "Filter by school ID"
// @Param        is_active query bool false "Filter by active status"
// @Param        is_current query bool false "Filter by current status"
// @Param        sort_by query string false "Sort by field (name, term_number, created_at)" default(created_at)
// @Param        sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success      200  {object}  dto.TermListResponse
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms [get]
func (h *TermHandler) ListTerms(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    req := &dto.ListTermsRequest{
        Page:      page,
        Limit:     limit,
        Search:    c.Query("search"),
        SessionID: c.Query("session_id"),
        SchoolID:  c.Query("school_id"),
        SortBy:    c.DefaultQuery("sort_by", "terms.created_at"),
        SortOrder: c.DefaultQuery("sort_order", "DESC"),
        IsActive:  nil,
        IsCurrent: nil,
    }

    if isActive := c.Query("is_active"); isActive != "" {
        val := isActive == "true"
        req.IsActive = &val
    }
    if isCurrent := c.Query("is_current"); isCurrent != "" {
        val := isCurrent == "true"
        req.IsCurrent = &val
    }

    ctx := c.Request.Context()
    resp, err := h.service.ListTerms(ctx, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to list terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// BulkDeleteTerms godoc
// @Summary      Bulk delete terms
// @Description  Delete multiple terms by IDs
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.TermBulkDeleteRequest true "List of IDs to delete"
// @Success      200  {object}  map[string]interface{}  "message + deleted_count"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/bulk [delete]
func (h *TermHandler) BulkDeleteTerms(c *gin.Context) {
    var req dto.TermBulkDeleteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Invalid request payload",
            "details": err.Error(),
        })
        return
    }

    if len(req.IDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "No IDs provided for bulk delete",
        })
        return
    }

    ctx := c.Request.Context()
    deletedCount, err := h.service.BulkDeleteTerms(ctx, req.IDs)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to delete terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":        "success",
        "message":       "Terms deleted successfully",
        "deleted_count": deletedCount,
    })
}

// GetTermStats godoc
// @Summary      Get term statistics
// @Description  Get statistics about terms (total, active, by session, etc.)
// @Description  Optional school_id filter - if not provided, returns stats for all schools
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "Filter by school ID"
// @Success      200  {object}  map[string]interface{}  "data (stats)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/stats [get]
func (h *TermHandler) GetTermStats(c *gin.Context) {
    schoolID := c.Query("school_id")
    ctx := c.Request.Context()

    stats, err := h.service.GetTermStats(ctx, schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch term statistics",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Statistics retrieved successfully",
        "data":    stats,
    })
}

// SearchTerms godoc
// @Summary      Search terms
// @Description  Search terms by name or term number
// @Tags         Academic
// @Produce      json
// @Param        q query string true "Search query"
// @Param        school_id query string false "Filter by school ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200  {object}  dto.TermListResponse
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /terms/search [get]
func (h *TermHandler) SearchTerms(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Search query is required",
        })
        return
    }

    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    schoolID := c.Query("school_id")
    ctx := c.Request.Context()

    resp, err := h.service.SearchTerms(ctx, query, schoolID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to search terms",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Search completed successfully",
        "data":    resp,
    })
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

// // CreateTerm godoc
// // @Summary      Create a new term
// // @Description  Add a term (e.g., "First Term") for an academic session.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateTermRequest true "Term details"
// // @Success      201  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms [post]
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

// // GetTerm godoc
// // @Summary      Get term by ID
// // @Description  Retrieve a single term.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Term ID"
// // @Success      200  {object}  map[string]interface{}  "data"
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms/{id} [get]
// func (h *TermHandler) GetTerm(c *gin.Context) {
//     id := c.Param("id")
    
//     term, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": term})
// }

// // GetTerms godoc
// // @Summary      Get all terms for a session
// // @Description  List terms belonging to an academic session.
// // @Tags         Academic
// // @Produce      json
// // @Param        sessionId path string true "Session ID"
// // @Success      200  {object}  map[string]interface{}  "data (list)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /session-terms/{sessionId} [get]
// func (h *TermHandler) GetTerms(c *gin.Context) {
//     sessionID := c.Param("sessionId")
    
//     terms, err := h.service.GetBySession(sessionID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": terms})
// }

// // GetCurrentTerm godoc
// // @Summary      Get current term for a session
// // @Description  Retrieve the currently active term within a session.
// // @Tags         Academic
// // @Produce      json
// // @Param        sessionId path string true "Session ID"
// // @Success      200  {object}  map[string]interface{}  "data"
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /session-terms/{sessionId}/current [get]
// func (h *TermHandler) GetCurrentTerm(c *gin.Context) {
//     sessionID := c.Param("sessionId")
    
//     term, err := h.service.GetCurrent(sessionID)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": term})
// }

// // UpdateTerm godoc
// // @Summary      Update a term
// // @Description  Modify an existing term.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Term ID"
// // @Param        request body dto.UpdateTermRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms/{id} [put]
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

// // DeleteTerm godoc
// // @Summary      Delete a term
// // @Description  Soft‑delete a term.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Term ID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms/{id} [delete]
// func (h *TermHandler) DeleteTerm(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
// }



// // ListAllTerms godoc
// // @Summary      List all terms
// // @Tags         Terms
// // @Produce      json
// // @Param        page query int false "Page number (default 1)"
// // @Param        limit query int false "Items per page (default 20, max 100)"
// // @Success      200  {object}  dto.TermListResponse
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms/list [get]
// func (h *TermHandler) ListAllTerms(c *gin.Context) {
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     ctx := c.Request.Context()
//     resp, err := h.termService.ListAllTerms(ctx, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "error":   "failed to list terms",
//             "details": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   resp,
//     })
// }

// // ListAllTermsBySchool godoc
// // @Summary      List all terms for a school
// // @Tags         Terms
// // @Produce      json
// // @Param        school_id query string true "School UUID"
// // @Param        page query int false "Page number (default 1)"
// // @Param        limit query int false "Items per page (default 20, max 100)"
// // @Success      200  {object}  dto.TermListResponse
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      401  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /terms/school [get]
// func (h *TermHandler) ListAllTermsBySchool(c *gin.Context) {
//     schoolID := c.Query("school_id")
//     if schoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
//         return
//     }

//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     ctx := c.Request.Context()
//     resp, err := h.termService.ListAllTermsBySchool(ctx, schoolID, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "error":   "failed to list terms",
//             "details": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   resp,
//     })
// }






// // package handler

// // import (
// //     "net/http"

// //     "cbt-api/internal/academic/dto"
// //     "cbt-api/internal/academic/service"

// //     "github.com/gin-gonic/gin"
// // )

// // type TermHandler struct {
// //     service *service.TermService
// // }

// // func NewTermHandler(service *service.TermService) *TermHandler {
// //     return &TermHandler{service: service}
// // }

// // // CreateTerm creates a new term
// // func (h *TermHandler) CreateTerm(c *gin.Context) {
// //     var req dto.CreateTermRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     term, err := h.service.Create(&req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusCreated, gin.H{
// //         "message": "Term created successfully",
// //         "data":    term,
// //     })
// // }

// // // GetTerm gets a term by ID
// // func (h *TermHandler) GetTerm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     term, err := h.service.GetByID(id)
// //     if err != nil {
// //         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"data": term})
// // }

// // // GetTerms gets all terms for a session
// // func (h *TermHandler) GetTerms(c *gin.Context) {
// //     sessionID := c.Param("sessionId")
    
// //     terms, err := h.service.GetBySession(sessionID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"data": terms})
// // }

// // // GetCurrentTerm gets the current term for a session
// // func (h *TermHandler) GetCurrentTerm(c *gin.Context) {
// //     sessionID := c.Param("sessionId")
    
// //     term, err := h.service.GetCurrent(sessionID)
// //     if err != nil {
// //         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"data": term})
// // }

// // // UpdateTerm updates a term
// // func (h *TermHandler) UpdateTerm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     var req dto.UpdateTermRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     term, err := h.service.Update(id, &req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{
// //         "message": "Term updated successfully",
// //         "data":    term,
// //     })
// // }

// // // DeleteTerm deletes a term
// // func (h *TermHandler) DeleteTerm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     if err := h.service.Delete(id); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
// // }