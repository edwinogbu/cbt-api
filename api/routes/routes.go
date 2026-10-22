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

	// SCHOOL NODE <-> CLOUD SYNC
	nodeSyncHandler "cbt-api/internal/nodesync/handler"
	nodeSyncMiddleware "cbt-api/internal/nodesync/middleware"
	nodeSyncRepo "cbt-api/internal/nodesync/repository"
	nodeSyncService "cbt-api/internal/nodesync/service"

	// PLATFORM OWNER: TENANT REGISTRY, BRANDING, AUDIT LOG, LAN SETUP DASHBOARD
	platformHandler "cbt-api/internal/platform/handler"
	platformRepo "cbt-api/internal/platform/repository"
	platformService "cbt-api/internal/platform/service"

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

	// SCHOOL NODE <-> CLOUD SYNC INITIALIZATION
	nodeSyncRepository := nodeSyncRepo.NewNodeSyncRepository(database.DB)
	nodeSyncSvc := nodeSyncService.NewNodeSyncService(nodeSyncRepository)
	nodeSyncH := nodeSyncHandler.NewNodeSyncHandler(nodeSyncSvc)

	// PLATFORM OWNER INITIALIZATION
	platformH := initPlatformHandler()

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
		// Subject is a global, platform-shared curriculum taxonomy (no
		// school_id column at all - confirmed against the live schema),
		// so there's no tenant-isolation dimension to scope these by.
		// But before this fix, mutating endpoints had no role gate
		// beyond AuthMiddleware - any authenticated user, including a
		// student or parent, could create/rename/delete a subject every
		// school's exam and question-bank data references, a platform-
		// wide vandalism vector. Reads stay open to any authenticated
		// user; writes now require the same roles the academic routes
		// already use for curriculum management.
		subject := v1.Group("/subjects")
		subject.Use(middleware.AuthMiddleware())
		{
			subject.GET("/list", subjectHandler.ListSubjects)
			subject.GET("/active", subjectHandler.ListActiveSubjects)
			subject.GET("/view/:id", subjectHandler.GetSubject)
		}
		subjectWrite := v1.Group("/subjects")
		subjectWrite.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
		{
			subjectWrite.POST("/create", subjectHandler.CreateSubject)
			subjectWrite.PUT("/update/:id", subjectHandler.UpdateSubject)
			subjectWrite.DELETE("/delete/:id", subjectHandler.DeleteSubject)
		}

		// ==================== EXAM ROUTES ====================
		// ALL 23 EXAM ENDPOINTS ORGANIZED UNDER /v1/exams
		setupExamRoutes(v1, examH, studentRepo)
		setupStudentExamRoutes(v1, examH, questionH, studentRepo)

		// ==================== QUESTION ROUTES ====================
		setupCBTQuestionRoutes(v1, questionH)

		// ==================== NEW ACTOR ROUTES ====================
		setupActorRoutes(v1, adminH, teacherH, parentH)

		// ==================== OFFLINE-FIRST SYNC ROUTES ====================
		setupSyncRoutes(v1, syncH, studentRepo)

		// ==================== SCHOOL NODE <-> CLOUD SYNC ROUTES ====================
		setupNodeSyncRoutes(v1, nodeSyncH, nodeSyncSvc)

		// ==================== PLATFORM OWNER ROUTES ====================
		setupPlatformRoutes(v1, platformH)
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

		// Admin endpoints (protected) - admin/super_admin, plus
		// platform_owner for cross-tenant oversight (same gate as every
		// other admin surface in this codebase). GetOnboardingStats in
		// particular reports platform-wide onboarding-funnel counts
		// (see OnboardingService.GetOnboardingStats), not any one
		// school's own data, so platform_owner reaching it is the point.
		admin := onboarding.Group("/admin")
		admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
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
func setupStudentExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, questionHandler *cbtQuestionHandler.QuestionHandler, studentRepo *academicRepo.StudentRepository) {
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
		student.GET("/report-card/pdf", handler.DownloadTermlyReportCardForUser)
		student.GET("/report-card/midterm", handler.GetMidTermProgressReportForUser)
		student.GET("/report-card/midterm/pdf", handler.DownloadMidTermProgressReportForUser)
		student.GET("/report-card/promotion", handler.GetPromotionReportCardForUser)
		student.GET("/report-card/promotion/pdf", handler.DownloadPromotionReportCardForUser)
		student.GET("/attendance/summary", handler.GetMyAttendanceSummary)
		student.POST("/practice", handler.StartPracticeForUser)
		// Reuses the exact same handler/service/tenant-scoping as the
		// teacher/admin-only GET /questions/academic/current (both end
		// up calling QuestionService.GetCurrentAcademicContext with
		// middleware.ResolveSchoolID's own school_id resolution, which
		// is already safe for any authenticated role - it forces the
		// caller's own school_id except for platform_owner, regardless
		// of which route group reaches it). Added so a student-facing
		// page (the termly report card) can resolve "my current
		// session/term" without widening the /questions group's own
		// role gate, which was deliberately tightened to exclude
		// students in an earlier fix precisely because that group also
		// bundles write/CRUD operations students must never reach.
		student.GET("/academic/current", questionHandler.GetCurrentAcademicContext)
	}
}

