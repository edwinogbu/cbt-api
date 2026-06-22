package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type EmailService struct {
	from      string
	host      string
	port      string
	username  string
	password  string
	templates *template.Template
}

var (
	instance *EmailService
	once     sync.Once
)

func NewEmailService() *EmailService {
	once.Do(func() {
		_ = godotenv.Load()

		instance = &EmailService{
			host:     os.Getenv("SMTP_HOST"),
			port:     os.Getenv("SMTP_PORT"),
			username: os.Getenv("SMTP_USER"),
			password: os.Getenv("SMTP_PASSWORD"),
			from:     os.Getenv("SMTP_FROM"),
		}

		// Validate SMTP config
		if instance.host == "" || instance.port == "" || instance.username == "" || instance.password == "" || instance.from == "" {
			log.Println("[EMAIL] SMTP config incomplete – emails disabled")
			return
		}

		// Find templates
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			log.Println("[EMAIL] Cannot locate source file")
			return
		}
		projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
		templatesDir := filepath.Join(projectRoot, "templates")

		var files []string
		emailDir := filepath.Join(templatesDir, "email")
		partialsDir := filepath.Join(templatesDir, "partials")
		emailFiles, _ := filepath.Glob(filepath.Join(emailDir, "*.html"))
		partialFiles, _ := filepath.Glob(filepath.Join(partialsDir, "*.html"))
		files = append(files, emailFiles...)
		files = append(files, partialFiles...)

		if len(files) == 0 {
			log.Printf("[EMAIL] No templates found in %s or %s", emailDir, partialsDir)
			return
		}

		tmpl := template.New("")
		for _, file := range files {
			if _, err := tmpl.ParseFiles(file); err != nil {
				log.Printf("[EMAIL] Warning parsing %s: %v", file, err)
			}
		}
		instance.templates = tmpl
		log.Printf("[EMAIL] Templates loaded successfully (%d files)", len(files))
	})
	return instance
}

type TemplateData struct {
	SiteName     string
	AppURL       string
	SupportEmail string
	FirstName    string
	Code         string
	Year         int
}

func (s *EmailService) SendWelcomeEmail(to, firstName, code string) error {
	if s.templates == nil {
		return fmt.Errorf("email service not initialized – templates not loaded")
	}
	data := TemplateData{
		SiteName:     getEnv("APP_NAME", "CBT Platform"),
		AppURL:       getEnv("APP_URL", "http://localhost:8080"),
		SupportEmail: getEnv("SUPPORT_EMAIL", "support@example.com"),
		FirstName:    firstName,
		Code:         code,
		Year:         time.Now().Year(),
	}
	htmlBody, err := s.renderTemplate("welcome.html", data)
	if err != nil {
		return fmt.Errorf("failed to render welcome template: %w", err)
	}
	subject := fmt.Sprintf("Welcome to %s – Verify Your Email", data.SiteName)
	return s.sendAsync(to, subject, htmlBody)
}

func (s *EmailService) SendPasswordResetEmail(to, firstName, code string) error {
	if s.templates == nil {
		return fmt.Errorf("email service not initialized – templates not loaded")
	}
	data := TemplateData{
		SiteName:     getEnv("APP_NAME", "CBT Platform"),
		AppURL:       getEnv("APP_URL", "http://localhost:8080"),
		SupportEmail: getEnv("SUPPORT_EMAIL", "support@example.com"),
		FirstName:    firstName,
		Code:         code,
		Year:         time.Now().Year(),
	}
	htmlBody, err := s.renderTemplate("reset_password.html", data)
	if err != nil {
		return fmt.Errorf("failed to render reset password template: %w", err)
	}
	subject := fmt.Sprintf("%s – Password Reset Request", data.SiteName)
	return s.sendAsync(to, subject, htmlBody)
}

func (s *EmailService) renderTemplate(templateName string, data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *EmailService) sendAsync(to, subject, htmlBody string) error {
	if s.host == "" {
		log.Printf("[EMAIL] Skipping send to %s: SMTP not configured", to)
		return nil
	}
	go func() {
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			err := s.sendSMTP(to, subject, htmlBody)
			if err == nil {
				log.Printf("[EMAIL] Sent to %s (subject: %s)", to, subject)
				return
			}
			log.Printf("[EMAIL] Attempt %d failed for %s: %v", i+1, to, err)
			if i < maxRetries-1 {
				time.Sleep(time.Duration(1<<uint(i)) * time.Second)
			}
		}
		log.Printf("[EMAIL] All attempts failed for %s", to)
	}()
	return nil
}

