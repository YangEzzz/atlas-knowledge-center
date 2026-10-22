package database_test

import (
	"encoding/json"
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
	"ty-notebook-api/internal/service"
)

func TestStandaloneDocumentLifecycle(t *testing.T) {
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires dedicated atlas_db; schema and fixtures roll back")
	}
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if env["DB_NAME"] != "atlas_db" {
		t.Fatal("dedicated atlas_db required")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(env["DB_USER"], env["DB_PASSWORD"]), Host: env["DB_HOST"] + ":" + env["DB_PORT"], Path: "/atlas_db"}
	q := dsn.Query()
	q.Set("sslmode", env["DB_SSLMODE"])
	dsn.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("database unavailable")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	defer tx.Rollback()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(tx.Error)
	must(database.AllowStandaloneDocuments(tx))
	must(database.AllowStandaloneDocuments(tx))
	ds := service.NewDocumentService(repository.NewDocumentRepository(tx))
	p := models.Project{Name: "独立文档事务回归", Type: "backend"}
	must(tx.Create(&p).Error)
	input := service.DocumentSaveInput{Type: "general", Title: "独立文档 A", Content: "<p>第一篇正文</p>", Status: "draft"}
	a, err := ds.Save(input, nil)
	must(err)
	input.Title = "独立文档 B"
	b, err := ds.Save(input, nil)
	must(err)
	if a.ID == b.ID || a.ProjectID != nil || b.ProjectID != nil {
		t.Fatal("standalone documents must be independent and have null projects")
	}
	data, err := json.Marshal(a)
	must(err)
	var jsonDoc map[string]any
	must(json.Unmarshal(data, &jsonDoc))
	if jsonDoc["projectId"] != nil {
		t.Fatal("API must return projectId:null")
	}
	input.ID = a.ID
	input.Title = "更新 A"
	input.Status = "published"
	a, err = ds.Save(input, nil)
	must(err)
	if a.CurrentVersionNo != 2 || a.Status != "published" {
		t.Fatal("update must target document ID and create a version")
	}
	untouched, err := ds.Get(b.ID)
	must(err)
	if untouched.Title != "独立文档 B" || untouched.CurrentVersionNo != 1 {
		t.Fatal("another document was overwritten")
	}
	list, total, err := ds.List(repository.DocumentListFilter{Keyword: "更新 A"})
	must(err)
	if total != 1 || list[0].ID != a.ID {
		t.Fatal("standalone document missing from search/list")
	}
	input.ProjectID = &p.ID
	input.Type = "intro"
	a, err = ds.Save(input, nil)
	must(err)
	legacy, err := ds.GetByProjectAndType(p.ID, "intro")
	must(err)
	if legacy.ID != a.ID {
		t.Fatal("project entry must open the same document")
	}
	// Saving a new object in an occupied project slot must not upsert another document.
	_, err = ds.Save(service.DocumentSaveInput{ProjectID: &p.ID, Type: "intro", Title: "不能覆盖 A"}, nil)
	if err == nil {
		t.Fatal("new document silently overwrote existing project content")
	}
	input.ProjectID = nil
	a, err = ds.Save(input, nil)
	must(err)
	if a.ProjectID != nil {
		t.Fatal("project association must be removable by ID")
	}
	must(ds.Delete(a.ID, nil))
	deleted, _, err := ds.ListDeleted(repository.DocumentListFilter{Type: "intro"})
	must(err)
	found := false
	for _, d := range deleted {
		found = found || d.ID == a.ID
	}
	if !found {
		t.Fatal("standalone document missing from trash")
	}
	a, err = ds.Restore(a.ID)
	must(err)
	versions, err := ds.ListVersions(a.ID)
	must(err)
	if len(versions) != 4 {
		t.Fatalf("want 4 versions, got %d", len(versions))
	}
	a, err = ds.RestoreVersion(versions[len(versions)-1].ID, service.DocumentVersionRestoreInput{}, nil)
	must(err)
	if a.ProjectID != nil || a.Content != "<p>第一篇正文</p>" {
		t.Fatal("version restore requires no project")
	}
	var audits []models.AuditLog
	must(tx.Where("object_type = ? AND object_id = ?", "document", a.ID).Find(&audits).Error)
	if len(audits) < 6 {
		t.Fatal("lifecycle audit missing")
	}
	if audits[len(audits)-1].ProjectID != nil {
		t.Fatal("standalone audit must keep null project")
	}
	// A deleted project must still block restoring its associated documents.
	projectDoc, err := ds.Save(service.DocumentSaveInput{ProjectID: &p.ID, Type: "faq", Title: "项目文档"}, nil)
	must(err)
	must(ds.Delete(projectDoc.ID, nil))
	must(tx.Delete(&p).Error)
	if _, err = ds.Restore(projectDoc.ID); err == nil {
		t.Fatal("deleted project must still block restoring child document")
	}
	// Failed writes must leave no orphan content/version/audit rows.
	bad := uint(0)
	if _, err = ds.Save(service.DocumentSaveInput{ProjectID: &bad, Type: "general", Title: "无效项目"}, nil); err == nil {
		t.Fatal("zero project ID must be rejected")
	}
}
