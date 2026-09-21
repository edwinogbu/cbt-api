package routes

import (
	academicHandler "cbt-api/internal/academic/handler"
	academicRepo "cbt-api/internal/academic/repository"
	academicService "cbt-api/internal/academic/service"
	authHandler "cbt-api/internal/auth/handler"
	authRepo "cbt-api/internal/auth/repository"
	authService "cbt-api/internal/auth/service"
	"cbt-api/internal/middleware"
	subscriptionHandler "cbt-api/internal/subscription/handler"
	subscriptionRepo "cbt-api/internal/subscription/repository"
	subscriptionService "cbt-api/internal/subscription/service"

	// CBT modules
	cbtExamHandler "cbt-api/internal/cbt/handler"
	cbtExamRepo "cbt-api/internal/cbt/repository"
	cbtExamService "cbt-api/internal/cbt/service"
	cbtQuestionHandler "cbt-api/internal/cbt/handler"
	cbtQuestionRepo "cbt-api/internal/cbt/repository"
	cbtQuestionService "cbt-api/internal/cbt/service"

	cbtSubjectHandler "cbt-api/internal/cbt/handler"
	cbtSubjectRepo "cbt-api/internal/cbt/repository"
	cbtSubjectService "cbt-api/internal/cbt/service"

	// NEW actor modules
	adminHandler "cbt-api/internal/admin/handler"
	adminService "cbt-api/internal/admin/service"
	teacherHandler "cbt-api/internal/teacher/handler"
	teacherService "cbt-api/internal/teacher/service"
	parentHandler "cbt-api/internal/parent/handler"
	parentService "cbt-api/internal/parent/service"

	// ONBOARDING
	onboardingHandler "cbt-api/internal/onboarding/handler"
	onboardingRepo "cbt-api/internal/onboarding/repository"
	onboardingService "cbt-api/internal/onboarding/service"

	// OFFLINE-FIRST SYNC
	syncHandler "cbt-api/internal/sync/handler"
	syncRepo "cbt-api/internal/sync/repository"
	syncService "cbt-api/internal/sync/service"

	"cbt-api/pkg/email"
	"cbt-api/pkg/payment"
	"cbt-api/pkg/database"
	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	// Swagger
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes configures all API routes for the application
func SetupRoutes(r *gin.Engine, q queue.Queue, e *engine.Engine) {
	// Initialize all handlers
	authH := initAuthHandler()
	schoolH := initSchoolHandler()
	sessionH := initSessionHandler()
	termH := initTermHandler()
	classLevelH := initClassLevelHandler()
	classArmH := initClassArmHandler()
	classH := initClassHandler()
	studentH := initStudentHandler()
	subscriptionH := initSubscriptionHandler()

	// CBT handlers
	examH := initExamHandler()
	questionH := initQuestionHandler(q, e)

	// Subject handler initialisation
	subjectRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
	subjectService := cbtSubjectService.NewSubjectService(subjectRepo, database.DB)
	subjectHandler := cbtSubjectHandler.NewSubjectHandler(subjectService)

	// NEW actor handlers
	adminH := initAdminHandler()
	teacherH := initTeacherHandler()
	parentH := initParentHandler()
	// Initialize repositories for middleware
	studentRepo := academicRepo.NewStudentRepository(database.DB)

	// ✅ ONBOARDING INITIALIZATION
	onboardingH := initOnboardingHandler()

	// OFFLINE-FIRST SYNC INITIALIZATION
	syncH := initSyncHandler()

	// API version 1 group
	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", authH.HealthCheck)

		// Setup auth routes
		setupAuthRoutes(v1, authH)

		// Setup academic routes (existing)
		setupAcademicRoutes(v1, schoolH, sessionH, termH, classLevelH, classArmH, classH, studentH)

		// Setup subscription routes (existing)
		setupSubscriptionRoutes(v1, subscriptionH)

		// ✅ ONBOARDING ROUTES - COMPLETE
		setupOnboardingRoutes(v1, onboardingH)

		// ========== SUBJECT ROUTES ==========
		subject := v1.Group("/subjects")
		subject.Use(middleware.AuthMiddleware())
		{
			subject.POST("/create", subjectHandler.CreateSubject)
			subject.GET("/list", subjectHandler.ListSubjects)
			subject.GET("/active", subjectHandler.ListActiveSubjects)
			subject.GET("/view/:id", subjectHandler.GetSubject)
			subject.PUT("/update/:id", subjectHandler.UpdateSubject)
			subject.DELETE("/delete/:id", subjectHandler.DeleteSubject)
		}

		// ==================== EXAM ROUTES ====================
		// ALL 23 EXAM ENDPOINTS ORGANIZED UNDER /v1/exams
		setupExamRoutes(v1, examH)
		setupStudentExamRoutes(v1, examH, studentRepo)

		// ==================== QUESTION ROUTES ====================
		setupCBTQuestionRoutes(v1, questionH)

		// ==================== NEW ACTOR ROUTES ====================
		setupActorRoutes(v1, adminH, teacherH, parentH)

		// ==================== OFFLINE-FIRST SYNC ROUTES ====================
		setupSyncRoutes(v1, syncH, studentRepo)
	}

	// Swagger UI endpoint (no version prefix, accessible directly)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// ============================================
// ONBOARDING ROUTES - COMPLETE
// ============================================
func setupOnboardingRoutes(rg *gin.RouterGroup, handler *onboardingHandler.OnboardingHandler) {
	onboarding := rg.Group("/onboarding")
	{
		// STEP 1: Get Available Gateways
		onboarding.GET("/gateways", handler.GetAvailableGateways)

		// STEP 1: Select Plan
		onboarding.POST("/start", handler.StartOnboarding)

		// STEP 2: Create Administrator Account
		onboarding.POST("/admin", handler.CreateAdminAccount)

		// STEP 3: Verify Email with OTP
		onboarding.POST("/verify-email", handler.VerifyEmail)

		// STEP 4: Register School Information
		onboarding.POST("/school", handler.RegisterSchool)

		// STEP 6: Create Subscription and Payment Intent
		onboarding.POST("/subscription", handler.CreateSubscription)

		// Status endpoints
		onboarding.GET("/status/:session_id", handler.GetStatus)
		onboarding.GET("/payment-status/:session_id", handler.CheckPaymentStatus)
		onboarding.GET("/session/email/:email", handler.GetSessionByEmail)

		// Resend endpoints
		onboarding.POST("/resend-otp", handler.ResendVerificationOTP)
		onboarding.POST("/resend-payment-link", handler.ResendPaymentLink)

		// Webhook endpoint
		onboarding.POST("/webhook/:gateway", handler.Webhook)

		// Health check
		onboarding.GET("/health", handler.HealthCheck)

		// Admin endpoints (protected)
		admin := onboarding.Group("/admin")
		admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
		{
			admin.POST("/complete/:session_id", handler.CompleteOnboarding)
			admin.DELETE("/cleanup", handler.CleanupExpiredSessions)
			admin.GET("/stats", handler.GetOnboardingStats)
		}
	}
}

// ============================================
// INITIALIZE ONBOARDING HANDLER - FIXED!
// ============================================
func initOnboardingHandler() *onboardingHandler.OnboardingHandler {
	// Use existing repositories
	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
	onboardingRepoInstance := onboardingRepo.NewOnboardingRepository(database.DB)

	// Initialize services
	paymentService := payment.NewPaymentService()
	emailService := email.NewEmailService()

	// Create onboarding config
	config := &onboardingService.OnboardingConfig{
		AppURL:             "http://localhost:8080",
		DefaultGateway:     models.GatewayPaystack,
		AllowedGateways:    []models.PaymentGateway{models.GatewayPaystack, models.GatewayFlutterwave, models.GatewayStripe},
		SessionExpiryHours: 24,
		PaymentExpiryHours: 48,
		EnableEmail:        true,
		EnableWebhooks:     true,
		WebhookSecret:      "your-webhook-secret",
		RetryAttempts:      3,
	}

	// ✅ CORRECTED - Only 5 parameters now!
	svc := onboardingService.NewOnboardingService(
		subRepo,              // Subscription repository
		paymentService,       // Payment service
		emailService,         // Email service
		onboardingRepoInstance, // Onboarding repository (self-contained!)
		config,               // Config
	)

	return onboardingHandler.NewOnboardingHandler(svc)
}

// ============================================
// setupStudentExamRoutes - All student exam routes with automatic context
// ============================================
func setupStudentExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, studentRepo *academicRepo.StudentRepository) {
	student := rg.Group("/student/exams")
	student.Use(middleware.AuthMiddleware())
	student.Use(middleware.StudentContextMiddleware(studentRepo))
	{
		student.GET("/dashboard", handler.GetStudentDashboardForUser)
		student.GET("/performance", handler.GetStudentPerformanceForUser)
		student.GET("/package/:examId", handler.GetExamPackageForUser)
		student.POST("/start/:examId", handler.StartExamForUser)
		student.GET("/attempt/:attemptId", handler.GetAttemptState)
		student.POST("/answer", handler.SaveAnswer)
		student.POST("/bulk-answer", handler.BulkSaveAnswers)
		student.POST("/submit", handler.SubmitExam)
		student.POST("/auto-save", handler.AutoSave)
		student.POST("/offline-submit", handler.OfflineSubmit)
		student.POST("/sync", handler.SyncOfflineAnswers)
		student.GET("/result/:examId", handler.GetStudentExamResultForUser)
		student.GET("/review/:attemptId", handler.GetStudentExamReviewForUser)
		student.GET("/termly-result", handler.GetStudentTermlyResultForUser)
		student.GET("/report-card", handler.GetTermlyReportCardForUser)
		student.POST("/practice", handler.StartPracticeForUser)
	}
}

// ============================================
// EXAM ROUTES - ALL 23 ENDPOINTS
// ============================================
func setupExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler) {
	exams := rg.Group("/exams")
	exams.Use(middleware.AuthMiddleware())
	{
		exams.POST("/create", handler.CreateExam)
		exams.GET("/:id", handler.GetExam)
		exams.GET("/list", handler.ListExams)
		exams.PUT("/:id", handler.UpdateExam)
		exams.DELETE("/:id", handler.DeleteExam)
		exams.POST("/:id/questions", handler.AddQuestionsToExam)
		exams.DELETE("/:id/questions/:questionId", handler.RemoveQuestionFromExam)
		exams.POST("/:id/assign", handler.AssignExam)
		exams.GET("/:id/assignments", handler.GetExamAssignments)
		exams.POST("/start", handler.StartExam)
		exams.POST("/save-answer", handler.SaveAnswer)
		exams.POST("/bulk-answer", handler.BulkSaveAnswers)
		exams.POST("/submit", handler.SubmitExam)
		exams.GET("/:id/results/class", handler.GetClassResults)
		exams.GET("/:id/results/export", handler.ExportResults)
		exams.GET("/:id/results/rankings", handler.GetExamRankings)
		exams.GET("/:id/statistics", handler.GetExamStatistics)
		exams.GET("/students/:studentId/performance", handler.GetStudentPerformance)
		exams.GET("/students/:studentId/exams/:examId/result", handler.GetStudentExamResult)
		exams.GET("/practice/:studentId/:subjectId", handler.StartPractice)
		exams.GET("/teacher/subjects/:subjectId/results", handler.GetTeacherSubjectResults)
		exams.GET("/teacher/classes/:classId/results", handler.GetTeacherClassResults)
		exams.GET("/teacher/performance/dashboard", handler.GetTeacherDashboard)
		exams.GET("/school/performance/overview", handler.GetSchoolPerformanceOverview)
		exams.POST("/create-with-context", handler.CreateExamWithContext)
		exams.GET("/subject-questions", handler.GetSubjectQuestionsForExam)
		exams.POST("/:id/bulk-questions", handler.BulkAddQuestionsToExam)
		exams.GET("/:id/preview", handler.PreviewExam)
		exams.POST("/:id/publish", handler.PublishExam)
	}
}

// ============================================
// ACTOR ROUTES (Admin, Teacher, Parent)
// ============================================
func setupActorRoutes(rg *gin.RouterGroup,
	adminH *adminHandler.AdminHandler,
	teacherH *teacherHandler.TeacherHandler,
	parentH *parentHandler.ParentHandler) {

	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		admin.POST("/teachers/assign", adminH.AssignTeacher)
		admin.DELETE("/teachers/unassign/:classId", adminH.UnassignTeacher)
		admin.GET("/users", adminH.ListUsers)
		admin.GET("/students", adminH.ListStudents)
		admin.GET("/classes", adminH.ListClasses)
		admin.GET("/teachers", adminH.ListTeachers)
		admin.DELETE("/students/:id/permanent", adminH.HardDeleteStudent)
	}

	teacher := rg.Group("/teacher")
	teacher.Use(middleware.AuthMiddleware(), middleware.TeacherOnly())
	{
		teacher.POST("/students", teacherH.CreateStudent)
		teacher.POST("/students/bulk", teacherH.BulkCreateStudents)
		teacher.GET("/students", teacherH.GetMyStudents)
		teacher.GET("/students/credentials", teacherH.GetAllStudentsWithCredentials)
		teacher.GET("/students/:id", teacherH.GetStudent)
		teacher.PUT("/students/:id", teacherH.UpdateStudent)
		teacher.POST("/students/:id/reset-password", teacherH.ResetPassword)
		teacher.POST("/students/:id/deactivate", teacherH.DeactivateStudent)
	}

	parent := rg.Group("/parent")
	parent.Use(middleware.AuthMiddleware(), middleware.ParentOnly())
	{
		parent.GET("/children", parentH.GetChildren)
		parent.GET("/child/:studentId/results", parentH.GetChildResults)
	}
}

// ============================================
// AUTH ROUTES
// ============================================
func setupAuthRoutes(rg *gin.RouterGroup, handler *authHandler.AuthHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/verify-2fa", handler.Verify2FALogin)
		auth.POST("/refresh", handler.RefreshToken)
		auth.POST("/forgot-password", handler.ForgotPassword)
		auth.POST("/reset-password", handler.ResetPassword)
		auth.POST("/send-otp", handler.SendVerificationOTP)
		auth.POST("/verify-email", handler.VerifyEmail)
	}

	protected := rg.Group("/auth")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/logout", handler.Logout)
		protected.POST("/logout-all", handler.LogoutAllDevices)
		protected.GET("/sessions", handler.GetSessions)
		protected.DELETE("/sessions/:sessionId", handler.RevokeSession)
		protected.POST("/change-password", handler.ChangePassword)
		protected.POST("/2fa/generate", handler.Generate2FA)
		protected.POST("/2fa/enable", handler.Enable2FA)
		protected.POST("/2fa/disable", handler.Disable2FA)
		protected.GET("/profile", handler.GetProfile)
		protected.PUT("/profile", handler.UpdateProfile)
		protected.DELETE("/profile", handler.DeleteAccount)
		protected.GET("/subscription/check", handler.CheckSubscription)
		protected.GET("/status", handler.GetAuthStatus)
	}
}

// ============================================
// ACADEMIC ROUTES
// ============================================
func setupAcademicRoutes(rg *gin.RouterGroup,
	schoolHandler *academicHandler.SchoolHandler,
	sessionHandler *academicHandler.SessionHandler,
	termHandler *academicHandler.TermHandler,
	classLevelHandler *academicHandler.ClassLevelHandler,
	classArmHandler *academicHandler.ClassArmHandler,
	classHandler *academicHandler.ClassHandler,
	studentHandler *academicHandler.StudentHandler) {

	academic := rg.Group("/")
	academic.Use(middleware.AuthMiddleware())
	{
		academic.POST("/schools", schoolHandler.CreateSchool)
		academic.GET("/schools", schoolHandler.GetAllSchools)
		academic.GET("/schools/:id", schoolHandler.GetSchool)
		academic.PUT("/schools/:id", schoolHandler.UpdateSchool)
		academic.DELETE("/schools/:id", schoolHandler.DeleteSchool)

		academic.POST("/sessions", sessionHandler.CreateSession)
		academic.GET("/school-sessions/:schoolId", sessionHandler.GetSessions)
		academic.GET("/school-sessions/:schoolId/current", sessionHandler.GetCurrentSession)
		academic.GET("/sessions/:id", sessionHandler.GetSession)
		academic.PUT("/sessions/:id", sessionHandler.UpdateSession)
		academic.DELETE("/sessions/:id", sessionHandler.DeleteSession)

		academic.GET("/sessions", sessionHandler.ListSessions)
		academic.GET("/sessions/stats", sessionHandler.GetSessionStats)
		academic.GET("/sessions/timeline", sessionHandler.GetSessionTimeline)
		academic.GET("/sessions/summary", sessionHandler.GetSessionSummary)
		academic.GET("/sessions/search", sessionHandler.SearchSessions)
		academic.DELETE("/sessions/bulk", sessionHandler.BulkDeleteSessions)

		academic.GET("/terms/list", termHandler.ListAllTerms)
		academic.GET("/terms/school", termHandler.ListAllTermsBySchool)
		academic.GET("/terms/all", termHandler.GetAllTerms)
		academic.GET("/terms", termHandler.ListTerms)
		academic.GET("/terms/stats", termHandler.GetTermStats)
		academic.GET("/terms/search", termHandler.SearchTerms)
		academic.DELETE("/terms/bulk", termHandler.BulkDeleteTerms)
		academic.POST("/terms", termHandler.CreateTerm)
		academic.GET("/session-terms/:sessionId", termHandler.GetTerms)
		academic.GET("/session-terms/:sessionId/current", termHandler.GetCurrentTerm)
		academic.GET("/terms/:id", termHandler.GetTerm)
		academic.PUT("/terms/:id", termHandler.UpdateTerm)
		academic.DELETE("/terms/:id", termHandler.DeleteTerm)

		academic.GET("/class-levels", classLevelHandler.ListClassLevels)
		academic.GET("/class-levels/stats", classLevelHandler.GetClassLevelStats)
		academic.GET("/class-levels/search", classLevelHandler.SearchClassLevels)
		academic.DELETE("/class-levels/bulk", classLevelHandler.BulkDeleteClassLevels)
		academic.POST("/class-levels", classLevelHandler.CreateClassLevel)
		academic.GET("/class-levels/:id", classLevelHandler.GetClassLevel)
		academic.PUT("/class-levels/:id", classLevelHandler.UpdateClassLevel)
		academic.DELETE("/class-levels/:id", classLevelHandler.DeleteClassLevel)
		academic.GET("/school-class-levels/:schoolId", classLevelHandler.GetClassLevelsBySchool)

		academic.GET("/class-arms", classArmHandler.ListClassArms)
		academic.GET("/class-arms/stats", classArmHandler.GetClassArmStats)
		academic.GET("/class-arms/search", classArmHandler.SearchClassArms)
		academic.DELETE("/class-arms/bulk", classArmHandler.BulkDeleteClassArms)
		academic.POST("/class-arms", classArmHandler.CreateClassArm)
		academic.GET("/school-class-arms/:schoolId", classArmHandler.GetClassArms)
		academic.GET("/class-arms/:id", classArmHandler.GetClassArm)
		academic.PUT("/class-arms/:id", classArmHandler.UpdateClassArm)
		academic.DELETE("/class-arms/:id", classArmHandler.DeleteClassArm)

		academic.GET("/classes", classHandler.ListClasses)
		academic.GET("/classes/teacher/:teacherId", classHandler.GetClassesByTeacher)
		academic.GET("/classes/arm/:classArmId", classHandler.GetClassesByArm)
		academic.GET("/classes/level/:classLevelId", classHandler.GetClassesByLevel)
		academic.DELETE("/classes/bulk", classHandler.BulkDeleteClasses)
		academic.GET("/classes/stats", classHandler.GetClassStats)
		academic.GET("/classes/search", classHandler.SearchClasses)

		academic.POST("/classes", classHandler.CreateClass)
		academic.GET("/school-classes/:schoolId", classHandler.GetClassesBySchool)
		academic.GET("/session-classes/:sessionId", classHandler.GetClassesBySession)
		academic.GET("/school-session-classes/:schoolId/:sessionId", classHandler.GetClassesBySchoolAndSession)
		academic.GET("/classes/:id", classHandler.GetClass)
		academic.PUT("/classes/:id", classHandler.UpdateClass)
		academic.DELETE("/classes/:id", classHandler.DeleteClass)

		academic.GET("/student/profile", studentHandler.GetStudentProfile)

		academic.POST("/students", studentHandler.CreateStudent)
		academic.GET("/school-students/:schoolId", studentHandler.GetStudentsBySchool)
		academic.GET("/class-students/:classId", studentHandler.GetStudentsByClass)
		academic.GET("/students/:id", studentHandler.GetStudent)
		academic.GET("/user-student/:userId", studentHandler.GetStudentByUser)
		academic.PUT("/students/:id", studentHandler.UpdateStudent)
		academic.DELETE("/students/:id", studentHandler.DeleteStudent)
		academic.POST("/students/:studentId/transfer", studentHandler.TransferClass)
	}
}

