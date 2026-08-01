package service

import (
	"math/rand"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"cbt-api/internal/models"
	"cbt-api/internal/onboarding/dto"
	onboardingRepo "cbt-api/internal/onboarding/repository"
	subscriptionRepo "cbt-api/internal/subscription/repository"
	"cbt-api/pkg/email"
	"cbt-api/pkg/payment"
	"cbt-api/pkg/utils"
)

// ============================================
// CONSTANTS
// ============================================

const (
	SessionExpiryHours = 24
	OTPLength          = 6
	MaxRetryAttempts   = 3
)

// ============================================
// ONBOARDING SERVICE
// ============================================

type OnboardingService struct {
	subscriptionRepo *subscriptionRepo.SubscriptionRepository
	paymentService   *payment.PaymentService
	emailService     *email.EmailService
	onboardingRepo   *onboardingRepo.OnboardingRepository
	sessionStore     SessionStore
	config           *OnboardingConfig
	logger           *log.Logger
}

type OnboardingConfig struct {
	AppURL              string
	DefaultGateway      models.PaymentGateway
	AllowedGateways     []models.PaymentGateway
	SessionExpiryHours  int
	PaymentExpiryHours  int
	EnableEmail         bool
	EnableWebhooks      bool
	WebhookSecret       string
	RetryAttempts       int
}

// ============================================
// SESSION STORE INTERFACE
// ============================================

type SessionStore interface {
	Create(ctx context.Context, session *OnboardingSession) error
	Get(ctx context.Context, id string) (*OnboardingSession, error)
	GetByEmail(ctx context.Context, email string) (*OnboardingSession, error)
	Update(ctx context.Context, session *OnboardingSession) error
	Delete(ctx context.Context, id string) error
	CleanupExpired(ctx context.Context) error
}

// ============================================
// ONBOARDING SESSION
// ============================================

