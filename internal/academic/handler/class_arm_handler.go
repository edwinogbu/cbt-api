// internal/academic/handler/class_arm_handler.go
package handler

import (
    "net/http"
    "strconv"
    "strings"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type ClassArmHandler struct {
    service *service.ClassArmService
}

func NewClassArmHandler(service *service.ClassArmService) *ClassArmHandler {
    return &ClassArmHandler{service: service}
}

// ============================================
// CREATE CLASS ARM
// ============================================

// CreateClassArm godoc
// @Summary      Create a new class arm
// @Description  Add a class arm (e.g., "Section A") for a school.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateClassArmRequest true "Class arm details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms [post]
func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
    var req dto.CreateClassArmRequest

    // Validate JSON binding
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Invalid request format",
            "errors":  err.Error(),
        })
        return
    }

    // Validate required fields
    if req.SchoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "school_id is required",
        })
        return
    }

    if req.Name == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "name is required",
        })
        return
    }

    // Validate name length
    if len(req.Name) < 2 || len(req.Name) > 255 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "name must be between 2 and 255 characters",
        })
        return
    }

    // Validate capacity
    if req.Capacity < 0 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "capacity cannot be negative",
        })
        return
    }

    // Trim whitespace
    req.Name = strings.TrimSpace(req.Name)
    req.ArmCode = strings.TrimSpace(req.ArmCode)
    req.RoomNumber = strings.TrimSpace(req.RoomNumber)

    arm, err := h.service.Create(&req)
    if err != nil {
        errMsg := err.Error()
        if strings.Contains(errMsg, "already exists") {
            c.JSON(http.StatusConflict, gin.H{
                "status":  "error",
                "message": "Class arm already exists",
                "errors":  errMsg,
            })
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Failed to create class arm",
            "errors":  errMsg,
        })
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "status":  "success",
        "message": "Class arm created successfully",
        "data":    arm,
    })
}

// ============================================
// GET CLASS ARM BY ID
// ============================================

// GetClassArm godoc
// @Summary      Get class arm by ID
// @Description  Retrieve a single class arm.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Arm ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/{id} [get]
func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
    id := c.Param("id")

    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "class arm ID is required",
        })
        return
    }

    arm, err := h.service.GetByID(id)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            c.JSON(http.StatusNotFound, gin.H{
                "status":  "error",
                "message": "Class arm not found",
                "errors":  err.Error(),
            })
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to retrieve class arm",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   arm,
    })
}

// ============================================
// GET CLASS ARMS BY SCHOOL
// ============================================

// GetClassArms godoc
// @Summary      Get all class arms for a school
// @Description  List all class arms belonging to a school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-class-arms/{schoolId} [get]
func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
    schoolID := c.Param("schoolId")

    if schoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "school ID is required",
        })
        return
    }

    arms, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to retrieve class arms",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   arms,
        "meta": gin.H{
            "count": len(arms),
        },
    })
}

// ============================================
// UPDATE CLASS ARM
// ============================================

// UpdateClassArm godoc
// @Summary      Update a class arm
// @Description  Modify an existing class arm.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Class Arm ID"
// @Param        request body dto.UpdateClassArmRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/{id} [put]
func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
    id := c.Param("id")

    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "class arm ID is required",
        })
        return
    }

    var req dto.UpdateClassArmRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Invalid request format",
            "errors":  err.Error(),
        })
        return
    }

    // Validate name if provided (string, not pointer)
    if req.Name != "" {
        trimmed := strings.TrimSpace(req.Name)
        if len(trimmed) < 2 || len(trimmed) > 255 {
            c.JSON(http.StatusBadRequest, gin.H{
                "status":  "error",
                "message": "Validation failed",
                "errors":  "name must be between 2 and 255 characters",
            })
            return
        }
        req.Name = trimmed
    }

    // Validate ArmCode if provided (string, not pointer)
    if req.ArmCode != "" {
        trimmed := strings.TrimSpace(req.ArmCode)
        req.ArmCode = trimmed
    }

    // Validate RoomNumber if provided (string, not pointer)
    if req.RoomNumber != "" {
        trimmed := strings.TrimSpace(req.RoomNumber)
        req.RoomNumber = trimmed
    }

    // Validate capacity if provided (pointer)
    if req.Capacity != nil && *req.Capacity < 0 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "capacity cannot be negative",
        })
        return
    }

    arm, err := h.service.Update(id, &req)
    if err != nil {
        errMsg := err.Error()
        if strings.Contains(errMsg, "not found") {
            c.JSON(http.StatusNotFound, gin.H{
                "status":  "error",
                "message": "Class arm not found",
                "errors":  errMsg,
            })
            return
        }
        if strings.Contains(errMsg, "already exists") {
            c.JSON(http.StatusConflict, gin.H{
                "status":  "error",
                "message": "Class arm already exists",
                "errors":  errMsg,
            })
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Failed to update class arm",
            "errors":  errMsg,
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Class arm updated successfully",
        "data":    arm,
    })
}

// ============================================
// DELETE CLASS ARM
// ============================================

// DeleteClassArm godoc
// @Summary      Delete a class arm
// @Description  Soft‑delete a class arm.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Arm ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/{id} [delete]
func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
    id := c.Param("id")

    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "class arm ID is required",
        })
        return
    }

    err := h.service.Delete(id)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            c.JSON(http.StatusNotFound, gin.H{
                "status":  "error",
                "message": "Class arm not found",
                "errors":  err.Error(),
            })
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Failed to delete class arm",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Class arm deleted successfully",
    })
}

// ============================================
// LIST CLASS ARMS
// ============================================