// ============================================
// SUBSCRIPTION ROUTES
// ============================================
func setupSubscriptionRoutes(rg *gin.RouterGroup, handler *subscriptionHandler.SubscriptionHandler) {
	subscription := rg.Group("/subscriptions")
	subscription.Use(middleware.AuthMiddleware())
	{
		subscription.POST("", handler.CreateSubscription)
		subscription.GET("", handler.GetSubscriptions)
		subscription.GET("/:id", handler.GetSubscription)
		subscription.GET("/school/:schoolId/current", handler.GetCurrentSubscription)
		subscription.PUT("/:id", handler.UpdateSubscription)
		subscription.POST("/:id/cancel", handler.CancelSubscription)
		subscription.POST("/:id/renew", handler.RenewSubscription)
		subscription.GET("/:id/usage", handler.GetSubscriptionUsage)
		subscription.GET("/:id/invoices", handler.GetInvoices)
		subscription.GET("/:id/transactions", handler.GetTransactions)
		subscription.POST("/:id/payment-intent", handler.CreatePaymentIntent)
		subscription.POST("/payment-intent/:id/confirm", handler.ConfirmPaymentIntent)
		subscription.POST("/verify", handler.VerifyPayment)
	}
	rg.POST("/webhook/:gateway", handler.HandleWebhook)
}

// ============================================
// CBT QUESTION BANK ROUTES
// ============================================
func setupCBTQuestionRoutes(rg *gin.RouterGroup, handler *cbtQuestionHandler.QuestionHandler) {
	q := rg.Group("/questions")
	q.Use(middleware.AuthMiddleware())
	{
		q.POST("/create", handler.CreateQuestion)
		q.GET("/:id", handler.GetQuestion)
		q.PUT("/update/:id", handler.UpdateQuestion)
		q.DELETE("/delete/:id", handler.DeleteQuestion)

		q.GET("/list", handler.ListQuestions)
		q.POST("/filter", handler.FilterQuestions)

		q.POST("/bulk", handler.BulkCreateQuestions)
		q.POST("/bulk-upload", handler.BulkUploadFile)
		q.POST("/bulk-delete", handler.BulkDelete)
		q.PUT("/bulk-status", handler.BulkUpdateStatus)

		q.POST("/tags/create", handler.CreateTag)
		q.GET("/tags/list", handler.ListTags)

		q.GET("/statistics", handler.GetStatistics)

		q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
		q.POST("/extract", handler.ExtractQuestionsFromText)
		q.GET("/jobs/:id", handler.GetJobStatus)

		q.GET("/by-term", handler.GetQuestionsByTerm)
		q.GET("/by-session", handler.GetQuestionsBySession)
		q.GET("/by-class", handler.GetQuestionsByClass)
		q.GET("/for-exam", handler.GetQuestionsForExam)
		q.GET("/context/summary", handler.GetQuestionContextSummary)
		q.POST("/with-context", handler.GetQuestionsWithContext)
		q.GET("/academic/current", handler.GetCurrentAcademicContext)

		// NEW GROUPED QUESTION ROUTES
		q.POST("/grouped", handler.GetQuestionsGrouped)
		q.GET("/by-exam-type", handler.GetQuestionsByExamType)
		q.GET("/by-term-grouped", handler.GetQuestionsByTermGrouped)
		q.GET("/by-session-grouped", handler.GetQuestionsBySessionGrouped)
		q.GET("/all-grouped", handler.GetAllQuestionsGrouped)
	}
}

// ============================================
// OFFLINE-FIRST SYNC ROUTES
// ============================================
// Implements the endpoint the frontend's offline-first sync engine
// (src/lib/storage/sync/{lanSync,cloudSync}.ts) already calls but which
// did not previously exist server-side. See internal/sync/service for the
// sessionId/attemptId convention and idempotency contract.
func setupSyncRoutes(rg *gin.RouterGroup, handler *syncHandler.SyncHandler, studentRepo *academicRepo.StudentRepository) {
	sync := rg.Group("/sync")
	sync.Use(middleware.AuthMiddleware())
	sync.Use(middleware.StudentContextMiddleware(studentRepo))
	{
		sync.POST("/:type", handler.Sync)
	}
}

func initSyncHandler() *syncHandler.SyncHandler {
	examRepo := cbtExamRepo.NewExamRepository(database.DB)
	questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
	examSvc := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)

	syncRepository := syncRepo.NewSyncRepository(database.DB)
	syncSvc := syncService.NewSyncService(syncRepository, examSvc)

	return syncHandler.NewSyncHandler(syncSvc)
}

// ============================================
// INITIALIZERS
// ============================================

func initSubscriptionHandler() *subscriptionHandler.SubscriptionHandler {
	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
	paymentService := payment.NewPaymentService()
	emailService := email.NewEmailService()
	subService := subscriptionService.NewSubscriptionService(subRepo, paymentService, emailService)
	return subscriptionHandler.NewSubscriptionHandler(subService)
}

func initStudentHandler() *academicHandler.StudentHandler {
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	userRepo := academicRepo.NewUserRepository(database.DB)
	classRepo := academicRepo.NewClassRepository(database.DB)
	service := academicService.NewStudentService(studentRepo, userRepo, classRepo)
	return academicHandler.NewStudentHandler(service)
}

func initClassHandler() *academicHandler.ClassHandler {
	classRepo := academicRepo.NewClassRepository(database.DB)
	classLevelRepo := academicRepo.NewClassLevelRepository(database.DB)
	classArmRepo := academicRepo.NewClassArmRepository(database.DB)
	sessionRepo := academicRepo.NewSessionRepository(database.DB)
	service := academicService.NewClassService(classRepo, classLevelRepo, classArmRepo, sessionRepo)
	return academicHandler.NewClassHandler(service)
}

func initClassLevelHandler() *academicHandler.ClassLevelHandler {
	repo := academicRepo.NewClassLevelRepository(database.DB)
	service := academicService.NewClassLevelService(repo)
	return academicHandler.NewClassLevelHandler(service)
}

func initClassArmHandler() *academicHandler.ClassArmHandler {
	repo := academicRepo.NewClassArmRepository(database.DB)
	service := academicService.NewClassArmService(repo)
	return academicHandler.NewClassArmHandler(service)
}

func initTermHandler() *academicHandler.TermHandler {
	repo := academicRepo.NewTermRepository(database.DB)
	service := academicService.NewTermService(repo)
	return academicHandler.NewTermHandler(service)
}

func initSessionHandler() *academicHandler.SessionHandler {
	repo := academicRepo.NewSessionRepository(database.DB)
	service := academicService.NewSessionService(repo)
	return academicHandler.NewSessionHandler(service)
}

func initSchoolHandler() *academicHandler.SchoolHandler {
	repo := academicRepo.NewSchoolRepository(database.DB)
	service := academicService.NewSchoolService(repo)
	return academicHandler.NewSchoolHandler(service)
}

func initAuthHandler() *authHandler.AuthHandler {
	repo := authRepo.NewAuthRepository(database.DB)
	parentSvc := initParentService()

	schoolRepo := academicRepo.NewSchoolRepository(database.DB)
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	parentRepo := academicRepo.NewParentRepository(database.DB)
	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)

	logger, _ := zap.NewProduction()

	svc := authService.NewAuthService(
		repo,
		parentSvc,
		schoolRepo,
		studentRepo,
		parentRepo,
		subRepo,
		logger,
	)
	return authHandler.NewAuthHandler(svc)
}

func initExamHandler() *cbtExamHandler.ExamHandler {
	examRepo := cbtExamRepo.NewExamRepository(database.DB)
	questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
	examService := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)
	return cbtExamHandler.NewExamHandler(examService)
}

func initQuestionHandler(queue queue.Queue, engine *engine.Engine) *cbtQuestionHandler.QuestionHandler {
	repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
	subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
	service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB, queue, engine)
	return cbtQuestionHandler.NewQuestionHandler(service)
}

func initAdminHandler() *adminHandler.AdminHandler {
	userRepo := academicRepo.NewUserRepository(database.DB)
	classRepo := academicRepo.NewClassRepository(database.DB)
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	svc := adminService.NewAdminService(userRepo, classRepo, studentRepo, database.DB)
	return adminHandler.NewAdminHandler(svc)
}

func initTeacherHandler() *teacherHandler.TeacherHandler {
	userRepo := academicRepo.NewUserRepository(database.DB)
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	classRepo := academicRepo.NewClassRepository(database.DB)
	schoolRepo := academicRepo.NewSchoolRepository(database.DB)
	logger, _ := zap.NewProduction()
	svc := teacherService.NewTeacherService(userRepo, studentRepo, classRepo, schoolRepo, database.DB, logger)
	return teacherHandler.NewTeacherHandler(svc)
}

func initParentHandler() *parentHandler.ParentHandler {
	parentRepo := academicRepo.NewParentRepository(database.DB)
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	classRepo := academicRepo.NewClassRepository(database.DB)
	examRepo := cbtExamRepo.NewExamRepository(database.DB)
	userRepo := academicRepo.NewUserRepository(database.DB)
	logger, _ := zap.NewProduction()
	svc := parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
	return parentHandler.NewParentHandler(svc)
}

func initParentService() *parentService.ParentService {
	parentRepo := academicRepo.NewParentRepository(database.DB)
	studentRepo := academicRepo.NewStudentRepository(database.DB)
	classRepo := academicRepo.NewClassRepository(database.DB)
	examRepo := cbtExamRepo.NewExamRepository(database.DB)
	userRepo := academicRepo.NewUserRepository(database.DB)
	logger, _ := zap.NewProduction()
	return parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
}

// ============================================
// PRINT ROUTES - UPDATED WITH ONBOARDING
// ============================================
func PrintRoutes() {
	println("")
	println("========================================")
	println("📋 AVAILABLE API ENDPOINTS")
	println("========================================")
	println("")
	println("🔓 PUBLIC ROUTES:")
	println("   POST   /api/v1/auth/register")
	println("   POST   /api/v1/auth/login")
	println("   POST   /api/v1/auth/verify-2fa")
	println("   POST   /api/v1/auth/refresh")
	println("   POST   /api/v1/auth/forgot-password")
	println("   POST   /api/v1/auth/reset-password")
	println("   POST   /api/v1/auth/send-otp")
	println("   POST   /api/v1/auth/verify-email")
	println("")
	println("🚀 ONBOARDING ROUTES (Public - No Auth):")
	println("   GET    /api/v1/onboarding/gateways")
	println("   POST   /api/v1/onboarding/start")
	println("   POST   /api/v1/onboarding/admin")
	println("   POST   /api/v1/onboarding/verify-email")
	println("   POST   /api/v1/onboarding/school")
	println("   POST   /api/v1/onboarding/subscription")
	println("   GET    /api/v1/onboarding/status/:session_id")
	println("   GET    /api/v1/onboarding/payment-status/:session_id")
	println("   GET    /api/v1/onboarding/session/email/:email")
	println("   POST   /api/v1/onboarding/resend-otp")
	println("   POST   /api/v1/onboarding/resend-payment-link")
	println("   POST   /api/v1/onboarding/webhook/:gateway")
	println("   GET    /api/v1/onboarding/health")
	println("")
	println("🔗 WEBHOOK (Public):")
	println("   POST   /api/v1/webhook/:gateway")
	println("")
	println("🔒 PROTECTED ROUTES (Bearer Token Required):")
	println("   POST   /api/v1/auth/logout")
	println("   POST   /api/v1/auth/logout-all")
	println("   GET    /api/v1/auth/sessions")
	println("   DELETE /api/v1/auth/sessions/:sessionId")
	println("   POST   /api/v1/auth/change-password")
	println("   POST   /api/v1/auth/2fa/generate")
	println("   POST   /api/v1/auth/2fa/enable")
	println("   POST   /api/v1/auth/2fa/disable")
	println("   GET    /api/v1/auth/profile")
	println("   PUT    /api/v1/auth/profile")
	println("   DELETE /api/v1/auth/profile")
	println("   GET    /api/v1/auth/subscription/check")
	println("   GET    /api/v1/auth/status")
	println("")
	println("🏫 ACADEMIC ROUTES (Protected):")
	println("   POST   /api/v1/schools")
	println("   GET    /api/v1/schools")
	println("   GET    /api/v1/schools/:id")
	println("   PUT    /api/v1/schools/:id")
	println("   DELETE /api/v1/schools/:id")
	println("   POST   /api/v1/sessions")
	println("   GET    /api/v1/school-sessions/:schoolId")
	println("   GET    /api/v1/sessions/:id")
	println("   POST   /api/v1/terms")
	println("   GET    /api/v1/session-terms/:sessionId")
	println("   POST   /api/v1/class-levels")
	println("   GET    /api/v1/school-class-levels/:schoolId")
	println("   POST   /api/v1/class-arms")
	println("   GET    /api/v1/school-class-arms/:schoolId")
	println("   POST   /api/v1/classes")
	println("   GET    /api/v1/school-classes/:schoolId")
	println("   POST   /api/v1/students")
	println("   GET    /api/v1/school-students/:schoolId")
	println("")
	println("📚 CBT SUBJECT ROUTES (Protected):")
	println("   POST   /api/v1/subjects/create")
	println("   GET    /api/v1/subjects/list")
	println("   GET    /api/v1/subjects/active")
	println("   GET    /api/v1/subjects/view/:id")
	println("   PUT    /api/v1/subjects/update/:id")
	println("   DELETE /api/v1/subjects/delete/:id")
	println("")
	println("📝 CBT EXAM ROUTES (Protected):")
	println("   📋 EXAM MANAGEMENT (9 Endpoints):")
	println("   POST   /api/v1/exams/create")
	println("   GET    /api/v1/exams/:id")
	println("   GET    /api/v1/exams/list")
	println("   PUT    /api/v1/exams/:id")
	println("   DELETE /api/v1/exams/:id")
	println("   POST   /api/v1/exams/:id/questions")
	println("   DELETE /api/v1/exams/:id/questions/:questionId")
	println("   POST   /api/v1/exams/:id/assign")
	println("   GET    /api/v1/exams/:id/assignments")
	println("")
	println("   ✍️ EXAM TAKING (3 Endpoints):")
	println("   POST   /api/v1/exams/start")
	println("   POST   /api/v1/exams/save-answer")
	println("   POST   /api/v1/exams/submit")
	println("")
	println("   📊 RESULTS & PERFORMANCE (4 Endpoints):")
	println("   GET    /api/v1/exams/:id/results/class")
	println("   GET    /api/v1/exams/:id/results/export")
	println("   GET    /api/v1/exams/:id/results/rankings")
	println("   GET    /api/v1/exams/:id/statistics")
	println("")
	println("   🎓 STUDENT PERFORMANCE (2 Endpoints):")
	println("   GET    /api/v1/exams/students/:studentId/performance")
	println("   GET    /api/v1/exams/students/:studentId/exams/:examId/result")
	println("")
	println("   🏋️ PRACTICE (1 Endpoint):")
	println("   GET    /api/v1/exams/practice/:studentId/:subjectId")
	println("")
	println("   👨‍🏫 TEACHER VIEWS (3 Endpoints):")
	println("   GET    /api/v1/exams/teacher/subjects/:subjectId/results")
	println("   GET    /api/v1/exams/teacher/classes/:classId/results")
	println("   GET    /api/v1/exams/teacher/performance/dashboard")
	println("")
	println("   🛡️ ADMIN VIEWS (1 Endpoint):")
	println("   GET    /api/v1/exams/school/performance/overview")
	println("")
	println("👨‍🏫 TEACHER ROUTES (Protected):")
	println("   POST   /api/v1/teacher/students")
	println("   POST   /api/v1/teacher/students/bulk")
	println("   GET    /api/v1/teacher/students")
	println("   GET    /api/v1/teacher/students/:id")
	println("   PUT    /api/v1/teacher/students/:id")
	println("   POST   /api/v1/teacher/students/:id/reset-password")
	println("   POST   /api/v1/teacher/students/:id/deactivate")
	println("")
	println("👪 PARENT ROUTES (Protected):")
	println("   GET    /api/v1/parent/children")
	println("   GET    /api/v1/parent/child/:studentId/results")
	println("")
	println("🛡️ ADMIN ROUTES (Protected):")
	println("   POST   /api/v1/admin/teachers/assign")
	println("   DELETE /api/v1/admin/teachers/unassign/:classId")
	println("   GET    /api/v1/admin/users")
	println("   GET    /api/v1/admin/students")
	println("   GET    /api/v1/admin/classes")
	println("   GET    /api/v1/admin/teachers")
	println("   DELETE /api/v1/admin/students/:id/permanent")
	println("   POST   /api/v1/onboarding/admin/complete/:session_id")
	println("   DELETE /api/v1/onboarding/admin/cleanup")
	println("   GET    /api/v1/onboarding/admin/stats")
	println("")
	println("💰 SUBSCRIPTION ROUTES (Protected):")
	println("   POST   /api/v1/subscriptions")
	println("   GET    /api/v1/subscriptions")
	println("   GET    /api/v1/subscriptions/:id")
	println("   GET    /api/v1/subscriptions/school/:schoolId/current")
	println("   PUT    /api/v1/subscriptions/:id")
	println("   POST   /api/v1/subscriptions/:id/cancel")
	println("   POST   /api/v1/subscriptions/:id/renew")
	println("   GET    /api/v1/subscriptions/:id/usage")
	println("   GET    /api/v1/subscriptions/:id/invoices")
	println("   GET    /api/v1/subscriptions/:id/transactions")
	println("   POST   /api/v1/subscriptions/:id/payment-intent")
	println("   POST   /api/v1/subscriptions/payment-intent/:id/confirm")
	println("   POST   /api/v1/subscriptions/verify")
	println("")
	println("📚 CBT QUESTION BANK ROUTES (Protected):")
	println("   POST   /api/v1/questions/create")
	println("   GET    /api/v1/questions/:id")
	println("   PUT    /api/v1/questions/update/:id")
	println("   DELETE /api/v1/questions/delete/:id")
	println("   GET    /api/v1/questions/list")
	println("   POST   /api/v1/questions/filter")
	println("   POST   /api/v1/questions/bulk")
	println("   POST   /api/v1/questions/bulk-upload")
	println("   POST   /api/v1/questions/bulk-delete")
	println("   GET    /api/v1/questions/statistics")
	println("   POST   /api/v1/questions/tags/create")
	println("   GET    /api/v1/questions/tags/list")
	println("   POST   /api/v1/questions/ai/generate")
	println("   POST   /api/v1/questions/extract")
	println("   GET    /api/v1/questions/jobs/:id")
	println("")
	println("========================================")
}


// package routes

// import (
// 	academicHandler "cbt-api/internal/academic/handler"
// 	academicRepo "cbt-api/internal/academic/repository"
// 	academicService "cbt-api/internal/academic/service"
// 	authHandler "cbt-api/internal/auth/handler"
// 	authRepo "cbt-api/internal/auth/repository"
// 	authService "cbt-api/internal/auth/service"
// 	"cbt-api/internal/middleware"
// 	subscriptionHandler "cbt-api/internal/subscription/handler"
// 	subscriptionRepo "cbt-api/internal/subscription/repository"
// 	subscriptionService "cbt-api/internal/subscription/service"

// 	// CBT modules
// 	cbtExamHandler "cbt-api/internal/cbt/handler"
// 	cbtExamRepo "cbt-api/internal/cbt/repository"
// 	cbtExamService "cbt-api/internal/cbt/service"
// 	cbtQuestionHandler "cbt-api/internal/cbt/handler"
// 	cbtQuestionRepo "cbt-api/internal/cbt/repository"
// 	cbtQuestionService "cbt-api/internal/cbt/service"

// 	cbtSubjectHandler "cbt-api/internal/cbt/handler"
// 	cbtSubjectRepo "cbt-api/internal/cbt/repository"
// 	cbtSubjectService "cbt-api/internal/cbt/service"

// 	// NEW actor modules
// 	adminHandler "cbt-api/internal/admin/handler"
// 	adminService "cbt-api/internal/admin/service"
// 	teacherHandler "cbt-api/internal/teacher/handler"
// 	teacherService "cbt-api/internal/teacher/service"
// 	parentHandler "cbt-api/internal/parent/handler"
// 	parentService "cbt-api/internal/parent/service"

// 	// ONBOARDING
// 	onboardingHandler "cbt-api/internal/onboarding/handler"
// 	onboardingService "cbt-api/internal/onboarding/service"

// 	"cbt-api/pkg/email"
// 	"cbt-api/pkg/payment"
// 	"cbt-api/pkg/database"
// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/ai/queue"
// 	"cbt-api/internal/models"

// 	"github.com/gin-gonic/gin"
// 	"go.uber.org/zap"

