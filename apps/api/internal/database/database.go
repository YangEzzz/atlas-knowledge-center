package database

import (
	"fmt"
	"log"
	"ty-notebook-api/internal/accounts"

	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/sso"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg config.Config) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}

	DB = db
	log.Println("Database connection successfully opened")

	if cfg.AutoMigrate {
		if err := DB.AutoMigrate(
			&models.PersonalEntry{},
			&accounts.Account{},
			&models.User{},
			&models.Project{},
			&models.ProjectOwner{},
			&models.ProjectEntry{},
			&models.ProjectDocument{},
			&models.DocumentVersion{},
			&models.PublicLink{},
			&sso.Session{},
			&sso.LoginTransaction{},
		); err != nil {
			log.Fatalf("Failed to auto migrate database: %v", err)
		}
		if err := AllowMultipleProjectEntries(DB); err != nil {
			log.Fatalf("Failed to migrate project entry index: %v", err)
		}
		log.Println("Database auto migration completed")
	} else {
		log.Println("Database auto migration skipped (AUTO_MIGRATE=false)")
	}
}

// AllowMultipleProjectEntries removes the legacy one-entry-per-type constraint.
// AutoMigrate does not remove indexes that disappear from model tags.
func AllowMultipleProjectEntries(db *gorm.DB) error {
	return db.Exec("DROP INDEX IF EXISTS idx_project_entry_unique").Error
}
