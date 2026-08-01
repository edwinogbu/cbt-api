package handler

import (
	"context"
	"net/http"

	"cbt-api/internal/models"
	"cbt-api/internal/onboarding/dto"
	"cbt-api/internal/onboarding/service"

	"github.com/gin-gonic/gin"
)

type OnboardingHandler struct {
	service *service.OnboardingService
}

func NewOnboardingHandler(service *service.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{service: service}
}

// ============================================
// RESPONSE HELPERS
// ============================================

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func successWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func created(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error:   message,
	})
}

func notFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   message,
	})
}

func conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Success: false,
		Error:   message,
	})
}

func internalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error:   message,
	})
}

func unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error:   message,
	})
}

// ============================================
// GET AVAILABLE GATEWAYS
// ============================================

// GetAvailableGateways returns all available payment gateways
// @Summary Get available payment gateways
// @Description Returns a list of all available payment gateways with their details
// @Tags Onboarding
// @Produce json
// @Success 200 {object} Response{data=[]dto.GatewayInfo}
// @Router /api/v1/onboarding/gateways [get]
func (h *OnboardingHandler) GetAvailableGateways(c *gin.Context) {
	gateways := h.service.GetAvailableGateways(c.Request.Context())
	successWithMessage(c, gateways, "Available payment gateways retrieved successfully")
}

// ============================================
// STEP 1: SELECT PLAN
// ============================================

// StartOnboarding handles plan selection
// @Summary Start onboarding - Select plan
// @Description User selects their subscription plan, interval, and payment gateway
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body dto.StartOnboardingRequest true "Plan selection details"
// @Success 200 {object} Response{data=dto.StartOnboardingResponse}
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/start [post]
func (h *OnboardingHandler) StartOnboarding(c *gin.Context) {
	var req dto.StartOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Add IP and User-Agent to context for logging
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, "ip_address", c.ClientIP())
	ctx = context.WithValue(ctx, "user_agent", c.GetHeader("User-Agent"))

	resp, err := h.service.StartOnboarding(ctx, &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, resp, "Plan selected successfully. Please proceed to create your admin account.")
}

// ============================================
// STEP 2: CREATE ADMIN ACCOUNT
// ============================================

// CreateAdminAccount handles admin account creation
// @Summary Create admin account
// @Description Creates an administrator account for the onboarding session
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body dto.CreateAdminRequest true "Admin account details"
// @Success 201 {object} Response{data=dto.CreateAdminResponse}
// @Failure 400 {object} Response
// @Failure 409 {object} Response
// @Router /api/v1/onboarding/admin [post]
func (h *OnboardingHandler) CreateAdminAccount(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	resp, err := h.service.CreateAdminAccount(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "email already registered" || err.Error() == "username already taken" {
			conflict(c, err.Error())
			return
		}
		badRequest(c, err.Error())
		return
	}

	created(c, resp, "Admin account created successfully. Please verify your email.")
}

// ============================================
// STEP 3: VERIFY EMAIL
// ============================================

// VerifyEmail handles email verification with OTP
// @Summary Verify email with OTP
// @Description Verifies the admin's email address using the OTP code sent via email
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body dto.VerifyEmailRequest true "Email verification details"
// @Success 200 {object} Response{data=dto.VerifyEmailResponse}
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/verify-email [post]
func (h *OnboardingHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	resp, err := h.service.VerifyEmail(c.Request.Context(), &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, resp, "Email verified successfully. Please proceed to register your school.")
}

// ============================================
// STEP 4: REGISTER SCHOOL
// ============================================

// RegisterSchool handles school registration
// @Summary Register school
// @Description Registers a school for the onboarding session
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body dto.RegisterSchoolRequest true "School registration details"
// @Success 200 {object} Response{data=dto.RegisterSchoolResponse}
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/school [post]
func (h *OnboardingHandler) RegisterSchool(c *gin.Context) {
	var req dto.RegisterSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	resp, err := h.service.RegisterSchool(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "school name already taken" {
			conflict(c, err.Error())
			return
		}
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, resp, "School registered successfully. Please review your information.")
}

// ============================================
// STEP 6: CREATE SUBSCRIPTION
// ============================================

// CreateSubscription handles subscription creation with payment link
// @Summary Create subscription with payment link
// @Description Creates a subscription and generates a payment link for the selected plan
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body dto.CreateSubscriptionRequest true "Subscription creation details"
// @Success 200 {object} Response{data=dto.CreateSubscriptionResponse}
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/subscription [post]
func (h *OnboardingHandler) CreateSubscription(c *gin.Context) {
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	resp, err := h.service.CreateSubscription(c.Request.Context(), &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, resp, "Subscription created successfully. Please complete the payment to activate your account.")
}

// ============================================
// GET ONBOARDING STATUS
// ============================================

