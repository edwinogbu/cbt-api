package repository

import (
    "cbt-api/internal/models"
    "cbt-api/pkg/utils"
    "fmt"
    "gorm.io/gorm"
    "log"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
    return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id string) (*models.User, error) {
    var user models.User
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
    var user models.User
    err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
    var user models.User
    err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) UpdateUser(user *models.User) error {
    return r.db.Save(user).Error
}

func (r *UserRepository) SoftDeleteUser(id string) error {
    return r.db.Delete(&models.User{}, "id = ?", id).Error
}

// ⭐ SAVE PLAIN PASSWORD - COMPLETE REWRITE WITH DIRECT SQL
func (r *UserRepository) SavePlainPassword(userID string, plainPassword string) error {
    log.Printf("🔐 [SavePlainPassword] Starting for user: %s", userID)
    log.Printf("📝 [SavePlainPassword] Password length: %d", len(plainPassword))
    
    // Validate
    if plainPassword == "" {
        return fmt.Errorf("cannot save empty password")
    }
    if userID == "" {
        return fmt.Errorf("user ID is required")
    }
    
    // Encrypt
    encrypted, err := utils.EncryptPlainPassword(plainPassword)
    if err != nil {
        log.Printf("❌ [SavePlainPassword] Encryption failed: %v", err)
        return fmt.Errorf("encryption failed: %w", err)
    }
    
    log.Printf("✅ [SavePlainPassword] Encrypted length: %d", len(encrypted))
    
    // ⭐ METHOD 1: Using Exec with raw SQL (most reliable)
    result := r.db.Exec(`
        UPDATE users 
        SET plain_password_encrypted = ? 
        WHERE id = ? AND deleted_at IS NULL
    `, encrypted, userID)
    
    if result.Error != nil {
        log.Printf("❌ [SavePlainPassword] Update failed: %v", result.Error)
        return fmt.Errorf("update failed: %w", result.Error)
    }
    
    log.Printf("✅ [SavePlainPassword] Rows affected: %d", result.RowsAffected)
    
    if result.RowsAffected == 0 {
        log.Printf("⚠️ [SavePlainPassword] No rows affected - user not found")
        return fmt.Errorf("user not found: %s", userID)
    }
    
    // ⭐ VERIFY the update worked
    var verifyUser models.User
    err = r.db.Raw(`
        SELECT id, plain_password_encrypted 
        FROM users 
        WHERE id = ? AND deleted_at IS NULL
    `, userID).Scan(&verifyUser).Error
    
    if err != nil {
        log.Printf("⚠️ [SavePlainPassword] Verification query failed: %v", err)
    } else if verifyUser.PlainPasswordEncrypted == "" {
        log.Printf("❌ [SavePlainPassword] VERIFICATION FAILED - column is empty!")
        // Try one more time with a different approach
        log.Printf("🔄 [SavePlainPassword] Retrying with Model update...")
        
        retryResult := r.db.Model(&models.User{}).
            Where("id = ?", userID).
            Update("plain_password_encrypted", encrypted)
        
        if retryResult.Error != nil {
            log.Printf("❌ [SavePlainPassword] Retry failed: %v", retryResult.Error)
            return fmt.Errorf("failed to save after retry: %w", retryResult.Error)
        }
        
        // Verify again
        var retryVerify models.User
        r.db.Raw(`SELECT plain_password_encrypted FROM users WHERE id = ?`, userID).Scan(&retryVerify)
        if retryVerify.PlainPasswordEncrypted != "" {
            log.Printf("✅ [SavePlainPassword] Retry successful!")
            return nil
        }
        
        return fmt.Errorf("failed to save plain password - column remains empty")
    }
    
    log.Printf("✅ [SavePlainPassword] SUCCESS! Password stored and verified")
    return nil
}

// ⭐ GET PLAIN PASSWORD - COMPLETE REWRITE
func (r *UserRepository) GetPlainPassword(userID string) (string, error) {
    log.Printf("🔓 [GetPlainPassword] Fetching for user: %s", userID)
    
    if userID == "" {
        return "", fmt.Errorf("user ID is required")
    }
    
    var encrypted string
    err := r.db.Raw(`
        SELECT plain_password_encrypted 
        FROM users 
        WHERE id = ? AND deleted_at IS NULL
    `, userID).Scan(&encrypted).Error
    
    if err != nil {
        log.Printf("❌ [GetPlainPassword] Query failed: %v", err)
        return "", err
    }
    
    log.Printf("📝 [GetPlainPassword] Encrypted value length: %d", len(encrypted))
    
    if encrypted == "" {
        log.Printf("⚠️ [GetPlainPassword] No encrypted password found")
        return "", nil
    }
    
    // Decrypt
    decrypted, err := utils.DecryptPlainPassword(encrypted)
    if err != nil {
        log.Printf("❌ [GetPlainPassword] Decryption failed: %v", err)
        return "", err
    }
    
    log.Printf("✅ [GetPlainPassword] Decrypted password length: %d", len(decrypted))
    return decrypted, nil
}


