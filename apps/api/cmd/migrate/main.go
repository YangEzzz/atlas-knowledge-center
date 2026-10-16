package main

import (
	"flag"
	"log"
	"ty-notebook-api/internal/accounts"
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/database"
	"ty-notebook-api/internal/models"
)

func main() {
	entryRemark := flag.Bool("entry-remark", false, "Add only the optional project entry remark column")
	personalAccounts := flag.Bool("accounts", false, "Create only the personal accounts table")
	multipleEntries := flag.Bool("multiple-project-entries", false, "Allow multiple project entries of the same type")
	flag.Parse()
	cfg := config.Load()
	if cfg.DBName != "atlas_db" {
		log.Fatal("Atlas initialization only supports the dedicated atlas_db database")
	}
	if *multipleEntries {
		cfg.AutoMigrate = false
		database.Connect(cfg)
		if err := database.AllowMultipleProjectEntries(database.DB); err != nil {
			log.Fatal("Unable to remove legacy project entry index: ", err)
		}
		log.Println("Multiple project entries of the same type enabled")
		return
	}
	if *entryRemark {
		cfg.AutoMigrate = false
		database.Connect(cfg)
		if !database.DB.Migrator().HasColumn(&models.ProjectEntry{}, "Remark") {
			if err := database.DB.Migrator().AddColumn(&models.ProjectEntry{}, "Remark"); err != nil {
				log.Fatal("Unable to add project entry remark column")
			}
		}
		log.Println("Project entry remark column ready")
		return
	}
	if *personalAccounts {
		cfg.AutoMigrate = false
		database.Connect(cfg)
		if err := database.DB.AutoMigrate(&accounts.Account{}); err != nil {
			log.Fatal("Unable to migrate personal accounts")
		}
		log.Println("Personal accounts table ready")
		return
	}
	cfg.AutoMigrate = true
	database.Connect(cfg)
}
