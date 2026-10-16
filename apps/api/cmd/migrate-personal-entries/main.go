package main

import (
	"gorm.io/gorm"
	"log"
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/database"
	"ty-notebook-api/internal/models"
)

func main() {
	cfg := config.Load()
	if cfg.DBName != "atlas_db" {
		log.Fatal("Personal entry migration requires atlas_db")
	}
	cfg.AutoMigrate = false
	database.Connect(cfg)
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL lock_timeout='5s'").Error; err != nil {
			return err
		}
		if err := tx.AutoMigrate(&models.PersonalEntry{}); err != nil {
			return err
		}
		for _, sql := range []string{
			"CREATE UNIQUE INDEX IF NOT EXISTS personal_entry_reference ON personal_entries(user_id,source,target_id) WHERE source <> 'private'",
			"ALTER TABLE accounts ADD COLUMN IF NOT EXISTS entry_id bigint NOT NULL DEFAULT 0",
			"ALTER TABLE accounts ADD COLUMN IF NOT EXISTS entry_cascade boolean NOT NULL DEFAULT false",
			"CREATE INDEX IF NOT EXISTS idx_accounts_entry_id ON accounts(entry_id)",
			"ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_single_target",
			"ALTER TABLE accounts ADD CONSTRAINT accounts_single_target CHECK ((project_id>0 AND tool_id=0 AND entry_id=0) OR (project_id=0 AND tool_id>0 AND entry_id=0) OR (project_id=0 AND tool_id=0 AND entry_id>0))",
		} {
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal("Personal entry migration failed and rolled back")
	}
	log.Println("Personal entry schema ready; existing accounts and encryption keys unchanged")
}
