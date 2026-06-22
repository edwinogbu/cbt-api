package repository

import (
	"time"

	"cbt-api/internal/models"
	"cbt-api/pkg/utils"
	"gorm.io/gorm"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// User operations
func (r *AuthRepository) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ADD THIS METHOD
func (r *AuthRepository) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *AuthRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *AuthRepository) UpdateLastLogin(userID string) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", now).Error
}

// OTP operations
func (r *AuthRepository) CreateOTP(otp *models.OTP) error {
	return r.db.Create(otp).Error
}

// FindValidOTP unchanged
func (r *AuthRepository) FindValidOTP(email, code, otpType string) (*models.OTP, error) {
	var user models.User
	if err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error; err != nil {
		return nil, err
	}
	var otp models.OTP
	err := r.db.Where("user_id = ? AND type = ? AND expires_at > ? AND used = false",
		user.ID, otpType, time.Now()).First(&otp).Error
	if err != nil {
		return nil, err
	}
	if !utils.CheckPasswordHash(code, otp.Code) {
		return nil, gorm.ErrRecordNotFound
	}
	return &otp, nil
}

func (r *AuthRepository) MarkOTPAsUsed(otpID uint) error {
	return r.db.Model(&models.OTP{}).Where("id = ?", otpID).Update("used", true).Error
}

func (r *AuthRepository) InvalidateUserOTPs(userID string, otpType string) error {
	return r.db.Model(&models.OTP{}).Where("user_id = ? AND type = ? AND used = false", userID, otpType).
		Update("used", true).Error
}

// Session operations
func (r *AuthRepository) CreateSession(session *models.UserSession) error {
	return r.db.Create(session).Error
}

func (r *AuthRepository) FindSessionByRefreshToken(refreshToken string) (*models.UserSession, error) {
	var session models.UserSession
	err := r.db.Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *AuthRepository) FindSessionsByUserID(userID string) ([]models.UserSession, error) {
	var sessions []models.UserSession
	err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *AuthRepository) DeleteSession(sessionID string) error {
	return r.db.Delete(&models.UserSession{}, "id = ?", sessionID).Error
}

func (r *AuthRepository) DeleteAllUserSessions(userID string, exceptSessionID string) error {
	query := r.db.Where("user_id = ?", userID)
	if exceptSessionID != "" {
		query = query.Where("id != ?", exceptSessionID)
	}
	return query.Delete(&models.UserSession{}).Error
}

// Two-factor operations
func (r *AuthRepository) UpdateTwoFactorSecret(userID, secret string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"two_factor_secret":  secret,
			"two_factor_enabled": false,
		}).Error
}

func (r *AuthRepository) EnableTwoFactor(userID string, secret string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"two_factor_secret":  secret,
			"two_factor_enabled": true,
		}).Error
}

func (r *AuthRepository) DisableTwoFactor(userID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"two_factor_secret":  "",
			"two_factor_enabled": false,
		}).Error
}

func (r *AuthRepository) UpdatePassword(userID, newPasswordHash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("password", newPasswordHash).Error
}

func (r *AuthRepository) VerifyEmail(userID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified": true,
			"status":         "active",
		}).Error
}

func (r *AuthRepository) SoftDeleteUser(userID string) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("deleted_at", now).Error
}


// package repository

// import (
// 	"time"

// 	"cbt-api/internal/models"
// 	"cbt-api/pkg/utils"
// 	"gorm.io/gorm"
// )

// type AuthRepository struct {
// 	db *gorm.DB
// }

// func NewAuthRepository(db *gorm.DB) *AuthRepository {
// 	return &AuthRepository{db: db}
// }

// // User operations
// func (r *AuthRepository) FindUserByEmail(email string) (*models.User, error) {
// 	var user models.User
// 	err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &user, nil
// }

// func (r *AuthRepository) FindUserByID(id string) (*models.User, error) {
// 	var user models.User
// 	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &user, nil
// }

// func (r *AuthRepository) CreateUser(user *models.User) error {
// 	return r.db.Create(user).Error
// }

// func (r *AuthRepository) UpdateUser(user *models.User) error {
// 	return r.db.Save(user).Error
// }

// func (r *AuthRepository) UpdateLastLogin(userID string) error {
// 	now := time.Now()
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", now).Error
// }

