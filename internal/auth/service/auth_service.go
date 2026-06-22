package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"cbt-api/internal/auth/dto"
	"cbt-api/internal/auth/repository"
	"cbt-api/internal/models"
	parentService "cbt-api/internal/parent/service"
	"cbt-api/pkg/email"
	"cbt-api/pkg/utils"

	"github.com/pquerna/otp/totp"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthService struct {
	repo          *repository.AuthRepository
	emailService  *email.EmailService
	parentService *parentService.ParentService
	logger        *zap.Logger
}

func NewAuthService(
	repo *repository.AuthRepository,
	parentService *parentService.ParentService,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		repo:          repo,
		emailService:  email.NewEmailService(),
		parentService: parentService,
		logger:        logger,
	}
}

func (s *AuthService) generateUniqueUsername(base string) string {
	username := base
	counter := 0
	for {
		_, err := s.repo.FindUserByUsername(username)
		if err != nil {
			break
		}
		counter++
		username = fmt.Sprintf("%s%d", base, counter)
	}
	return username
}

// Register with role, username, parent linking
func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.LoginResponse, error) {
	if req.Role == "parent" && req.AdmissionNumber == "" {
		return nil, errors.New("admission_number is required for parent registration")
	}
	if req.Role != "student" && req.Email == "" {
		return nil, errors.New("email is required for non-student roles")
	}
	if req.Role == "student" && req.Username == "" {
		base := req.FirstName + req.LastName
		if len(base) > 20 {
			base = base[:20]
		}
		req.Username = s.generateUniqueUsername(base)
	}
	if req.Username != "" {
		existing, _ := s.repo.FindUserByUsername(req.Username)
		if existing != nil {
			return nil, errors.New("username already taken")
		}
	}
	if req.Email != "" {
		existing, _ := s.repo.FindUserByEmail(req.Email)
		if existing != nil {
			return nil, errors.New("email already registered")
		}
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var role models.UserRole
	switch req.Role {
	case "admin":
		role = models.RoleAdmin
	case "teacher":
		role = models.RoleTeacher
	case "parent":
		role = models.RoleParent
	default:
		role = models.RoleStudent
	}
	var emailPtr *string
	if req.Email != "" {
		emailPtr = &req.Email
	}
	user := &models.User{
		Username:      req.Username,
		Email:         emailPtr,
		Password:      hashedPassword,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		PhoneNumber:   req.PhoneNumber,
		Role:          role,
		Status:        models.StatusPending,
		IsActive:      true,
		EmailVerified: false,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}
	if req.Email != "" {
		go s.SendVerificationOTP(req.Email, "email_verification")
	}
	if role == models.RoleParent && s.parentService != nil && req.AdmissionNumber != "" {
		if err := s.parentService.AutoLinkParentOnRegister(user.ID, req.AdmissionNumber); err != nil {
			s.logger.Error("parent auto‑link failed", zap.String("parent_id", user.ID), zap.Error(err))
		}
	}

	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, err
	}
	session := &models.UserSession{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	_ = s.repo.CreateSession(session)

	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		TokenType:    "Bearer",
		User: dto.UserDTO{
			ID:               user.ID,
			Username:         user.Username,
			Email:            emailStr,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			PhoneNumber:      user.PhoneNumber,
			Role:             string(user.Role),
			Status:           string(user.Status),
			EmailVerified:    user.EmailVerified,
			TwoFactorEnabled: user.TwoFactorEnabled,
			CreatedAt:        user.CreatedAt,
		},
	}, nil
}

// Login with username or email
func (s *AuthService) Login(req *dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
	var user *models.User
	var err error
	if req.Username != "" {
		user, err = s.repo.FindUserByUsername(req.Username)
	} else if req.Email != "" {
		user, err = s.repo.FindUserByEmail(req.Email)
	} else {
		return nil, errors.New("username or email required")
	}
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}
	if user.TwoFactorEnabled {
		emailStr := ""
		if user.Email != nil {
			emailStr = *user.Email
		}
		return &dto.LoginResponse{
			TokenType: "2FA_REQUIRED",
			User: dto.UserDTO{
				ID:               user.ID,
				Username:         user.Username,
				Email:            emailStr,
				FirstName:        user.FirstName,
				LastName:         user.LastName,
				TwoFactorEnabled: true,
			},
		}, nil
	}
	return s.completeLogin(user, clientIP, userAgent)
}

