package repository

import (
	"errors"
	"time"

	"cbt-api/internal/models"
	"gorm.io/gorm"
)

type OnboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

// ============================================
// SESSION OPERATIONS
// ============================================

// CreateSession creates a new onboarding session
func (r *OnboardingRepository) CreateSession(session *models.OnboardingSession) error {
	return r.db.Create(session).Error
}

// FindSessionByToken finds a session by its token
func (r *OnboardingRepository) FindSessionByToken(token string) (*models.OnboardingSession, error) {
	var session models.OnboardingSession
	err := r.db.Where("session_token = ? AND deleted_at IS NULL", token).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("session expired")
	}
	return &session, nil
}

// FindSessionByID finds a session by its ID
func (r *OnboardingRepository) FindSessionByID(id string) (*models.OnboardingSession, error) {
	var session models.OnboardingSession
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("session expired")
	}
	return &session, nil
}

// UpdateSession updates an existing session
func (r *OnboardingRepository) UpdateSession(session *models.OnboardingSession) error {
	session.UpdatedAt = time.Now()
	return r.db.Save(session).Error
}

// DeleteSession soft deletes a session
func (r *OnboardingRepository) DeleteSession(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
}

// HardDeleteSession permanently deletes a session
func (r *OnboardingRepository) HardDeleteSession(id string) error {
	return r.db.Unscoped().Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
}