// 	// Swagger
// 	swaggerFiles "github.com/swaggo/files"
// 	ginSwagger "github.com/swaggo/gin-swagger"
// )

// // SetupRoutes configures all API routes for the application
// func SetupRoutes(r *gin.Engine, q queue.Queue, e *engine.Engine) {
// 	// Initialize all handlers
// 	authH := initAuthHandler()
// 	schoolH := initSchoolHandler()
// 	sessionH := initSessionHandler()
// 	termH := initTermHandler()
// 	classLevelH := initClassLevelHandler()
// 	classArmH := initClassArmHandler()
// 	classH := initClassHandler()
// 	studentH := initStudentHandler()
// 	subscriptionH := initSubscriptionHandler()

// 	// CBT handlers
// 	examH := initExamHandler()
// 	questionH := initQuestionHandler(q, e)

// 	// Subject handler initialisation
// 	subjectRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// 	subjectService := cbtSubjectService.NewSubjectService(subjectRepo, database.DB)
// 	subjectHandler := cbtSubjectHandler.NewSubjectHandler(subjectService)

// 	// NEW actor handlers
// 	adminH := initAdminHandler()
// 	teacherH := initTeacherHandler()
// 	parentH := initParentHandler()
// 	// Initialize repositories for middleware
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)

// 	// ✅ ONBOARDING INITIALIZATION - CLEAN, NO WRAPPERS!
// 	onboardingH := initOnboardingHandler()

// 	// API version 1 group
// 	v1 := r.Group("/api/v1")
// 	{
// 		// Health check
// 		v1.GET("/health", authH.HealthCheck)

// 		// Setup auth routes
// 		setupAuthRoutes(v1, authH)

// 		// Setup academic routes (existing)
// 		setupAcademicRoutes(v1, schoolH, sessionH, termH, classLevelH, classArmH, classH, studentH)

// 		// Setup subscription routes (existing)
// 		setupSubscriptionRoutes(v1, subscriptionH)

// 		// ✅ ONBOARDING ROUTES - COMPLETE
// 		setupOnboardingRoutes(v1, onboardingH)

// 		// ========== SUBJECT ROUTES ==========
// 		subject := v1.Group("/subjects")
// 		subject.Use(middleware.AuthMiddleware())
// 		{
// 			subject.POST("/create", subjectHandler.CreateSubject)
// 			subject.GET("/list", subjectHandler.ListSubjects)
// 			subject.GET("/active", subjectHandler.ListActiveSubjects)
// 			subject.GET("/view/:id", subjectHandler.GetSubject)
// 			subject.PUT("/update/:id", subjectHandler.UpdateSubject)
// 			subject.DELETE("/delete/:id", subjectHandler.DeleteSubject)
// 		}

// 		// ==================== EXAM ROUTES ====================
// 		// ALL 23 EXAM ENDPOINTS ORGANIZED UNDER /v1/exams
// 		setupExamRoutes(v1, examH)
// 		setupStudentExamRoutes(v1, examH, studentRepo)

// 		// ==================== QUESTION ROUTES ====================
// 		setupCBTQuestionRoutes(v1, questionH)

// 		// ==================== NEW ACTOR ROUTES ====================
// 		setupActorRoutes(v1, adminH, teacherH, parentH)
// 	}

// 	// Swagger UI endpoint (no version prefix, accessible directly)
// 	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
// }

// // ============================================
// // ONBOARDING ROUTES - COMPLETE
// // ============================================
// func setupOnboardingRoutes(rg *gin.RouterGroup, handler *onboardingHandler.OnboardingHandler) {
// 	onboarding := rg.Group("/onboarding")
// 	{
// 		// STEP 1: Get Available Gateways
// 		onboarding.GET("/gateways", handler.GetAvailableGateways)

// 		// STEP 1: Select Plan
// 		onboarding.POST("/start", handler.StartOnboarding)

// 		// STEP 2: Create Administrator Account
// 		onboarding.POST("/admin", handler.CreateAdminAccount)

// 		// STEP 3: Verify Email with OTP
// 		onboarding.POST("/verify-email", handler.VerifyEmail)

// 		// STEP 4: Register School Information
// 		onboarding.POST("/school", handler.RegisterSchool)

// 		// STEP 6: Create Subscription and Payment Intent
// 		onboarding.POST("/subscription", handler.CreateSubscription)

// 		// Status endpoints
// 		onboarding.GET("/status/:session_id", handler.GetStatus)
// 		onboarding.GET("/payment-status/:session_id", handler.CheckPaymentStatus)
// 		onboarding.GET("/session/email/:email", handler.GetSessionByEmail)

// 		// Resend endpoints
// 		onboarding.POST("/resend-otp", handler.ResendVerificationOTP)
// 		onboarding.POST("/resend-payment-link", handler.ResendPaymentLink)

// 		// Webhook endpoint
// 		onboarding.POST("/webhook/:gateway", handler.Webhook)

// 		// Health check
// 		onboarding.GET("/health", handler.HealthCheck)

// 		// Admin endpoints (protected)
// 		admin := onboarding.Group("/admin")
// 		admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
// 		{
// 			admin.POST("/complete/:session_id", handler.CompleteOnboarding)
// 			admin.DELETE("/cleanup", handler.CleanupExpiredSessions)
// 			admin.GET("/stats", handler.GetOnboardingStats)
// 		}
// 	}
// }

// // ============================================
// // INITIALIZE ONBOARDING HANDLER - CLEAN!
// // ============================================
// func initOnboardingHandler() *onboardingHandler.OnboardingHandler {
// 	// Use existing repositories
// 	authRepo := authRepo.NewAuthRepository(database.DB)
// 	schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// 	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)

// 	// Initialize services
// 	paymentService := payment.NewPaymentService()
// 	emailService := email.NewEmailService()

// 	// ✅ NO WRAPPERS NEEDED!
// 	// The service now handles all wrapping internally.

// 	// Create onboarding config
// 	config := &onboardingService.OnboardingConfig{
// 		AppURL:             "http://localhost:8080",
// 		DefaultGateway:     models.GatewayPaystack,
// 		AllowedGateways:    []models.PaymentGateway{models.GatewayPaystack, models.GatewayFlutterwave, models.GatewayStripe},
// 		SessionExpiryHours: 24,
// 		PaymentExpiryHours: 48,
// 		EnableEmail:        true,
// 		EnableWebhooks:     true,
// 		WebhookSecret:      "your-webhook-secret",
// 		RetryAttempts:      3,
// 	}

// 	// ✅ Create onboarding service - PASS RAW REPOS, SERVICE WRAPS INTERNALLY!
// 	svc := onboardingService.NewOnboardingService(
// 		subRepo,
// 		paymentService,
// 		emailService,
// 		authRepo,   // ← Raw auth repo - service wraps it!
// 		schoolRepo, // ← Raw school repo - service wraps it!
// 		authRepo,   // ← Raw auth repo (for user operations) - service wraps it!
// 		config,
// 	)

// 	return onboardingHandler.NewOnboardingHandler(svc)
// }

// // ============================================
// // setupStudentExamRoutes - All student exam routes with automatic context
// // ============================================
// func setupStudentExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, studentRepo *academicRepo.StudentRepository) {
// 	student := rg.Group("/student/exams")
// 	student.Use(middleware.AuthMiddleware())
// 	student.Use(middleware.StudentContextMiddleware(studentRepo))
// 	{
// 		student.GET("/dashboard", handler.GetStudentDashboardForUser)
// 		student.GET("/performance", handler.GetStudentPerformanceForUser)
// 		student.POST("/start/:examId", handler.StartExamForUser)
// 		student.GET("/attempt/:attemptId", handler.GetAttemptState)
// 		student.POST("/answer", handler.SaveAnswer)
// 		student.POST("/bulk-answer", handler.BulkSaveAnswers)
// 		student.POST("/submit", handler.SubmitExam)
// 		student.POST("/auto-save", handler.AutoSave)
// 		student.POST("/offline-submit", handler.OfflineSubmit)
// 		student.POST("/sync", handler.SyncOfflineAnswers)
// 		student.GET("/result/:examId", handler.GetStudentExamResultForUser)
// 		student.GET("/review/:attemptId", handler.GetStudentExamReviewForUser)
// 		student.GET("/termly-result", handler.GetStudentTermlyResultForUser)
// 		student.GET("/report-card", handler.GetTermlyReportCardForUser)
// 		student.POST("/practice", handler.StartPracticeForUser)
// 	}
// }

// // ============================================
// // EXAM ROUTES - ALL 23 ENDPOINTS
// // ============================================
// func setupExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler) {
// 	exams := rg.Group("/exams")
// 	exams.Use(middleware.AuthMiddleware())
// 	{
// 		exams.POST("/create", handler.CreateExam)
// 		exams.GET("/:id", handler.GetExam)
// 		exams.GET("/list", handler.ListExams)
// 		exams.PUT("/:id", handler.UpdateExam)
// 		exams.DELETE("/:id", handler.DeleteExam)
// 		exams.POST("/:id/questions", handler.AddQuestionsToExam)
// 		exams.DELETE("/:id/questions/:questionId", handler.RemoveQuestionFromExam)
// 		exams.POST("/:id/assign", handler.AssignExam)
// 		exams.GET("/:id/assignments", handler.GetExamAssignments)
// 		exams.POST("/start", handler.StartExam)
// 		exams.POST("/save-answer", handler.SaveAnswer)
// 		exams.POST("/bulk-answer", handler.BulkSaveAnswers)
// 		exams.POST("/submit", handler.SubmitExam)
// 		exams.GET("/:id/results/class", handler.GetClassResults)
// 		exams.GET("/:id/results/export", handler.ExportResults)
// 		exams.GET("/:id/results/rankings", handler.GetExamRankings)
// 		exams.GET("/:id/statistics", handler.GetExamStatistics)
// 		exams.GET("/students/:studentId/performance", handler.GetStudentPerformance)
// 		exams.GET("/students/:studentId/exams/:examId/result", handler.GetStudentExamResult)
// 		exams.GET("/practice/:studentId/:subjectId", handler.StartPractice)
// 		exams.GET("/teacher/subjects/:subjectId/results", handler.GetTeacherSubjectResults)
// 		exams.GET("/teacher/classes/:classId/results", handler.GetTeacherClassResults)
// 		exams.GET("/teacher/performance/dashboard", handler.GetTeacherDashboard)
// 		exams.GET("/school/performance/overview", handler.GetSchoolPerformanceOverview)
// 		exams.POST("/create-with-context", handler.CreateExamWithContext)
// 		exams.GET("/subject-questions", handler.GetSubjectQuestionsForExam)
// 		exams.POST("/:id/bulk-questions", handler.BulkAddQuestionsToExam)
// 		exams.GET("/:id/preview", handler.PreviewExam)
// 		exams.POST("/:id/publish", handler.PublishExam)
// 	}
// }

// // ============================================
// // ACTOR ROUTES (Admin, Teacher, Parent)
// // ============================================
// func setupActorRoutes(rg *gin.RouterGroup,
// 	adminH *adminHandler.AdminHandler,
// 	teacherH *teacherHandler.TeacherHandler,
// 	parentH *parentHandler.ParentHandler) {

// 	admin := rg.Group("/admin")
// 	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
// 	{
// 		admin.POST("/teachers/assign", adminH.AssignTeacher)
// 		admin.DELETE("/teachers/unassign/:classId", adminH.UnassignTeacher)
// 		admin.GET("/users", adminH.ListUsers)
// 		admin.GET("/students", adminH.ListStudents)
// 		admin.GET("/classes", adminH.ListClasses)
// 		admin.GET("/teachers", adminH.ListTeachers)
// 		admin.DELETE("/students/:id/permanent", adminH.HardDeleteStudent)
// 	}

// 	teacher := rg.Group("/teacher")
// 	teacher.Use(middleware.AuthMiddleware(), middleware.TeacherOnly())
// 	{
// 		teacher.POST("/students", teacherH.CreateStudent)
// 		teacher.POST("/students/bulk", teacherH.BulkCreateStudents)
// 		teacher.GET("/students", teacherH.GetMyStudents)
// 		teacher.GET("/students/credentials", teacherH.GetAllStudentsWithCredentials)
// 		teacher.GET("/students/:id", teacherH.GetStudent)
// 		teacher.PUT("/students/:id", teacherH.UpdateStudent)
// 		teacher.POST("/students/:id/reset-password", teacherH.ResetPassword)
// 		teacher.POST("/students/:id/deactivate", teacherH.DeactivateStudent)
// 	}

// 	parent := rg.Group("/parent")
// 	parent.Use(middleware.AuthMiddleware(), middleware.ParentOnly())
// 	{
// 		parent.GET("/children", parentH.GetChildren)
// 		parent.GET("/child/:studentId/results", parentH.GetChildResults)
// 	}
// }

// // ============================================
// // AUTH ROUTES
// // ============================================
// func setupAuthRoutes(rg *gin.RouterGroup, handler *authHandler.AuthHandler) {
// 	auth := rg.Group("/auth")
// 	{
// 		auth.POST("/register", handler.Register)
// 		auth.POST("/login", handler.Login)
// 		auth.POST("/verify-2fa", handler.Verify2FALogin)
// 		auth.POST("/refresh", handler.RefreshToken)
// 		auth.POST("/forgot-password", handler.ForgotPassword)
// 		auth.POST("/reset-password", handler.ResetPassword)
// 		auth.POST("/send-otp", handler.SendVerificationOTP)
// 		auth.POST("/verify-email", handler.VerifyEmail)
// 	}

// 	protected := rg.Group("/auth")
// 	protected.Use(middleware.AuthMiddleware())
// 	{
// 		protected.POST("/logout", handler.Logout)
// 		protected.POST("/logout-all", handler.LogoutAllDevices)
// 		protected.GET("/sessions", handler.GetSessions)
// 		protected.DELETE("/sessions/:sessionId", handler.RevokeSession)
// 		protected.POST("/change-password", handler.ChangePassword)
// 		protected.POST("/2fa/generate", handler.Generate2FA)
// 		protected.POST("/2fa/enable", handler.Enable2FA)
// 		protected.POST("/2fa/disable", handler.Disable2FA)
// 		protected.GET("/profile", handler.GetProfile)
// 		protected.PUT("/profile", handler.UpdateProfile)
// 		protected.DELETE("/profile", handler.DeleteAccount)
// 		protected.GET("/subscription/check", handler.CheckSubscription)
// 		protected.GET("/status", handler.GetAuthStatus)
// 	}
// }

// // ============================================
// // ACADEMIC ROUTES
// // ============================================
// func setupAcademicRoutes(rg *gin.RouterGroup,
// 	schoolHandler *academicHandler.SchoolHandler,
// 	sessionHandler *academicHandler.SessionHandler,
// 	termHandler *academicHandler.TermHandler,
// 	classLevelHandler *academicHandler.ClassLevelHandler,
// 	classArmHandler *academicHandler.ClassArmHandler,
// 	classHandler *academicHandler.ClassHandler,
// 	studentHandler *academicHandler.StudentHandler) {

// 	academic := rg.Group("/")
// 	academic.Use(middleware.AuthMiddleware())
// 	{
// 		academic.POST("/schools", schoolHandler.CreateSchool)
// 		academic.GET("/schools", schoolHandler.GetAllSchools)
// 		academic.GET("/schools/:id", schoolHandler.GetSchool)
// 		academic.PUT("/schools/:id", schoolHandler.UpdateSchool)
// 		academic.DELETE("/schools/:id", schoolHandler.DeleteSchool)

// 		academic.POST("/sessions", sessionHandler.CreateSession)
// 		academic.GET("/school-sessions/:schoolId", sessionHandler.GetSessions)
// 		academic.GET("/school-sessions/:schoolId/current", sessionHandler.GetCurrentSession)
// 		academic.GET("/sessions/:id", sessionHandler.GetSession)
// 		academic.PUT("/sessions/:id", sessionHandler.UpdateSession)
// 		academic.DELETE("/sessions/:id", sessionHandler.DeleteSession)

// 		academic.GET("/sessions", sessionHandler.ListSessions)
// 		academic.GET("/sessions/stats", sessionHandler.GetSessionStats)
// 		academic.GET("/sessions/timeline", sessionHandler.GetSessionTimeline)
// 		academic.GET("/sessions/summary", sessionHandler.GetSessionSummary)
// 		academic.GET("/sessions/search", sessionHandler.SearchSessions)
// 		academic.DELETE("/sessions/bulk", sessionHandler.BulkDeleteSessions)

// 		academic.GET("/terms/list", termHandler.ListAllTerms)
// 		academic.GET("/terms/school", termHandler.ListAllTermsBySchool)
// 		academic.GET("/terms/all", termHandler.GetAllTerms)
// 		academic.GET("/terms", termHandler.ListTerms)
// 		academic.GET("/terms/stats", termHandler.GetTermStats)
// 		academic.GET("/terms/search", termHandler.SearchTerms)
// 		academic.DELETE("/terms/bulk", termHandler.BulkDeleteTerms)
// 		academic.POST("/terms", termHandler.CreateTerm)
// 		academic.GET("/session-terms/:sessionId", termHandler.GetTerms)
// 		academic.GET("/session-terms/:sessionId/current", termHandler.GetCurrentTerm)
// 		academic.GET("/terms/:id", termHandler.GetTerm)
// 		academic.PUT("/terms/:id", termHandler.UpdateTerm)
// 		academic.DELETE("/terms/:id", termHandler.DeleteTerm)

// 		academic.GET("/class-levels", classLevelHandler.ListClassLevels)
// 		academic.GET("/class-levels/stats", classLevelHandler.GetClassLevelStats)
// 		academic.GET("/class-levels/search", classLevelHandler.SearchClassLevels)
// 		academic.DELETE("/class-levels/bulk", classLevelHandler.BulkDeleteClassLevels)
// 		academic.POST("/class-levels", classLevelHandler.CreateClassLevel)
// 		academic.GET("/class-levels/:id", classLevelHandler.GetClassLevel)
// 		academic.PUT("/class-levels/:id", classLevelHandler.UpdateClassLevel)
// 		academic.DELETE("/class-levels/:id", classLevelHandler.DeleteClassLevel)
// 		academic.GET("/school-class-levels/:schoolId", classLevelHandler.GetClassLevelsBySchool)

// 		academic.GET("/class-arms", classArmHandler.ListClassArms)
// 		academic.GET("/class-arms/stats", classArmHandler.GetClassArmStats)
// 		academic.GET("/class-arms/search", classArmHandler.SearchClassArms)
// 		academic.DELETE("/class-arms/bulk", classArmHandler.BulkDeleteClassArms)
// 		academic.POST("/class-arms", classArmHandler.CreateClassArm)
// 		academic.GET("/school-class-arms/:schoolId", classArmHandler.GetClassArms)
// 		academic.GET("/class-arms/:id", classArmHandler.GetClassArm)
// 		academic.PUT("/class-arms/:id", classArmHandler.UpdateClassArm)
// 		academic.DELETE("/class-arms/:id", classArmHandler.DeleteClassArm)

// 		academic.GET("/classes", classHandler.ListClasses)
// 		academic.GET("/classes/teacher/:teacherId", classHandler.GetClassesByTeacher)
// 		academic.GET("/classes/arm/:classArmId", classHandler.GetClassesByArm)
// 		academic.GET("/classes/level/:classLevelId", classHandler.GetClassesByLevel)
// 		academic.DELETE("/classes/bulk", classHandler.BulkDeleteClasses)
// 		academic.GET("/classes/stats", classHandler.GetClassStats)
// 		academic.GET("/classes/search", classHandler.SearchClasses)

// 		academic.POST("/classes", classHandler.CreateClass)
// 		academic.GET("/school-classes/:schoolId", classHandler.GetClassesBySchool)
// 		academic.GET("/session-classes/:sessionId", classHandler.GetClassesBySession)
// 		academic.GET("/school-session-classes/:schoolId/:sessionId", classHandler.GetClassesBySchoolAndSession)
// 		academic.GET("/classes/:id", classHandler.GetClass)
// 		academic.PUT("/classes/:id", classHandler.UpdateClass)
// 		academic.DELETE("/classes/:id", classHandler.DeleteClass)

// 		academic.GET("/student/profile", studentHandler.GetStudentProfile)

// 		academic.POST("/students", studentHandler.CreateStudent)
// 		academic.GET("/school-students/:schoolId", studentHandler.GetStudentsBySchool)
// 		academic.GET("/class-students/:classId", studentHandler.GetStudentsByClass)
// 		academic.GET("/students/:id", studentHandler.GetStudent)
// 		academic.GET("/user-student/:userId", studentHandler.GetStudentByUser)
// 		academic.PUT("/students/:id", studentHandler.UpdateStudent)
// 		academic.DELETE("/students/:id", studentHandler.DeleteStudent)
// 		academic.POST("/students/:studentId/transfer", studentHandler.TransferClass)
// 	}
// }

