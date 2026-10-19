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
        c.Set("school_id", claims["school_id"])
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

// PlatformOwnerOnly gates platform-wide endpoints (tenant registry,
// platform-controlled branding, LAN Setup Dashboard) to the exact
// "platform_owner" role claim from a verified JWT - deliberately NOT
// "admin" or "super_admin", which are school-scoped. RoleMiddleware does
// an exact string match against allowedRoles with no inheritance, so an
// admin or super_admin token is rejected here the same as any other role;
// there is no bypass by school membership, a supplied school/tenant ID,
// or any other client-controlled value - only the server-verified JWT
// role claim set at login/bootstrap time (see models.RolePlatformOwner).
func PlatformOwnerOnly() gin.HandlerFunc {
    return RoleMiddleware("platform_owner")
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

// GetSchoolID retrieves the authenticated user's school_id from the
// verified JWT claim set by AuthMiddleware - never from a client-supplied
// query param, header, or request body field. Returns "" for a user with
// no school (e.g. a models.RolePlatformOwner token, or an unauthenticated
// request where AuthMiddleware never ran).
func GetSchoolID(c *gin.Context) string {
    schoolID, _ := c.Get("school_id")
    if schoolID == nil {
        return ""
    }
    s, _ := schoolID.(string)
    return s
}

// IsPlatformOwner reports whether the authenticated request's verified
// role claim is platform_owner - the only role meant to act across every
// tenant rather than being confined to GetSchoolID's value.
func IsPlatformOwner(c *gin.Context) bool {
    return GetRole(c) == "platform_owner"
}

// ResolveSchoolID is the shared rule for every list/search/stats/create
// handler that currently reads a school_id from a client-supplied query
// param or request body field: for an authenticated platform_owner, that
// client-supplied value is honored as-is (possibly "" meaning "every
// school" for a list/filter, since platform_owner is deliberately not
// confined to one tenant); for every other role, it is discarded
// entirely and replaced with GetSchoolID(c) - a client-supplied
// school_id must never be able to independently establish access. Use
// this wherever a handler would otherwise do `c.Query("school_id")` or
// bind a SchoolID request field directly into a DB filter/write.
func ResolveSchoolID(c *gin.Context, clientSuppliedSchoolID string) string {
    if IsPlatformOwner(c) {
        return clientSuppliedSchoolID
    }
    return GetSchoolID(c)
}

// ResolveSchoolIDOrNoResults wraps ResolveSchoolID with the guard several
// repository queries in this codebase need: more than one list/stats/
// search query treats an EMPTY school_id filter as "no filter at all"
// (i.e. every school), not "no results" - so an empty string reaching
// those queries is unscoped, not merely unfiltered. An empty result from
// ResolveSchoolID is only ever legitimate for an authenticated
// platform_owner (who may deliberately want every school); for anyone
// else it means "no school to scope to" and the caller must return an
// empty/zero result directly, never pass "" into such a query. ok is
// false in exactly that case.
func ResolveSchoolIDOrNoResults(c *gin.Context, clientSuppliedSchoolID string) (schoolID string, ok bool) {
    resolved := ResolveSchoolID(c, clientSuppliedSchoolID)
    if resolved == "" && !IsPlatformOwner(c) {
        return "", false
    }
    return resolved, true
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