// GetStatus returns the current status of an onboarding session
// @Summary Get onboarding status
// @Description Retrieves the current status and progress of an onboarding session
// @Tags Onboarding
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} Response{data=dto.OnboardingStatusResponse}
// @Failure 404 {object} Response
// @Router /api/v1/onboarding/status/{session_id} [get]
func (h *OnboardingHandler) GetStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		badRequest(c, "session_id is required")
		return
	}

	resp, err := h.service.GetOnboardingStatus(c.Request.Context(), sessionID)
	if err != nil {
		notFound(c, err.Error())
		return
	}

	successWithMessage(c, resp, "Onboarding status retrieved successfully")
}

// ============================================
// CHECK PAYMENT STATUS
// ============================================

// CheckPaymentStatus checks if payment has been completed
// @Summary Check payment status
// @Description Checks if the payment for the onboarding session has been completed
// @Tags Onboarding
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} Response{data=dto.PaymentStatusResponse}
// @Failure 404 {object} Response
// @Router /api/v1/onboarding/payment-status/{session_id} [get]
func (h *OnboardingHandler) CheckPaymentStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		badRequest(c, "session_id is required")
		return
	}

	resp, err := h.service.CheckPaymentStatus(c.Request.Context(), sessionID)
	if err != nil {
		notFound(c, err.Error())
		return
	}

	message := "Payment status retrieved successfully"
	if resp.IsComplete {
		message = "Payment completed successfully! Your account is being activated."
	} else if resp.Status == "pending" {
		message = "Payment is still pending. Please complete the payment."
	}

	successWithMessage(c, resp, message)
}

// ============================================
// GET SESSION BY EMAIL
// ============================================

// GetSessionByEmail retrieves an onboarding session by email
// @Summary Get session by email
// @Description Retrieves an onboarding session by the user's email address
// @Tags Onboarding
// @Produce json
// @Param email path string true "User Email"
// @Success 200 {object} Response{data=dto.GetSessionByEmailResponse}
// @Failure 404 {object} Response
// @Router /api/v1/onboarding/session/email/{email} [get]
func (h *OnboardingHandler) GetSessionByEmail(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		badRequest(c, "email is required")
		return
	}

	resp, err := h.service.GetSessionByEmail(c.Request.Context(), email)
	if err != nil {
		notFound(c, err.Error())
		return
	}

	successWithMessage(c, resp, "Session retrieved successfully")
}

// ============================================
// RESEND OTP
// ============================================

// ResendVerificationOTP resends the email verification OTP
// @Summary Resend verification OTP
// @Description Resends the email verification OTP to the user's email address
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body object true "Session ID" Example({"session_id":"uuid"})
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/resend-otp [post]
func (h *OnboardingHandler) ResendVerificationOTP(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	err := h.service.ResendVerificationOTP(c.Request.Context(), req.SessionID)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, gin.H{
		"message": "Verification OTP has been resent successfully. Please check your email.",
	}, "OTP resent successfully")
}

// ============================================
// RESEND PAYMENT LINK
// ============================================

// ResendPaymentLink resends the payment link email
// @Summary Resend payment link
// @Description Resends the payment link email to the user's email address
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param request body object true "Session ID" Example({"session_id":"uuid"})
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/resend-payment-link [post]
func (h *OnboardingHandler) ResendPaymentLink(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	err := h.service.ResendPaymentLink(c.Request.Context(), req.SessionID)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, gin.H{
		"message": "Payment link has been resent successfully. Please check your email.",
	}, "Payment link resent successfully")
}

// ============================================
// COMPLETE ONBOARDING (Admin Only)
// ============================================

// CompleteOnboarding manually completes the onboarding process
// @Summary Complete onboarding (Admin)
// @Description Manually completes the onboarding process for a session
// @Tags Onboarding
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID"
// @Success 200 {object} Response{data=dto.ActivationResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /api/v1/onboarding/admin/complete/{session_id} [post]
func (h *OnboardingHandler) CompleteOnboarding(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		badRequest(c, "session_id is required")
		return
	}

	resp, err := h.service.CompleteOnboarding(c.Request.Context(), sessionID)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	if resp.Success {
		successWithMessage(c, resp, resp.Message)
	} else {
		badRequest(c, resp.Message)
	}
}

// ============================================
// CLEANUP EXPIRED SESSIONS (Admin Only)
// ============================================

// CleanupExpiredSessions removes expired sessions
// @Summary Cleanup expired sessions (Admin)
// @Description Removes all expired onboarding sessions from the system
// @Tags Onboarding
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/onboarding/admin/cleanup [delete]
func (h *OnboardingHandler) CleanupExpiredSessions(c *gin.Context) {
	err := h.service.CleanupExpiredSessions(c.Request.Context())
	if err != nil {
		internalServerError(c, err.Error())
		return
	}

	successWithMessage(c, gin.H{
		"message": "Expired sessions cleaned up successfully.",
	}, "Cleanup completed successfully")
}

// ============================================
// GET ONBOARDING STATISTICS (Admin Only)
// ============================================

