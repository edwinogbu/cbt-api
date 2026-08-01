package handler

import (
    "net/http"
    "strconv"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type ClassLevelHandler struct {
    service *service.ClassLevelService
}

func NewClassLevelHandler(service *service.ClassLevelService) *ClassLevelHandler {
    return &ClassLevelHandler{service: service}
}

// CreateClassLevel godoc
// @Summary      Create a new class level
// @Description  Add a class level (e.g., "Grade 10") for a school.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateClassLevelRequest true "Class level details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels [post]
func (h *ClassLevelHandler) CreateClassLevel(c *gin.Context) {
    var req dto.CreateClassLevelRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Invalid request payload",
            "details": err.Error(),
        })
        return
    }
    
    level, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to create class level",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Class level created successfully",
        "data":    level,
    })
}

// GetClassLevel godoc
// @Summary      Get class level by ID
// @Description  Retrieve a single class level.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Level ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/{id} [get]
func (h *ClassLevelHandler) GetClassLevel(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "ID is required",
        })
        return
    }
    
    level, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "Class level not found",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": level,
    })
}

// GetClassLevelsBySchool godoc
// @Summary      Get all class levels for a school
// @Description  List all class levels belonging to a school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/school/{schoolId} [get]
func (h *ClassLevelHandler) GetClassLevelsBySchool(c *gin.Context) {
    schoolID := c.Param("schoolId")
    if schoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "School ID is required",
        })
        return
    }
    
    levels, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch class levels",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": levels,
        "count": len(levels),
    })
}

// GetClassLevelsByCategory godoc
// @Summary      Get class levels by category
// @Description  List class levels by category (JSS, SSS, PRIMARY)
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Param        category path string true "Category (JSS, SSS, PRIMARY)"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/school/{schoolId}/category/{category} [get]
func (h *ClassLevelHandler) GetClassLevelsByCategory(c *gin.Context) {
    schoolID := c.Param("schoolId")
    category := c.Param("category")
    
    if schoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "School ID is required",
        })
        return
    }
    if category == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Category is required",
        })
        return
    }
    
    // Validate category
    validCategories := map[string]bool{"JSS": true, "SSS": true, "PRIMARY": true}
    if !validCategories[category] {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid category. Must be JSS, SSS, or PRIMARY",
        })
        return
    }
    
    levels, err := h.service.GetByCategory(schoolID, category)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch class levels by category",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data":     levels,
        "count":    len(levels),
        "category": category,
    })
}

// UpdateClassLevel godoc
// @Summary      Update a class level
// @Description  Modify an existing class level.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Class Level ID"
// @Param        request body dto.UpdateClassLevelRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/{id} [put]
func (h *ClassLevelHandler) UpdateClassLevel(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "ID is required",
        })
        return
    }
    
    var req dto.UpdateClassLevelRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Invalid request payload",
            "details": err.Error(),
        })
        return
    }
    
    level, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to update class level",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class level updated successfully",
        "data":    level,
    })
}

// // DeleteClassLevel godoc
// // @Summary      Delete a class level
// // @Description  Soft‑delete a class level.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [delete]
// func (h *ClassLevelHandler) DeleteClassLevel(c *gin.Context) {
//     id := c.Param("id")
//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "error": "ID is required",
//         })
//         return
//     }
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "error":   "Failed to delete class level",
//             "details": err.Error(),
//         })
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Class level deleted successfully",
//     })
// }

// DeleteClassLevel godoc
// @Summary      Delete a class level
// @Description  Soft‑delete a class level.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Level ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/{id} [delete]
func (h *ClassLevelHandler) DeleteClassLevel(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "ID is required",
        })
        return
    }
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to delete class level",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class level deleted successfully",
    })
}


// BulkDeleteClassLevels godoc
// @Summary      Bulk delete class levels
// @Description  Delete multiple class levels by IDs
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.ClassLevelBulkDeleteRequest true "List of IDs to delete"
// @Success      200  {object}  map[string]interface{}  "message + deleted_count"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/bulk [delete]
func (h *ClassLevelHandler) BulkDeleteClassLevels(c *gin.Context) {
    var req dto.ClassLevelBulkDeleteRequest
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
    
    deletedCount, err := h.service.BulkDeleteClassLevels(req.IDs)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to delete class levels",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":       "Class levels deleted successfully",
        "deleted_count": deletedCount,
    })
}

// ============================================================
// NEW METHODS FOR COMPLETE CRUD (ALL PARAMETERS OPTIONAL)
// ============================================================

