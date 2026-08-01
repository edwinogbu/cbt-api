package handler

import (
    "net/http"
    "strconv"  // ✅ ADD THIS

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type ClassHandler struct {
    service *service.ClassService
}

func NewClassHandler(service *service.ClassService) *ClassHandler {
    return &ClassHandler{service: service}
}

// CreateClass godoc
// @Summary      Create a new class
// @Description  Create a class combining a class level and a class arm.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateClassRequest true "Class details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes [post]
func (h *ClassHandler) CreateClass(c *gin.Context) {
    var req dto.CreateClassRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    class, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Class created successfully",
        "data":    class,
    })
}

// GetClass godoc
// @Summary      Get class by ID
// @Description  Retrieve a single class.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/{id} [get]
func (h *ClassHandler) GetClass(c *gin.Context) {
    id := c.Param("id")
    
    class, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": class})
}

// GetClassesBySchool godoc
// @Summary      Get all classes for a school
// @Description  List classes belonging to a specific school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-classes/{schoolId} [get]
func (h *ClassHandler) GetClassesBySchool(c *gin.Context) {
    schoolID := c.Param("schoolId")
    
    classes, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": classes})
}

// GetClassesBySession godoc
// @Summary      Get all classes for a session
// @Description  List classes belonging to a specific academic session.
// @Tags         Academic
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /session-classes/{sessionId} [get]
func (h *ClassHandler) GetClassesBySession(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    classes, err := h.service.GetBySession(sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": classes})
}

// GetClassesBySchoolAndSession godoc
// @Summary      Get classes by school and session
// @Description  List classes for a given school and academic session.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Param        sessionId path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-session-classes/{schoolId}/{sessionId} [get]
func (h *ClassHandler) GetClassesBySchoolAndSession(c *gin.Context) {
    schoolID := c.Param("schoolId")
    sessionID := c.Param("sessionId")
    
    classes, err := h.service.GetBySchoolAndSession(schoolID, sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": classes})
}

// UpdateClass godoc
// @Summary      Update a class
// @Description  Modify an existing class.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Class ID"
// @Param        request body dto.UpdateClassRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/{id} [put]
func (h *ClassHandler) UpdateClass(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateClassRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    class, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class updated successfully",
        "data":    class,
    })
}

// DeleteClass godoc
// @Summary      Delete a class
// @Description  Soft‑delete a class.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/{id} [delete]
func (h *ClassHandler) DeleteClass(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Class deleted successfully"})
}

// ============================================================
// NEW HANDLER METHODS FOR MISSING ENDPOINTS
// ============================================================

// ListClasses godoc
// @Summary      List all classes with pagination, filtering, sorting
// @Description  Get paginated list of classes with optional filters.
// @Description  ALL parameters are optional - returns all classes if no params.
// @Tags         Academic
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20) max(100)
// @Param        search query string false "Search by class code or room number"
// @Param        school_id query string false "Filter by school ID"
// @Param        session_id query string false "Filter by session ID"
// @Param        class_level_id query string false "Filter by class level ID"
// @Param        class_arm_id query string false "Filter by class arm ID"
// @Param        teacher_id query string false "Filter by teacher ID"
// @Param        is_active query bool false "Filter by active status"
// @Param        sort_by query string false "Sort by field (class_code, room_number, created_at)" default(created_at)
// @Param        sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success      200  {object}  dto.ClassListResponse
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes [get]
func (h *ClassHandler) ListClasses(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    req := &dto.ListClassesRequest{
        Page:         page,
        Limit:        limit,
        Search:       c.Query("search"),
        SchoolID:     c.Query("school_id"),
        SessionID:    c.Query("session_id"),
        ClassLevelID: c.Query("class_level_id"),
        ClassArmID:   c.Query("class_arm_id"),
        TeacherID:    c.Query("teacher_id"),
        SortBy:       c.DefaultQuery("sort_by", "classes.created_at"),
        SortOrder:    c.DefaultQuery("sort_order", "DESC"),
        IsActive:     nil,
    }

    if isActive := c.Query("is_active"); isActive != "" {
        val := isActive == "true"
        req.IsActive = &val
    }

    ctx := c.Request.Context()
    resp, err := h.service.ListClasses(ctx, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to list classes",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   resp,
    })
}

// GetClassesByTeacher godoc
// @Summary      Get classes by teacher
// @Description  List classes assigned to a specific teacher.
// @Tags         Academic
// @Produce      json
// @Param        teacherId path string true "Teacher ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/teacher/{teacherId} [get]
func (h *ClassHandler) GetClassesByTeacher(c *gin.Context) {
    teacherID := c.Param("teacherId")
    ctx := c.Request.Context()

    classes, err := h.service.GetByTeacher(ctx, teacherID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to fetch classes",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   classes,
        "count":  len(classes),
    })
}