func (s *AuthService) completeLogin(user *models.User, clientIP, userAgent string) (*dto.LoginResponse, error) {
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}
	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, err
	}
	session := &models.UserSession{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIP:     clientIP,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	_ = s.repo.CreateSession(session)
	_ = s.repo.UpdateLastLogin(user.ID)
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		TokenType:    "Bearer",
		User: dto.UserDTO{
			ID:               user.ID,
			Username:         user.Username,
			Email:            emailStr,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			PhoneNumber:      user.PhoneNumber,
			Role:             string(user.Role),
			Status:           string(user.Status),
			EmailVerified:    user.EmailVerified,
			TwoFactorEnabled: user.TwoFactorEnabled,
			CreatedAt:        user.CreatedAt,
		},
	}, nil
}

// Verify2FALogin verifies 2FA code during login
func (s *AuthService) Verify2FALogin(email, code, clientIP, userAgent string) (*dto.LoginResponse, error) {
    user, err := s.repo.FindUserByEmail(email)
    if err != nil {
        return nil, errors.New("user not found")
    }
    // Verify TOTP code
    valid := totp.Validate(code, user.TwoFactorSecret)
    if !valid {
        return nil, errors.New("invalid 2FA code")
    }
    return s.completeLogin(user, clientIP, userAgent)
}


func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
	session, err := s.repo.FindSessionByRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}
	user, err := s.repo.FindUserByID(session.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	accessToken, newRefreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, err
	}
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	_ = s.repo.UpdateUser(user)
	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) Logout(userID, sessionID string) error {
	return s.repo.DeleteSession(sessionID)
}

func (s *AuthService) LogoutAllDevices(userID, currentSessionID string) error {
	return s.repo.DeleteAllUserSessions(userID, currentSessionID)
}

func (s *AuthService) SendVerificationOTP(email, otpType string) error {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil
	}
	s.repo.InvalidateUserOTPs(user.ID, otpType)
	code := generateOTP(6)
	hashedCode, _ := utils.HashPassword(code)
	otp := &models.OTP{
		UserID:    user.ID,
		Code:      hashedCode,
		Type:      otpType,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.repo.CreateOTP(otp); err != nil {
		return err
	}
	go s.sendOTPEmail(email, code, otpType)
	return nil
}

func (s *AuthService) VerifyOTP(email, code, otpType string) error {
	otp, err := s.repo.FindValidOTP(email, code, otpType)
	if err != nil {
		return errors.New("invalid or expired code")
	}
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return err
	}
	if !utils.CheckPasswordHash(code, otp.Code) {
		return errors.New("invalid code")
	}
	if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
		return err
	}
	switch otpType {
	case "email_verification":
		return s.repo.VerifyEmail(user.ID)
	}
	return nil
}

func (s *AuthService) ResetPassword(req *dto.ResetPasswordRequest) error {
	otp, err := s.repo.FindValidOTP(req.Email, req.Code, "password_reset")
	if err != nil {
		return errors.New("invalid or expired code")
	}
	user, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return err
	}
	if !utils.CheckPasswordHash(req.Code, otp.Code) {
		return errors.New("invalid code")
	}
	s.repo.MarkOTPAsUsed(otp.ID)
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(user.ID, hashedPassword); err != nil {
		return err
	}
	s.repo.DeleteAllUserSessions(user.ID, "")
	return nil
}

func (s *AuthService) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
		return errors.New("current password is incorrect")
	}
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(userID, hashedPassword)
}

