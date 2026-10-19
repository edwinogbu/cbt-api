package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"cbt-api/internal/subscription/dto"
	"cbt-api/internal/subscription/repository"
)

// ---- Phase 2: subscription-module tenant isolation ----
//
// Before this fix: GetSubscription/UpdateSubscription/CancelSubscription/
// RenewSubscription/CreatePaymentIntent took no school parameter at all -
// any authenticated user could read, modify, cancel, renew, or generate
// a real payment link against ANY other school's subscription, not just
// read its billing data.

func setupSubscriptionTenantTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE payment_intents (id TEXT PRIMARY KEY, subscription_id TEXT, school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE payment_transactions (
			id TEXT PRIMARY KEY, subscription_id TEXT, payment_intent_id TEXT, invoice_id TEXT,
			school_id TEXT, user_id TEXT,
			amount TEXT, fee TEXT DEFAULT '0', tax TEXT DEFAULT '0', currency TEXT DEFAULT 'NGN',
			payment_method TEXT, payment_status TEXT DEFAULT 'pending', gateway TEXT,
			reference TEXT, transaction_id TEXT, gateway_transaction_id TEXT, description TEXT,
			paid_at DATETIME, is_locked INTEGER DEFAULT 0, metadata TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTenantTestSubscriptionService(db *gorm.DB) *SubscriptionService {
	return NewSubscriptionService(repository.NewSubscriptionRepository(db), nil, nil)
}

func seedSubscriptionForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO subscriptions (id, school_id, tier, status, gateway, amount, currency, payment_interval, start_date, end_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now', '+30 days'))",
		id, schoolID, "premium", "active", "paystack", "50000", "NGN", "monthly",
	).Error; err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}
}

func TestSubscriptionGet_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.GetSubscription("sub-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's subscription, got no error")
	}
}

func TestSubscriptionGet_SameTenant_Allowed(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	resp, err := s.GetSubscription("sub-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own subscription, got error: %v", err)
	}
	if resp.ID != "sub-1" {
		t.Fatalf("expected sub-1, got %s", resp.ID)
	}
}

func TestSubscriptionGet_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.GetSubscription("sub-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's subscription, got error: %v", err)
	}
}

func TestSubscriptionGet_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.GetSubscription("sub-1", "", false); err == nil {
		t.Fatal("expected an empty school_id on a non-platform-owner caller to be rejected, got no error")
	}
}

func TestSubscriptionUpdate_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	newTier := "enterprise"

	if _, err := s.UpdateSubscription("sub-1", &dto.UpdateSubscriptionRequest{Tier: &newTier}, "attacker-user", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected updating School A's subscription, got no error")
	}

	var tier string
	db.Table("subscriptions").Select("tier").Where("id = ?", "sub-1").Scan(&tier)
	if tier == newTier {
		t.Fatal("subscription tier was changed by a cross-tenant update that should have been rejected")
	}
}

// TestSubscriptionCancel_CrossTenant_Rejected is the sabotage scenario
// this fix exists for: before it, any authenticated user could cancel
// another school's active subscription outright.
func TestSubscriptionCancel_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if err := s.CancelSubscription("sub-1", true, "competitor sabotage", "attacker-user", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected cancelling School A's subscription, got no error")
	}

	var status string
	db.Table("subscriptions").Select("status").Where("id = ?", "sub-1").Scan(&status)
	if status != "active" {
		t.Fatalf("subscription was cancelled by a cross-tenant request that should have been rejected, status=%q", status)
	}
}

func TestSubscriptionCancel_SameTenant_Allowed(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if err := s.CancelSubscription("sub-1", true, "no longer needed", "owner-user", "school-A", false); err != nil {
		t.Fatalf("expected School A to cancel its own subscription, got error: %v", err)
	}
}

func TestSubscriptionRenew_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.RenewSubscription("sub-1", &dto.RenewSubscriptionRequest{Interval: "monthly"}, "attacker-user", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected renewing School A's subscription, got no error")
	}
}

// ---- GetInvoices / GetTransactions: were previously handler-level
// stubs that always returned an empty list for any subscription ID,
// so there was never a cross-tenant leak to exercise - but also never
// a school's own real billing data. Now that they're wired to the
// real, already-existing FindInvoicesBySubscription/
// FindPaymentTransactionsBySubscription repository queries, they need
// the same ownership check as every other subscription-scoped
// endpoint in this file.

func seedInvoiceForSchool(t *testing.T, db *gorm.DB, id, subID, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO invoices (id, subscription_id, school_id, invoice_number, amount, total, status, due_date) VALUES (?, ?, ?, ?, '50000', '50000', 'pending', datetime('now', '+7 days'))",
		id, subID, schoolID, "INV-"+id,
	).Error; err != nil {
		t.Fatalf("failed to seed invoice: %v", err)
	}
}

func seedPaymentTransactionForSchool(t *testing.T, db *gorm.DB, id, subID, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO payment_transactions (id, subscription_id, school_id, amount, currency, payment_status, gateway, reference) VALUES (?, ?, ?, '50000', 'NGN', 'paid', 'paystack', ?)",
		id, subID, schoolID, "ref-"+id,
	).Error; err != nil {
		t.Fatalf("failed to seed payment transaction: %v", err)
	}
}

func TestSubscriptionGetInvoices_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedInvoiceForSchool(t, db, "inv-1", "sub-1", "school-A")

	if _, err := s.GetInvoices("sub-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's invoices, got no error")
	}
}

func TestSubscriptionGetInvoices_SameTenant_Allowed(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedInvoiceForSchool(t, db, "inv-1", "sub-1", "school-A")
	seedInvoiceForSchool(t, db, "inv-2", "sub-1", "school-A")

	invoices, err := s.GetInvoices("sub-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own invoices, got error: %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("expected 2 invoices for sub-1, got %d", len(invoices))
	}
}

func TestSubscriptionGetInvoices_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedInvoiceForSchool(t, db, "inv-1", "sub-1", "school-A")

	if _, err := s.GetInvoices("sub-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's invoices, got error: %v", err)
	}
}

func TestSubscriptionGetTransactions_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentTransactionForSchool(t, db, "txn-1", "sub-1", "school-A")

	if _, err := s.GetTransactions("sub-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's transactions, got no error")
	}
}

func TestSubscriptionGetTransactions_SameTenant_Allowed(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentTransactionForSchool(t, db, "txn-1", "sub-1", "school-A")
	seedPaymentTransactionForSchool(t, db, "txn-2", "sub-1", "school-A")

	transactions, err := s.GetTransactions("sub-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own transactions, got error: %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("expected 2 transactions for sub-1, got %d", len(transactions))
	}
}

func TestSubscriptionGetTransactions_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentTransactionForSchool(t, db, "txn-1", "sub-1", "school-A")

	if _, err := s.GetTransactions("sub-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's transactions, got error: %v", err)
	}
}
