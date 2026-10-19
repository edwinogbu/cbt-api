package service

import (
	"context"
	"testing"

	"cbt-api/internal/onboarding/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- GetOnboardingStats: real data, not a hardcoded stub ----
//
// Before this fix, GetOnboardingStats always returned a map of zeros
// regardless of the real onboarding_sessions table - a misleading
// "looks like live data" response to any caller, platform_owner
// included. It now reports real counts from the DB-backed
// OnboardingRepository (not the separate, unrelated in-memory
// sessionStore CompleteOnboarding/CleanupExpiredSessions use).

func setupOnboardingStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE onboarding_sessions (
		id TEXT PRIMARY KEY, session_token TEXT, plan TEXT, status TEXT,
		user_id TEXT, school_id TEXT, subscription_id TEXT,
		expires_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func seedOnboardingSession(t *testing.T, db *gorm.DB, id, status string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO onboarding_sessions (id, session_token, status, expires_at, created_at, updated_at) VALUES (?, ?, ?, datetime('now', '+1 day'), datetime('now'), datetime('now'))",
		id, id+"-token", status,
	).Error; err != nil {
		t.Fatalf("failed to seed onboarding session: %v", err)
	}
}

func newTestOnboardingService(db *gorm.DB) *OnboardingService {
	return &OnboardingService{onboardingRepo: repository.NewOnboardingRepository(db)}
}

func TestGetOnboardingStats_ReportsRealCounts(t *testing.T) {
	db := setupOnboardingStatsTestDB(t)
	s := newTestOnboardingService(db)
	seedOnboardingSession(t, db, "s-1", "PLAN_SELECTED")
	seedOnboardingSession(t, db, "s-2", "ACCOUNT_CREATED")
	seedOnboardingSession(t, db, "s-3", "EMAIL_VERIFIED")
	seedOnboardingSession(t, db, "s-4", "SCHOOL_CREATED")
	seedOnboardingSession(t, db, "s-5", "PAYMENT_PENDING")
	seedOnboardingSession(t, db, "s-6", "PAYMENT_PENDING")
	seedOnboardingSession(t, db, "s-7", "COMPLETE")
	seedOnboardingSession(t, db, "s-8", "EXPIRED")

	stats, err := s.GetOnboardingStats(context.Background())
	if err != nil {
		t.Fatalf("expected real stats, got error: %v", err)
	}

	if stats["total_sessions"] != 8 {
		t.Fatalf("expected total_sessions=8, got %d", stats["total_sessions"])
	}
	if stats["pending"] != 4 {
		t.Fatalf("expected pending=4 (plan_selected+account_created+email_verified+school_created), got %d", stats["pending"])
	}
	if stats["payment_pending"] != 2 {
		t.Fatalf("expected payment_pending=2, got %d", stats["payment_pending"])
	}
	if stats["completed"] != 1 {
		t.Fatalf("expected completed=1, got %d", stats["completed"])
	}
	if stats["expired"] != 1 {
		t.Fatalf("expected expired=1, got %d", stats["expired"])
	}
}

func TestGetOnboardingStats_EmptyTable_AllZero(t *testing.T) {
	db := setupOnboardingStatsTestDB(t)
	s := newTestOnboardingService(db)

	stats, err := s.GetOnboardingStats(context.Background())
	if err != nil {
		t.Fatalf("expected no error on an empty table, got: %v", err)
	}
	for key, got := range stats {
		if got != 0 {
			t.Fatalf("expected %s=0 on an empty table, got %d", key, got)
		}
	}
}
