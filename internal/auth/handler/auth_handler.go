package handler

import (
	"net/http"
	"strings"
	"time"

	"cbt-api/internal/auth/dto"
	"cbt-api/internal/auth/service"
	"cbt-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account (student role by default). An email verification OTP will be sent.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration details"
// @Success      201  {object}  map[string]interface{}  "message + data (LoginResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	response, err := h.service.Register(&req)
	if err != nil {
		if strings.Contains(err.Error(), "email already registered") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email address is already registered. Please use a different email or login."})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully. Please verify your email.",
		"data":    response,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate with email and password. If 2FA is enabled, a 2FA token is required.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200  {object}  map[string]interface{}  "message + data (LoginResponse or 2FA_REQUIRED)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login request: " + err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response, err := h.service.Login(&req, clientIP, userAgent)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "invalid credentials"):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect email or password. Please try again."})
		case strings.Contains(errMsg, "account is deactivated"):
			c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been deactivated. Contact support for assistance."})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data":    response,
	})
}

// Verify2FALogin godoc
// @Summary      Verify 2FA code during login
// @Description  Complete 2FA authentication after login returned "2FA_REQUIRED".
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object true "2FA verification"  (email, code)
// @Success      200  {object}  map[string]interface{}  "message + data (LoginResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/verify-2fa [post]
func (h *AuthHandler) Verify2FALogin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 2FA verification request: " + err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response, err := h.service.Verify2FALogin(req.Email, req.Code, clientIP, userAgent)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "No account found with this email address."})
		} else if strings.Contains(errMsg, "invalid 2FA code") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA verification code. Please try again."})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA verification successful",
		"data":    response,
	})
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Obtain a new access token using a valid refresh token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshTokenRequest true "Refresh token"
// @Success      200  {object}  map[string]interface{}  "message + data (TokenResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid refresh token request: " + err.Error()})
		return
	}

	response, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid refresh token") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token is invalid or has expired. Please login again."})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"data":    response,
	})
}

// Logout godoc
// @Summary      Logout from current session
// @Description  Invalidate the current session (access token). Requires authentication.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessionID := middleware.GetSessionID(c)

	if err := h.service.Logout(userID, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log out. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAllDevices godoc
// @Summary      Logout from all devices
// @Description  Invalidate all sessions for the authenticated user, except the current one.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAllDevices(c *gin.Context) {
	userID := middleware.GetUserID(c)
	currentSessionID := middleware.GetSessionID(c)

	if err := h.service.LogoutAllDevices(userID, currentSessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log out from all devices. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices successfully"})
}

// SendVerificationOTP godoc
// @Summary      Send OTP for email verification or password reset
// @Description  Send a one‑time passcode to the user's email address.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.SendOTPRequest true "OTP request (email, type)"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/send-otp [post]
func (h *AuthHandler) SendVerificationOTP(c *gin.Context) {
	var req dto.SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.service.SendVerificationOTP(req.Email, req.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification code. Please try again later."})
		return
	}

	message := "Verification code sent"
	if req.Type == "email_verification" {
		message = "Email verification code sent successfully. Check your inbox."
	} else if req.Type == "password_reset" {
		message = "Password reset code sent successfully. Check your inbox."
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// VerifyEmail godoc
// @Summary      Verify email address with OTP
// @Description  Confirm the user's email using the code received via /send-otp.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.VerifyEmailRequest true "Verification code"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification request: " + err.Error()})
		return
	}

	if err := h.service.VerifyOTP(req.Email, req.Code, "email_verification"); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid or expired code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "The verification code is invalid or has expired. Request a new code."})
		} else if strings.Contains(errMsg, "invalid code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect verification code. Please check and try again."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully. You can now log in."})
}

// ForgotPassword godoc
// @Summary      Request password reset OTP
// @Description  Send a password reset OTP to the registered email (if exists).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.ForgotPasswordRequest true "Email address"
// @Success      200  {object}  map[string]interface{}  "message (always same for security)"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.service.SendVerificationOTP(req.Email, "password_reset"); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "If your email is registered, you will receive a password reset code."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If your email is registered, you will receive a password reset code."})
}

