package handler

import (
	"context"
	"os"
	"net/http"
	"time"

	"cbt-api/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// serverStartedAt is set once, at package init (this process's own
// start) - so GetSystemHealth reports real process uptime, not anything
// derived from client input or the database.
var serverStartedAt = time.Now()

type componentHealth struct {
	Connected bool   `json:"connected"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type systemHealthResponse struct {
	NodeName      string          `json:"node_name"`
	Environment   string          `json:"environment"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	ServerTime    string          `json:"server_time"`
	Database      componentHealth `json:"database"`
	Redis         componentHealth `json:"redis"`
}

// GetSystemHealth reports this node's own live operational status via
// real, bounded checks against the actual database and Redis connections
// this process holds - never an assumed or cached "ok". Built for the
// LAN Setup Dashboard's health-monitoring view (Master Implementation
// Plan section 1: "guided setup... health monitoring"), so a school
// admin running this node on their own LAN, with no internet, can see at
// a glance whether their own node's dependencies are actually up, and
// how long it has been running since its last restart.
//
// Deliberately NOT included here: a "connected clients" count. This
// backend has no session-tracking store (no Redis-backed or DB-backed
// session table - the JWT's session_id claim is embedded in the token
// for audit logging only, never persisted anywhere queryable), so there
// is no real number to report. Fabricating one would violate this
// engagement's own "do not fabricate... completion claims" rule; adding
// real session tracking to support it is a separate, larger piece of
// work, not a one-line addition to a read-only health endpoint.
func GetSystemHealth(c *gin.Context) {
	resp := systemHealthResponse{
		NodeName:      envOrDefault("APP_NAME", "CBT API"),
		Environment:   envOrDefault("ENVIRONMENT", "development"),
		UptimeSeconds: int64(time.Since(serverStartedAt).Seconds()),
		ServerTime:    time.Now().UTC().Format(time.RFC3339),
		Database:      checkDatabaseHealth(),
		Redis:         checkRedisHealth(),
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func checkDatabaseHealth() componentHealth {
	start := time.Now()
	sqlDB, err := database.DB.DB()
	if err != nil {
		return componentHealth{Connected: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return componentHealth{Connected: false, LatencyMS: time.Since(start).Milliseconds(), Error: err.Error()}
	}
	return componentHealth{Connected: true, LatencyMS: time.Since(start).Milliseconds()}
}

// checkRedisHealth mirrors cmd/server/main.go's own Redis Queue
// initialization precedence (REDIS_URL, else REDIS_ADDR, else
// localhost:6379) so this health check reports on the same Redis this
// process actually connects to for its AI job queue - not a different
// config surface (config.go's RedisConfig reads REDIS_HOST/REDIS_PORT,
// which main.go's actual queue init never reads).
func checkRedisHealth() componentHealth {
	var addr string
	var opts *redis.Options
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		var err error
		opts, err = redis.ParseURL(redisURL)
		if err != nil {
			return componentHealth{Connected: false, Error: "invalid REDIS_URL: " + err.Error()}
		}
	} else {
		addr = envOrDefault("REDIS_ADDR", "localhost:6379")
		opts = &redis.Options{Addr: addr}
	}

	client := redis.NewClient(opts)
	defer client.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return componentHealth{Connected: false, LatencyMS: time.Since(start).Milliseconds(), Error: err.Error()}
	}
	return componentHealth{Connected: true, LatencyMS: time.Since(start).Milliseconds()}
}
