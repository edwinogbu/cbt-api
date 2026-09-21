package middleware

import (
	"net/http"
	"strings"

	"cbt-api/internal/nodesync/service"

	"github.com/gin-gonic/gin"
)

const contextKeySchoolID = "nodesync_school_id"

// NodeAuthMiddleware authenticates a School Node's push/pull calls via
// "Authorization: Bearer <node api key>" - a separate credential space
// from the human-user JWT auth (middleware.AuthMiddleware), since a node
// is a long-lived server-to-server caller, not a signed-in person.
func NodeAuthMiddleware(svc *service.NodeSyncService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing node credential"})
			return
		}
		key := strings.TrimPrefix(header, "Bearer ")

		schoolID, _, err := svc.AuthenticateNode(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid node credential"})
			return
		}

		c.Set(contextKeySchoolID, schoolID)
		c.Next()
	}
}

func GetSchoolID(c *gin.Context) string {
	if v, ok := c.Get(contextKeySchoolID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
