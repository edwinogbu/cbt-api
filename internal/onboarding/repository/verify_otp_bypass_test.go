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

// ---- VerifyOnboardEmailWithOTP: code must actually be checked ----
//
// Before this fix, this method looked up an OTP row by
// user_id/type/used/expiry alone and never compared the submitted code
// against the stored hash at all. Any string - empty, wrong, anything -
// verified the email as long as one unused, unexpired OTP row existed
// for that user, which StartOnboarding always creates. That's a
// complete bypass of email-ownership proof: anyone could register an
// onboarding admin account under a THIRD PARTY's email and mark it
// "verified" without ever receiving or guessing the real code.

func setupVerifyOTPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, email_verified INTEGER DEFAULT 0, status TEXT,
			updated_at DATETIME, deleted_at DATETIME
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

func seedOnboardUser(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, email_verified, status) VALUES (?, 0, 'pending')", id).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedOnboardOTP(t *testing.T, db *gorm.DB, userID, rawCode string) {
	t.Helper()
	hashed, err := utils.HashPassword(rawCode)
	if err != nil {
		t.Fatalf("failed to hash code: %v", err)
	}
	otp := &models.OTP{
		UserID:    userID,
		Code:      hashed,
		Type:      "email_verification",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := db.Create(otp).Error; err != nil {
		t.Fatalf("failed to seed otp: %v", err)
	}
}

func TestVerifyOnboardEmailWithOTP_WrongCode_Rejected(t *testing.T) {
	db := setupVerifyOTPTestDB(t)
	seedOnboardUser(t, db, "user-1")
	seedOnboardOTP(t, db, "user-1", "123456")

	repo := NewOnboardingRepository(db)
	if err := repo.VerifyOnboardEmailWithOTP("user-1", "000000"); err == nil {
		t.Fatal("expected a wrong code to be rejected, but email was verified")
	}

	var verified int
	if err := db.Table("users").Where("id = ?", "user-1").Pluck("email_verified", &verified).Error; err != nil {
		t.Fatalf("failed to read user: %v", err)
	}
	if verified != 0 {
		t.Fatal("email was marked verified despite a wrong code")
	}
}

func TestVerifyOnboardEmailWithOTP_EmptyCode_Rejected(t *testing.T) {
	db := setupVerifyOTPTestDB(t)
	seedOnboardUser(t, db, "user-2")
	seedOnboardOTP(t, db, "user-2", "654321")

	repo := NewOnboardingRepository(db)
	if err := repo.VerifyOnboardEmailWithOTP("user-2", ""); err == nil {
		t.Fatal("expected an empty code to be rejected, but email was verified")
	}
}

func TestVerifyOnboardEmailWithOTP_CorrectCode_Succeeds(t *testing.T) {
	db := setupVerifyOTPTestDB(t)
	seedOnboardUser(t, db, "user-3")
	seedOnboardOTP(t, db, "user-3", "111222")

	repo := NewOnboardingRepository(db)
	if err := repo.VerifyOnboardEmailWithOTP("user-3", "111222"); err != nil {
		t.Fatalf("expected the correct code to succeed, got error: %v", err)
	}

	var verified int
	if err := db.Table("users").Where("id = ?", "user-3").Pluck("email_verified", &verified).Error; err != nil {
		t.Fatalf("failed to read user: %v", err)
	}
	if verified != 1 {
		t.Fatal("expected email_verified to be set after a correct code")
	}
}