func (s *AuthService) Generate2FASecret(userID string) (*dto.TwoFactorResponse, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	accountName := user.Username
	if user.Email != nil {
		accountName = *user.Email
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CBT-API",
		AccountName: accountName,
		Period:      30,
		Digits:      6,
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateTwoFactorSecret(userID, key.Secret()); err != nil {
		return nil, err
	}
	return &dto.TwoFactorResponse{
		Secret:    key.Secret(),
		QRCodeURL: key.URL(),
		Enabled:   false,
		Message:   "Scan QR code with Google Authenticator",
	}, nil
}

func (s *AuthService) Enable2FA(userID, code string) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if !totp.Validate(code, user.TwoFactorSecret) {
		return errors.New("invalid 2FA code")
	}
	return s.repo.EnableTwoFactor(userID, user.TwoFactorSecret)
}

func (s *AuthService) Disable2FA(userID, code string) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if !user.TwoFactorEnabled {
		return errors.New("2FA is not enabled")
	}
	if !totp.Validate(code, user.TwoFactorSecret) {
		return errors.New("invalid 2FA code")
	}
	return s.repo.DisableTwoFactor(userID)
}

func (s *AuthService) GetUserSessions(userID, currentSessionID string) ([]dto.SessionDTO, error) {
	sessions, err := s.repo.FindSessionsByUserID(userID)
	if err != nil {
		return nil, err
	}
	var sessionDTOs []dto.SessionDTO
	for _, session := range sessions {
		sessionDTOs = append(sessionDTOs, dto.SessionDTO{
			ID:        session.ID,
			UserAgent: session.UserAgent,
			ClientIP:  session.ClientIP,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
			IsCurrent: session.ID == currentSessionID,
		})
	}
	return sessionDTOs, nil
}

func (s *AuthService) RevokeSession(userID, sessionID string) error {
	return s.repo.DeleteSession(sessionID)
}

func (s *AuthService) GetProfile(userID string) (*dto.UserDTO, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}
	return &dto.UserDTO{
		ID:               user.ID,
		Username:         user.Username,
		Email:            emailStr,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		PhoneNumber:      user.PhoneNumber,
		Role:             string(user.Role),
		Status:           string(user.Status),
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        user.CreatedAt,
	}, nil
}

func (s *AuthService) UpdateProfile(userID, firstName, lastName, phoneNumber string) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if firstName != "" {
		user.FirstName = firstName
	}
	if lastName != "" {
		user.LastName = lastName
	}
	if phoneNumber != "" {
		user.PhoneNumber = phoneNumber
	}
	return s.repo.UpdateUser(user)
}

func (s *AuthService) DeleteAccount(userID string) error {
	s.repo.DeleteAllUserSessions(userID, "")
	return s.repo.SoftDeleteUser(userID)
}