// package repository

// import (
//     "cbt-api/internal/models"
//     "cbt-api/pkg/utils"
//     "gorm.io/gorm"
// )

// type UserRepository struct {
//     db *gorm.DB
// }

// func NewUserRepository(db *gorm.DB) *UserRepository {
//     return &UserRepository{db: db}
// }

// func (r *UserRepository) CreateUser(user *models.User) error {
//     return r.db.Create(user).Error
// }

// func (r *UserRepository) FindByID(id string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) UpdateUser(user *models.User) error {
//     return r.db.Save(user).Error
// }

// func (r *UserRepository) SoftDeleteUser(id string) error {
//     return r.db.Delete(&models.User{}, "id = ?", id).Error
// }

// // // Save plain password encrypted
// // func (r *UserRepository) SavePlainPassword(userID string, plainPassword string) error {
// //     encrypted, err := utils.EncryptPlainPassword(plainPassword)
// //     if err != nil {
// //         return err
// //     }
// //     return r.db.Model(&models.User{}).
// //         Where("id = ?", userID).
// //         Update("plain_password_encrypted", encrypted).Error
// // }

// // SavePlainPassword - Updated with debug logging
// func (r *UserRepository) SavePlainPassword(userID string, plainPassword string) error {
//     // Log the attempt
//     println("🔐 Attempting to save plain password for user:", userID)
//     println("📝 Plain password length:", len(plainPassword))
    
//     // Encrypt the password
//     encrypted, err := utils.EncryptPlainPassword(plainPassword)
//     if err != nil {
//         println("❌ Encryption failed:", err.Error())
//         return err
//     }
    
//     println("✅ Encryption successful, encrypted length:", len(encrypted))
    
//     // Update the database
//     result := r.db.Model(&models.User{}).
//         Where("id = ?", userID).
//         Update("plain_password_encrypted", encrypted)
    
//     if result.Error != nil {
//         println("❌ Database update failed:", result.Error.Error())
//         return result.Error
//     }
    
//     println("✅ Database updated, rows affected:", result.RowsAffected)
//     return nil
// }

// // GetPlainPassword - Updated with debug logging
// func (r *UserRepository) GetPlainPassword(userID string) (string, error) {
//     var user models.User
//     err := r.db.Select("plain_password_encrypted").
//         Where("id = ? AND deleted_at IS NULL", userID).
//         First(&user).Error
//     if err != nil {
//         println("❌ Failed to fetch user:", err.Error())
//         return "", err
//     }
    
//     println("🔍 Retrieved plain_password_encrypted for user:", userID)
//     println("📝 Encrypted value exists:", user.PlainPasswordEncrypted != "")
    
//     if user.PlainPasswordEncrypted == "" {
//         return "", nil
//     }
    
//     decrypted, err := utils.DecryptPlainPassword(user.PlainPasswordEncrypted)
//     if err != nil {
//         println("❌ Decryption failed:", err.Error())
//         return "", err
//     }
    
//     println("✅ Decryption successful")
//     return decrypted, nil
// }



// // Get plain password decrypted
// func (r *UserRepository) GetPlainPassword(userID string) (string, error) {
//     var user models.User
//     err := r.db.Select("plain_password_encrypted").
//         Where("id = ? AND deleted_at IS NULL", userID).
//         First(&user).Error
//     if err != nil {
//         return "", err
//     }
    
//     if user.PlainPasswordEncrypted == "" {
//         return "", nil
//     }
    
//     return utils.DecryptPlainPassword(user.PlainPasswordEncrypted)
// }

// package repository

// import (
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
//     "time"
// )

// type UserRepository struct {
//     db *gorm.DB
// }

// func NewUserRepository(db *gorm.DB) *UserRepository {
//     return &UserRepository{db: db}
// }

// func (r *UserRepository) FindByID(id string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) Create(user *models.User) error {
//     return r.db.Create(user).Error
// }

// func (r *UserRepository) Update(user *models.User) error {
//     return r.db.Save(user).Error
// }

// func (r *UserRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.User{}).Error
// }

// func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) CreateUser(user *models.User) error {
//     return r.db.Create(user).Error
// }

// func (r *UserRepository) UpdateUser(user *models.User) error {
//     return r.db.Save(user).Error
// }

// func (r *UserRepository) SoftDeleteUser(id string) error {
//     return r.db.Model(&models.User{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
// }






// package repository

// import (
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
// )

// type UserRepository struct {
//     db *gorm.DB
// }

// func NewUserRepository(db *gorm.DB) *UserRepository {
//     return &UserRepository{db: db}
// }

// func (r *UserRepository) FindByID(id string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) Create(user *models.User) error {
//     return r.db.Create(user).Error
// }

// func (r *UserRepository) Update(user *models.User) error {
//     return r.db.Save(user).Error
// }

// func (r *UserRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.User{}).Error
// }