// ListClassArms godoc
// @Summary      List all class arms with optional pagination and filters
// @Description  Get a paginated list of class arms. All parameters are optional.
// @Tags         Academic
// @Produce      json
// @Param        page query int false "Page number (default: 1)"
// @Param        limit query int false "Items per page (default: 20, max: 100)"
// @Param        search query string false "Search by name or code"
// @Param        school_id query string false "Filter by school ID"
// @Param        min_capacity query int false "Minimum capacity"
// @Param        max_capacity query int false "Maximum capacity"
// @Param        sort_by query string false "Sort by field (name, arm_code, capacity, sort_order, created_at)"
// @Param        sort_order query string false "Sort order (asc, desc)"
// @Param        is_active query boolean false "Filter by active status"
// @Success      200  {object}  dto.ClassArmListResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms [get]
func (h *ClassArmHandler) ListClassArms(c *gin.Context) {
    var req dto.ListClassArmsRequest

    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Invalid query parameters",
            "errors":  err.Error(),
        })
        return
    }

    // Set defaults
    if req.Page < 1 {
        req.Page = 1
    }
    if req.Limit < 1 || req.Limit > 100 {
        req.Limit = 20
    }

    // Validate sort_by if provided
    validSortFields := []string{"name", "arm_code", "capacity", "sort_order", "created_at"}
    if req.SortBy != "" {
        isValid := false
        for _, field := range validSortFields {
            if req.SortBy == field {
                isValid = true
                break
            }
        }
        if !isValid {
            c.JSON(http.StatusBadRequest, gin.H{
                "status":  "error",
                "message": "Validation failed",
                "errors":  "invalid sort_by field. Allowed: name, arm_code, capacity, sort_order, created_at",
            })
            return
        }
    }

    if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "sort_order must be 'asc' or 'desc'",
        })
        return
    }

    if req.MinCapacity < 0 {
        req.MinCapacity = 0
    }
    if req.MaxCapacity < 0 {
        req.MaxCapacity = 0
    }
    if req.MinCapacity > 0 && req.MaxCapacity > 0 && req.MinCapacity > req.MaxCapacity {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "min_capacity cannot be greater than max_capacity",
        })
        return
    }

    ctx := c.Request.Context()
    response, err := h.service.ListClassArms(ctx, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to retrieve class arms",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   response,
    })
}

// ============================================
// GET CLASS ARM STATS
// ============================================

// GetClassArmStats godoc
// @Summary      Get class arm statistics
// @Description  Get statistics for class arms (total, active, inactive, capacity)
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "School ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/stats [get]
func (h *ClassArmHandler) GetClassArmStats(c *gin.Context) {
    schoolID := c.Query("school_id")

    if schoolID != "" && len(schoolID) != 36 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "invalid school_id format",
        })
        return
    }

    ctx := c.Request.Context()
    stats, err := h.service.GetClassArmStats(ctx, schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to retrieve statistics",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   stats,
    })
}

// ============================================
// BULK DELETE CLASS ARMS
// ============================================

// BulkDeleteClassArms godoc
// @Summary      Bulk delete class arms
// @Description  Delete multiple class arms by IDs
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.BulkDeleteClassArmRequest true "Class arm IDs to delete"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/bulk [delete]
func (h *ClassArmHandler) BulkDeleteClassArms(c *gin.Context) {
    var req dto.BulkDeleteClassArmRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Invalid request format",
            "errors":  err.Error(),
        })
        return
    }

    if len(req.IDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "no class arm IDs provided",
        })
        return
    }

    if len(req.IDs) > 100 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "maximum 100 IDs allowed per request",
        })
        return
    }

    ctx := c.Request.Context()
    count, err := h.service.BulkDeleteClassArms(ctx, req.IDs)
    if err != nil {
        errMsg := err.Error()
        if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no matching") {
            c.JSON(http.StatusNotFound, gin.H{
                "status":  "error",
                "message": "Class arms not found",
                "errors":  errMsg,
            })
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to delete class arms",
            "errors":  errMsg,
        })
        return
    }

    message := "Class arm deleted successfully"
    if count > 1 {
        message = "Class arms deleted successfully"
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": message,
        "data": gin.H{
            "deleted_count":   count,
            "requested_count": len(req.IDs),
        },
    })
}

// ============================================
// SEARCH CLASS ARMS
// ============================================

// SearchClassArms godoc
// @Summary      Search class arms
// @Description  Search class arms by name or code
// @Tags         Academic
// @Produce      json
// @Param        q query string true "Search query"
// @Param        limit query int false "Items per page (default: 20, max: 100)"
// @Param        school_id query string false "School ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/search [get]
func (h *ClassArmHandler) SearchClassArms(c *gin.Context) {
    query := c.Query("q")

    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "search query is required",
        })
        return
    }

    if len(query) < 2 {
        c.JSON(http.StatusBadRequest, gin.H{
            "status":  "error",
            "message": "Validation failed",
            "errors":  "search query must be at least 2 characters",
        })
        return
    }

    limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
    if err != nil || limit < 1 || limit > 100 {
        limit = 20
    }

    ctx := c.Request.Context()
    arms, err := h.service.SearchClassArms(ctx, query, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "status":  "error",
            "message": "Failed to search class arms",
            "errors":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   arms,
        "meta": gin.H{
            "count": len(arms),
            "query": query,
        },
    })
}


// // internal/academic/handler/class_arm_handler.go
// package handler

// import (
//     "net/http"
//     "strconv"
//     "strings"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type ClassArmHandler struct {
//     service *service.ClassArmService
// }