// GetOnboardingStats returns statistics about onboarding sessions
// @Summary Get onboarding statistics (Admin)
// @Description Retrieves statistics about all onboarding sessions
// @Tags Onboarding
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/onboarding/admin/stats [get]
func (h *OnboardingHandler) GetOnboardingStats(c *gin.Context) {
	stats, err := h.service.GetOnboardingStats(c.Request.Context())
	if err != nil {
		internalServerError(c, err.Error())
		return
	}

	successWithMessage(c, stats, "Statistics retrieved successfully")
}

// ============================================
// WEBHOOK ENDPOINT (Public)
// ============================================

// Webhook handles payment webhooks from various gateways
// @Summary Payment webhook
// @Description Handles payment webhook callbacks from payment gateways
// @Tags Onboarding
// @Accept json
// @Produce json
// @Param gateway path string true "Gateway name (paystack, flutterwave, stripe)"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/onboarding/webhook/{gateway} [post]
func (h *OnboardingHandler) Webhook(c *gin.Context) {
	gateway := c.Param("gateway")
	if gateway == "" {
		badRequest(c, "gateway is required")
		return
	}

	// Get signature from header based on gateway
	var signature string
	switch gateway {
	case "paystack":
		signature = c.GetHeader("x-paystack-signature")
	case "flutterwave":
		signature = c.GetHeader("verif-hash")
	case "stripe":
		signature = c.GetHeader("stripe-signature")
	default:
		badRequest(c, "unsupported gateway")
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		badRequest(c, "Failed to read request body")
		return
	}

	// Parse gateway
	var gatewayEnum models.PaymentGateway
	switch gateway {
	case "paystack":
		gatewayEnum = models.GatewayPaystack
	case "flutterwave":
		gatewayEnum = models.GatewayFlutterwave
	case "stripe":
		gatewayEnum = models.GatewayStripe
	default:
		badRequest(c, "unsupported gateway")
		return
	}

	// Process webhook
	err = h.service.ProcessWebhook(c.Request.Context(), body, signature, gatewayEnum)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	successWithMessage(c, gin.H{
		"status": "processed",
	}, "Webhook processed successfully")
}

// ============================================
// HEALTH CHECK FOR ONBOARDING
// ============================================

// HealthCheck checks if the onboarding service is healthy
// @Summary Onboarding health check
// @Description Checks if the onboarding service is healthy and responsive
// @Tags Onboarding
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/onboarding/health [get]
func (h *OnboardingHandler) HealthCheck(c *gin.Context) {
	successWithMessage(c, gin.H{
		"status":  "healthy",
		"service": "onboarding",
	}, "Onboarding service is healthy")
}


// package handler

// import (
// 	"context"
// 	"net/http"

// 	"cbt-api/internal/models"
// 	"cbt-api/internal/onboarding/dto"
// 	"cbt-api/internal/onboarding/service"

// 	"github.com/gin-gonic/gin"
// )

// type OnboardingHandler struct {
// 	service *service.OnboardingService
// }

// func NewOnboardingHandler(service *service.OnboardingService) *OnboardingHandler {
// 	return &OnboardingHandler{service: service}
// }

// // ============================================
// // RESPONSE HELPERS
// // ============================================

// type Response struct {
// 	Success bool        `json:"success"`
// 	Message string      `json:"message,omitempty"`
// 	Data    interface{} `json:"data,omitempty"`
// 	Error   string      `json:"error,omitempty"`
// }

// func success(c *gin.Context, data interface{}) {
// 	c.JSON(http.StatusOK, Response{
// 		Success: true,
// 		Data:    data,
// 	})
// }

// func successWithMessage(c *gin.Context, data interface{}, message string) {
// 	c.JSON(http.StatusOK, Response{
// 		Success: true,
// 		Message: message,
// 		Data:    data,
// 	})
// }

// func badRequest(c *gin.Context, message string) {
// 	c.JSON(http.StatusBadRequest, Response{
// 		Success: false,
// 		Error:   message,
// 	})
// }

// func notFound(c *gin.Context, message string) {
// 	c.JSON(http.StatusNotFound, Response{
// 		Success: false,
// 		Error:   message,
// 	})
// }

// func internalServerError(c *gin.Context, message string) {
// 	c.JSON(http.StatusInternalServerError, Response{
// 		Success: false,
// 		Error:   message,
// 	})
// }

// // ============================================
// // GET AVAILABLE GATEWAYS
// // ============================================

// // GetAvailableGateways returns all available payment gateways
// // GET /api/v1/onboarding/gateways
// func (h *OnboardingHandler) GetAvailableGateways(c *gin.Context) {
// 	gateways := h.service.GetAvailableGateways(c.Request.Context())
// 	successWithMessage(c, gateways, "Available payment gateways retrieved successfully")
// }

// // ============================================
// // STEP 1: SELECT PLAN
// // ============================================

// // StartOnboarding handles plan selection
// // POST /api/v1/onboarding/start
// func (h *OnboardingHandler) StartOnboarding(c *gin.Context) {
// 	var req dto.StartOnboardingRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	// Add IP and User-Agent to context for logging
// 	ctx := c.Request.Context()
// 	ctx = context.WithValue(ctx, "ip_address", c.ClientIP())
// 	ctx = context.WithValue(ctx, "user_agent", c.GetHeader("User-Agent"))

