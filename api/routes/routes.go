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

    "cbt-api/pkg/email"
    "cbt-api/pkg/payment"
    "cbt-api/pkg/database"
    "cbt-api/internal/ai/engine"
    "cbt-api/internal/ai/queue"

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

        // Setup CBT routes (existing)
        setupCBTExamRoutes(v1, examH)
        setupCBTQuestionRoutes(v1, questionH)

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

        // ==================== NEW ACTOR ROUTES ====================

        // Admin routes (role: admin)
        admin := v1.Group("/admin")
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

        // Teacher routes (role: teacher)
        teacher := v1.Group("/teacher")
        teacher.Use(middleware.AuthMiddleware(), middleware.TeacherOnly())
        {
            teacher.POST("/students", teacherH.CreateStudent)
            teacher.POST("/students/bulk", teacherH.BulkCreateStudents)
            teacher.GET("/students", teacherH.GetMyStudents)
            teacher.GET("/students/:id", teacherH.GetStudent)
            teacher.PUT("/students/:id", teacherH.UpdateStudent)
            teacher.POST("/students/:id/reset-password", teacherH.ResetPassword)
            teacher.POST("/students/:id/deactivate", teacherH.DeactivateStudent)
        }

        // Parent routes (role: parent)
        parent := v1.Group("/parent")
        parent.Use(middleware.AuthMiddleware(), middleware.ParentOnly())
        {
            parent.GET("/children", parentH.GetChildren)
            parent.GET("/child/:studentId/results", parentH.GetChildResults)
        }
    }

    // Swagger UI endpoint (no version prefix, accessible directly)
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// ============================================
// AUTH ROUTES (unchanged)
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
    }
}

// ============================================
// ACADEMIC ROUTES (unchanged)
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

        academic.POST("/terms", termHandler.CreateTerm)
        academic.GET("/session-terms/:sessionId", termHandler.GetTerms)
        academic.GET("/session-terms/:sessionId/current", termHandler.GetCurrentTerm)
        academic.GET("/terms/:id", termHandler.GetTerm)
        academic.PUT("/terms/:id", termHandler.UpdateTerm)
        academic.DELETE("/terms/:id", termHandler.DeleteTerm)

        academic.POST("/class-levels", classLevelHandler.CreateClassLevel)
        academic.GET("/school-class-levels/:schoolId", classLevelHandler.GetClassLevels)
        academic.GET("/class-levels/:id", classLevelHandler.GetClassLevel)
        academic.PUT("/class-levels/:id", classLevelHandler.UpdateClassLevel)
        academic.DELETE("/class-levels/:id", classLevelHandler.DeleteClassLevel)

        academic.POST("/class-arms", classArmHandler.CreateClassArm)
        academic.GET("/school-class-arms/:schoolId", classArmHandler.GetClassArms)
        academic.GET("/class-arms/:id", classArmHandler.GetClassArm)
        academic.PUT("/class-arms/:id", classArmHandler.UpdateClassArm)
        academic.DELETE("/class-arms/:id", classArmHandler.DeleteClassArm)

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
// SUBSCRIPTION ROUTES (unchanged)
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
// CBT EXAM ROUTES (unchanged)
// ============================================
func setupCBTExamRoutes(rg *gin.RouterGroup, handler *cbtExamHandler.ExamHandler) {
    exam := rg.Group("/exams")
    exam.Use(middleware.AuthMiddleware())
    {
        exam.POST("/start", handler.StartExam)
        exam.POST("/save-answer", handler.SaveAnswer)
        exam.POST("/submit", handler.SubmitExam)
        exam.GET("/practice/:studentId/:subjectId", handler.StartPractice)
    }
}

// ============================================
// CBT QUESTION BANK ROUTES (fixed)
// ============================================
func setupCBTQuestionRoutes(rg *gin.RouterGroup, handler *cbtQuestionHandler.QuestionHandler) {
    q := rg.Group("/questions")
    q.Use(middleware.AuthMiddleware())
    {
        // Single question CRUD
        q.POST("/create", handler.CreateQuestion)
        q.GET("/:id", handler.GetQuestion)
        q.PUT("/update/:id", handler.UpdateQuestion)
        q.DELETE("/delete/:id", handler.DeleteQuestion)
        
        // Listing & filtering
        q.GET("/list", handler.ListQuestions)
        q.POST("/filter", handler.FilterQuestions)
        
        // Bulk operations
        q.POST("/bulk", handler.BulkCreateQuestions)           // JSON array
        q.POST("/bulk-upload", handler.BulkUploadFile)         // file upload (CSV/JSON/Excel)
        q.POST("/bulk-delete", handler.BulkDelete)
        
        // Tags
        q.POST("/tags/create", handler.CreateTag)
        q.GET("/tags/list", handler.ListTags)
        
        // Statistics
        q.GET("/statistics", handler.GetStatistics)

        q.POST("/ai/generate", handler.GenerateQuestionsWithAI)
        q.POST("/extract", handler.ExtractQuestionsFromText)
        q.GET("/jobs/:id", handler.GetJobStatus)
    }
}

// ============================================
// INITIALIZERS (existing + new)
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

// ✅ CORRECTED: initAuthHandler now passes parentService and logger
func initAuthHandler() *authHandler.AuthHandler {
    repo := authRepo.NewAuthRepository(database.DB)
    parentSvc := initParentService()
    logger, _ := zap.NewProduction()
    svc := authService.NewAuthService(repo, parentSvc, logger)
    return authHandler.NewAuthHandler(svc)
}

func initExamHandler() *cbtExamHandler.ExamHandler {
    examRepo := cbtExamRepo.NewExamRepository(database.DB)
    questionRepo := cbtQuestionRepo.NewQuestionRepository(database.DB)
    examService := cbtExamService.NewExamService(examRepo, questionRepo, database.DB)
    return cbtExamHandler.NewExamHandler(examService)
}

// func initQuestionHandler() *cbtQuestionHandler.QuestionHandler {
//     repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
//     subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
//     service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB)
//     return cbtQuestionHandler.NewQuestionHandler(service)
// }

func initQuestionHandler(queue queue.Queue, engine *engine.Engine) *cbtQuestionHandler.QuestionHandler {
    repo := cbtQuestionRepo.NewQuestionRepository(database.DB)
    subRepo := cbtSubjectRepo.NewSubjectRepository(database.DB)
    service := cbtQuestionService.NewQuestionService(repo, subRepo, database.DB, queue, engine)
    return cbtQuestionHandler.NewQuestionHandler(service)
}
// ============================================
// NEW ACTOR INITIALIZERS
// ============================================

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

// ✅ NEW: initParentService returns the parent service instance (for use in auth)
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
// PRINT ROUTES (optional, unchanged)
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
    println("   DELETE /api/v1/admin/students/:id/permanent")
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
    println("📝 CBT EXAM ROUTES (Protected):")
    println("   POST   /api/v1/exams/start")
    println("   POST   /api/v1/exams/save-answer")
    println("   POST   /api/v1/exams/submit")
    println("   GET    /api/v1/exams/practice/:studentId/:subjectId")
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
    println("")
    println("🔗 WEBHOOK (Public):")
    println("   POST   /api/v1/webhook/:gateway")
    println("")
    println("========================================")
}


