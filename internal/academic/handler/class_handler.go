package handler

import (
    "net/http"

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