package repository

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"cbt-api/internal/models"
)

// ---- ClaimPaymentIntentFinalization: idempotency and replay protection ----
//
// ActivateOnboarding used to finalize a payment intent with a plain
// read-then-write: check pi.IsFinalized, then later save pi with
// IsFinalized=true. Two concurrent webhook deliveries for the same
// payment (a gateway retrying on timeout, or a captured valid
// signature replayed later) could both read IsFinalized=false before
// either commits, and both would then re-run every side effect of
// activation - double-crediting the subscription, inserting a second
// PaymentTransaction row, emailing the customer twice. This method
// replaces that read-then-write with a single conditional UPDATE
// (WHERE is_finalized = false) that the database resolves atomically,
// so only one caller can ever observe RowsAffected > 0.

func setupPaymentIntentFinalizationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE payment_intents (
		id TEXT PRIMARY KEY,
		subscription_id TEXT, invoice_id TEXT, school_id TEXT, user_id TEXT,
		idempotency_key TEXT, gateway TEXT, amount TEXT, currency TEXT,
		reference TEXT, status TEXT DEFAULT 'pending',
		is_finalized INTEGER DEFAULT 0,
		paid_at DATETIME, gateway_response TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func seedPaymentIntent(t *testing.T, db *gorm.DB, id string, isFinalized bool) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO payment_intents (id, idempotency_key, reference, status, is_finalized, created_at, updated_at) VALUES (?, ?, ?, 'pending', ?, datetime('now'), datetime('now'))",
		id, id+"-idem", id+"-ref", isFinalized,
	).Error; err != nil {
		t.Fatalf("failed to seed payment intent: %v", err)
	}
}

func TestClaimPaymentIntentFinalization_FirstCallerWinsAndPersists(t *testing.T) {
	db := setupPaymentIntentFinalizationTestDB(t)
	r := NewSubscriptionRepository(db)
	seedPaymentIntent(t, db, "pi-1", false)

	claimed, err := r.ClaimPaymentIntentFinalization("pi-1", time.Now(), models.JSONMap{"gateway_ref": "abc123"})
	if err != nil {
		t.Fatalf("expected no error claiming an unfinalized intent, got: %v", err)
	}
	if !claimed {
		t.Fatal("expected the first caller to win the claim")
	}

	var pi models.PaymentIntent
	if err := db.Where("id = ?", "pi-1").First(&pi).Error; err != nil {
		t.Fatalf("failed to reload payment intent: %v", err)
	}
	if !pi.IsFinalized {
		t.Fatal("expected is_finalized to be persisted as true")
	}
	if pi.Status != models.IntentSucceeded {
		t.Fatalf("expected status to be persisted as succeeded, got: %s", pi.Status)
	}
	if pi.PaidAt == nil {
		t.Fatal("expected paid_at to be persisted")
	}
}

func TestClaimPaymentIntentFinalization_SecondCallerLoses(t *testing.T) {
	db := setupPaymentIntentFinalizationTestDB(t)
	r := NewSubscriptionRepository(db)
	seedPaymentIntent(t, db, "pi-2", false)

	firstClaimed, err := r.ClaimPaymentIntentFinalization("pi-2", time.Now(), models.JSONMap{})
	if err != nil || !firstClaimed {
		t.Fatalf("expected the first claim to succeed, got claimed=%v err=%v", firstClaimed, err)
	}

	// Simulates a second, concurrent/replayed webhook delivery for the
	// exact same payment landing after the first has already finalized it.
	secondClaimed, err := r.ClaimPaymentIntentFinalization("pi-2", time.Now(), models.JSONMap{})
	if err != nil {
		t.Fatalf("expected no error on a already-finalized intent, got: %v", err)
	}
	if secondClaimed {
		t.Fatal("expected the second caller to lose the claim - double-finalization would re-run every activation side effect")
	}
}

func TestClaimPaymentIntentFinalization_AlreadyFinalizedNeverReClaimed(t *testing.T) {
	db := setupPaymentIntentFinalizationTestDB(t)
	r := NewSubscriptionRepository(db)
	seedPaymentIntent(t, db, "pi-3", true) // seeded as already finalized, e.g. by an earlier run

	claimed, err := r.ClaimPaymentIntentFinalization("pi-3", time.Now(), models.JSONMap{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if claimed {
		t.Fatal("expected an already-finalized intent to never be re-claimed")
	}
}
