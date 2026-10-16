package database_test

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/database"
	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/repository"
)

func TestMultipleProjectEntriesMigrationAndSave(t *testing.T) {
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires dedicated atlas_db; all changes roll back")
	}
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if env["DB_NAME"] != "atlas_db" {
		t.Fatal("requires atlas_db")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(env["DB_USER"], env["DB_PASSWORD"]), Host: env["DB_HOST"] + ":" + env["DB_PORT"], Path: "/atlas_db"}
	q := dsn.Query()
	q.Set("sslmode", env["DB_SSLMODE"])
	dsn.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("unable to open Atlas database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	for range 2 {
		if err := database.AllowMultipleProjectEntries(tx); err != nil {
			t.Fatal(err)
		}
	}
	p := models.Project{Name: "同类型发布入口回归", Type: "backend"}
	if err := tx.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewProjectRepository(tx)
	entries := []models.ProjectEntry{
		{ProjectID: p.ID, Type: "deploy", Title: "测试发布系统", URL: "https://deploy-test.example.com", Remark: "测试环境", SortOrder: 0},
		{ProjectID: p.ID, Type: "deploy", Title: "线上发布系统", URL: "https://deploy.example.com", Remark: "线上环境", SortOrder: 1},
	}
	for pass := range 2 {
		if pass == 1 {
			entries[0], entries[1] = entries[1], entries[0]
			for i := range entries {
				entries[i].ID = 0
				entries[i].SortOrder = i
			}
			entries[0].Remark = "更新后的线上说明"
		}
		if err := repo.ReplaceEntries(p.ID, entries); err != nil {
			t.Fatal(err)
		}
		stored, err := repo.FindByID(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(stored.Entries) != 2 {
			t.Fatalf("want two deploy entries, got %d", len(stored.Entries))
		}
		for i, entry := range stored.Entries {
			want := entries[i]
			if entry.Type != "deploy" || entry.Title != want.Title || entry.URL != want.URL || entry.Remark != want.Remark || entry.SortOrder != i {
				t.Fatalf("entry %d did not survive save/reorder: %#v", i, entry)
			}
		}
	}
}