// func NewClassArmHandler(service *service.ClassArmService) *ClassArmHandler {
//     return &ClassArmHandler{service: service}
// }

// // ============================================
// // CREATE CLASS ARM
// // ============================================

// // CreateClassArm godoc
// // @Summary      Create a new class arm
// // @Description  Add a class arm (e.g., "Section A") for a school.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.CreateClassArmRequest true "Class arm details"
// // @Success      201  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      409  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms [post]
// func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
//     var req dto.CreateClassArmRequest

//     // Validate JSON binding
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request format",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     // Validate required fields
//     if req.SchoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "school_id is required",
//         })
//         return
//     }

//     if req.Name == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "name is required",
//         })
//         return
//     }

//     // Validate name length
//     if len(req.Name) < 2 || len(req.Name) > 255 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "name must be between 2 and 255 characters",
//         })
//         return
//     }

//     // Validate capacity
//     if req.Capacity < 0 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "capacity cannot be negative",
//         })
//         return
//     }

//     // Trim whitespace
//     req.Name = strings.TrimSpace(req.Name)
//     req.ArmCode = strings.TrimSpace(req.ArmCode)
//     req.RoomNumber = strings.TrimSpace(req.RoomNumber)

//     arm, err := h.service.Create(&req)
//     if err != nil {
//         errMsg := err.Error()
//         if strings.Contains(errMsg, "already exists") {
//             c.JSON(http.StatusConflict, gin.H{
//                 "status":  "error",
//                 "message": "Class arm already exists",
//                 "errors":  errMsg,
//             })
//             return
//         }
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Failed to create class arm",
//             "errors":  errMsg,
//         })
//         return
//     }

//     c.JSON(http.StatusCreated, gin.H{
//         "status":  "success",
//         "message": "Class arm created successfully",
//         "data":    arm,
//     })
// }

// // ============================================
// // GET CLASS ARM BY ID
// // ============================================

// // GetClassArm godoc
// // @Summary      Get class arm by ID
// // @Description  Retrieve a single class arm.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Arm ID"
// // @Success      200  {object}  map[string]interface{}  "data"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/{id} [get]
// func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
//     id := c.Param("id")

//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "class arm ID is required",
//         })
//         return
//     }

//     arm, err := h.service.GetByID(id)
//     if err != nil {
//         if strings.Contains(err.Error(), "not found") {
//             c.JSON(http.StatusNotFound, gin.H{
//                 "status":  "error",
//                 "message": "Class arm not found",
//                 "errors":  err.Error(),
//             })
//             return
//         }
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to retrieve class arm",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   arm,
//     })
// }

// // ============================================
// // GET CLASS ARMS BY SCHOOL
// // ============================================

// // GetClassArms godoc
// // @Summary      Get all class arms for a school
// // @Description  List all class arms belonging to a school.
// // @Tags         Academic
// // @Produce      json
// // @Param        schoolId path string true "School ID"
// // @Success      200  {object}  map[string]interface{}  "data (list)"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /school-class-arms/{schoolId} [get]
// func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
//     schoolID := c.Param("schoolId")

//     if schoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "school ID is required",
//         })
//         return
//     }

//     arms, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to retrieve class arms",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   arms,
//         "meta": gin.H{
//             "count": len(arms),
//         },
//     })
// }

// // ============================================
// // UPDATE CLASS ARM
// // ============================================

// // UpdateClassArm godoc
// // @Summary      Update a class arm
// // @Description  Modify an existing class arm.
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        id path string true "Class Arm ID"
// // @Param        request body dto.UpdateClassArmRequest true "Fields to update"
// // @Success      200  {object}  map[string]interface{}  "message + data"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/{id} [put]
// func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
//     id := c.Param("id")

//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "class arm ID is required",
//         })
//         return
//     }

//     var req dto.UpdateClassArmRequest

//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request format",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     // Validate name if provided
//     if req.Name != nil && *req.Name != "" {
//         trimmed := strings.TrimSpace(*req.Name)
//         if len(trimmed) < 2 || len(trimmed) > 255 {
//             c.JSON(http.StatusBadRequest, gin.H{
//                 "status":  "error",
//                 "message": "Validation failed",
//                 "errors":  "name must be between 2 and 255 characters",
//             })
//             return
//         }
//         req.Name = &trimmed
//     }

//     // Validate capacity if provided
//     if req.Capacity != nil && *req.Capacity < 0 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "capacity cannot be negative",
//         })
//         return
//     }

//     // Trim whitespace for optional fields
//     if req.ArmCode != nil && *req.ArmCode != "" {
//         trimmed := strings.TrimSpace(*req.ArmCode)
//         req.ArmCode = &trimmed
//     }
//     if req.RoomNumber != nil && *req.RoomNumber != "" {
//         trimmed := strings.TrimSpace(*req.RoomNumber)
//         req.RoomNumber = &trimmed
//     }

//     arm, err := h.service.Update(id, &req)
//     if err != nil {
//         errMsg := err.Error()
//         if strings.Contains(errMsg, "not found") {
//             c.JSON(http.StatusNotFound, gin.H{
//                 "status":  "error",
//                 "message": "Class arm not found",
//                 "errors":  errMsg,
//             })
//             return
//         }
//         if strings.Contains(errMsg, "already exists") {
//             c.JSON(http.StatusConflict, gin.H{
//                 "status":  "error",
//                 "message": "Class arm already exists",
//                 "errors":  errMsg,
//             })
//             return
//         }
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Failed to update class arm",
//             "errors":  errMsg,
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Class arm updated successfully",
//         "data":    arm,
//     })
// }

