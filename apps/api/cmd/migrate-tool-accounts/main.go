package main

import (
	"log"
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/database"

	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	if cfg.DBName != "atlas_db" {
		log.Fatal("Tool account migration requires atlas_db")
	}
	cfg.AutoMigrate = false
	database.Connect(cfg)
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, sql := range []string{
			"SET LOCAL lock_timeout = '5s'",
			"SET LOCAL statement_timeout = '30s'",
			"ALTER TABLE accounts ADD COLUMN IF NOT EXISTS tool_id bigint NOT NULL DEFAULT 0",
			"CREATE INDEX IF NOT EXISTS idx_accounts_tool_id ON accounts (tool_id)",
			`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'accounts'::regclass AND conname = 'accounts_single_target') THEN ALTER TABLE accounts ADD CONSTRAINT accounts_single_target CHECK ((project_id > 0 AND tool_id = 0) OR (project_id = 0 AND tool_id > 0)); END IF; END $$`,
		} {
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal("Tool account migration failed; transaction rolled back")
	}
	log.Println("Tool account column, index and target constraint ready; existing records and encryption keys unchanged")
}
