package handler

import (
    "net/http"

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
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    level, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
    
    level, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": level})
}

// GetClassLevels godoc
// @Summary      Get all class levels for a school
// @Description  List all class levels belonging to a school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-class-levels/{schoolId} [get]
func (h *ClassLevelHandler) GetClassLevels(c *gin.Context) {
    schoolID := c.Param("schoolId")
    
    levels, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": levels})
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
    
    var req dto.UpdateClassLevelRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    level, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class level updated successfully",
        "data":    level,
    })
}

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
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Class level deleted successfully"})
}



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

// func (h *ClassLevelHandler) GetClassLevel(c *gin.Context) {
//     id := c.Param("id")
    
//     level, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": level})
// }

// func (h *ClassLevelHandler) GetClassLevels(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     levels, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": levels})
// }

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

// func (h *ClassLevelHandler) DeleteClassLevel(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Class level deleted successfully"})
// }