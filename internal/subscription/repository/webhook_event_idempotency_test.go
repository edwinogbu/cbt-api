package repository

import (
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"cbt-api/internal/models"
)

// ---- CreateWebhookEvent: idempotency_key is the atomic dedup gate ----
//
// SubscriptionService.ProcessWebhook relies on webhook_events'
// uniqueIndex on idempotency_key to atomically resolve which of
// several deliveries of the same event (a gateway retry, or a replayed
// signature) gets to proceed. These tests prove that constraint itself
// holds - including under real concurrent access, not just a
// sequential second call - independent of how the service layer
// classifies the resulting error.

func setupWebhookEventTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE webhook_events (
		id TEXT PRIMARY KEY, gateway TEXT, event_type TEXT, reference TEXT,
		idempotency_key TEXT UNIQUE, status TEXT DEFAULT 'pending',
		retry_count INTEGER DEFAULT 0, processed_at DATETIME, last_retry_at DATETIME,
		error_message TEXT, payload TEXT, created_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func TestCreateWebhookEvent_DuplicateIdempotencyKeyRejected(t *testing.T) {
	db := setupWebhookEventTestDB(t)
	r := NewSubscriptionRepository(db)

	first := &models.WebhookEvent{
		ID: uuid.New().String(), Gateway: models.GatewayStripe, EventType: "payment_success",
		Reference: "ref-1", IdempotencyKey: "stripe:payment_success:ref-1", Status: "pending", CreatedAt: time.Now(),
	}
	if err := r.CreateWebhookEvent(first); err != nil {
		t.Fatalf("expected the first delivery to be recorded, got: %v", err)
	}

	second := &models.WebhookEvent{
		ID: uuid.New().String(), Gateway: models.GatewayStripe, EventType: "payment_success",
		Reference: "ref-1", IdempotencyKey: "stripe:payment_success:ref-1", Status: "pending", CreatedAt: time.Now(),
	}
	if err := r.CreateWebhookEvent(second); err == nil {
		t.Fatal("expected a duplicate idempotency_key to be rejected by the unique constraint, got no error")
	}
}

func TestCreateWebhookEvent_ConcurrentDuplicatesOnlyOneWins(t *testing.T) {
	db := setupWebhookEventTestDB(t)
	r := NewSubscriptionRepository(db)
	const idempotencyKey = "stripe:payment_success:concurrent-ref"
	const concurrency = 20

	var wg sync.WaitGroup
	results := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			results[n] = r.CreateWebhookEvent(&models.WebhookEvent{
				ID: uuid.New().String(), Gateway: models.GatewayStripe, EventType: "payment_success",
				Reference: "concurrent-ref", IdempotencyKey: idempotencyKey, Status: "pending", CreatedAt: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	succeeded := 0
	for _, err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly 1 of %d concurrent deliveries of the same event to win, got %d", concurrency, succeeded)
	}
}