// // OTP operations
// func (r *AuthRepository) CreateOTP(otp *models.OTP) error {
// 	return r.db.Create(otp).Error
// }

// // FindValidOTP retrieves a valid (unused, not expired) OTP for the given email and type,
// // then verifies the plain code against the stored bcrypt hash.
// func (r *AuthRepository) FindValidOTP(email, code, otpType string) (*models.OTP, error) {
// 	// 1. Find the user by email
// 	var user models.User
// 	if err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error; err != nil {
// 		return nil, err
// 	}

// 	// 2. Find an OTP record that matches user, type, not used, not expired
// 	var otp models.OTP
// 	err := r.db.Where("user_id = ? AND type = ? AND expires_at > ? AND used = false",
// 		user.ID, otpType, time.Now()).First(&otp).Error
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 3. Verify the plain code against the stored hash
// 	if !utils.CheckPasswordHash(code, otp.Code) {
// 		return nil, gorm.ErrRecordNotFound // code mismatch -> treat as not found
// 	}

// 	return &otp, nil
// }

// func (r *AuthRepository) MarkOTPAsUsed(otpID uint) error {
// 	return r.db.Model(&models.OTP{}).Where("id = ?", otpID).Update("used", true).Error
// }

// func (r *AuthRepository) InvalidateUserOTPs(userID string, otpType string) error {
// 	return r.db.Model(&models.OTP{}).Where("user_id = ? AND type = ? AND used = false", userID, otpType).
// 		Update("used", true).Error
// }

// // Session operations
// func (r *AuthRepository) CreateSession(session *models.UserSession) error {
// 	return r.db.Create(session).Error
// }

// func (r *AuthRepository) FindSessionByRefreshToken(refreshToken string) (*models.UserSession, error) {
// 	var session models.UserSession
// 	err := r.db.Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).First(&session).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &session, nil
// }

// func (r *AuthRepository) FindSessionsByUserID(userID string) ([]models.UserSession, error) {
// 	var sessions []models.UserSession
// 	err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).
// 		Order("created_at DESC").Find(&sessions).Error
// 	return sessions, err
// }

// func (r *AuthRepository) DeleteSession(sessionID string) error {
// 	return r.db.Delete(&models.UserSession{}, "id = ?", sessionID).Error
// }

// func (r *AuthRepository) DeleteAllUserSessions(userID string, exceptSessionID string) error {
// 	query := r.db.Where("user_id = ?", userID)
// 	if exceptSessionID != "" {
// 		query = query.Where("id != ?", exceptSessionID)
// 	}
// 	return query.Delete(&models.UserSession{}).Error
// }

// // Two-factor operations
// func (r *AuthRepository) UpdateTwoFactorSecret(userID, secret string) error {
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).
// 		Updates(map[string]interface{}{
// 			"two_factor_secret": secret,
// 			"two_factor_enabled": false,
// 		}).Error
// }

// func (r *AuthRepository) EnableTwoFactor(userID string, secret string) error {
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).
// 		Updates(map[string]interface{}{
// 			"two_factor_secret": secret,
// 			"two_factor_enabled": true,
// 		}).Error
// }

// func (r *AuthRepository) DisableTwoFactor(userID string) error {
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).
// 		Updates(map[string]interface{}{
// 			"two_factor_secret": "",
// 			"two_factor_enabled": false,
// 		}).Error
// }

// func (r *AuthRepository) UpdatePassword(userID, newPasswordHash string) error {
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).
// 		Update("password", newPasswordHash).Error
// }

// func (r *AuthRepository) VerifyEmail(userID string) error {
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).
// 		Updates(map[string]interface{}{
// 			"email_verified": true,
// 			"status":         "active",
// 		}).Error
// }

// // SoftDeleteUser soft deletes a user
// func (r *AuthRepository) SoftDeleteUser(userID string) error {
// 	now := time.Now()
// 	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("deleted_at", now).Error
// }

// // FindUserByUsername retrieves a user by username (case‑sensitive)
// func (r *AuthRepository) FindUserByUsername(username string) (*models.User, error) {
// 	var user models.User
// 	err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &user, nil
// }



// // package repository

// // import (
// //     "time"

// //     "cbt-api/internal/models"
// //     "gorm.io/gorm"
// // )

