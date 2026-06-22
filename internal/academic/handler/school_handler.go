package handler

import (
    "net/http"
    "strconv"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type SchoolHandler struct {
    service *service.SchoolService
}

func NewSchoolHandler(service *service.SchoolService) *SchoolHandler {
    return &SchoolHandler{service: service}
}

// CreateSchool godoc
// @Summary Create a new school
// @Tags Schools
// @Accept json
// @Produce json
// @Param request body dto.CreateSchoolRequest true "School details"
// @Success 201 {object} map[string]interface{}
// @Router /schools [post]
func (h *SchoolHandler) CreateSchool(c *gin.Context) {
    var req dto.CreateSchoolRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    school, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "School created successfully",
        "data":    school,
    })
}

// GetSchool godoc
// @Summary Get school by ID
// @Tags Schools
// @Produce json
// @Param id path string true "School ID"
// @Success 200 {object} map[string]interface{}
// @Router /schools/{id} [get]
func (h *SchoolHandler) GetSchool(c *gin.Context) {
    id := c.Param("id")
    
    school, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": school})
}

// GetAllSchools godoc
// @Summary Get all schools
// @Tags Schools
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /schools [get]
func (h *SchoolHandler) GetAllSchools(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 10
    }
    
    schools, total, err := h.service.GetAll(page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": schools,
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// UpdateSchool godoc
// @Summary Update school
// @Tags Schools
// @Accept json
// @Produce json
// @Param id path string true "School ID"
// @Param request body dto.UpdateSchoolRequest true "School details"
// @Success 200 {object} map[string]interface{}
// @Router /schools/{id} [put]
func (h *SchoolHandler) UpdateSchool(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateSchoolRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    school, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "School updated successfully",
        "data":    school,
    })
}

// DeleteSchool godoc
// @Summary Delete school
// @Tags Schools
// @Produce json
// @Param id path string true "School ID"
// @Success 200 {object} map[string]interface{}
// @Router /schools/{id} [delete]
func (h *SchoolHandler) DeleteSchool(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "School deleted successfully"})
}



// package handler

// import (
//     "net/http"
//     "strconv"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type SchoolHandler struct {
//     service *service.SchoolService
// }

// func NewSchoolHandler(service *service.SchoolService) *SchoolHandler {
//     return &SchoolHandler{service: service}
// }

// // CreateSchool godoc
// // @Summary Create a new school
// // @Tags Schools
// // @Accept json
// // @Produce json
// // @Param request body dto.CreateSchoolRequest true "School details"
// // @Success 201 {object} map[string]interface{}
// // @Router /api/v1/schools [post]
// func (h *SchoolHandler) CreateSchool(c *gin.Context) {
//     var req dto.CreateSchoolRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     school, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "School created successfully",
//         "data":    school,
//     })
// }

// // GetSchool godoc
// // @Summary Get school by ID
// // @Tags Schools
// // @Produce json
// // @Param id path string true "School ID"
// // @Success 200 {object} map[string]interface{}
// // @Router /api/v1/schools/{id} [get]
// func (h *SchoolHandler) GetSchool(c *gin.Context) {
//     id := c.Param("id")
    
//     school, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": school})
// }

// // GetAllSchools godoc
// // @Summary Get all schools
// // @Tags Schools
// // @Produce json
// // @Param page query int false "Page number"
// // @Param limit query int false "Items per page"
// // @Success 200 {object} map[string]interface{}
// // @Router /api/v1/schools [get]
// func (h *SchoolHandler) GetAllSchools(c *gin.Context) {
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    
//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 10
//     }
    
//     schools, total, err := h.service.GetAll(page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "data": schools,
//         "meta": gin.H{
//             "page":  page,
//             "limit": limit,
//             "total": total,
//         },
//     })
// }

// // UpdateSchool godoc
// // @Summary Update school
// // @Tags Schools
// // @Accept json
// // @Produce json
// // @Param id path string true "School ID"
// // @Param request body dto.UpdateSchoolRequest true "School details"
// // @Success 200 {object} map[string]interface{}
// // @Router /api/v1/schools/{id} [put]
// func (h *SchoolHandler) UpdateSchool(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateSchoolRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     school, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "School updated successfully",
//         "data":    school,
//     })
// }

// // DeleteSchool godoc
// // @Summary Delete school
// // @Tags Schools
// // @Produce json
// // @Param id path string true "School ID"
// // @Success 200 {object} map[string]interface{}
// // @Router /api/v1/schools/{id} [delete]
// func (h *SchoolHandler) DeleteSchool(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "School deleted successfully"})
// }