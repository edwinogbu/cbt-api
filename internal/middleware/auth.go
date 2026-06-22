package middleware

import (
    "net/http"
    "strings"

    "cbt-api/pkg/utils"

    "github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT and stores user info in context
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims, err := utils.ValidateJWT(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims["user_id"])
        c.Set("email", claims["email"])
        c.Set("role", claims["role"])
        c.Set("session_id", claims["session_id"])
        c.Next()
    }
}

// RoleMiddleware is a generic role checker
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, exists := c.Get("role")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
            c.Abort()
            return
        }

        roleStr := userRole.(string)
        for _, role := range allowedRoles {
            if roleStr == role {
                c.Next()
                return
            }
        }

        c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
        c.Abort()
    }
}

// Convenience role checkers (use after AuthMiddleware)
func AdminOnly() gin.HandlerFunc {
    return RoleMiddleware("admin")
}

func TeacherOnly() gin.HandlerFunc {
    return RoleMiddleware("teacher")
}

func StudentOnly() gin.HandlerFunc {
    return RoleMiddleware("student")
}

func ParentOnly() gin.HandlerFunc {
    return RoleMiddleware("parent")
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) string {
    userID, _ := c.Get("user_id")
    if userID == nil {
        return ""
    }
    return userID.(string)
}

// GetEmail retrieves email from context
func GetEmail(c *gin.Context) string {
    email, _ := c.Get("email")
    if email == nil {
        return ""
    }
    return email.(string)
}

// GetRole retrieves role from context
func GetRole(c *gin.Context) string {
    role, _ := c.Get("role")
    if role == nil {
        return ""
    }
    return role.(string)
}

// GetSessionID retrieves session ID from context
func GetSessionID(c *gin.Context) string {
    sessionID, _ := c.Get("session_id")
    if sessionID == nil {
        return ""
    }
    return sessionID.(string)
}




// package middleware

// import (
//     "net/http"
//     "strings"

//     "cbt-api/pkg/utils"

//     "github.com/gin-gonic/gin"
// )

// func AuthMiddleware() gin.HandlerFunc {
//     return func(c *gin.Context) {
//         authHeader := c.GetHeader("Authorization")
//         if authHeader == "" {
//             c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
//             c.Abort()
//             return
//         }

//         parts := strings.Split(authHeader, " ")
//         if len(parts) != 2 || parts[0] != "Bearer" {
//             c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
//             c.Abort()
//             return
//         }

//         tokenString := parts[1]
//         claims, err := utils.ValidateJWT(tokenString)
//         if err != nil {
//             c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
//             c.Abort()
//             return
//         }

//         c.Set("user_id", claims["user_id"])
//         c.Set("email", claims["email"])
//         c.Set("role", claims["role"])
//         c.Set("session_id", claims["session_id"])
//         c.Next()
//     }
// }

// func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
//     return func(c *gin.Context) {
//         userRole, exists := c.Get("role")
//         if !exists {
//             c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
//             c.Abort()
//             return
//         }

//         roleStr := userRole.(string)
//         for _, role := range allowedRoles {
//             if roleStr == role {
//                 c.Next()
//                 return
//             }
//         }

//         c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
//         c.Abort()
//     }
// }

// // GetUserID retrieves user ID from context
// func GetUserID(c *gin.Context) string {
//     userID, _ := c.Get("user_id")
//     return userID.(string)
// }

// // GetEmail retrieves email from context
// func GetEmail(c *gin.Context) string {
//     email, _ := c.Get("email")
//     return email.(string)
// }

// // GetRole retrieves role from context
// func GetRole(c *gin.Context) string {
//     role, _ := c.Get("role")
//     return role.(string)
// }

// // GetSessionID retrieves session ID from context
// func GetSessionID(c *gin.Context) string {
//     sessionID, _ := c.Get("session_id")
//     if sessionID == nil {
//         return ""
//     }
//     return sessionID.(string)
// }