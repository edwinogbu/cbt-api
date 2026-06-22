package handler

import (
	"net/http"
	"strconv"

	"cbt-api/internal/admin/dto"
	"cbt-api/internal/admin/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{service: svc}
}

// AssignTeacher godoc
// @Summary      Assign teacher to class
// @Description  Admin only – links a teacher to a class
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body dto.AssignTeacherRequest true "Assignment details"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/teachers/assign [post]
func (h *AdminHandler) AssignTeacher(c *gin.Context) {
	var req dto.AssignTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AssignTeacher(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher assigned to class successfully"})
}

// UnassignTeacher godoc
// @Summary      Remove teacher from class
// @Description  Admin only – sets teacher_id = NULL
// @Tags         Admin
// @Produce      json
// @Param        classId path string true "Class ID"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/teachers/unassign/{classId} [delete]
func (h *AdminHandler) UnassignTeacher(c *gin.Context) {
	classID := c.Param("classId")
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	if err := h.service.UnassignTeacher(classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher unassigned successfully"})
}

// ListUsers godoc
// @Summary      List all users (filter by role)
// @Description  Admin only – returns paginated list of users
// @Tags         Admin
// @Produce      json
// @Param        role query string false "Filter by role" Enums(admin,teacher,student,parent)
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search by name, email, username"
// @Success      200 {object} map[string]interface{} "data, meta"
// @Security     BearerAuth
// @Router       /admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var query dto.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 20
	}

	users, total, err := h.service.ListUsersByRole(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{
			"page":  query.Page,
			"limit": query.Limit,
			"total": total,
		},
	})
}

// ListStudents godoc
// @Summary      List all students (admin overview)
// @Description  Admin only – returns all students with class info
// @Tags         Admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search by admission no or name"
// @Success      200 {object} map[string]interface{} "data, meta"
// @Security     BearerAuth
// @Router       /admin/students [get]
func (h *AdminHandler) ListStudents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	students, total, err := h.service.ListAllStudents(page, limit, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": students,
		"meta": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"search": search,
		},
	})
}

// HardDeleteStudent godoc
// @Summary      Permanently delete a student (hard delete)
// @Description  Admin only – removes student record completely from database
// @Tags         Admin
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {object} map[string]interface{} "message"
// @Failure      400 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /admin/students/{id}/permanent [delete]
func (h *AdminHandler) HardDeleteStudent(c *gin.Context) {
	studentID := c.Param("id")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student id is required"})
		return
	}

	if err := h.service.HardDeleteStudent(studentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student permanently deleted"})
}


// ListClasses godoc
// @Summary      List all classes (admin overview)
// @Description  Admin only – returns all classes with level, arm, teacher details
// @Tags         Admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search by class code, level, or arm"
// @Success      200 {object} map[string]interface{} "data, meta"
// @Security     BearerAuth
// @Router       /admin/classes [get]
func (h *AdminHandler) ListClasses(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    search := c.Query("search")

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    classes, total, err := h.service.ListAllClasses(page, limit, search)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "data": classes,
        "meta": gin.H{
            "page":   page,
            "limit":  limit,
            "total":  total,
            "search": search,
        },
    })
}

// ListTeachers godoc
// @Summary      List all teachers (admin overview)
// @Description  Admin only – returns all teachers with personal details and assigned classes
// @Tags         Admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search by name, email, username"
// @Success      200 {object} map[string]interface{} "data, meta"
// @Security     BearerAuth
// @Router       /admin/teachers [get]
func (h *AdminHandler) ListTeachers(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    search := c.Query("search")

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    teachers, total, err := h.service.ListAllTeachers(page, limit, search)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "data": teachers,
        "meta": gin.H{
            "page":   page,
            "limit":  limit,
            "total":  total,
            "search": search,
        },
    })
}


// // ListClasses godoc
// // @Summary      List all classes (admin overview)
// // @Description  Admin only – returns all classes with school, level, arm, teacher details
// // @Tags         Admin
// // @Produce      json
// // @Param        page query int false "Page number" default(1)
// // @Param        limit query int false "Items per page" default(20)
// // @Param        search query string false "Search by class code, level, or arm"
// // @Success      200 {object} map[string]interface{} "data, meta"
// // @Security     BearerAuth
// // @Router       /admin/classes [get]
// func (h *AdminHandler) ListClasses(c *gin.Context) {
//     page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
//     limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
//     search := c.Query("search")

//     if page < 1 {
//         page = 1
//     }
//     if limit < 1 || limit > 100 {
//         limit = 20
//     }

//     classes, total, err := h.service.ListAllClasses(page, limit, search)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{
//         "data": classes,
//         "meta": gin.H{
//             "page":   page,
//             "limit":  limit,
//             "total":  total,
//             "search": search,
//         },
//     })
// }