// GetClassesByArm godoc
// @Summary      Get classes by class arm
// @Description  List classes belonging to a specific class arm.
// @Tags         Academic
// @Produce      json
// @Param        classArmId path string true "Class Arm ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/arm/{classArmId} [get]
func (h *ClassHandler) GetClassesByArm(c *gin.Context) {
    classArmID := c.Param("classArmId")
    ctx := c.Request.Context()

    classes, err := h.service.GetByArm(ctx, classArmID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to fetch classes",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   classes,
        "count":  len(classes),
    })
}

// GetClassesByLevel godoc
// @Summary      Get classes by class level
// @Description  List classes belonging to a specific class level.
// @Tags         Academic
// @Produce      json
// @Param        classLevelId path string true "Class Level ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/level/{classLevelId} [get]
func (h *ClassHandler) GetClassesByLevel(c *gin.Context) {
    classLevelID := c.Param("classLevelId")
    ctx := c.Request.Context()

    classes, err := h.service.GetByLevel(ctx, classLevelID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "failed to fetch classes",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "success",
        "data":   classes,
        "count":  len(classes),
    })
}

// BulkDeleteClasses godoc
// @Summary      Bulk delete classes
// @Description  Delete multiple classes by IDs
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.ClassBulkDeleteRequest true "List of IDs to delete"
// @Success      200  {object}  map[string]interface{}  "message + deleted_count"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/bulk [delete]
func (h *ClassHandler) BulkDeleteClasses(c *gin.Context) {
    var req dto.ClassBulkDeleteRequest
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
    deletedCount, err := h.service.BulkDeleteClasses(ctx, req.IDs)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Failed to delete classes",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":        "success",
        "message":       "Classes deleted successfully",
        "deleted_count": deletedCount,
    })
}

// GetClassStats godoc
// @Summary      Get class statistics
// @Description  Get statistics about classes (total, active, by school, by level, etc.)
// @Description  Optional school_id filter - if not provided, returns stats for all schools
// @Tags         Academic
// @Produce      json
// @Param        school_id query string false "Filter by school ID"
// @Success      200  {object}  map[string]interface{}  "data (stats)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/stats [get]
func (h *ClassHandler) GetClassStats(c *gin.Context) {
    schoolID := c.Query("school_id")
    ctx := c.Request.Context()

    stats, err := h.service.GetClassStats(ctx, schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch class statistics",
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

// SearchClasses godoc
// @Summary      Search classes
// @Description  Search classes by class code or room number
// @Tags         Academic
// @Produce      json
// @Param        q query string true "Search query"
// @Param        school_id query string false "Filter by school ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200  {object}  dto.ClassListResponse
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /classes/search [get]
func (h *ClassHandler) SearchClasses(c *gin.Context) {
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

    resp, err := h.service.SearchClasses(ctx, query, schoolID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to search classes",
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

// type ClassHandler struct {
//     service *service.ClassService
// }

// func NewClassHandler(service *service.ClassService) *ClassHandler {
//     return &ClassHandler{service: service}
// }

// // CreateClass creates a new class
// func (h *ClassHandler) CreateClass(c *gin.Context) {
//     var req dto.CreateClassRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     class, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Class created successfully",
//         "data":    class,
//     })
// }

// // GetClass gets a class by ID
// func (h *ClassHandler) GetClass(c *gin.Context) {
//     id := c.Param("id")
    
//     class, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": class})
// }

// // GetClassesBySchool gets all classes for a school
// func (h *ClassHandler) GetClassesBySchool(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     classes, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": classes})
// }

// // GetClassesBySession gets all classes for a session
// func (h *ClassHandler) GetClassesBySession(c *gin.Context) {
//     sessionID := c.Param("sessionId")
    
//     classes, err := h.service.GetBySession(sessionID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": classes})
// }

// // GetClassesBySchoolAndSession gets all classes for a school and session
// func (h *ClassHandler) GetClassesBySchoolAndSession(c *gin.Context) {
//     schoolID := c.Param("schoolId")
//     sessionID := c.Param("sessionId")
    
//     classes, err := h.service.GetBySchoolAndSession(schoolID, sessionID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": classes})
// }

// // UpdateClass updates a class
// func (h *ClassHandler) UpdateClass(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateClassRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     class, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Class updated successfully",
//         "data":    class,
//     })
// }

// // DeleteClass deletes a class
// func (h *ClassHandler) DeleteClass(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Class deleted successfully"})
// }