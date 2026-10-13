package database

import (
	"fmt"
	"log"

	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/models"

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
			&models.User{},
			&models.Project{},
			&models.ProjectOwner{},
			&models.ProjectEntry{},
			&models.ProjectDocument{},
			&models.DocumentVersion{},
			&models.PublicLink{},
		); err != nil {
			log.Fatalf("Failed to auto migrate database: %v", err)
		}
		log.Println("Database auto migration completed")
	} else {
		log.Println("Database auto migration skipped (AUTO_MIGRATE=false)")
	}
}
