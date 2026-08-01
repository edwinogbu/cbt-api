// middleware/student_context.go

package middleware

import (
    "net/http"
    
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"  
    "github.com/gin-gonic/gin"
)

func StudentContextMiddleware(studentRepo *repository.StudentRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := GetUserID(c)
        if userID == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        student, err := studentRepo.FindByUserID(userID)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "student profile not found"})
            return
        }

        // Set student context - these values are stored in Gin context
        c.Set("student", student)
        c.Set("student_id", student.ID)
        c.Set("class_id", student.ClassID)
        c.Set("school_id", student.SchoolID)
        
        c.Next()
    }
}

// Helper functions to get student data from Gin context
func GetStudent(c *gin.Context) *models.Student {
    if val, exists := c.Get("student"); exists {
        if student, ok := val.(*models.Student); ok {
            return student
        }
    }
    return nil
}

func GetStudentID(c *gin.Context) string {
    if val, exists := c.Get("student_id"); exists {
        if id, ok := val.(string); ok {
            return id
        }
    }
    return ""
}

func GetStudentClassID(c *gin.Context) string {
    if val, exists := c.Get("class_id"); exists {
        if id, ok := val.(string); ok {
            return id
        }
    }
    return ""
}

func GetStudentSchoolID(c *gin.Context) string {
    if val, exists := c.Get("school_id"); exists {
        if id, ok := val.(string); ok {
            return id
        }
    }
    return ""
}


// // middleware/student_context.go
// package middleware

// import (
//     "net/http"
    
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"  
//     "github.com/gin-gonic/gin"
// )

// func StudentContextMiddleware(studentRepo *repository.StudentRepository) gin.HandlerFunc {
//     return func(c *gin.Context) {
//         userID := GetUserID(c)
//         if userID == "" {
//             c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//             return
//         }

//         // ✅ FIX: Remove context - only pass userID
//         student, err := studentRepo.FindByUserID(userID)  // ← Only 1 argument
//         if err != nil {
//             c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "student profile not found"})
//             return
//         }

//         // Set student context
//         c.Set("student", student)
//         c.Set("student_id", student.ID)
//         c.Set("class_id", student.ClassID)
//         c.Set("school_id", student.SchoolID)
        
//         c.Next()
//     }
// }

// // Helper functions to get student data
// func GetStudent(c *gin.Context) *models.Student {
//     if val, exists := c.Get("student"); exists {
//         if student, ok := val.(*models.Student); ok {
//             return student
//         }
//     }
//     return nil
// }

// func GetStudentID(c *gin.Context) string {
//     if val, exists := c.Get("student_id"); exists {
//         if id, ok := val.(string); ok {
//             return id
//         }
//     }
//     return ""
// }

// func GetStudentClassID(c *gin.Context) string {
//     if val, exists := c.Get("class_id"); exists {
//         if id, ok := val.(string); ok {
//             return id
//         }
//     }
//     return ""
// }

// func GetStudentSchoolID(c *gin.Context) string {
//     if val, exists := c.Get("school_id"); exists {
//         if id, ok := val.(string); ok {
//             return id
//         }
//     }
//     return ""
// }

