package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Email    EmailConfig
    Redis    RedisConfig
    App      AppConfig
}

type ServerConfig struct {
    Port         string
    Environment  string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

type DatabaseConfig struct {
    Host            string
    Port            string
    User            string
    Password        string
    DBName          string
    SSLMode         string
    Timezone        string
    MaxConnections  int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type JWTConfig struct {
    Secret             string
    RefreshSecret      string
    AccessExpiry       time.Duration
    RefreshExpiry      time.Duration
    Issuer             string
    Audience           string
}

type EmailConfig struct {
    Host       string
    Port       int
    Username   string
    Password   string
    From       string
    FromName   string
    UseTLS     bool
}

type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
    PoolSize int
}

type AppConfig struct {
    Name     string
    Version  string
    Debug    bool
    Timezone string
}

// LoadConfig loads all configuration from environment variables
func LoadConfig() *Config {
    return &Config{
        Server: ServerConfig{
            Port:         getEnv("PORT", "8080"),
            Environment:  getEnv("ENVIRONMENT", "development"),
            ReadTimeout:  getEnvDuration("READ_TIMEOUT", 30),
            WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 30),
            IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 60),
        },
        Database: DatabaseConfig{
            Host:            getEnv("DB_HOST", "localhost"),
            Port:            getEnv("DB_PORT", "5432"),
            User:            getEnv("DB_USER", "postgres"),
            Password:        getEnv("DB_PASSWORD", "postgres"),
            DBName:          getEnv("DB_NAME", "cbt_db"),
            SSLMode:         getEnv("DB_SSLMODE", "disable"),
            Timezone:        getEnv("DB_TIMEZONE", "UTC"),
            MaxConnections:  getEnvInt("DB_MAX_CONNECTIONS", 25),
            MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
            ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 300),
        },
        JWT: JWTConfig{
            Secret:        getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
            RefreshSecret: getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-key-change-in-production"),
            AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 24),      // 24 hours
            RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 168),     // 7 days
            Issuer:        getEnv("JWT_ISSUER", "cbt-api"),
            Audience:      getEnv("JWT_AUDIENCE", "cbt-api-users"),
        },
        Email: EmailConfig{
            Host:     getEnv("EMAIL_HOST", "smtp.gmail.com"),
            Port:     getEnvInt("EMAIL_PORT", 587),
            Username: getEnv("EMAIL_USERNAME", ""),
            Password: getEnv("EMAIL_PASSWORD", ""),
            From:     getEnv("EMAIL_FROM", "noreply@cbt-api.com"),
            FromName: getEnv("EMAIL_FROM_NAME", "CBT API"),
            UseTLS:   getEnvBool("EMAIL_USE_TLS", true),
        },
        Redis: RedisConfig{
            Host:     getEnv("REDIS_HOST", "localhost"),
            Port:     getEnv("REDIS_PORT", "6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
            PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
        },
        App: AppConfig{
            Name:     getEnv("APP_NAME", "CBT API"),
            Version:  getEnv("APP_VERSION", "1.0.0"),
            Debug:    getEnvBool("APP_DEBUG", true),
            Timezone: getEnv("APP_TIMEZONE", "UTC"),
        },
    }
}

// Helper functions
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolVal, err := strconv.ParseBool(value); err == nil {
            return boolVal
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue int) time.Duration {
    return time.Duration(getEnvInt(key, defaultValue)) * time.Hour
}

// GetDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
    return "host=" + c.Host +
        " user=" + c.User +
        " password=" + c.Password +
        " dbname=" + c.DBName +
        " port=" + c.Port +
        " sslmode=" + c.SSLMode +
        " TimeZone=" + c.Timezone
}

// IsProduction returns true if environment is production
func (c *ServerConfig) IsProduction() bool {
    return c.Environment == "production"
}

// IsDevelopment returns true if environment is development
func (c *ServerConfig) IsDevelopment() bool {
    return c.Environment == "development"
}

// IsTest returns true if environment is test
func (c *ServerConfig) IsTest() bool {
    return c.Environment == "test"
}