package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"ty-notebook-api/internal/accounts"
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/database"
)

func main() {
	cfg := config.Load()
	if cfg.DBName != "atlas_db" {
		log.Fatal("Personal accounts setup requires atlas_db")
	}
	cfg.AutoMigrate = false
	database.Connect(cfg)
	if cfg.AccountKeyID == "" && cfg.AccountKeys == "" {
		if database.DB.Migrator().HasTable(&accounts.Account{}) {
			var count int64
			if err := database.DB.Unscoped().Model(&accounts.Account{}).Count(&count).Error; err != nil || count > 0 {
				log.Fatal("Existing accounts require the original encryption keys; restore the key configuration before setup")
			}
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatal("Unable to generate encryption key")
		}
		raw, _ := json.Marshal(map[string]string{"v1": base64.StdEncoding.EncodeToString(key)})
		target := ".env"
		if _, err := os.Stat(".env-production"); err == nil {
			target = ".env-production"
		}
		file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal("Unable to persist encryption configuration")
		}
		_, err = file.WriteString("\nACCOUNT_KEY_ID=v1\nACCOUNT_KEYS='" + string(raw) + "'\nACCOUNT_REVEAL_POLICY=session\n")
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			log.Fatal("Unable to persist encryption configuration")
		}
		cfg.AccountKeyID = "v1"
		cfg.AccountKeys = string(raw)
	}
	if _, err := accounts.NewCrypt(cfg.AccountKeyID, cfg.AccountKeys); err != nil {
		log.Fatal("Invalid account key configuration; existing keys were not replaced")
	}
	if err := database.DB.AutoMigrate(&accounts.Account{}); err != nil {
		log.Fatal("Unable to migrate personal accounts")
	}
	log.Println("Personal accounts ready. Back up encryption keys separately from the database; keys were not printed.")
}
