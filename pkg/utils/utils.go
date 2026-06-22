package utils

import (
    "crypto/rand"
    "encoding/hex"
    "math/big"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

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
    return token.SignedString([]byte("your-secret-key"))
}

func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte("your-secret-key"), nil
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