// // type AuthRepository struct {
// //     db *gorm.DB
// // }

// // func NewAuthRepository(db *gorm.DB) *AuthRepository {
// //     return &AuthRepository{db: db}
// // }

// // // User operations
// // func (r *AuthRepository) FindUserByEmail(email string) (*models.User, error) {
// //     var user models.User
// //     err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &user, nil
// // }

// // func (r *AuthRepository) FindUserByID(id string) (*models.User, error) {
// //     var user models.User
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &user, nil
// // }

// // func (r *AuthRepository) CreateUser(user *models.User) error {
// //     return r.db.Create(user).Error
// // }

// // func (r *AuthRepository) UpdateUser(user *models.User) error {
// //     return r.db.Save(user).Error
// // }

// // func (r *AuthRepository) UpdateLastLogin(userID string) error {
// //     now := time.Now()
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", now).Error
// // }

// // // OTP operations
// // func (r *AuthRepository) CreateOTP(otp *models.OTP) error {
// //     return r.db.Create(otp).Error
// // }

// // func (r *AuthRepository) FindValidOTP(email, code, otpType string) (*models.OTP, error) {
// //     var user models.User
// //     if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
// //         return nil, err
// //     }

// //     var otp models.OTP
// //     err := r.db.Where("user_id = ? AND code = ? AND type = ? AND expires_at > ? AND used = false",
// //         user.ID, code, otpType, time.Now()).First(&otp).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &otp, nil
// // }

// // func (r *AuthRepository) MarkOTPAsUsed(otpID uint) error {
// //     return r.db.Model(&models.OTP{}).Where("id = ?", otpID).Update("used", true).Error
// // }

// // func (r *AuthRepository) InvalidateUserOTPs(userID string, otpType string) error {
// //     return r.db.Model(&models.OTP{}).Where("user_id = ? AND type = ? AND used = false", userID, otpType).
// //         Update("used", true).Error
// // }

// // // Session operations
// // func (r *AuthRepository) CreateSession(session *models.UserSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *AuthRepository) FindSessionByRefreshToken(refreshToken string) (*models.UserSession, error) {
// //     var session models.UserSession
// //     err := r.db.Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // func (r *AuthRepository) FindSessionsByUserID(userID string) ([]models.UserSession, error) {
// //     var sessions []models.UserSession
// //     err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).
// //         Order("created_at DESC").Find(&sessions).Error
// //     return sessions, err
// // }

// // func (r *AuthRepository) DeleteSession(sessionID string) error {
// //     return r.db.Delete(&models.UserSession{}, "id = ?", sessionID).Error
// // }

// // func (r *AuthRepository) DeleteAllUserSessions(userID string, exceptSessionID string) error {
// //     query := r.db.Where("user_id = ?", userID)
// //     if exceptSessionID != "" {
// //         query = query.Where("id != ?", exceptSessionID)
// //     }
// //     return query.Delete(&models.UserSession{}).Error
// // }

// // func (r *AuthRepository) UpdateTwoFactorSecret(userID, secret string) error {
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).
// //         Updates(map[string]interface{}{
// //             "two_factor_secret": secret,
// //             "two_factor_enabled": false,
// //         }).Error
// // }

// // func (r *AuthRepository) EnableTwoFactor(userID string, secret string) error {
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).
// //         Updates(map[string]interface{}{
// //             "two_factor_secret": secret,
// //             "two_factor_enabled": true,
// //         }).Error
// // }

// // func (r *AuthRepository) DisableTwoFactor(userID string) error {
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).
// //         Updates(map[string]interface{}{
// //             "two_factor_secret": "",
// //             "two_factor_enabled": false,
// //         }).Error
// // }

// // func (r *AuthRepository) UpdatePassword(userID, newPasswordHash string) error {
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).
// //         Update("password", newPasswordHash).Error
// // }

// // func (r *AuthRepository) VerifyEmail(userID string) error {
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).
// //         Updates(map[string]interface{}{
// //             "email_verified": true,
// //             "status":         "active",
// //         }).Error
// // }


// // // SoftDeleteUser soft deletes a user
// // func (r *AuthRepository) SoftDeleteUser(userID string) error {
// //     now := time.Now()
// //     return r.db.Model(&models.User{}).Where("id = ?", userID).Update("deleted_at", now).Error
// // }