// 	resp, err := h.service.StartOnboarding(ctx, &req)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Plan selected successfully. Please proceed to create your admin account.")
// }

// // ============================================
// // STEP 2: CREATE ADMIN ACCOUNT
// // ============================================

// // CreateAdminAccount handles admin account creation
// // POST /api/v1/onboarding/admin
// func (h *OnboardingHandler) CreateAdminAccount(c *gin.Context) {
// 	var req dto.CreateAdminRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	resp, err := h.service.CreateAdminAccount(c.Request.Context(), &req)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Admin account created successfully. Please verify your email.")
// }

// // ============================================
// // STEP 3: VERIFY EMAIL
// // ============================================

// // VerifyEmail handles email verification
// // POST /api/v1/onboarding/verify-email
// func (h *OnboardingHandler) VerifyEmail(c *gin.Context) {
// 	var req dto.VerifyEmailRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	resp, err := h.service.VerifyEmail(c.Request.Context(), &req)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Email verified successfully. Please proceed to register your school.")
// }

// // ============================================
// // STEP 4: REGISTER SCHOOL
// // ============================================

// // RegisterSchool handles school registration
// // POST /api/v1/onboarding/school
// func (h *OnboardingHandler) RegisterSchool(c *gin.Context) {
// 	var req dto.RegisterSchoolRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	resp, err := h.service.RegisterSchool(c.Request.Context(), &req)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "School registered successfully. Please review your information.")
// }

// // ============================================
// // STEP 6: CREATE SUBSCRIPTION
// // ============================================

// // CreateSubscription handles subscription creation with payment link
// // POST /api/v1/onboarding/subscription
// func (h *OnboardingHandler) CreateSubscription(c *gin.Context) {
// 	var req dto.CreateSubscriptionRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	resp, err := h.service.CreateSubscription(c.Request.Context(), &req)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Subscription created successfully. Please complete the payment to activate your account.")
// }

// // ============================================
// // GET ONBOARDING STATUS
// // ============================================

// // GetStatus returns the current status of an onboarding session
// // GET /api/v1/onboarding/status/:session_id
// func (h *OnboardingHandler) GetStatus(c *gin.Context) {
// 	sessionID := c.Param("session_id")
// 	if sessionID == "" {
// 		badRequest(c, "session_id is required")
// 		return
// 	}

// 	resp, err := h.service.GetOnboardingStatus(c.Request.Context(), sessionID)
// 	if err != nil {
// 		notFound(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Onboarding status retrieved successfully")
// }

// // ============================================
// // CHECK PAYMENT STATUS
// // ============================================

// // CheckPaymentStatus checks if payment has been completed
// // GET /api/v1/onboarding/payment-status/:session_id
// func (h *OnboardingHandler) CheckPaymentStatus(c *gin.Context) {
// 	sessionID := c.Param("session_id")
// 	if sessionID == "" {
// 		badRequest(c, "session_id is required")
// 		return
// 	}

// 	resp, err := h.service.CheckPaymentStatus(c.Request.Context(), sessionID)
// 	if err != nil {
// 		notFound(c, err.Error())
// 		return
// 	}

// 	message := "Payment status retrieved successfully"
// 	if resp.IsComplete {
// 		message = "Payment completed successfully! Your account is being activated."
// 	} else if resp.Status == "pending" {
// 		message = "Payment is still pending. Please complete the payment."
// 	}

// 	successWithMessage(c, resp, message)
// }

// // ============================================
// // GET SESSION BY EMAIL
// // ============================================

// // GetSessionByEmail retrieves an onboarding session by email
// // GET /api/v1/onboarding/session/email/:email
// func (h *OnboardingHandler) GetSessionByEmail(c *gin.Context) {
// 	email := c.Param("email")
// 	if email == "" {
// 		badRequest(c, "email is required")
// 		return
// 	}

// 	resp, err := h.service.GetSessionByEmail(c.Request.Context(), email)
// 	if err != nil {
// 		notFound(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, resp, "Session retrieved successfully")
// }

// // ============================================
// // RESEND OTP
// // ============================================

// // ResendVerificationOTP resends the email verification OTP
// // POST /api/v1/onboarding/resend-otp
// func (h *OnboardingHandler) ResendVerificationOTP(c *gin.Context) {
// 	var req struct {
// 		SessionID string `json:"session_id" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	err := h.service.ResendVerificationOTP(c.Request.Context(), req.SessionID)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, gin.H{
// 		"message": "Verification OTP has been resent successfully. Please check your email.",
// 	}, "OTP resent successfully")
// }

// // ============================================
// // RESEND PAYMENT LINK
// // ============================================