// ListClassLevels godoc
// @Summary      List all class levels with pagination, filtering, sorting
// @Description  Get paginated list of class levels with optional filters and sorting.
// @Description  All parameters are optional - calling without any parameters returns all class levels.
// @Tags         Academic
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20) max(100)
// @Param        search query string false "Search by name or category"
// @Param        school_id query string false "Filter by school ID"
// @Param        category query string false "Filter by category (JSS, SSS, PRIMARY)"
// @Param        is_active query bool false "Filter by active status"
// @Param        sort_by query string false "Sort by field (name, level_number, category, sort_order, created_at)" default(level_number)
// @Param        sort_order query string false "Sort order (asc, desc)" default(asc)
// @Success      200  {object}  map[string]interface{}  "data (paginated list) + pagination meta"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels [get]
func (h *ClassLevelHandler) ListClassLevels(c *gin.Context) {
    // Parse query parameters with defaults (ALL OPTIONAL)
    page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
    if err != nil || page < 1 {
        page = 1
    }
    
    limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
    if err != nil || limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }
    
    req := &dto.ListClassLevelsRequest{
        Page:     page,
        Limit:    limit,
        Search:   c.Query("search"),
        SchoolID: c.Query("school_id"),
        Category: c.Query("category"),
        SortBy:   c.DefaultQuery("sort_by", "level_number"),
        SortDir:  c.DefaultQuery("sort_order", "asc"),
        IsActive: nil,
    }
    
    // Parse is_active if provided
    if isActive := c.Query("is_active"); isActive != "" {
        val := isActive == "true"
        req.IsActive = &val
    }
    
    result, err := h.service.List(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch class levels",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class levels retrieved successfully",
        "data": result.Items,
        "pagination": gin.H{
            "page":        result.Page,
            "limit":       result.Limit,
            "total":       result.Total,
            "total_pages": result.TotalPages,
        },
    })
}


// // BulkDeleteClassLevels godoc
// // @Summary      Bulk delete class levels
// // @Description  Delete multiple class levels by IDs
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.ClassLevelBulkDeleteRequest true "List of IDs to delete"
// // @Success      200  {object}  map[string]interface{}  "message + deleted_count"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/bulk [delete]
// func (h *ClassLevelHandler) BulkDeleteClassLevels(c *gin.Context) {
//     var req dto.ClassLevelBulkDeleteRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "error":   "Invalid request payload",
//             "details": err.Error(),
//         })
//         return
//     }
    
//     if len(req.IDs) == 0 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "error": "No IDs provided for bulk delete",
//         })
//         return
//     }
    
//     deletedCount, err := h.service.BulkDelete(req.IDs)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "error":   "Failed to delete class levels",
//             "details": err.Error(),
//         })
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message":       "Class levels deleted successfully",
//         "deleted_count": deletedCount,
//     })
// }

// GetClassLevelStats godoc
// @Summary      Get class level statistics
// @Description  Get statistics about class levels (total, active, by category, etc.)
// @Description  Optional school_id filter - if not provided, returns stats for all schools
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "Filter by school ID"
// @Success      200  {object}  map[string]interface{}  "data (stats)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/stats [get]
func (h *ClassLevelHandler) GetClassLevelStats(c *gin.Context) {
    schoolID := c.Query("school_id")
    
    stats, err := h.service.GetStats(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch class level statistics",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Statistics retrieved successfully",
        "data":    stats,
    })
}

// SearchClassLevels godoc
// @Summary      Search class levels
// @Description  Search class levels by name or category
// @Tags         Academic
// @Produce      json
// @Param        q query string true "Search query"
// @Param        school_id query string false "Filter by school ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200  {object}  map[string]interface{}  "data (search results)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-levels/search [get]
func (h *ClassLevelHandler) SearchClassLevels(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Search query is required",
        })
        return
    }
    
    page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
    if err != nil || page < 1 {
        page = 1
    }
    
    limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
    if err != nil || limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }
    
    schoolID := c.Query("school_id")
    
    result, err := h.service.Search(schoolID, query, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to search class levels",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Search completed successfully",
        "data": result.Items,
        "pagination": gin.H{
            "page":        result.Page,
            "limit":       result.Limit,
            "total":       result.Total,
            "total_pages": result.TotalPages,
        },
    })
}



// package handler

// import (
//     "net/http"
//     "strconv"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type ClassLevelHandler struct {
//     service *service.ClassLevelService
// }

// func NewClassLevelHandler(service *service.ClassLevelService) *ClassLevelHandler {
//     return &ClassLevelHandler{service: service}
// }

// // CreateClassLevel godoc
// // @Summary      Create a new class level
// // @Description  Add a class level (e.g., "Grade 10") for a school.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateClassLevelRequest true "Class level details"
// // @Success      201  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels [post]
// func (h *ClassLevelHandler) CreateClassLevel(c *gin.Context) {
//     var req dto.CreateClassLevelRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request payload",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     // Validate required fields
//     if req.SchoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "school_id is required",
//         })
//         return
//     }
//     if req.Name == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "name is required",
//         })
//         return
//     }
//     if req.LevelNumber < 1 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "level_number must be greater than 0",
//         })
//         return
//     }

