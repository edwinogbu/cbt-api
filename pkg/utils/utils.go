package utils

import (
    "crypto/rand"
    "encoding/hex"
    "log"
    "math/big"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

// jwtSecret is the actual signing/validation secret used by every
// protected route (via middleware.AuthMiddleware -> ValidateJWT). This
// was previously hardcoded to the literal "your-secret-key" below,
// completely bypassing config.go's JWT_SECRET env var and its
// production-startup safety check (validateProductionSecrets) - that
// check validated a *different*, unused Config.JWT.Secret value, so a
// production deployment could start up clean while every JWT in the
// system was still signed with a secret checked into this public
// source file, letting anyone forge a valid token for any user_id and
// any role (including admin) without ever touching the database.
// Reads JWT_SECRET directly (same env var name and insecure-default
// string as config.go) rather than depending on config.LoadConfig()'s
// singleton, so this package stays correct regardless of load order.
const insecureDefaultJWTSecret = "your-secret-key-change-in-production"

var jwtSecret = func() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = insecureDefaultJWTSecret
    }
    if os.Getenv("ENVIRONMENT") == "production" && secret == insecureDefaultJWTSecret {
        log.Fatal("FATAL: JWT_SECRET must be set to a real, per-deployment secret in production (ENVIRONMENT=production). Refusing to start with the insecure default.")
    }
    return []byte(secret)
}()

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

func GenerateUUID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func GenerateJWT(userID, email, role string, expiry time.Duration) (string, error) {
    claims := jwt.MapClaims{
        "user_id":    userID,
        "email":      email,
        "role":       role,
        "session_id": GenerateUUID(),
        "exp":        time.Now().Add(expiry).Unix(),
        "iat":        time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    return nil, jwt.ErrSignatureInvalid
}

// GenerateRandomPassword generates a cryptographically secure random password
// containing uppercase, lowercase, digits, and special characters.
func GenerateRandomPassword(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
    password := make([]byte, length)
    for i := range password {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        password[i] = charset[n.Int64()]
    }
    return string(password)
}



// package utils

// import (
//     "crypto/rand"
//     "encoding/hex"
//     "time"

//     "github.com/golang-jwt/jwt/v5"
//     "golang.org/x/crypto/bcrypt"
// )

// func HashPassword(password string) (string, error) {
//     bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
//     return string(bytes), err
// }

// func CheckPasswordHash(password, hash string) bool {
//     err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
//     return err == nil
// }

// func GenerateUUID() string {
//     bytes := make([]byte, 16)
//     rand.Read(bytes)
//     return hex.EncodeToString(bytes)
// }

// func GenerateJWT(userID, email, role string, expiry time.Duration) (string, error) {
//     claims := jwt.MapClaims{
//         "user_id":    userID,
//         "email":      email,
//         "role":       role,
//         "session_id": GenerateUUID(),
//         "exp":        time.Now().Add(expiry).Unix(),
//         "iat":        time.Now().Unix(),
//     }
//     token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
//     return token.SignedString([]byte("your-secret-key"))
// }

// func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
//     token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
//         return []byte("your-secret-key"), nil
//     })
//     if err != nil {
//         return nil, err
//     }
//     if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
//         return claims, nil
//     }
//     return nil, jwt.ErrSignatureInvalid
// }