// // ResendPaymentLink resends the payment link email
// // POST /api/v1/onboarding/resend-payment-link
// func (h *OnboardingHandler) ResendPaymentLink(c *gin.Context) {
// 	var req struct {
// 		SessionID string `json:"session_id" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		badRequest(c, "Invalid request: "+err.Error())
// 		return
// 	}

// 	err := h.service.ResendPaymentLink(c.Request.Context(), req.SessionID)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, gin.H{
// 		"message": "Payment link has been resent successfully. Please check your email.",
// 	}, "Payment link resent successfully")
// }

// // ============================================
// // COMPLETE ONBOARDING (Manual Admin)
// // ============================================

// // CompleteOnboarding manually completes the onboarding process
// // POST /api/v1/onboarding/admin/complete/:session_id
// func (h *OnboardingHandler) CompleteOnboarding(c *gin.Context) {
// 	sessionID := c.Param("session_id")
// 	if sessionID == "" {
// 		badRequest(c, "session_id is required")
// 		return
// 	}

// 	resp, err := h.service.CompleteOnboarding(c.Request.Context(), sessionID)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	if resp.Success {
// 		successWithMessage(c, resp, resp.Message)
// 	} else {
// 		badRequest(c, resp.Message)
// 	}
// }

// // ============================================
// // CLEANUP EXPIRED SESSIONS (Admin Only)
// // ============================================

// // CleanupExpiredSessions removes expired sessions
// // DELETE /api/v1/onboarding/admin/cleanup
// func (h *OnboardingHandler) CleanupExpiredSessions(c *gin.Context) {
// 	err := h.service.CleanupExpiredSessions(c.Request.Context())
// 	if err != nil {
// 		internalServerError(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, gin.H{
// 		"message": "Expired sessions cleaned up successfully.",
// 	}, "Cleanup completed successfully")
// }

// // ============================================
// // GET ONBOARDING STATISTICS (Admin Only)
// // ============================================

// // GetOnboardingStats returns statistics about onboarding sessions
// // GET /api/v1/onboarding/admin/stats
// func (h *OnboardingHandler) GetOnboardingStats(c *gin.Context) {
// 	stats, err := h.service.GetOnboardingStats(c.Request.Context())
// 	if err != nil {
// 		internalServerError(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, stats, "Statistics retrieved successfully")
// }

// // ============================================
// // WEBHOOK ENDPOINT (Public)
// // ============================================

// // Webhook handles payment webhooks from various gateways
// // POST /api/v1/onboarding/webhook/:gateway
// func (h *OnboardingHandler) Webhook(c *gin.Context) {
// 	gateway := c.Param("gateway")
// 	if gateway == "" {
// 		badRequest(c, "gateway is required")
// 		return
// 	}

// 	// Get signature from header based on gateway
// 	var signature string
// 	switch gateway {
// 	case "paystack":
// 		signature = c.GetHeader("x-paystack-signature")
// 	case "flutterwave":
// 		signature = c.GetHeader("verif-hash")
// 	case "stripe":
// 		signature = c.GetHeader("stripe-signature")
// 	default:
// 		badRequest(c, "unsupported gateway")
// 		return
// 	}

// 	body, err := c.GetRawData()
// 	if err != nil {
// 		badRequest(c, "Failed to read request body")
// 		return
// 	}

// 	// Parse gateway
// 	var gatewayEnum models.PaymentGateway
// 	switch gateway {
// 	case "paystack":
// 		gatewayEnum = models.GatewayPaystack
// 	case "flutterwave":
// 		gatewayEnum = models.GatewayFlutterwave
// 	case "stripe":
// 		gatewayEnum = models.GatewayStripe
// 	default:
// 		badRequest(c, "unsupported gateway")
// 		return
// 	}

// 	// Process webhook
// 	err = h.service.ProcessWebhook(c.Request.Context(), body, signature, gatewayEnum)
// 	if err != nil {
// 		badRequest(c, err.Error())
// 		return
// 	}

// 	successWithMessage(c, gin.H{
// 		"status": "processed",
// 	}, "Webhook processed successfully")
// }




// // package handler

// // import (
// // 	"context"
// // 	// "fmt"
// // 	"net/http"

// // 	"cbt-api/internal/models"
// // 	"cbt-api/internal/onboarding/service"

// // 	"github.com/gin-gonic/gin"
// // )

// // type OnboardingHandler struct {
// // 	service *service.OnboardingService
// // }

// // func NewOnboardingHandler(service *service.OnboardingService) *OnboardingHandler {
// // 	return &OnboardingHandler{service: service}
// // }

// // // ============================================
// // // RESPONSE HELPERS
// // // ============================================

// // type Response struct {
// // 	Success bool        `json:"success"`
// // 	Message string      `json:"message,omitempty"`
// // 	Data    interface{} `json:"data,omitempty"`
// // 	Error   string      `json:"error,omitempty"`
// // }

// // func success(c *gin.Context, data interface{}) {
// // 	c.JSON(http.StatusOK, Response{
// // 		Success: true,
// // 		Data:    data,
// // 	})
// // }