// // ============================================
// // SUBSCRIPTION ROUTES
// // ============================================
// func setupSubscriptionRoutes(rg *gin.RouterGroup, handler *subscriptionHandler.SubscriptionHandler) {
// 	subscription := rg.Group("/subscriptions")
// 	subscription.Use(middleware.AuthMiddleware())
// 	{
// 		subscription.POST("", handler.CreateSubscription)
// 		subscription.GET("", handler.GetSubscriptions)
// 		subscription.GET("/:id", handler.GetSubscription)
// 		subscription.GET("/school/:schoolId/current", handler.GetCurrentSubscription)
// 		subscription.PUT("/:id", handler.UpdateSubscription)
// 		subscription.POST("/:id/cancel", handler.CancelSubscription)
// 		subscription.POST("/:id/renew", handler.RenewSubscription)
// 		subscription.GET("/:id/usage", handler.GetSubscriptionUsage)
// 		subscription.GET("/:id/invoices", handler.GetInvoices)
// 		subscription.GET("/:id/transactions", handler.GetTransactions)
// 		subscription.POST("/:id/payment-intent", handler.CreatePaymentIntent)
// 		subscription.POST("/payment-intent/:id/confirm", handler.ConfirmPaymentIntent)
// 		subscription.POST("/verify", handler.VerifyPayment)
// 	}
// 	rg.POST("/webhook/:gateway", handler.HandleWebhook)
// }

// // ============================================
// // CBT QUESTION BANK ROUTES
// // ============================================
// func setupCBTQuestionRoutes(rg *gin.RouterGroup, handler *cbtQuestionHandler.QuestionHandler) {
// 	q := rg.Group("/questions")
// 	q.Use(middleware.AuthMiddleware())
// 	{
// 		q.POST("/create", handler.CreateQuestion)
// 		q.GET("/:id", handler.GetQuestion)
// 		q.PUT("/update/:id", handler.UpdateQuestion)
// 		q.DELETE("/delete/:id", handler.DeleteQuestion)

// 		q.GET("/list", handler.ListQuestions)
// 		q.POST("/filter", handler.FilterQuestions)

// 		q.POST("/bulk", handler.BulkCreateQuestions)
// 		q.POST("/bulk-upload", handler.BulkUploadFile)
// 		q.POST("/bulk-delete", handler.BulkDelete)
// 		q.PUT("/bulk-status", handler.BulkUpdateStatus)

// 		q.POST("/tags/create", handler.CreateTag)
// 		q.GET("/tags/list", handler.ListTags)

// 		q.GET("/statistics", handler.GetStatistics)

// 		q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
// 		q.POST("/extract", handler.ExtractQuestionsFromText)
// 		q.GET("/jobs/:id", handler.GetJobStatus)

// 		q.GET("/by-term", handler.GetQuestionsByTerm)
// 		q.GET("/by-session", handler.GetQuestionsBySession)
// 		q.GET("/by-class", handler.GetQuestionsByClass)
// 		q.GET("/for-exam", handler.GetQuestionsForExam)
// 		q.GET("/context/summary", handler.GetQuestionContextSummary)
// 		q.POST("/with-context", handler.GetQuestionsWithContext)
// 		q.GET("/academic/current", handler.GetCurrentAcademicContext)

// 		// NEW GROUPED QUESTION ROUTES
// 		q.POST("/grouped", handler.GetQuestionsGrouped)
// 		q.GET("/by-exam-type", handler.GetQuestionsByExamType)
// 		q.GET("/by-term-grouped", handler.GetQuestionsByTermGrouped)
// 		q.GET("/by-session-grouped", handler.GetQuestionsBySessionGrouped)
// 		q.GET("/all-grouped", handler.GetAllQuestionsGrouped)
// 	}
// }

// // ============================================
// // INITIALIZERS
// // ============================================

// func initSubscriptionHandler() *subscriptionHandler.SubscriptionHandler {
// 	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
// 	paymentService := payment.NewPaymentService()
// 	emailService := email.NewEmailService()
// 	subService := subscriptionService.NewSubscriptionService(subRepo, paymentService, emailService)
// 	return subscriptionHandler.NewSubscriptionHandler(subService)
// }

// func initStudentHandler() *academicHandler.StudentHandler {
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	userRepo := academicRepo.NewUserRepository(database.DB)
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	service := academicService.NewStudentService(studentRepo, userRepo, classRepo)
// 	return academicHandler.NewStudentHandler(service)
// }

// func initClassHandler() *academicHandler.ClassHandler {
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	classLevelRepo := academicRepo.NewClassLevelRepository(database.DB)
// 	classArmRepo := academicRepo.NewClassArmRepository(database.DB)
// 	sessionRepo := academicRepo.NewSessionRepository(database.DB)
// 	service := academicService.NewClassService(classRepo, classLevelRepo, classArmRepo, sessionRepo)
// 	return academicHandler.NewClassHandler(service)
// }

// func initClassLevelHandler() *academicHandler.ClassLevelHandler {
// 	repo := academicRepo.NewClassLevelRepository(database.DB)
// 	service := academicService.NewClassLevelService(repo)
// 	return academicHandler.NewClassLevelHandler(service)
// }

// func initClassArmHandler() *academicHandler.ClassArmHandler {
// 	repo := academicRepo.NewClassArmRepository(database.DB)
// 	service := academicService.NewClassArmService(repo)
// 	return academicHandler.NewClassArmHandler(service)
// }

// func initTermHandler() *academicHandler.TermHandler {
// 	repo := academicRepo.NewTermRepository(database.DB)
// 	service := academicService.NewTermService(repo)
// 	return academicHandler.NewTermHandler(service)
// }

// func initSessionHandler() *academicHandler.SessionHandler {
// 	repo := academicRepo.NewSessionRepository(database.DB)
// 	service := academicService.NewSessionService(repo)
// 	return academicHandler.NewSessionHandler(service)
// }

// func initSchoolHandler() *academicHandler.SchoolHandler {
// 	repo := academicRepo.NewSchoolRepository(database.DB)
// 	service := academicService.NewSchoolService(repo)
// 	return academicHandler.NewSchoolHandler(service)
// }

// func initAuthHandler() *authHandler.AuthHandler {
// 	repo := authRepo.NewAuthRepository(database.DB)
// 	parentSvc := initParentService()

// 	schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	parentRepo := academicRepo.NewParentRepository(database.DB)
// 	subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)

// 	logger, _ := zap.NewProduction()

// 	svc := authService.NewAuthService(
// 		repo,
// 		parentSvc,
// 		schoolRepo,
// 		studentRepo,
// 		parentRepo,
// 		subRepo,
// 		logger,
// 	)
// 	return authHandler.NewAuthHandler(svc)
// }

// func initExamHandler() *cbtExamHandler.ExamHandler {
// 	examRepo := cbtExamRepo.NewExamRepository(database.DB)
// 	questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// 	examService := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)
// 	return cbtExamHandler.NewExamHandler(examService)
// }

// func initQuestionHandler(queue queue.Queue, engine *engine.Engine) *cbtQuestionHandler.QuestionHandler {
// 	repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// 	subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// 	service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB, queue, engine)
// 	return cbtQuestionHandler.NewQuestionHandler(service)
// }

// func initAdminHandler() *adminHandler.AdminHandler {
// 	userRepo := academicRepo.NewUserRepository(database.DB)
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	svc := adminService.NewAdminService(userRepo, classRepo, studentRepo, database.DB)
// 	return adminHandler.NewAdminHandler(svc)
// }

// func initTeacherHandler() *teacherHandler.TeacherHandler {
// 	userRepo := academicRepo.NewUserRepository(database.DB)
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// 	logger, _ := zap.NewProduction()
// 	svc := teacherService.NewTeacherService(userRepo, studentRepo, classRepo, schoolRepo, database.DB, logger)
// 	return teacherHandler.NewTeacherHandler(svc)
// }

// func initParentHandler() *parentHandler.ParentHandler {
// 	parentRepo := academicRepo.NewParentRepository(database.DB)
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	examRepo := cbtExamRepo.NewExamRepository(database.DB)
// 	userRepo := academicRepo.NewUserRepository(database.DB)
// 	logger, _ := zap.NewProduction()
// 	svc := parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// 	return parentHandler.NewParentHandler(svc)
// }

// func initParentService() *parentService.ParentService {
// 	parentRepo := academicRepo.NewParentRepository(database.DB)
// 	studentRepo := academicRepo.NewStudentRepository(database.DB)
// 	classRepo := academicRepo.NewClassRepository(database.DB)
// 	examRepo := cbtExamRepo.NewExamRepository(database.DB)
// 	userRepo := academicRepo.NewUserRepository(database.DB)
// 	logger, _ := zap.NewProduction()
// 	return parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// }

// // ============================================
// // PRINT ROUTES - UPDATED WITH ONBOARDING
// // ============================================
// func PrintRoutes() {
// 	println("")
// 	println("========================================")
// 	println("📋 AVAILABLE API ENDPOINTS")
// 	println("========================================")
// 	println("")
// 	println("🔓 PUBLIC ROUTES:")
// 	println("   POST   /api/v1/auth/register")
// 	println("   POST   /api/v1/auth/login")
// 	println("   POST   /api/v1/auth/verify-2fa")
// 	println("   POST   /api/v1/auth/refresh")
// 	println("   POST   /api/v1/auth/forgot-password")
// 	println("   POST   /api/v1/auth/reset-password")
// 	println("   POST   /api/v1/auth/send-otp")
// 	println("   POST   /api/v1/auth/verify-email")
// 	println("")
// 	println("🚀 ONBOARDING ROUTES (Public - No Auth):")
// 	println("   GET    /api/v1/onboarding/gateways")
// 	println("   POST   /api/v1/onboarding/start")
// 	println("   POST   /api/v1/onboarding/admin")
// 	println("   POST   /api/v1/onboarding/verify-email")
// 	println("   POST   /api/v1/onboarding/school")
// 	println("   POST   /api/v1/onboarding/subscription")
// 	println("   GET    /api/v1/onboarding/status/:session_id")
// 	println("   GET    /api/v1/onboarding/payment-status/:session_id")
// 	println("   GET    /api/v1/onboarding/session/email/:email")
// 	println("   POST   /api/v1/onboarding/resend-otp")
// 	println("   POST   /api/v1/onboarding/resend-payment-link")
// 	println("   POST   /api/v1/onboarding/webhook/:gateway")
// 	println("   GET    /api/v1/onboarding/health")
// 	println("")
// 	println("🔗 WEBHOOK (Public):")
// 	println("   POST   /api/v1/webhook/:gateway")
// 	println("")
// 	println("🔒 PROTECTED ROUTES (Bearer Token Required):")
// 	println("   POST   /api/v1/auth/logout")
// 	println("   POST   /api/v1/auth/logout-all")
// 	println("   GET    /api/v1/auth/sessions")
// 	println("   DELETE /api/v1/auth/sessions/:sessionId")
// 	println("   POST   /api/v1/auth/change-password")
// 	println("   POST   /api/v1/auth/2fa/generate")
// 	println("   POST   /api/v1/auth/2fa/enable")
// 	println("   POST   /api/v1/auth/2fa/disable")
// 	println("   GET    /api/v1/auth/profile")
// 	println("   PUT    /api/v1/auth/profile")
// 	println("   DELETE /api/v1/auth/profile")
// 	println("   GET    /api/v1/auth/subscription/check")
// 	println("   GET    /api/v1/auth/status")
// 	println("")
// 	println("🏫 ACADEMIC ROUTES (Protected):")
// 	println("   POST   /api/v1/schools")
// 	println("   GET    /api/v1/schools")
// 	println("   GET    /api/v1/schools/:id")
// 	println("   PUT    /api/v1/schools/:id")
// 	println("   DELETE /api/v1/schools/:id")
// 	println("   POST   /api/v1/sessions")
// 	println("   GET    /api/v1/school-sessions/:schoolId")
// 	println("   GET    /api/v1/sessions/:id")
// 	println("   POST   /api/v1/terms")
// 	println("   GET    /api/v1/session-terms/:sessionId")
// 	println("   POST   /api/v1/class-levels")
// 	println("   GET    /api/v1/school-class-levels/:schoolId")
// 	println("   POST   /api/v1/class-arms")
// 	println("   GET    /api/v1/school-class-arms/:schoolId")
// 	println("   POST   /api/v1/classes")
// 	println("   GET    /api/v1/school-classes/:schoolId")
// 	println("   POST   /api/v1/students")
// 	println("   GET    /api/v1/school-students/:schoolId")
// 	println("")
// 	println("📚 CBT SUBJECT ROUTES (Protected):")
// 	println("   POST   /api/v1/subjects/create")
// 	println("   GET    /api/v1/subjects/list")
// 	println("   GET    /api/v1/subjects/active")
// 	println("   GET    /api/v1/subjects/view/:id")
// 	println("   PUT    /api/v1/subjects/update/:id")
// 	println("   DELETE /api/v1/subjects/delete/:id")
// 	println("")
// 	println("📝 CBT EXAM ROUTES (Protected):")
// 	println("   📋 EXAM MANAGEMENT (9 Endpoints):")
// 	println("   POST   /api/v1/exams/create")
// 	println("   GET    /api/v1/exams/:id")
// 	println("   GET    /api/v1/exams/list")
// 	println("   PUT    /api/v1/exams/:id")
// 	println("   DELETE /api/v1/exams/:id")
// 	println("   POST   /api/v1/exams/:id/questions")
// 	println("   DELETE /api/v1/exams/:id/questions/:questionId")
// 	println("   POST   /api/v1/exams/:id/assign")
// 	println("   GET    /api/v1/exams/:id/assignments")
// 	println("")
// 	println("   ✍️ EXAM TAKING (3 Endpoints):")
// 	println("   POST   /api/v1/exams/start")
// 	println("   POST   /api/v1/exams/save-answer")
// 	println("   POST   /api/v1/exams/submit")
// 	println("")
// 	println("   📊 RESULTS & PERFORMANCE (4 Endpoints):")
// 	println("   GET    /api/v1/exams/:id/results/class")
// 	println("   GET    /api/v1/exams/:id/results/export")
// 	println("   GET    /api/v1/exams/:id/results/rankings")
// 	println("   GET    /api/v1/exams/:id/statistics")
// 	println("")
// 	println("   🎓 STUDENT PERFORMANCE (2 Endpoints):")
// 	println("   GET    /api/v1/exams/students/:studentId/performance")
// 	println("   GET    /api/v1/exams/students/:studentId/exams/:examId/result")
// 	println("")
// 	println("   🏋️ PRACTICE (1 Endpoint):")
// 	println("   GET    /api/v1/exams/practice/:studentId/:subjectId")
// 	println("")
// 	println("   👨‍🏫 TEACHER VIEWS (3 Endpoints):")
// 	println("   GET    /api/v1/exams/teacher/subjects/:subjectId/results")
// 	println("   GET    /api/v1/exams/teacher/classes/:classId/results")
// 	println("   GET    /api/v1/exams/teacher/performance/dashboard")
// 	println("")
// 	println("   🛡️ ADMIN VIEWS (1 Endpoint):")
// 	println("   GET    /api/v1/exams/school/performance/overview")
// 	println("")
// 	println("👨‍🏫 TEACHER ROUTES (Protected):")
// 	println("   POST   /api/v1/teacher/students")
// 	println("   POST   /api/v1/teacher/students/bulk")
// 	println("   GET    /api/v1/teacher/students")
// 	println("   GET    /api/v1/teacher/students/:id")
// 	println("   PUT    /api/v1/teacher/students/:id")
// 	println("   POST   /api/v1/teacher/students/:id/reset-password")
// 	println("   POST   /api/v1/teacher/students/:id/deactivate")
// 	println("")
// 	println("👪 PARENT ROUTES (Protected):")
// 	println("   GET    /api/v1/parent/children")
// 	println("   GET    /api/v1/parent/child/:studentId/results")
// 	println("")
// 	println("🛡️ ADMIN ROUTES (Protected):")
// 	println("   POST   /api/v1/admin/teachers/assign")
// 	println("   DELETE /api/v1/admin/teachers/unassign/:classId")
// 	println("   GET    /api/v1/admin/users")
// 	println("   GET    /api/v1/admin/students")
// 	println("   GET    /api/v1/admin/classes")
// 	println("   GET    /api/v1/admin/teachers")
// 	println("   DELETE /api/v1/admin/students/:id/permanent")
// 	println("   POST   /api/v1/onboarding/admin/complete/:session_id")
// 	println("   DELETE /api/v1/onboarding/admin/cleanup")
// 	println("   GET    /api/v1/onboarding/admin/stats")
// 	println("")
// 	println("💰 SUBSCRIPTION ROUTES (Protected):")
// 	println("   POST   /api/v1/subscriptions")
// 	println("   GET    /api/v1/subscriptions")
// 	println("   GET    /api/v1/subscriptions/:id")
// 	println("   GET    /api/v1/subscriptions/school/:schoolId/current")
// 	println("   PUT    /api/v1/subscriptions/:id")
// 	println("   POST   /api/v1/subscriptions/:id/cancel")
// 	println("   POST   /api/v1/subscriptions/:id/renew")
// 	println("   GET    /api/v1/subscriptions/:id/usage")
// 	println("   GET    /api/v1/subscriptions/:id/invoices")
// 	println("   GET    /api/v1/subscriptions/:id/transactions")
// 	println("   POST   /api/v1/subscriptions/:id/payment-intent")
// 	println("   POST   /api/v1/subscriptions/payment-intent/:id/confirm")
// 	println("   POST   /api/v1/subscriptions/verify")
// 	println("")
// 	println("📚 CBT QUESTION BANK ROUTES (Protected):")
// 	println("   POST   /api/v1/questions/create")
// 	println("   GET    /api/v1/questions/:id")
// 	println("   PUT    /api/v1/questions/update/:id")
// 	println("   DELETE /api/v1/questions/delete/:id")
// 	println("   GET    /api/v1/questions/list")
// 	println("   POST   /api/v1/questions/filter")
// 	println("   POST   /api/v1/questions/bulk")
// 	println("   POST   /api/v1/questions/bulk-upload")
// 	println("   POST   /api/v1/questions/bulk-delete")
// 	println("   GET    /api/v1/questions/statistics")
// 	println("   POST   /api/v1/questions/tags/create")
// 	println("   GET    /api/v1/questions/tags/list")
// 	println("   POST   /api/v1/questions/ai/generate")
// 	println("   POST   /api/v1/questions/extract")
// 	println("   GET    /api/v1/questions/jobs/:id")
// 	println("")
// 	println("========================================")
// }


// // package routes

// // import (
// //     academicHandler "cbt-api/internal/academic/handler"
// //     academicRepo "cbt-api/internal/academic/repository"
// //     academicService "cbt-api/internal/academic/service"
// //     authHandler "cbt-api/internal/auth/handler"
// //     authRepo "cbt-api/internal/auth/repository"
// //     authService "cbt-api/internal/auth/service"
// //     "cbt-api/internal/middleware"
// //     subscriptionHandler "cbt-api/internal/subscription/handler"
// //     subscriptionRepo "cbt-api/internal/subscription/repository"
// //     subscriptionService "cbt-api/internal/subscription/service"

// //     // CBT modules
// //     cbtExamHandler "cbt-api/internal/cbt/handler"
// //     cbtExamRepo "cbt-api/internal/cbt/repository"
// //     cbtExamService "cbt-api/internal/cbt/service"
// //     cbtQuestionHandler "cbt-api/internal/cbt/handler"
// //     cbtQuestionRepo "cbt-api/internal/cbt/repository"
// //     cbtQuestionService "cbt-api/internal/cbt/service"
    
// //     cbtSubjectHandler "cbt-api/internal/cbt/handler"
// //     cbtSubjectRepo "cbt-api/internal/cbt/repository"
// //     cbtSubjectService "cbt-api/internal/cbt/service"

// //     // NEW actor modules
// //     adminHandler "cbt-api/internal/admin/handler"
// //     adminService "cbt-api/internal/admin/service"
// //     teacherHandler "cbt-api/internal/teacher/handler"
// //     teacherService "cbt-api/internal/teacher/service"
// //     parentHandler "cbt-api/internal/parent/handler"
// //     parentService "cbt-api/internal/parent/service"

// //     // ONBOARDING
// //     onboardingHandler "cbt-api/internal/onboarding/handler"
// //     onboardingService "cbt-api/internal/onboarding/service"

// //     "cbt-api/pkg/email"
// //     "cbt-api/pkg/payment"
// //     "cbt-api/pkg/database"
// //     "cbt-api/internal/ai/engine"
// //     "cbt-api/internal/ai/queue"
// //     "cbt-api/internal/models" 


// //     "github.com/gin-gonic/gin"
// //     "go.uber.org/zap"