// ResetPassword godoc
// @Summary      Reset password using OTP
// @Description  Change password after verifying the reset OTP.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.ResetPasswordRequest true "Reset details (email, code, new_password)"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reset request: " + err.Error()})
		return
	}

	if err := h.service.ResetPassword(&req); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid or expired code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "The reset code is invalid or has expired. Request a new code."})
		} else if strings.Contains(errMsg, "invalid code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect reset code. Please check and try again."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now log in with your new password."})
}

// ChangePassword godoc
// @Summary      Change current user's password (authenticated)
// @Description  Requires current password and new password.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.ChangePasswordRequest true "Current and new password"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.service.ChangePassword(userID, &req); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "current password is incorrect") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect. Please try again."})
		} else if strings.Contains(errMsg, "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User account not found."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully. Use your new password for future logins."})
}

// Generate2FA godoc
// @Summary      Generate 2FA secret (authenticated)
// @Description  Generate a TOTP secret and QR code for Google Authenticator.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "message + data (TwoFactorResponse)"
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/2fa/generate [post]
func (h *AuthHandler) Generate2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)

	response, err := h.service.Generate2FASecret(userID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User account not found."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate 2FA secret. Please try again."})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA secret generated. Scan the QR code with your authenticator app.",
		"data":    response,
	})
}

// Enable2FA godoc
// @Summary      Enable two‑factor authentication
// @Description  Verify a TOTP code and enable 2FA for the account.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.Enable2FARequest true "TOTP code"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/2fa/enable [post]
func (h *AuthHandler) Enable2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.service.Enable2FA(userID, req.Code); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid 2FA code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 2FA code. Please check your authenticator app and try again."})
		} else if strings.Contains(errMsg, "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User account not found."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled successfully. Future logins will require a verification code."})
}

// Disable2FA godoc
// @Summary      Disable two‑factor authentication
// @Description  Disable 2FA for the account after verifying the TOTP code.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.Disable2FARequest true "TOTP code"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/2fa/disable [post]
func (h *AuthHandler) Disable2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.service.Disable2FA(userID, req.Code); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid 2FA code") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 2FA code. Please check your authenticator app and try again."})
		} else if strings.Contains(errMsg, "2FA is not enabled") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Two-factor authentication is not enabled for this account."})
		} else if strings.Contains(errMsg, "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User account not found."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled successfully. Two-factor authentication is no longer required."})
}

// GetSessions godoc
// @Summary      Get all active sessions
// @Description  List all active sessions for the authenticated user.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "data (list of sessions)"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/sessions [get]
func (h *AuthHandler) GetSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	currentSessionID := middleware.GetSessionID(c)

	sessions, err := h.service.GetUserSessions(userID, currentSessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve active sessions. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

// RevokeSession godoc
// @Summary      Revoke a specific session
// @Description  Force logout a device by revoking its session.
// @Tags         Authentication
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/sessions/{sessionId} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	if err := h.service.RevokeSession(userID, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not revoke session. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked successfully. The device has been logged out."})
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Retrieve the authenticated user's profile information.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "data (UserDTO)"
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	profile, err := h.service.GetProfile(userID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found."})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update the authenticated user's profile (first name, last name, phone number).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object true "Profile fields"  (first_name, last_name, phone_number)
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		PhoneNumber string `json:"phone_number"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update request: " + err.Error()})
		return
	}

	if err := h.service.UpdateProfile(userID, req.FirstName, req.LastName, req.PhoneNumber); err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User account not found."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update profile. Please try again."})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully."})
}

// DeleteAccount godoc
// @Summary      Delete user account (soft delete)
// @Description  Permanently soft‑delete the authenticated user's account.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /auth/profile [delete]
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.service.DeleteAccount(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete account. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully. We're sorry to see you go."})
}

// HealthCheck godoc
// @Summary      Health check endpoint
// @Description  Check if the API is running.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "status, message, timestamp"
// @Router       /health [get]
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"message":   "CBT API is running",
		"timestamp": time.Now().Unix(),
	})
}
