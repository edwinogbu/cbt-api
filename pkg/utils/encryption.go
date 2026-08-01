package utils

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
    "log"
)

// ⭐ FIXED: Use a consistent 32-byte key
var masterKey = []byte("01234567890123456789012345678901") // 32 bytes exactly

func EncryptPlainPassword(plainText string) (string, error) {
    log.Printf("🔐 [Encrypt] Starting encryption...")
    log.Printf("📝 [Encrypt] Plain text length: %d", len(plainText))
    
    if plainText == "" {
        return "", fmt.Errorf("cannot encrypt empty password")
    }
    
    // Create cipher
    block, err := aes.NewCipher(masterKey)
    if err != nil {
        log.Printf("❌ [Encrypt] Failed to create cipher: %v", err)
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        log.Printf("❌ [Encrypt] Failed to create GCM: %v", err)
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }

    // Generate nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        log.Printf("❌ [Encrypt] Failed to generate nonce: %v", err)
        return "", fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt
    ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
    
    // Encode to base64
    encoded := base64.StdEncoding.EncodeToString(ciphertext)
    
    if encoded == "" {
        log.Printf("❌ [Encrypt] Encoded result is empty!")
        return "", fmt.Errorf("encoded result is empty")
    }
    
    log.Printf("✅ [Encrypt] Success! Encoded length: %d", len(encoded))
    return encoded, nil
}

func DecryptPlainPassword(encryptedText string) (string, error) {
    log.Printf("🔓 [Decrypt] Starting decryption...")
    log.Printf("📝 [Decrypt] Encrypted text length: %d", len(encryptedText))
    
    if encryptedText == "" {
        log.Printf("⚠️ [Decrypt] Empty encrypted text")
        return "", nil
    }
    
    // Decode from base64
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
    if err != nil {
        log.Printf("❌ [Decrypt] Failed to decode base64: %v", err)
        return "", fmt.Errorf("failed to decode: %w", err)
    }

    // Create cipher
    block, err := aes.NewCipher(masterKey)
    if err != nil {
        log.Printf("❌ [Decrypt] Failed to create cipher: %v", err)
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        log.Printf("❌ [Decrypt] Failed to create GCM: %v", err)
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }

    // Verify length
    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        log.Printf("❌ [Decrypt] Ciphertext too short: %d < %d", len(ciphertext), nonceSize)
        return "", fmt.Errorf("ciphertext too short")
    }

    // Decrypt
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        log.Printf("❌ [Decrypt] Decryption failed: %v", err)
        return "", fmt.Errorf("decryption failed: %w", err)
    }

    log.Printf("✅ [Decrypt] Success! Plain text length: %d", len(plaintext))
    return string(plaintext), nil
}


// package utils

// import (
//     "crypto/aes"
//     "crypto/cipher"
//     "crypto/rand"
//     "encoding/base64"
//     "errors"
//     "io"
//     "os"
// )

// var masterKey []byte

// func init() {
//     key := os.Getenv("ENCRYPTION_MASTER_KEY")
//     if key == "" {
//         key = "32-byte-long-key-for-aes-256-encryption"
//     }
//     masterKey = []byte(key)
// }

// func EncryptPlainPassword(plainText string) (string, error) {
//     block, err := aes.NewCipher(masterKey)
//     if err != nil {
//         return "", err
//     }

//     gcm, err := cipher.NewGCM(block)
//     if err != nil {
//         return "", err
//     }

//     nonce := make([]byte, gcm.NonceSize())
//     if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
//         return "", err
//     }

//     ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
//     return base64.StdEncoding.EncodeToString(ciphertext), nil
// }

// func DecryptPlainPassword(encryptedText string) (string, error) {
//     ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
//     if err != nil {
//         return "", err
//     }

//     block, err := aes.NewCipher(masterKey)
//     if err != nil {
//         return "", err
//     }

//     gcm, err := cipher.NewGCM(block)
//     if err != nil {
//         return "", err
//     }

//     nonceSize := gcm.NonceSize()
//     if len(ciphertext) < nonceSize {
//         return "", errors.New("ciphertext too short")
//     }

//     nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
//     plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
//     if err != nil {
//         return "", err
//     }

//     return string(plaintext), nil
// }