//     level, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusCreated, gin.H{
//         "status":  "success",
//         "message": "Class level created successfully",
//         "data":    level,
//     })
// }

// // GetClassLevel godoc
// // @Summary      Get class level by ID
// // @Description  Retrieve a single class level.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Success      200  {object}  map[string]interface{}  "data"
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [get]
// func (h *ClassLevelHandler) GetClassLevel(c *gin.Context) {
//     id := c.Param("id")
//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "ID is required",
//         })
//         return
//     }

//     level, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   level,
//     })
// }

// // GetClassLevelsBySchool godoc
// // @Summary      Get all class levels for a school
// // @Description  List all class levels belonging to a school.
// // @Tags         Academic
// // @Produce      json
// // @Param        schoolId path string true "School ID"
// // @Success      200  {object}  map[string]interface{}  "data (list)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/school/{schoolId} [get]
// func (h *ClassLevelHandler) GetClassLevelsBySchool(c *gin.Context) {
//     schoolID := c.Param("schoolId")
//     if schoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "School ID is required",
//         })
//         return
//     }

//     levels, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   levels,
//         "count":  len(levels),
//     })
// }

// // GetClassLevelsByCategory godoc
// // @Summary      Get class levels by category
// // @Description  List class levels by category (JSS, SSS, PRIMARY)
// // @Tags         Academic
// // @Produce      json
// // @Param        schoolId path string true "School ID"
// // @Param        category path string true "Category (JSS, SSS, PRIMARY)"
// // @Success      200  {object}  map[string]interface{}  "data (list)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/school/{schoolId}/category/{category} [get]
// func (h *ClassLevelHandler) GetClassLevelsByCategory(c *gin.Context) {
//     schoolID := c.Param("schoolId")
//     category := c.Param("category")

//     if schoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "School ID is required",
//         })
//         return
//     }
//     if category == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Category is required",
//         })
//         return
//     }

//     levels, err := h.service.GetByCategory(schoolID, category)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   levels,
//         "count":  len(levels),
//     })
// }

// // UpdateClassLevel godoc
// // @Summary      Update a class level
// // @Description  Modify an existing class level.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Param        request body dto.UpdateClassLevelRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [put]
// func (h *ClassLevelHandler) UpdateClassLevel(c *gin.Context) {
//     id := c.Param("id")
//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "ID is required",
//         })
//         return
//     }

//     var req dto.UpdateClassLevelRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request payload",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     level, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Class level updated successfully",
//         "data":    level,
//     })
// }

// // DeleteClassLevel godoc
// // @Summary      Delete a class level
// // @Description  Soft‑delete a class level.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [delete]
// func (h *ClassLevelHandler) DeleteClassLevel(c *gin.Context) {
//     id := c.Param("id")
//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "ID is required",
//         })
//         return
//     }

//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Class level deleted successfully",
//     })
// }

// // ============================================================
// // NEW METHODS FOR COMPLETE CRUD (ALL PARAMETERS OPTIONAL)
// // ============================================================

// // ListClassLevels godoc
// // @Summary      List all class levels with pagination, filtering, sorting
// // @Description  Get paginated list of class levels with optional filters and sorting.
// // @Description  All parameters are optional - omit them to get all class levels.
// // @Tags         Academic
// // @Produce      json
// // @Param        page query int false "Page number" default(1)
// // @Param        limit query int false "Items per page" default(20) max(100)
// // @Param        search query string false "Search by name"
// // @Param        school_id query string false "Filter by school ID"
// // @Param        category query string false "Filter by category (JSS, SSS, PRIMARY)"
// // @Param        is_active query bool false "Filter by active status"
// // @Param        sort_by query string false "Sort by field (name, level_number, category, sort_order, created_at)" default(level_number)
// // @Param        sort_order query string false "Sort order (asc, desc)" default(asc)
// // @Success      200  {object}  map[string]interface{}  "data (paginated list)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels [get]
// func (h *ClassLevelHandler) ListClassLevels(c *gin.Context) {
//     // Parse optional query parameters with defaults
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

//     req := &dto.ListClassLevelsRequest{
//         Page:     page,
//         Limit:    limit,
//         Search:   c.Query("search"),
//         SchoolID: c.Query("school_id"),
//         Category: c.Query("category"),
//         SortBy:   c.DefaultQuery("sort_by", "level_number"),
//         SortDir:  c.DefaultQuery("sort_order", "asc"),
//         IsActive: nil,
//     }

//     // Parse is_active if provided (optional)
//     if isActive := c.Query("is_active"); isActive != "" {
//         val := isActive == "true"
//         req.IsActive = &val
//     }

