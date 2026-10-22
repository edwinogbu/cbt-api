package middleware

import (
    "github.com/gin-gonic/gin"
)

// CORSMiddleware allows every origin to read responses, with no
// credentials mode: this API is pure Bearer-token auth (no code in
// this repo ever sets a cookie - grep "SetCookie"/"c.Cookie" comes back
// empty), so there is nothing a cross-origin page could piggyback on by
// having a browser attach credentials automatically. Access-Control-
// Allow-Credentials: true combined with a wildcard origin used to be
// set here too; browsers already refuse that exact combination
// (the Fetch spec forbids a wildcard origin for a credentialed
// request), so it was a dead, contradictory header - the textbook CORS
// misconfiguration a security scanner always flags - not a working
// capability for any real client. Removed rather than fixed, since
// nothing here ever needs cookie-based credentials.
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}