type OnboardingSession struct {
	ID              string                    `json:"id"`
	Email           string                    `json:"email"`
	UserID          string                    `json:"user_id"`
	SchoolID        string                    `json:"school_id"`
	SubscriptionID  string                    `json:"subscription_id"`
	PaymentIntentID string                    `json:"payment_intent_id"`
	Tier            models.SubscriptionTier   `json:"tier"`
	Interval        models.PaymentInterval    `json:"interval"`
	Gateway         models.PaymentGateway     `json:"gateway"`
	Amount          decimal.Decimal           `json:"amount"`
	Status          string                    `json:"status"`
	Steps           OnboardingSteps           `json:"steps"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	ExpiresAt       time.Time                 `json:"expires_at"`
	Metadata        map[string]interface{}    `json:"metadata"`
}

type OnboardingSteps struct {
	Step1Completed bool `json:"step1_completed"`
	Step2Completed bool `json:"step2_completed"`
	Step3Completed bool `json:"step3_completed"`
	Step4Completed bool `json:"step4_completed"`
	Step5Completed bool `json:"step5_completed"`
	Step6Completed bool `json:"step6_completed"`
}

// ============================================
// IN-MEMORY SESSION STORE - BUILT IN
// ============================================

type InMemorySessionStore struct {
	sessions map[string]*OnboardingSession
	emailIdx map[string]string
	mu       sync.RWMutex
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*OnboardingSession),
		emailIdx: make(map[string]string),
	}
}

func (s *InMemorySessionStore) Create(ctx context.Context, session *OnboardingSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	s.emailIdx[session.Email] = session.ID
	return nil
}

func (s *InMemorySessionStore) Get(ctx context.Context, id string) (*OnboardingSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (s *InMemorySessionStore) GetByEmail(ctx context.Context, email string) (*OnboardingSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.emailIdx[email]
	if !ok {
		return nil, errors.New("session not found")
	}
	return s.sessions[id], nil
}

func (s *InMemorySessionStore) Update(ctx context.Context, session *OnboardingSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.emailIdx, session.Email)
	}
	delete(s.sessions, id)
	return nil
}

func (s *InMemorySessionStore) CleanupExpired(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if session.ExpiresAt.Before(now) {
			delete(s.emailIdx, session.Email)
			delete(s.sessions, id)
		}
	}
	return nil
}

// ============================================
// CONSTRUCTOR - SELF-CONTAINED
// ============================================

func NewOnboardingService(
	subscriptionRepo *subscriptionRepo.SubscriptionRepository,
	paymentService *payment.PaymentService,
	emailService *email.EmailService,
	onboardingRepo *onboardingRepo.OnboardingRepository,
	config *OnboardingConfig,
) *OnboardingService {
	if config == nil {
		config = &OnboardingConfig{
			AppURL:              os.Getenv("APP_URL"),
			DefaultGateway:      models.GatewayPaystack,
			AllowedGateways:     []models.PaymentGateway{models.GatewayPaystack, models.GatewayFlutterwave, models.GatewayStripe},
			SessionExpiryHours:  SessionExpiryHours,
			PaymentExpiryHours:  48,
			EnableEmail:         true,
			EnableWebhooks:      true,
			WebhookSecret:       os.Getenv("WEBHOOK_SECRET"),
			RetryAttempts:       MaxRetryAttempts,
		}
	}

	sessionStore := NewInMemorySessionStore()

	return &OnboardingService{
		subscriptionRepo: subscriptionRepo,
		paymentService:   paymentService,
		emailService:     emailService,
		onboardingRepo:   onboardingRepo,
		sessionStore:     sessionStore,
		config:           config,
		logger:           log.New(os.Stdout, "[ONBOARDING] ", log.LstdFlags|log.Lshortfile),
	}
}

// ============================================
// STEP 1: SELECT PLAN
// ============================================

func (s *OnboardingService) StartOnboarding(ctx context.Context, req *dto.StartOnboardingRequest) (*dto.StartOnboardingResponse, error) {
	s.logger.Printf("[INFO] Step 1: User selects plan - email=%s, tier=%s, interval=%s, gateway=%s",
		req.Email, req.Tier, req.Interval, req.Gateway)

	tier := models.SubscriptionTier(req.Tier)
	interval := models.PaymentInterval(req.Interval)
	amount, exists := models.Pricing[tier][interval]
	if !exists {
		s.logger.Printf("[ERROR] Invalid pricing for tier=%s, interval=%s", req.Tier, req.Interval)
		return nil, fmt.Errorf("invalid pricing for selected tier and interval")
	}

	gateway := models.PaymentGateway(req.Gateway)
	if !s.isGatewayAvailable(gateway) {
		s.logger.Printf("[ERROR] Gateway %s is not available", req.Gateway)
		return nil, fmt.Errorf("gateway %s is not available", req.Gateway)
	}

	existing, _ := s.sessionStore.GetByEmail(ctx, req.Email)
	if existing != nil && existing.Status != "expired" && existing.Status != "completed" {
		s.logger.Printf("[INFO] Found existing session for email=%s, session_id=%s", req.Email, existing.ID)
		return &dto.StartOnboardingResponse{
			SessionID: existing.ID,
			Amount:    existing.Amount,
			Tier:      string(existing.Tier),
			Interval:  string(existing.Interval),
			Gateway:   string(existing.Gateway),
			Message:   "Existing onboarding session found. Please continue.",
			ExpiresAt: existing.ExpiresAt,
		}, nil
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &OnboardingSession{
		ID:       sessionID,
		Email:    req.Email,
		Tier:     tier,
		Interval: interval,
		Gateway:  gateway,
		Amount:   amount,
		Status:   "pending",
		Steps: OnboardingSteps{
			Step1Completed: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(time.Duration(s.config.SessionExpiryHours) * time.Hour),
		Metadata: map[string]interface{}{
			"ip_address": ctx.Value("ip_address"),
			"user_agent": ctx.Value("user_agent"),
		},
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		s.logger.Printf("[ERROR] Failed to create session: %v", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.logger.Printf("[SUCCESS] Session created: %s for email=%s", sessionID, req.Email)

	return &dto.StartOnboardingResponse{
		SessionID: sessionID,
		Amount:    amount,
		Tier:      string(tier),
		Interval:  string(interval),
		Gateway:   string(gateway),
		Message:   "Plan selected. Please create your admin account.",
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// ============================================
// STEP 2: CREATE ADMIN ACCOUNT
// ============================================

func (s *OnboardingService) CreateAdminAccount(ctx context.Context, req *dto.CreateAdminRequest) (*dto.CreateAdminResponse, error) {
	s.logger.Printf("[INFO] Step 2: Creating admin account - session=%s, email=%s", req.SessionID, req.Email)

	session, err := s.getValidSession(ctx, req.SessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return nil, err
	}

	if session.Steps.Step2Completed {
		s.logger.Printf("[WARN] Admin account already created for session=%s", req.SessionID)
		return nil, errors.New("admin account already created")
	}

	if session.Email != req.Email {
		s.logger.Printf("[ERROR] Email mismatch: session=%s, expected=%s, got=%s", req.SessionID, session.Email, req.Email)
		return nil, errors.New("email does not match session")
	}

	existing, _ := s.onboardingRepo.FindOnboardUserByEmail(req.Email)
	if existing != nil {
		s.logger.Printf("[ERROR] Email already registered: %s", req.Email)
		return nil, errors.New("email already registered")
	}

	existingUser, _ := s.onboardingRepo.FindOnboardUserByUsername(req.Username)
	if existingUser != nil {
		s.logger.Printf("[ERROR] Username already taken: %s", req.Username)
		return nil, errors.New("username already taken")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New().String()
	user := &models.User{
		ID:            userID,
		Username:      req.Username,
		Email:         &req.Email,
		Password:      hashedPassword,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		PhoneNumber:   req.PhoneNumber,
		Role:          models.RoleAdmin,
		Status:        models.StatusPending,
		IsActive:      true,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.onboardingRepo.CreateOnboardUser(user); err != nil {
		s.logger.Printf("[ERROR] Failed to create user account: %v", err)
		return nil, fmt.Errorf("failed to create user account: %w", err)
	}

	session.UserID = userID
	session.Steps.Step2Completed = true
	session.UpdatedAt = time.Now()
	s.sessionStore.Update(ctx, session)

	s.logger.Printf("[SUCCESS] Admin account created: user_id=%s, email=%s", userID, req.Email)

	// Send verification OTP
	go s.sendVerificationOTPWithTracking(req.Email, user.FirstName, user.ID)

	return &dto.CreateAdminResponse{
		UserID:   userID,
		Email:    req.Email,
		Username: user.Username,
		Message:  "Admin account created. Please verify your email.",
		NextStep: "verify_email",
	}, nil
}

// ============================================
// STEP 3: VERIFY EMAIL
// ============================================

func (s *OnboardingService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) (*dto.VerifyEmailResponse, error) {
	s.logger.Printf("[INFO] Step 3: Verifying email - session=%s, email=%s, otp=%s", req.SessionID, req.Email, req.OTPCode)

	session, err := s.getValidSession(ctx, req.SessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return nil, err
	}

	if session.Steps.Step3Completed {
		s.logger.Printf("[WARN] Email already verified for session=%s", req.SessionID)
		return nil, errors.New("email already verified")
	}

	if session.Email != req.Email {
		s.logger.Printf("[ERROR] Email mismatch: session=%s, expected=%s, got=%s", req.SessionID, session.Email, req.Email)
		return nil, errors.New("email does not match session")
	}

	// Verify email with OTP code using onboarding repo
	if err := s.onboardingRepo.VerifyOnboardEmailWithOTP(session.UserID, req.OTPCode); err != nil {
		s.logger.Printf("[ERROR] OTP verification failed for user=%s, code=%s: %v", session.UserID, req.OTPCode, err)
		return nil, errors.New("invalid or expired verification code")
	}

	// Get updated user
	user, err := s.onboardingRepo.FindOnboardUserByID(session.UserID)
	if err != nil {
		s.logger.Printf("[ERROR] User not found: %s", session.UserID)
		return nil, errors.New("user not found")
	}

	session.Steps.Step3Completed = true
	session.UpdatedAt = time.Now()
	s.sessionStore.Update(ctx, session)

	s.logger.Printf("[SUCCESS] Email verified successfully: user_id=%s, email=%s", user.ID, req.Email)

	return &dto.VerifyEmailResponse{
		UserID:   user.ID,
		Email:    req.Email,
		Message:  "Email verified successfully.",
		NextStep: "register_school",
	}, nil
}

// ============================================
// STEP 4: REGISTER SCHOOL
// ============================================

func (s *OnboardingService) RegisterSchool(ctx context.Context, req *dto.RegisterSchoolRequest) (*dto.RegisterSchoolResponse, error) {
	s.logger.Printf("[INFO] Step 4: Registering school - session=%s, name=%s", req.SessionID, req.SchoolName)

	session, err := s.getValidSession(ctx, req.SessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return nil, err
	}

	if session.Steps.Step4Completed {
		s.logger.Printf("[WARN] School already registered for session=%s", req.SessionID)
		return nil, errors.New("school already registered")
	}

	user, err := s.onboardingRepo.FindOnboardUserByID(session.UserID)
	if err != nil {
		s.logger.Printf("[ERROR] User not found: %s", session.UserID)
		return nil, errors.New("user not found")
	}

	existing, _ := s.onboardingRepo.FindOnboardSchoolByName(req.SchoolName)
	if existing != nil {
		s.logger.Printf("[ERROR] School name already taken: %s", req.SchoolName)
		return nil, errors.New("school name already taken")
	}

	schoolCode := generateSchoolCode(req.SchoolName)

	school := &models.School{
		ID:          uuid.New().String(),
		Name:        req.SchoolName,
		Code:        schoolCode,
		Address:     req.Address,
		Phone:       req.Phone,
		Email:       req.SchoolEmail,
		Status:      models.SchoolStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.onboardingRepo.CreateOnboardSchool(school); err != nil {
		s.logger.Printf("[ERROR] Failed to create school: %v", err)
		return nil, fmt.Errorf("failed to create school: %w", err)
	}

	user.SchoolID = &school.ID
	user.UpdatedAt = time.Now()
	if err := s.onboardingRepo.UpdateOnboardUser(user); err != nil {
		s.logger.Printf("[ERROR] Failed to update user with school: %v", err)
		return nil, fmt.Errorf("failed to update user with school: %w", err)
	}

	session.SchoolID = school.ID
	session.Steps.Step4Completed = true
	session.UpdatedAt = time.Now()
	s.sessionStore.Update(ctx, session)

	s.logger.Printf("[SUCCESS] School registered: school_id=%s, code=%s, name=%s", school.ID, schoolCode, req.SchoolName)

	// Send school registration email
	go s.sendSchoolRegistrationEmailWithTracking(user, school)

	return &dto.RegisterSchoolResponse{
		SchoolID:   school.ID,
		SchoolCode: schoolCode,
		Message:    "School registered successfully.",
		NextStep:   "review",
	}, nil
}

// ============================================
// STEP 6: CREATE SUBSCRIPTION WITH PAYMENT LINK
// ============================================

func (s *OnboardingService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest) (*dto.CreateSubscriptionResponse, error) {
	s.logger.Printf("[INFO] Step 6: Creating subscription - session=%s", req.SessionID)

	session, err := s.getValidSession(ctx, req.SessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return nil, err
	}

	if session.Steps.Step6Completed {
		s.logger.Printf("[WARN] Subscription already created for session=%s", req.SessionID)
		return nil, errors.New("subscription already created")
	}

	if !session.Steps.Step4Completed {
		s.logger.Printf("[ERROR] School registration not completed for session=%s", req.SessionID)
		return nil, errors.New("must complete school registration first")
	}

	user, err := s.onboardingRepo.FindOnboardUserByID(session.UserID)
	if err != nil {
		s.logger.Printf("[ERROR] User not found: %s", session.UserID)
		return nil, errors.New("user not found")
	}

	school, err := s.onboardingRepo.FindOnboardSchoolByID(session.SchoolID)
	if err != nil {
		s.logger.Printf("[ERROR] School not found: %s", session.SchoolID)
		return nil, errors.New("school not found")
	}

	existing, _ := s.subscriptionRepo.FindCurrentBySchool(school.ID)
	if existing != nil {
		s.logger.Printf("[ERROR] School already has a subscription: %s", school.ID)
		return nil, errors.New("school already has a subscription")
	}

	gateway := session.Gateway
	if gateway == "" {
		gateway = s.config.DefaultGateway
	}

	now := time.Now()
	subscriptionID := uuid.New().String()

	subscription := &models.Subscription{
		ID:              subscriptionID,
		UserID:          user.ID,
		SchoolID:        school.ID,
		Tier:            session.Tier,
		Status:          models.SubStatusPending,
		Gateway:         gateway,
		Amount:          session.Amount,
		Currency:        models.CurrencyNGN,
		PaymentInterval: session.Interval,
		StartDate:       now,
		EndDate:         now.AddDate(1, 0, 0),
		AutoRenew:       true,
		MaxStudents:     models.TierLimits[session.Tier].MaxStudents,
		MaxTeachers:     models.TierLimits[session.Tier].MaxTeachers,
		MaxExams:        models.TierLimits[session.Tier].MaxExams,
		MaxQuestions:    models.TierLimits[session.Tier].MaxQuestions,
		MaxStorageMB:    models.TierLimits[session.Tier].MaxStorageMB,
		Features:        getFeaturesForTier(session.Tier),
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       user.ID,
		Metadata: map[string]interface{}{
			"onboarding":      true,
			"session_id":      session.ID,
			"gateway":         string(gateway),
			"original_amount": session.Amount.String(),
		},
	}

	if err := s.subscriptionRepo.Create(subscription); err != nil {
		s.logger.Printf("[ERROR] Failed to create subscription: %v", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	invoice := &models.Invoice{
		ID:             uuid.New().String(),
		SubscriptionID: subscriptionID,
		SchoolID:       school.ID,
		UserID:         user.ID,
		InvoiceNumber:  generateInvoiceNumber(),
		Amount:         session.Amount,
		Tax:            decimal.Zero,
		Discount:       decimal.Zero,
		Total:          session.Amount,
		Currency:       models.CurrencyNGN,
		Status:         models.InvoicePending,
		DueDate:        now.AddDate(0, 0, 7),
		Items: map[string]interface{}{
			"tier":       string(session.Tier),
			"interval":   string(session.Interval),
			"onboarding": true,
			"gateway":    string(gateway),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.subscriptionRepo.CreateInvoice(invoice); err != nil {
		s.subscriptionRepo.Delete(subscriptionID)
		s.logger.Printf("[ERROR] Failed to create invoice: %v", err)
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	callbackURL := req.CallbackURL
	if callbackURL == "" {
		callbackURL = s.getDefaultCallbackURL(gateway)
	}
	successURL := req.SuccessURL
	if successURL == "" {
		successURL = s.getDefaultSuccessURL(gateway)
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = s.getDefaultCancelURL(gateway)
	}

	paymentReq := &payment.PaymentRequest{
		SchoolID:    school.ID,
		UserID:      user.ID,
		Amount:      session.Amount,
		Currency:    models.CurrencyNGN,
		Email:       *user.Email,
		CallbackURL: callbackURL,
		SuccessURL:  successURL,
		CancelURL:   cancelURL,
		Metadata: map[string]interface{}{
			"subscription_id": subscriptionID,
			"invoice_id":      invoice.ID,
			"onboarding":      true,
			"session_id":      session.ID,
			"tier":            string(session.Tier),
			"interval":        string(session.Interval),
			"gateway":         string(gateway),
		},
	}

	paymentIntent, err := s.paymentService.CreatePayment(ctx, gateway, paymentReq)
	if err != nil {
		s.subscriptionRepo.Delete(subscriptionID)
		s.logger.Printf("[ERROR] Failed to create payment with %s: %v", gateway, err)
		return nil, fmt.Errorf("failed to create payment with %s: %w", gateway, err)
	}

	paymentIntentID := uuid.New().String()
	dbPI := &models.PaymentIntent{
		ID:               paymentIntentID,
		SubscriptionID:   subscriptionID,
		InvoiceID:        invoice.ID,
		SchoolID:         school.ID,
		UserID:           user.ID,
		IdempotencyKey:   fmt.Sprintf("%s_%d", subscriptionID, now.UnixNano()),
		Gateway:          gateway,
		Amount:           session.Amount,
		Currency:         models.CurrencyNGN,
		Reference:        paymentIntent.Reference,
		AuthorizationURL: paymentIntent.AuthorizationURL,
		Status:           models.IntentPending,
		ExpiresAt:        timePtr(now.Add(time.Duration(s.config.PaymentExpiryHours) * time.Hour)),
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata: map[string]interface{}{
			"onboarding": true,
			"session_id": session.ID,
			"gateway":    string(gateway),
		},
	}

	if err := s.subscriptionRepo.CreatePaymentIntent(dbPI); err != nil {
		s.logger.Printf("[ERROR] Failed to create payment intent: %v", err)
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	history := &models.SubscriptionHistory{
		ID:             uuid.New().String(),
		SubscriptionID: subscriptionID,
		SchoolID:       school.ID,
		UserID:         user.ID,
		NewTier:        session.Tier,
		NewStatus:      models.SubStatusPending,
		NewAmount:      session.Amount,
		ChangeReason:   "subscription_created_pending_payment",
		ChangedBy:      user.ID,
		Metadata: map[string]interface{}{
			"onboarding": true,
			"session_id": session.ID,
			"gateway":    string(gateway),
		},
		CreatedAt: now,
	}
	s.subscriptionRepo.CreateHistory(history)

	session.SubscriptionID = subscriptionID
	session.PaymentIntentID = dbPI.ID
	session.Steps.Step6Completed = true
	session.Status = "payment_pending"
	session.UpdatedAt = now
	s.sessionStore.Update(ctx, session)

	s.logger.Printf("[SUCCESS] Subscription created: subscription_id=%s, amount=%s", subscriptionID, session.Amount.String())

	// Send payment link email
	go s.sendPaymentLinkEmailWithTracking(user, school, subscription, invoice, dbPI, gateway)

	return &dto.CreateSubscriptionResponse{
		SubscriptionID:  subscriptionID,
		PaymentIntentID: dbPI.ID,
		PaymentLink:     dbPI.AuthorizationURL,
		Amount:          session.Amount,
		Gateway:         string(gateway),
		Reference:       paymentIntent.Reference,
		Message:         fmt.Sprintf("Subscription created with %s. Please complete payment to activate your account.", string(gateway)),
		NextStep:        "payment",
		ExpiresAt:       *dbPI.ExpiresAt,
	}, nil
}

// ============================================
// GET AVAILABLE GATEWAYS
// ============================================

func (s *OnboardingService) GetAvailableGateways(ctx context.Context) []dto.GatewayInfo {
	available := s.paymentService.GetAvailableGateways()

	gatewayInfoMap := map[models.PaymentGateway]dto.GatewayInfo{
		models.GatewayPaystack: {
			Name:        "Paystack",
			Slug:        "paystack",
			Description: "Paystack - Popular Nigerian payment gateway",
			IsAvailable: false,
			Currency:    "NGN",
		},
		models.GatewayFlutterwave: {
			Name:        "Flutterwave",
			Slug:        "flutterwave",
			Description: "Flutterwave - African payment gateway",
			IsAvailable: false,
			Currency:    "NGN, USD, EUR, GBP",
		},
		models.GatewayStripe: {
			Name:        "Stripe",
			Slug:        "stripe",
			Description: "Stripe - Global payment gateway",
			IsAvailable: false,
			Currency:    "USD, EUR, GBP",
		},
	}

	var result []dto.GatewayInfo
	for _, g := range available {
		if info, ok := gatewayInfoMap[g]; ok {
			info.IsAvailable = true
			result = append(result, info)
		}
	}

	return result
}

// ============================================
// WEBHOOK - STEP 9: ACTIVATE EVERYTHING
// ============================================

// func (s *OnboardingService) ActivateOnboarding(ctx context.Context, reference string, gateway models.PaymentGateway) error {
// 	s.logger.Printf("[INFO] Step 9: Activating onboarding - reference=%s, gateway=%s", reference, gateway)

// 	pi, err := s.subscriptionRepo.FindPaymentIntentByReference(reference)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] Payment intent not found: %s", reference)
// 		return errors.New("payment intent not found")
// 	}

// 	if isOnboarding, ok := pi.Metadata["onboarding"]; !ok || !isOnboarding.(bool) {
// 		s.logger.Printf("[INFO] Not an onboarding payment intent, skipping")
// 		return nil
// 	}

// 	if pi.IsFinalized {
// 		s.logger.Printf("[INFO] Payment intent already finalized: %s", pi.ID)
// 		return nil
// 	}

// 	verification, err := s.paymentService.VerifyPayment(ctx, gateway, reference)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] Payment verification failed: %v", err)
// 		return fmt.Errorf("payment verification failed: %w", err)
// 	}

// 	if verification.Status != "succeeded" {
// 		s.logger.Printf("[ERROR] Payment not successful: status=%s", verification.Status)
// 		return errors.New("payment not successful")
// 	}

// 	sub, err := s.subscriptionRepo.FindByID(pi.SubscriptionID)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] Subscription not found: %s", pi.SubscriptionID)
// 		return errors.New("subscription not found")
// 	}

// 	sessionID, _ := sub.Metadata["session_id"].(string)
// 	session, _ := s.sessionStore.Get(ctx, sessionID)

// 	now := time.Now()

// 	sub.Status = models.SubStatusActive
// 	sub.StartDate = now
// 	sub.EndDate = calculateEndDate(now, sub.PaymentInterval)
// 	sub.LastPaymentDate = &now
// 	sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
// 	sub.UpdatedAt = now
// 	sub.UpdatedBy = pi.UserID
// 	sub.GatewayData = verification.GatewayData

// 	if err := s.subscriptionRepo.Update(sub); err != nil {
// 		s.logger.Printf("[ERROR] Failed to update subscription: %v", err)
// 		return fmt.Errorf("failed to update subscription: %w", err)
// 	}

// 	school, err := s.onboardingRepo.FindOnboardSchoolByID(sub.SchoolID)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] School not found: %s", sub.SchoolID)
// 		return errors.New("school not found")
// 	}

// 	school.Status = models.SchoolStatusActive
// 	school.UpdatedAt = now
// 	if err := s.onboardingRepo.UpdateOnboardSchool(school); err != nil {
// 		s.logger.Printf("[ERROR] Failed to update school: %v", err)
// 		return fmt.Errorf("failed to update school: %w", err)
// 	}

// 	user, err := s.onboardingRepo.FindOnboardUserByID(pi.UserID)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] User not found: %s", pi.UserID)
// 		return errors.New("user not found")
// 	}

// 	if user.Status == models.StatusPending {
// 		user.Status = models.StatusActive
// 		user.UpdatedAt = now
// 		if err := s.onboardingRepo.UpdateOnboardUser(user); err != nil {
// 			s.logger.Printf("[ERROR] Failed to update user: %v", err)
// 			return fmt.Errorf("failed to update user: %w", err)
// 		}
// 	}

// 	if err := s.subscriptionRepo.MarkInvoiceAsPaid(pi.InvoiceID, now); err != nil {
// 		s.logger.Printf("[ERROR] Failed to mark invoice as paid: %v", err)
// 		return fmt.Errorf("failed to mark invoice as paid: %w", err)
// 	}

// 	pi.Status = models.IntentSucceeded
// 	pi.PaidAt = &now
// 	pi.IsFinalized = true
// 	pi.GatewayResponse = verification.GatewayData
// 	pi.UpdatedAt = now
// 	if err := s.subscriptionRepo.UpdatePaymentIntent(pi); err != nil {
// 		s.logger.Printf("[ERROR] Failed to update payment intent: %v", err)
// 		return fmt.Errorf("failed to update payment intent: %w", err)
// 	}

// 	transaction := &models.PaymentTransaction{
// 		ID:                    uuid.New().String(),
// 		SubscriptionID:        sub.ID,
// 		PaymentIntentID:       pi.ID,
// 		InvoiceID:             pi.InvoiceID,
// 		SchoolID:              sub.SchoolID,
// 		UserID:                pi.UserID,
// 		Amount:                pi.Amount,
// 		Currency:              pi.Currency,
// 		PaymentMethod:         models.MethodCard,
// 		PaymentStatus:         models.PaymentPaid,
// 		Gateway:               gateway,
// 		Reference:             pi.Reference,
// 		TransactionID:         pi.Reference,
// 		GatewayTransactionID:  getTransactionIDFromVerification(verification, gateway),
// 		PaidAt:                &now,
// 		IsLocked:              true,
// 		CreatedAt:             now,
// 		UpdatedAt:             now,
// 		Metadata: map[string]interface{}{
// 			"onboarding": true,
// 			"gateway":    string(gateway),
// 		},
// 	}
// 	s.subscriptionRepo.CreatePaymentTransaction(transaction)

// 	if session != nil {
// 		session.Status = "completed"
// 		session.UpdatedAt = now
// 		s.sessionStore.Update(ctx, session)
// 	}

// 	s.logger.Printf("[SUCCESS] Onboarding activated: subscription_id=%s, school_id=%s, user_id=%s", sub.ID, school.ID, user.ID)

// 	// Send welcome emails
// 	go s.sendOnboardingWelcomeEmailsWithTracking(user, school, sub, gateway)

// 	return nil
// }

func (s *OnboardingService) ActivateOnboarding(ctx context.Context, reference string, gateway models.PaymentGateway) error {
    s.logger.Printf("[INFO] Step 9: Activating onboarding - reference=%s, gateway=%s", reference, gateway)

    // Find payment intent
    pi, err := s.subscriptionRepo.FindPaymentIntentByReference(reference)
    if err != nil {
        s.logger.Printf("[ERROR] Payment intent not found: %s", reference)
        return errors.New("payment intent not found")
    }

    // Check if this is an onboarding payment
    if isOnboarding, ok := pi.Metadata["onboarding"]; !ok || !isOnboarding.(bool) {
        s.logger.Printf("[INFO] Not an onboarding payment intent, skipping")
        return nil
    }

    // Check if already finalized
    if pi.IsFinalized {
        s.logger.Printf("[INFO] Payment intent already finalized: %s", pi.ID)
        return nil
    }

    // Verify payment
    verification, err := s.paymentService.VerifyPayment(ctx, gateway, reference)
    if err != nil {
        s.logger.Printf("[ERROR] Payment verification failed: %v", err)
        return fmt.Errorf("payment verification failed: %w", err)
    }

    // ✅ FIX: Check verification status safely
    if verification == nil {
        s.logger.Printf("[ERROR] Verification response is nil")
        return errors.New("payment verification returned nil")
    }

    if verification.Status != "succeeded" {
        s.logger.Printf("[ERROR] Payment not successful: status=%s", verification.Status)
        return errors.New("payment not successful")
    }

    // ✅ FIX: Ensure GatewayData is not nil before using
    if verification.GatewayData == nil {
        verification.GatewayData = make(map[string]interface{})
        s.logger.Printf("[WARN] GatewayData was nil, initialized empty map")
    }

    // Get subscription
    sub, err := s.subscriptionRepo.FindByID(pi.SubscriptionID)
    if err != nil {
        s.logger.Printf("[ERROR] Subscription not found: %s", pi.SubscriptionID)
        return errors.New("subscription not found")
    }

    // Get session
    sessionID, _ := sub.Metadata["session_id"].(string)
    session, _ := s.sessionStore.Get(ctx, sessionID)

    now := time.Now()

    // Update subscription
    sub.Status = models.SubStatusActive
    sub.StartDate = now
    sub.EndDate = calculateEndDate(now, sub.PaymentInterval)
    sub.LastPaymentDate = &now
    sub.NextPaymentDate = calculateNextPaymentDate(sub.EndDate, sub.PaymentInterval)
    sub.UpdatedAt = now
    sub.UpdatedBy = pi.UserID
    sub.GatewayData = verification.GatewayData

    if err := s.subscriptionRepo.Update(sub); err != nil {
        s.logger.Printf("[ERROR] Failed to update subscription: %v", err)
        return fmt.Errorf("failed to update subscription: %w", err)
    }

    // Update school
    school, err := s.onboardingRepo.FindOnboardSchoolByID(sub.SchoolID)
    if err != nil {
        s.logger.Printf("[ERROR] School not found: %s", sub.SchoolID)
        return errors.New("school not found")
    }

    school.Status = models.SchoolStatusActive
    school.UpdatedAt = now
    if err := s.onboardingRepo.UpdateOnboardSchool(school); err != nil {
        s.logger.Printf("[ERROR] Failed to update school: %v", err)
        return fmt.Errorf("failed to update school: %w", err)
    }

    // Update user
    user, err := s.onboardingRepo.FindOnboardUserByID(pi.UserID)
    if err != nil {
        s.logger.Printf("[ERROR] User not found: %s", pi.UserID)
        return errors.New("user not found")
    }

    if user.Status == models.StatusPending {
        user.Status = models.StatusActive
        user.UpdatedAt = now
        if err := s.onboardingRepo.UpdateOnboardUser(user); err != nil {
            s.logger.Printf("[ERROR] Failed to update user: %v", err)
            return fmt.Errorf("failed to update user: %w", err)
        }
    }

    // Mark invoice as paid
    if err := s.subscriptionRepo.MarkInvoiceAsPaid(pi.InvoiceID, now); err != nil {
        s.logger.Printf("[ERROR] Failed to mark invoice as paid: %v", err)
        return fmt.Errorf("failed to mark invoice as paid: %w", err)
    }

    // Update payment intent
    pi.Status = models.IntentSucceeded
    pi.PaidAt = &now
    pi.IsFinalized = true
    pi.GatewayResponse = verification.GatewayData
    pi.UpdatedAt = now
    if err := s.subscriptionRepo.UpdatePaymentIntent(pi); err != nil {
        s.logger.Printf("[ERROR] Failed to update payment intent: %v", err)
        return fmt.Errorf("failed to update payment intent: %w", err)
    }

    // ✅ FIX: Create transaction with safe gateway transaction ID
    transactionID := getTransactionIDFromVerification(verification, gateway)
    s.logger.Printf("[INFO] Gateway transaction ID: %s", transactionID)

    transaction := &models.PaymentTransaction{
        ID:                    uuid.New().String(),
        SubscriptionID:        sub.ID,
        PaymentIntentID:       pi.ID,
        InvoiceID:             pi.InvoiceID,
        SchoolID:              sub.SchoolID,
        UserID:                pi.UserID,
        Amount:                pi.Amount,
        Currency:              pi.Currency,
        PaymentMethod:         models.MethodCard,
        PaymentStatus:         models.PaymentPaid,
        Gateway:               gateway,
        Reference:             pi.Reference,
        TransactionID:         pi.Reference,
        GatewayTransactionID:  transactionID,
        PaidAt:                &now,
        IsLocked:              true,
        CreatedAt:             now,
        UpdatedAt:             now,
        Metadata: map[string]interface{}{
            "onboarding": true,
            "gateway":    string(gateway),
        },
    }
    s.subscriptionRepo.CreatePaymentTransaction(transaction)

    // Update session
    if session != nil {
        session.Status = "completed"
        session.UpdatedAt = now
        s.sessionStore.Update(ctx, session)
    }

    s.logger.Printf("[SUCCESS] Onboarding activated: subscription_id=%s, school_id=%s, user_id=%s", sub.ID, school.ID, user.ID)

    // Send welcome emails
    go s.sendOnboardingWelcomeEmailsWithTracking(user, school, sub, gateway)

    return nil
}

// ============================================
// GET ONBOARDING STATUS
// ============================================

func (s *OnboardingService) GetOnboardingStatus(ctx context.Context, sessionID string) (*dto.OnboardingStatusResponse, error) {
	session, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Session not found: %s", sessionID)
		return nil, errors.New("session not found")
	}

	return &dto.OnboardingStatusResponse{
		SessionID:       session.ID,
		Status:          session.Status,
		Tier:            string(session.Tier),
		Interval:        string(session.Interval),
		Gateway:         string(session.Gateway),
		Amount:          session.Amount.String(),
		HasUser:         session.UserID != "",
		HasSchool:       session.SchoolID != "",
		HasSubscription: session.SubscriptionID != "",
		EmailVerified:   session.Steps.Step3Completed,
		NextStep:        s.getNextStep(session),
		ExpiresAt:       session.ExpiresAt,
	}, nil
}

// ============================================
// CHECK PAYMENT STATUS
// ============================================

func (s *OnboardingService) CheckPaymentStatus(ctx context.Context, sessionID string) (*dto.PaymentStatusResponse, error) {
	session, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Session not found: %s", sessionID)
		return nil, errors.New("session not found")
	}

	if session.Status == "completed" {
		return &dto.PaymentStatusResponse{
			Status:     "completed",
			Message:    "Payment completed successfully! Your account is being activated.",
			IsComplete: true,
		}, nil
	}

	if session.Status == "payment_pending" {
		pi, _ := s.subscriptionRepo.FindPaymentIntentByID(session.PaymentIntentID)
		paymentLink := ""
		if pi != nil {
			paymentLink = pi.AuthorizationURL
		}
		return &dto.PaymentStatusResponse{
			Status:      "pending",
			Message:     "Payment is still pending. Please complete the payment.",
			IsComplete:  false,
			PaymentLink: paymentLink,
		}, nil
	}

	return &dto.PaymentStatusResponse{
		Status:     session.Status,
		Message:    "Session is in progress.",
		IsComplete: false,
	}, nil
}

// ============================================
// GET SESSION BY EMAIL
// ============================================

func (s *OnboardingService) GetSessionByEmail(ctx context.Context, email string) (*dto.GetSessionByEmailResponse, error) {
	session, err := s.sessionStore.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Printf("[ERROR] Session not found for email: %s", email)
		return nil, errors.New("session not found")
	}

	return &dto.GetSessionByEmailResponse{
		SessionID: session.ID,
		Email:     session.Email,
		Status:    session.Status,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// ============================================
// RESEND VERIFICATION OTP
// ============================================

func (s *OnboardingService) ResendVerificationOTP(ctx context.Context, sessionID string) error {
	session, err := s.getValidSession(ctx, sessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return err
	}

	if session.Steps.Step3Completed {
		s.logger.Printf("[WARN] Email already verified for session=%s", sessionID)
		return errors.New("email already verified")
	}

	user, err := s.onboardingRepo.FindOnboardUserByID(session.UserID)
	if err != nil {
		s.logger.Printf("[ERROR] User not found: %s", session.UserID)
		return errors.New("user not found")
	}

	if user.Email == nil {
		s.logger.Printf("[ERROR] User has no email: %s", session.UserID)
		return errors.New("user has no email")
	}

	go s.sendVerificationOTPWithTracking(*user.Email, user.FirstName, user.ID)

	s.logger.Printf("[SUCCESS] OTP resent to %s", *user.Email)
	return nil
}

// ============================================
// RESEND PAYMENT LINK
// ============================================

func (s *OnboardingService) ResendPaymentLink(ctx context.Context, sessionID string) error {
	session, err := s.getValidSession(ctx, sessionID)
	if err != nil {
		s.logger.Printf("[ERROR] Invalid session: %v", err)
		return err
	}

	if session.Status != "payment_pending" {
		s.logger.Printf("[WARN] No pending payment for session=%s", sessionID)
		return errors.New("no pending payment found")
	}

	user, err := s.onboardingRepo.FindOnboardUserByID(session.UserID)
	if err != nil {
		s.logger.Printf("[ERROR] User not found: %s", session.UserID)
		return errors.New("user not found")
	}

	school, err := s.onboardingRepo.FindOnboardSchoolByID(session.SchoolID)
	if err != nil {
		s.logger.Printf("[ERROR] School not found: %s", session.SchoolID)
		return errors.New("school not found")
	}

	sub, err := s.subscriptionRepo.FindByID(session.SubscriptionID)
	if err != nil {
		s.logger.Printf("[ERROR] Subscription not found: %s", session.SubscriptionID)
		return errors.New("subscription not found")
	}

	pi, err := s.subscriptionRepo.FindPaymentIntentByID(session.PaymentIntentID)
	if err != nil {
		s.logger.Printf("[ERROR] Payment intent not found: %s", session.PaymentIntentID)
		return errors.New("payment intent not found")
	}

	go s.sendPaymentLinkEmailWithTracking(user, school, sub, nil, pi, session.Gateway)

	s.logger.Printf("[SUCCESS] Payment link resent to %s", *user.Email)
	return nil
}

// ============================================
// COMPLETE ONBOARDING (Manual Admin)
// ============================================

// func (s *OnboardingService) CompleteOnboarding(ctx context.Context, sessionID string) (*dto.ActivationResponse, error) {
// 	session, err := s.getValidSession(ctx, sessionID)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] Invalid session: %v", err)
// 		return nil, err
// 	}

// 	if session.Status == "completed" {
// 		return &dto.ActivationResponse{
// 			Success:        true,
// 			Message:        "Onboarding already completed.",
// 			SchoolID:       session.SchoolID,
// 			AdminID:        session.UserID,
// 			SubscriptionID: session.SubscriptionID,
// 			RedirectURL:    fmt.Sprintf("%s/dashboard", s.config.AppURL),
// 		}, nil
// 	}

// 	if session.Status != "payment_pending" {
// 		s.logger.Printf("[ERROR] Payment not initiated for session=%s", sessionID)
// 		return nil, errors.New("payment not initiated")
// 	}

// 	pi, err := s.subscriptionRepo.FindPaymentIntentByID(session.PaymentIntentID)
// 	if err != nil {
// 		s.logger.Printf("[ERROR] Payment intent not found: %s", session.PaymentIntentID)
// 		return nil, errors.New("payment intent not found")
// 	}

// 	if pi.IsFinalized {
// 		if err := s.ActivateOnboarding(ctx, pi.Reference, pi.Gateway); err != nil {
// 			s.logger.Printf("[ERROR] Failed to activate onboarding: %v", err)
// 			return nil, err
// 		}
// 	} else {
// 		verification, err := s.paymentService.VerifyPayment(ctx, pi.Gateway, pi.Reference)
// 		if err != nil {
// 			s.logger.Printf("[ERROR] Payment verification failed: %v", err)
// 			return nil, fmt.Errorf("payment verification failed: %w", err)
// 		}

// 		if verification.Status != "succeeded" {
// 			s.logger.Printf("[ERROR] Payment not successful: status=%s", verification.Status)
// 			return nil, errors.New("payment not successful")
// 		}

// 		if err := s.ActivateOnboarding(ctx, pi.Reference, pi.Gateway); err != nil {
// 			s.logger.Printf("[ERROR] Failed to activate onboarding: %v", err)
// 			return nil, err
// 		}
// 	}

// 	return &dto.ActivationResponse{
// 		Success:        true,
// 		Message:        "Onboarding completed successfully!",
// 		SchoolID:       session.SchoolID,
// 		AdminID:        session.UserID,
// 		SubscriptionID: session.SubscriptionID,
// 		RedirectURL:    fmt.Sprintf("%s/dashboard", s.config.AppURL),
// 	}, nil
// }
func (s *OnboardingService) CompleteOnboarding(ctx context.Context, sessionID string) (*dto.ActivationResponse, error) {
    s.logger.Printf("[INFO] Admin completing onboarding - session=%s", sessionID)

    session, err := s.getValidSession(ctx, sessionID)
    if err != nil {
        s.logger.Printf("[ERROR] Invalid session: %v", err)
        return nil, err
    }

    // Check if already completed
    if session.Status == "completed" {
        s.logger.Printf("[INFO] Onboarding already completed for session=%s", sessionID)
        return &dto.ActivationResponse{
            Success:        true,
            Message:        "Onboarding already completed.",
            SchoolID:       session.SchoolID,
            AdminID:        session.UserID,
            SubscriptionID: session.SubscriptionID,
            RedirectURL:    fmt.Sprintf("%s/dashboard", s.config.AppURL),
        }, nil
    }

    // Check if payment is initiated
    if session.Status != "payment_pending" {
        s.logger.Printf("[ERROR] Payment not initiated for session=%s", sessionID)
        return nil, errors.New("payment not initiated")
    }

    // Get payment intent
    pi, err := s.subscriptionRepo.FindPaymentIntentByID(session.PaymentIntentID)
    if err != nil {
        s.logger.Printf("[ERROR] Payment intent not found: %s", session.PaymentIntentID)
        return nil, errors.New("payment intent not found")
    }

    // If payment is already finalized, just activate
    if pi.IsFinalized {
        s.logger.Printf("[INFO] Payment already finalized, activating...")
        if err := s.ActivateOnboarding(ctx, pi.Reference, pi.Gateway); err != nil {
            s.logger.Printf("[ERROR] Failed to activate onboarding: %v", err)
            return nil, err
        }
    } else {
        // Verify payment
        s.logger.Printf("[INFO] Verifying payment - reference=%s, gateway=%s", pi.Reference, pi.Gateway)
        
        verification, err := s.paymentService.VerifyPayment(ctx, pi.Gateway, pi.Reference)
        if err != nil {
            s.logger.Printf("[ERROR] Payment verification failed: %v", err)
            return nil, fmt.Errorf("payment verification failed: %w", err)
        }

        // ✅ FIX: Check verification is not nil
        if verification == nil {
            s.logger.Printf("[ERROR] Payment verification returned nil")
            return nil, errors.New("payment verification returned nil")
        }

        if verification.Status != "succeeded" {
            s.logger.Printf("[ERROR] Payment not successful: status=%s", verification.Status)
            return nil, errors.New("payment not successful")
        }

        // Activate
        if err := s.ActivateOnboarding(ctx, pi.Reference, pi.Gateway); err != nil {
            s.logger.Printf("[ERROR] Failed to activate onboarding: %v", err)
            return nil, err
        }
    }

    // Get updated session
    session, _ = s.sessionStore.Get(ctx, sessionID)

    return &dto.ActivationResponse{
        Success:        true,
        Message:        "Onboarding completed successfully!",
        SchoolID:       session.SchoolID,
        AdminID:        session.UserID,
        SubscriptionID: session.SubscriptionID,
        RedirectURL:    fmt.Sprintf("%s/dashboard", s.config.AppURL),
    }, nil
}

// ============================================
// CLEANUP EXPIRED SESSIONS
// ============================================

func (s *OnboardingService) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessionStore.CleanupExpired(ctx)
}

// ============================================
// GET ONBOARDING STATISTICS
// ============================================

func (s *OnboardingService) GetOnboardingStats(ctx context.Context) (map[string]int64, error) {
	return map[string]int64{
		"total_sessions":     0,
		"pending":            0,
		"payment_pending":    0,
		"completed":          0,
		"expired":            0,
	}, nil
}

// ============================================
// PROCESS WEBHOOK
// ============================================

func (s *OnboardingService) ProcessWebhook(ctx context.Context, payload []byte, signature string, gateway models.PaymentGateway) error {
	s.logger.Printf("[INFO] Processing webhook for gateway: %s", gateway)

	gw, err := s.paymentService.GetGateway(gateway)
	if err != nil {
		s.logger.Printf("[ERROR] Gateway not found: %s", gateway)
		return fmt.Errorf("gateway not found: %w", err)
	}

	event, err := gw.ParseWebhook(ctx, payload, signature)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to parse webhook: %v", err)
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	switch event.Type {
	case "payment_success":
		s.logger.Printf("[INFO] Payment success webhook received for reference: %s", event.Reference)
		if err := s.ActivateOnboarding(ctx, event.Reference, gateway); err != nil {
			s.logger.Printf("[ERROR] Failed to activate onboarding: %v", err)
			return err
		}
		s.logger.Printf("[SUCCESS] Onboarding activated successfully for reference: %s", event.Reference)
	default:
		s.logger.Printf("[INFO] Unhandled webhook event type: %s", event.Type)
	}

	return nil
}

// ============================================
// EMAIL METHODS - WITH TRACKING
// ============================================

func (s *OnboardingService) sendVerificationOTPWithTracking(email, firstName, userID string) {
	s.logger.Printf("[INFO] Starting sendVerificationOTP for %s", email)

	if !s.config.EnableEmail {
		s.logger.Printf("[WARN] Email disabled - skipping OTP for %s", email)
		return
	}

	if s.emailService == nil {
		s.logger.Printf("[ERROR] Email service is nil - cannot send OTP to %s", email)
		return
	}

	// Invalidate existing OTPs for this user
	if err := s.onboardingRepo.InvalidateOnboardUserOTPs(userID, "email_verification"); err != nil {
		s.logger.Printf("[WARN] Failed to invalidate existing OTPs: %v", err)
	}

	// Generate 6-digit OTP code
	code := generateOTP(6)

	// Hash the code for storage
	hashedCode, err := utils.HashPassword(code)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to hash OTP for %s: %v", email, err)
		return
	}

	// Store OTP without ID (let GORM auto-generate)
	otp := &models.OTP{
		UserID:    userID,
		Code:      hashedCode,
		Type:      "email_verification",
		Used:      false,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := s.onboardingRepo.CreateOnboardOTP(otp); err != nil {
		s.logger.Printf("[ERROR] Failed to store OTP for %s: %v", email, err)
		return
	}

	s.logger.Printf("[INFO] Sending verification OTP to: %s with code: %s", email, code)

	err = s.emailService.SendAdminVerifyEmail(email, firstName, code)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to send admin verify email to %s: %v", email, err)
		return
	}

	s.logger.Printf("[SUCCESS] Admin verify email sent to %s with OTP: %s", email, code)
}

func (s *OnboardingService) sendPaymentLinkEmailWithTracking(user *models.User, school *models.School, sub *models.Subscription, inv *models.Invoice, pi *models.PaymentIntent, gateway models.PaymentGateway) {
	s.logger.Printf("[INFO] Starting sendPaymentLinkEmail for %s", *user.Email)

	if !s.config.EnableEmail {
		s.logger.Printf("[WARN] Email disabled - skipping payment link for %s", *user.Email)
		return
	}

	if s.emailService == nil {
		s.logger.Printf("[ERROR] Email service is nil - cannot send payment link to %s", *user.Email)
		return
	}

	s.logger.Printf("[INFO] Sending payment link email to: %s via %s", *user.Email, gateway)

	err := s.emailService.SendPaymentLinkEmail(
		*user.Email,
		user.FirstName,
		string(sub.Tier),
		sub.Amount.String(),
		pi.AuthorizationURL,
	)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to send payment link email to %s: %v", *user.Email, err)
		return
	}

	s.logger.Printf("[SUCCESS] Payment link email sent to %s", *user.Email)
}

func (s *OnboardingService) sendSchoolRegistrationEmailWithTracking(user *models.User, school *models.School) {
	s.logger.Printf("[INFO] Starting sendSchoolRegistrationEmail for %s", *user.Email)

	if !s.config.EnableEmail {
		s.logger.Printf("[WARN] Email disabled - skipping school registration email for %s", *user.Email)
		return
	}

	if s.emailService == nil {
		s.logger.Printf("[ERROR] Email service is nil - cannot send school registration email to %s", *user.Email)
		return
	}

	s.logger.Printf("[INFO] Sending school registration email to: %s", *user.Email)

	err := s.emailService.SendSchoolRegistrationEmail(
		*user.Email,
		user.FirstName,
		school.Name,
		school.Code,
	)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to send school registration email to %s: %v", *user.Email, err)
		return
	}

	s.logger.Printf("[SUCCESS] School registration email sent to %s", *user.Email)
}

func (s *OnboardingService) sendOnboardingWelcomeEmailsWithTracking(user *models.User, school *models.School, sub *models.Subscription, gateway models.PaymentGateway) {
	s.logger.Printf("[INFO] Starting sendOnboardingWelcomeEmails for %s", *user.Email)

	if !s.config.EnableEmail {
		s.logger.Printf("[WARN] Email disabled - skipping welcome email for %s", *user.Email)
		return
	}

	if s.emailService == nil {
		s.logger.Printf("[ERROR] Email service is nil - cannot send welcome email to %s", *user.Email)
		return
	}

	s.logger.Printf("[INFO] Sending onboarding completion email to: %s", *user.Email)

	err := s.emailService.SendOnboardingCompleteEmail(
		*user.Email,
		user.FirstName,
		school.Name,
		string(sub.Tier),
		sub.Amount.String(),
	)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to send onboarding complete email to %s: %v", *user.Email, err)
		return
	}

	s.logger.Printf("[SUCCESS] Onboarding complete email sent to %s", *user.Email)
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func (s *OnboardingService) isGatewayAvailable(gateway models.PaymentGateway) bool {
	available := s.paymentService.GetAvailableGateways()
	for _, g := range available {
		if g == gateway {
			return true
		}
	}
	return false
}

func (s *OnboardingService) getValidSession(ctx context.Context, sessionID string) (*OnboardingSession, error) {
	session, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return nil, errors.New("invalid or expired session")
	}
	if session.Status == "expired" {
		return nil, errors.New("session expired")
	}
	if session.Status == "completed" {
		return nil, errors.New("session already completed")
	}
	return session, nil
}

func (s *OnboardingService) getNextStep(session *OnboardingSession) string {
	if !session.Steps.Step1Completed {
		return "select_plan"
	}
	if !session.Steps.Step2Completed {
		return "create_admin"
	}
	if !session.Steps.Step3Completed {
		return "verify_email"
	}
	if !session.Steps.Step4Completed {
		return "register_school"
	}
	if !session.Steps.Step6Completed {
		return "create_subscription"
	}
	if session.Status == "payment_pending" {
		return "complete_payment"
	}
	if session.Status == "completed" {
		return "complete"
	}
	return "review"
}

func (s *OnboardingService) getDefaultCallbackURL(gateway models.PaymentGateway) string {
	baseURL := s.config.AppURL
	if baseURL == "" {
		baseURL = "https://yourapp.com"
	}
	return fmt.Sprintf("%s/api/webhooks/%s", baseURL, string(gateway))
}

func (s *OnboardingService) getDefaultSuccessURL(gateway models.PaymentGateway) string {
	baseURL := s.config.AppURL
	if baseURL == "" {
		baseURL = "https://yourapp.com"
	}
	return fmt.Sprintf("%s/payment/success?gateway=%s", baseURL, string(gateway))
}

func (s *OnboardingService) getDefaultCancelURL(gateway models.PaymentGateway) string {
	baseURL := s.config.AppURL
	if baseURL == "" {
		baseURL = "https://yourapp.com"
	}
	return fmt.Sprintf("%s/payment/cancel?gateway=%s", baseURL, string(gateway))
}

// func getTransactionIDFromVerification(verification *payment.PaymentVerification, gateway models.PaymentGateway) string {
// 	if verification == nil || verification.GatewayData == nil {
// 		return ""
// 	}

// 	switch gateway {
// 	case models.GatewayPaystack:
// 		if id, ok := verification.GatewayData["id"].(float64); ok {
// 			return fmt.Sprintf("%.0f", id)
// 		}
// 		if id, ok := verification.GatewayData["id"].(string); ok {
// 			return id
// 		}
// 	case models.GatewayFlutterwave:
// 		if id, ok := verification.GatewayData["id"].(string); ok {
// 			return id
// 		}
// 		if id, ok := verification.GatewayData["transaction_id"].(float64); ok {
// 			return fmt.Sprintf("%.0f", id)
// 		}
// 	case models.GatewayStripe:
// 		if id, ok := verification.GatewayData["id"].(string); ok {
// 			return id
// 		}
// 	}
// 	return verification.Reference
// }

func getTransactionIDFromVerification(verification *payment.PaymentVerification, gateway models.PaymentGateway) string {
    // Handle nil verification
    if verification == nil {
        return ""
    }
    
    // Handle nil GatewayData
    if verification.GatewayData == nil {
        return verification.Reference
    }

    switch gateway {
    case models.GatewayPaystack:
        if id, ok := verification.GatewayData["id"]; ok && id != nil {
            switch v := id.(type) {
            case float64:
                return fmt.Sprintf("%.0f", v)
            case string:
                return v
            default:
                return fmt.Sprintf("%v", v)
            }
        }

    case models.GatewayFlutterwave:
        // Check for "id" field
        if id, ok := verification.GatewayData["id"]; ok && id != nil {
            switch v := id.(type) {
            case string:
                return v
            case float64:
                return fmt.Sprintf("%.0f", v)
            default:
                return fmt.Sprintf("%v", v)
            }
        }
        // Check for "transaction_id" field
        if id, ok := verification.GatewayData["transaction_id"]; ok && id != nil {
            switch v := id.(type) {
            case float64:
                return fmt.Sprintf("%.0f", v)
            case string:
                return v
            default:
                return fmt.Sprintf("%v", v)
            }
        }

    case models.GatewayStripe:
        if id, ok := verification.GatewayData["id"]; ok && id != nil {
            switch v := id.(type) {
            case string:
                return v
            default:
                return fmt.Sprintf("%v", v)
            }
        }
    }

    // Fallback to reference
    return verification.Reference
}

func calculateEndDate(start time.Time, interval models.PaymentInterval) time.Time {
	switch interval {
	case models.IntervalMonthly:
		return start.AddDate(0, 1, 0)
	case models.IntervalQuarterly:
		return start.AddDate(0, 3, 0)
	case models.IntervalYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

func calculateNextPaymentDate(currentEnd time.Time, interval models.PaymentInterval) *time.Time {
	next := calculateEndDate(currentEnd, interval)
	return &next
}

func getFeaturesForTier(tier models.SubscriptionTier) models.JSONMap {
	features := make(models.JSONMap)
	switch tier {
	case models.TierBasic:
		features["analytics"] = false
		features["api_access"] = false
		features["priority_support"] = false
		features["bulk_import"] = false
		features["white_label"] = false
	case models.TierPremium:
		features["analytics"] = true
		features["api_access"] = true
		features["priority_support"] = false
		features["bulk_import"] = true
		features["white_label"] = false
		features["custom_reports"] = true
	case models.TierEnterprise:
		features["analytics"] = true
		features["api_access"] = true
		features["priority_support"] = true
		features["bulk_import"] = true
		features["white_label"] = true
		features["custom_reports"] = true
		features["dedicated_server"] = true
		features["sla"] = true
	}
	return features
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func generateSchoolCode(name string) string {
	return fmt.Sprintf("SCH-%d", time.Now().UnixNano()%1000000)
}

func generateInvoiceNumber() string {
	return fmt.Sprintf("INV-%d", time.Now().UnixNano()%100000000)
}

// func generateOTP(length int) string {
// 	const charset = "0123456789"
// 	otp := make([]byte, length)
// 	for i := range otp {
// 		otp[i] = charset[int(time.Now().UnixNano())%len(charset)]
// 	}
// 	return string(otp)
// }

func generateOTP(length int) string {
    // Use a seeded random generator
    rng := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    const charset = "0123456789"
    otp := make([]byte, length)
    for i := range otp {
        otp[i] = charset[rng.Intn(len(charset))]
    }
    return string(otp)
}