//     result, err := h.service.List(req)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "message": "Class levels retrieved successfully",
//         "data":    result.Items,
//         "pagination": gin.H{
//             "page":        result.Page,
//             "limit":       result.Limit,
//             "total":       result.Total,
//             "total_pages": result.TotalPages,
//         },
//     })
// }

// // BulkDeleteClassLevels godoc
// // @Summary      Bulk delete class levels
// // @Description  Delete multiple class levels by IDs
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkDeleteRequest true "List of IDs to delete"
// // @Success      200  {object}  map[string]interface{}  "message + deleted_count"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/bulk [delete]
// func (h *ClassLevelHandler) BulkDeleteClassLevels(c *gin.Context) {
//     var req dto.BulkDeleteRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request payload",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     if len(req.IDs) == 0 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "No IDs provided",
//         })
//         return
//     }

//     deletedCount, err := h.service.BulkDelete(req.IDs)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":        "success",
//         "message":       "Class levels deleted successfully",
//         "deleted_count": deletedCount,
//     })
// }

// // GetClassLevelStats godoc
// // @Summary      Get class level statistics
// // @Description  Get statistics about class levels (total, active, by category, etc.)
// // @Tags         Academic
// // @Produce      json
// // @Param        school_id query string false "Filter by school ID"
// // @Success      200  {object}  map[string]interface{}  "data (stats)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/stats [get]
// func (h *ClassLevelHandler) GetClassLevelStats(c *gin.Context) {
//     schoolID := c.Query("school_id")

//     stats, err := h.service.GetStats(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Statistics retrieved successfully",
//         "data":    stats,
//     })
// }

// // SearchClassLevels godoc
// // @Summary      Search class levels
// // @Description  Search class levels by name or other criteria
// // @Tags         Academic
// // @Produce      json
// // @Param        q query string true "Search query"
// // @Param        school_id query string false "Filter by school ID"
// // @Param        page query int false "Page number" default(1)
// // @Param        limit query int false "Items per page" default(20)
// // @Success      200  {object}  map[string]interface{}  "data (search results)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/search [get]
// func (h *ClassLevelHandler) SearchClassLevels(c *gin.Context) {
//     query := c.Query("q")
//     if query == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Search query is required",
//         })
//         return
//     }

//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
//     schoolID := c.Query("school_id")

//     result, err := h.service.Search(schoolID, query, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Search results retrieved successfully",
//         "data":    result.Items,
//         "pagination": gin.H{
//             "page":        result.Page,
//             "limit":       result.Limit,
//             "total":       result.Total,
//             "total_pages": result.TotalPages,
//         },
//     })
// }



// package handler

// import (
//     "net/http"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type ClassLevelHandler struct {
//     service *service.ClassLevelService
// }

// func NewClassLevelHandler(service *service.ClassLevelService) *ClassLevelHandler {
//     return &ClassLevelHandler{service: service}
// }

// // CreateClassLevel godoc
// // @Summary      Create a new class level
// // @Description  Add a class level (e.g., "Grade 10") for a school.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateClassLevelRequest true "Class level details"
// // @Success      201  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels [post]
// func (h *ClassLevelHandler) CreateClassLevel(c *gin.Context) {
//     var req dto.CreateClassLevelRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     level, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Class level created successfully",
//         "data":    level,
//     })
// }

// // GetClassLevel godoc
// // @Summary      Get class level by ID
// // @Description  Retrieve a single class level.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Success      200  {object}  map[string]interface{}  "data"
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [get]
// func (h *ClassLevelHandler) GetClassLevel(c *gin.Context) {
//     id := c.Param("id")
    
//     level, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": level})
// }

// // GetClassLevels godoc
// // @Summary      Get all class levels for a school
// // @Description  List all class levels belonging to a school.
// // @Tags         Academic
// // @Produce      json
// // @Param        schoolId path string true "School ID"
// // @Success      200  {object}  map[string]interface{}  "data (list)"
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /school-class-levels/{schoolId} [get]
// func (h *ClassLevelHandler) GetClassLevels(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     levels, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": levels})
// }

// // UpdateClassLevel godoc
// // @Summary      Update a class level
// // @Description  Modify an existing class level.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Param        request body dto.UpdateClassLevelRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [put]
// func (h *ClassLevelHandler) UpdateClassLevel(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateClassLevelRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     level, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Class level updated successfully",
//         "data":    level,
//     })
// }

// // DeleteClassLevel godoc
// // @Summary      Delete a class level
// // @Description  Soft‑delete a class level.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Level ID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-levels/{id} [delete]
// func (h *ClassLevelHandler) DeleteClassLevel(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Class level deleted successfully"})
// }

