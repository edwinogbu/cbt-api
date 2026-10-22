package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"cbt-api/internal/subscription/dto"
	"cbt-api/internal/subscription/repository"
	"cbt-api/pkg/payment"
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
		`CREATE TABLE payment_intents (
			id TEXT PRIMARY KEY, subscription_id TEXT, invoice_id TEXT, school_id TEXT, user_id TEXT,
			idempotency_key TEXT, gateway TEXT, amount TEXT, reference TEXT,
			status TEXT DEFAULT 'pending', is_finalized INTEGER DEFAULT 0,
			gateway_response TEXT, paid_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE payment_transactions (
			id TEXT PRIMARY KEY, subscription_id TEXT, payment_intent_id TEXT, invoice_id TEXT,
			school_id TEXT, user_id TEXT,
			amount TEXT, fee TEXT DEFAULT '0', tax TEXT DEFAULT '0', currency TEXT DEFAULT 'NGN',
			payment_method TEXT, payment_status TEXT DEFAULT 'pending', gateway TEXT,
			reference TEXT, transaction_id TEXT, gateway_transaction_id TEXT, description TEXT,
			paid_at DATETIME, is_locked INTEGER DEFAULT 0, metadata TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE students (id TEXT PRIMARY KEY, user_id TEXT, school_id TEXT, class_id TEXT, admission_no TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, school_id TEXT, role TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE classes (id TEXT PRIMARY KEY, school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE exams (id TEXT PRIMARY KEY, class_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE question_bank (id TEXT PRIMARY KEY, school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
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

// newTenantTestSubscriptionServiceWithPayment is for tests that need to
// get past ConfirmPaymentIntent's ownership gate into the gateway
// verification call it makes next - a nil paymentService would panic
// there, unlike every other method in this file, which returns before
// ever touching it on the cross-tenant-rejected path.
func newTenantTestSubscriptionServiceWithPayment(db *gorm.DB) *SubscriptionService {
	return NewSubscriptionService(repository.NewSubscriptionRepository(db), payment.NewPaymentService(), nil)
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

// ---- GetSubscriptionUsage: was a handler-level stub always returning
// nil, so - same as GetInvoices/GetTransactions before their fix -
// there was never a cross-tenant leak, just never any real data
// either. Now wired to real COUNT queries across the academic (students,
// via users for teachers) and cbt (exams, question_bank) tables, it
// needs the same ownership check as every other subscription-scoped
// endpoint.

func seedStudentForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO students (id, school_id, admission_no) VALUES (?, ?, ?)", id, schoolID, "ADM-"+id).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func seedTeacherForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, school_id, role) VALUES (?, ?, 'teacher')", id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed teacher: %v", err)
	}
}

func seedClassForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO classes (id, school_id) VALUES (?, ?)", id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedExamForClass(t *testing.T, db *gorm.DB, id, classID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO exams (id, class_id) VALUES (?, ?)", id, classID).Error; err != nil {
		t.Fatalf("failed to seed exam: %v", err)
	}
}

func seedQuestionForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO question_bank (id, school_id) VALUES (?, ?)", id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed question: %v", err)
	}
}

func TestSubscriptionGetUsage_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.GetSubscriptionUsage("sub-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's usage, got no error")
	}
}

func TestSubscriptionGetUsage_SameTenant_ReportsRealCounts(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	seedStudentForSchool(t, db, "stu-1", "school-A")
	seedStudentForSchool(t, db, "stu-2", "school-A")
	seedStudentForSchool(t, db, "stu-3", "school-B") // a different school's student must not be counted

	seedTeacherForSchool(t, db, "tea-1", "school-A")
	seedTeacherForSchool(t, db, "tea-2", "school-B") // must not be counted

	seedClassForSchool(t, db, "cls-1", "school-A")
	seedClassForSchool(t, db, "cls-2", "school-B")
	seedExamForClass(t, db, "exm-1", "cls-1")
	seedExamForClass(t, db, "exm-2", "cls-1")
	seedExamForClass(t, db, "exm-3", "cls-2") // a different school's exam (via its class) must not be counted

	seedQuestionForSchool(t, db, "q-1", "school-A")
	seedQuestionForSchool(t, db, "q-2", "school-B") // must not be counted

	usage, err := s.GetSubscriptionUsage("sub-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own usage, got error: %v", err)
	}
	if usage.CurrentStudents != 2 {
		t.Fatalf("expected 2 students for school-A, got %d", usage.CurrentStudents)
	}
	if usage.CurrentTeachers != 1 {
		t.Fatalf("expected 1 teacher for school-A, got %d", usage.CurrentTeachers)
	}
	if usage.CurrentExams != 2 {
		t.Fatalf("expected 2 exams for school-A, got %d", usage.CurrentExams)
	}
	if usage.CurrentQuestions != 1 {
		t.Fatalf("expected 1 question for school-A, got %d", usage.CurrentQuestions)
	}
	// seedSubscriptionForSchool relies on the schema's DEFAULT max_students=100
	if usage.MaxStudents != 100 {
		t.Fatalf("expected default max_students=100, got %d", usage.MaxStudents)
	}
	if usage.StudentPercent != 2 {
		t.Fatalf("expected student_percent=2 (2/100), got %v", usage.StudentPercent)
	}
	if usage.StorageUsedMB != 0 || usage.StoragePercent != 0 {
		t.Fatalf("expected storage to stay 0 (not tracked), got used=%d percent=%v", usage.StorageUsedMB, usage.StoragePercent)
	}
}

func TestSubscriptionGetUsage_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")

	if _, err := s.GetSubscriptionUsage("sub-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's usage, got error: %v", err)
	}
}

// ---- ConfirmPaymentIntent: before this fix, took no school parameter
// at all - any authenticated admin/super_admin could force this
// manual gateway re-verification/activation flow against ANY other
// school's payment intent just by knowing or guessing its UUID.

func seedPaymentIntentForSchool(t *testing.T, db *gorm.DB, id, subID, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO payment_intents (id, subscription_id, school_id, idempotency_key, gateway, amount, reference, status, is_finalized) VALUES (?, ?, ?, ?, 'unregistered-test-gateway', '50000', ?, 'pending', 0)",
		id, subID, schoolID, id+"-idem", "ref-"+id,
	).Error; err != nil {
		t.Fatalf("failed to seed payment intent: %v", err)
	}
}

func TestConfirmPaymentIntent_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.ConfirmPaymentIntent("pi-1", "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected confirming School A's payment intent, got no error")
	}
	if err.Error() != "payment intent not found" {
		t.Fatalf(`expected "payment intent not found", got: %v`, err)
	}

	var isFinalized bool
	db.Table("payment_intents").Select("is_finalized").Where("id = ?", "pi-1").Scan(&isFinalized)
	if isFinalized {
		t.Fatal("payment intent was finalized by a cross-tenant request that should have been rejected")
	}
}

func TestConfirmPaymentIntent_SameTenant_PassesOwnershipGate(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionServiceWithPayment(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.ConfirmPaymentIntent("pi-1", "school-A", false)
	if err == nil {
		t.Fatal("expected an error past the ownership gate (unsupported gateway), got nil")
	}
	if err.Error() == "payment intent not found" {
		t.Fatal("same-tenant caller was incorrectly rejected by the ownership gate")
	}
	if err.Error() != "unsupported gateway: unregistered-test-gateway" {
		t.Fatalf("expected the ownership gate to be passed and fail at gateway verification instead, got: %v", err)
	}
}

func TestConfirmPaymentIntent_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionServiceWithPayment(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.ConfirmPaymentIntent("pi-1", "", true)
	if err == nil || err.Error() == "payment intent not found" {
		t.Fatalf("expected platform_owner to pass the ownership gate (and fail at gateway verification instead), got: %v", err)
	}
}

// ---- VerifyPayment: before this fix, this was a complete stub - the
// handler returned a hardcoded "Payment verified successfully" message
// unconditionally, with no DB lookup, no ownership check, and no real
// gateway call. Mirrors the ConfirmPaymentIntent trio exactly, but keyed
// by gateway reference instead of payment intent ID.

func TestVerifyPayment_CrossTenant_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionService(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.VerifyPayment("ref-pi-1", "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected verifying School A's payment intent, got no error")
	}
	if err.Error() != "payment intent not found" {
		t.Fatalf(`expected "payment intent not found", got: %v`, err)
	}

	var isFinalized bool
	db.Table("payment_intents").Select("is_finalized").Where("id = ?", "pi-1").Scan(&isFinalized)
	if isFinalized {
		t.Fatal("payment intent was finalized by a cross-tenant request that should have been rejected")
	}
}

func TestVerifyPayment_SameTenant_PassesOwnershipGate(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionServiceWithPayment(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.VerifyPayment("ref-pi-1", "school-A", false)
	if err == nil {
		t.Fatal("expected an error past the ownership gate (unsupported gateway), got nil")
	}
	if err.Error() == "payment intent not found" {
		t.Fatal("same-tenant caller was incorrectly rejected by the ownership gate")
	}
	if err.Error() != "unsupported gateway: unregistered-test-gateway" {
		t.Fatalf("expected the ownership gate to be passed and fail at gateway verification instead, got: %v", err)
	}
}

func TestVerifyPayment_PlatformOwner_Bypasses(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionServiceWithPayment(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")

	err := s.VerifyPayment("ref-pi-1", "", true)
	if err == nil || err.Error() == "payment intent not found" {
		t.Fatalf("expected platform_owner to pass the ownership gate (and fail at gateway verification instead), got: %v", err)
	}
}

func TestVerifyPayment_AlreadyFinalized_Rejected(t *testing.T) {
	db := setupSubscriptionTenantTestDB(t)
	s := newTenantTestSubscriptionServiceWithPayment(db)
	seedSubscriptionForSchool(t, db, "sub-1", "school-A")
	seedPaymentIntentForSchool(t, db, "pi-1", "sub-1", "school-A")
	if err := db.Exec("UPDATE payment_intents SET is_finalized = 1 WHERE id = ?", "pi-1").Error; err != nil {
		t.Fatalf("failed to mark payment intent finalized: %v", err)
	}

	err := s.VerifyPayment("ref-pi-1", "school-A", false)
	if err == nil || err.Error() != "payment intent already finalized" {
		t.Fatalf(`expected "payment intent already finalized", got: %v`, err)
	}
}