// // func successWithMessage(c *gin.Context, data interface{}, message string) {
// // 	c.JSON(http.StatusOK, Response{
// // 		Success: true,
// // 		Message: message,
// // 		Data:    data,
// // 	})
// // }

// // func errorResponse(c *gin.Context, statusCode int, err string) {
// // 	c.JSON(statusCode, Response{
// // 		Success: false,
// // 		Error:   err,
// // 	})
// // }

// // func badRequest(c *gin.Context, message string) {
// // 	c.JSON(http.StatusBadRequest, Response{
// // 		Success: false,
// // 		Error:   message,
// // 	})
// // }

// // func notFound(c *gin.Context, message string) {
// // 	c.JSON(http.StatusNotFound, Response{
// // 		Success: false,
// // 		Error:   message,
// // 	})
// // }

// // func internalServerError(c *gin.Context, message string) {
// // 	c.JSON(http.StatusInternalServerError, Response{
// // 		Success: false,
// // 		Error:   message,
// // 	})
// // }

// // // ============================================
// // // GET AVAILABLE GATEWAYS
// // // ============================================

// // // GetAvailableGateways returns all available payment gateways
// // // GET /api/onboarding/gateways
// // func (h *OnboardingHandler) GetAvailableGateways(c *gin.Context) {
// // 	gateways := h.service.GetAvailableGateways(c.Request.Context())
// // 	successWithMessage(c, gateways, "Available payment gateways retrieved successfully")
// // }

// // // ============================================
// // // STEP 1: SELECT PLAN
// // // ============================================

// // // StartOnboarding handles plan selection
// // // POST /api/onboarding/start
// // func (h *OnboardingHandler) StartOnboarding(c *gin.Context) {
// // 	var req service.StartOnboardingRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	// Add IP and User-Agent to context for logging
// // 	ctx := c.Request.Context()
// // 	ctx = context.WithValue(ctx, "ip_address", c.ClientIP())
// // 	ctx = context.WithValue(ctx, "user_agent", c.GetHeader("User-Agent"))

// // 	resp, err := h.service.StartOnboarding(ctx, &req)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, resp, "Plan selected successfully. Please proceed to create your admin account.")
// // }

// // // ============================================
// // // STEP 2: CREATE ADMIN ACCOUNT
// // // ============================================

// // // CreateAdminAccount handles admin account creation
// // // POST /api/onboarding/admin
// // func (h *OnboardingHandler) CreateAdminAccount(c *gin.Context) {
// // 	var req service.CreateAdminRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	resp, err := h.service.CreateAdminAccount(c.Request.Context(), &req)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, resp, "Admin account created successfully. Please verify your email.")
// // }

// // // ============================================
// // // STEP 3: VERIFY EMAIL
// // // ============================================

// // // VerifyEmail handles email verification
// // // POST /api/onboarding/verify-email
// // func (h *OnboardingHandler) VerifyEmail(c *gin.Context) {
// // 	var req service.VerifyEmailRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	resp, err := h.service.VerifyEmail(c.Request.Context(), &req)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, resp, "Email verified successfully. Please proceed to register your school.")
// // }

// // // ============================================
// // // STEP 4: REGISTER SCHOOL
// // // ============================================

// // // RegisterSchool handles school registration
// // // POST /api/onboarding/school
// // func (h *OnboardingHandler) RegisterSchool(c *gin.Context) {
// // 	var req service.RegisterSchoolRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	resp, err := h.service.RegisterSchool(c.Request.Context(), &req)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, resp, "School registered successfully. Please review your information.")
// // }

// // // ============================================
// // // STEP 6: CREATE SUBSCRIPTION
// // // ============================================

// // // CreateSubscription handles subscription creation with payment link
// // // POST /api/onboarding/subscription
// // func (h *OnboardingHandler) CreateSubscription(c *gin.Context) {
// // 	var req service.CreateSubscriptionRequest
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	resp, err := h.service.CreateSubscription(c.Request.Context(), &req)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, resp, "Subscription created successfully. Please complete the payment to activate your account.")
// // }

// // // ============================================
// // // GET ONBOARDING STATUS
// // // ============================================

// // // GetStatus returns the current status of an onboarding session
// // // GET /api/onboarding/status/:session_id
// // func (h *OnboardingHandler) GetStatus(c *gin.Context) {
// // 	sessionID := c.Param("session_id")
// // 	if sessionID == "" {
// // 		badRequest(c, "session_id is required")
// // 		return
// // 	}

// // 	session, err := h.service.GetOnboardingStatus(c.Request.Context(), sessionID)
// // 	if err != nil {
// // 		notFound(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, session, "Onboarding status retrieved successfully")
// // }

// // // ============================================
// // // CHECK PAYMENT STATUS
// // // ============================================

// // // CheckPaymentStatus checks if payment has been completed
// // // GET /api/onboarding/payment-status/:session_id
// // func (h *OnboardingHandler) CheckPaymentStatus(c *gin.Context) {
// // 	sessionID := c.Param("session_id")
// // 	if sessionID == "" {
// // 		badRequest(c, "session_id is required")
// // 		return
// // 	}

