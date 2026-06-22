package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cbt-api/api/routes"
	"cbt-api/config"
	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/engine_worker"
	"cbt-api/internal/ai/providers"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/middleware"
	"cbt-api/pkg/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	// "github.com/joho/godototenv"  // ✅ ADD THIS IMPORT
    "github.com/joho/godotenv"
	_ "cbt-api/api/docs"
)

func main() {
	// ✅ LOAD .env FIRST - BEFORE ANYTHING ELSE
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Warning: .env file not found, using environment variables")
	} else {
		log.Println("✅ .env file loaded successfully")
	}

	// ✅ DEBUG: Print loaded API keys (masked)
	log.Println("=== ENVIRONMENT VARIABLES ===")
	log.Printf("GEMINI_API_KEY: %s", maskKey(os.Getenv("GEMINI_API_KEY")))
	log.Printf("OPENAI_API_KEY: %s", maskKey(os.Getenv("OPENAI_API_KEY")))
	log.Printf("ANTHROPIC_API_KEY: %s", maskKey(os.Getenv("ANTHROPIC_API_KEY")))
	log.Println("==============================")

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	database.ConnectDB()

	// Set Gin mode based on environment
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// ============================================
	// AI INFRASTRUCTURE INITIALIZATION
	// ============================================

	// 1. Redis Queue
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisQueue := queue.NewRedisQueue(redisAddr)
	log.Printf("✅ Redis Queue initialized at %s", redisAddr)

	// 2. AI Providers (with graceful fallback)
	providerRouter := providers.NewRouter()
	
	// Register providers
	openAI := providers.NewOpenAIProvider()
	anthropic := providers.NewAnthropicProvider()
	gemini := providers.NewGeminiProvider()
	fallback := providers.NewFallbackProvider()
	
	providerRouter.Register(openAI)
	providerRouter.Register(anthropic)
	providerRouter.Register(gemini)
	providerRouter.Register(fallback)

	log.Printf("✅ Registered AI providers: %v", providerRouter.List())

	// 3. Engine Configuration
	engineConfig := engine.DefaultConfig().
		WithRefinement(2).
		WithThreshold(70).
		WithTimeout(90)

	// 4. Exam Engine
	examEngine := engine.NewEngine(providerRouter, engineConfig)
	log.Println("✅ Exam Engine initialized")

	// 5. AI Worker (runs in background)
	aiWorker := engine_worker.NewAIWorker(redisQueue, examEngine, database.DB)
	go func() {
		log.Println("🚀 Starting AI Worker...")
		aiWorker.Start()
	}()
	log.Println("✅ AI Worker started successfully")

	// ============================================
	// GIN ROUTER SETUP
	// ============================================

	// Setup router
	r := gin.Default()

	// Global middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RecoveryMiddleware())

	// Setup all routes with AI infrastructure
	routes.SetupRoutes(r, redisQueue, examEngine)

	// Print all available routes (optional)
	routes.PrintRoutes()

	// ============================================
	// SERVER STARTUP
	// ============================================

	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("========================================")
	log.Printf("🚀 CBT API Server starting on port %s", port)
	log.Printf("📚 Environment: %s", cfg.Server.Environment)
	log.Printf("🤖 AI Providers: %v", providerRouter.List())
	log.Printf("🔄 Redis Queue: %s", redisAddr)
	log.Printf("========================================")
	log.Printf("✅ Server is ready to accept requests")
	log.Printf("📖 Swagger UI available at http://localhost:%s/swagger/index.html", port)
	log.Printf("========================================")

	// Start server with graceful shutdown
	server := r

	go func() {
		if err := server.Run(":" + port); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// ============================================
	// GRACEFUL SHUTDOWN
	// ============================================

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server gracefully...")

	// Close Redis connection
	if err := redisQueue.Close(); err != nil {
		log.Printf("⚠️ Error closing Redis connection: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}

// Helper function to mask API keys for logging
func maskKey(key string) string {
	if key == "" {
		return "NOT SET"
	}
	if len(key) < 10 {
		return "INVALID (too short)"
	}
	if len(key) < 20 {
		return key[:5] + "..." + key[len(key)-3:]
	}
	return key[:10] + "..." + key[len(key)-5:]
}




// package main

// import (
// 	// "context"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"cbt-api/api/routes"
// 	"cbt-api/config"
// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/ai/engine_worker"
// 	"cbt-api/internal/ai/providers"
// 	"cbt-api/internal/ai/queue"
// 	"cbt-api/internal/middleware"
// 	"cbt-api/pkg/database"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"

// 	// Swagger documentation (auto‑generated)
// 	_ "cbt-api/api/docs"
// )

// // @title           CBT API
// // @version         1.0
// // @description     Computer‑Based Testing API with subscriptions, academic management, and CBT modules.
// // @termsOfService  http://swagger.io/terms/

// // @contact.name   API Support
// // @contact.email  support@cbt-api.com

// // @license.name  MIT
// // @license.url   https://opensource.org/licenses/MIT

// // @host      localhost:8080
// // @BasePath  /api/v1

// // @securityDefinitions.apikey BearerAuth
// // @in header
// // @name Authorization
// // @description Enter **Bearer &lt;token&gt;**. Example: `Bearer eyJhbGciOiJIUzI1NiIs...`

// func main() {
// 	// Load configuration
// 	cfg := config.LoadConfig()

// 	// Initialize database
// 	database.ConnectDB()

// 	// Set Gin mode based on environment
// 	if cfg.Server.IsProduction() {
// 		gin.SetMode(gin.ReleaseMode)
// 	}

// 	// ============================================
// 	// AI INFRASTRUCTURE INITIALIZATION
// 	// ============================================

// 	// 1. Redis Queue
// 	redisAddr := os.Getenv("REDIS_ADDR")
// 	if redisAddr == "" {
// 		redisAddr = "localhost:6379"
// 	}
// 	redisQueue := queue.NewRedisQueue(redisAddr)

// 	// 2. AI Providers (with graceful fallback)
// 	providerRouter := providers.NewRouter()
// 	providerRouter.Register(providers.NewOpenAIProvider())
// 	providerRouter.Register(providers.NewAnthropicProvider())
// 	providerRouter.Register(providers.NewGeminiProvider())
// 	providerRouter.Register(providers.NewFallbackProvider()) // Always available

// 	log.Printf("✅ Registered AI providers: %v", providerRouter.List())

// 	// 3. Engine Configuration
// 	engineConfig := engine.DefaultConfig().
// 		WithRefinement(2).
// 		WithThreshold(70).
// 		WithTimeout(90)

// 	// 4. Exam Engine
// 	examEngine := engine.NewEngine(providerRouter, engineConfig)

// 	// 5. AI Worker (runs in background)
// 	aiWorker := engine_worker.NewAIWorker(redisQueue, examEngine, database.DB)
// 	go aiWorker.Start()

// 	log.Println("✅ AI Worker started successfully")

// 	// ============================================
// 	// GIN ROUTER SETUP
// 	// ============================================

// 	// Setup router
// 	r := gin.Default()

// 	// Global middleware
// 	r.Use(cors.New(cors.Config{
// 		AllowOrigins:     []string{"*"},
// 		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// 		ExposeHeaders:    []string{"Content-Length"},
// 		AllowCredentials: true,
// 		MaxAge:           12 * time.Hour,
// 	}))
// 	r.Use(middleware.LoggerMiddleware())
// 	r.Use(middleware.RecoveryMiddleware())

// 	// Setup all routes with AI infrastructure
// 	routes.SetupRoutes(r, redisQueue, examEngine)

// 	// Print all available routes (optional)
// 	routes.PrintRoutes()

// 	// ============================================
// 	// SERVER STARTUP
// 	// ============================================

// 	port := cfg.Server.Port
// 	if port == "" {
// 		port = "8080"
// 	}

// 	log.Printf("========================================")
// 	log.Printf("🚀 CBT API Server starting on port %s", port)
// 	log.Printf("📚 Environment: %s", cfg.Server.Environment)
// 	log.Printf("🤖 AI Providers: %v", providerRouter.List())
// 	log.Printf("🔄 Redis Queue: %s", redisAddr)
// 	log.Printf("========================================")
// 	log.Printf("✅ Server is ready to accept requests")
// 	log.Printf("📖 Swagger UI available at http://localhost:%s/swagger/index.html", port)
// 	log.Printf("========================================")

// 	// Start server with graceful shutdown
// 	server := r

// 	go func() {
// 		if err := server.Run(":" + port); err != nil {
// 			log.Fatal("Failed to start server:", err)
// 		}
// 	}()

// 	// ============================================
// 	// GRACEFUL SHUTDOWN
// 	// ============================================

// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
// 	<-quit

// 	log.Println("🛑 Shutting down server gracefully...")

// 	// Close Redis connection
// 	if err := redisQueue.Close(); err != nil {
// 		log.Printf("⚠️ Error closing Redis connection: %v", err)
// 	}

// 	log.Println("✅ Server shutdown complete")
// }

// // package main

// // import (
// //     "log"

// //     "cbt-api/api/routes"
// //     "cbt-api/config"
// //     "cbt-api/internal/middleware"
// //     "cbt-api/pkg/database"

// //     "github.com/gin-contrib/cors"
// //     "github.com/gin-gonic/gin"

// //     // Swagger documentation (auto‑generated)
// //     _ "cbt-api/api/docs"
// // )

// // // @title           CBT API
// // // @version         1.0
// // // @description     Computer‑Based Testing API with subscriptions, academic management, and CBT modules.
// // // @termsOfService  http://swagger.io/terms/

// // // @contact.name   API Support
// // // @contact.email  support@cbt-api.com

// // // @license.name  MIT
// // // @license.url   https://opensource.org/licenses/MIT

// // // @host      localhost:8080
// // // @BasePath  /api/v1

// // // @securityDefinitions.apikey BearerAuth
// // // @in header
// // // @name Authorization
// // // @description Enter **Bearer &lt;token&gt;**. Example: `Bearer eyJhbGciOiJIUzI1NiIs...`

// // func main() {
// //     // Load configuration
// //     cfg := config.LoadConfig()

// //     // Initialize database
// //     database.ConnectDB()

// //     // Set Gin mode based on environment
// //     if cfg.Server.IsProduction() {
// //         gin.SetMode(gin.ReleaseMode)
// //     }

// //     // Setup router
// //     r := gin.Default()

// //     // Global middleware
// //     r.Use(cors.New(cors.Config{
// //         AllowOrigins:     []string{"*"},
// //         AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// //         AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// //         ExposeHeaders:    []string{"Content-Length"},
// //         AllowCredentials: true,
// //     }))
// //     r.Use(middleware.LoggerMiddleware())
// //     r.Use(middleware.RecoveryMiddleware())

// //     // Setup all routes (including Swagger)
// //     routes.SetupRoutes(r)

// //     // Print all available routes (optional)
// //     routes.PrintRoutes()

// //     // Start server
// //     port := cfg.Server.Port
// //     if port == "" {
// //         port = "8080"
// //     }

// //     log.Printf("========================================")
// //     log.Printf("🚀 CBT API Server starting on port %s", port)
// //     log.Printf("📚 Environment: %s", cfg.Server.Environment)
// //     log.Printf("========================================")
// //     log.Printf("✅ Server is ready to accept requests")
// //     log.Printf("📖 Swagger UI available at http://localhost:%s/swagger/index.html", port)
// //     log.Printf("========================================")

// //     if err := r.Run(":" + port); err != nil {
// //         log.Fatal("Failed to start server:", err)
// //     }
// // }




// // package main

// // import (
// //     "log"

// //     "cbt-api/api/routes"
// //     "cbt-api/config"
// //     "cbt-api/internal/middleware"
// //     "cbt-api/pkg/database"

// //     "github.com/gin-contrib/cors"
// //     "github.com/gin-gonic/gin"

// //     // Swagger documentation (auto‑generated)
// //     _ "cbt-api/api/docs"
// // )

// // // @title           CBT API
// // // @version         1.0
// // // @description     Computer‑Based Testing API with subscriptions, academic management, and CBT modules.
// // // @termsOfService  http://swagger.io/terms/

// // // @contact.name   API Support
// // // @contact.email  support@cbt-api.com

// // // @license.name  MIT
// // // @license.url   https://opensource.org/licenses/MIT

// // // @host      localhost:8080
// // // @BasePath  /api/v1

// // // @securityDefinitions.apikey BearerAuth
// // // @in header
// // // @name Authorization
// // // @description Type "Bearer " followed by a valid access token.

// // func main() {
// //     // Load configuration
// //     cfg := config.LoadConfig()

// //     // Initialize database
// //     database.ConnectDB()

// //     // Set Gin mode based on environment
// //     if cfg.Server.IsProduction() {
// //         gin.SetMode(gin.ReleaseMode)
// //     }

// //     // Setup router
// //     r := gin.Default()

// //     // Global middleware
// //     r.Use(cors.New(cors.Config{
// //         AllowOrigins:     []string{"*"},
// //         AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// //         AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// //         ExposeHeaders:    []string{"Content-Length"},
// //         AllowCredentials: true,
// //     }))
// //     r.Use(middleware.LoggerMiddleware())
// //     r.Use(middleware.RecoveryMiddleware())

// //     // Setup all routes (including Swagger)
// //     routes.SetupRoutes(r)

// //     // Print all available routes (optional)
// //     routes.PrintRoutes()

// //     // Start server
// //     port := cfg.Server.Port
// //     if port == "" {
// //         port = "8080"
// //     }

// //     log.Printf("========================================")
// //     log.Printf("🚀 CBT API Server starting on port %s", port)
// //     log.Printf("📚 Environment: %s", cfg.Server.Environment)
// //     log.Printf("========================================")
// //     log.Printf("✅ Server is ready to accept requests")
// //     log.Printf("📖 Swagger UI available at http://localhost:%s/swagger/index.html", port)
// //     log.Printf("========================================")

// //     if err := r.Run(":" + port); err != nil {
// //         log.Fatal("Failed to start server:", err)
// //     }
// // }







// // package main

// // import (
// //     "log"
// //     // "os"

// //     "cbt-api/api/routes"
// //     "cbt-api/config"
// //     "cbt-api/internal/middleware"
// //     "cbt-api/pkg/database"

// //     "github.com/gin-contrib/cors"
// //     "github.com/gin-gonic/gin"
// // )

// // func main() {
// //     // Load configuration
// //     cfg := config.LoadConfig()
    
// //     // Initialize database
// //     database.ConnectDB()

// //     // Set Gin mode based on environment
// //     if cfg.Server.IsProduction() {
// //         gin.SetMode(gin.ReleaseMode)
// //     }

// //     // Setup router
// //     r := gin.Default()
    
// //     // Global middleware
// //     r.Use(cors.New(cors.Config{
// //         AllowOrigins:     []string{"*"},
// //         AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// //         AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// //         ExposeHeaders:    []string{"Content-Length"},
// //         AllowCredentials: true,
// //     }))
// //     r.Use(middleware.LoggerMiddleware())
// //     r.Use(middleware.RecoveryMiddleware())

// //     // Setup all routes
// //     routes.SetupRoutes(r)

// //     // Print all available routes
// //     routes.PrintRoutes()

// //     // Start server
// //     port := cfg.Server.Port
// //     if port == "" {
// //         port = "8080"
// //     }

// //     log.Printf("========================================")
// //     log.Printf("🚀 CBT API Server starting on port %s", port)
// //     log.Printf("📚 Environment: %s", cfg.Server.Environment)
// //     log.Printf("========================================")
// //     log.Printf("✅ Server is ready to accept requests")
// //     log.Printf("========================================")

// //     if err := r.Run(":" + port); err != nil {
// //         log.Fatal("Failed to start server:", err)
// //     }
// // }