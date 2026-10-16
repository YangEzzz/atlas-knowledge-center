package repository

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/models"
)

func TestProjectSearchResourcesAndOwners(t *testing.T) {
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires dedicated atlas_db; all fixtures roll back")
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
	defer tx.Rollback()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	unique := fmt.Sprintf("atlas-ui-search-%d", time.Now().UnixNano())
	p := models.Project{Name: "搜索回归项目", Type: "backend", Owners: []models.ProjectOwner{{OwnerName: unique + "-owner", RoleType: "backend"}}, Entries: []models.ProjectEntry{{Type: "staging", Title: "测试环境", URL: "https://" + unique + ".example.com"}}}
	if err = tx.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewProjectRepository(tx)
	entries := []models.ProjectEntry{{ProjectID: p.ID, Type: "staging", Title: "测试环境", URL: p.Entries[0].URL, Remark: "仅内网访问\n先连接 VPN"}}
	if err = repo.ReplaceEntries(p.ID, entries); err != nil {
		t.Fatal(err)
	}
	var stored models.ProjectEntry
	if err = tx.Where("project_id = ?", p.ID).First(&stored).Error; err != nil || stored.Remark != entries[0].Remark {
		t.Fatal("entry remark did not persist")
	}
	entries[0].Remark = ""
	if err = repo.ReplaceEntries(p.ID, entries); err != nil {
		t.Fatal(err)
	}
	p.Entries[0] = models.ProjectEntry{}
	if err = tx.Where("project_id = ?", p.ID).First(&p.Entries[0]).Error; err != nil || p.Entries[0].Remark != "" {
		t.Fatal("entry remark could not be cleared")
	}
	check := func(keyword string, want int64) {
		t.Helper()
		items, total, err := repo.List(ProjectListFilter{Keyword: keyword, Page: 1, PageSize: 12})
		if err != nil {
			t.Fatal(err)
		}
		if total != want || int64(len(items)) != want {
			t.Fatalf("query %q returned %d/%d, want %d", keyword, len(items), total, want)
		}
	}
	check(unique+".example.com", 1)
	check(unique+"-owner", 1)
	if err = tx.Delete(&p.Entries[0]).Error; err != nil {
		t.Fatal(err)
	}
	check(unique+".example.com", 0)
	if err = tx.Delete(&p).Error; err != nil {
		t.Fatal(err)
	}
	check(unique+"-owner", 0)
}
