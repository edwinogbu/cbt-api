package dto

import "time"

type RegisterRequest struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password" binding:"required,min=6"`
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	PhoneNumber     string `json:"phone_number"`
	Role            string `json:"role" binding:"required,oneof=student teacher admin parent"`
	AdmissionNumber string `json:"admission_number"` // required if role == "parent"
}

type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type SendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Type  string `json:"type" binding:"required,oneof=email_verification password_reset"`
}

type Enable2FARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

type Disable2FARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

type Verify2FARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

type RevokeSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// Responses
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	User         UserDTO   `json:"user"`
}

type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

type UserDTO struct {
	ID               string    `json:"id"`
	Username         string    `json:"username"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	PhoneNumber      string    `json:"phone_number"`
	Role             string    `json:"role"`
	Status           string    `json:"status"`
	EmailVerified    bool      `json:"email_verified"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

type SessionDTO struct {
	ID        string    `json:"id"`
	UserAgent string    `json:"user_agent"`
	ClientIP  string    `json:"client_ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsCurrent bool      `json:"is_current"`
}

type TwoFactorResponse struct {
	Secret     string `json:"secret"`
	QRCodeURL  string `json:"qr_code_url"`
	Enabled    bool   `json:"enabled"`
	Message    string `json:"message"`
}

type MessageResponse struct {
	Message string `json:"message"`
}





// package dto

// import "time"

// // RegisterRequest – enhanced with role, username, admission number
// type RegisterRequest struct {
// 	Username        string `json:"username"`        // optional – auto‑generated for students
// 	Email           string `json:"email"`           // optional for students
// 	Password        string `json:"password" binding:"required,min=6"`
// 	FirstName       string `json:"first_name" binding:"required"`
// 	LastName        string `json:"last_name" binding:"required"`
// 	PhoneNumber     string `json:"phone_number"`
// 	Role            string `json:"role" binding:"required,oneof=student teacher admin parent"`
// 	AdmissionNumber string `json:"admission_number"` // required if role == "parent"
// }

// // LoginRequest – accepts username OR email
// type LoginRequest struct {
// 	Username string `json:"username"`
// 	Email    string `json:"email"`
// 	Password string `json:"password" binding:"required"`
// }

// // RefreshTokenRequest unchanged
// type RefreshTokenRequest struct {
// 	RefreshToken string `json:"refresh_token" binding:"required"`
// }

// // ForgotPasswordRequest unchanged
// type ForgotPasswordRequest struct {
// 	Email string `json:"email" binding:"required,email"`
// }

// // ResetPasswordRequest unchanged
// type ResetPasswordRequest struct {
// 	Email       string `json:"email" binding:"required,email"`
// 	Code        string `json:"code" binding:"required,len=6"`
// 	NewPassword string `json:"new_password" binding:"required,min=6"`
// }

// // ChangePasswordRequest unchanged
// type ChangePasswordRequest struct {
// 	CurrentPassword string `json:"current_password" binding:"required"`
// 	NewPassword     string `json:"new_password" binding:"required,min=6"`
// }

// // VerifyEmailRequest unchanged
// type VerifyEmailRequest struct {
// 	Email string `json:"email" binding:"required,email"`
// 	Code  string `json:"code" binding:"required,len=6"`
// }

// // SendOTPRequest unchanged
// type SendOTPRequest struct {
// 	Email string `json:"email" binding:"required,email"`
// 	Type  string `json:"type" binding:"required,oneof=email_verification password_reset"`
// }

// // Enable2FARequest unchanged
// type Enable2FARequest struct {
// 	Code string `json:"code" binding:"required,len=6"`
// }

// // Disable2FARequest unchanged
// type Disable2FARequest struct {
// 	Code string `json:"code" binding:"required,len=6"`
// }

// // Verify2FARequest unchanged
// type Verify2FARequest struct {
// 	Code string `json:"code" binding:"required,len=6"`
// }

// // RevokeSessionRequest unchanged
// type RevokeSessionRequest struct {
// 	SessionID string `json:"session_id" binding:"required"`
// }

// // Response DTOs unchanged (LoginResponse, TokenResponse, UserDTO, etc.)
// // But UserDTO should include Username field (add it)
// type UserDTO struct {
// 	ID               string    `json:"id"`
// 	Username         string    `json:"username"`
// 	Email            string    `json:"email"`
// 	FirstName        string    `json:"first_name"`
// 	LastName         string    `json:"last_name"`
// 	PhoneNumber      string    `json:"phone_number"`
// 	Role             string    `json:"role"`
// 	Status           string    `json:"status"`
// 	EmailVerified    bool      `json:"email_verified"`
// 	TwoFactorEnabled bool      `json:"two_factor_enabled"`
// 	CreatedAt        time.Time `json:"created_at"`
// }

// // LoginResponse, TokenResponse, SessionDTO, TwoFactorResponse, MessageResponse unchanged
// type LoginResponse struct {
// 	AccessToken  string    `json:"access_token"`
// 	RefreshToken string    `json:"refresh_token"`
// 	ExpiresAt    time.Time `json:"expires_at"`
// 	TokenType    string    `json:"token_type"`
// 	User         UserDTO   `json:"user"`
// }

// type TokenResponse struct {
// 	AccessToken  string    `json:"access_token"`
// 	RefreshToken string    `json:"refresh_token"`
// 	ExpiresAt    time.Time `json:"expires_at"`
// 	TokenType    string    `json:"token_type"`
// }

// type SessionDTO struct {
// 	ID        string    `json:"id"`
// 	UserAgent string    `json:"user_agent"`
// 	ClientIP  string    `json:"client_ip"`
// 	CreatedAt time.Time `json:"created_at"`
// 	ExpiresAt time.Time `json:"expires_at"`
// 	IsCurrent bool      `json:"is_current"`
// }

// type TwoFactorResponse struct {
// 	Secret    string `json:"secret"`
// 	QRCodeURL string `json:"qr_code_url"`
// 	Enabled   bool   `json:"enabled"`
// 	Message   string `json:"message"`
// }

// type MessageResponse struct {
// 	Message string `json:"message"`
// }



// // package dto

// // import "time"

// // // Request DTOs
// // type RegisterRequest struct {
// //     Email       string `json:"email" binding:"required,email"`
// //     Password    string `json:"password" binding:"required,min=6"`
// //     FirstName   string `json:"first_name" binding:"required"`
// //     LastName    string `json:"last_name" binding:"required"`
// //     PhoneNumber string `json:"phone_number"`
// // }

// // type LoginRequest struct {
// //     Email    string `json:"email" binding:"required,email"`
// //     Password string `json:"password" binding:"required"`
// // }

// // type RefreshTokenRequest struct {
// //     RefreshToken string `json:"refresh_token" binding:"required"`
// // }

// // type ForgotPasswordRequest struct {
// //     Email string `json:"email" binding:"required,email"`
// // }

// // type ResetPasswordRequest struct {
// //     Email       string `json:"email" binding:"required,email"`
// //     Code        string `json:"code" binding:"required,len=6"`
// //     NewPassword string `json:"new_password" binding:"required,min=6"`
// // }

// // type ChangePasswordRequest struct {
// //     CurrentPassword string `json:"current_password" binding:"required"`
// //     NewPassword     string `json:"new_password" binding:"required,min=6"`
// // }

// // type VerifyEmailRequest struct {
// //     Email string `json:"email" binding:"required,email"`
// //     Code  string `json:"code" binding:"required,len=6"`
// // }

// // type SendOTPRequest struct {
// //     Email string `json:"email" binding:"required,email"`
// //     Type  string `json:"type" binding:"required,oneof=email_verification password_reset"`
// // }

// // type Enable2FARequest struct {
// //     Code string `json:"code" binding:"required,len=6"`
// // }

// // type Disable2FARequest struct {
// //     Code string `json:"code" binding:"required,len=6"`
// // }

// // type Verify2FARequest struct {
// //     Code string `json:"code" binding:"required,len=6"`
// // }

// // type RevokeSessionRequest struct {
// //     SessionID string `json:"session_id" binding:"required"`
// // }

// // // Response DTOs
// // type LoginResponse struct {
// //     AccessToken  string    `json:"access_token"`
// //     RefreshToken string    `json:"refresh_token"`
// //     ExpiresAt    time.Time `json:"expires_at"`
// //     TokenType    string    `json:"token_type"`
// //     User         UserDTO   `json:"user"`
// // }

// // type TokenResponse struct {
// //     AccessToken  string    `json:"access_token"`
// //     RefreshToken string    `json:"refresh_token"`
// //     ExpiresAt    time.Time `json:"expires_at"`
// //     TokenType    string    `json:"token_type"`
// // }

// // type UserDTO struct {
// //     ID               string    `json:"id"`
// //     Email            string    `json:"email"`
// //     FirstName        string    `json:"first_name"`
// //     LastName         string    `json:"last_name"`
// //     PhoneNumber      string    `json:"phone_number"`
// //     Role             string    `json:"role"`
// //     Status           string    `json:"status"`
// //     EmailVerified    bool      `json:"email_verified"`
// //     TwoFactorEnabled bool      `json:"two_factor_enabled"`
// //     CreatedAt        time.Time `json:"created_at"`
// // }

// // type SessionDTO struct {
// //     ID        string    `json:"id"`
// //     UserAgent string    `json:"user_agent"`
// //     ClientIP  string    `json:"client_ip"`
// //     CreatedAt time.Time `json:"created_at"`
// //     ExpiresAt time.Time `json:"expires_at"`
// //     IsCurrent bool      `json:"is_current"`
// // }

// // type TwoFactorResponse struct {
// //     Secret     string `json:"secret"`
// //     QRCodeURL  string `json:"qr_code_url"`
// //     Enabled    bool   `json:"enabled"`
// //     Message    string `json:"message"`
// // }

// // type MessageResponse struct {
// //     Message string `json:"message"`
// // }