// // ============================================
// // DELETE CLASS ARM
// // ============================================

// // DeleteClassArm godoc
// // @Summary      Delete a class arm
// // @Description  Soft‑delete a class arm.
// // @Tags         Academic
// // @Produce      json
// // @Param        id path string true "Class Arm ID"
// // @Success      200  {object}  map[string]interface{}  "message"
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/{id} [delete]
// func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
//     id := c.Param("id")

//     if id == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "class arm ID is required",
//         })
//         return
//     }

//     err := h.service.Delete(id)
//     if err != nil {
//         if strings.Contains(err.Error(), "not found") {
//             c.JSON(http.StatusNotFound, gin.H{
//                 "status":  "error",
//                 "message": "Class arm not found",
//                 "errors":  err.Error(),
//             })
//             return
//         }
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Failed to delete class arm",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": "Class arm deleted successfully",
//     })
// }

// // ============================================
// // LIST CLASS ARMS
// // ============================================

// // ListClassArms godoc
// // @Summary      List all class arms with optional pagination and filters
// // @Description  Get a paginated list of class arms. All parameters are optional.
// // @Tags         Academic
// // @Produce      json
// // @Param        page query int false "Page number (default: 1)"
// // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // @Param        search query string false "Search by name or code"
// // @Param        school_id query string false "Filter by school ID"
// // @Param        min_capacity query int false "Minimum capacity"
// // @Param        max_capacity query int false "Maximum capacity"
// // @Param        sort_by query string false "Sort by field (name, arm_code, capacity, sort_order, created_at)"
// // @Param        sort_order query string false "Sort order (asc, desc)"
// // @Param        is_active query boolean false "Filter by active status"
// // @Success      200  {object}  dto.ClassArmListResponse
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms [get]
// func (h *ClassArmHandler) ListClassArms(c *gin.Context) {
//     var req dto.ListClassArmsRequest

//     if err := c.ShouldBindQuery(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid query parameters",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     // Set defaults
//     if req.Page < 1 {
//         req.Page = 1
//     }
//     if req.Limit < 1 || req.Limit > 100 {
//         req.Limit = 20
//     }

//     // Validate sort_by if provided
//     validSortFields := []string{"name", "arm_code", "capacity", "sort_order", "created_at"}
//     if req.SortBy != "" {
//         isValid := false
//         for _, field := range validSortFields {
//             if req.SortBy == field {
//                 isValid = true
//                 break
//             }
//         }
//         if !isValid {
//             c.JSON(http.StatusBadRequest, gin.H{
//                 "status":  "error",
//                 "message": "Validation failed",
//                 "errors":  "invalid sort_by field. Allowed: name, arm_code, capacity, sort_order, created_at",
//             })
//             return
//         }
//     }

//     if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "sort_order must be 'asc' or 'desc'",
//         })
//         return
//     }

//     if req.MinCapacity < 0 {
//         req.MinCapacity = 0
//     }
//     if req.MaxCapacity < 0 {
//         req.MaxCapacity = 0
//     }
//     if req.MinCapacity > 0 && req.MaxCapacity > 0 && req.MinCapacity > req.MaxCapacity {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "min_capacity cannot be greater than max_capacity",
//         })
//         return
//     }

//     ctx := c.Request.Context()
//     response, err := h.service.ListClassArms(ctx, &req)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to retrieve class arms",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   response,
//     })
// }

// // ============================================
// // GET CLASS ARM STATS
// // ============================================

// // GetClassArmStats godoc
// // @Summary      Get class arm statistics
// // @Description  Get statistics for class arms (total, active, inactive, capacity)
// // @Tags         Academic
// // @Produce      json
// // @Param        school_id query string false "School ID"
// // @Success      200  {object}  map[string]interface{}
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/stats [get]
// func (h *ClassArmHandler) GetClassArmStats(c *gin.Context) {
//     schoolID := c.Query("school_id")

//     if schoolID != "" && len(schoolID) != 36 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "invalid school_id format",
//         })
//         return
//     }

//     ctx := c.Request.Context()
//     stats, err := h.service.GetClassArmStats(ctx, schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to retrieve statistics",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   stats,
//     })
// }

// // ============================================
// // BULK DELETE CLASS ARMS
// // ============================================

// // BulkDeleteClassArms godoc
// // @Summary      Bulk delete class arms
// // @Description  Delete multiple class arms by IDs
// // @Tags         Academic
// // @Accept       json
// // @Produce      json
// // @Param        request body dto.BulkDeleteClassArmRequest true "Class arm IDs to delete"
// // @Success      200  {object}  map[string]interface{}
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      404  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/bulk [delete]
// func (h *ClassArmHandler) BulkDeleteClassArms(c *gin.Context) {
//     var req dto.BulkDeleteClassArmRequest

//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Invalid request format",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     if len(req.IDs) == 0 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "no class arm IDs provided",
//         })
//         return
//     }

//     if len(req.IDs) > 100 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "maximum 100 IDs allowed per request",
//         })
//         return
//     }

//     ctx := c.Request.Context()
//     count, err := h.service.BulkDeleteClassArms(ctx, req.IDs)
//     if err != nil {
//         errMsg := err.Error()
//         if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no matching") {
//             c.JSON(http.StatusNotFound, gin.H{
//                 "status":  "error",
//                 "message": "Class arms not found",
//                 "errors":  errMsg,
//             })
//             return
//         }
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to delete class arms",
//             "errors":  errMsg,
//         })
//         return
//     }

