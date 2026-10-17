# Migrations

This project's normal schema management is GORM's `AutoMigrate`
(`pkg/database/postgres.go`, gated by `AUTO_MIGRATE=true`) - it creates
and updates tables from the Go model structs automatically on boot.

Files in this directory are the exception: migrations that are
deliberately **not** expressed as a GORM struct tag and so are **not**
run automatically. Each file explains why in its own header comment -
usually because applying it safely requires a manual check against
production data first (e.g. scanning for existing duplicate rows
before adding a uniqueness constraint) that AutoMigrate has no
mechanism to perform, and which this Claude session could not run
itself for lack of production database access.

Apply these by hand, in order, following each file's own instructions.
