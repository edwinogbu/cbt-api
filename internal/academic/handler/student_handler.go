package handler

import (
    "net/http"
    "strconv"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/service"

    "github.com/gin-gonic/gin"
)

type StudentHandler struct {
    service *service.StudentService
}

func NewStudentHandler(service *service.StudentService) *StudentHandler {
    return &StudentHandler{service: service}
}

// CreateStudent godoc
// @Summary      Create a new student
// @Description  Associate a user as a student in a school and class.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateStudentRequest true "Student details"
// @Success      201  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /students [post]
func (h *StudentHandler) CreateStudent(c *gin.Context) {
    var req dto.CreateStudentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    student, err := h.service.Create(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "Student created successfully",
        "data":    student,
    })
}

// GetStudent godoc
// @Summary      Get student by ID
// @Description  Retrieve a single student record.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /students/{id} [get]
func (h *StudentHandler) GetStudent(c *gin.Context) {
    id := c.Param("id")
    
    student, err := h.service.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": student})
}

// GetStudentByUser godoc
// @Summary      Get student by user ID
// @Description  Find student record using the associated user ID.
// @Tags         Academic
// @Produce      json
// @Param        userId path string true "User ID"
// @Success      200  {object}  map[string]interface{}  "data"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /user-student/{userId} [get]
func (h *StudentHandler) GetStudentByUser(c *gin.Context) {
    userID := c.Param("userId")
    
    student, err := h.service.GetByUserID(userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": student})
}

// GetStudentsBySchool godoc
// @Summary      Get all students in a school
// @Description  List students belonging to a school, with pagination.
// @Tags         Academic
// @Produce      json
// @Param        schoolId path string true "School ID"
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20)"
// @Success      200  {object}  map[string]interface{}  "data (list), meta"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /school-students/{schoolId} [get]
func (h *StudentHandler) GetStudentsBySchool(c *gin.Context) {
    schoolID := c.Param("schoolId")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    students, total, err := h.service.GetBySchool(schoolID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": students,
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// GetStudentsByClass godoc
// @Summary      Get all students in a class
// @Description  List students belonging to a class, with pagination.
// @Tags         Academic
// @Produce      json
// @Param        classId path string true "Class ID"
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20)"
// @Success      200  {object}  map[string]interface{}  "data (list), meta"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /class-students/{classId} [get]
func (h *StudentHandler) GetStudentsByClass(c *gin.Context) {
    classID := c.Param("classId")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    students, total, err := h.service.GetByClass(classID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": students,
        "meta": gin.H{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// UpdateStudent godoc
// @Summary      Update a student
// @Description  Modify a student's details (e.g., admission number, class).
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        id path string true "Student ID"
// @Param        request body dto.UpdateStudentRequest true "Fields to update"
// @Success      200  {object}  map[string]interface{}  "message + data"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /students/{id} [put]
func (h *StudentHandler) UpdateStudent(c *gin.Context) {
    id := c.Param("id")
    
    var req dto.UpdateStudentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    student, err := h.service.Update(id, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Student updated successfully",
        "data":    student,
    })
}

// DeleteStudent godoc
// @Summary      Delete a student
// @Description  Soft‑delete a student record.
// @Tags         Academic
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /students/{id} [delete]
func (h *StudentHandler) DeleteStudent(c *gin.Context) {
    id := c.Param("id")
    
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Student deleted successfully"})
}

// TransferClass godoc
// @Summary      Transfer a student to another class
// @Description  Move a student from their current class to a new class.
// @Tags         Academic
// @Accept       json
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        request body object true "new_class_id"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /students/{studentId}/transfer [post]
func (h *StudentHandler) TransferClass(c *gin.Context) {
    studentID := c.Param("studentId")
    
    var req struct {
        NewClassID string `json:"new_class_id" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := h.service.TransferClass(studentID, req.NewClassID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Student transferred successfully"})
}



// package handler

// import (
//     "net/http"
//     "strconv"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/service"

//     "github.com/gin-gonic/gin"
// )

// type StudentHandler struct {
//     service *service.StudentService
// }

// func NewStudentHandler(service *service.StudentService) *StudentHandler {
//     return &StudentHandler{service: service}
// }

// // CreateStudent creates a new student
// func (h *StudentHandler) CreateStudent(c *gin.Context) {
//     var req dto.CreateStudentRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     student, err := h.service.Create(&req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Student created successfully",
//         "data":    student,
//     })
// }

// // GetStudent gets a student by ID
// func (h *StudentHandler) GetStudent(c *gin.Context) {
//     id := c.Param("id")
    
//     student, err := h.service.GetByID(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": student})
// }

// // GetStudentByUser gets a student by user ID
// func (h *StudentHandler) GetStudentByUser(c *gin.Context) {
//     userID := c.Param("userId")
    
//     student, err := h.service.GetByUserID(userID)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"data": student})
// }

// // GetStudentsBySchool gets all students in a school
// func (h *StudentHandler) GetStudentsBySchool(c *gin.Context) {
//     schoolID := c.Param("schoolId")
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
//     students, total, err := h.service.GetBySchool(schoolID, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "data": students,
//         "meta": gin.H{
//             "page":  page,
//             "limit": limit,
//             "total": total,
//         },
//     })
// }

// // GetStudentsByClass gets all students in a class
// func (h *StudentHandler) GetStudentsByClass(c *gin.Context) {
//     classID := c.Param("classId")
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
//     students, total, err := h.service.GetByClass(classID, page, limit)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "data": students,
//         "meta": gin.H{
//             "page":  page,
//             "limit": limit,
//             "total": total,
//         },
//     })
// }

// // UpdateStudent updates a student
// func (h *StudentHandler) UpdateStudent(c *gin.Context) {
//     id := c.Param("id")
    
//     var req dto.UpdateStudentRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     student, err := h.service.Update(id, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Student updated successfully",
//         "data":    student,
//     })
// }

// // DeleteStudent deletes a student
// func (h *StudentHandler) DeleteStudent(c *gin.Context) {
//     id := c.Param("id")
    
//     if err := h.service.Delete(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Student deleted successfully"})
// }

// // TransferClass transfers a student to another class
// func (h *StudentHandler) TransferClass(c *gin.Context) {
//     studentID := c.Param("studentId")
    
//     var req struct {
//         NewClassID string `json:"new_class_id" binding:"required"`
//     }
    
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     if err := h.service.TransferClass(studentID, req.NewClassID); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
    
//     c.JSON(http.StatusOK, gin.H{"message": "Student transferred successfully"})
// }