//     message := "Class arm deleted successfully"
//     if count > 1 {
//         message = "Class arms deleted successfully"
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status":  "success",
//         "message": message,
//         "data": gin.H{
//             "deleted_count":   count,
//             "requested_count": len(req.IDs),
//         },
//     })
// }

// // ============================================
// // SEARCH CLASS ARMS
// // ============================================

// // SearchClassArms godoc
// // @Summary      Search class arms
// // @Description  Search class arms by name or code
// // @Tags         Academic
// // @Produce      json
// // @Param        q query string true "Search query"
// // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // @Param        school_id query string false "School ID"
// // @Success      200  {object}  map[string]interface{}
// // @Failure      400  {object}  map[string]interface{}
// // @Failure      500  {object}  map[string]interface{}
// // @Security     BearerAuth
// // @Router       /class-arms/search [get]
// func (h *ClassArmHandler) SearchClassArms(c *gin.Context) {
//     query := c.Query("q")

//     if query == "" {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "search query is required",
//         })
//         return
//     }

//     if len(query) < 2 {
//         c.JSON(http.StatusBadRequest, gin.H{
//             "status":  "error",
//             "message": "Validation failed",
//             "errors":  "search query must be at least 2 characters",
//         })
//         return
//     }

//     limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
//     if err != nil || limit < 1 || limit > 100 {
//         limit = 20
//     }

//     ctx := c.Request.Context()
//     arms, err := h.service.SearchClassArms(ctx, query, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{
//             "status":  "error",
//             "message": "Failed to search class arms",
//             "errors":  err.Error(),
//         })
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "status": "success",
//         "data":   arms,
//         "meta": gin.H{
//             "count": len(arms),
//             "query": query,
//         },
//     })
// }



// // // internal/academic/handler/class_arm_handler.go
// // package handler

// // import (
// //     "net/http"
// //     "strconv"
// //     "strings"

// //     "cbt-api/internal/academic/dto"
// //     "cbt-api/internal/academic/service"

// //     "github.com/gin-gonic/gin"
// // )

// // type ClassArmHandler struct {
// //     service *service.ClassArmService
// // }

// // func NewClassArmHandler(service *service.ClassArmService) *ClassArmHandler {
// //     return &ClassArmHandler{service: service}
// // }

// // // ============================================
// // // CREATE CLASS ARM
// // // ============================================

// // // CreateClassArm godoc
// // // @Summary      Create a new class arm
// // // @Description  Add a class arm (e.g., "Section A") for a school.
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.CreateClassArmRequest true "Class arm details"
// // // @Success      201  {object}  map[string]interface{}  "message + data"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      409  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms [post]
// // func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
// //     var req dto.CreateClassArmRequest

// //     // Validate JSON binding
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Invalid request format",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     // Validate required fields
// //     if req.SchoolID == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "school_id is required",
// //         })
// //         return
// //     }

// //     if req.Name == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "name is required",
// //         })
// //         return
// //     }

// //     // Validate name length
// //     if len(req.Name) < 2 || len(req.Name) > 255 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "name must be between 2 and 255 characters",
// //         })
// //         return
// //     }

// //     // Validate capacity
// //     if req.Capacity < 0 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "capacity cannot be negative",
// //         })
// //         return
// //     }

// //     // Trim whitespace
// //     req.Name = strings.TrimSpace(req.Name)
// //     req.ArmCode = strings.TrimSpace(req.ArmCode)
// //     req.RoomNumber = strings.TrimSpace(req.RoomNumber)

