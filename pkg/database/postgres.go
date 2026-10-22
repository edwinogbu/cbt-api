package database

import (
    "fmt"
    "log"
    "os"
    "strings"

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
    seedPlatformOwner()
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

// seedAdmin creates the first admin account on a fresh database, but only
// when ADMIN_EMAIL and ADMIN_PASSWORD are both set - matching the behavior
// .env.school-node.example already documents ("Leave blank to skip and
// create the first admin manually instead"), which this function never
// actually implemented: it used to run unconditionally, querying for an
// admin with email="" and, finding none, inserting one with a nil email
// and a hash of the empty string. On a database that already had such a
// row (e.g. from a prior run before this fix), that insert then failed on
// a unique-constraint violation that was never checked - and the success
// message printed anyway, regardless of whether anything was created.
func seedAdmin() {
    adminEmailStr := os.Getenv("ADMIN_EMAIL")
    adminPasswordStr := os.Getenv("ADMIN_PASSWORD")
    if adminEmailStr == "" || adminPasswordStr == "" {
        return
    }

    var admin models.User
    result := DB.Where("email = ?", adminEmailStr).First(&admin)
    if result.Error == nil {
        return // admin with this email already exists
    }

    hashedPassword, err := utils.HashPassword(adminPasswordStr)
    if err != nil {
        log.Printf("❌ Admin user not created: failed to hash ADMIN_PASSWORD: %v", err)
        return
    }
    admin = models.User{
        Email:       &adminEmailStr,
        Password:    hashedPassword,
        FirstName:   "Admin",
        LastName:    "User",
        Role:        models.RoleAdmin,
        Status:      models.StatusActive,
        IsActive:    true,
    }
    if err := DB.Create(&admin).Error; err != nil {
        log.Printf("❌ Admin user not created: %v", err)
        return
    }
    fmt.Println("✅ Admin user created")
    fmt.Printf("   Email: %s\n", adminEmailStr)
    fmt.Printf("   Password: %s\n", adminPasswordStr)
}

// seedPlatformOwner is the ONLY way a user can ever receive
// models.RolePlatformOwner as their primary role (see that constant's
// doc comment - AuthService.GrantRole explicitly refuses to grant it,
// and public registration's role allowlist doesn't include it either).
// It mirrors seedAdmin's gating: runs only when both
// PLATFORM_OWNER_EMAIL and PLATFORM_OWNER_PASSWORD are set (an operator
// deliberately provisioning the account via environment/secrets
// configuration, never a request parameter), and is a no-op if a user
// with that email already exists - so it is safe to leave these env
// vars set across every restart without re-seeding or erroring.
//
// The created account has no SchoolID: a platform owner is not scoped
// to any school, by design (see the role's doc comment).
func seedPlatformOwner() {
    ownerEmailStr := os.Getenv("PLATFORM_OWNER_EMAIL")
    ownerPasswordStr := os.Getenv("PLATFORM_OWNER_PASSWORD")
    if ownerEmailStr == "" || ownerPasswordStr == "" {
        return
    }

    var existing models.User
    result := DB.Where("email = ?", ownerEmailStr).First(&existing)
    if result.Error == nil {
        return // a user with this email already exists - nothing to do
    }

    hashedPassword, err := utils.HashPassword(ownerPasswordStr)
    if err != nil {
        log.Printf("❌ Platform owner not created: failed to hash PLATFORM_OWNER_PASSWORD: %v", err)
        return
    }

    owner := models.User{
        Username:  uniquePlatformOwnerUsername(ownerEmailStr),
        Email:     &ownerEmailStr,
        Password:  hashedPassword,
        FirstName: "Platform",
        LastName:  "Owner",
        Role:      models.RolePlatformOwner,
        Status:    models.StatusActive,
        IsActive:  true,
    }
    if err := DB.Create(&owner).Error; err != nil {
        log.Printf("❌ Platform owner not created: %v", err)
        return
    }
    fmt.Println("✅ Platform owner created")
    fmt.Printf("   Email: %s\n", ownerEmailStr)
}

// uniquePlatformOwnerUsername derives a username from the bootstrap
// email's local part. users.username has a NOT NULL unique index, so
// leaving it at its zero value (as seedAdmin historically did - see
// that function's doc comment) risks colliding with any other seeded
// user that also left it blank. Falls back to a numeric suffix on
// collision, checked against the real table rather than assumed unique.
func uniquePlatformOwnerUsername(email string) string {
    base := sanitizeUsername("owner_" + strings.SplitN(email, "@", 2)[0])
    if base == "" {
        base = "platform_owner"
    }
    username := base
    for suffix := 2; ; suffix++ {
        var existing models.User
        if err := DB.Where("username = ?", username).First(&existing).Error; err != nil {
            return username // not found in users table - available
        }
        username = fmt.Sprintf("%s%d", base, suffix)
    }
}

func sanitizeUsername(s string) string {
    var b strings.Builder
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
            b.WriteRune(r)
        }
    }
    return b.String()
}