// //     // Swagger
// //     swaggerFiles "github.com/swaggo/files"
// //     ginSwagger "github.com/swaggo/gin-swagger"
// // )

// // // SetupRoutes configures all API routes for the application
// // func SetupRoutes(r *gin.Engine, q queue.Queue, e *engine.Engine) {
// //     // Initialize all handlers
// //     authH := initAuthHandler()
// //     schoolH := initSchoolHandler()
// //     sessionH := initSessionHandler()
// //     termH := initTermHandler()
// //     classLevelH := initClassLevelHandler()
// //     classArmH := initClassArmHandler()
// //     classH := initClassHandler()
// //     studentH := initStudentHandler()
// //     subscriptionH := initSubscriptionHandler()

// //     // CBT handlers
// //     examH := initExamHandler()
// //     questionH := initQuestionHandler(q, e) 

// //     // Subject handler initialisation
// //     subjectRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// //     subjectService := cbtSubjectService.NewSubjectService(subjectRepo, database.DB)
// //     subjectHandler := cbtSubjectHandler.NewSubjectHandler(subjectService)

// //     // NEW actor handlers
// //     adminH := initAdminHandler()
// //     teacherH := initTeacherHandler()
// //     parentH := initParentHandler()
// //     // Initialize repositories for middleware
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)

// //     // ✅ ONBOARDING INITIALIZATION
// //     onboardingH := initOnboardingHandler()

// //     // API version 1 group
// //     v1 := r.Group("/api/v1")
// //     {
// //         // Health check
// //         v1.GET("/health", authH.HealthCheck)

// //         // Setup auth routes
// //         setupAuthRoutes(v1, authH)

// //         // Setup academic routes (existing)
// //         setupAcademicRoutes(v1, schoolH, sessionH, termH, classLevelH, classArmH, classH, studentH)

// //         // Setup subscription routes (existing)
// //         setupSubscriptionRoutes(v1, subscriptionH)

// //         // ✅ ONBOARDING ROUTES - COMPLETE
// //         setupOnboardingRoutes(v1, onboardingH)

// //         // ========== SUBJECT ROUTES ==========
// //         subject := v1.Group("/subjects")
// //         subject.Use(middleware.AuthMiddleware())
// //         {
// //             subject.POST("/create", subjectHandler.CreateSubject)
// //             subject.GET("/list", subjectHandler.ListSubjects)
// //             subject.GET("/active", subjectHandler.ListActiveSubjects)
// //             subject.GET("/view/:id", subjectHandler.GetSubject)
// //             subject.PUT("/update/:id", subjectHandler.UpdateSubject)
// //             subject.DELETE("/delete/:id", subjectHandler.DeleteSubject)
// //         }

// //         // ==================== EXAM ROUTES ====================
// //         // ALL 23 EXAM ENDPOINTS ORGANIZED UNDER /v1/exams
// //         setupExamRoutes(v1, examH)
// //         setupStudentExamRoutes(v1, examH, studentRepo)

// //         // ==================== QUESTION ROUTES ====================
// //         setupCBTQuestionRoutes(v1, questionH)

// //         // ==================== NEW ACTOR ROUTES ====================
// //         setupActorRoutes(v1, adminH, teacherH, parentH)
// //     }

// //     // Swagger UI endpoint (no version prefix, accessible directly)
// //     r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
// // }

// // // ============================================
// // // ONBOARDING ROUTES - COMPLETE
// // // ============================================
// // func setupOnboardingRoutes(rg *gin.RouterGroup, handler *onboardingHandler.OnboardingHandler) {
// //     onboarding := rg.Group("/onboarding")
// //     {
// //         // STEP 1: Get Available Gateways
// //         onboarding.GET("/gateways", handler.GetAvailableGateways)

// //         // STEP 1: Select Plan
// //         onboarding.POST("/start", handler.StartOnboarding)

// //         // STEP 2: Create Administrator Account
// //         onboarding.POST("/admin", handler.CreateAdminAccount)

// //         // STEP 3: Verify Email with OTP
// //         onboarding.POST("/verify-email", handler.VerifyEmail)

// //         // STEP 4: Register School Information
// //         onboarding.POST("/school", handler.RegisterSchool)

// //         // STEP 6: Create Subscription and Payment Intent
// //         onboarding.POST("/subscription", handler.CreateSubscription)

// //         // Status endpoints
// //         onboarding.GET("/status/:session_id", handler.GetStatus)
// //         onboarding.GET("/payment-status/:session_id", handler.CheckPaymentStatus)
// //         onboarding.GET("/session/email/:email", handler.GetSessionByEmail)

// //         // Resend endpoints
// //         onboarding.POST("/resend-otp", handler.ResendVerificationOTP)
// //         onboarding.POST("/resend-payment-link", handler.ResendPaymentLink)

// //         // Webhook endpoint
// //         onboarding.POST("/webhook/:gateway", handler.Webhook)

// //         // Admin endpoints (protected)
// //         admin := onboarding.Group("/admin")
// //         admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
// //         {
// //             admin.POST("/complete/:session_id", handler.CompleteOnboarding)
// //             admin.DELETE("/cleanup", handler.CleanupExpiredSessions)
// //             admin.GET("/stats", handler.GetOnboardingStats)
// //         }
// //     }
// // }

// // // ============================================
// // // INITIALIZE ONBOARDING HANDLER
// // // ============================================
// // func initOnboardingHandler() *onboardingHandler.OnboardingHandler {
// //     // Use existing repositories
// //     authRepo := authRepo.NewAuthRepository(database.DB)
// //     schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// //     subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
    
// //     // Initialize services
// //     paymentService := payment.NewPaymentService()
// //     emailService := email.NewEmailService()
    
// //     // Create session store (in-memory for now, can be replaced with Redis)
// //     sessionStore := onboardingService.NewInMemorySessionStore()
    
// //     // Create onboarding config
// //     config := &onboardingService.OnboardingConfig{
// //         AppURL:              "http://localhost:8080",
// //         DefaultGateway:      models.GatewayPaystack,
// //         AllowedGateways:     []models.PaymentGateway{models.GatewayPaystack, models.GatewayFlutterwave, models.GatewayStripe},
// //         SessionExpiryHours:  24,
// //         PaymentExpiryHours:  48,
// //         EnableEmail:         true,
// //         EnableWebhooks:      true,
// //         WebhookSecret:       "your-webhook-secret",
// //         RetryAttempts:       3,
// //     }
    
// //     // Create auth repository wrapper
// //     authRepoWrapper := &authRepoWrapper{authRepo: authRepo}
    
// //     // Create school repository wrapper
// //     schoolRepoWrapper := &schoolRepoWrapper{schoolRepo: schoolRepo}
    
// //     // Create user repository wrapper
// //     userRepoWrapper := &userRepoWrapper{authRepo: authRepo}
    
// //     // Create onboarding service
// //     svc := onboardingService.NewOnboardingService(
// //         subRepo,
// //         paymentService,
// //         emailService,
// //         authRepoWrapper,
// //         schoolRepoWrapper,
// //         userRepoWrapper,
// //         sessionStore,
// //         config,
// //     )
    
// //     return onboardingHandler.NewOnboardingHandler(svc)
// // }

// // // ============================================
// // // ONBOARDING REPOSITORY WRAPPERS
// // // ============================================

// // // authRepoWrapper wraps AuthRepository to implement AuthRepository interface
// // type authRepoWrapper struct {
// //     authRepo *authRepo.AuthRepository
// // }

// // func (r *authRepoWrapper) FindUserByEmail(email string) (*models.User, error) {
// //     return r.authRepo.FindUserByEmail(email)
// // }

// // func (r *authRepoWrapper) FindUserByID(id string) (*models.User, error) {
// //     return r.authRepo.FindUserByID(id)
// // }

// // func (r *authRepoWrapper) FindUserByUsername(username string) (*models.User, error) {
// //     return r.authRepo.FindUserByUsername(username)
// // }

// // func (r *authRepoWrapper) CreateUser(user *models.User) error {
// //     return r.authRepo.CreateUser(user)
// // }

// // func (r *authRepoWrapper) UpdateUser(user *models.User) error {
// //     return r.authRepo.UpdateUser(user)
// // }

// // func (r *authRepoWrapper) VerifyEmail(userID string) error {
// //     return r.authRepo.VerifyEmail(userID)
// // }

// // // schoolRepoWrapper wraps SchoolRepository to implement SchoolRepository interface
// // type schoolRepoWrapper struct {
// //     schoolRepo *academicRepo.SchoolRepository
// // }

// // func (r *schoolRepoWrapper) FindByID(id string) (*models.School, error) {
// //     return r.schoolRepo.FindByID(id)
// // }

// // func (r *schoolRepoWrapper) FindByName(name string) (*models.School, error) {
// //     return r.schoolRepo.FindByName(name)
// // }

// // func (r *schoolRepoWrapper) FindByCode(code string) (*models.School, error) {
// //     return r.schoolRepo.FindByCode(code)
// // }

// // func (r *schoolRepoWrapper) Create(school *models.School) error {
// //     return r.schoolRepo.Create(school)
// // }

// // func (r *schoolRepoWrapper) Update(school *models.School) error {
// //     return r.schoolRepo.Update(school)
// // }

// // // userRepoWrapper wraps AuthRepository to implement UserRepository interface
// // type userRepoWrapper struct {
// //     authRepo *authRepo.AuthRepository
// // }

// // func (r *userRepoWrapper) FindByEmail(email string) (*models.User, error) {
// //     return r.authRepo.FindUserByEmail(email)
// // }

// // func (r *userRepoWrapper) FindByID(id string) (*models.User, error) {
// //     return r.authRepo.FindUserByID(id)
// // }

// // func (r *userRepoWrapper) FindByUsername(username string) (*models.User, error) {
// //     return r.authRepo.FindUserByUsername(username)
// // }

// // func (r *userRepoWrapper) Create(user *models.User) error {
// //     return r.authRepo.CreateUser(user)
// // }

// // func (r *userRepoWrapper) Update(user *models.User) error {
// //     return r.authRepo.UpdateUser(user)
// // }

// // // ============================================
// // // THE REST OF YOUR FUNCTIONS (unchanged)
// // // ============================================

// // // setupStudentExamRoutes - All student exam routes with automatic context
// // func setupStudentExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, studentRepo *academicRepo.StudentRepository) {
// // 	student := rg.Group("/student/exams")
// // 	student.Use(middleware.AuthMiddleware())
// // 	student.Use(middleware.StudentContextMiddleware(studentRepo))
// // 	{
// // 		student.GET("/dashboard", handler.GetStudentDashboardForUser)
// // 		student.GET("/performance", handler.GetStudentPerformanceForUser)
// // 		student.POST("/start/:examId", handler.StartExamForUser)
// // 		student.GET("/attempt/:attemptId", handler.GetAttemptState)
// // 		student.POST("/answer", handler.SaveAnswer)
// // 		student.POST("/bulk-answer", handler.BulkSaveAnswers)
// // 		student.POST("/submit", handler.SubmitExam)
// // 		student.POST("/auto-save", handler.AutoSave)
// // 		student.POST("/offline-submit", handler.OfflineSubmit)
// // 		student.POST("/sync", handler.SyncOfflineAnswers)
// // 		student.GET("/result/:examId", handler.GetStudentExamResultForUser)
// // 		student.GET("/review/:attemptId", handler.GetStudentExamReviewForUser)
// // 		student.GET("/termly-result", handler.GetStudentTermlyResultForUser)
// // 		student.GET("/report-card", handler.GetTermlyReportCardForUser)
// // 		student.POST("/practice", handler.StartPracticeForUser)
// // 	}
// // }

// // // ============================================
// // // EXAM ROUTES - ALL 23 ENDPOINTS
// // // ============================================
// // func setupExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler) {
// //     exams := rg.Group("/exams")
// //     exams.Use(middleware.AuthMiddleware())
// //     {
// //         exams.POST("/create", handler.CreateExam)
// //         exams.GET("/:id", handler.GetExam)
// //         exams.GET("/list", handler.ListExams)
// //         exams.PUT("/:id", handler.UpdateExam)
// //         exams.DELETE("/:id", handler.DeleteExam)
// //         exams.POST("/:id/questions", handler.AddQuestionsToExam)
// //         exams.DELETE("/:id/questions/:questionId", handler.RemoveQuestionFromExam)
// //         exams.POST("/:id/assign", handler.AssignExam)
// //         exams.GET("/:id/assignments", handler.GetExamAssignments)
// //         exams.POST("/start", handler.StartExam)
// //         exams.POST("/save-answer", handler.SaveAnswer)
// //         exams.POST("/bulk-answer", handler.BulkSaveAnswers)
// //         exams.POST("/submit", handler.SubmitExam)
// //         exams.GET("/:id/results/class", handler.GetClassResults)
// //         exams.GET("/:id/results/export", handler.ExportResults)
// //         exams.GET("/:id/results/rankings", handler.GetExamRankings)
// //         exams.GET("/:id/statistics", handler.GetExamStatistics)
// //         exams.GET("/students/:studentId/performance", handler.GetStudentPerformance)
// //         exams.GET("/students/:studentId/exams/:examId/result", handler.GetStudentExamResult)
// //         exams.GET("/practice/:studentId/:subjectId", handler.StartPractice)
// //         exams.GET("/teacher/subjects/:subjectId/results", handler.GetTeacherSubjectResults)
// //         exams.GET("/teacher/classes/:classId/results", handler.GetTeacherClassResults)
// //         exams.GET("/teacher/performance/dashboard", handler.GetTeacherDashboard)
// //         exams.GET("/school/performance/overview", handler.GetSchoolPerformanceOverview)
// //         exams.POST("/create-with-context", handler.CreateExamWithContext)
// //         exams.GET("/subject-questions", handler.GetSubjectQuestionsForExam)
// //         exams.POST("/:id/bulk-questions", handler.BulkAddQuestionsToExam)
// //         exams.GET("/:id/preview", handler.PreviewExam)
// //         exams.POST("/:id/publish", handler.PublishExam)
// //     }
// // }

// // // ============================================
// // // ACTOR ROUTES (Admin, Teacher, Parent)
// // // ============================================
// // func setupActorRoutes(rg *gin.RouterGroup, 
// //     adminH *adminHandler.AdminHandler,
// //     teacherH *teacherHandler.TeacherHandler,
// //     parentH *parentHandler.ParentHandler) {

// //     admin := rg.Group("/admin")
// //     admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
// //     {
// //         admin.POST("/teachers/assign", adminH.AssignTeacher)
// //         admin.DELETE("/teachers/unassign/:classId", adminH.UnassignTeacher)
// //         admin.GET("/users", adminH.ListUsers)
// //         admin.GET("/students", adminH.ListStudents)
// //         admin.GET("/classes", adminH.ListClasses)
// //         admin.GET("/teachers", adminH.ListTeachers)
// //         admin.DELETE("/students/:id/permanent", adminH.HardDeleteStudent)
// //     }

// //     teacher := rg.Group("/teacher")
// //     teacher.Use(middleware.AuthMiddleware(), middleware.TeacherOnly())
// //     {
// //         teacher.POST("/students", teacherH.CreateStudent)
// //         teacher.POST("/students/bulk", teacherH.BulkCreateStudents)
// //         teacher.GET("/students", teacherH.GetMyStudents)
// //         teacher.GET("/students/credentials", teacherH.GetAllStudentsWithCredentials)
// //         teacher.GET("/students/:id", teacherH.GetStudent)
// //         teacher.PUT("/students/:id", teacherH.UpdateStudent)
// //         teacher.POST("/students/:id/reset-password", teacherH.ResetPassword)
// //         teacher.POST("/students/:id/deactivate", teacherH.DeactivateStudent)
// //     }

// //     parent := rg.Group("/parent")
// //     parent.Use(middleware.AuthMiddleware(), middleware.ParentOnly())
// //     {
// //         parent.GET("/children", parentH.GetChildren)
// //         parent.GET("/child/:studentId/results", parentH.GetChildResults)
// //     }
// // }

// // // ============================================
// // // AUTH ROUTES
// // // ============================================
// // func setupAuthRoutes(rg *gin.RouterGroup, handler *authHandler.AuthHandler) {
// //     auth := rg.Group("/auth")
// //     {
// //         auth.POST("/register", handler.Register)
// //         auth.POST("/login", handler.Login)
// //         auth.POST("/verify-2fa", handler.Verify2FALogin)
// //         auth.POST("/refresh", handler.RefreshToken)
// //         auth.POST("/forgot-password", handler.ForgotPassword)
// //         auth.POST("/reset-password", handler.ResetPassword)
// //         auth.POST("/send-otp", handler.SendVerificationOTP)
// //         auth.POST("/verify-email", handler.VerifyEmail)
// //     }

// //     protected := rg.Group("/auth")
// //     protected.Use(middleware.AuthMiddleware())
// //     {
// //         protected.POST("/logout", handler.Logout)
// //         protected.POST("/logout-all", handler.LogoutAllDevices)
// //         protected.GET("/sessions", handler.GetSessions)
// //         protected.DELETE("/sessions/:sessionId", handler.RevokeSession)
// //         protected.POST("/change-password", handler.ChangePassword)
// //         protected.POST("/2fa/generate", handler.Generate2FA)
// //         protected.POST("/2fa/enable", handler.Enable2FA)
// //         protected.POST("/2fa/disable", handler.Disable2FA)
// //         protected.GET("/profile", handler.GetProfile)
// //         protected.PUT("/profile", handler.UpdateProfile)
// //         protected.DELETE("/profile", handler.DeleteAccount)
// //         protected.GET("/subscription/check", handler.CheckSubscription)
// //     }
// // }

// // // ============================================
// // // ACADEMIC ROUTES
// // // ============================================
// // func setupAcademicRoutes(rg *gin.RouterGroup,
// //     schoolHandler *academicHandler.SchoolHandler,
// //     sessionHandler *academicHandler.SessionHandler,
// //     termHandler *academicHandler.TermHandler,
// //     classLevelHandler *academicHandler.ClassLevelHandler,
// //     classArmHandler *academicHandler.ClassArmHandler,
// //     classHandler *academicHandler.ClassHandler,
// //     studentHandler *academicHandler.StudentHandler) {

// //     academic := rg.Group("/")
// //     academic.Use(middleware.AuthMiddleware())
// //     {
// //         academic.POST("/schools", schoolHandler.CreateSchool)
// //         academic.GET("/schools", schoolHandler.GetAllSchools)
// //         academic.GET("/schools/:id", schoolHandler.GetSchool)
// //         academic.PUT("/schools/:id", schoolHandler.UpdateSchool)
// //         academic.DELETE("/schools/:id", schoolHandler.DeleteSchool)

// //         academic.POST("/sessions", sessionHandler.CreateSession)
// //         academic.GET("/school-sessions/:schoolId", sessionHandler.GetSessions)
// //         academic.GET("/school-sessions/:schoolId/current", sessionHandler.GetCurrentSession)
// //         academic.GET("/sessions/:id", sessionHandler.GetSession)
// //         academic.PUT("/sessions/:id", sessionHandler.UpdateSession)
// //         academic.DELETE("/sessions/:id", sessionHandler.DeleteSession)

// //         academic.GET("/sessions", sessionHandler.ListSessions)
// //         academic.GET("/sessions/stats", sessionHandler.GetSessionStats)
// //         academic.GET("/sessions/timeline", sessionHandler.GetSessionTimeline)
// //         academic.GET("/sessions/summary", sessionHandler.GetSessionSummary)
// //         academic.GET("/sessions/search", sessionHandler.SearchSessions)
// //         academic.DELETE("/sessions/bulk", sessionHandler.BulkDeleteSessions)

// //         academic.GET("/terms/list", termHandler.ListAllTerms)
// //         academic.GET("/terms/school", termHandler.ListAllTermsBySchool)
// //         academic.GET("/terms/all", termHandler.GetAllTerms)
// //         academic.GET("/terms", termHandler.ListTerms)
// //         academic.GET("/terms/stats", termHandler.GetTermStats)
// //         academic.GET("/terms/search", termHandler.SearchTerms)
// //         academic.DELETE("/terms/bulk", termHandler.BulkDeleteTerms)
// //         academic.POST("/terms", termHandler.CreateTerm)
// //         academic.GET("/session-terms/:sessionId", termHandler.GetTerms)
// //         academic.GET("/session-terms/:sessionId/current", termHandler.GetCurrentTerm)
// //         academic.GET("/terms/:id", termHandler.GetTerm)
// //         academic.PUT("/terms/:id", termHandler.UpdateTerm)
// //         academic.DELETE("/terms/:id", termHandler.DeleteTerm)

