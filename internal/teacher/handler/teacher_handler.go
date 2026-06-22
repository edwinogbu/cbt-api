package handler

import (
	"net/http"
	"strconv"
	"strings"

	"cbt-api/internal/middleware"
	"cbt-api/internal/teacher/dto"
	"cbt-api/internal/teacher/service"
	"github.com/gin-gonic/gin"
)

type TeacherHandler struct {
	service *service.TeacherService
}

func NewTeacherHandler(svc *service.TeacherService) *TeacherHandler {
	return &TeacherHandler{service: svc}
}

// CreateStudent handles single student creation
// @Summary      Create a single student
// @Description  Teacher creates a student account (user + student profile). Auto‑generates admission number, username, password, and returns complete student record with class, teacher, school, graduation year.
// @Tags         Teacher
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateStudentByTeacherRequest true "Student details (minimal: school_id, class_id, first_name, last_name)"
// @Success      201 {object} map[string]interface{} "message, data (CompleteStudentResponse)"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students [post]
func (h *TeacherHandler) CreateStudent(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	var req dto.CreateStudentByTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateStudent(&req, teacherID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Student created successfully. Complete record generated.",
		"data":    resp,
	})
}

// GetMyStudents lists students in teacher's class
// @Summary      List students in teacher's class
// @Tags         Teacher
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{} "data, meta"
// @Security     BearerAuth
// @Router       /teacher/students [get]
func (h *TeacherHandler) GetMyStudents(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	students, total, err := h.service.GetMyStudents(teacherID, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

// GetStudent returns a single student by ID (teacher's class only)
// @Summary      Get a single student by ID (teacher's class only)
// @Tags         Teacher
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {object} map[string]interface{} "data"
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students/{id} [get]
func (h *TeacherHandler) GetStudent(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	studentID := c.Param("id")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
		return
	}
	student, err := h.service.GetStudentByID(studentID, teacherID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": student})
}

// UpdateStudent updates student details (teacher's class only)
// @Summary      Update student details
// @Tags         Teacher
// @Accept       json
// @Produce      json
// @Param        id path string true "Student ID"
// @Param        request body dto.UpdateStudentByTeacherRequest true "Fields to update"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students/{id} [put]
func (h *TeacherHandler) UpdateStudent(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	studentID := c.Param("id")
	var req dto.UpdateStudentByTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateStudent(studentID, &req, teacherID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
}

// ResetPassword resets a student's password
// @Summary      Reset student password
// @Tags         Teacher
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {object} map[string]interface{} "data (new credentials)"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students/{id}/reset-password [post]
func (h *TeacherHandler) ResetPassword(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	studentID := c.Param("id")
	resp, err := h.service.ResetStudentPassword(studentID, teacherID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
		"data":    resp,
	})
}

// DeactivateStudent deactivates a student (set IsActive=false and status)
// @Summary      Deactivate a student
// @Tags         Teacher
// @Accept       json
// @Produce      json
// @Param        id path string true "Student ID"
// @Param        request body dto.DeactivateStudentRequest true "Reason for deactivation"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students/{id}/deactivate [post]
func (h *TeacherHandler) DeactivateStudent(c *gin.Context) {
	teacherID := middleware.GetUserID(c)
	studentID := c.Param("id")
	var req dto.DeactivateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.DeactivateStudent(studentID, teacherID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deactivated successfully"})
}

// BulkCreateStudents handles Excel file upload for bulk student creation
// @Summary      Bulk upload students via Excel
// @Tags         Teacher
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Excel file"
// @Success      200 {object} map[string]interface{} "summary"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /teacher/students/bulk [post]
func (h *TeacherHandler) BulkCreateStudents(c *gin.Context) {
	teacherID := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	filename := header.Filename
	if !strings.HasSuffix(filename, ".xlsx") && !strings.HasSuffix(filename, ".xls") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Excel files (.xlsx, .xls) are allowed"})
		return
	}

	created, errorsList, err := h.service.BulkCreateStudents(file, teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"message": "Bulk upload completed",
		"total":   len(created) + len(errorsList),
		"success": len(created),
		"failed":  len(errorsList),
		"data":    created,
	}
	if len(errorsList) > 0 {
		response["errors"] = errorsList
	}
	c.JSON(http.StatusOK, response)
}


// package handler

// import (
// 	"net/http"
// 	"strconv"
// 	"strings"

// 	"cbt-api/internal/middleware"
// 	"cbt-api/internal/teacher/dto"
// 	"cbt-api/internal/teacher/service"
// 	"github.com/gin-gonic/gin"
// )

// type TeacherHandler struct {
// 	service *service.TeacherService
// }

// func NewTeacherHandler(svc *service.TeacherService) *TeacherHandler {
// 	return &TeacherHandler{service: svc}
// }

// // CreateStudent – now returns complete student record
// func (h *TeacherHandler) CreateStudent(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	var req dto.CreateStudentByTeacherRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	resp, err := h.service.CreateStudent(&req, teacherID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusCreated, gin.H{
// 		"message": "Student created successfully. Complete record generated.",
// 		"data":    resp,
// 	})
// }

// // GetMyStudents (unchanged)
// func (h *TeacherHandler) GetMyStudents(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
// 	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}
// 	students, total, err := h.service.GetMyStudents(teacherID, page, limit)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"data": students,
// 		"meta": gin.H{
// 			"page":  page,
// 			"limit": limit,
// 			"total": total,
// 		},
// 	})
// }

// // GetStudent (unchanged)
// func (h *TeacherHandler) GetStudent(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	studentID := c.Param("id")
// 	if studentID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
// 		return
// 	}
// 	student, err := h.service.GetStudentByID(studentID, teacherID)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"data": student})
// }

// // UpdateStudent (unchanged)
// func (h *TeacherHandler) UpdateStudent(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	studentID := c.Param("id")
// 	var req dto.UpdateStudentByTeacherRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	if err := h.service.UpdateStudent(studentID, &req, teacherID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
// }

// // ResetPassword (unchanged)
// func (h *TeacherHandler) ResetPassword(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	studentID := c.Param("id")
// 	resp, err := h.service.ResetStudentPassword(studentID, teacherID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Password reset successfully",
// 		"data":    resp,
// 	})
// }

// // DeactivateStudent (unchanged)
// func (h *TeacherHandler) DeactivateStudent(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)
// 	studentID := c.Param("id")
// 	var req dto.DeactivateStudentRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	if err := h.service.DeactivateStudent(studentID, teacherID, req.Reason); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"message": "Student deactivated successfully"})
// }

// // BulkCreateStudents (unchanged – response already uses new slice type)
// func (h *TeacherHandler) BulkCreateStudents(c *gin.Context) {
// 	teacherID := middleware.GetUserID(c)

// 	file, header, err := c.Request.FormFile("file")
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
// 		return
// 	}
// 	defer file.Close()

// 	filename := header.Filename
// 	if !strings.HasSuffix(filename, ".xlsx") && !strings.HasSuffix(filename, ".xls") {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Excel files (.xlsx, .xls) are allowed"})
// 		return
// 	}

// 	created, errorsList, err := h.service.BulkCreateStudents(file, teacherID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	response := gin.H{
// 		"message": "Bulk upload completed",
// 		"total":   len(created) + len(errorsList),
// 		"success": len(created),
// 		"failed":  len(errorsList),
// 		"data":    created,
// 	}
// 	if len(errorsList) > 0 {
// 		response["errors"] = errorsList
// 	}
// 	c.JSON(http.StatusOK, response)
// }


// // package handler

// // import (
// // 	"net/http"
// // 	"strconv"
// // 	"strings"

// // 	"cbt-api/internal/middleware"
// // 	"cbt-api/internal/teacher/dto"
// // 	"cbt-api/internal/teacher/service"
// // 	"github.com/gin-gonic/gin"
// // )

// // type TeacherHandler struct {
// // 	service *service.TeacherService
// // }

// // func NewTeacherHandler(svc *service.TeacherService) *TeacherHandler {
// // 	return &TeacherHandler{service: svc}
// // }

// // // CreateStudent handles single student creation
// // // @Summary      Create a single student
// // // @Description  Teacher creates a student account (user + student profile). Returns username and generated password.
// // // @Tags         Teacher
// // // @Accept       json
// // // @Produce      json
// // // @Param        request body dto.CreateStudentByTeacherRequest true "Student details"
// // // @Success      201 {object} map[string]interface{} "message, data"
// // // @Failure      400 {object} map[string]interface{}
// // // @Failure      403 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students [post]
// // func (h *TeacherHandler) CreateStudent(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	var req dto.CreateStudentByTeacherRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}

// // 	resp, err := h.service.CreateStudent(&req, teacherID)
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusCreated, gin.H{
// // 		"message": "Student created successfully. Credentials generated.",
// // 		"data":    resp,
// // 	})
// // }

// // // GetMyStudents lists students in teacher's class
// // // @Summary      List students in teacher's class
// // // @Tags         Teacher
// // // @Produce      json
// // // @Param        page query int false "Page number" default(1)
// // // @Param        limit query int false "Items per page" default(20)
// // // @Success      200 {object} map[string]interface{} "data, meta"
// // // @Security     BearerAuth
// // // @Router       /teacher/students [get]
// // func (h *TeacherHandler) GetMyStudents(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
// // 	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}
// // 	students, total, err := h.service.GetMyStudents(teacherID, page, limit)
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, gin.H{
// // 		"data": students,
// // 		"meta": gin.H{
// // 			"page":  page,
// // 			"limit": limit,
// // 			"total": total,
// // 		},
// // 	})
// // }

// // // GetStudent returns a single student by ID (teacher's class only)
// // // @Summary      Get a single student by ID (teacher's class only)
// // // @Tags         Teacher
// // // @Produce      json
// // // @Param        id path string true "Student ID"
// // // @Success      200 {object} map[string]interface{} "data"
// // // @Failure      404 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students/{id} [get]
// // func (h *TeacherHandler) GetStudent(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	studentID := c.Param("id")
// // 	if studentID == "" {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": "student id required"})
// // 		return
// // 	}
// // 	student, err := h.service.GetStudentByID(studentID, teacherID)
// // 	if err != nil {
// // 		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, gin.H{"data": student})
// // }

// // // UpdateStudent updates student details (teacher's class only)
// // // @Summary      Update student details
// // // @Tags         Teacher
// // // @Accept       json
// // // @Produce      json
// // // @Param        id path string true "Student ID"
// // // @Param        request body dto.UpdateStudentByTeacherRequest true "Fields to update"
// // // @Success      200 {object} map[string]interface{} "message"
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students/{id} [put]
// // func (h *TeacherHandler) UpdateStudent(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	studentID := c.Param("id")
// // 	var req dto.UpdateStudentByTeacherRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	if err := h.service.UpdateStudent(studentID, &req, teacherID); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
// // }

// // // ResetPassword resets a student's password
// // // @Summary      Reset student password
// // // @Tags         Teacher
// // // @Produce      json
// // // @Param        id path string true "Student ID"
// // // @Success      200 {object} map[string]interface{} "data (new credentials)"
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students/{id}/reset-password [post]
// // func (h *TeacherHandler) ResetPassword(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	studentID := c.Param("id")
// // 	resp, err := h.service.ResetStudentPassword(studentID, teacherID)
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, gin.H{
// // 		"message": "Password reset successfully",
// // 		"data":    resp,
// // 	})
// // }

// // // DeactivateStudent deactivates a student (set IsActive=false and status)
// // // @Summary      Deactivate a student
// // // @Tags         Teacher
// // // @Accept       json
// // // @Produce      json
// // // @Param        id path string true "Student ID"
// // // @Param        request body dto.DeactivateStudentRequest true "Reason for deactivation"
// // // @Success      200 {object} map[string]interface{} "message"
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students/{id}/deactivate [post]
// // func (h *TeacherHandler) DeactivateStudent(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)
// // 	studentID := c.Param("id")
// // 	var req dto.DeactivateStudentRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	if err := h.service.DeactivateStudent(studentID, teacherID, req.Reason); err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // 		return
// // 	}
// // 	c.JSON(http.StatusOK, gin.H{"message": "Student deactivated successfully"})
// // }

// // // BulkCreateStudents handles Excel file upload for bulk student creation
// // // @Summary      Bulk upload students via Excel
// // // @Tags         Teacher
// // // @Accept       multipart/form-data
// // // @Produce      json
// // // @Param        file formData file true "Excel file"
// // // @Success      200 {object} map[string]interface{} "summary"
// // // @Failure      400 {object} map[string]interface{}
// // // @Security     BearerAuth
// // // @Router       /teacher/students/bulk [post]
// // func (h *TeacherHandler) BulkCreateStudents(c *gin.Context) {
// // 	teacherID := middleware.GetUserID(c)

// // 	// Get uploaded file
// // 	file, header, err := c.Request.FormFile("file")
// // 	if err != nil {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
// // 		return
// // 	}
// // 	defer file.Close()

// // 	// Validate file extension
// // 	filename := header.Filename
// // 	if !strings.HasSuffix(filename, ".xlsx") && !strings.HasSuffix(filename, ".xls") {
// // 		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Excel files (.xlsx, .xls) are allowed"})
// // 		return
// // 	}

// // 	// Process bulk creation
// // 	created, errorsList, err := h.service.BulkCreateStudents(file, teacherID)
// // 	if err != nil {
// // 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// // 		return
// // 	}

// // 	response := gin.H{
// // 		"message": "Bulk upload completed",
// // 		"total":   len(created) + len(errorsList),
// // 		"success": len(created),
// // 		"failed":  len(errorsList),
// // 		"data":    created,
// // 	}
// // 	if len(errorsList) > 0 {
// // 		response["errors"] = errorsList
// // 	}
// // 	c.JSON(http.StatusOK, response)
// // }