// FindSessionByUserID finds a session by user ID
func (r *OnboardingRepository) FindSessionByUserID(userID string) (*models.OnboardingSession, error) {
	var session models.OnboardingSession
	err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindSessionBySchoolID finds a session by school ID
func (r *OnboardingRepository) FindSessionBySchoolID(schoolID string) (*models.OnboardingSession, error) {
	var session models.OnboardingSession
	err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
		Order("created_at DESC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindSessionBySubscriptionID finds a session by subscription ID
func (r *OnboardingRepository) FindSessionBySubscriptionID(subscriptionID string) (*models.OnboardingSession, error) {
	var session models.OnboardingSession
	err := r.db.Where("subscription_id = ? AND deleted_at IS NULL", subscriptionID).
		Order("created_at DESC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindSessionsByStatus finds all sessions with a given status
func (r *OnboardingRepository) FindSessionsByStatus(status models.OnboardingStatus) ([]models.OnboardingSession, error) {
	var sessions []models.OnboardingSession
	err := r.db.Where("status = ? AND deleted_at IS NULL", status).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// FindPendingSessions finds all sessions that are not complete or expired
func (r *OnboardingRepository) FindPendingSessions() ([]models.OnboardingSession, error) {
	var sessions []models.OnboardingSession
	err := r.db.Where("status != ? AND status != ? AND expires_at > ? AND deleted_at IS NULL",
		models.OnboardingStatusComplete,
		models.OnboardingStatusExpired,
		time.Now()).
		Order("created_at ASC").
		Find(&sessions).Error
	return sessions, err
}

// ============================================
// CLEANUP OPERATIONS
// ============================================

// DeleteExpiredSessions deletes all expired sessions
func (r *OnboardingRepository) DeleteExpiredSessions() error {
	return r.db.Where("expires_at < ? AND status != ? AND deleted_at IS NULL",
		time.Now(),
		models.OnboardingStatusComplete).
		Delete(&models.OnboardingSession{}).Error
}

// MarkExpiredSessions marks all expired sessions as EXPIRED
func (r *OnboardingRepository) MarkExpiredSessions() error {
	return r.db.Model(&models.OnboardingSession{}).
		Where("expires_at < ? AND status != ? AND status != ? AND deleted_at IS NULL",
			time.Now(),
			models.OnboardingStatusComplete,
			models.OnboardingStatusExpired).
		Update("status", models.OnboardingStatusExpired).Error
}

// CountActiveSessions counts active (non-expired, non-complete) sessions
func (r *OnboardingRepository) CountActiveSessions() (int64, error) {
	var count int64
	err := r.db.Model(&models.OnboardingSession{}).
		Where("expires_at > ? AND status != ? AND status != ? AND deleted_at IS NULL",
			time.Now(),
			models.OnboardingStatusComplete,
			models.OnboardingStatusExpired).
		Count(&count).Error
	return count, err
}

// ============================================
// TRANSACTION OPERATIONS
// ============================================

// BeginTx starts a new transaction
func (r *OnboardingRepository) BeginTx() *gorm.DB {
	return r.db.Begin()
}

// CommitTx commits a transaction
func (r *OnboardingRepository) CommitTx(tx *gorm.DB) error {
	return tx.Commit().Error
}

// RollbackTx rolls back a transaction
func (r *OnboardingRepository) RollbackTx(tx *gorm.DB) error {
	return tx.Rollback().Error
}

// ============================================
// BULK OPERATIONS
// ============================================

// DeleteAllSessionsByUserID deletes all sessions for a user
func (r *OnboardingRepository) DeleteAllSessionsByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.OnboardingSession{}).Error
}

// DeleteAllSessionsBySchoolID deletes all sessions for a school
func (r *OnboardingRepository) DeleteAllSessionsBySchoolID(schoolID string) error {
	return r.db.Where("school_id = ?", schoolID).Delete(&models.OnboardingSession{}).Error
}

// ============================================
// HELPER CHECKS
// ============================================

// SessionExists checks if a session exists by token
func (r *OnboardingRepository) SessionExists(token string) (bool, error) {
	var count int64
	err := r.db.Model(&models.OnboardingSession{}).
		Where("session_token = ? AND deleted_at IS NULL", token).
		Count(&count).Error
	return count > 0, err
}

// IsSessionValid checks if a session is valid (exists and not expired)
func (r *OnboardingRepository) IsSessionValid(token string) (bool, error) {
	var count int64
	err := r.db.Model(&models.OnboardingSession{}).
		Where("session_token = ? AND expires_at > ? AND deleted_at IS NULL",
			token, time.Now()).
		Count(&count).Error
	return count > 0, err
}

// GetSessionProgress gets the current step/status of a session
func (r *OnboardingRepository) GetSessionProgress(token string) (models.OnboardingStatus, error) {
	var session models.OnboardingSession
	err := r.db.Select("status").Where("session_token = ? AND deleted_at IS NULL", token).
		First(&session).Error
	if err != nil {
		return "", err
	}
	return session.Status, nil
}

// ============================================
// ONBOARDING USER OPERATIONS (Self-contained)
// ============================================

// CreateOnboardUser creates a user specifically for onboarding
func (r *OnboardingRepository) CreateOnboardUser(user *models.User) error {
	return r.db.Create(user).Error
}

// FindOnboardUserByEmail finds a user by email
func (r *OnboardingRepository) FindOnboardUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindOnboardUserByID finds a user by ID
func (r *OnboardingRepository) FindOnboardUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindOnboardUserByUsername finds a user by username
func (r *OnboardingRepository) FindOnboardUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateOnboardUser updates a user
func (r *OnboardingRepository) UpdateOnboardUser(user *models.User) error {
	return r.db.Save(user).Error
}

// VerifyOnboardUserEmail verifies a user's email
func (r *OnboardingRepository) VerifyOnboardUserEmail(userID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified": true,
			"status":         "active",
		}).Error
}

// ============================================
// ONBOARDING OTP OPERATIONS (Self-contained)
// ============================================

// CreateOnboardOTP creates a new OTP record
func (r *OnboardingRepository) CreateOnboardOTP(otp *models.OTP) error {
	return r.db.Create(otp).Error
}

// FindValidOnboardOTP finds a valid (unused, not expired) OTP
func (r *OnboardingRepository) FindValidOnboardOTP(userID, code, otpType string) (*models.OTP, error) {
	var otp models.OTP
	err := r.db.Where("user_id = ? AND type = ? AND expires_at > ? AND used = false",
		userID, otpType, time.Now()).First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// MarkOnboardOTPAsUsed marks an OTP as used
func (r *OnboardingRepository) MarkOnboardOTPAsUsed(otpID uint) error {
	return r.db.Model(&models.OTP{}).Where("id = ?", otpID).Update("used", true).Error
}

// InvalidateOnboardUserOTPs invalidates all OTPs for a user
func (r *OnboardingRepository) InvalidateOnboardUserOTPs(userID, otpType string) error {
	return r.db.Model(&models.OTP{}).
		Where("user_id = ? AND type = ? AND used = ?", userID, otpType, false).
		Update("used", true).Error
}

// VerifyOnboardEmailWithOTP verifies email with OTP code
func (r *OnboardingRepository) VerifyOnboardEmailWithOTP(userID, code string) error {
	var otp models.OTP
	err := r.db.Where("user_id = ? AND type = ? AND used = ? AND expires_at > ?",
		userID, "email_verification", false, time.Now()).
		First(&otp).Error
	if err != nil {
		return errors.New("invalid or expired verification code")
	}

	if err := r.db.Model(&otp).Update("used", true).Error; err != nil {
		return err
	}

	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified": true,
			"status":         "active",
		}).Error
}

// ============================================
// ONBOARDING SCHOOL OPERATIONS (Self-contained)
// ============================================

// CreateOnboardSchool creates a school
func (r *OnboardingRepository) CreateOnboardSchool(school *models.School) error {
	return r.db.Create(school).Error
}

// FindOnboardSchoolByID finds a school by ID
func (r *OnboardingRepository) FindOnboardSchoolByID(id string) (*models.School, error) {
	var school models.School
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// FindOnboardSchoolByName finds a school by name
func (r *OnboardingRepository) FindOnboardSchoolByName(name string) (*models.School, error) {
	var school models.School
	err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// FindOnboardSchoolByCode finds a school by code
func (r *OnboardingRepository) FindOnboardSchoolByCode(code string) (*models.School, error) {
	var school models.School
	err := r.db.Where("code = ? AND deleted_at IS NULL", code).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// UpdateOnboardSchool updates a school
func (r *OnboardingRepository) UpdateOnboardSchool(school *models.School) error {
	return r.db.Save(school).Error
}


// package repository

// import (
// 	"errors"
// 	"time"

// 	"cbt-api/internal/models"
// 	"gorm.io/gorm"
// )

// type OnboardingRepository struct {
// 	db *gorm.DB
// }

// func NewOnboardingRepository(db *gorm.DB) *OnboardingRepository {
// 	return &OnboardingRepository{db: db}
// }

// // ============================================
// // SESSION OPERATIONS
// // ============================================

// // CreateSession creates a new onboarding session
// func (r *OnboardingRepository) CreateSession(session *models.OnboardingSession) error {
// 	return r.db.Create(session).Error
// }

// // FindSessionByToken finds a session by its token
// func (r *OnboardingRepository) FindSessionByToken(token string) (*models.OnboardingSession, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Where("session_token = ? AND deleted_at IS NULL", token).
// 		First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Check if expired
// 	if session.ExpiresAt.Before(time.Now()) {
// 		return nil, errors.New("session expired")
// 	}

// 	return &session, nil
// }

// // FindSessionByID finds a session by its ID
// func (r *OnboardingRepository) FindSessionByID(id string) (*models.OnboardingSession, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
// 		First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}

// 	if session.ExpiresAt.Before(time.Now()) {
// 		return nil, errors.New("session expired")
// 	}

// 	return &session, nil
// }

// // UpdateSession updates an existing session
// func (r *OnboardingRepository) UpdateSession(session *models.OnboardingSession) error {
// 	session.UpdatedAt = time.Now()
// 	return r.db.Save(session).Error
// }

// // DeleteSession soft deletes a session
// func (r *OnboardingRepository) DeleteSession(id string) error {
// 	return r.db.Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
// }

// // HardDeleteSession permanently deletes a session
// func (r *OnboardingRepository) HardDeleteSession(id string) error {
// 	return r.db.Unscoped().Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
// }

// // FindSessionByUserID finds a session by user ID
// func (r *OnboardingRepository) FindSessionByUserID(userID string) (*models.OnboardingSession, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).
// 		Order("created_at DESC").
// 		First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &session, nil
// }

// // FindSessionBySchoolID finds a session by school ID
// func (r *OnboardingRepository) FindSessionBySchoolID(schoolID string) (*models.OnboardingSession, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
// 		Order("created_at DESC").
// 		First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &session, nil
// }

// // FindSessionBySubscriptionID finds a session by subscription ID
// func (r *OnboardingRepository) FindSessionBySubscriptionID(subscriptionID string) (*models.OnboardingSession, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Where("subscription_id = ? AND deleted_at IS NULL", subscriptionID).
// 		Order("created_at DESC").
// 		First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &session, nil
// }

// // FindSessionsByStatus finds all sessions with a given status
// func (r *OnboardingRepository) FindSessionsByStatus(status models.OnboardingStatus) ([]models.OnboardingSession, error) {
// 	var sessions []models.OnboardingSession
// 	err := r.db.Where("status = ? AND deleted_at IS NULL", status).
// 		Order("created_at DESC").
// 		Find(&sessions).Error
// 	return sessions, err
// }

// // FindPendingSessions finds all sessions that are not complete or expired
// func (r *OnboardingRepository) FindPendingSessions() ([]models.OnboardingSession, error) {
// 	var sessions []models.OnboardingSession
// 	err := r.db.Where("status != ? AND status != ? AND expires_at > ? AND deleted_at IS NULL",
// 		models.OnboardingStatusComplete,
// 		models.OnboardingStatusExpired,
// 		time.Now()).
// 		Order("created_at ASC").
// 		Find(&sessions).Error
// 	return sessions, err
// }

// // ============================================
// // CLEANUP OPERATIONS
// // ============================================

// // DeleteExpiredSessions deletes all expired sessions
// func (r *OnboardingRepository) DeleteExpiredSessions() error {
// 	return r.db.Where("expires_at < ? AND status != ? AND deleted_at IS NULL",
// 		time.Now(),
// 		models.OnboardingStatusComplete).
// 		Delete(&models.OnboardingSession{}).Error
// }

// // MarkExpiredSessions marks all expired sessions as EXPIRED
// func (r *OnboardingRepository) MarkExpiredSessions() error {
// 	return r.db.Model(&models.OnboardingSession{}).
// 		Where("expires_at < ? AND status != ? AND status != ? AND deleted_at IS NULL",
// 			time.Now(),
// 			models.OnboardingStatusComplete,
// 			models.OnboardingStatusExpired).
// 		Update("status", models.OnboardingStatusExpired).Error
// }

// // CountActiveSessions counts active (non-expired, non-complete) sessions
// func (r *OnboardingRepository) CountActiveSessions() (int64, error) {
// 	var count int64
// 	err := r.db.Model(&models.OnboardingSession{}).
// 		Where("expires_at > ? AND status != ? AND status != ? AND deleted_at IS NULL",
// 			time.Now(),
// 			models.OnboardingStatusComplete,
// 			models.OnboardingStatusExpired).
// 		Count(&count).Error
// 	return count, err
// }

// // ============================================
// // TRANSACTION OPERATIONS
// // ============================================

// // BeginTx starts a new transaction
// func (r *OnboardingRepository) BeginTx() *gorm.DB {
// 	return r.db.Begin()
// }

// // CommitTx commits a transaction
// func (r *OnboardingRepository) CommitTx(tx *gorm.DB) error {
// 	return tx.Commit().Error
// }

// // RollbackTx rolls back a transaction
// func (r *OnboardingRepository) RollbackTx(tx *gorm.DB) error {
// 	return tx.Rollback().Error
// }

// // ============================================
// // BULK OPERATIONS
// // ============================================

// // DeleteAllSessionsByUserID deletes all sessions for a user
// func (r *OnboardingRepository) DeleteAllSessionsByUserID(userID string) error {
// 	return r.db.Where("user_id = ?", userID).Delete(&models.OnboardingSession{}).Error
// }

// // DeleteAllSessionsBySchoolID deletes all sessions for a school
// func (r *OnboardingRepository) DeleteAllSessionsBySchoolID(schoolID string) error {
// 	return r.db.Where("school_id = ?", schoolID).Delete(&models.OnboardingSession{}).Error
// }

// // ============================================
// // HELPER CHECKS
// // ============================================

// // SessionExists checks if a session exists by token
// func (r *OnboardingRepository) SessionExists(token string) (bool, error) {
// 	var count int64
// 	err := r.db.Model(&models.OnboardingSession{}).
// 		Where("session_token = ? AND deleted_at IS NULL", token).
// 		Count(&count).Error
// 	return count > 0, err
// }

// // IsSessionValid checks if a session is valid (exists and not expired)
// func (r *OnboardingRepository) IsSessionValid(token string) (bool, error) {
// 	var count int64
// 	err := r.db.Model(&models.OnboardingSession{}).
// 		Where("session_token = ? AND expires_at > ? AND deleted_at IS NULL",
// 			token, time.Now()).
// 		Count(&count).Error
// 	return count > 0, err
// }

// // GetSessionProgress gets the current step/status of a session
// func (r *OnboardingRepository) GetSessionProgress(token string) (models.OnboardingStatus, error) {
// 	var session models.OnboardingSession
// 	err := r.db.Select("status").Where("session_token = ? AND deleted_at IS NULL", token).
// 		First(&session).Error
// 	if err != nil {
// 		return "", err
// 	}
// 	return session.Status, nil
// }


// // package repository

// // import (
// //     "errors"
// //     "time"

// //     "cbt-api/internal/models"
// //     "gorm.io/gorm"
// // )

// // type OnboardingRepository struct {
// //     db *gorm.DB
// // }

// // func NewOnboardingRepository(db *gorm.DB) *OnboardingRepository {
// //     return &OnboardingRepository{db: db}
// // }

// // // ============================================
// // // SESSION OPERATIONS
// // // ============================================

// // // CreateSession creates a new onboarding session
// // func (r *OnboardingRepository) CreateSession(session *models.OnboardingSession) error {
// //     return r.db.Create(session).Error
// // }

// // // FindSessionByToken finds a session by its token
// // func (r *OnboardingRepository) FindSessionByToken(token string) (*models.OnboardingSession, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Where("session_token = ? AND deleted_at IS NULL", token).
// //         First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }

// //     // Check if expired
// //     if session.ExpiresAt.Before(time.Now()) {
// //         return nil, errors.New("session expired")
// //     }

// //     return &session, nil
// // }

// // // FindSessionByID finds a session by its ID
// // func (r *OnboardingRepository) FindSessionByID(id string) (*models.OnboardingSession, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).
// //         First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }

// //     if session.ExpiresAt.Before(time.Now()) {
// //         return nil, errors.New("session expired")
// //     }

// //     return &session, nil
// // }

// // // UpdateSession updates an existing session
// // func (r *OnboardingRepository) UpdateSession(session *models.OnboardingSession) error {
// //     session.UpdatedAt = time.Now()
// //     return r.db.Save(session).Error
// // }

// // // DeleteSession soft deletes a session
// // func (r *OnboardingRepository) DeleteSession(id string) error {
// //     return r.db.Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
// // }

// // // HardDeleteSession permanently deletes a session
// // func (r *OnboardingRepository) HardDeleteSession(id string) error {
// //     return r.db.Unscoped().Where("id = ?", id).Delete(&models.OnboardingSession{}).Error
// // }

// // // FindSessionByUserID finds a session by user ID
// // func (r *OnboardingRepository) FindSessionByUserID(userID string) (*models.OnboardingSession, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).
// //         Order("created_at DESC").
// //         First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // // FindSessionBySchoolID finds a session by school ID
// // func (r *OnboardingRepository) FindSessionBySchoolID(schoolID string) (*models.OnboardingSession, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Where("school_id = ? AND deleted_at IS NULL", schoolID).
// //         Order("created_at DESC").
// //         First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // // FindSessionBySubscriptionID finds a session by subscription ID
// // func (r *OnboardingRepository) FindSessionBySubscriptionID(subscriptionID string) (*models.OnboardingSession, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Where("subscription_id = ? AND deleted_at IS NULL", subscriptionID).
// //         Order("created_at DESC").
// //         First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // // FindSessionsByStatus finds all sessions with a given status
// // func (r *OnboardingRepository) FindSessionsByStatus(status models.OnboardingStatus) ([]models.OnboardingSession, error) {
// //     var sessions []models.OnboardingSession
// //     err := r.db.Where("status = ? AND deleted_at IS NULL", status).
// //         Order("created_at DESC").
// //         Find(&sessions).Error
// //     return sessions, err
// // }

// // // FindPendingSessions finds all sessions that are not complete or expired
// // func (r *OnboardingRepository) FindPendingSessions() ([]models.OnboardingSession, error) {
// //     var sessions []models.OnboardingSession
// //     err := r.db.Where("status != ? AND status != ? AND expires_at > ? AND deleted_at IS NULL",
// //         models.OnboardingStatusComplete,
// //         models.OnboardingStatusExpired,
// //         time.Now()).
// //         Order("created_at ASC").
// //         Find(&sessions).Error
// //     return sessions, err
// // }

// // // ============================================
// // // CLEANUP OPERATIONS
// // // ============================================

// // // DeleteExpiredSessions deletes all expired sessions
// // func (r *OnboardingRepository) DeleteExpiredSessions() error {
// //     return r.db.Where("expires_at < ? AND status != ? AND deleted_at IS NULL",
// //         time.Now(),
// //         models.OnboardingStatusComplete).
// //         Delete(&models.OnboardingSession{}).Error
// // }

// // // MarkExpiredSessions marks all expired sessions as EXPIRED
// // func (r *OnboardingRepository) MarkExpiredSessions() error {
// //     return r.db.Model(&models.OnboardingSession{}).
// //         Where("expires_at < ? AND status != ? AND status != ? AND deleted_at IS NULL",
// //             time.Now(),
// //             models.OnboardingStatusComplete,
// //             models.OnboardingStatusExpired).
// //         Update("status", models.OnboardingStatusExpired).Error
// // }

// // // CountActiveSessions counts active (non-expired, non-complete) sessions
// // func (r *OnboardingRepository) CountActiveSessions() (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.OnboardingSession{}).
// //         Where("expires_at > ? AND status != ? AND status != ? AND deleted_at IS NULL",
// //             time.Now(),
// //             models.OnboardingStatusComplete,
// //             models.OnboardingStatusExpired).
// //         Count(&count).Error
// //     return count, err
// // }

// // // ============================================
// // // TRANSACTION OPERATIONS
// // // ============================================

// // // BeginTx starts a new transaction
// // func (r *OnboardingRepository) BeginTx() *gorm.DB {
// //     return r.db.Begin()
// // }

// // // CommitTx commits a transaction
// // func (r *OnboardingRepository) CommitTx(tx *gorm.DB) error {
// //     return tx.Commit().Error
// // }

// // // RollbackTx rolls back a transaction
// // func (r *OnboardingRepository) RollbackTx(tx *gorm.DB) error {
// //     return tx.Rollback().Error
// // }

// // // ============================================
// // // BULK OPERATIONS
// // // ============================================

// // // DeleteAllSessionsByUserID deletes all sessions for a user
// // func (r *OnboardingRepository) DeleteAllSessionsByUserID(userID string) error {
// //     return r.db.Where("user_id = ?", userID).Delete(&models.OnboardingSession{}).Error
// // }

// // // DeleteAllSessionsBySchoolID deletes all sessions for a school
// // func (r *OnboardingRepository) DeleteAllSessionsBySchoolID(schoolID string) error {
// //     return r.db.Where("school_id = ?", schoolID).Delete(&models.OnboardingSession{}).Error
// // }

// // // ============================================
// // // HELPER CHECKS
// // // ============================================

// // // SessionExists checks if a session exists by token
// // func (r *OnboardingRepository) SessionExists(token string) (bool, error) {
// //     var count int64
// //     err := r.db.Model(&models.OnboardingSession{}).
// //         Where("session_token = ? AND deleted_at IS NULL", token).
// //         Count(&count).Error
// //     return count > 0, err
// // }

// // // IsSessionValid checks if a session is valid (exists and not expired)
// // func (r *OnboardingRepository) IsSessionValid(token string) (bool, error) {
// //     var count int64
// //     err := r.db.Model(&models.OnboardingSession{}).
// //         Where("session_token = ? AND expires_at > ? AND deleted_at IS NULL",
// //             token, time.Now()).
// //         Count(&count).Error
// //     return count > 0, err
// // }

// // // GetSessionProgress gets the current step/status of a session
// // func (r *OnboardingRepository) GetSessionProgress(token string) (models.OnboardingStatus, error) {
// //     var session models.OnboardingSession
// //     err := r.db.Select("status").Where("session_token = ? AND deleted_at IS NULL", token).
// //         First(&session).Error
// //     if err != nil {
// //         return "", err
// //     }
// //     return session.Status, nil
// // }

// // // internal/auth/repository/auth_repository.go

// // // CreateOTP creates a new OTP record
// // func (r *AuthRepository) CreateOTP(otp *models.OTP) error {
// //     return r.db.Create(otp).Error
// // }

// // // FindValidOTP finds a valid (unused, not expired) OTP
// // func (r *AuthRepository) FindValidOTP(email, code, otpType string) (*models.OTP, error) {
// //     var otp models.OTP
// //     err := r.db.Where("email = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?", 
// //         email, code, otpType, false, time.Now()).
// //         First(&otp).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &otp, nil
// // }

// // // MarkOTPAsUsed marks an OTP as used
// // func (r *AuthRepository) MarkOTPAsUsed(otpID string) error {
// //     return r.db.Model(&models.OTP{}).Where("id = ?", otpID).Update("used", true).Error
// // }

// // // InvalidateUserOTPs invalidates all OTPs for a user
// // func (r *AuthRepository) InvalidateUserOTPs(userID, otpType string) error {
// //     return r.db.Model(&models.OTP{}).
// //         Where("user_id = ? AND type = ? AND used = ?", userID, otpType, false).
// //         Update("used", true).Error
// // }

// // // VerifyEmailWithOTP verifies email with OTP code
// // func (r *AuthRepository) VerifyEmailWithOTP(userID, code string) error {
// //     // Find the OTP record for this user
// //     var otp models.OTP
// //     err := r.db.Where("user_id = ? AND code = ? AND type = ? AND used = ? AND expires_at > ?", 
// //         userID, code, "email_verification", false, time.Now()).
// //         First(&otp).Error
// //     if err != nil {
// //         return errors.New("invalid or expired verification code")
// //     }
    
// //     // Mark OTP as used
// //     if err := r.db.Model(&otp).Update("used", true).Error; err != nil {
// //         return err
// //     }
    
// //     // Update user's email_verified status
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).Update("email_verified", true).Error
// // }