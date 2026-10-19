package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
	"gorm.io/gorm"

	"cbt-api/internal/models"
	"cbt-api/internal/subscription/repository"
	"cbt-api/pkg/payment"
)

// ---- SubscriptionService.ProcessWebhook: idempotency check ----
//
// Before this fix, CreateWebhookEvent's returned error was discarded
// unchecked - so a duplicate delivery of the same event (a gateway
// retrying on timeout, or a previously-valid signature replayed later)
// silently fell straight through the unique-constraint violation on
// idempotency_key and re-entered the payment_success branch a second
// time, relying only on an incidental `sub.Status == Pending` check to
// avoid reprocessing - a plain read-then-write with the same race a
// truly concurrent (not just sequential) duplicate delivery could
// still win. These tests drive the real, signature-verified path (same
// Stripe secret/signing helper used in pkg/payment/webhook_secret_test.go)
// to prove the exact same payload delivered twice is recorded once.

func setupWebhookIdempotencyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE subscriptions (
			id TEXT PRIMARY KEY, user_id TEXT, school_id TEXT NOT NULL,
			tier TEXT, status TEXT DEFAULT 'pending', gateway TEXT,
			gateway_customer_id TEXT, gateway_subscription_id TEXT,
			amount TEXT, currency TEXT DEFAULT 'NGN', payment_interval TEXT DEFAULT 'monthly',
			start_date DATETIME, end_date DATETIME, trial_ends_at DATETIME,
			last_payment_date DATETIME, next_payment_date DATETIME,
			auto_renew INTEGER DEFAULT 0, cancel_at_end_date INTEGER DEFAULT 0,
			max_students INTEGER DEFAULT 100, max_teachers INTEGER DEFAULT 10,
			max_exams INTEGER DEFAULT 50, max_questions INTEGER DEFAULT 500, max_storage_mb INTEGER DEFAULT 1024,
			current_invoice_id TEXT, features TEXT, metadata TEXT, gateway_data TEXT, email_notifications TEXT,
			created_by TEXT, updated_by TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE invoices (
			id TEXT PRIMARY KEY, subscription_id TEXT, school_id TEXT, user_id TEXT,
			invoice_number TEXT, amount TEXT, tax TEXT DEFAULT '0', discount TEXT DEFAULT '0',
			total TEXT, currency TEXT DEFAULT 'NGN', status TEXT DEFAULT 'pending',
			due_date DATETIME, paid_at DATETIME, pdf_url TEXT, items TEXT, metadata TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE payment_intents (id TEXT PRIMARY KEY, subscription_id TEXT, school_id TEXT, reference TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE payment_transactions (id TEXT PRIMARY KEY, subscription_id TEXT, school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		// webhook_events has no updated_at column in production either
		// (confirmed against the real Postgres schema) - UpdateWebhookEventStatus
		// writing one is a separate, pre-existing bug this test isn't
		// about, so it's included here only so that call doesn't block
		// these idempotency assertions.
		`CREATE TABLE webhook_events (
			id TEXT PRIMARY KEY, gateway TEXT, event_type TEXT, reference TEXT,
			idempotency_key TEXT UNIQUE, status TEXT DEFAULT 'pending',
			retry_count INTEGER DEFAULT 0, processed_at DATETIME, last_retry_at DATETIME,
			error_message TEXT, payload TEXT, created_at DATETIME, updated_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedWebhookTestSubscription(t *testing.T, db *gorm.DB, subID, piID, invoiceID, reference string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO subscriptions (id, school_id, tier, status, gateway, amount, payment_interval, start_date, end_date) VALUES (?, 'school-1', 'basic', 'pending', 'stripe', '29.99', 'monthly', datetime('now'), datetime('now', '+30 days'))",
		subID,
	).Error; err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO invoices (id, subscription_id, school_id, status) VALUES (?, ?, 'school-1', 'pending')",
		invoiceID, subID,
	).Error; err != nil {
		t.Fatalf("failed to seed invoice: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO payment_intents (id, subscription_id, school_id, reference) VALUES (?, ?, 'school-1', ?)",
		piID, subID, reference,
	).Error; err != nil {
		t.Fatalf("failed to seed payment intent: %v", err)
	}
}

// buildSignedStripeCheckoutWebhook builds a real, correctly-signed
// Stripe "checkout.session.completed" webhook delivery (payload + its
// Stripe-Signature header) for the given checkout session/reference,
// exactly as the real StripeGateway.ParseWebhook would need to verify it.
func buildSignedStripeCheckoutWebhook(t *testing.T, secret, sessionID string) (payload []byte, header string) {
	t.Helper()
	payload, err := json.Marshal(map[string]interface{}{
		"id":          "evt_" + sessionID,
		"type":        "checkout.session.completed",
		"api_version": stripe.APIVersion,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     sessionID,
				"object": "checkout.session",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build webhook payload: %v", err)
	}
	ts := time.Now()
	sig := webhook.ComputeSignature(ts, payload, secret)
	header = fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(sig))
	return payload, header
}

func TestProcessWebhook_DuplicateDeliveryRecordedOnce(t *testing.T) {
	secret := "whsec_test_idempotency_secret"
	t.Setenv("STRIPE_WEBHOOK_SECRET", secret)
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_idempotency")

	db := setupWebhookIdempotencyTestDB(t)
	const reference = "cs_test_idempotency_session"
	seedWebhookTestSubscription(t, db, "sub-1", "pi-1", "inv-1", reference)

	svc := NewSubscriptionService(repository.NewSubscriptionRepository(db), payment.NewPaymentService(), nil)
	payload, header := buildSignedStripeCheckoutWebhook(t, secret, reference)

	if err := svc.ProcessWebhook(context.Background(), models.GatewayStripe, payload, header); err != nil {
		t.Fatalf("expected the first delivery to process cleanly, got: %v", err)
	}

	var countAfterFirst int64
	db.Table("webhook_events").Where("reference = ?", reference).Count(&countAfterFirst)
	if countAfterFirst != 1 {
		t.Fatalf("expected exactly 1 webhook_events row after the first delivery, got %d", countAfterFirst)
	}

	// The DB's own unique constraint on idempotency_key already stops a
	// second webhook_events ROW from being inserted regardless of
	// whether the application checks CreateWebhookEvent's error - that
	// was never the gap. The actual bug was control flow: a discarded
	// error let execution fall straight through the constraint
	// violation into reprocessing the payment_success branch a second
	// time. In today's code that second pass happens to be a no-op
	// anyway, because sub.Status is no longer "pending" after the
	// first call - but that guard is itself a plain read-then-write
	// with the same TOCTOU race as the one closed in ActivateOnboarding
	// (see payment_intent_finalization_test.go / the ClaimPaymentIntentFinalization
	// commit): two deliveries arriving concurrently, not sequentially,
	// can both observe "pending" before either commits. Asserting on
	// sub.Status/updated_at here would pass or fail by accident of that
	// other guard, not on what this fix changed - so assert on control
	// flow instead: capture the package's log output and check the
	// payment_success branch's own log line ran exactly once.
	prevLogOutput := log.Writer()
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(prevLogOutput)

	// The exact same event, delivered again - a gateway retry, or a
	// previously-valid signature replayed later (the payload and
	// signature are byte-for-byte identical to the first delivery).
	if err := svc.ProcessWebhook(context.Background(), models.GatewayStripe, payload, header); err != nil {
		t.Fatalf("expected a duplicate delivery to be handled as a no-op, got error: %v", err)
	}
	log.SetOutput(prevLogOutput)

	output := logBuf.String()
	if !strings.Contains(output, "Duplicate delivery") {
		t.Fatalf("expected the duplicate to be logged as such, got: %s", output)
	}
	if strings.Contains(output, "Payment success for reference") {
		t.Fatalf("expected the duplicate delivery to be skipped before re-entering the payment_success branch, got: %s", output)
	}

	var countAfterSecond int64
	db.Table("webhook_events").Where("reference = ?", reference).Count(&countAfterSecond)
	if countAfterSecond != 1 {
		t.Fatalf("expected the duplicate delivery to NOT create a second webhook_events row, got %d total", countAfterSecond)
	}
}

func TestProcessWebhook_DifferentReferencesBothRecorded(t *testing.T) {
	secret := "whsec_test_idempotency_secret_2"
	t.Setenv("STRIPE_WEBHOOK_SECRET", secret)
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_idempotency_2")

	db := setupWebhookIdempotencyTestDB(t)
	seedWebhookTestSubscription(t, db, "sub-a", "pi-a", "inv-a", "cs_test_ref_a")
	seedWebhookTestSubscription(t, db, "sub-b", "pi-b", "inv-b", "cs_test_ref_b")

	svc := NewSubscriptionService(repository.NewSubscriptionRepository(db), payment.NewPaymentService(), nil)

	payloadA, headerA := buildSignedStripeCheckoutWebhook(t, secret, "cs_test_ref_a")
	payloadB, headerB := buildSignedStripeCheckoutWebhook(t, secret, "cs_test_ref_b")

	if err := svc.ProcessWebhook(context.Background(), models.GatewayStripe, payloadA, headerA); err != nil {
		t.Fatalf("expected the first event to process cleanly, got: %v", err)
	}
	if err := svc.ProcessWebhook(context.Background(), models.GatewayStripe, payloadB, headerB); err != nil {
		t.Fatalf("expected the second, genuinely different event to process cleanly, got: %v", err)
	}

	var totalEvents int64
	db.Table("webhook_events").Count(&totalEvents)
	if totalEvents != 2 {
		t.Fatalf("expected two distinct events to both be recorded, got %d", totalEvents)
	}
}
