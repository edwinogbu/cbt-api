package config

import (
    "sync"
)

var (
    once     sync.Once
    instance *Config
)

// Get returns the singleton config instance
func Get() *Config {
    once.Do(func() {
        instance = LoadConfig()
    })
    return instance
}

// GetDSN returns the PostgreSQL connection string from the global config
func GetDSN() string {
    return Get().Database.GetDSN()
}

// IsProduction checks if the app is running in production mode
func IsProduction() bool {
    return Get().Server.IsProduction()
}

// IsDevelopment checks if the app is running in development mode
func IsDevelopment() bool {
    return Get().Server.IsDevelopment()
}