// //         academic.GET("/class-levels", classLevelHandler.ListClassLevels)
// //         academic.GET("/class-levels/stats", classLevelHandler.GetClassLevelStats)
// //         academic.GET("/class-levels/search", classLevelHandler.SearchClassLevels)
// //         academic.DELETE("/class-levels/bulk", classLevelHandler.BulkDeleteClassLevels)
// //         academic.POST("/class-levels", classLevelHandler.CreateClassLevel)
// //         academic.GET("/class-levels/:id", classLevelHandler.GetClassLevel)
// //         academic.PUT("/class-levels/:id", classLevelHandler.UpdateClassLevel)
// //         academic.DELETE("/class-levels/:id", classLevelHandler.DeleteClassLevel)
// //         academic.GET("/school-class-levels/:schoolId", classLevelHandler.GetClassLevelsBySchool)

// //         academic.GET("/class-arms", classArmHandler.ListClassArms)
// //         academic.GET("/class-arms/stats", classArmHandler.GetClassArmStats)
// //         academic.GET("/class-arms/search", classArmHandler.SearchClassArms)
// //         academic.DELETE("/class-arms/bulk", classArmHandler.BulkDeleteClassArms)
// //         academic.POST("/class-arms", classArmHandler.CreateClassArm)
// //         academic.GET("/school-class-arms/:schoolId", classArmHandler.GetClassArms)
// //         academic.GET("/class-arms/:id", classArmHandler.GetClassArm)
// //         academic.PUT("/class-arms/:id", classArmHandler.UpdateClassArm)
// //         academic.DELETE("/class-arms/:id", classArmHandler.DeleteClassArm)

// //         academic.GET("/classes", classHandler.ListClasses)
// //         academic.GET("/classes/teacher/:teacherId", classHandler.GetClassesByTeacher)
// //         academic.GET("/classes/arm/:classArmId", classHandler.GetClassesByArm)
// //         academic.GET("/classes/level/:classLevelId", classHandler.GetClassesByLevel)
// //         academic.DELETE("/classes/bulk", classHandler.BulkDeleteClasses)
// //         academic.GET("/classes/stats", classHandler.GetClassStats)
// //         academic.GET("/classes/search", classHandler.SearchClasses)

// //         academic.POST("/classes", classHandler.CreateClass)
// //         academic.GET("/school-classes/:schoolId", classHandler.GetClassesBySchool)
// //         academic.GET("/session-classes/:sessionId", classHandler.GetClassesBySession)
// //         academic.GET("/school-session-classes/:schoolId/:sessionId", classHandler.GetClassesBySchoolAndSession)
// //         academic.GET("/classes/:id", classHandler.GetClass)
// //         academic.PUT("/classes/:id", classHandler.UpdateClass)
// //         academic.DELETE("/classes/:id", classHandler.DeleteClass)

// //         academic.GET("/student/profile", studentHandler.GetStudentProfile)

// //         academic.POST("/students", studentHandler.CreateStudent)
// //         academic.GET("/school-students/:schoolId", studentHandler.GetStudentsBySchool)
// //         academic.GET("/class-students/:classId", studentHandler.GetStudentsByClass)
// //         academic.GET("/students/:id", studentHandler.GetStudent)
// //         academic.GET("/user-student/:userId", studentHandler.GetStudentByUser)
// //         academic.PUT("/students/:id", studentHandler.UpdateStudent)
// //         academic.DELETE("/students/:id", studentHandler.DeleteStudent)
// //         academic.POST("/students/:studentId/transfer", studentHandler.TransferClass)
// //     }
// // }

// // // ============================================
// // // SUBSCRIPTION ROUTES
// // // ============================================
// // func setupSubscriptionRoutes(rg *gin.RouterGroup, handler *subscriptionHandler.SubscriptionHandler) {
// //     subscription := rg.Group("/subscriptions")
// //     subscription.Use(middleware.AuthMiddleware())
// //     {
// //         subscription.POST("", handler.CreateSubscription)
// //         subscription.GET("", handler.GetSubscriptions)
// //         subscription.GET("/:id", handler.GetSubscription)
// //         subscription.GET("/school/:schoolId/current", handler.GetCurrentSubscription)
// //         subscription.PUT("/:id", handler.UpdateSubscription)
// //         subscription.POST("/:id/cancel", handler.CancelSubscription)
// //         subscription.POST("/:id/renew", handler.RenewSubscription)
// //         subscription.GET("/:id/usage", handler.GetSubscriptionUsage)
// //         subscription.GET("/:id/invoices", handler.GetInvoices)
// //         subscription.GET("/:id/transactions", handler.GetTransactions)
// //         subscription.POST("/:id/payment-intent", handler.CreatePaymentIntent)
// //         subscription.POST("/payment-intent/:id/confirm", handler.ConfirmPaymentIntent)
// //         subscription.POST("/verify", handler.VerifyPayment)
// //     }
// //     rg.POST("/webhook/:gateway", handler.HandleWebhook)
// // }

// // // ============================================
// // // CBT QUESTION BANK ROUTES
// // // ============================================
// // func setupCBTQuestionRoutes(rg *gin.RouterGroup, handler *cbtQuestionHandler.QuestionHandler) {
// //     q := rg.Group("/questions")
// //     q.Use(middleware.AuthMiddleware())
// //     {
// //         q.POST("/create", handler.CreateQuestion)
// //         q.GET("/:id", handler.GetQuestion)
// //         q.PUT("/update/:id", handler.UpdateQuestion)
// //         q.DELETE("/delete/:id", handler.DeleteQuestion)
        
// //         q.GET("/list", handler.ListQuestions)
// //         q.POST("/filter", handler.FilterQuestions)
        
// //         q.POST("/bulk", handler.BulkCreateQuestions)
// //         q.POST("/bulk-upload", handler.BulkUploadFile)
// //         q.POST("/bulk-delete", handler.BulkDelete)
// //         q.PUT("/bulk-status", handler.BulkUpdateStatus)
        
// //         q.POST("/tags/create", handler.CreateTag)
// //         q.GET("/tags/list", handler.ListTags)
        
// //         q.GET("/statistics", handler.GetStatistics)

// //         q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
// //         q.POST("/extract", handler.ExtractQuestionsFromText)
// //         q.GET("/jobs/:id", handler.GetJobStatus)

// //         q.GET("/by-term", handler.GetQuestionsByTerm)
// //         q.GET("/by-session", handler.GetQuestionsBySession)
// //         q.GET("/by-class", handler.GetQuestionsByClass)
// //         q.GET("/for-exam", handler.GetQuestionsForExam)
// //         q.GET("/context/summary", handler.GetQuestionContextSummary)
// //         q.POST("/with-context", handler.GetQuestionsWithContext)
// //         q.GET("/academic/current", handler.GetCurrentAcademicContext)

// //         // NEW GROUPED QUESTION ROUTES
// //         q.POST("/grouped", handler.GetQuestionsGrouped)
// //         q.GET("/by-exam-type", handler.GetQuestionsByExamType)
// //         q.GET("/by-term-grouped", handler.GetQuestionsByTermGrouped)
// //         q.GET("/by-session-grouped", handler.GetQuestionsBySessionGrouped)
// //         q.GET("/all-grouped", handler.GetAllQuestionsGrouped)
// //     }
// // }

// // // ============================================
// // // INITIALIZERS
// // // ============================================

// // func initSubscriptionHandler() *subscriptionHandler.SubscriptionHandler {
// //     subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
// //     paymentService := payment.NewPaymentService()
// //     emailService := email.NewEmailService()
// //     subService := subscriptionService.NewSubscriptionService(subRepo, paymentService, emailService)
// //     return subscriptionHandler.NewSubscriptionHandler(subService)
// // }

// // func initStudentHandler() *academicHandler.StudentHandler {
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     userRepo := academicRepo.NewUserRepository(database.DB)
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     service := academicService.NewStudentService(studentRepo, userRepo, classRepo)
// //     return academicHandler.NewStudentHandler(service)
// // }

// // func initClassHandler() *academicHandler.ClassHandler {
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     classLevelRepo := academicRepo.NewClassLevelRepository(database.DB)
// //     classArmRepo := academicRepo.NewClassArmRepository(database.DB)
// //     sessionRepo := academicRepo.NewSessionRepository(database.DB)
// //     service := academicService.NewClassService(classRepo, classLevelRepo, classArmRepo, sessionRepo)
// //     return academicHandler.NewClassHandler(service)
// // }

// // func initClassLevelHandler() *academicHandler.ClassLevelHandler {
// //     repo := academicRepo.NewClassLevelRepository(database.DB)
// //     service := academicService.NewClassLevelService(repo)
// //     return academicHandler.NewClassLevelHandler(service)
// // }

// // func initClassArmHandler() *academicHandler.ClassArmHandler {
// //     repo := academicRepo.NewClassArmRepository(database.DB)
// //     service := academicService.NewClassArmService(repo)
// //     return academicHandler.NewClassArmHandler(service)
// // }

// // func initTermHandler() *academicHandler.TermHandler {
// //     repo := academicRepo.NewTermRepository(database.DB)
// //     service := academicService.NewTermService(repo)
// //     return academicHandler.NewTermHandler(service)
// // }

// // func initSessionHandler() *academicHandler.SessionHandler {
// //     repo := academicRepo.NewSessionRepository(database.DB)
// //     service := academicService.NewSessionService(repo)
// //     return academicHandler.NewSessionHandler(service)
// // }

// // func initSchoolHandler() *academicHandler.SchoolHandler {
// //     repo := academicRepo.NewSchoolRepository(database.DB)
// //     service := academicService.NewSchoolService(repo)
// //     return academicHandler.NewSchoolHandler(service)
// // }

// // func initAuthHandler() *authHandler.AuthHandler {
// //     repo := authRepo.NewAuthRepository(database.DB)
// //     parentSvc := initParentService()
    
// //     schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     parentRepo := academicRepo.NewParentRepository(database.DB)
// //     subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
    
// //     logger, _ := zap.NewProduction()
    
// //     svc := authService.NewAuthService(
// //         repo, 
// //         parentSvc, 
// //         schoolRepo, 
// //         studentRepo, 
// //         parentRepo, 
// //         subRepo, 
// //         logger,
// //     )
// //     return authHandler.NewAuthHandler(svc)
// // }

// // func initExamHandler() *cbtExamHandler.ExamHandler {
// //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// //     questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// //     examService := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)
// //     return cbtExamHandler.NewExamHandler(examService)
// // }

// // func initQuestionHandler(queue queue.Queue, engine *engine.Engine) *cbtQuestionHandler.QuestionHandler {
// //     repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// //     subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// //     service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB, queue, engine)
// //     return cbtQuestionHandler.NewQuestionHandler(service)
// // }

// // func initAdminHandler() *adminHandler.AdminHandler {
// //     userRepo := academicRepo.NewUserRepository(database.DB)
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     svc := adminService.NewAdminService(userRepo, classRepo, studentRepo, database.DB)
// //     return adminHandler.NewAdminHandler(svc)
// // }

// // func initTeacherHandler() *teacherHandler.TeacherHandler {
// //     userRepo := academicRepo.NewUserRepository(database.DB)
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// //     logger, _ := zap.NewProduction()
// //     svc := teacherService.NewTeacherService(userRepo, studentRepo, classRepo, schoolRepo, database.DB, logger)
// //     return teacherHandler.NewTeacherHandler(svc)
// // }

// // func initParentHandler() *parentHandler.ParentHandler {
// //     parentRepo := academicRepo.NewParentRepository(database.DB)
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// //     userRepo := academicRepo.NewUserRepository(database.DB)
// //     logger, _ := zap.NewProduction()
// //     svc := parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// //     return parentHandler.NewParentHandler(svc)
// // }

// // func initParentService() *parentService.ParentService {
// //     parentRepo := academicRepo.NewParentRepository(database.DB)
// //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// //     classRepo := academicRepo.NewClassRepository(database.DB)
// //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// //     userRepo := academicRepo.NewUserRepository(database.DB)
// //     logger, _ := zap.NewProduction()
// //     return parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// // }

// // // ============================================
// // // PRINT ROUTES - UPDATED WITH ONBOARDING
// // // ============================================
// // func PrintRoutes() {
// //     println("")
// //     println("========================================")
// //     println("📋 AVAILABLE API ENDPOINTS")
// //     println("========================================")
// //     println("")
// //     println("🔓 PUBLIC ROUTES:")
// //     println("   POST   /api/v1/auth/register")
// //     println("   POST   /api/v1/auth/login")
// //     println("   POST   /api/v1/auth/verify-2fa")
// //     println("   POST   /api/v1/auth/refresh")
// //     println("   POST   /api/v1/auth/forgot-password")
// //     println("   POST   /api/v1/auth/reset-password")
// //     println("   POST   /api/v1/auth/send-otp")
// //     println("   POST   /api/v1/auth/verify-email")
// //     println("   GET    /api/v1/auth/status")
// //     println("")
// //     println("🚀 ONBOARDING ROUTES (Public - No Auth):")
// //     println("   GET    /api/v1/onboarding/gateways")
// //     println("   POST   /api/v1/onboarding/start")
// //     println("   POST   /api/v1/onboarding/admin")
// //     println("   POST   /api/v1/onboarding/verify-email")
// //     println("   POST   /api/v1/onboarding/school")
// //     println("   POST   /api/v1/onboarding/subscription")
// //     println("   GET    /api/v1/onboarding/status/:session_id")
// //     println("   GET    /api/v1/onboarding/payment-status/:session_id")
// //     println("   GET    /api/v1/onboarding/session/email/:email")
// //     println("   POST   /api/v1/onboarding/resend-otp")
// //     println("   POST   /api/v1/onboarding/resend-payment-link")
// //     println("   POST   /api/v1/onboarding/webhook/:gateway")
// //     println("")
// //     println("🔗 WEBHOOK (Public):")
// //     println("   POST   /api/v1/webhook/paystack")
// //     println("   POST   /api/v1/webhook/flutterwave")
// //     println("   POST   /api/v1/webhook/stripe")
// //     println("")
// //     println("🔒 PROTECTED ROUTES (Bearer Token Required):")
// //     println("   POST   /api/v1/auth/logout")
// //     println("   POST   /api/v1/auth/logout-all")
// //     println("   GET    /api/v1/auth/sessions")
// //     println("   DELETE /api/v1/auth/sessions/:sessionId")
// //     println("   POST   /api/v1/auth/change-password")
// //     println("   POST   /api/v1/auth/2fa/generate")
// //     println("   POST   /api/v1/auth/2fa/enable")
// //     println("   POST   /api/v1/auth/2fa/disable")
// //     println("   GET    /api/v1/auth/profile")
// //     println("   PUT    /api/v1/auth/profile")
// //     println("   DELETE /api/v1/auth/profile")
// //     println("   GET    /api/v1/auth/subscription/check")
// //     println("")
// //     println("🏫 ACADEMIC ROUTES (Protected):")
// //     println("   POST   /api/v1/schools")
// //     println("   GET    /api/v1/schools")
// //     println("   GET    /api/v1/schools/:id")
// //     println("   PUT    /api/v1/schools/:id")
// //     println("   DELETE /api/v1/schools/:id")
// //     println("   POST   /api/v1/sessions")
// //     println("   GET    /api/v1/school-sessions/:schoolId")
// //     println("   GET    /api/v1/sessions/:id")
// //     println("   POST   /api/v1/terms")
// //     println("   GET    /api/v1/session-terms/:sessionId")
// //     println("   POST   /api/v1/class-levels")
// //     println("   GET    /api/v1/school-class-levels/:schoolId")
// //     println("   POST   /api/v1/class-arms")
// //     println("   GET    /api/v1/school-class-arms/:schoolId")
// //     println("   POST   /api/v1/classes")
// //     println("   GET    /api/v1/school-classes/:schoolId")
// //     println("   POST   /api/v1/students")
// //     println("   GET    /api/v1/school-students/:schoolId")
// //     println("")
// //     println("📚 CBT SUBJECT ROUTES (Protected):")
// //     println("   POST   /api/v1/subjects/create")
// //     println("   GET    /api/v1/subjects/list")
// //     println("   GET    /api/v1/subjects/active")
// //     println("   GET    /api/v1/subjects/view/:id")
// //     println("   PUT    /api/v1/subjects/update/:id")
// //     println("   DELETE /api/v1/subjects/delete/:id")
// //     println("")
// //     println("📝 CBT EXAM ROUTES (Protected):")
// //     println("   📋 EXAM MANAGEMENT (9 Endpoints):")
// //     println("   POST   /api/v1/exams/create")
// //     println("   GET    /api/v1/exams/:id")
// //     println("   GET    /api/v1/exams/list")
// //     println("   PUT    /api/v1/exams/:id")
// //     println("   DELETE /api/v1/exams/:id")
// //     println("   POST   /api/v1/exams/:id/questions")
// //     println("   DELETE /api/v1/exams/:id/questions/:questionId")
// //     println("   POST   /api/v1/exams/:id/assign")
// //     println("   GET    /api/v1/exams/:id/assignments")
// //     println("")
// //     println("   ✍️ EXAM TAKING (3 Endpoints):")
// //     println("   POST   /api/v1/exams/start")
// //     println("   POST   /api/v1/exams/save-answer")
// //     println("   POST   /api/v1/exams/submit")
// //     println("")
// //     println("   📊 RESULTS & PERFORMANCE (4 Endpoints):")
// //     println("   GET    /api/v1/exams/:id/results/class")
// //     println("   GET    /api/v1/exams/:id/results/export")
// //     println("   GET    /api/v1/exams/:id/results/rankings")
// //     println("   GET    /api/v1/exams/:id/statistics")
// //     println("")
// //     println("   🎓 STUDENT PERFORMANCE (2 Endpoints):")
// //     println("   GET    /api/v1/exams/students/:studentId/performance")
// //     println("   GET    /api/v1/exams/students/:studentId/exams/:examId/result")
// //     println("")
// //     println("   🏋️ PRACTICE (1 Endpoint):")
// //     println("   GET    /api/v1/exams/practice/:studentId/:subjectId")
// //     println("")
// //     println("   👨‍🏫 TEACHER VIEWS (3 Endpoints):")
// //     println("   GET    /api/v1/exams/teacher/subjects/:subjectId/results")
// //     println("   GET    /api/v1/exams/teacher/classes/:classId/results")
// //     println("   GET    /api/v1/exams/teacher/performance/dashboard")
// //     println("")
// //     println("   🛡️ ADMIN VIEWS (1 Endpoint):")
// //     println("   GET    /api/v1/exams/school/performance/overview")
// //     println("")
// //     println("👨‍🏫 TEACHER ROUTES (Protected):")
// //     println("   POST   /api/v1/teacher/students")
// //     println("   POST   /api/v1/teacher/students/bulk")
// //     println("   GET    /api/v1/teacher/students")
// //     println("   GET    /api/v1/teacher/students/:id")
// //     println("   PUT    /api/v1/teacher/students/:id")
// //     println("   POST   /api/v1/teacher/students/:id/reset-password")
// //     println("   POST   /api/v1/teacher/students/:id/deactivate")
// //     println("")
// //     println("👪 PARENT ROUTES (Protected):")
// //     println("   GET    /api/v1/parent/children")
// //     println("   GET    /api/v1/parent/child/:studentId/results")
// //     println("")
// //     println("🛡️ ADMIN ROUTES (Protected):")
// //     println("   POST   /api/v1/admin/teachers/assign")
// //     println("   DELETE /api/v1/admin/teachers/unassign/:classId")
// //     println("   GET    /api/v1/admin/users")
// //     println("   GET    /api/v1/admin/students")
// //     println("   GET    /api/v1/admin/classes")
// //     println("   GET    /api/v1/admin/teachers")
// //     println("   DELETE /api/v1/admin/students/:id/permanent")
// //     println("   POST   /api/v1/onboarding/admin/complete/:session_id")
// //     println("   DELETE /api/v1/onboarding/admin/cleanup")
// //     println("   GET    /api/v1/onboarding/admin/stats")
// //     println("")
// //     println("💰 SUBSCRIPTION ROUTES (Protected):")
// //     println("   POST   /api/v1/subscriptions")
// //     println("   GET    /api/v1/subscriptions")
// //     println("   GET    /api/v1/subscriptions/:id")
// //     println("   GET    /api/v1/subscriptions/school/:schoolId/current")
// //     println("   PUT    /api/v1/subscriptions/:id")
// //     println("   POST   /api/v1/subscriptions/:id/cancel")
// //     println("   POST   /api/v1/subscriptions/:id/renew")
// //     println("   GET    /api/v1/subscriptions/:id/usage")
// //     println("   GET    /api/v1/subscriptions/:id/invoices")
// //     println("   GET    /api/v1/subscriptions/:id/transactions")
// //     println("   POST   /api/v1/subscriptions/:id/payment-intent")
// //     println("   POST   /api/v1/subscriptions/payment-intent/:id/confirm")
// //     println("   POST   /api/v1/subscriptions/verify")
// //     println("")
// //     println("📚 CBT QUESTION BANK ROUTES (Protected):")
// //     println("   POST   /api/v1/questions/create")
// //     println("   GET    /api/v1/questions/:id")
// //     println("   PUT    /api/v1/questions/update/:id")
// //     println("   DELETE /api/v1/questions/delete/:id")
// //     println("   GET    /api/v1/questions/list")
// //     println("   POST   /api/v1/questions/filter")
// //     println("   POST   /api/v1/questions/bulk")
// //     println("   POST   /api/v1/questions/bulk-upload")
// //     println("   POST   /api/v1/questions/bulk-delete")
// //     println("   GET    /api/v1/questions/statistics")
// //     println("   POST   /api/v1/questions/tags/create")
// //     println("   GET    /api/v1/questions/tags/list")
// //     println("   POST   /api/v1/questions/ai/generate")
// //     println("   POST   /api/v1/questions/extract")
// //     println("   GET    /api/v1/questions/jobs/:id")
// //     println("")
// //     println("========================================")
// // }