// //     arm, err := h.service.Create(&req)
// //     if err != nil {
// //         // Check for duplicate error
// //         errMsg := err.Error()
// //         if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "already exists") {
// //             c.JSON(http.StatusConflict, gin.H{
// //                 "status":  "error",
// //                 "message": "Class arm already exists",
// //                 "errors":  errMsg,
// //             })
// //             return
// //         }
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Failed to create class arm",
// //             "errors":  errMsg,
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusCreated, gin.H{
// //         "status":  "success",
// //         "message": "Class arm created successfully",
// //         "data":    arm,
// //     })
// // }

// // // ============================================
// // // GET CLASS ARM BY ID
// // // ============================================

// // // GetClassArm godoc
// // // @Summary      Get class arm by ID
// // // @Description  Retrieve a single class arm.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Success      200  {object}  map[string]interface{}  "data"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      404  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [get]
// // func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
// //     id := c.Param("id")

// //     // Validate ID
// //     if id == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "class arm ID is required",
// //         })
// //         return
// //     }

// //     arm, err := h.service.GetByID(id)
// //     if err != nil {
// //         if strings.Contains(err.Error(), "not found") {
// //             c.JSON(http.StatusNotFound, gin.H{
// //                 "status":  "error",
// //                 "message": "Class arm not found",
// //                 "errors":  err.Error(),
// //             })
// //             return
// //         }
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to retrieve class arm",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   arm,
// //     })
// // }

// // // ============================================
// // // GET CLASS ARMS BY SCHOOL
// // // ============================================

// // // GetClassArms godoc
// // // @Summary      Get all class arms for a school
// // // @Description  List all class arms belonging to a school.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        schoolId path string true "School ID"
// // // @Success      200  {object}  map[string]interface{}  "data (list)"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /school-class-arms/{schoolId} [get]
// // func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
// //     schoolID := c.Param("schoolId")

// //     // Validate school ID
// //     if schoolID == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "school ID is required",
// //         })
// //         return
// //     }

// //     arms, err := h.service.GetBySchool(schoolID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to retrieve class arms",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   arms,
// //         "meta": gin.H{
// //             "count": len(arms),
// //         },
// //     })
// // }

// // // ============================================
// // // UPDATE CLASS ARM
// // // ============================================

// // // UpdateClassArm godoc
// // // @Summary      Update a class arm
// // // @Description  Modify an existing class arm.
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Param        request body dto.UpdateClassArmRequest true "Fields to update"
// // // @Success      200  {object}  map[string]interface{}  "message + data"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      404  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [put]
// // func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
// //     id := c.Param("id")

// //     // Validate ID
// //     if id == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "class arm ID is required",
// //         })
// //         return
// //     }

// //     var req dto.UpdateClassArmRequest

// //     // Validate JSON binding
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Invalid request format",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     // Validate name if provided
// //     if req.Name != nil {
// //         trimmed := strings.TrimSpace(*req.Name)
// //         req.Name = &trimmed
// //         if len(*req.Name) < 2 || len(*req.Name) > 255 {
// //             c.JSON(http.StatusBadRequest, gin.H{
// //                 "status":  "error",
// //                 "message": "Validation failed",
// //                 "errors":  "name must be between 2 and 255 characters",
// //             })
// //             return
// //         }
// //     }

// //     // Validate capacity if provided
// //     if req.Capacity != nil && *req.Capacity < 0 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "capacity cannot be negative",
// //         })
// //         return
// //     }

// //     // Trim whitespace for optional fields
// //     if req.ArmCode != nil {
// //         trimmed := strings.TrimSpace(*req.ArmCode)
// //         req.ArmCode = &trimmed
// //     }
// //     if req.RoomNumber != nil {
// //         trimmed := strings.TrimSpace(*req.RoomNumber)
// //         req.RoomNumber = &trimmed
// //     }

// //     arm, err := h.service.Update(id, &req)
// //     if err != nil {
// //         errMsg := err.Error()
// //         if strings.Contains(errMsg, "not found") {
// //             c.JSON(http.StatusNotFound, gin.H{
// //                 "status":  "error",
// //                 "message": "Class arm not found",
// //                 "errors":  errMsg,
// //             })
// //             return
// //         }
// //         if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "already exists") {
// //             c.JSON(http.StatusConflict, gin.H{
// //                 "status":  "error",
// //                 "message": "Class arm already exists",
// //                 "errors":  errMsg,
// //             })
// //             return
// //         }
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Failed to update class arm",
// //             "errors":  errMsg,
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status":  "success",
// //         "message": "Class arm updated successfully",
// //         "data":    arm,
// //     })
// // }

// // // ============================================
// // // DELETE CLASS ARM
// // // ============================================

// // // DeleteClassArm godoc
// // // @Summary      Delete a class arm
// // // @Description  Soft‑delete a class arm.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Success      200  {object}  map[string]interface{}  "message"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      404  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [delete]
// // func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
// //     id := c.Param("id")

// //     // Validate ID
// //     if id == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "class arm ID is required",
// //         })
// //         return
// //     }

// //     err := h.service.Delete(id)
// //     if err != nil {
// //         if strings.Contains(err.Error(), "not found") {
// //             c.JSON(http.StatusNotFound, gin.H{
// //                 "status":  "error",
// //                 "message": "Class arm not found",
// //                 "errors":  err.Error(),
// //             })
// //             return
// //         }
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Failed to delete class arm",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status":  "success",
// //         "message": "Class arm deleted successfully",
// //     })
// // }

// // // ============================================
// // // LIST CLASS ARMS (NEW)
// // // ============================================

// // // ListClassArms godoc
// // // @Summary      List all class arms with optional pagination and filters
// // // @Description  Get a paginated list of class arms. All parameters are optional.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        page query int false "Page number (default: 1)"
// // // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // // @Param        search query string false "Search by name or code"
// // // @Param        school_id query string false "Filter by school ID"
// // // @Param        min_capacity query int false "Minimum capacity"
// // // @Param        max_capacity query int false "Maximum capacity"
// // // @Param        sort_by query string false "Sort by field (name, arm_code, capacity, sort_order, created_at)"
// // // @Param        sort_order query string false "Sort order (asc, desc)"
// // // @Param        is_active query boolean false "Filter by active status"
// // // @Success      200  {object}  dto.ClassArmListResponse
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms [get]
// // func (h *ClassArmHandler) ListClassArms(c *gin.Context) {
// //     var req dto.ListClassArmsRequest

// //     // Bind query parameters (all optional)
// //     if err := c.ShouldBindQuery(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Invalid query parameters",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     // Set defaults for pagination (only if not provided)
// //     if req.Page < 1 {
// //         req.Page = 1
// //     }

// //     if req.Limit < 1 || req.Limit > 100 {
// //         req.Limit = 20
// //     }

// //     // Validate sort_by if provided (optional)
// //     validSortFields := []string{"name", "arm_code", "capacity", "sort_order", "created_at"}
// //     if req.SortBy != "" {
// //         isValid := false
// //         for _, field := range validSortFields {
// //             if req.SortBy == field {
// //                 isValid = true
// //                 break
// //             }
// //         }
// //         if !isValid {
// //             c.JSON(http.StatusBadRequest, gin.H{
// //                 "status":  "error",
// //                 "message": "Validation failed",
// //                 "errors":  "invalid sort_by field. Allowed: name, arm_code, capacity, sort_order, created_at",
// //             })
// //             return
// //         }
// //     }

// //     // Validate sort_order if provided (optional)
// //     if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "sort_order must be 'asc' or 'desc'",
// //         })
// //         return
// //     }

// //     // Validate capacity range if provided (optional)
// //     if req.MinCapacity < 0 {
// //         req.MinCapacity = 0
// //     }
// //     if req.MaxCapacity < 0 {
// //         req.MaxCapacity = 0
// //     }
// //     if req.MinCapacity > 0 && req.MaxCapacity > 0 && req.MinCapacity > req.MaxCapacity {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "min_capacity cannot be greater than max_capacity",
// //         })
// //         return
// //     }

// //     ctx := c.Request.Context()
// //     response, err := h.service.ListClassArms(ctx, &req)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to retrieve class arms",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   response,
// //     })
// // }

// // // ============================================
// // // GET CLASS ARM STATS (NEW)
// // // ============================================

// // // GetClassArmStats godoc
// // // @Summary      Get class arm statistics
// // // @Description  Get statistics for class arms (total, active, inactive, capacity)
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        school_id query string false "School ID"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/stats [get]
// // func (h *ClassArmHandler) GetClassArmStats(c *gin.Context) {
// //     schoolID := c.Query("school_id")

// //     // Validate school_id format (optional, but if provided should be valid)
// //     if schoolID != "" && len(schoolID) != 36 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "invalid school_id format",
// //         })
// //         return
// //     }

// //     ctx := c.Request.Context()
// //     stats, err := h.service.GetClassArmStats(ctx, schoolID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to retrieve statistics",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   stats,
// //     })
// // }

// // // ============================================
// // // BULK DELETE CLASS ARMS (NEW)
// // // ============================================

// // // BulkDeleteClassArms godoc
// // // @Summary      Bulk delete class arms
// // // @Description  Delete multiple class arms by IDs
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.BulkDeleteClassArmRequest true "Class arm IDs to delete"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      404  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/bulk [delete]
// // func (h *ClassArmHandler) BulkDeleteClassArms(c *gin.Context) {
// //     var req dto.BulkDeleteClassArmRequest

// //     // Validate JSON binding
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Invalid request format",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     // Validate IDs
// //     if len(req.IDs) == 0 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "no class arm IDs provided",
// //         })
// //         return
// //     }

// //     // Validate each ID is not empty
// //     var validIDs []string
// //     for _, id := range req.IDs {
// //         if id != "" {
// //             validIDs = append(validIDs, id)
// //         }
// //     }

// //     if len(validIDs) == 0 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "all provided IDs are empty",
// //         })
// //         return
// //     }

// //     // Limit bulk delete to prevent performance issues
// //     if len(validIDs) > 100 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "maximum 100 IDs allowed per request",
// //         })
// //         return
// //     }

// //     ctx := c.Request.Context()
// //     count, err := h.service.BulkDeleteClassArms(ctx, validIDs)
// //     if err != nil {
// //         errMsg := err.Error()
// //         if strings.Contains(errMsg, "not found") {
// //             c.JSON(http.StatusNotFound, gin.H{
// //                 "status":  "error",
// //                 "message": "Some class arms not found",
// //                 "errors":  errMsg,
// //             })
// //             return
// //         }
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to delete class arms",
// //             "errors":  errMsg,
// //         })
// //         return
// //     }

// //     if count == 0 {
// //         c.JSON(http.StatusNotFound, gin.H{
// //             "status":  "error",
// //             "message": "No class arms found to delete",
// //             "errors":  "The provided IDs may not exist or were already deleted",
// //         })
// //         return
// //     }

// //     message := "Class arm deleted successfully"
// //     if count > 1 {
// //         message = "Class arms deleted successfully"
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status":  "success",
// //         "message": message,
// //         "data": gin.H{
// //             "deleted_count":   count,
// //             "requested_count": len(validIDs),
// //         },
// //     })
// // }

// // // ============================================
// // // SEARCH CLASS ARMS (NEW)
// // // ============================================

// // // SearchClassArms godoc
// // // @Summary      Search class arms
// // // @Description  Search class arms by name or code
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        q query string true "Search query"
// // // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // // @Param        school_id query string false "School ID"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/search [get]
// // func (h *ClassArmHandler) SearchClassArms(c *gin.Context) {
// //     query := c.Query("q")

// //     // Validate search query
// //     if query == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "search query is required",
// //         })
// //         return
// //     }

// //     // Validate query length
// //     if len(query) < 2 {
// //         c.JSON(http.StatusBadRequest, gin.H{
// //             "status":  "error",
// //             "message": "Validation failed",
// //             "errors":  "search query must be at least 2 characters",
// //         })
// //         return
// //     }

// //     // Validate and sanitize limit
// //     limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
// //     if err != nil || limit < 1 || limit > 100 {
// //         limit = 20
// //     }

// //     ctx := c.Request.Context()
// //     arms, err := h.service.SearchClassArms(ctx, query, limit)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{
// //             "status":  "error",
// //             "message": "Failed to search class arms",
// //             "errors":  err.Error(),
// //         })
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   arms,
// //         "meta": gin.H{
// //             "count": len(arms),
// //             "query": query,
// //         },
// //     })
// // }


// // package handler

// // import (
// //     "net/http"
// //     "strconv"
    

// //     "cbt-api/internal/academic/dto"
// //     "cbt-api/internal/academic/service"

// //     "github.com/gin-gonic/gin"
// // )

// // type ClassArmHandler struct {
// //     service *service.ClassArmService
// // }

// // func NewClassArmHandler(service *service.ClassArmService) *ClassArmHandler {
// //     return &ClassArmHandler{service: service}
// // }

// // // CreateClassArm godoc
// // // @Summary      Create a new class arm
// // // @Description  Add a class arm (e.g., "Section A") for a school.
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.CreateClassArmRequest true "Class arm details"
// // // @Success      201  {object}  map[string]interface{}  "message + data"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms [post]
// // func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
// //     var req dto.CreateClassArmRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     arm, err := h.service.Create(&req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusCreated, gin.H{
// //         "message": "Class arm created successfully",
// //         "data":    arm,
// //     })
// // }

// // // GetClassArm godoc
// // // @Summary      Get class arm by ID
// // // @Description  Retrieve a single class arm.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Success      200  {object}  map[string]interface{}  "data"
// // // @Failure      404  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [get]
// // func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     arm, err := h.service.GetByID(id)
// //     if err != nil {
// //         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"data": arm})
// // }

// // // GetClassArms godoc
// // // @Summary      Get all class arms for a school
// // // @Description  List all class arms belonging to a school.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        schoolId path string true "School ID"
// // // @Success      200  {object}  map[string]interface{}  "data (list)"
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /school-class-arms/{schoolId} [get]
// // func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
// //     schoolID := c.Param("schoolId")
    
// //     arms, err := h.service.GetBySchool(schoolID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"data": arms})
// // }

// // // UpdateClassArm godoc
// // // @Summary      Update a class arm
// // // @Description  Modify an existing class arm.
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Param        request body dto.UpdateClassArmRequest true "Fields to update"
// // // @Success      200  {object}  map[string]interface{}  "message + data"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [put]
// // func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     var req dto.UpdateClassArmRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     arm, err := h.service.Update(id, &req)
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{
// //         "message": "Class arm updated successfully",
// //         "data":    arm,
// //     })
// // }

// // // DeleteClassArm godoc
// // // @Summary      Delete a class arm
// // // @Description  Soft‑delete a class arm.
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        id path string true "Class Arm ID"
// // // @Success      200  {object}  map[string]interface{}  "message"
// // // @Failure      400  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/{id} [delete]
// // func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
// //     id := c.Param("id")
    
// //     if err := h.service.Delete(id); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }
    
// //     c.JSON(http.StatusOK, gin.H{"message": "Class arm deleted successfully"})
// // }

// // // Add these new handler methods to the existing file

// // // ListClassArms godoc
// // // @Summary      List all class arms with pagination and filters
// // // @Description  Get a paginated list of class arms with filtering and sorting
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        page query int false "Page number (default: 1)"
// // // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // // @Param        search query string false "Search by name or code"
// // // @Param        school_id query string false "Filter by school ID"
// // // @Param        min_capacity query int false "Minimum capacity"
// // // @Param        max_capacity query int false "Maximum capacity"
// // // @Param        sort_by query string false "Sort by field (name, arm_code, capacity, sort_order, created_at)"
// // // @Param        sort_order query string false "Sort order (asc, desc)"
// // // @Param        is_active query boolean false "Filter by active status"
// // // @Success      200  {object}  dto.ClassArmListResponse
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms [get]
// // func (h *ClassArmHandler) ListClassArms(c *gin.Context) {
// //     var req dto.ListClassArmsRequest
// //     if err := c.ShouldBindQuery(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }

// //     ctx := c.Request.Context()
// //     response, err := h.service.ListClassArms(ctx, &req)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   response,
// //     })
// // }

// // // GetClassArmStats godoc
// // // @Summary      Get class arm statistics
// // // @Description  Get statistics for class arms (total, active, inactive, capacity)
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        school_id query string false "School ID"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/stats [get]
// // func (h *ClassArmHandler) GetClassArmStats(c *gin.Context) {
// //     schoolID := c.Query("school_id")

// //     ctx := c.Request.Context()
// //     stats, err := h.service.GetClassArmStats(ctx, schoolID)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   stats,
// //     })
// // }

// // // BulkDeleteClassArms godoc
// // // @Summary      Bulk delete class arms
// // // @Description  Delete multiple class arms by IDs
// // // @Tags         Academic
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.BulkDeleteRequest true "Class arm IDs to delete"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/bulk [delete]
// // func (h *ClassArmHandler) BulkDeleteClassArms(c *gin.Context) {
// //     var req dto.BulkDeleteRequest
// //     if err := c.ShouldBindJSON(&req); err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// //         return
// //     }

// //     if len(req.IDs) == 0 {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": "no class arm IDs provided"})
// //         return
// //     }

// //     ctx := c.Request.Context()
// //     count, err := h.service.BulkDeleteClassArms(ctx, req.IDs)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status":  "success",
// //         "message": "Class arms deleted successfully",
// //         "data": gin.H{
// //             "deleted_count": count,
// //         },
// //     })
// // }

// // // SearchClassArms godoc
// // // @Summary      Search class arms
// // // @Description  Search class arms by name or code with pagination
// // // @Tags         Academic
// // // @Produce      json
// // // @Param        q query string true "Search query"
// // // @Param        limit query int false "Items per page (default: 20, max: 100)"
// // // @Param        school_id query string false "School ID"
// // // @Success      200  {object}  map[string]interface{}
// // // @Failure      400  {object}  map[string]interface{}
// // // @Failure      500  {object}  map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /class-arms/search [get]
// // func (h *ClassArmHandler) SearchClassArms(c *gin.Context) {
// //     query := c.Query("q")
// //     if query == "" {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
// //         return
// //     }

// //     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
// //     if limit < 1 || limit > 100 {
// //         limit = 20
// //     }

// //     ctx := c.Request.Context()
// //     arms, err := h.service.SearchClassArms(ctx, query, limit)
// //     if err != nil {
// //         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// //         return
// //     }

// //     c.JSON(http.StatusOK, gin.H{
// //         "status": "success",
// //         "data":   arms,
// //         "count":  len(arms),
// //     })
// // }