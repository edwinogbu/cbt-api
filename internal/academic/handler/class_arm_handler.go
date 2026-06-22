package handler

import (
    "net/http"

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

// CreateClassArm godoc
// @Summary      Create a new class arm
// @Description  Add a class arm (e.g., "Section A") for a school.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateClassArmRequest true "Class arm details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms [post]
func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
    var req dto.CreateClassArmRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    arm, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Class arm created successfully",
        "data":    arm,
    })
}

// GetClassArm godoc
// @Summary      Get class arm by ID
// @Description  Retrieve a single class arm.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Arm ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/{id} [get]
func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
    id := c.Param("id")
    
    arm, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": arm})
}

// GetClassArms godoc
// @Summary      Get all class arms for a school
// @Description  List all class arms belonging to a school.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Success      200  {object}  map[string]interface{}  "data (list)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-class-arms/{schoolId} [get]
func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
    schoolID := c.Param("schoolId")
    
    arms, err := h.service.GetBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": arms})
}

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
// @Security     BearerAuth
// @Router       /class-arms/{id} [put]
func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateClassArmRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    arm, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Class arm updated successfully",
        "data":    arm,
    })
}

// DeleteClassArm godoc
// @Summary      Delete a class arm
// @Description  Soft‑delete a class arm.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Class Arm ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-arms/{id} [delete]
func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Class arm deleted successfully"})
}




// package handler

// import (
//     "net/http"

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

// func (h *ClassArmHandler) CreateClassArm(c *gin.Context) {
//     var req dto.CreateClassArmRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     arm, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Class arm created successfully",
//         "data":    arm,
//     })
// }

// func (h *ClassArmHandler) GetClassArm(c *gin.Context) {
//     id := c.Param("id")
    
//     arm, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": arm})
// }

// func (h *ClassArmHandler) GetClassArms(c *gin.Context) {
//     schoolID := c.Param("schoolId")
    
//     arms, err := h.service.GetBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": arms})
// }

// func (h *ClassArmHandler) UpdateClassArm(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateClassArmRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     arm, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Class arm updated successfully",
//         "data":    arm,
//     })
// }

// func (h *ClassArmHandler) DeleteClassArm(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Class arm deleted successfully"})
// }