// // // package routes

// // // import (
// // //     academicHandler "cbt-api/internal/academic/handler"
// // //     academicRepo "cbt-api/internal/academic/repository"
// // //     academicService "cbt-api/internal/academic/service"
// // //     authHandler "cbt-api/internal/auth/handler"
// // //     authRepo "cbt-api/internal/auth/repository"
// // //     authService "cbt-api/internal/auth/service"
// // //     "cbt-api/internal/middleware"
// // //     subscriptionHandler "cbt-api/internal/subscription/handler"
// // //     subscriptionRepo "cbt-api/internal/subscription/repository"
// // //     subscriptionService "cbt-api/internal/subscription/service"

// // //     // CBT modules
// // //     cbtExamHandler "cbt-api/internal/cbt/handler"
// // //     cbtExamRepo "cbt-api/internal/cbt/repository"
// // //     cbtExamService "cbt-api/internal/cbt/service"
// // //     cbtQuestionHandler "cbt-api/internal/cbt/handler"
// // //     cbtQuestionRepo "cbt-api/internal/cbt/repository"
// // //     cbtQuestionService "cbt-api/internal/cbt/service"
    
// // //     cbtSubjectHandler "cbt-api/internal/cbt/handler"
// // //     cbtSubjectRepo "cbt-api/internal/cbt/repository"
// // //     cbtSubjectService "cbt-api/internal/cbt/service"

// // //     // NEW actor modules
// // //     adminHandler "cbt-api/internal/admin/handler"
// // //     adminService "cbt-api/internal/admin/service"
// // //     teacherHandler "cbt-api/internal/teacher/handler"
// // //     teacherService "cbt-api/internal/teacher/service"
// // //     parentHandler "cbt-api/internal/parent/handler"
// // //     parentService "cbt-api/internal/parent/service"

// // //     "cbt-api/pkg/email"
// // //     "cbt-api/pkg/payment"
// // //     "cbt-api/pkg/database"
// // //     "cbt-api/internal/ai/engine"
// // //     "cbt-api/internal/ai/queue"

// // //     "github.com/gin-gonic/gin"
// // //     "go.uber.org/zap"

// // //     // Swagger
// // //     swaggerFiles "github.com/swaggo/files"
// // //     ginSwagger "github.com/swaggo/gin-swagger"
// // // )

// // // // SetupRoutes configures all API routes for the application
// // // func SetupRoutes(r *gin.Engine, q queue.Queue, e *engine.Engine) {
// // //     // Initialize all handlers
// // //     authH := initAuthHandler()
// // //     schoolH := initSchoolHandler()
// // //     sessionH := initSessionHandler()
// // //     termH := initTermHandler()
// // //     classLevelH := initClassLevelHandler()
// // //     classArmH := initClassArmHandler()
// // //     classH := initClassHandler()
// // //     studentH := initStudentHandler()
// // //     subscriptionH := initSubscriptionHandler()

// // //     // CBT handlers
// // //     examH := initExamHandler()
// // //     questionH := initQuestionHandler(q, e) 

// // //     // Subject handler initialisation
// // //     subjectRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// // //     subjectService := cbtSubjectService.NewSubjectService(subjectRepo, database.DB)
// // //     subjectHandler := cbtSubjectHandler.NewSubjectHandler(subjectService)

// // //     // NEW actor handlers
// // //     adminH := initAdminHandler()
// // //     teacherH := initTeacherHandler()
// // //     parentH := initParentHandler()
// // //     // Initialize repositories for middleware
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)


// // //     // API version 1 group
// // //     v1 := r.Group("/api/v1")
// // //     {
// // //         // Health check
// // //         v1.GET("/health", authH.HealthCheck)

// // //         // Setup auth routes
// // //         setupAuthRoutes(v1, authH)

// // //         // Setup academic routes (existing)
// // //         setupAcademicRoutes(v1, schoolH, sessionH, termH, classLevelH, classArmH, classH, studentH)

// // //         // Setup subscription routes (existing)
// // //         setupSubscriptionRoutes(v1, subscriptionH)

// // //         // ========== SUBJECT ROUTES ==========
// // //         subject := v1.Group("/subjects")
// // //         subject.Use(middleware.AuthMiddleware())
// // //         {
// // //             subject.POST("/create", subjectHandler.CreateSubject)
// // //             subject.GET("/list", subjectHandler.ListSubjects)
// // //             subject.GET("/active", subjectHandler.ListActiveSubjects)
// // //             subject.GET("/view/:id", subjectHandler.GetSubject)
// // //             subject.PUT("/update/:id", subjectHandler.UpdateSubject)
// // //             subject.DELETE("/delete/:id", subjectHandler.DeleteSubject)
// // //         }

// // //         // ==================== EXAM ROUTES ====================
// // //         // ALL 23 EXAM ENDPOINTS ORGANIZED UNDER /v1/exams
// // //         setupExamRoutes(v1, examH)
// // //         setupStudentExamRoutes(v1, examH, studentRepo)


// // //         // ==================== QUESTION ROUTES ====================
// // //         setupCBTQuestionRoutes(v1, questionH)

// // //         // ==================== NEW ACTOR ROUTES ====================
// // //         setupActorRoutes(v1, adminH, teacherH, parentH)
// // //     }

// // //     // Swagger UI endpoint (no version prefix, accessible directly)
// // //     r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
// // // }


// // // // setupStudentExamRoutes - All student exam routes with automatic context
// // // func setupStudentExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, studentRepo *academicRepo.StudentRepository) {
// // // 	student := rg.Group("/student/exams")
// // // 	student.Use(middleware.AuthMiddleware())
// // // 	student.Use(middleware.StudentContextMiddleware(studentRepo))
// // // 	{
// // // 		// ============================================
// // // 		// DASHBOARD & OVERVIEW - No student ID needed
// // // 		// ============================================
// // // 		student.GET("/dashboard", handler.GetStudentDashboardForUser)
// // // 		student.GET("/performance", handler.GetStudentPerformanceForUser)
		
// // // 		// ============================================
// // // 		// EXAM TAKING - Exam ID only
// // // 		// ============================================
// // // 		student.POST("/start/:examId", handler.StartExamForUser)
// // // 		student.GET("/attempt/:attemptId", handler.GetAttemptState)
// // // 		student.POST("/answer", handler.SaveAnswer)
// // // 		student.POST("/bulk-answer", handler.BulkSaveAnswers)
// // // 		student.POST("/submit", handler.SubmitExam)
// // // 		student.POST("/auto-save", handler.AutoSave)
// // // 		student.POST("/offline-submit", handler.OfflineSubmit)
// // // 		student.POST("/sync", handler.SyncOfflineAnswers)
		
// // // 		// ============================================
// // // 		// RESULTS - Exam ID only
// // // 		// ============================================
// // // 		student.GET("/result/:examId", handler.GetStudentExamResultForUser)
// // // 		student.GET("/review/:attemptId", handler.GetStudentExamReviewForUser)
		
// // // 		// ============================================
// // // 		// TERMLY RESULTS - Query parameters only
// // // 		// ============================================
// // // 		student.GET("/termly-result", handler.GetStudentTermlyResultForUser)
// // // 		student.GET("/report-card", handler.GetTermlyReportCardForUser)
		
// // // 		// ============================================
// // // 		// PRACTICE - Subject ID only
// // // 		// ============================================
// // // 		student.POST("/practice", handler.StartPracticeForUser)
// // // 	}
// // // }


// // // // ============================================
// // // // EXAM ROUTES - ALL 23 ENDPOINTS
// // // // ============================================
// // // func setupExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler) {
// // //     // Base exam group with /v1/exams prefix
// // //     exams := rg.Group("/exams")
// // //     exams.Use(middleware.AuthMiddleware())
// // //     {
// // //         // ============================================
// // //         // GROUP 1: EXAM MANAGEMENT (9 Endpoints)
// // //         // ============================================
        
// // //         // Create a new exam
// // //         exams.POST("/create", handler.CreateExam)
        
// // //         // Get exam by ID
// // //         exams.GET("/:id", handler.GetExam)
        
// // //         // List all exams with pagination
// // //         exams.GET("/list", handler.ListExams)
        
// // //         // Update an exam
// // //         exams.PUT("/:id", handler.UpdateExam)
        
// // //         // Delete an exam
// // //         exams.DELETE("/:id", handler.DeleteExam)
        
// // //         // Add questions to an exam
// // //         exams.POST("/:id/questions", handler.AddQuestionsToExam)
        
// // //         // Remove a question from an exam
// // //         exams.DELETE("/:id/questions/:questionId", handler.RemoveQuestionFromExam)
        
// // //         // Assign exam to students or class
// // //         exams.POST("/:id/assign", handler.AssignExam)
        
// // //         // Get exam assignments
// // //         exams.GET("/:id/assignments", handler.GetExamAssignments)

// // //         // ============================================
// // //         // GROUP 2: EXAM TAKING (3 Endpoints)
// // //         // ============================================
        
// // //         // Start an exam attempt
// // //         exams.POST("/start", handler.StartExam)
        
// // //         // Save a single answer
// // //         exams.POST("/save-answer", handler.SaveAnswer)
// // //         exams.POST("/bulk-answer", handler.BulkSaveAnswers)

        
// // //         // Submit an exam
// // //         exams.POST("/submit", handler.SubmitExam)

// // //         // ============================================
// // //         // GROUP 3: RESULTS & PERFORMANCE (4 Endpoints)
// // //         // ============================================
        
// // //         // Get all students in class with exam scores
// // //         exams.GET("/:id/results/class", handler.GetClassResults)
        
// // //         // Export exam results (CSV/Excel)
// // //         exams.GET("/:id/results/export", handler.ExportResults)
        
// // //         // Get student rankings for an exam
// // //         exams.GET("/:id/results/rankings", handler.GetExamRankings)
        
// // //         // Get exam statistics
// // //         exams.GET("/:id/statistics", handler.GetExamStatistics)

// // //         // ============================================
// // //         // GROUP 4: STUDENT PERFORMANCE (2 Endpoints)
// // //         // ============================================
        
// // //         // Get student's performance summary
// // //         exams.GET("/students/:studentId/performance", handler.GetStudentPerformance)
        
// // //         // Get specific exam result for a student
// // //         exams.GET("/students/:studentId/exams/:examId/result", handler.GetStudentExamResult)

// // //         // ============================================
// // //         // GROUP 5: PRACTICE (1 Endpoint)
// // //         // ============================================
        
// // //         // Start a practice session
// // //         exams.GET("/practice/:studentId/:subjectId", handler.StartPractice)

// // //         // ============================================
// // //         // GROUP 6: TEACHER VIEWS (3 Endpoints)
// // //         // ============================================
        
// // //         // Teacher view - subject results
// // //         exams.GET("/teacher/subjects/:subjectId/results", handler.GetTeacherSubjectResults)
        
// // //         // Teacher view - class performance
// // //         exams.GET("/teacher/classes/:classId/results", handler.GetTeacherClassResults)
        
// // //         // Teacher dashboard
// // //         exams.GET("/teacher/performance/dashboard", handler.GetTeacherDashboard)

// // //         // ============================================
// // //         // GROUP 7: ADMIN VIEWS (1 Endpoint)
// // //         // ============================================
        
// // //         // School-wide performance overview (Admin)
// // //         exams.GET("/school/performance/overview", handler.GetSchoolPerformanceOverview)

// // //         // ============================================
// // //         // NEW PROFESSIONAL CBT FLOW ROUTES - ADD THESE
// // //         // ============================================
// // //         exams.POST("/create-with-context", handler.CreateExamWithContext)
// // //         exams.GET("/subject-questions", handler.GetSubjectQuestionsForExam)
// // //         exams.POST("/:id/bulk-questions", handler.BulkAddQuestionsToExam)
// // //         exams.GET("/:id/preview", handler.PreviewExam)
// // //         exams.POST("/:id/publish", handler.PublishExam)
// // //     }
// // // }

// // // // ============================================
// // // // ACTOR ROUTES (Admin, Teacher, Parent)
// // // // ============================================
// // // func setupActorRoutes(rg *gin.RouterGroup, 
// // //     adminH *adminHandler.AdminHandler,
// // //     teacherH *teacherHandler.TeacherHandler,
// // //     parentH *parentHandler.ParentHandler) {

// // //     // Admin routes (role: admin)
// // //     admin := rg.Group("/admin")
// // //     admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
// // //     {
// // //         admin.POST("/teachers/assign", adminH.AssignTeacher)
// // //         admin.DELETE("/teachers/unassign/:classId", adminH.UnassignTeacher)
// // //         admin.GET("/users", adminH.ListUsers)
// // //         admin.GET("/students", adminH.ListStudents)
// // //         admin.GET("/classes", adminH.ListClasses)
// // //         admin.GET("/teachers", adminH.ListTeachers)
// // //         admin.DELETE("/students/:id/permanent", adminH.HardDeleteStudent)
// // //     }

// // //     // Teacher routes (role: teacher)
// // //     teacher := rg.Group("/teacher")
// // //     teacher.Use(middleware.AuthMiddleware(), middleware.TeacherOnly())
// // //     {
// // //         teacher.POST("/students", teacherH.CreateStudent)
// // //         teacher.POST("/students/bulk", teacherH.BulkCreateStudents)
// // //         teacher.GET("/students", teacherH.GetMyStudents)
// // //         teacher.GET("/students/credentials", teacherH.GetAllStudentsWithCredentials)
// // //         teacher.GET("/students/:id", teacherH.GetStudent)
// // //         teacher.PUT("/students/:id", teacherH.UpdateStudent)
// // //         teacher.POST("/students/:id/reset-password", teacherH.ResetPassword)
// // //         teacher.POST("/students/:id/deactivate", teacherH.DeactivateStudent)
// // //     }

// // //     // Parent routes (role: parent)
// // //     parent := rg.Group("/parent")
// // //     parent.Use(middleware.AuthMiddleware(), middleware.ParentOnly())
// // //     {
// // //         parent.GET("/children", parentH.GetChildren)
// // //         parent.GET("/child/:studentId/results", parentH.GetChildResults)
// // //     }
// // // }

// // // // ============================================
// // // // AUTH ROUTES
// // // // ============================================
// // // func setupAuthRoutes(rg *gin.RouterGroup, handler *authHandler.AuthHandler) {
// // //     auth := rg.Group("/auth")
// // //     {
// // //         auth.POST("/register", handler.Register)
// // //         auth.POST("/login", handler.Login)
// // //         auth.POST("/verify-2fa", handler.Verify2FALogin)
// // //         auth.POST("/refresh", handler.RefreshToken)
// // //         auth.POST("/forgot-password", handler.ForgotPassword)
// // //         auth.POST("/reset-password", handler.ResetPassword)
// // //         auth.POST("/send-otp", handler.SendVerificationOTP)
// // //         auth.POST("/verify-email", handler.VerifyEmail)
// // //     }

// // //     protected := rg.Group("/auth")
// // //     protected.Use(middleware.AuthMiddleware())
// // //     {
// // //         protected.POST("/logout", handler.Logout)
// // //         protected.POST("/logout-all", handler.LogoutAllDevices)
// // //         protected.GET("/sessions", handler.GetSessions)
// // //         protected.DELETE("/sessions/:sessionId", handler.RevokeSession)
// // //         protected.POST("/change-password", handler.ChangePassword)
// // //         protected.POST("/2fa/generate", handler.Generate2FA)
// // //         protected.POST("/2fa/enable", handler.Enable2FA)
// // //         protected.POST("/2fa/disable", handler.Disable2FA)
// // //         protected.GET("/profile", handler.GetProfile)
// // //         protected.PUT("/profile", handler.UpdateProfile)
// // //         protected.DELETE("/profile", handler.DeleteAccount)
// // //         protected.GET("/subscription/check", handler.CheckSubscription) // ✅ ADDED
// // //     }
// // // }

// // // // ============================================
// // // // ACADEMIC ROUTES
// // // // ============================================
// // // func setupAcademicRoutes(rg *gin.RouterGroup,
// // //     schoolHandler *academicHandler.SchoolHandler,
// // //     sessionHandler *academicHandler.SessionHandler,
// // //     termHandler *academicHandler.TermHandler,
// // //     classLevelHandler *academicHandler.ClassLevelHandler,
// // //     classArmHandler *academicHandler.ClassArmHandler,
// // //     classHandler *academicHandler.ClassHandler,
// // //     studentHandler *academicHandler.StudentHandler) {

// // //     academic := rg.Group("/")
// // //     academic.Use(middleware.AuthMiddleware())
// // //     {
// // //         academic.POST("/schools", schoolHandler.CreateSchool)
// // //         academic.GET("/schools", schoolHandler.GetAllSchools)
// // //         academic.GET("/schools/:id", schoolHandler.GetSchool)
// // //         academic.PUT("/schools/:id", schoolHandler.UpdateSchool)
// // //         academic.DELETE("/schools/:id", schoolHandler.DeleteSchool)

// // //         academic.POST("/sessions", sessionHandler.CreateSession)
// // //         academic.GET("/school-sessions/:schoolId", sessionHandler.GetSessions)
// // //         academic.GET("/school-sessions/:schoolId/current", sessionHandler.GetCurrentSession)
// // //         academic.GET("/sessions/:id", sessionHandler.GetSession)
// // //         academic.PUT("/sessions/:id", sessionHandler.UpdateSession)
// // //         academic.DELETE("/sessions/:id", sessionHandler.DeleteSession)

// // //         academic.GET("/sessions", sessionHandler.ListSessions)
// // //         academic.GET("/sessions/stats", sessionHandler.GetSessionStats)
// // //         academic.GET("/sessions/timeline", sessionHandler.GetSessionTimeline)
// // //         academic.GET("/sessions/summary", sessionHandler.GetSessionSummary)
// // //         academic.GET("/sessions/search", sessionHandler.SearchSessions)
// // //         academic.DELETE("/sessions/bulk", sessionHandler.BulkDeleteSessions)

// // //         academic.GET("/terms/list", termHandler.ListAllTerms)
// // //         academic.GET("/terms/school", termHandler.ListAllTermsBySchool)
// // //         academic.GET("/terms/all", termHandler.GetAllTerms)
// // //         academic.GET("/terms", termHandler.ListTerms)
// // //         academic.GET("/terms/stats", termHandler.GetTermStats)
// // //         academic.GET("/terms/search", termHandler.SearchTerms)
// // //         academic.DELETE("/terms/bulk", termHandler.BulkDeleteTerms)
// // //         academic.POST("/terms", termHandler.CreateTerm)
// // //         academic.GET("/session-terms/:sessionId", termHandler.GetTerms)
// // //         academic.GET("/session-terms/:sessionId/current", termHandler.GetCurrentTerm)
// // //         academic.GET("/terms/:id", termHandler.GetTerm)
// // //         academic.PUT("/terms/:id", termHandler.UpdateTerm)
// // //         academic.DELETE("/terms/:id", termHandler.DeleteTerm)

// // //         academic.GET("/class-levels", classLevelHandler.ListClassLevels)
// // //         academic.GET("/class-levels/stats", classLevelHandler.GetClassLevelStats)
// // //         academic.GET("/class-levels/search", classLevelHandler.SearchClassLevels)
// // //         academic.DELETE("/class-levels/bulk", classLevelHandler.BulkDeleteClassLevels)
// // //         academic.POST("/class-levels", classLevelHandler.CreateClassLevel)
// // //         academic.GET("/class-levels/:id", classLevelHandler.GetClassLevel)
// // //         academic.PUT("/class-levels/:id", classLevelHandler.UpdateClassLevel)
// // //         academic.DELETE("/class-levels/:id", classLevelHandler.DeleteClassLevel)
// // //         academic.GET("/school-class-levels/:schoolId", classLevelHandler.GetClassLevelsBySchool)

