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

	// Use the existing academic repositories
	academicRepo "cbt-api/internal/academic/repository"
	subRepo "cbt-api/internal/subscription/repository"

	"github.com/pquerna/otp/totp"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthService struct {
	repo             *repository.AuthRepository
	emailService     *email.EmailService
	parentService    *parentService.ParentService
	schoolRepo       *academicRepo.SchoolRepository
	studentRepo      *academicRepo.StudentRepository
	parentStudentRepo *academicRepo.ParentRepository
	subscriptionRepo *subRepo.SubscriptionRepository
	logger           *zap.Logger
}

func NewAuthService(
	repo *repository.AuthRepository,
	parentService *parentService.ParentService,
	schoolRepo *academicRepo.SchoolRepository,
	studentRepo *academicRepo.StudentRepository,
	parentStudentRepo *academicRepo.ParentRepository,
	subscriptionRepo *subRepo.SubscriptionRepository,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		repo:             repo,
		emailService:     email.NewEmailService(),
		parentService:    parentService,
		schoolRepo:       schoolRepo,
		studentRepo:      studentRepo,
		parentStudentRepo: parentStudentRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
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

// Register with role, username, parent linking, and school code validation
// Register self-registers a new account. Before this fix, role=admin
// was accepted here same as teacher/parent: anyone who obtained an
// EXISTING school's school_code (itself leaked to any authenticated
// teacher via GET /schools, which returns every school's code - see
// that handler's own fix) could self-register as that school's admin
// and log in immediately (Login never checks Status, and IsActive is
// unconditionally true here) - full administrative control over a
// real, already-operating tenant with zero human approval step. This
// is categorically different from the onboarding flow's admin
// creation, which always provisions a brand-new school for that admin
// rather than attaching to an existing one. Admin accounts for an
// existing school must now come only from onboarding (new school) or
// GrantRole (an existing admin/platform_owner explicitly elevating a
// user they already vetted) - both already properly scoped.
func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.LoginResponse, error) {
	if req.Role == "admin" {
		return nil, errors.New("self-registration as admin is not permitted; use the onboarding flow to create a new school, or ask an existing admin to grant you the role")
	}
	// Validate school code for non-student roles
	var schoolID *string
	if req.Role != "student" {
		if req.SchoolCode == "" {
			return nil, errors.New("school_code is required for teacher, admin, and parent roles")
		}

		school, err := s.schoolRepo.FindByCode(req.SchoolCode)
		if err != nil {
			return nil, errors.New("invalid school code. Please check and try again")
		}
		schoolID = &school.ID
	}

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
		SchoolID:      schoolID, // Set school ID for non-student roles
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}
	if req.Email != "" {
		go s.SendVerificationOTP(req.Email, "email_verification")
	}
	if role == models.RoleParent && s.parentService != nil && req.AdmissionNumber != "" && schoolID != nil {
		if err := s.parentService.AutoLinkParentOnRegister(user.ID, req.AdmissionNumber, *schoolID); err != nil {
			s.logger.Error("parent auto‑link failed", zap.String("parent_id", user.ID), zap.Error(err))
		}
	}

	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role), schoolIDClaim(user))
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
			SchoolID:         user.SchoolID,
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
				SchoolID:         user.SchoolID,
			},
		}, nil
	}
	return s.completeLogin(user, clientIP, userAgent)
}

func (s *AuthService) completeLogin(user *models.User, clientIP, userAgent string) (*dto.LoginResponse, error) {
	return s.completeLoginWithRole(user, string(user.Role), clientIP, userAgent)
}

// completeLoginWithRole issues tokens for activeRole rather than always
// the account's primary user.Role - used by both the normal login path
// (activeRole == user.Role) and SwitchRole (activeRole == the granted
// role being switched to). Every existing permission check reads the
// JWT's role claim, so this is the one place that needs to know the
// difference between "primary role" and "active role for this session".
func (s *AuthService) completeLoginWithRole(user *models.User, activeRole, clientIP, userAgent string) (*dto.LoginResponse, error) {
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}
	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, activeRole, schoolIDClaim(user))
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
	availableRoles, err := s.GetAvailableRoles(user.ID)
	if err != nil {
		// Available-role lookup failing shouldn't block login/switch -
		// the active role (already validated by the caller) still works.
		availableRoles = []string{activeRole}
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
			Role:             activeRole,
			Status:           string(user.Status),
			EmailVerified:    user.EmailVerified,
			TwoFactorEnabled: user.TwoFactorEnabled,
			SchoolID:         user.SchoolID,
			CreatedAt:        user.CreatedAt,
			AvailableRoles:   availableRoles,
		},
	}, nil
}