// // 	status, err := h.service.CheckPaymentStatus(c.Request.Context(), sessionID)
// // 	if err != nil {
// // 		notFound(c, err.Error())
// // 		return
// // 	}

// // 	message := "Payment status retrieved successfully"
// // 	if status.IsComplete {
// // 		message = "Payment completed successfully! Your account is being activated."
// // 	} else if status.Status == "pending" {
// // 		message = "Payment is still pending. Please complete the payment."
// // 	}

// // 	successWithMessage(c, status, message)
// // }

// // // ============================================
// // // GET SESSION BY EMAIL
// // // ============================================

// // // GetSessionByEmail retrieves an onboarding session by email
// // // GET /api/onboarding/session/email/:email
// // func (h *OnboardingHandler) GetSessionByEmail(c *gin.Context) {
// // 	email := c.Param("email")
// // 	if email == "" {
// // 		badRequest(c, "email is required")
// // 		return
// // 	}

// // 	session, err := h.service.GetSessionByEmail(c.Request.Context(), email)
// // 	if err != nil {
// // 		notFound(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, session, "Session retrieved successfully")
// // }

// // // ============================================
// // // RESEND OTP
// // // ============================================

// // // ResendVerificationOTP resends the email verification OTP
// // // POST /api/onboarding/resend-otp
// // func (h *OnboardingHandler) ResendVerificationOTP(c *gin.Context) {
// // 	var req struct {
// // 		SessionID string `json:"session_id" binding:"required"`
// // 	}
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	err := h.service.ResendVerificationOTP(c.Request.Context(), req.SessionID)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, gin.H{
// // 		"message": "Verification OTP has been resent successfully. Please check your email.",
// // 	}, "OTP resent successfully")
// // }

// // // ============================================
// // // RESEND PAYMENT LINK
// // // ============================================

// // // ResendPaymentLink resends the payment link email
// // // POST /api/onboarding/resend-payment-link
// // func (h *OnboardingHandler) ResendPaymentLink(c *gin.Context) {
// // 	var req struct {
// // 		SessionID string `json:"session_id" binding:"required"`
// // 	}
// // 	if err := c.ShouldBindJSON(&req); err != nil {
// // 		badRequest(c, "Invalid request: "+err.Error())
// // 		return
// // 	}

// // 	err := h.service.ResendPaymentLink(c.Request.Context(), req.SessionID)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, gin.H{
// // 		"message": "Payment link has been resent successfully. Please check your email.",
// // 	}, "Payment link resent successfully")
// // }

// // // ============================================
// // // COMPLETE ONBOARDING (Manual Admin)
// // // ============================================

// // // CompleteOnboarding manually completes the onboarding process
// // // POST /api/onboarding/admin/complete/:session_id
// // func (h *OnboardingHandler) CompleteOnboarding(c *gin.Context) {
// // 	sessionID := c.Param("session_id")
// // 	if sessionID == "" {
// // 		badRequest(c, "session_id is required")
// // 		return
// // 	}

// // 	resp, err := h.service.CompleteOnboarding(c.Request.Context(), sessionID)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	if resp.Success {
// // 		successWithMessage(c, resp, resp.Message)
// // 	} else {
// // 		badRequest(c, resp.Message)
// // 	}
// // }

// // // ============================================
// // // CLEANUP EXPIRED SESSIONS (Admin Only)
// // // ============================================

// // // CleanupExpiredSessions removes expired sessions
// // // DELETE /api/onboarding/admin/cleanup
// // func (h *OnboardingHandler) CleanupExpiredSessions(c *gin.Context) {
// // 	err := h.service.CleanupExpiredSessions(c.Request.Context())
// // 	if err != nil {
// // 		internalServerError(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, gin.H{
// // 		"message": "Expired sessions cleaned up successfully.",
// // 	}, "Cleanup completed successfully")
// // }

// // // ============================================
// // // GET ONBOARDING STATISTICS (Admin Only)
// // // ============================================

// // // GetOnboardingStats returns statistics about onboarding sessions
// // // GET /api/onboarding/admin/stats
// // func (h *OnboardingHandler) GetOnboardingStats(c *gin.Context) {
// // 	stats, err := h.service.GetOnboardingStats(c.Request.Context())
// // 	if err != nil {
// // 		internalServerError(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, stats, "Statistics retrieved successfully")
// // }

// // // ============================================
// // // WEBHOOK ENDPOINT (Public)
// // // ============================================

// // // Webhook handles payment webhooks from various gateways
// // // POST /api/onboarding/webhook/:gateway
// // func (h *OnboardingHandler) Webhook(c *gin.Context) {
// // 	gateway := c.Param("gateway")
// // 	if gateway == "" {
// // 		badRequest(c, "gateway is required")
// // 		return
// // 	}

// // 	// Get signature from header based on gateway
// // 	var signature string
// // 	switch gateway {
// // 	case "paystack":
// // 		signature = c.GetHeader("x-paystack-signature")
// // 	case "flutterwave":
// // 		signature = c.GetHeader("verif-hash")
// // 	case "stripe":
// // 		signature = c.GetHeader("stripe-signature")
// // 	default:
// // 		badRequest(c, "unsupported gateway")
// // 		return
// // 	}

// // 	body, err := c.GetRawData()
// // 	if err != nil {
// // 		badRequest(c, "Failed to read request body")
// // 		return
// // 	}

// // 	// Parse gateway
// // 	var gatewayEnum models.PaymentGateway
// // 	switch gateway {
// // 	case "paystack":
// // 		gatewayEnum = models.GatewayPaystack
// // 	case "flutterwave":
// // 		gatewayEnum = models.GatewayFlutterwave
// // 	case "stripe":
// // 		gatewayEnum = models.GatewayStripe
// // 	default:
// // 		badRequest(c, "unsupported gateway")
// // 		return
// // 	}

// // 	// Process webhook
// // 	err = h.service.ProcessWebhook(c.Request.Context(), body, signature, gatewayEnum)
// // 	if err != nil {
// // 		badRequest(c, err.Error())
// // 		return
// // 	}

// // 	successWithMessage(c, gin.H{
// // 		"status": "processed",
// // 	}, "Webhook processed successfully")
// // }



// // // package handler

// // // import (
// // //     "net/http"

// // //     "github.com/gin-gonic/gin"

// // //     "cbt-api/internal/onboarding/dto"
// // //     "cbt-api/internal/onboarding/service"
// // // )

// // // type OnboardingHandler struct {
// // //     service *service.OnboardingService
// // // }

// // // func NewOnboardingHandler(svc *service.OnboardingService) *OnboardingHandler {
// // //     return &OnboardingHandler{service: svc}
// // // }

// // // // SelectPlan godoc
// // // // @Summary      Select subscription plan
// // // // @Description  Start onboarding by selecting a plan
// // // // @Tags         Onboarding
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.PlanSelectionRequest true "Plan selection"
// // // // @Success      200  {object}  map[string]interface{}
// // // // @Router       /onboarding/plan [post]
// // // func (h *OnboardingHandler) SelectPlan(c *gin.Context) {
// // //     var req dto.PlanSelectionRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     response, err := h.service.SelectPlan(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     c.JSON(http.StatusOK, gin.H{"message": "Plan selected", "data": response})
// // // }

// // // // CreateAdministrator godoc
// // // // @Summary      Create administrator account
// // // // @Tags         Onboarding
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.CreateAdminRequest true "Admin details"
// // // // @Success      201  {object}  map[string]interface{}
// // // // @Router       /onboarding/admin [post]
// // // func (h *OnboardingHandler) CreateAdministrator(c *gin.Context) {
// // //     var req dto.CreateAdminRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     response, err := h.service.CreateAdministrator(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     c.JSON(http.StatusCreated, gin.H{"message": "Admin created. Please verify email.", "data": response})
// // // }

// // // // VerifyEmail godoc
// // // // @Summary      Verify email with OTP
// // // // @Tags         Onboarding
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.VerifyEmailRequest true "OTP verification"
// // // // @Success      200  {object}  map[string]interface{}
// // // // @Router       /onboarding/verify-email [post]
// // // func (h *OnboardingHandler) VerifyEmail(c *gin.Context) {
// // //     var req dto.VerifyEmailRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     response, err := h.service.VerifyEmail(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     c.JSON(http.StatusOK, gin.H{"message": "Email verified", "data": response})
// // // }

// // // // CreateSchool godoc
// // // // @Summary      Register school information
// // // // @Tags         Onboarding
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.SchoolRegistrationRequest true "School details"
// // // // @Success      201  {object}  map[string]interface{}
// // // // @Router       /onboarding/school [post]
// // // func (h *OnboardingHandler) CreateSchool(c *gin.Context) {
// // //     var req dto.SchoolRegistrationRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     response, err := h.service.CreateSchool(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     c.JSON(http.StatusCreated, gin.H{"message": "School registered", "data": response})
// // // }

// // // // CreateSubscription godoc
// // // // @Summary      Create subscription and payment intent
// // // // @Tags         Onboarding
// // // // @Accept       json
// // // // @Produce      json
// // // // @Param        request body dto.CreateSubscriptionOnboardingRequest true "Subscription creation"
// // // // @Success      201  {object}  map[string]interface{}
// // // // @Router       /onboarding/subscription [post]
// // // func (h *OnboardingHandler) CreateSubscription(c *gin.Context) {
// // //     var req dto.CreateSubscriptionOnboardingRequest
// // //     if err := c.ShouldBindJSON(&req); err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     response, err := h.service.CreateSubscription(&req)
// // //     if err != nil {
// // //         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// // //         return
// // //     }

// // //     c.JSON(http.StatusCreated, gin.H{"message": "Subscription created. Please complete payment.", "data": response})
// // // }