// // //         academic.GET("/class-arms", classArmHandler.ListClassArms)
// // //         academic.GET("/class-arms/stats", classArmHandler.GetClassArmStats)
// // //         academic.GET("/class-arms/search", classArmHandler.SearchClassArms)
// // //         academic.DELETE("/class-arms/bulk", classArmHandler.BulkDeleteClassArms)
// // //         academic.POST("/class-arms", classArmHandler.CreateClassArm)
// // //         academic.GET("/school-class-arms/:schoolId", classArmHandler.GetClassArms)
// // //         academic.GET("/class-arms/:id", classArmHandler.GetClassArm)
// // //         academic.PUT("/class-arms/:id", classArmHandler.UpdateClassArm)
// // //         academic.DELETE("/class-arms/:id", classArmHandler.DeleteClassArm)

// // //         academic.GET("/classes", classHandler.ListClasses)
// // //         academic.GET("/classes/teacher/:teacherId", classHandler.GetClassesByTeacher)
// // //         academic.GET("/classes/arm/:classArmId", classHandler.GetClassesByArm)
// // //         academic.GET("/classes/level/:classLevelId", classHandler.GetClassesByLevel)
// // //         academic.DELETE("/classes/bulk", classHandler.BulkDeleteClasses)
// // //         academic.GET("/classes/stats", classHandler.GetClassStats)
// // //         academic.GET("/classes/search", classHandler.SearchClasses)

// // //         academic.POST("/classes", classHandler.CreateClass)
// // //         academic.GET("/school-classes/:schoolId", classHandler.GetClassesBySchool)
// // //         academic.GET("/session-classes/:sessionId", classHandler.GetClassesBySession)
// // //         academic.GET("/school-session-classes/:schoolId/:sessionId", classHandler.GetClassesBySchoolAndSession)
// // //         academic.GET("/classes/:id", classHandler.GetClass)
// // //         academic.PUT("/classes/:id", classHandler.UpdateClass)
// // //         academic.DELETE("/classes/:id", classHandler.DeleteClass)

// // //         // NEW: Get current student's profile (no ID needed)
// // //         academic.GET("/student/profile", studentHandler.GetStudentProfile)

// // //         academic.POST("/students", studentHandler.CreateStudent)
// // //         academic.GET("/school-students/:schoolId", studentHandler.GetStudentsBySchool)
// // //         academic.GET("/class-students/:classId", studentHandler.GetStudentsByClass)
// // //         academic.GET("/students/:id", studentHandler.GetStudent)
// // //         academic.GET("/user-student/:userId", studentHandler.GetStudentByUser)
// // //         academic.PUT("/students/:id", studentHandler.UpdateStudent)
// // //         academic.DELETE("/students/:id", studentHandler.DeleteStudent)
// // //         academic.POST("/students/:studentId/transfer", studentHandler.TransferClass)
// // //     }
// // // }

// // // // ============================================
// // // // SUBSCRIPTION ROUTES
// // // // ============================================
// // // func setupSubscriptionRoutes(rg *gin.RouterGroup, handler *subscriptionHandler.SubscriptionHandler) {
// // //     subscription := rg.Group("/subscriptions")
// // //     subscription.Use(middleware.AuthMiddleware())
// // //     {
// // //         subscription.POST("", handler.CreateSubscription)
// // //         subscription.GET("", handler.GetSubscriptions)
// // //         subscription.GET("/:id", handler.GetSubscription)
// // //         subscription.GET("/school/:schoolId/current", handler.GetCurrentSubscription)
// // //         subscription.PUT("/:id", handler.UpdateSubscription)
// // //         subscription.POST("/:id/cancel", handler.CancelSubscription)
// // //         subscription.POST("/:id/renew", handler.RenewSubscription)
// // //         subscription.GET("/:id/usage", handler.GetSubscriptionUsage)
// // //         subscription.GET("/:id/invoices", handler.GetInvoices)
// // //         subscription.GET("/:id/transactions", handler.GetTransactions)
// // //         subscription.POST("/:id/payment-intent", handler.CreatePaymentIntent)
// // //         subscription.POST("/payment-intent/:id/confirm", handler.ConfirmPaymentIntent)
// // //         subscription.POST("/verify", handler.VerifyPayment)
// // //     }
// // //     rg.POST("/webhook/:gateway", handler.HandleWebhook)
// // // }

// // // // ============================================
// // // // CBT QUESTION BANK ROUTES
// // // // ============================================
// // // func setupCBTQuestionRoutes(rg *gin.RouterGroup, handler *cbtQuestionHandler.QuestionHandler) {
// // //     q := rg.Group("/questions")
// // //     q.Use(middleware.AuthMiddleware())
// // //     {
// // //         q.POST("/create", handler.CreateQuestion)
// // //         q.GET("/:id", handler.GetQuestion)
// // //         q.PUT("/update/:id", handler.UpdateQuestion)
// // //         q.DELETE("/delete/:id", handler.DeleteQuestion)
        
// // //         q.GET("/list", handler.ListQuestions)
// // //         q.POST("/filter", handler.FilterQuestions)
        
// // //         q.POST("/bulk", handler.BulkCreateQuestions)
// // //         q.POST("/bulk-upload", handler.BulkUploadFile)
// // //         q.POST("/bulk-delete", handler.BulkDelete)
// // //         q.PUT("/bulk-status", handler.BulkUpdateStatus)
        
// // //         q.POST("/tags/create", handler.CreateTag)
// // //         q.GET("/tags/list", handler.ListTags)
        
// // //         q.GET("/statistics", handler.GetStatistics)

// // //         q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
// // //         q.POST("/extract", handler.ExtractQuestionsFromText)
// // //         q.GET("/jobs/:id", handler.GetJobStatus)

// // //         q.GET("/by-term", handler.GetQuestionsByTerm)
// // //         q.GET("/by-session", handler.GetQuestionsBySession)
// // //         q.GET("/by-class", handler.GetQuestionsByClass)
// // //         q.GET("/for-exam", handler.GetQuestionsForExam)
// // //         q.GET("/context/summary", handler.GetQuestionContextSummary)
// // //         q.POST("/with-context", handler.GetQuestionsWithContext)
// // //         q.GET("/academic/current", handler.GetCurrentAcademicContext)
// // //     }
// // // }

// // // // ============================================
// // // // INITIALIZERS
// // // // ============================================

// // // func initSubscriptionHandler() *subscriptionHandler.SubscriptionHandler {
// // //     subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
// // //     paymentService := payment.NewPaymentService()
// // //     emailService := email.NewEmailService()
// // //     subService := subscriptionService.NewSubscriptionService(subRepo, paymentService, emailService)
// // //     return subscriptionHandler.NewSubscriptionHandler(subService)
// // // }

// // // func initStudentHandler() *academicHandler.StudentHandler {
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     userRepo := academicRepo.NewUserRepository(database.DB)
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     service := academicService.NewStudentService(studentRepo, userRepo, classRepo)
// // //     return academicHandler.NewStudentHandler(service)
// // // }

// // // func initClassHandler() *academicHandler.ClassHandler {
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     classLevelRepo := academicRepo.NewClassLevelRepository(database.DB)
// // //     classArmRepo := academicRepo.NewClassArmRepository(database.DB)
// // //     sessionRepo := academicRepo.NewSessionRepository(database.DB)
// // //     service := academicService.NewClassService(classRepo, classLevelRepo, classArmRepo, sessionRepo)
// // //     return academicHandler.NewClassHandler(service)
// // // }

// // // func initClassLevelHandler() *academicHandler.ClassLevelHandler {
// // //     repo := academicRepo.NewClassLevelRepository(database.DB)
// // //     service := academicService.NewClassLevelService(repo)
// // //     return academicHandler.NewClassLevelHandler(service)
// // // }

// // // func initClassArmHandler() *academicHandler.ClassArmHandler {
// // //     repo := academicRepo.NewClassArmRepository(database.DB)
// // //     service := academicService.NewClassArmService(repo)
// // //     return academicHandler.NewClassArmHandler(service)
// // // }

// // // func initTermHandler() *academicHandler.TermHandler {
// // //     repo := academicRepo.NewTermRepository(database.DB)
// // //     service := academicService.NewTermService(repo)
// // //     return academicHandler.NewTermHandler(service)
// // // }

// // // func initSessionHandler() *academicHandler.SessionHandler {
// // //     repo := academicRepo.NewSessionRepository(database.DB)
// // //     service := academicService.NewSessionService(repo)
// // //     return academicHandler.NewSessionHandler(service)
// // // }

// // // func initSchoolHandler() *academicHandler.SchoolHandler {
// // //     repo := academicRepo.NewSchoolRepository(database.DB)
// // //     service := academicService.NewSchoolService(repo)
// // //     return academicHandler.NewSchoolHandler(service)
// // // }

// // // // ============================================
// // // // FIXED: initAuthHandler - Now passes all required dependencies
// // // // ============================================
// // // func initAuthHandler() *authHandler.AuthHandler {
// // //     repo := authRepo.NewAuthRepository(database.DB)
// // //     parentSvc := initParentService()
    
// // //     // Create the additional repositories needed for AuthService
// // //     schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     parentRepo := academicRepo.NewParentRepository(database.DB)
// // //     subRepo := subscriptionRepo.NewSubscriptionRepository(database.DB)
    
// // //     logger, _ := zap.NewProduction()
    
// // //     svc := authService.NewAuthService(
// // //         repo, 
// // //         parentSvc, 
// // //         schoolRepo, 
// // //         studentRepo, 
// // //         parentRepo, 
// // //         subRepo, 
// // //         logger,
// // //     )
// // //     return authHandler.NewAuthHandler(svc)
// // // }

// // // func initExamHandler() *cbtExamHandler.ExamHandler {
// // //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// // //     questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// // //     examService := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)
// // //     return cbtExamHandler.NewExamHandler(examService)
// // // }

// // // func initQuestionHandler(queue queue.Queue, engine *engine.Engine) *cbtQuestionHandler.QuestionHandler {
// // //     repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
// // //     subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
// // //     service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB, queue, engine)
// // //     return cbtQuestionHandler.NewQuestionHandler(service)
// // // }

// // // // ============================================
// // // // NEW ACTOR INITIALIZERS
// // // // ============================================

// // // func initAdminHandler() *adminHandler.AdminHandler {
// // //     userRepo := academicRepo.NewUserRepository(database.DB)
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     svc := adminService.NewAdminService(userRepo, classRepo, studentRepo, database.DB)
// // //     return adminHandler.NewAdminHandler(svc)
// // // }

// // // func initTeacherHandler() *teacherHandler.TeacherHandler {
// // //     userRepo := academicRepo.NewUserRepository(database.DB)
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     schoolRepo := academicRepo.NewSchoolRepository(database.DB)
// // //     logger, _ := zap.NewProduction()
// // //     svc := teacherService.NewTeacherService(userRepo, studentRepo, classRepo, schoolRepo, database.DB, logger)
// // //     return teacherHandler.NewTeacherHandler(svc)
// // // }

// // // func initParentHandler() *parentHandler.ParentHandler {
// // //     parentRepo := academicRepo.NewParentRepository(database.DB)
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// // //     userRepo := academicRepo.NewUserRepository(database.DB)
// // //     logger, _ := zap.NewProduction()
// // //     svc := parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// // //     return parentHandler.NewParentHandler(svc)
// // // }

// // // func initParentService() *parentService.ParentService {
// // //     parentRepo := academicRepo.NewParentRepository(database.DB)
// // //     studentRepo := academicRepo.NewStudentRepository(database.DB)
// // //     classRepo := academicRepo.NewClassRepository(database.DB)
// // //     examRepo := cbtExamRepo.NewExamRepository(database.DB)
// // //     userRepo := academicRepo.NewUserRepository(database.DB)
// // //     logger, _ := zap.NewProduction()
// // //     return parentService.NewParentService(parentRepo, studentRepo, classRepo, examRepo, userRepo, database.DB, logger)
// // // }

// // // // ============================================
// // // // PRINT ROUTES
// // // // ============================================
// // // func PrintRoutes() {
// // //     println("")
// // //     println("========================================")
// // //     println("📋 AVAILABLE API ENDPOINTS")
// // //     println("========================================")
// // //     println("")
// // //     println("🔓 PUBLIC ROUTES:")
// // //     println("   POST   /api/v1/auth/register")
// // //     println("   POST   /api/v1/auth/login")
// // //     println("   POST   /api/v1/auth/verify-2fa")
// // //     println("   POST   /api/v1/auth/refresh")
// // //     println("   POST   /api/v1/auth/forgot-password")
// // //     println("   POST   /api/v1/auth/reset-password")
// // //     println("   POST   /api/v1/auth/send-otp")
// // //     println("   POST   /api/v1/auth/verify-email")
// // //     println("")
// // //     println("🔒 PROTECTED ROUTES (Bearer Token Required):")
// // //     println("   POST   /api/v1/auth/logout")
// // //     println("   POST   /api/v1/auth/logout-all")
// // //     println("   GET    /api/v1/auth/sessions")
// // //     println("   DELETE /api/v1/auth/sessions/:sessionId")
// // //     println("   POST   /api/v1/auth/change-password")
// // //     println("   POST   /api/v1/auth/2fa/generate")
// // //     println("   POST   /api/v1/auth/2fa/enable")
// // //     println("   POST   /api/v1/auth/2fa/disable")
// // //     println("   GET    /api/v1/auth/profile")
// // //     println("   PUT    /api/v1/auth/profile")
// // //     println("   DELETE /api/v1/auth/profile")
// // //     println("   GET    /api/v1/auth/subscription/check")
// // //     println("")
// // //     println("🏫 ACADEMIC ROUTES (Protected):")
// // //     println("   POST   /api/v1/schools")
// // //     println("   GET    /api/v1/schools")
// // //     println("   GET    /api/v1/schools/:id")
// // //     println("   PUT    /api/v1/schools/:id")
// // //     println("   DELETE /api/v1/schools/:id")
// // //     println("   POST   /api/v1/sessions")
// // //     println("   GET    /api/v1/school-sessions/:schoolId")
// // //     println("   GET    /api/v1/sessions/:id")
// // //     println("   POST   /api/v1/terms")
// // //     println("   GET    /api/v1/session-terms/:sessionId")
// // //     println("   POST   /api/v1/class-levels")
// // //     println("   GET    /api/v1/school-class-levels/:schoolId")
// // //     println("   POST   /api/v1/class-arms")
// // //     println("   GET    /api/v1/school-class-arms/:schoolId")
// // //     println("   POST   /api/v1/classes")
// // //     println("   GET    /api/v1/school-classes/:schoolId")
// // //     println("   POST   /api/v1/students")
// // //     println("   GET    /api/v1/school-students/:schoolId")
// // //     println("")
// // //     println("📚 CBT SUBJECT ROUTES (Protected):")
// // //     println("   POST   /api/v1/subjects/create")
// // //     println("   GET    /api/v1/subjects/list")
// // //     println("   GET    /api/v1/subjects/active")
// // //     println("   GET    /api/v1/subjects/view/:id")
// // //     println("   PUT    /api/v1/subjects/update/:id")
// // //     println("   DELETE /api/v1/subjects/delete/:id")
// // //     println("")
// // //     println("📝 CBT EXAM ROUTES (Protected):")
// // //     println("   📋 EXAM MANAGEMENT (9 Endpoints):")
// // //     println("   POST   /api/v1/exams/create")
// // //     println("   GET    /api/v1/exams/:id")
// // //     println("   GET    /api/v1/exams/list")
// // //     println("   PUT    /api/v1/exams/:id")
// // //     println("   DELETE /api/v1/exams/:id")
// // //     println("   POST   /api/v1/exams/:id/questions")
// // //     println("   DELETE /api/v1/exams/:id/questions/:questionId")
// // //     println("   POST   /api/v1/exams/:id/assign")
// // //     println("   GET    /api/v1/exams/:id/assignments")
// // //     println("")
// // //     println("   ✍️ EXAM TAKING (3 Endpoints):")
// // //     println("   POST   /api/v1/exams/start")
// // //     println("   POST   /api/v1/exams/save-answer")
// // //     println("   POST   /api/v1/exams/submit")
// // //     println("")
// // //     println("   📊 RESULTS & PERFORMANCE (4 Endpoints):")
// // //     println("   GET    /api/v1/exams/:id/results/class")
// // //     println("   GET    /api/v1/exams/:id/results/export")
// // //     println("   GET    /api/v1/exams/:id/results/rankings")
// // //     println("   GET    /api/v1/exams/:id/statistics")
// // //     println("")
// // //     println("   🎓 STUDENT PERFORMANCE (2 Endpoints):")
// // //     println("   GET    /api/v1/exams/students/:studentId/performance")
// // //     println("   GET    /api/v1/exams/students/:studentId/exams/:examId/result")
// // //     println("")
// // //     println("   🏋️ PRACTICE (1 Endpoint):")
// // //     println("   GET    /api/v1/exams/practice/:studentId/:subjectId")
// // //     println("")
// // //     println("   👨‍🏫 TEACHER VIEWS (3 Endpoints):")
// // //     println("   GET    /api/v1/exams/teacher/subjects/:subjectId/results")
// // //     println("   GET    /api/v1/exams/teacher/classes/:classId/results")
// // //     println("   GET    /api/v1/exams/teacher/performance/dashboard")
// // //     println("")
// // //     println("   🛡️ ADMIN VIEWS (1 Endpoint):")
// // //     println("   GET    /api/v1/exams/school/performance/overview")
// // //     println("")
// // //     println("👨‍🏫 TEACHER ROUTES (Protected):")
// // //     println("   POST   /api/v1/teacher/students")
// // //     println("   POST   /api/v1/teacher/students/bulk")
// // //     println("   GET    /api/v1/teacher/students")
// // //     println("   GET    /api/v1/teacher/students/:id")
// // //     println("   PUT    /api/v1/teacher/students/:id")
// // //     println("   POST   /api/v1/teacher/students/:id/reset-password")
// // //     println("   POST   /api/v1/teacher/students/:id/deactivate")
// // //     println("")
// // //     println("👪 PARENT ROUTES (Protected):")
// // //     println("   GET    /api/v1/parent/children")
// // //     println("   GET    /api/v1/parent/child/:studentId/results")
// // //     println("")
// // //     println("🛡️ ADMIN ROUTES (Protected):")
// // //     println("   POST   /api/v1/admin/teachers/assign")
// // //     println("   DELETE /api/v1/admin/teachers/unassign/:classId")
// // //     println("   GET    /api/v1/admin/users")
// // //     println("   GET    /api/v1/admin/students")
// // //     println("   GET    /api/v1/admin/classes")
// // //     println("   GET    /api/v1/admin/teachers")
// // //     println("   DELETE /api/v1/admin/students/:id/permanent")
// // //     println("")
// // //     println("💰 SUBSCRIPTION ROUTES (Protected):")
// // //     println("   POST   /api/v1/subscriptions")
// // //     println("   GET    /api/v1/subscriptions")
// // //     println("   GET    /api/v1/subscriptions/:id")
// // //     println("   GET    /api/v1/subscriptions/school/:schoolId/current")
// // //     println("   PUT    /api/v1/subscriptions/:id")
// // //     println("   POST   /api/v1/subscriptions/:id/cancel")
// // //     println("   POST   /api/v1/subscriptions/:id/renew")
// // //     println("   GET    /api/v1/subscriptions/:id/usage")
// // //     println("   GET    /api/v1/subscriptions/:id/invoices")
// // //     println("   GET    /api/v1/subscriptions/:id/transactions")
// // //     println("   POST   /api/v1/subscriptions/:id/payment-intent")
// // //     println("   POST   /api/v1/subscriptions/payment-intent/:id/confirm")
// // //     println("   POST   /api/v1/subscriptions/verify")
// // //     println("")
// // //     println("📚 CBT QUESTION BANK ROUTES (Protected):")
// // //     println("   POST   /api/v1/questions/create")
// // //     println("   GET    /api/v1/questions/:id")
// // //     println("   PUT    /api/v1/questions/update/:id")
// // //     println("   DELETE /api/v1/questions/delete/:id")
// // //     println("   GET    /api/v1/questions/list")
// // //     println("   POST   /api/v1/questions/filter")
// // //     println("   POST   /api/v1/questions/bulk")
// // //     println("   POST   /api/v1/questions/bulk-upload")
// // //     println("   POST   /api/v1/questions/bulk-delete")
// // //     println("   GET    /api/v1/questions/statistics")
// // //     println("   POST   /api/v1/questions/tags/create")
// // //     println("   GET    /api/v1/questions/tags/list")
// // //     println("   POST   /api/v1/questions/ai/generate")
// // //     println("   POST   /api/v1/questions/extract")
// // //     println("   GET    /api/v1/questions/jobs/:id")
// // //     println("")
// // //     println("🔗 WEBHOOK (Public):")
// // //     println("   POST   /api/v1/webhook/:gateway")
// // //     println("")
// // //     println("========================================")
// // // }