// ============================================
// EXAM ROUTES - ALL 23 ENDPOINTS
// ============================================
func setupExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler, studentRepo *academicRepo.StudentRepository) {
	exams := rg.Group("/exams")
	exams.Use(middleware.AuthMiddleware())
	{
		// These four handlers (StartExam/SaveAnswer/BulkSaveAnswers/SubmitExam)
		// call middleware.GetStudentID(c), which reads the "student_id" key
		// set by StudentContextMiddleware - every other route in this group
		// only has AuthMiddleware, which never sets that key. Before this
		// fix, every request through this exact group (as opposed to the
		// parallel /student/exams/* group, which already had both
		// middlewares - see setupStudentExamRoutes below) got "student
		// context not found" (401) regardless of whose attempt it was,
		// for every student, not just a cross-tenant one. Confirmed via a
		// live request with a real student JWT. The live frontend doesn't
		// call this group's save-answer/bulk-answer/submit paths (it uses
		// /student/exams/answer etc., via student.service.ts), but this
		// group's routes are still real, reachable, public API surface
		// (e.g. for a future client, mobile app, or direct integration).
		exams.POST("/start", middleware.StudentContextMiddleware(studentRepo), handler.StartExam)
		exams.POST("/save-answer", middleware.StudentContextMiddleware(studentRepo), handler.SaveAnswer)
		exams.POST("/bulk-answer", middleware.StudentContextMiddleware(studentRepo), handler.BulkSaveAnswers)
		exams.POST("/submit", middleware.StudentContextMiddleware(studentRepo), handler.SubmitExam)
		// These two ignore their :studentId path param entirely and
		// resolve the real student from middleware.GetUserID(c) instead
		// (self-service only), so they're safe under plain AuthMiddleware.
		exams.GET("/students/:studentId/exams/:examId/result", handler.GetStudentExamResult)
		exams.GET("/practice/:studentId/:subjectId", handler.StartPractice)

		// Everything below is a teacher/admin/super_admin surface, plus
		// platform_owner for cross-tenant oversight (same gate as the
		// question-bank and academic route groups): students/parents have
		// no legitimate call into exam creation, grading-adjacent data,
		// or cross-student/class reporting. Before this fix this entire
		// group (23+ routes) had ONLY AuthMiddleware - any authenticated
		// role, students included, could call CreateExam, DeleteExam,
		// PublishExam, GetClassResults, ExportResults, etc. for any exam
		// in their own school. Each handler below still resolves its own
		// school scope from the verified JWT (authorizeExamTenant/
		// authorizeClassTenant), so this role gate is a second,
		// independent layer, not the only one.
		examsStaff := exams.Group("")
		examsStaff.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
		{
			examsStaff.POST("/create", handler.CreateExam)
			examsStaff.GET("/:id", handler.GetExam)
			examsStaff.GET("/list", handler.ListExams)
			examsStaff.PUT("/:id", handler.UpdateExam)
			examsStaff.DELETE("/:id", handler.DeleteExam)
			examsStaff.POST("/:id/questions", handler.AddQuestionsToExam)
			examsStaff.DELETE("/:id/questions/:questionId", handler.RemoveQuestionFromExam)
			examsStaff.POST("/:id/assign", handler.AssignExam)
			examsStaff.GET("/:id/assignments", handler.GetExamAssignments)
			examsStaff.GET("/:id/results/class", handler.GetClassResults)
			examsStaff.GET("/:id/results/export", handler.ExportResults)
			examsStaff.GET("/:id/results/rankings", handler.GetExamRankings)
			examsStaff.GET("/:id/statistics", handler.GetExamStatistics)
			examsStaff.GET("/students/:studentId/performance", handler.GetStudentPerformance)
			examsStaff.GET("/teacher/subjects/:subjectId/results", handler.GetTeacherSubjectResults)
			examsStaff.GET("/teacher/classes/:classId/results", handler.GetTeacherClassResults)
			examsStaff.GET("/teacher/performance/dashboard", handler.GetTeacherDashboard)
			examsStaff.GET("/school/performance/overview", handler.GetSchoolPerformanceOverview)
			examsStaff.POST("/create-with-context", handler.CreateExamWithContext)
			examsStaff.GET("/subject-questions", handler.GetSubjectQuestionsForExam)
			examsStaff.POST("/:id/bulk-questions", handler.BulkAddQuestionsToExam)
			examsStaff.GET("/:id/preview", handler.PreviewExam)
			examsStaff.POST("/:id/publish", handler.PublishExam)
		}

		// Manual/essay marking - teacher/admin/super_admin, plus
		// platform_owner for cross-tenant support (same role gate as the
		// question-bank group below); the tenant boundary itself is
		// enforced inside ExamService.MarkDescriptiveAnswer/
		// GetPendingManualMarking (authorizeExamTenant / the school_id
		// filter), not just this role check.
		marking := exams.Group("/marking")
		marking.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
		{
			marking.GET("/pending", handler.GetPendingManualMarking)
			marking.POST("/grade", handler.MarkDescriptiveAnswer)
		}

		// Report-card comments: a teacher sets their own remark, but
		// "principal" has no distinct role in this system - an
		// admin/super_admin stands in for the principal's sign-off, so
		// that endpoint is deliberately NOT open to "teacher".
		reportCardRemarks := exams.Group("/report-card")
		{
			teacherRemark := reportCardRemarks.Group("")
			teacherRemark.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
			teacherRemark.POST("/teacher-remark", handler.SetTeacherReportCardRemark)

			principalRemark := reportCardRemarks.Group("")
			principalRemark.Use(middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
			principalRemark.POST("/principal-remark", handler.SetPrincipalReportCardRemark)

			// Lets an authorized school admin preview their own
			// school's report-card template (real name/logo/brand
			// color, illustrative sample student data) before any real
			// result is ever printed.
			templatePreview := reportCardRemarks.Group("")
			templatePreview.Use(middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
			templatePreview.GET("/preview", handler.PreviewReportCardTemplate)

			midTermRemark := reportCardRemarks.Group("")
			midTermRemark.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
			midTermRemark.POST("/midterm-remark", handler.SetMidTermRemark)

			// Recording the FINAL promotion decision is never open to
			// "teacher" - a recommendation (computed, shown on the
			// report itself) is not the same as the school's
			// authorized decision, which only admin-tier roles may
			// record.
			promotionDecision := reportCardRemarks.Group("")
			promotionDecision.Use(middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
			promotionDecision.POST("/promotion-decision", handler.RecordPromotionDecision)

			// Approving/rejecting a report card is the same "principal"
			// sign-off as the promotion decision above - not open to
			// "teacher" for the same reason. Reading approval status/
			// history is informational and open to teacher too, so
			// staff preparing a class's results can see what is already
			// approved without needing admin-tier access.
			reportCardApproval := reportCardRemarks.Group("")
			reportCardApproval.Use(middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
			reportCardApproval.POST("/approve", handler.ApproveReportCard)
			reportCardApproval.POST("/reject", handler.RejectReportCard)
			reportCardApproval.POST("/batch-approve", handler.BatchApproveReportCards)
			// Integrity verification is admin-tier, not teacher-readable
			// like plain approval-status/history below - it's a
			// compliance/security check (did anything bypass the API and
			// edit an audit row directly?), not routine reading.
			reportCardApproval.GET("/approval-history/verify", handler.VerifyReportCardAuditIntegrity)

			reportCardApprovalRead := reportCardRemarks.Group("")
			reportCardApprovalRead.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
			reportCardApprovalRead.GET("/approval-status", handler.GetReportCardApprovalStatus)
			reportCardApprovalRead.GET("/approval-history", handler.GetReportCardApprovalHistory)
		}

		// Class-wide report-card batch review, bulk PDF, and
		// spreadsheet export - same staff role gate as the rest of
		// examsStaff above (teacher/admin/super_admin/platform_owner),
		// with the class-level tenant boundary enforced inside
		// ExamService.authorizeClassTenant, not just this role check.
		reportCardBatch := exams.Group("/classes/:classId/report-cards")
		reportCardBatch.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
		{
			reportCardBatch.GET("", handler.GetClassReportCardSummary)
			reportCardBatch.GET("/batch-pdf", handler.DownloadClassReportCardsZIP)
			reportCardBatch.GET("/export", handler.ExportClassReportCards)
		}

		// Attendance: a teacher marks/reads their own class's register;
		// the tenant boundary is enforced inside
		// ExamService.authorizeClassTenant, same as reportCardBatch above.
		attendance := exams.Group("/attendance")
		attendance.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
		{
			attendance.POST("/mark", handler.MarkClassAttendance)
			attendance.GET("/class/:classId", handler.GetClassAttendanceForDate)
			attendance.GET("/summary", handler.GetStudentAttendanceSummary)
		}

		// Academic Broad Sheet / Student Score Register system.
		// Assessment settings (weighted components, grading boundaries)
		// and teacher-subject assignments are school-policy
		// configuration - admin-tier, same as every other academic
		// configuration group (classes/sessions/terms), plus
		// vp_academics (an academic administrator per the spec, granted
		// the same way as hod/examination_officer - see
		// models.RoleVPAcademics's own doc comment).
		assessmentSettings := exams.Group("/academic")
		assessmentSettings.Use(middleware.RoleMiddleware("admin", "super_admin", "platform_owner", "vp_academics"))
		{
			assessmentSettings.POST("/assessment-schemes", handler.UpsertAssessmentScheme)
			assessmentSettings.GET("/assessment-schemes", handler.ListAssessmentSchemes)
			assessmentSettings.DELETE("/assessment-schemes/:id", handler.DeactivateAssessmentScheme)
			assessmentSettings.POST("/grading-scales", handler.UpsertGradingScale)
			assessmentSettings.GET("/grading-scales", handler.ListGradingScales)
			assessmentSettings.DELETE("/grading-scales/:id", handler.DeactivateGradingScale)
			assessmentSettings.POST("/teacher-subject-assignments", handler.CreateTeacherSubjectAssignment)
			assessmentSettings.GET("/teacher-subject-assignments", handler.ListTeacherSubjectAssignments)
			assessmentSettings.DELETE("/teacher-subject-assignments/:id", handler.DeleteTeacherSubjectAssignment)
		}

		// A teacher's own "my subjects" landing view, and the Student
		// Score Register itself - open to every staff role that might
		// BE an assigned subject teacher or need oversight read access
		// (HOD/Examination Officer/VP Academics); ExamService.
		// SaveScoreRegister/authorizeClassTenant enforce the actual
		// class-tenant boundary, the same pattern reportCardBatch/
		// attendance above already use rather than narrowing this role
		// gate further.
		scoreRegister := exams.Group("")
		scoreRegister.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner", "hod", "examination_officer", "vp_academics"))
		{
			scoreRegister.GET("/teacher/my-subjects", handler.GetMySubjectAssignments)
			scoreRegister.GET("/score-register", handler.GetScoreRegister)
			scoreRegister.POST("/score-register/save", handler.SaveScoreRegister)
			scoreRegister.GET("/score-register/status", handler.GetSubjectResultSubmissionStatus)
			// Submit is the subject teacher's own action - same broad
			// staff-role gate as the group above (ExamService.
			// authorizeClassTenant still enforces the tenant boundary);
			// narrower than that for who may actually be a subject
			// teacher would need per-assignment enforcement this
			// iteration doesn't add (see GetMySubjectAssignments' own
			// doc comment on that same gap).
			scoreRegister.POST("/score-register/submit", handler.SubmitSubjectResults)
		}

		// Review/return is HOD/Examination Officer tier (plus admin, who
		// can always stand in); approve is the academic-administrator
		// tier - admin/super_admin/vp_academics/platform_owner. Two
		// separate groups, not scoreRegister above, since these must
		// never be reachable by a plain "teacher" (a teacher approving
		// their own submission would defeat the whole approval chain).
		subjectResultReview := exams.Group("/score-register")
		subjectResultReview.Use(middleware.RoleMiddleware("hod", "examination_officer", "admin", "super_admin", "platform_owner"))
		{
			subjectResultReview.POST("/review", handler.ReviewSubjectResults)
			subjectResultReview.POST("/return", handler.ReturnSubjectResults)
		}

		subjectResultApproval := exams.Group("/score-register")
		subjectResultApproval.Use(middleware.RoleMiddleware("admin", "super_admin", "vp_academics", "platform_owner"))
		subjectResultApproval.POST("/approve", handler.ApproveSubjectResults)

		// Broad sheets: the Class Teacher view is open to the same
		// staff roles as the score register itself (a class's own
		// teacher, or any oversight role); the Subject (HOD/Examination
		// Officer) and School-wide (VP Academics) views are oversight-
		// only, same gate as the review/approval groups above - a plain
		// "teacher" has no business pulling another teacher's subject
		// standing across classes, or the whole school's monitor.
		classBroadSheet := exams.Group("/broadsheet")
		classBroadSheet.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner", "hod", "examination_officer", "vp_academics"))
		classBroadSheet.GET("/class", handler.GetClassBroadSheet)

		oversightBroadSheet := exams.Group("/broadsheet")
		oversightBroadSheet.Use(middleware.RoleMiddleware("hod", "examination_officer", "admin", "super_admin", "vp_academics", "platform_owner"))
		{
			oversightBroadSheet.GET("/subject", handler.GetSubjectBroadSheet)
			oversightBroadSheet.GET("/school", handler.GetSchoolBroadSheet)
		}
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
	// AdminOnly() is an exact-match RoleMiddleware("admin") - it never
	// included super_admin or platform_owner, even though every method
	// AdminHandler calls into (AdminService.AssignTeacher/ListUsersByRole/
	// ListAllStudents/ListAllClasses/ListAllTeachers/HardDeleteStudent/
	// UnassignTeacher) is already role-agnostic once past this gate - each
	// takes schoolID+isPlatformOwner and scopes/bypasses on those alone,
	// matching the RoleMiddleware("admin", "super_admin", "platform_owner")
	// gate every other admin-ish route group in this codebase already
	// uses (academic/questions/subscriptions/exams). Confirmed live: a
	// real super_admin's own dashboard got 403 "Insufficient permissions"
	// on every one of these calls (ListUsers/ListStudents/ListClasses/
	// ListTeachers), rendering it empty - not a deliberate restriction,
	// just this one group never updated when super_admin was introduced.
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
	{
		admin.POST("/teachers/assign", adminH.AssignTeacher)
		admin.DELETE("/teachers/unassign/:classId", adminH.UnassignTeacher)
		admin.GET("/users", adminH.ListUsers)
		admin.GET("/students", adminH.ListStudents)
		admin.GET("/classes", adminH.ListClasses)
		admin.GET("/teachers", adminH.ListTeachers)
		admin.DELETE("/students/:id/permanent", adminH.HardDeleteStudent)
		// Read-only, filesystem-sourced backup status (Master
		// Implementation Plan section 7) - reports what
		// scripts/backup-school-node.sh has actually written to
		// BACKUP_DIR, never a claimed/assumed status. Same role gate
		// as every other /admin/* route; a backup is a per-node/
		// per-school concern, not platform-owner-exclusive.
		admin.GET("/backups", adminHandler.ListBackups)
		// Live, real-time node health (Master Implementation Plan
		// section 1 "guided setup... health monitoring") - DB/Redis
		// connectivity + latency, process uptime, node identity. Same
		// role gate as every other /admin/* route.
		admin.GET("/system/health", adminHandler.GetSystemHealth)
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

		// Multi-role support: switching re-issues tokens with a
		// different active role (never requires logging out) - see
		// internal/auth/service/auth_service.go's SwitchRole.
		protected.GET("/available-roles", handler.GetAvailableRoles)
		protected.POST("/switch-role", handler.SwitchRole)
	}

	// Role grant/revoke management - who gets which extra role is an
	// admin decision, enforced server-side, plus platform_owner for
	// cross-tenant support (same gate as academic/subscription/questions).
	// AuthService.GrantRole/RevokeRole independently verify the target
	// user belongs to the caller's own school (platform_owner exempt),
	// so this role gate alone was never the only thing standing between
	// a school admin and another school's user roster.
	roleAdmin := rg.Group("/auth/users")
	roleAdmin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
	{
		roleAdmin.POST("/:id/roles", handler.GrantRole)
		roleAdmin.DELETE("/:id/roles/:role", handler.RevokeRole)
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

	// A student's own profile lookup (StudentHandler.GetStudentProfile
	// reads the caller's own user_id from the verified JWT, never a
	// client-supplied one - see its own doc comment) is the one
	// legitimate non-staff read in this whole route group, so it gets
	// its own unrestricted-by-role group, authenticated but otherwise
	// open to every role.
	academicSelfProfile := rg.Group("/")
	academicSelfProfile.Use(middleware.AuthMiddleware())
	{
		academicSelfProfile.GET("/student/profile", studentHandler.GetStudentProfile)
	}

	// Every other academic-module route is school/class/term/student
	// management - teachers and admins (school-scoped, enforced in the
	// handler/service layer) and platform_owner (cross-tenant). Students
	// and parents have no legitimate call into it: before this gate, any
	// authenticated role - students included - could call these CRUD
	// endpoints, matching the gate already applied to /questions.
	academic := rg.Group("/")
	academic.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "teacher", "platform_owner"))
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
	// Billing/subscription management is an admin-only surface (plus
	// platform_owner, cross-tenant) - before this gate, any authenticated
	// role, students and parents included, could call these endpoints.
	subscription.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
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
	// Question bank management is a teacher/admin surface, plus
	// platform_owner for cross-tenant oversight/support (same gate as
	// the academic and subscription route groups): students and parents
	// have no legitimate call into it (confirmed - the frontend's
	// question.service.ts is only wired into /dashboard/admin and teacher
	// pages), and the FKs trusted inside these handlers (school/class/subject)
	// are only safe once the caller's role is restricted here. Every
	// handler in this group resolves its own school scope from the
	// verified JWT (see middleware.GetSchoolID/IsPlatformOwner) rather
	// than trusting a client-supplied value, so platform_owner's
	// cross-tenant reach is bounded by that, not by this role gate alone.
	q.Use(middleware.RoleMiddleware("teacher", "admin", "super_admin", "platform_owner"))
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

		q.GET("/templates/:format", handler.DownloadImportTemplate)

		q.GET("/statistics", handler.GetStatistics)

		q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
		q.POST("/extract", handler.ExtractQuestionsFromText)
		q.GET("/jobs/:id", handler.GetJobStatus)

		// Synchronous, review-first AI endpoints: these never write to the
		// question bank. The teacher reviews the returned drafts and saves
		// approved ones via POST /questions/bulk (above).
		q.POST("/ai/extract-lesson", handler.ExtractLessonStructure)
		q.POST("/ai/generate-grounded", handler.GenerateGroundedQuestions)

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

// ============================================
// SCHOOL NODE <-> CLOUD SYNC ROUTES
// ============================================
// Go-to-Go sync between a School CBT Node (see Dockerfile/
// docker-compose.school-node.yml) and this cloud instance, using the same
// idempotent-upsert conventions as the device sync routes above.
// /nodesync/* authenticates via a node's own API key (nodeSyncMiddleware.
// NodeAuthMiddleware), never the human-user JWT - a node is a long-lived
// server-to-server caller. Credential issuance is a human-admin action,
// so it sits under the existing /admin group instead.
func setupNodeSyncRoutes(rg *gin.RouterGroup, handler *nodeSyncHandler.NodeSyncHandler, svc *nodeSyncService.NodeSyncService) {
	nodesync := rg.Group("/nodesync")
	nodesync.Use(nodeSyncMiddleware.NodeAuthMiddleware(svc))
	{
		nodesync.POST("/push", handler.Push)
		nodesync.GET("/pull", handler.Pull)
	}

	// LAN Setup Dashboard: issuing a School CBT Node credential - admin/
	// super_admin for their own school, plus platform_owner for
	// cross-tenant support (same gate as every other admin surface in
	// this codebase). The tenant boundary itself is enforced inside
	// NodeSyncHandler.CreateCredential (RequireOwnSchool against the
	// named school_id), not just this role check - a credential names
	// its own target school in the request body, so the role check alone
	// was never enough.
	adminNodes := rg.Group("/admin/school-nodes")
	adminNodes.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware("admin", "super_admin", "platform_owner"))
	{
		adminNodes.POST("/credentials", handler.CreateCredential)
	}
}

// ============================================
// PLATFORM OWNER ROUTES (Master Implementation Plan Phases 3-4, 10, 11)
// ============================================
// Tenant registry, platform-controlled branding, cross-tenant audit log,
// and the LAN Setup Dashboard. Every route here is gated to EXACTLY
// "platform_owner" via middleware.PlatformOwnerOnly() - never "admin" or
// "super_admin", which are school-scoped roles. Unlike every other admin-ish
// group in this file (which adds platform_owner alongside admin/super_admin
// for cross-tenant oversight), this group grants NO access at all to
// admin/super_admin: a school administrator must never read or write
// another school's tenant record, another school's LAN configuration, or
// the platform-wide audit log, regardless of their own school's role.
func setupPlatformRoutes(rg *gin.RouterGroup, handler *platformHandler.PlatformHandler) {
	platform := rg.Group("/platform")
	platform.Use(middleware.AuthMiddleware(), middleware.PlatformOwnerOnly())
	{
		platform.GET("/tenants", handler.ListTenants)
		platform.GET("/tenants/:id", handler.GetTenant)
		platform.PUT("/tenants/:id/branding", handler.UpdateBranding)
		platform.PUT("/tenants/:id/provision", handler.ProvisionTenant)
		platform.PUT("/tenants/:id/suspend", handler.SuspendTenant)

		platform.GET("/audit-logs", handler.ListAuditLogs)

		platform.GET("/lan-config/:schoolId", handler.GetLANConfig)
		platform.PUT("/lan-config/:schoolId", handler.UpdateLANConfig)
		platform.PUT("/lan-config/:schoolId/active", handler.SetLANConfigActive)
		platform.POST("/lan-config/:schoolId/test", handler.TestLANConnectivity)
		platform.GET("/lan-config/:schoolId/history", handler.ListLANConfigHistory)
		platform.POST("/lan-config/:schoolId/rollback", handler.RollbackLANConfig)
	}
}

func initPlatformHandler() *platformHandler.PlatformHandler {
	repo := platformRepo.NewPlatformRepository(database.DB)
	svc := platformService.NewPlatformService(repo)
	return platformHandler.NewPlatformHandler(svc)
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
	sessionRepo := academicRepo.NewSessionRepository(database.DB)
	service := academicService.NewTermService(repo, sessionRepo)
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