// sendSMTP – uses explicit TLS on port 465 if STARTTLS fails, logs details
func (s *EmailService) sendSMTP(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Build email message
	headers := map[string]string{
		"From":         s.from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}
	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + htmlBody

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// Try STARTTLS (Mailtrap requires it)
	client, err := smtp.Dial(addr)
	if err != nil {
		log.Printf("[EMAIL] Dial error: %v", err)
		return err
	}
	defer client.Close()

	if err = client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
		log.Printf("[EMAIL] StartTLS error: %v", err)
		return err
	}
	if err = client.Auth(auth); err != nil {
		log.Printf("[EMAIL] Auth error: %v", err)
		return err
	}
	if err = client.Mail(s.from); err != nil {
		log.Printf("[EMAIL] Mail command error: %v", err)
		return err
	}
	if err = client.Rcpt(to); err != nil {
		log.Printf("[EMAIL] Rcpt command error: %v", err)
		return err
	}
	w, err := client.Data()
	if err != nil {
		log.Printf("[EMAIL] Data command error: %v", err)
		return err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		log.Printf("[EMAIL] Write error: %v", err)
		return err
	}
	if err = w.Close(); err != nil {
		log.Printf("[EMAIL] Close error: %v", err)
		return err
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}






// package email

// import (
// 	"bytes"
// 	"crypto/tls"
// 	"fmt"
// 	"html/template"
// 	"log"
// 	"net/smtp"
// 	"os"
// 	"path/filepath"
// 	"runtime"
// 	"sync"
// 	"time"

// 	"github.com/joho/godotenv"
// )

// type EmailService struct {
// 	from      string
// 	host      string
// 	port      string
// 	username  string
// 	password  string
// 	templates *template.Template
// }

// var (
// 	instance *EmailService
// 	once     sync.Once
// )

// // NewEmailService initializes the email service (singleton)
// func NewEmailService() *EmailService {
// 	once.Do(func() {
// 		_ = godotenv.Load()

// 		instance = &EmailService{
// 			host:     os.Getenv("SMTP_HOST"),
// 			port:     os.Getenv("SMTP_PORT"),
// 			username: os.Getenv("SMTP_USER"),
// 			password: os.Getenv("SMTP_PASSWORD"),
// 			from:     os.Getenv("SMTP_FROM"),
// 		}

// 		if instance.host == "" || instance.port == "" || instance.username == "" || instance.password == "" {
// 			log.Println("[EMAIL] SMTP config incomplete – emails disabled")
// 			return
// 		}

// 		// Locate project root using the source file's location
// 		_, currentFile, _, ok := runtime.Caller(0)
// 		if !ok {
// 			log.Println("[EMAIL] Cannot locate source file")
// 			return
// 		}
// 		projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
// 		templatesDir := filepath.Join(projectRoot, "templates")

// 		// Collect all HTML files from email and partials folders
// 		var files []string
// 		emailDir := filepath.Join(templatesDir, "email")
// 		partialsDir := filepath.Join(templatesDir, "partials")
// 		emailFiles, _ := filepath.Glob(filepath.Join(emailDir, "*.html"))
// 		partialFiles, _ := filepath.Glob(filepath.Join(partialsDir, "*.html"))
// 		files = append(files, emailFiles...)
// 		files = append(files, partialFiles...)

// 		if len(files) == 0 {
// 			log.Printf("[EMAIL] No templates found in %s or %s", emailDir, partialsDir)
// 			return
// 		}

// 		// Parse files one by one; duplicate definitions are logged but do not stop loading
// 		tmpl := template.New("")
// 		for _, file := range files {
// 			if _, err := tmpl.ParseFiles(file); err != nil {
// 				log.Printf("[EMAIL] Warning parsing %s: %v", file, err)
// 			}
// 		}
// 		instance.templates = tmpl
// 		log.Printf("[EMAIL] Templates loaded successfully (%d files)", len(files))
// 	})
// 	return instance
// }

// // TemplateData holds common data passed to email templates
// type TemplateData struct {
// 	SiteName     string
// 	AppURL       string
// 	SupportEmail string
// 	FirstName    string
// 	Code         string
// 	Year         int
// }

// // SendWelcomeEmail sends a welcome email with OTP verification code
// func (s *EmailService) SendWelcomeEmail(to, firstName, code string) error {
// 	if s.templates == nil {
// 		return fmt.Errorf("email service not initialized – templates not loaded")
// 	}
// 	data := TemplateData{
// 		SiteName:     getEnv("APP_NAME", "CBT Platform"),
// 		AppURL:       getEnv("APP_URL", "http://localhost:8080"),
// 		SupportEmail: getEnv("SUPPORT_EMAIL", "support@example.com"),
// 		FirstName:    firstName,
// 		Code:         code,
// 		Year:         time.Now().Year(),
// 	}
// 	htmlBody, err := s.renderTemplate("welcome.html", data)
// 	if err != nil {
// 		return fmt.Errorf("failed to render welcome template: %w", err)
// 	}
// 	subject := fmt.Sprintf("Welcome to %s – Verify Your Email", data.SiteName)
// 	return s.sendAsync(to, subject, htmlBody)
// }

