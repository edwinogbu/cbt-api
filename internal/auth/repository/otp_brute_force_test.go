package repository

import (
	"testing"
	"time"

	"cbt-api/internal/models"
	"cbt-api/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- FindValidOTP: brute-force attempt cap ----
//
// Before this fix, a password-reset OTP had no limit on how many wrong
// codes could be tried against it. A 6-digit OTP (generateOTP(6)) has
// only 1,000,000 possible values, it stays valid for 10 minutes, and
// there is no rate limiter anywhere else in this app guarding
// /auth/reset-password - so an attacker who knew a victim's email could
// brute-force the code within its validity window and take over any
// account, admins included. This caps a single OTP at maxOTPAttempts
// wrong guesses before it is treated as exhausted (gorm.ErrRecordNotFound),
// forcing the attacker back through ForgotPassword, which invalidates
// the old OTP and issues a fresh one - email-delivered, not guessable
// from the attacker's own request.

func setupOTPBruteForceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, email TEXT, deleted_at DATETIME
		)`,
		`CREATE TABLE otps (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT, code TEXT, type TEXT,
			used INTEGER DEFAULT 0, attempts INTEGER DEFAULT 0, expires_at DATETIME, created_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedOTPUser(t *testing.T, db *gorm.DB, id, email string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, email) VALUES (?, ?)", id, email).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedOTPFor(t *testing.T, db *gorm.DB, userID, rawCode, otpType string) {
	t.Helper()
	hashed, err := utils.HashPassword(rawCode)
	if err != nil {
		t.Fatalf("failed to hash code: %v", err)
	}
	otp := &models.OTP{
		UserID:    userID,
		Code:      hashed,
		Type:      otpType,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := db.Create(otp).Error; err != nil {
		t.Fatalf("failed to seed otp: %v", err)
	}
}

func TestFindValidOTP_CorrectCode_SucceedsImmediately(t *testing.T) {
	db := setupOTPBruteForceTestDB(t)
	seedOTPUser(t, db, "user-1", "victim@example.com")
	seedOTPFor(t, db, "user-1", "123456", "password_reset")

	repo := NewAuthRepository(db)
	otp, err := repo.FindValidOTP("victim@example.com", "123456", "password_reset")
	if err != nil {
		t.Fatalf("expected the correct code to succeed, got error: %v", err)
	}
	if otp.UserID != "user-1" {
		t.Fatalf("expected otp for user-1, got %q", otp.UserID)
	}
}

func TestFindValidOTP_WrongCode_RecordedAsAttempt(t *testing.T) {
	db := setupOTPBruteForceTestDB(t)
	seedOTPUser(t, db, "user-1", "victim@example.com")
	seedOTPFor(t, db, "user-1", "123456", "password_reset")

	repo := NewAuthRepository(db)
	if _, err := repo.FindValidOTP("victim@example.com", "000000", "password_reset"); err == nil {
		t.Fatal("expected a wrong code to be rejected")
	}

	var otp models.OTP
	if err := db.Where("user_id = ?", "user-1").First(&otp).Error; err != nil {
		t.Fatalf("failed to load otp: %v", err)
	}
	if otp.Attempts != 1 {
		t.Fatalf("expected attempts=1 after one wrong guess, got %d", otp.Attempts)
	}
}

// TestFindValidOTP_BruteForce_LockedOutAfterMaxAttempts is the core
// regression: even the RIGHT code must be rejected once maxOTPAttempts
// wrong guesses have already been spent against this OTP, closing the
// unbounded-guessing window a static 6-digit code would otherwise leave
// open for the whole 10-minute validity period.
func TestFindValidOTP_BruteForce_LockedOutAfterMaxAttempts(t *testing.T) {
	db := setupOTPBruteForceTestDB(t)
	seedOTPUser(t, db, "user-1", "victim@example.com")
	seedOTPFor(t, db, "user-1", "123456", "password_reset")

	repo := NewAuthRepository(db)
	for i := 0; i < maxOTPAttempts; i++ {
		if _, err := repo.FindValidOTP("victim@example.com", "000000", "password_reset"); err == nil {
			t.Fatalf("expected wrong-code attempt %d to be rejected", i+1)
		}
	}

	if _, err := repo.FindValidOTP("victim@example.com", "123456", "password_reset"); err == nil {
		t.Fatal("expected the correct code to be rejected once maxOTPAttempts wrong guesses were spent, got no error")
	}
}