// GetAvailableRoles returns every role this account can act as: its
// primary Role plus any extra UserRoleGrant rows, deduplicated.
func (s *AuthService) GetAvailableRoles(userID string) ([]string, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	grants, err := s.repo.FindRoleGrantsByUserID(userID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{string(user.Role): true}
	roles := []string{string(user.Role)}
	for _, g := range grants {
		if !seen[string(g.Role)] {
			seen[string(g.Role)] = true
			roles = append(roles, string(g.Role))
		}
	}
	return roles, nil
}

// SwitchRole re-issues tokens with a different active role, without
// requiring the user to log out. targetRole must be the account's
// primary role or one of its granted roles (GetAvailableRoles) - this
// is the server-side authorization check; it does not trust the
// client's claim that it's allowed to switch, and the resulting JWT's
// role claim is what every downstream permission check actually
// enforces (switching never writes to user.Role, so revoking a grant
// takes effect the next time that role is checked against the grants
// table - see RevokeRole).
func (s *AuthService) SwitchRole(userID, targetRole, clientIP, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	available, err := s.GetAvailableRoles(userID)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, r := range available {
		if r == targetRole {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("role not granted to this account")
	}
	return s.completeLoginWithRole(user, targetRole, clientIP, userAgent)
}

// GrantRole gives an additional role to a user. grantedBy is the admin
// user ID performing the grant, recorded for audit purposes. schoolID is
// the caller's own verified school (never a client-supplied value) and
// isPlatformOwner marks the one role meant to act across every tenant;
// every other caller may only grant a role to a user in their own
// school - a role grant never changes which school the target user's
// JWT claims (completeLoginWithRole always uses the target's own
// user.SchoolID), so without this check a School A admin could elevate
// a School B user's privileges within School B without ever being
// School B staff themselves. Before this fix there was no school check
// of any kind - FindUserByID looks up any user by raw ID regardless of
// school.
//
// platform_owner is deliberately never grantable here, however trusted
// the caller - this is the platform/school authority boundary from the
// multi-tenancy plan's Phase 1.3 ("do not trust... a user-controlled
// role value as proof of [platform] ownership"). An admin or super_admin
// calling this endpoint is always acting within their own school; it can
// never be used to mint platform-wide authority, for that admin or
// anyone else. The only path to platform_owner is the server-side
// bootstrap in pkg/database/postgres.go's seedPlatformOwner().
func (s *AuthService) GrantRole(targetUserID, role, grantedBy, schoolID string, isPlatformOwner bool) error {
	if models.UserRole(role) == models.RolePlatformOwner {
		return errors.New("platform_owner cannot be granted through this endpoint")
	}
	if schoolID == "" && !isPlatformOwner {
		return errors.New("no school scope for this account")
	}
	targetUser, err := s.repo.FindUserByID(targetUserID)
	if err != nil {
		return errors.New("user not found")
	}
	if !isPlatformOwner && schoolIDClaim(targetUser) != schoolID {
		return errors.New("user not found")
	}
	grant := &models.UserRoleGrant{
		ID:        uuid.New().String(),
		UserID:    targetUserID,
		Role:      models.UserRole(role),
		GrantedBy: &grantedBy,
	}
	return s.repo.CreateRoleGrant(grant)
}

// RevokeRole removes a previously granted role. It never removes a
// user's primary Role - only additional grants. Scoped to schoolID for
// the same reason as GrantRole: without a target-user ownership check
// (absent before this fix, along with any user lookup at all), a School
// A admin could revoke a School B user's additional role.
func (s *AuthService) RevokeRole(targetUserID, role, schoolID string, isPlatformOwner bool) error {
	if schoolID == "" && !isPlatformOwner {
		return errors.New("no school scope for this account")
	}
	targetUser, err := s.repo.FindUserByID(targetUserID)
	if err != nil {
		return errors.New("user not found")
	}
	if !isPlatformOwner && schoolIDClaim(targetUser) != schoolID {
		return errors.New("user not found")
	}
	return s.repo.DeleteRoleGrant(targetUserID, models.UserRole(role))
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
	accessToken, newRefreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role), schoolIDClaim(user))
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
	// sessionID here is always server-derived (middleware.GetSessionID,
	// from the caller's own JWT), never client input, so this is scoped
	// defensively rather than to close a live exploit. Zero rows affected
	// (session already gone, e.g. a double logout) is not an error -
	// logout must stay idempotent.
	_, err := s.repo.DeleteSessionForUser(sessionID, userID)
	return err
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

// RevokeSession deletes one of the CALLER's own sessions. sessionID
// comes straight from the URL path (c.Param("sessionId")) - client
// input - so without the userID ownership check below, any
// authenticated user could force-logout any OTHER user's session by ID,
// regardless of whose account it belongs to. A mismatch reports the
// same generic error as a nonexistent session, so a caller can't use
// this to probe which session IDs are real.
func (s *AuthService) RevokeSession(userID, sessionID string) error {
	rows, err := s.repo.DeleteSessionForUser(sessionID, userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("session not found")
	}
	return nil
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
		SchoolID:         user.SchoolID,
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

func (s *AuthService) generateTokens(userID, identifier, role, schoolID string) (string, string, error) {
	accessToken, err := utils.GenerateJWT(userID, identifier, role, schoolID, 24*time.Hour)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := utils.GenerateJWT(userID, identifier, role, schoolID, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// schoolIDClaim converts a user's possibly-nil SchoolID into the plain
// string GenerateJWT signs as the "school_id" claim ("" means no school -
// e.g. models.RolePlatformOwner).
func schoolIDClaim(user *models.User) string {
	if user == nil || user.SchoolID == nil {
		return ""
	}
	return *user.SchoolID
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

// ============================================
// SUBSCRIPTION CHECK METHOD
// ============================================

// CheckSubscription checks the subscription status for a user's school.
// schoolID is always derived server-side from the caller's own record
// (never a client-supplied value, and there is no such parameter on
// this endpoint to begin with) - a student's own student record, a
// teacher/admin/super_admin's own user.SchoolID, or a parent's own
// children's school. super_admin was missing from that resolution
// (grouped with teacher/admin, both of which set user.SchoolID the
// same way) even though it is - like them - a school-level role, not
// platform_owner's cross-tenant tier; that gap failed closed
// ("unsupported user role") rather than leaking anything, but meant a
// super_admin couldn't check their own school's subscription at all.
func (s *AuthService) CheckSubscription(userID string) (*dto.SubscriptionCheckResponse, error) {
	// Get user
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Determine school ID based on role
	var schoolID string
	switch user.Role {
	case models.RoleStudent:
		// Query students table using existing student repo
		student, err := s.studentRepo.FindByUserID(userID)
		if err != nil {
			return nil, errors.New("student record not found")
		}
		schoolID = student.SchoolID

	case models.RoleTeacher, models.RoleAdmin, models.RoleSuperAdmin:
		if user.SchoolID == nil {
			return nil, errors.New("user not associated with any school")
		}
		schoolID = *user.SchoolID

	case models.RoleParent:
		// Get student(s) associated with parent using existing parent repo
		students, err := s.parentStudentRepo.FindStudentsByParentID(userID)
		if err != nil || len(students) == 0 {
			return nil, errors.New("parent not associated with any students")
		}
		// Get school from first student
		schoolID = students[0].SchoolID

	default:
		return nil, errors.New("unsupported user role")
	}

	if schoolID == "" {
		return nil, errors.New("user not associated with any school")
	}

	// Query subscriptions table for this school_id
	subscription, err := s.subscriptionRepo.FindCurrentBySchool(schoolID)
	if err != nil {
		return &dto.SubscriptionCheckResponse{
			HasSubscription: false,
			IsActive:        false,
			SchoolID:        schoolID,
			Message:         "No active subscription found for this school",
		}, nil
	}

	// Check if subscription is active and not expired
	isActive := subscription.Status == models.SubStatusActive &&
		subscription.EndDate.After(time.Now())

	// Build response
	return &dto.SubscriptionCheckResponse{
		HasSubscription: true,
		IsActive:        isActive,
		SchoolID:        schoolID,
		Subscription: &dto.SubscriptionDetail{
			ID:       subscription.ID,
			Tier:     string(subscription.Tier),
			EndDate:  subscription.EndDate,
			Status:   string(subscription.Status),
			Features: subscription.Features,
		},
		Message: func() string {
			if isActive {
				return "Access granted"
			}
			return "Subscription is inactive or expired"
		}(),
	}, nil
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



// ============================================
// GET AUTH STATUS - For Middleware
// ============================================

func (s *AuthService) GetAuthStatus(userID string) (*dto.AuthStatusResponse, error) {
    user, err := s.repo.FindUserByID(userID)
    if err != nil {
        return nil, errors.New("user not found")
    }

    emailStr := ""
    if user.Email != nil {
        emailStr = *user.Email
    }

    response := &dto.AuthStatusResponse{
        Authenticated:  true,
        UserID:         user.ID,
        Username:       user.Username,
        Email:          emailStr,
        Role:           string(user.Role),
        Status:         string(user.Status),
        EmailVerified:  user.EmailVerified,
        HasSchool:      user.SchoolID != nil,
        DashboardRoute: s.getDashboardRoute(string(user.Role)),
    }

    if user.SchoolID != nil {
        school, err := s.schoolRepo.FindByID(*user.SchoolID)
        if err == nil {
            response.SchoolID = &school.ID
            response.SchoolStatus = string(school.Status)
            response.SchoolName = school.Name
        }

        // ✅ ONLY ADMIN sees subscription
        if user.Role == models.RoleAdmin {
            sub, err := s.subscriptionRepo.FindCurrentBySchool(*user.SchoolID)
            if err == nil && sub != nil {
                response.HasSubscription = true
                response.SubscriptionStatus = string(sub.Status)
                response.SubscriptionTier = string(sub.Tier)
                response.SubscriptionEndsAt = &sub.EndDate

                daysRemaining := int(sub.EndDate.Sub(time.Now()).Hours() / 24)
                if daysRemaining < 0 {
                    daysRemaining = 0
                }
                response.DaysRemaining = daysRemaining
            }
        }
    }

    return response, nil
}

func (s *AuthService) getDashboardRoute(role string) string {
    switch role {
    case "super_admin":
        return "/dashboard/super-admin"
    case "admin":
        return "/dashboard/admin"
    case "teacher":
        return "/dashboard/teacher"
    case "student":
        return "/dashboard/student"
    case "parent":
        return "/dashboard/parent"
    default:
        return "/dashboard"
    }
}


// package service

// import (
// 	"crypto/rand"
// 	"errors"
// 	"fmt"
// 	"time"

// 	"cbt-api/internal/auth/dto"
// 	"cbt-api/internal/auth/repository"
// 	"cbt-api/internal/models"
// 	parentService "cbt-api/internal/parent/service"
// 	parentStudentRepo "cbt-api/internal/parent_student/repository"
// 	schoolRepo "cbt-api/internal/school/repository"
// 	studentRepo "cbt-api/internal/student/repository"
// 	subRepo "cbt-api/internal/subscription/repository"
// 	"cbt-api/pkg/email"
// 	"cbt-api/pkg/utils"

// 	"github.com/pquerna/otp/totp"
// 	"github.com/google/uuid"
// 	"go.uber.org/zap"
// )

// type AuthService struct {
// 	repo             *repository.AuthRepository
// 	emailService     *email.EmailService
// 	parentService    *parentService.ParentService
// 	schoolRepo       *schoolRepo.SchoolRepository
// 	subscriptionRepo *subRepo.SubscriptionRepository
// 	logger           *zap.Logger
// }

// func NewAuthService(
// 	repo *repository.AuthRepository,
// 	parentService *parentService.ParentService,
// 	schoolRepo *schoolRepo.SchoolRepository,
// 	subscriptionRepo *subRepo.SubscriptionRepository,
// 	logger *zap.Logger,
// ) *AuthService {
// 	return &AuthService{
// 		repo:             repo,
// 		emailService:     email.NewEmailService(),
// 		parentService:    parentService,
// 		schoolRepo:       schoolRepo,
// 		subscriptionRepo: subscriptionRepo,
// 		logger:           logger,
// 	}
// }

// func (s *AuthService) generateUniqueUsername(base string) string {
// 	username := base
// 	counter := 0
// 	for {
// 		_, err := s.repo.FindUserByUsername(username)
// 		if err != nil {
// 			break
// 		}
// 		counter++
// 		username = fmt.Sprintf("%s%d", base, counter)
// 	}
// 	return username
// }

// // Register with role, username, parent linking, and school code validation
// func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.LoginResponse, error) {
// 	// Validate school code for non-student roles
// 	var schoolID *string
// 	if req.Role != "student" {
// 		if req.SchoolCode == "" {
// 			return nil, errors.New("school_code is required for teacher, admin, and parent roles")
// 		}

// 		school, err := s.schoolRepo.FindByCode(req.SchoolCode)
// 		if err != nil {
// 			return nil, errors.New("invalid school code. Please check and try again")
// 		}
// 		schoolID = &school.ID
// 	}

// 	if req.Role == "parent" && req.AdmissionNumber == "" {
// 		return nil, errors.New("admission_number is required for parent registration")
// 	}
// 	if req.Role != "student" && req.Email == "" {
// 		return nil, errors.New("email is required for non-student roles")
// 	}
// 	if req.Role == "student" && req.Username == "" {
// 		base := req.FirstName + req.LastName
// 		if len(base) > 20 {
// 			base = base[:20]
// 		}
// 		req.Username = s.generateUniqueUsername(base)
// 	}
// 	if req.Username != "" {
// 		existing, _ := s.repo.FindUserByUsername(req.Username)
// 		if existing != nil {
// 			return nil, errors.New("username already taken")
// 		}
// 	}
// 	if req.Email != "" {
// 		existing, _ := s.repo.FindUserByEmail(req.Email)
// 		if existing != nil {
// 			return nil, errors.New("email already registered")
// 		}
// 	}

// 	hashedPassword, err := utils.HashPassword(req.Password)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var role models.UserRole
// 	switch req.Role {
// 	case "admin":
// 		role = models.RoleAdmin
// 	case "teacher":
// 		role = models.RoleTeacher
// 	case "parent":
// 		role = models.RoleParent
// 	default:
// 		role = models.RoleStudent
// 	}
// 	var emailPtr *string
// 	if req.Email != "" {
// 		emailPtr = &req.Email
// 	}
// 	user := &models.User{
// 		Username:      req.Username,
// 		Email:         emailPtr,
// 		Password:      hashedPassword,
// 		FirstName:     req.FirstName,
// 		LastName:      req.LastName,
// 		PhoneNumber:   req.PhoneNumber,
// 		Role:          role,
// 		Status:        models.StatusPending,
// 		IsActive:      true,
// 		EmailVerified: false,
// 		SchoolID:      schoolID, // Set school ID for non-student roles
// 	}
// 	if err := s.repo.CreateUser(user); err != nil {
// 		return nil, err
// 	}
// 	if req.Email != "" {
// 		go s.SendVerificationOTP(req.Email, "email_verification")
// 	}
// 	if role == models.RoleParent && s.parentService != nil && req.AdmissionNumber != "" {
// 		if err := s.parentService.AutoLinkParentOnRegister(user.ID, req.AdmissionNumber); err != nil {
// 			s.logger.Error("parent auto‑link failed", zap.String("parent_id", user.ID), zap.Error(err))
// 		}
// 	}

// 	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session := &models.UserSession{
// 		ID:           uuid.New().String(),
// 		UserID:       user.ID,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
// 	}
// 	_ = s.repo.CreateSession(session)

// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	return &dto.LoginResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 		User: dto.UserDTO{
// 			ID:               user.ID,
// 			Username:         user.Username,
// 			Email:            emailStr,
// 			FirstName:        user.FirstName,
// 			LastName:         user.LastName,
// 			PhoneNumber:      user.PhoneNumber,
// 			Role:             string(user.Role),
// 			Status:           string(user.Status),
// 			EmailVerified:    user.EmailVerified,
// 			TwoFactorEnabled: user.TwoFactorEnabled,
// 			SchoolID:         user.SchoolID,
// 			CreatedAt:        user.CreatedAt,
// 		},
// 	}, nil
// }

// // Login with username or email
// func (s *AuthService) Login(req *dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
// 	var user *models.User
// 	var err error
// 	if req.Username != "" {
// 		user, err = s.repo.FindUserByUsername(req.Username)
// 	} else if req.Email != "" {
// 		user, err = s.repo.FindUserByEmail(req.Email)
// 	} else {
// 		return nil, errors.New("username or email required")
// 	}
// 	if err != nil {
// 		return nil, errors.New("invalid credentials")
// 	}
// 	if !utils.CheckPasswordHash(req.Password, user.Password) {
// 		return nil, errors.New("invalid credentials")
// 	}
// 	if !user.IsActive {
// 		return nil, errors.New("account is deactivated")
// 	}
// 	if user.TwoFactorEnabled {
// 		emailStr := ""
// 		if user.Email != nil {
// 			emailStr = *user.Email
// 		}
// 		return &dto.LoginResponse{
// 			TokenType: "2FA_REQUIRED",
// 			User: dto.UserDTO{
// 				ID:               user.ID,
// 				Username:         user.Username,
// 				Email:            emailStr,
// 				FirstName:        user.FirstName,
// 				LastName:         user.LastName,
// 				TwoFactorEnabled: true,
// 				SchoolID:         user.SchoolID,
// 			},
// 		}, nil
// 	}
// 	return s.completeLogin(user, clientIP, userAgent)
// }

// func (s *AuthService) completeLogin(user *models.User, clientIP, userAgent string) (*dto.LoginResponse, error) {
// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session := &models.UserSession{
// 		ID:           uuid.New().String(),
// 		UserID:       user.ID,
// 		RefreshToken: refreshToken,
// 		UserAgent:    userAgent,
// 		ClientIP:     clientIP,
// 		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
// 	}
// 	_ = s.repo.CreateSession(session)
// 	_ = s.repo.UpdateLastLogin(user.ID)
// 	return &dto.LoginResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 		User: dto.UserDTO{
// 			ID:               user.ID,
// 			Username:         user.Username,
// 			Email:            emailStr,
// 			FirstName:        user.FirstName,
// 			LastName:         user.LastName,
// 			PhoneNumber:      user.PhoneNumber,
// 			Role:             string(user.Role),
// 			Status:           string(user.Status),
// 			EmailVerified:    user.EmailVerified,
// 			TwoFactorEnabled: user.TwoFactorEnabled,
// 			SchoolID:         user.SchoolID,
// 			CreatedAt:        user.CreatedAt,
// 		},
// 	}, nil
// }

// // Verify2FALogin verifies 2FA code during login
// func (s *AuthService) Verify2FALogin(email, code, clientIP, userAgent string) (*dto.LoginResponse, error) {
// 	user, err := s.repo.FindUserByEmail(email)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	// Verify TOTP code
// 	valid := totp.Validate(code, user.TwoFactorSecret)
// 	if !valid {
// 		return nil, errors.New("invalid 2FA code")
// 	}
// 	return s.completeLogin(user, clientIP, userAgent)
// }

// func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
// 	session, err := s.repo.FindSessionByRefreshToken(refreshToken)
// 	if err != nil {
// 		return nil, errors.New("invalid refresh token")
// 	}
// 	user, err := s.repo.FindUserByID(session.UserID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	accessToken, newRefreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session.RefreshToken = newRefreshToken
// 	session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
// 	_ = s.repo.UpdateUser(user)
// 	return &dto.TokenResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: newRefreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 	}, nil
// }

// func (s *AuthService) Logout(userID, sessionID string) error {
// 	return s.repo.DeleteSession(sessionID)
// }

// func (s *AuthService) LogoutAllDevices(userID, currentSessionID string) error {
// 	return s.repo.DeleteAllUserSessions(userID, currentSessionID)
// }

// func (s *AuthService) SendVerificationOTP(email, otpType string) error {
// 	user, err := s.repo.FindUserByEmail(email)
// 	if err != nil {
// 		return nil
// 	}
// 	s.repo.InvalidateUserOTPs(user.ID, otpType)
// 	code := generateOTP(6)
// 	hashedCode, _ := utils.HashPassword(code)
// 	otp := &models.OTP{
// 		UserID:    user.ID,
// 		Code:      hashedCode,
// 		Type:      otpType,
// 		ExpiresAt: time.Now().Add(10 * time.Minute),
// 	}
// 	if err := s.repo.CreateOTP(otp); err != nil {
// 		return err
// 	}
// 	go s.sendOTPEmail(email, code, otpType)
// 	return nil
// }

// func (s *AuthService) VerifyOTP(email, code, otpType string) error {
// 	otp, err := s.repo.FindValidOTP(email, code, otpType)
// 	if err != nil {
// 		return errors.New("invalid or expired code")
// 	}
// 	user, err := s.repo.FindUserByEmail(email)
// 	if err != nil {
// 		return err
// 	}
// 	if !utils.CheckPasswordHash(code, otp.Code) {
// 		return errors.New("invalid code")
// 	}
// 	if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
// 		return err
// 	}
// 	switch otpType {
// 	case "email_verification":
// 		return s.repo.VerifyEmail(user.ID)
// 	}
// 	return nil
// }

// func (s *AuthService) ResetPassword(req *dto.ResetPasswordRequest) error {
// 	otp, err := s.repo.FindValidOTP(req.Email, req.Code, "password_reset")
// 	if err != nil {
// 		return errors.New("invalid or expired code")
// 	}
// 	user, err := s.repo.FindUserByEmail(req.Email)
// 	if err != nil {
// 		return err
// 	}
// 	if !utils.CheckPasswordHash(req.Code, otp.Code) {
// 		return errors.New("invalid code")
// 	}
// 	s.repo.MarkOTPAsUsed(otp.ID)
// 	hashedPassword, err := utils.HashPassword(req.NewPassword)
// 	if err != nil {
// 		return err
// 	}
// 	if err := s.repo.UpdatePassword(user.ID, hashedPassword); err != nil {
// 		return err
// 	}
// 	s.repo.DeleteAllUserSessions(user.ID, "")
// 	return nil
// }

// func (s *AuthService) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
// 		return errors.New("current password is incorrect")
// 	}
// 	hashedPassword, err := utils.HashPassword(req.NewPassword)
// 	if err != nil {
// 		return err
// 	}
// 	return s.repo.UpdatePassword(userID, hashedPassword)
// }

// func (s *AuthService) Generate2FASecret(userID string) (*dto.TwoFactorResponse, error) {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	accountName := user.Username
// 	if user.Email != nil {
// 		accountName = *user.Email
// 	}
// 	key, err := totp.Generate(totp.GenerateOpts{
// 		Issuer:      "CBT-API",
// 		AccountName: accountName,
// 		Period:      30,
// 		Digits:      6,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err := s.repo.UpdateTwoFactorSecret(userID, key.Secret()); err != nil {
// 		return nil, err
// 	}
// 	return &dto.TwoFactorResponse{
// 		Secret:    key.Secret(),
// 		QRCodeURL: key.URL(),
// 		Enabled:   false,
// 		Message:   "Scan QR code with Google Authenticator",
// 	}, nil
// }

// func (s *AuthService) Enable2FA(userID, code string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !totp.Validate(code, user.TwoFactorSecret) {
// 		return errors.New("invalid 2FA code")
// 	}
// 	return s.repo.EnableTwoFactor(userID, user.TwoFactorSecret)
// }

// func (s *AuthService) Disable2FA(userID, code string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !user.TwoFactorEnabled {
// 		return errors.New("2FA is not enabled")
// 	}
// 	if !totp.Validate(code, user.TwoFactorSecret) {
// 		return errors.New("invalid 2FA code")
// 	}
// 	return s.repo.DisableTwoFactor(userID)
// }

// func (s *AuthService) GetUserSessions(userID, currentSessionID string) ([]dto.SessionDTO, error) {
// 	sessions, err := s.repo.FindSessionsByUserID(userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var sessionDTOs []dto.SessionDTO
// 	for _, session := range sessions {
// 		sessionDTOs = append(sessionDTOs, dto.SessionDTO{
// 			ID:        session.ID,
// 			UserAgent: session.UserAgent,
// 			ClientIP:  session.ClientIP,
// 			CreatedAt: session.CreatedAt,
// 			ExpiresAt: session.ExpiresAt,
// 			IsCurrent: session.ID == currentSessionID,
// 		})
// 	}
// 	return sessionDTOs, nil
// }

// func (s *AuthService) RevokeSession(userID, sessionID string) error {
// 	return s.repo.DeleteSession(sessionID)
// }

// func (s *AuthService) GetProfile(userID string) (*dto.UserDTO, error) {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	return &dto.UserDTO{
// 		ID:               user.ID,
// 		Username:         user.Username,
// 		Email:            emailStr,
// 		FirstName:        user.FirstName,
// 		LastName:         user.LastName,
// 		PhoneNumber:      user.PhoneNumber,
// 		Role:             string(user.Role),
// 		Status:           string(user.Status),
// 		EmailVerified:    user.EmailVerified,
// 		TwoFactorEnabled: user.TwoFactorEnabled,
// 		SchoolID:         user.SchoolID,
// 		CreatedAt:        user.CreatedAt,
// 	}, nil
// }

// func (s *AuthService) UpdateProfile(userID, firstName, lastName, phoneNumber string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if firstName != "" {
// 		user.FirstName = firstName
// 	}
// 	if lastName != "" {
// 		user.LastName = lastName
// 	}
// 	if phoneNumber != "" {
// 		user.PhoneNumber = phoneNumber
// 	}
// 	return s.repo.UpdateUser(user)
// }

// func (s *AuthService) DeleteAccount(userID string) error {
// 	s.repo.DeleteAllUserSessions(userID, "")
// 	return s.repo.SoftDeleteUser(userID)
// }

// func (s *AuthService) generateTokens(userID, identifier, role string) (string, string, error) {
// 	accessToken, err := utils.GenerateJWT(userID, identifier, role, 24*time.Hour)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	refreshToken, err := utils.GenerateJWT(userID, identifier, role, 7*24*time.Hour)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	return accessToken, refreshToken, nil
// }

// func (s *AuthService) sendOTPEmail(emailAddr, code, otpType string) {
// 	user, err := s.repo.FindUserByEmail(emailAddr)
// 	if err != nil {
// 		fmt.Printf("Failed to get user for email: %v\n", err)
// 		return
// 	}
// 	var sendErr error
// 	if otpType == "email_verification" {
// 		sendErr = s.emailService.SendWelcomeEmail(emailAddr, user.FirstName, code)
// 	} else {
// 		sendErr = s.emailService.SendPasswordResetEmail(emailAddr, user.FirstName, code)
// 	}
// 	if sendErr != nil {
// 		fmt.Printf("Failed to send email: %v\n", sendErr)
// 	} else {
// 		fmt.Printf("Email sent successfully to %s\n", emailAddr)
// 	}
// }

// // ============================================
// // ADDED: SUBSCRIPTION CHECK METHOD
// // ============================================

// // CheckSubscription checks the subscription status for a user's school
// func (s *AuthService) CheckSubscription(userID string) (*dto.SubscriptionCheckResponse, error) {
// 	// Get user
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}

// 	// Determine school ID based on role
// 	var schoolID string
// 	switch user.Role {
// 	case models.RoleStudent:
// 		// Query students table
// 		studentRepoInstance := studentRepo.NewStudentRepository(s.repo.GetDB())
// 		student, err := studentRepoInstance.FindByUserID(userID)
// 		if err != nil {
// 			return nil, errors.New("student record not found")
// 		}
// 		schoolID = student.SchoolID

// 	case models.RoleTeacher, models.RoleAdmin:
// 		if user.SchoolID == nil {
// 			return nil, errors.New("user not associated with any school")
// 		}
// 		schoolID = *user.SchoolID

// 	case models.RoleParent:
// 		// Get student(s) associated with parent
// 		parentStudentRepoInstance := parentStudentRepo.NewParentStudentRepository(s.repo.GetDB())
// 		students, err := parentStudentRepoInstance.FindStudentsByParentID(userID)
// 		if err != nil || len(students) == 0 {
// 			return nil, errors.New("parent not associated with any students")
// 		}
// 		// Get school from first student
// 		schoolID = students[0].SchoolID

// 	default:
// 		return nil, errors.New("unsupported user role")
// 	}

// 	if schoolID == "" {
// 		return nil, errors.New("user not associated with any school")
// 	}

// 	// Query subscriptions table for this school_id
// 	subscription, err := s.subscriptionRepo.FindCurrentBySchool(schoolID)
// 	if err != nil {
// 		return &dto.SubscriptionCheckResponse{
// 			HasSubscription: false,
// 			IsActive:        false,
// 			SchoolID:        schoolID,
// 			Message:         "No active subscription found for this school",
// 		}, nil
// 	}

// 	// Check if subscription is active and not expired
// 	isActive := subscription.Status == models.SubStatusActive &&
// 		subscription.EndDate.After(time.Now())

// 	// Build response
// 	return &dto.SubscriptionCheckResponse{
// 		HasSubscription: true,
// 		IsActive:        isActive,
// 		SchoolID:        schoolID,
// 		Subscription: &dto.SubscriptionDetail{
// 			ID:      subscription.ID,
// 			Tier:    string(subscription.Tier),
// 			EndDate: subscription.EndDate,
// 			Status:  string(subscription.Status),
// 			Features: subscription.Features,
// 		},
// 		Message: func() string {
// 			if isActive {
// 				return "Access granted"
// 			}
// 			return "Subscription is inactive or expired"
// 		}(),
// 	}, nil
// }

// func generateOTP(length int) string {
// 	const charset = "0123456789"
// 	otp := make([]byte, length)
// 	rand.Read(otp)
// 	for i := range otp {
// 		otp[i] = charset[int(otp[i])%len(charset)]
// 	}
// 	return string(otp)
// }




// package service

// import (
// 	"crypto/rand"
// 	"errors"
// 	"fmt"
// 	"time"

// 	"cbt-api/internal/auth/dto"
// 	"cbt-api/internal/auth/repository"
// 	"cbt-api/internal/models"
// 	parentService "cbt-api/internal/parent/service"
// 	"cbt-api/pkg/email"
// 	"cbt-api/pkg/utils"

// 	"github.com/pquerna/otp/totp"
// 	"github.com/google/uuid"
// 	"go.uber.org/zap"
// )

// type AuthService struct {
// 	repo          *repository.AuthRepository
// 	emailService  *email.EmailService
// 	parentService *parentService.ParentService
// 	schoolRepo       *schoolRepo.SchoolRepository      // ADDED
// 	subscriptionRepo *subRepo.SubscriptionRepository   // ADDED
// 	logger        *zap.Logger
// }

// func NewAuthService(
// 	repo *repository.AuthRepository,
// 	parentService *parentService.ParentService,
// 	schoolRepo *schoolRepo.SchoolRepository,           // ADDED
// 	subscriptionRepo *subRepo.SubscriptionRepository,  // ADDED
// 	logger *zap.Logger,
// ) *AuthService {
// 	return &AuthService{
// 		repo:          repo,
// 		emailService:  email.NewEmailService(),
// 		parentService: parentService,
// 		logger:        logger,
// 	}
// }

// func (s *AuthService) generateUniqueUsername(base string) string {
// 	username := base
// 	counter := 0
// 	for {
// 		_, err := s.repo.FindUserByUsername(username)
// 		if err != nil {
// 			break
// 		}
// 		counter++
// 		username = fmt.Sprintf("%s%d", base, counter)
// 	}
// 	return username
// }

// // Register with role, username, parent linking
// func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.LoginResponse, error) {
// 	if req.Role == "parent" && req.AdmissionNumber == "" {
// 		return nil, errors.New("admission_number is required for parent registration")
// 	}
// 	if req.Role != "student" && req.Email == "" {
// 		return nil, errors.New("email is required for non-student roles")
// 	}
// 	if req.Role == "student" && req.Username == "" {
// 		base := req.FirstName + req.LastName
// 		if len(base) > 20 {
// 			base = base[:20]
// 		}
// 		req.Username = s.generateUniqueUsername(base)
// 	}
// 	if req.Username != "" {
// 		existing, _ := s.repo.FindUserByUsername(req.Username)
// 		if existing != nil {
// 			return nil, errors.New("username already taken")
// 		}
// 	}
// 	if req.Email != "" {
// 		existing, _ := s.repo.FindUserByEmail(req.Email)
// 		if existing != nil {
// 			return nil, errors.New("email already registered")
// 		}
// 	}

// 	hashedPassword, err := utils.HashPassword(req.Password)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var role models.UserRole
// 	switch req.Role {
// 	case "admin":
// 		role = models.RoleAdmin
// 	case "teacher":
// 		role = models.RoleTeacher
// 	case "parent":
// 		role = models.RoleParent
// 	default:
// 		role = models.RoleStudent
// 	}
// 	var emailPtr *string
// 	if req.Email != "" {
// 		emailPtr = &req.Email
// 	}
// 	user := &models.User{
// 		Username:      req.Username,
// 		Email:         emailPtr,
// 		Password:      hashedPassword,
// 		FirstName:     req.FirstName,
// 		LastName:      req.LastName,
// 		PhoneNumber:   req.PhoneNumber,
// 		Role:          role,
// 		Status:        models.StatusPending,
// 		IsActive:      true,
// 		EmailVerified: false,
// 	}
// 	if err := s.repo.CreateUser(user); err != nil {
// 		return nil, err
// 	}
// 	if req.Email != "" {
// 		go s.SendVerificationOTP(req.Email, "email_verification")
// 	}
// 	if role == models.RoleParent && s.parentService != nil && req.AdmissionNumber != "" {
// 		if err := s.parentService.AutoLinkParentOnRegister(user.ID, req.AdmissionNumber); err != nil {
// 			s.logger.Error("parent auto‑link failed", zap.String("parent_id", user.ID), zap.Error(err))
// 		}
// 	}

// 	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session := &models.UserSession{
// 		ID:           uuid.New().String(),
// 		UserID:       user.ID,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
// 	}
// 	_ = s.repo.CreateSession(session)

// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	return &dto.LoginResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 		User: dto.UserDTO{
// 			ID:               user.ID,
// 			Username:         user.Username,
// 			Email:            emailStr,
// 			FirstName:        user.FirstName,
// 			LastName:         user.LastName,
// 			PhoneNumber:      user.PhoneNumber,
// 			Role:             string(user.Role),
// 			Status:           string(user.Status),
// 			EmailVerified:    user.EmailVerified,
// 			TwoFactorEnabled: user.TwoFactorEnabled,
// 			CreatedAt:        user.CreatedAt,
// 		},
// 	}, nil
// }

// // Login with username or email
// func (s *AuthService) Login(req *dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
// 	var user *models.User
// 	var err error
// 	if req.Username != "" {
// 		user, err = s.repo.FindUserByUsername(req.Username)
// 	} else if req.Email != "" {
// 		user, err = s.repo.FindUserByEmail(req.Email)
// 	} else {
// 		return nil, errors.New("username or email required")
// 	}
// 	if err != nil {
// 		return nil, errors.New("invalid credentials")
// 	}
// 	if !utils.CheckPasswordHash(req.Password, user.Password) {
// 		return nil, errors.New("invalid credentials")
// 	}
// 	if !user.IsActive {
// 		return nil, errors.New("account is deactivated")
// 	}
// 	if user.TwoFactorEnabled {
// 		emailStr := ""
// 		if user.Email != nil {
// 			emailStr = *user.Email
// 		}
// 		return &dto.LoginResponse{
// 			TokenType: "2FA_REQUIRED",
// 			User: dto.UserDTO{
// 				ID:               user.ID,
// 				Username:         user.Username,
// 				Email:            emailStr,
// 				FirstName:        user.FirstName,
// 				LastName:         user.LastName,
// 				TwoFactorEnabled: true,
// 			},
// 		}, nil
// 	}
// 	return s.completeLogin(user, clientIP, userAgent)
// }

// func (s *AuthService) completeLogin(user *models.User, clientIP, userAgent string) (*dto.LoginResponse, error) {
// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	accessToken, refreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session := &models.UserSession{
// 		ID:           uuid.New().String(),
// 		UserID:       user.ID,
// 		RefreshToken: refreshToken,
// 		UserAgent:    userAgent,
// 		ClientIP:     clientIP,
// 		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
// 	}
// 	_ = s.repo.CreateSession(session)
// 	_ = s.repo.UpdateLastLogin(user.ID)
// 	return &dto.LoginResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 		User: dto.UserDTO{
// 			ID:               user.ID,
// 			Username:         user.Username,
// 			Email:            emailStr,
// 			FirstName:        user.FirstName,
// 			LastName:         user.LastName,
// 			PhoneNumber:      user.PhoneNumber,
// 			Role:             string(user.Role),
// 			Status:           string(user.Status),
// 			EmailVerified:    user.EmailVerified,
// 			TwoFactorEnabled: user.TwoFactorEnabled,
// 			CreatedAt:        user.CreatedAt,
// 		},
// 	}, nil
// }

// // Verify2FALogin verifies 2FA code during login
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


// func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
// 	session, err := s.repo.FindSessionByRefreshToken(refreshToken)
// 	if err != nil {
// 		return nil, errors.New("invalid refresh token")
// 	}
// 	user, err := s.repo.FindUserByID(session.UserID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	accessToken, newRefreshToken, err := s.generateTokens(user.ID, user.Username, string(user.Role))
// 	if err != nil {
// 		return nil, err
// 	}
// 	session.RefreshToken = newRefreshToken
// 	session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
// 	_ = s.repo.UpdateUser(user)
// 	return &dto.TokenResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: newRefreshToken,
// 		ExpiresAt:    time.Now().Add(24 * time.Hour),
// 		TokenType:    "Bearer",
// 	}, nil
// }

// func (s *AuthService) Logout(userID, sessionID string) error {
// 	return s.repo.DeleteSession(sessionID)
// }

// func (s *AuthService) LogoutAllDevices(userID, currentSessionID string) error {
// 	return s.repo.DeleteAllUserSessions(userID, currentSessionID)
// }

// func (s *AuthService) SendVerificationOTP(email, otpType string) error {
// 	user, err := s.repo.FindUserByEmail(email)
// 	if err != nil {
// 		return nil
// 	}
// 	s.repo.InvalidateUserOTPs(user.ID, otpType)
// 	code := generateOTP(6)
// 	hashedCode, _ := utils.HashPassword(code)
// 	otp := &models.OTP{
// 		UserID:    user.ID,
// 		Code:      hashedCode,
// 		Type:      otpType,
// 		ExpiresAt: time.Now().Add(10 * time.Minute),
// 	}
// 	if err := s.repo.CreateOTP(otp); err != nil {
// 		return err
// 	}
// 	go s.sendOTPEmail(email, code, otpType)
// 	return nil
// }

// func (s *AuthService) VerifyOTP(email, code, otpType string) error {
// 	otp, err := s.repo.FindValidOTP(email, code, otpType)
// 	if err != nil {
// 		return errors.New("invalid or expired code")
// 	}
// 	user, err := s.repo.FindUserByEmail(email)
// 	if err != nil {
// 		return err
// 	}
// 	if !utils.CheckPasswordHash(code, otp.Code) {
// 		return errors.New("invalid code")
// 	}
// 	if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
// 		return err
// 	}
// 	switch otpType {
// 	case "email_verification":
// 		return s.repo.VerifyEmail(user.ID)
// 	}
// 	return nil
// }

// func (s *AuthService) ResetPassword(req *dto.ResetPasswordRequest) error {
// 	otp, err := s.repo.FindValidOTP(req.Email, req.Code, "password_reset")
// 	if err != nil {
// 		return errors.New("invalid or expired code")
// 	}
// 	user, err := s.repo.FindUserByEmail(req.Email)
// 	if err != nil {
// 		return err
// 	}
// 	if !utils.CheckPasswordHash(req.Code, otp.Code) {
// 		return errors.New("invalid code")
// 	}
// 	s.repo.MarkOTPAsUsed(otp.ID)
// 	hashedPassword, err := utils.HashPassword(req.NewPassword)
// 	if err != nil {
// 		return err
// 	}
// 	if err := s.repo.UpdatePassword(user.ID, hashedPassword); err != nil {
// 		return err
// 	}
// 	s.repo.DeleteAllUserSessions(user.ID, "")
// 	return nil
// }

// func (s *AuthService) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
// 		return errors.New("current password is incorrect")
// 	}
// 	hashedPassword, err := utils.HashPassword(req.NewPassword)
// 	if err != nil {
// 		return err
// 	}
// 	return s.repo.UpdatePassword(userID, hashedPassword)
// }

// func (s *AuthService) Generate2FASecret(userID string) (*dto.TwoFactorResponse, error) {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	accountName := user.Username
// 	if user.Email != nil {
// 		accountName = *user.Email
// 	}
// 	key, err := totp.Generate(totp.GenerateOpts{
// 		Issuer:      "CBT-API",
// 		AccountName: accountName,
// 		Period:      30,
// 		Digits:      6,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err := s.repo.UpdateTwoFactorSecret(userID, key.Secret()); err != nil {
// 		return nil, err
// 	}
// 	return &dto.TwoFactorResponse{
// 		Secret:    key.Secret(),
// 		QRCodeURL: key.URL(),
// 		Enabled:   false,
// 		Message:   "Scan QR code with Google Authenticator",
// 	}, nil
// }

// func (s *AuthService) Enable2FA(userID, code string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !totp.Validate(code, user.TwoFactorSecret) {
// 		return errors.New("invalid 2FA code")
// 	}
// 	return s.repo.EnableTwoFactor(userID, user.TwoFactorSecret)
// }

// func (s *AuthService) Disable2FA(userID, code string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if !user.TwoFactorEnabled {
// 		return errors.New("2FA is not enabled")
// 	}
// 	if !totp.Validate(code, user.TwoFactorSecret) {
// 		return errors.New("invalid 2FA code")
// 	}
// 	return s.repo.DisableTwoFactor(userID)
// }

// func (s *AuthService) GetUserSessions(userID, currentSessionID string) ([]dto.SessionDTO, error) {
// 	sessions, err := s.repo.FindSessionsByUserID(userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var sessionDTOs []dto.SessionDTO
// 	for _, session := range sessions {
// 		sessionDTOs = append(sessionDTOs, dto.SessionDTO{
// 			ID:        session.ID,
// 			UserAgent: session.UserAgent,
// 			ClientIP:  session.ClientIP,
// 			CreatedAt: session.CreatedAt,
// 			ExpiresAt: session.ExpiresAt,
// 			IsCurrent: session.ID == currentSessionID,
// 		})
// 	}
// 	return sessionDTOs, nil
// }

// func (s *AuthService) RevokeSession(userID, sessionID string) error {
// 	return s.repo.DeleteSession(sessionID)
// }

// func (s *AuthService) GetProfile(userID string) (*dto.UserDTO, error) {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return nil, errors.New("user not found")
// 	}
// 	emailStr := ""
// 	if user.Email != nil {
// 		emailStr = *user.Email
// 	}
// 	return &dto.UserDTO{
// 		ID:               user.ID,
// 		Username:         user.Username,
// 		Email:            emailStr,
// 		FirstName:        user.FirstName,
// 		LastName:         user.LastName,
// 		PhoneNumber:      user.PhoneNumber,
// 		Role:             string(user.Role),
// 		Status:           string(user.Status),
// 		EmailVerified:    user.EmailVerified,
// 		TwoFactorEnabled: user.TwoFactorEnabled,
// 		CreatedAt:        user.CreatedAt,
// 	}, nil
// }

// func (s *AuthService) UpdateProfile(userID, firstName, lastName, phoneNumber string) error {
// 	user, err := s.repo.FindUserByID(userID)
// 	if err != nil {
// 		return errors.New("user not found")
// 	}
// 	if firstName != "" {
// 		user.FirstName = firstName
// 	}
// 	if lastName != "" {
// 		user.LastName = lastName
// 	}
// 	if phoneNumber != "" {
// 		user.PhoneNumber = phoneNumber
// 	}
// 	return s.repo.UpdateUser(user)
// }

// func (s *AuthService) DeleteAccount(userID string) error {
// 	s.repo.DeleteAllUserSessions(userID, "")
// 	return s.repo.SoftDeleteUser(userID)
// }

// func (s *AuthService) generateTokens(userID, identifier, role string) (string, string, error) {
// 	accessToken, err := utils.GenerateJWT(userID, identifier, role, 24*time.Hour)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	refreshToken, err := utils.GenerateJWT(userID, identifier, role, 7*24*time.Hour)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	return accessToken, refreshToken, nil
// }

// func (s *AuthService) sendOTPEmail(emailAddr, code, otpType string) {
// 	user, err := s.repo.FindUserByEmail(emailAddr)
// 	if err != nil {
// 		fmt.Printf("Failed to get user for email: %v\n", err)
// 		return
// 	}
// 	var sendErr error
// 	if otpType == "email_verification" {
// 		sendErr = s.emailService.SendWelcomeEmail(emailAddr, user.FirstName, code)
// 	} else {
// 		sendErr = s.emailService.SendPasswordResetEmail(emailAddr, user.FirstName, code)
// 	}
// 	if sendErr != nil {
// 		fmt.Printf("Failed to send email: %v\n", sendErr)
// 	} else {
// 		fmt.Printf("Email sent successfully to %s\n", emailAddr)
// 	}
// }

// func generateOTP(length int) string {
// 	const charset = "0123456789"
// 	otp := make([]byte, length)
// 	rand.Read(otp)
// 	for i := range otp {
// 		otp[i] = charset[int(otp[i])%len(charset)]
// 	}
// 	return string(otp)
// }