// // SendPasswordResetEmail sends a password reset email with OTP code
// func (s *EmailService) SendPasswordResetEmail(to, firstName, code string) error {
// 	if s.templates == nil {
// 		return fmt.Errorf("email service not initialized – templates not loaded")
// 	}
// 	data := TemplateData{
// 		SiteName:     getEnv("APP_NAME", "CBT Platform"),
// 		AppURL:       getEnv("APP_URL", "http://localhost:8080"),
// 		SupportEmail: getEnv("SUPPORT_EMAIL", "support@example.com"),
// 		FirstName:    firstName,
// 		Code:         code,
// 		Year:         time.Now().Year(),
// 	}
// 	htmlBody, err := s.renderTemplate("reset_password.html", data)
// 	if err != nil {
// 		return fmt.Errorf("failed to render reset password template: %w", err)
// 	}
// 	subject := fmt.Sprintf("%s – Password Reset Request", data.SiteName)
// 	return s.sendAsync(to, subject, htmlBody)
// }

// // renderTemplate executes a named template with the given data
// func (s *EmailService) renderTemplate(templateName string, data TemplateData) (string, error) {
// 	var buf bytes.Buffer
// 	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
// 		return "", err
// 	}
// 	return buf.String(), nil
// }

// // sendAsync sends email asynchronously with retries (non‑blocking)
// func (s *EmailService) sendAsync(to, subject, htmlBody string) error {
// 	if s.host == "" {
// 		log.Printf("[EMAIL] Skipping send to %s: SMTP not configured", to)
// 		return nil
// 	}
// 	go func() {
// 		maxRetries := 3
// 		for i := 0; i < maxRetries; i++ {
// 			err := s.sendSMTP(to, subject, htmlBody)
// 			if err == nil {
// 				log.Printf("[EMAIL] Sent to %s (subject: %s)", to, subject)
// 				return
// 			}
// 			log.Printf("[EMAIL] Attempt %d failed for %s: %v", i+1, to, err)
// 			if i < maxRetries-1 {
// 				time.Sleep(time.Duration(1<<uint(i)) * time.Second)
// 			}
// 		}
// 		log.Printf("[EMAIL] All attempts failed for %s", to)
// 	}()
// 	return nil
// }

// // sendSMTP sends the email synchronously via SMTP
// func (s *EmailService) sendSMTP(to, subject, htmlBody string) error {
// 	addr := fmt.Sprintf("%s:%s", s.host, s.port)

// 	headers := map[string]string{
// 		"From":         s.from,
// 		"To":           to,
// 		"Subject":      subject,
// 		"MIME-Version": "1.0",
// 		"Content-Type": "text/html; charset=UTF-8",
// 	}
// 	message := ""
// 	for k, v := range headers {
// 		message += fmt.Sprintf("%s: %s\r\n", k, v)
// 	}
// 	message += "\r\n" + htmlBody

// 	auth := smtp.PlainAuth("", s.username, s.password, s.host)
// 	client, err := smtp.Dial(addr)
// 	if err != nil {
// 		return fmt.Errorf("dial failed: %w", err)
// 	}
// 	defer client.Close()

// 	if err = client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
// 		return fmt.Errorf("starttls failed: %w", err)
// 	}
// 	if err = client.Auth(auth); err != nil {
// 		return fmt.Errorf("auth failed: %w", err)
// 	}
// 	if err = client.Mail(s.from); err != nil {
// 		return fmt.Errorf("mail command failed: %w", err)
// 	}
// 	if err = client.Rcpt(to); err != nil {
// 		return fmt.Errorf("rcpt command failed: %w", err)
// 	}
// 	w, err := client.Data()
// 	if err != nil {
// 		return fmt.Errorf("data command failed: %w", err)
// 	}
// 	if _, err = w.Write([]byte(message)); err != nil {
// 		return fmt.Errorf("write failed: %w", err)
// 	}
// 	if err = w.Close(); err != nil {
// 		return fmt.Errorf("close failed: %w", err)
// 	}
// 	return nil
// }

// // getEnv reads an environment variable with a fallback value
// func getEnv(key, fallback string) string {
// 	if val := os.Getenv(key); val != "" {
// 		return val
// 	}
// 	return fallback
// }