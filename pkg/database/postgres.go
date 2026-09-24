package database

import (
    "fmt"
    "log"
    "os"

    "cbt-api/internal/models"
    "cbt-api/pkg/utils"

    "github.com/joho/godotenv"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }

    // DATABASE_URL (a full postgres:// connection string, as managed
    // providers like Neon issue) takes precedence - it can carry
    // parameters the key=value DSN below can't express cleanly, like
    // Neon's channel_binding=require. Falls back to the individual DB_*
    // vars for local/dev use.
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
            os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
            os.Getenv("DB_NAME"), os.Getenv("DB_PORT"), os.Getenv("DB_SSLMODE"))
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    fmt.Println("✅ Database connected")

    if os.Getenv("AUTO_MIGRATE") == "true" {
        log.Println("🔄 Running auto-migration...")
        if err := db.AutoMigrate(models.GetAllModels()...); err != nil {
            log.Fatal("Migration failed:", err)
        }
        fmt.Printf("✅ Synced %d tables\n", len(models.GetAllModels()))
        for _, name := range models.GetModelNames() {
            fmt.Printf("   - %s\n", name)
        }
    }

    DB = db
    seedAdmin()
}

// func seedAdmin() {
//     var admin models.User
//     result := DB.Where("email = ?", os.Getenv("ADMIN_EMAIL")).First(&admin)

//     if result.Error != nil {
//         hashedPassword, _ := utils.HashPassword(os.Getenv("ADMIN_PASSWORD"))
//         admin = models.User{
//             Email:       os.Getenv("ADMIN_EMAIL"),
//             Password:    hashedPassword,
//             FirstName:   "Admin",
//             LastName:    "User",
//             Role:        models.RoleAdmin,
//             Status:      models.StatusActive,
//             IsActive:    true,
//         }
//         DB.Create(&admin)
//         fmt.Println("✅ Admin user created")
//         fmt.Printf("   Email: %s\n", os.Getenv("ADMIN_EMAIL"))
//         fmt.Printf("   Password: %s\n", os.Getenv("ADMIN_PASSWORD"))
//     }
// }

func seedAdmin() {
    var admin models.User
    adminEmailStr := os.Getenv("ADMIN_EMAIL")
    result := DB.Where("email = ?", adminEmailStr).First(&admin)

    if result.Error != nil {
        hashedPassword, _ := utils.HashPassword(os.Getenv("ADMIN_PASSWORD"))
        var emailPtr *string
        if adminEmailStr != "" {
            emailPtr = &adminEmailStr
        }
        admin = models.User{
            Email:       emailPtr,
            Password:    hashedPassword,
            FirstName:   "Admin",
            LastName:    "User",
            Role:        models.RoleAdmin,
            Status:      models.StatusActive,
            IsActive:    true,
        }
        DB.Create(&admin)
        fmt.Println("✅ Admin user created")
        fmt.Printf("   Email: %s\n", adminEmailStr)
        fmt.Printf("   Password: %s\n", os.Getenv("ADMIN_PASSWORD"))
    }
}