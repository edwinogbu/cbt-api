package middleware

import (
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

func LoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start)

        log.Printf("[%s] %s %s - %v",
            c.Request.Method,
            c.Request.URL.Path,
            c.ClientIP(),
            duration,
        )
    }
}

// RecoveryMiddleware recovers from panics and prevents server crashes
func RecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("🔥 PANIC RECOVERED: %v", err)
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "Internal server error",
                })
                c.Abort()
            }
        }()
        c.Next()
    }
}