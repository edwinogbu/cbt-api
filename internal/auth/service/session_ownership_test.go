package service

import (
	"testing"
	"time"

	"cbt-api/internal/auth/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- RevokeSession/Logout: session ownership ----
//
// Before this fix, both Logout and RevokeSession took a userID parameter
// they never used, and deleted a session purely by sessionID via
// DeleteSession(sessionID). RevokeSession's sessionID comes straight from
// the URL path (c.Param("sessionId")) - client input - so any
// authenticated user who obtained another user's session ID could
// force-logout that user's device, regardless of whose session it
// actually was. Logout's sessionID is always server-derived from the
// caller's own JWT, so it wasn't independently exploitable, but the
// missing check was still a latent bug in a method whose whole signature
// implies an ownership boundary.

func setupSessionOwnershipTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := `CREATE TABLE user_sessions (
		id TEXT PRIMARY KEY, user_id TEXT, refresh_token TEXT,
		user_agent TEXT, client_ip TEXT, expires_at DATETIME, created_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	return db
}

func seedSession(t *testing.T, db *gorm.DB, id, userID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO user_sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		id, userID, time.Now().Add(time.Hour), time.Now(),
	).Error; err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
}

func sessionExists(t *testing.T, db *gorm.DB, id string) bool {
	t.Helper()
	var count int64
	if err := db.Table("user_sessions").Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("failed to count session: %v", err)
	}
	return count > 0
}

func TestRevokeSession_OtherUsersSession_Rejected(t *testing.T) {
	db := setupSessionOwnershipTestDB(t)
	seedSession(t, db, "session-victim", "victim-user")

	s := &AuthService{repo: repository.NewAuthRepository(db)}
	err := s.RevokeSession("attacker-user", "session-victim")
	if err == nil {
		t.Fatal("expected an attacker to be rejected when revoking another user's session")
	}
	if !sessionExists(t, db, "session-victim") {
		t.Fatal("victim's session was deleted by a non-owner caller")
	}
}

func TestRevokeSession_OwnSession_Succeeds(t *testing.T) {
	db := setupSessionOwnershipTestDB(t)
	seedSession(t, db, "session-mine", "owner-user")

	s := &AuthService{repo: repository.NewAuthRepository(db)}
	if err := s.RevokeSession("owner-user", "session-mine"); err != nil {
		t.Fatalf("expected the owner to revoke their own session, got error: %v", err)
	}
	if sessionExists(t, db, "session-mine") {
		t.Fatal("expected the session to be deleted")
	}
}

func TestLogout_AlreadyGoneSession_StaysIdempotent(t *testing.T) {
	db := setupSessionOwnershipTestDB(t)

	s := &AuthService{repo: repository.NewAuthRepository(db)}
	if err := s.Logout("owner-user", "session-already-deleted"); err != nil {
		t.Fatalf("expected logout of an already-gone session to succeed (idempotent), got error: %v", err)
	}
}