func (s *AuthService) generateTokens(userID, identifier, role string) (string, string, error) {
	accessToken, err := utils.GenerateJWT(userID, identifier, role, 24*time.Hour)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := utils.GenerateJWT(userID, identifier, role, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (s *AuthService) sendOTPEmail(emailAddr, code, otpType string) {
	user, err := s.repo.FindUserByEmail(emailAddr)
	if err != nil {
		fmt.Printf("Failed to get user for email: %v\n", err)
		return
	}
	var sendErr error
	if otpType == "email_verification" {
		sendErr = s.emailService.SendWelcomeEmail(emailAddr, user.FirstName, code)
	} else {
		sendErr = s.emailService.SendPasswordResetEmail(emailAddr, user.FirstName, code)
	}
	if sendErr != nil {
		fmt.Printf("Failed to send email: %v\n", sendErr)
	} else {
		fmt.Printf("Email sent successfully to %s\n", emailAddr)
	}
}

func generateOTP(length int) string {
	const charset = "0123456789"
	otp := make([]byte, length)
	rand.Read(otp)
	for i := range otp {
		otp[i] = charset[int(otp[i])%len(charset)]
	}
	return string(otp)
}


// package service

// import (
//     "crypto/rand"
//     "errors"
//     "fmt"
//     "time"

//     "cbt-api/internal/auth/dto"
//     "cbt-api/internal/auth/repository"
//     "cbt-api/internal/models"
// 	"cbt-api/pkg/email"
//     "cbt-api/pkg/utils"

//     "github.com/pquerna/otp/totp"
//     "github.com/google/uuid"
// )

// type AuthService struct {
//     repo *repository.AuthRepository
// 	emailService *email.EmailService

// }

// func NewAuthService(repo *repository.AuthRepository) *AuthService {
//     return &AuthService{
// 		repo: repo,
// 	    emailService: email.NewEmailService(),
// 		}
// }

// // Register new user
// func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.LoginResponse, error) {
//     existingUser, _ := s.repo.FindUserByEmail(req.Email)
//     if existingUser != nil {
//         return nil, errors.New("email already registered")
//     }

//     hashedPassword, err := utils.HashPassword(req.Password)
//     if err != nil {
//         return nil, err
//     }

//     user := &models.User{
//         Email:       req.Email,
//         Password:    hashedPassword,
//         FirstName:   req.FirstName,
//         LastName:    req.LastName,
//         PhoneNumber: req.PhoneNumber,
//         Role:        models.RoleStudent,
//         Status:      models.StatusPending,
//         IsActive:    true,
//     }

//     if err := s.repo.CreateUser(user); err != nil {
//         return nil, err
//     }

//     // Generate and send verification OTP
//     go s.SendVerificationOTP(req.Email, "email_verification")

//     // Generate tokens - user.ID is already string, no .String()
//     accessToken, refreshToken, err := s.generateTokens(user.ID, user.Email, string(user.Role))
//     if err != nil {
//         return nil, err
//     }

//     // Create session
//     session := &models.UserSession{
//         ID:           uuid.New().String(),
//         UserID:       user.ID,  // No .String()
//         RefreshToken: refreshToken,
//         ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
//     }
//     s.repo.CreateSession(session)

//     return &dto.LoginResponse{
//         AccessToken:  accessToken,
//         RefreshToken: refreshToken,
//         ExpiresAt:    time.Now().Add(24 * time.Hour),
//         TokenType:    "Bearer",
//         User: dto.UserDTO{
//             ID:            user.ID,  // No .String()
//             Email:         user.Email,
//             FirstName:     user.FirstName,
//             LastName:      user.LastName,
//             PhoneNumber:   user.PhoneNumber,
//             Role:          string(user.Role),
//             Status:        string(user.Status),
//             EmailVerified: user.EmailVerified,
//             CreatedAt:     user.CreatedAt,
//         },
//     }, nil
// }

// // Login user
// func (s *AuthService) Login(req *dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
//     user, err := s.repo.FindUserByEmail(req.Email)
//     if err != nil {
//         return nil, errors.New("invalid credentials")
//     }

//     if !utils.CheckPasswordHash(req.Password, user.Password) {
//         return nil, errors.New("invalid credentials")
//     }

//     if !user.IsActive {
//         return nil, errors.New("account is deactivated")
//     }

//     // Check if 2FA is enabled
//     if user.TwoFactorEnabled {
//         return &dto.LoginResponse{
//             TokenType: "2FA_REQUIRED",
//             User: dto.UserDTO{
//                 ID:               user.ID,  // No .String()
//                 Email:            user.Email,
//                 FirstName:        user.FirstName,
//                 LastName:         user.LastName,
//                 TwoFactorEnabled: true,
//             },
//         }, nil
//     }

//     return s.completeLogin(user, clientIP, userAgent)
// }

// // Verify 2FA during login
// func (s *AuthService) Verify2FALogin(email, code, clientIP, userAgent string) (*dto.LoginResponse, error) {
//     user, err := s.repo.FindUserByEmail(email)
//     if err != nil {
//         return nil, errors.New("user not found")
//     }

//     // Verify TOTP code
//     valid := totp.Validate(code, user.TwoFactorSecret)
//     if !valid {
//         return nil, errors.New("invalid 2FA code")
//     }

//     return s.completeLogin(user, clientIP, userAgent)
// }

// // Complete login after verification
// func (s *AuthService) completeLogin(user *models.User, clientIP, userAgent string) (*dto.LoginResponse, error) {
//     accessToken, refreshToken, err := s.generateTokens(user.ID, user.Email, string(user.Role))
//     if err != nil {
//         return nil, err
//     }

//     // Create session
//     session := &models.UserSession{
//         ID:           uuid.New().String(),
//         UserID:       user.ID,  // No .String()
//         RefreshToken: refreshToken,
//         UserAgent:    userAgent,
//         ClientIP:     clientIP,
//         ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
//     }
//     s.repo.CreateSession(session)
//     s.repo.UpdateLastLogin(user.ID)  // No .String()

//     return &dto.LoginResponse{
//         AccessToken:  accessToken,
//         RefreshToken: refreshToken,
//         ExpiresAt:    time.Now().Add(24 * time.Hour),
//         TokenType:    "Bearer",
//         User: dto.UserDTO{
//             ID:               user.ID,  // No .String()
//             Email:            user.Email,
//             FirstName:        user.FirstName,
//             LastName:         user.LastName,
//             PhoneNumber:      user.PhoneNumber,
//             Role:             string(user.Role),
//             Status:           string(user.Status),
//             EmailVerified:    user.EmailVerified,
//             TwoFactorEnabled: user.TwoFactorEnabled,
//             CreatedAt:        user.CreatedAt,
//         },
//     }, nil
// }

// // Refresh token
// func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
//     session, err := s.repo.FindSessionByRefreshToken(refreshToken)
//     if err != nil {
//         return nil, errors.New("invalid refresh token")
//     }

//     user, err := s.repo.FindUserByID(session.UserID)
//     if err != nil {
//         return nil, errors.New("user not found")
//     }

//     // Generate new tokens
//     accessToken, newRefreshToken, err := s.generateTokens(user.ID, user.Email, string(user.Role))
//     if err != nil {
//         return nil, err
//     }

//     // Update session with new refresh token
//     session.RefreshToken = newRefreshToken
//     session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
//     s.repo.UpdateUser(user)

//     return &dto.TokenResponse{
//         AccessToken:  accessToken,
//         RefreshToken: newRefreshToken,
//         ExpiresAt:    time.Now().Add(24 * time.Hour),
//         TokenType:    "Bearer",
//     }, nil
// }

// // Logout
// func (s *AuthService) Logout(userID, sessionID string) error {
//     return s.repo.DeleteSession(sessionID)
// }

// // Logout all devices
// func (s *AuthService) LogoutAllDevices(userID, currentSessionID string) error {
//     return s.repo.DeleteAllUserSessions(userID, currentSessionID)
// }

// // Send verification OTP (email verification or password reset)
// func (s *AuthService) SendVerificationOTP(email, otpType string) error {
//     user, err := s.repo.FindUserByEmail(email)
//     if err != nil {
//         // Don't reveal that email doesn't exist for security
//         return nil
//     }

//     // Invalidate existing OTPs of this type
//     s.repo.InvalidateUserOTPs(user.ID, otpType)

//     // Generate OTP code
//     code := generateOTP(6)
    
//     // Hash the code for storage
//     hashedCode, _ := utils.HashPassword(code)

//     otp := &models.OTP{
//         UserID:    user.ID,
//         Code:      hashedCode,
//         Type:      otpType,
//         ExpiresAt: time.Now().Add(10 * time.Minute),
//     }

//     if err := s.repo.CreateOTP(otp); err != nil {
//         return err
//     }

//     // Send email (implement email sender)
//     go s.sendOTPEmail(email, code, otpType)

//     return nil
// }

// // Verify OTP
// func (s *AuthService) VerifyOTP(email, code, otpType string) error {
//     otp, err := s.repo.FindValidOTP(email, code, otpType)
//     if err != nil {
//         return errors.New("invalid or expired code")
//     }

//     user, err := s.repo.FindUserByEmail(email)
//     if err != nil {
//         return err
//     }

//     // Verify the code matches (stored as hash)
//     if !utils.CheckPasswordHash(code, otp.Code) {
//         return errors.New("invalid code")
//     }

//     if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
//         return err
//     }

//     // Perform action based on type
//     switch otpType {
//     case "email_verification":
//         return s.repo.VerifyEmail(user.ID)
//     case "password_reset":
//         // Password reset is handled separately
//         return nil
//     }

//     return nil
// }

// // Reset password
// func (s *AuthService) ResetPassword(req *dto.ResetPasswordRequest) error {
//     // Verify OTP first
//     otp, err := s.repo.FindValidOTP(req.Email, req.Code, "password_reset")
//     if err != nil {
//         return errors.New("invalid or expired code")
//     }

//     user, err := s.repo.FindUserByEmail(req.Email)
//     if err != nil {
//         return err
//     }

//     // Verify code
//     if !utils.CheckPasswordHash(req.Code, otp.Code) {
//         return errors.New("invalid code")
//     }

//     // Mark OTP as used
//     s.repo.MarkOTPAsUsed(otp.ID)

//     // Hash new password
//     hashedPassword, err := utils.HashPassword(req.NewPassword)
//     if err != nil {
//         return err
//     }

//     // Update password
//     if err := s.repo.UpdatePassword(user.ID, hashedPassword); err != nil {
//         return err
//     }

//     // Invalidate all sessions for security
//     s.repo.DeleteAllUserSessions(user.ID, "")

//     return nil
// }

// // Change password
// func (s *AuthService) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return errors.New("user not found")
//     }

//     if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
//         return errors.New("current password is incorrect")
//     }

//     hashedPassword, err := utils.HashPassword(req.NewPassword)
//     if err != nil {
//         return err
//     }

//     return s.repo.UpdatePassword(userID, hashedPassword)
// }

// // Generate 2FA secret
// func (s *AuthService) Generate2FASecret(userID string) (*dto.TwoFactorResponse, error) {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return nil, errors.New("user not found")
//     }

//     // Generate TOTP secret
//     key, err := totp.Generate(totp.GenerateOpts{
//         Issuer:      "CBT-API",
//         AccountName: user.Email,
//         Period:      30,
//         Digits:      6,
//     })
//     if err != nil {
//         return nil, err
//     }

//     // Save secret temporarily
//     if err := s.repo.UpdateTwoFactorSecret(userID, key.Secret()); err != nil {
//         return nil, err
//     }

//     return &dto.TwoFactorResponse{
//         Secret:    key.Secret(),
//         QRCodeURL: key.URL(),
//         Enabled:   false,
//         Message:   "Scan QR code with Google Authenticator",
//     }, nil
// }

// // Enable 2FA
// func (s *AuthService) Enable2FA(userID, code string) error {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return errors.New("user not found")
//     }

//     // Verify TOTP code
//     valid := totp.Validate(code, user.TwoFactorSecret)
//     if !valid {
//         return errors.New("invalid 2FA code")
//     }

//     return s.repo.EnableTwoFactor(userID, user.TwoFactorSecret)
// }

// // Disable 2FA
// func (s *AuthService) Disable2FA(userID, code string) error {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return errors.New("user not found")
//     }

//     if !user.TwoFactorEnabled {
//         return errors.New("2FA is not enabled")
//     }

//     // Verify TOTP code before disabling
//     valid := totp.Validate(code, user.TwoFactorSecret)
//     if !valid {
//         return errors.New("invalid 2FA code")
//     }

//     return s.repo.DisableTwoFactor(userID)
// }

// // Get user sessions
// func (s *AuthService) GetUserSessions(userID, currentSessionID string) ([]dto.SessionDTO, error) {
//     sessions, err := s.repo.FindSessionsByUserID(userID)
//     if err != nil {
//         return nil, err
//     }

//     var sessionDTOs []dto.SessionDTO
//     for _, session := range sessions {
//         sessionDTOs = append(sessionDTOs, dto.SessionDTO{
//             ID:        session.ID,
//             UserAgent: session.UserAgent,
//             ClientIP:  session.ClientIP,
//             CreatedAt: session.CreatedAt,
//             ExpiresAt: session.ExpiresAt,
//             IsCurrent: session.ID == currentSessionID,
//         })
//     }

//     return sessionDTOs, nil
// }

// // Revoke session
// func (s *AuthService) RevokeSession(userID, sessionID string) error {
//     return s.repo.DeleteSession(sessionID)
// }

// // Get profile
// func (s *AuthService) GetProfile(userID string) (*dto.UserDTO, error) {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return nil, errors.New("user not found")
//     }

//     return &dto.UserDTO{
//         ID:               user.ID,  // No .String()
//         Email:            user.Email,
//         FirstName:        user.FirstName,
//         LastName:         user.LastName,
//         PhoneNumber:      user.PhoneNumber,
//         Role:             string(user.Role),
//         Status:           string(user.Status),
//         EmailVerified:    user.EmailVerified,
//         TwoFactorEnabled: user.TwoFactorEnabled,
//         CreatedAt:        user.CreatedAt,
//     }, nil
// }

// // UpdateProfile updates user profile
// func (s *AuthService) UpdateProfile(userID, firstName, lastName, phoneNumber string) error {
//     user, err := s.repo.FindUserByID(userID)
//     if err != nil {
//         return errors.New("user not found")
//     }
    
//     if firstName != "" {
//         user.FirstName = firstName
//     }
//     if lastName != "" {
//         user.LastName = lastName
//     }
//     if phoneNumber != "" {
//         user.PhoneNumber = phoneNumber
//     }
    
//     return s.repo.UpdateUser(user)
// }

// // DeleteAccount soft deletes user account
// func (s *AuthService) DeleteAccount(userID string) error {
//     // First, delete all sessions
//     s.repo.DeleteAllUserSessions(userID, "")
    
//     // Then soft delete the user
//     return s.repo.SoftDeleteUser(userID)
// }

// // Helper methods
// func (s *AuthService) generateTokens(userID, email, role string) (string, string, error) {
//     accessToken, err := utils.GenerateJWT(userID, email, role, 24*time.Hour)
//     if err != nil {
//         return "", "", err
//     }

//     refreshToken, err := utils.GenerateJWT(userID, email, role, 7*24*time.Hour)
//     if err != nil {
//         return "", "", err
//     }

//     return accessToken, refreshToken, nil
// }

// // func (s *AuthService) sendOTPEmail(email, code, otpType string) {
// //     // Implement email sending logic here
// //     fmt.Printf("Sending %s OTP %s to %s\n", otpType, code, email)
// // }

// func (s *AuthService) sendOTPEmail(emailAddr, code, otpType string) {
//     // Get user to get first name
//     user, err := s.repo.FindUserByEmail(emailAddr)
//     if err != nil {
//         fmt.Printf("Failed to get user for email: %v\n", err)
//         return
//     }
    
//     var sendErr error
//     if otpType == "email_verification" {
//         sendErr = s.emailService.SendWelcomeEmail(emailAddr, user.FirstName, code)
//     } else {
//         sendErr = s.emailService.SendPasswordResetEmail(emailAddr, user.FirstName, code)
//     }
    
//     if sendErr != nil {
//         fmt.Printf("Failed to send email: %v\n", sendErr)
//     } else {
//         fmt.Printf("Email sent successfully to %s\n", emailAddr)
//     }
// }

// func generateOTP(length int) string {
//     const charset = "0123456789"
//     otp := make([]byte, length)
//     rand.Read(otp)
//     for i := range otp {
//         otp[i] = charset[int(otp[i])%len(charset)]
//     }
//     return string(otp)
// }

