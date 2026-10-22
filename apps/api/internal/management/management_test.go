package management

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/lifecycle"
	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/repository"
	"ty-notebook-api/internal/service"
)

func TestManagementLifecycle(t *testing.T) {
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires atlas_db; all fixtures roll back")
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
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	key := fmt.Sprintf("management-%d", time.Now().UnixNano())
	u := models.User{ExternalUserID: key, Username: key, DisplayName: "管理回归", RoleCode: "admin", Status: "active"}
	must(tx.Create(&u).Error)
	member := models.User{ExternalUserID: key + "-member", RoleCode: "developer", Status: "active"}
	must(tx.Create(&member).Error)
	user := &u.ID
	h := &Handler{DB: tx}
	pRepo := repository.NewProjectRepository(tx)
	ps := service.NewProjectService(pRepo)
	input := service.ProjectSaveInput{Name: key + "-project", Type: "backend", Entries: []service.ProjectEntryInput{{Type: "deploy", Title: key + "-entry1", URL: "https://example.com/a"}, {Type: "deploy", Title: key + "-entry2", URL: "https://example.com/b"}}}
	p, err := ps.Create(input, user)
	must(err)
	if len(p.Entries) != 2 {
		t.Fatal("entries missing")
	}
	first, second := p.Entries[0].ID, p.Entries[1].ID
	input.Entries[0].ID = first
	input.Entries[1].ID = second
	checkProjectUpdate := func(wantProject, wantEntry int) {
		t.Helper()
		var previous models.AuditLog
		must(tx.Where("actor_id = ?", u.ID).Order("id desc").First(&previous).Error)
		var updateErr error
		p, updateErr = ps.Update(p.ID, input, user)
		must(updateErr)
		var logs []models.AuditLog
		must(tx.Where("actor_id = ? AND id > ?", u.ID, previous.ID).Find(&logs).Error)
		projects, entries := 0, 0
		for _, log := range logs {
			if log.Action != "update" {
				t.Fatalf("unexpected audit action: %s", log.Action)
			}
			switch log.ObjectType {
			case "project":
				projects++
				if strings.Contains(log.Fields, "资源入口") {
					t.Fatal("entry change duplicated in project audit")
				}
			case "entry":
				entries++
			default:
				t.Fatalf("unexpected audit type: %s", log.ObjectType)
			}
		}
		if projects != wantProject || entries != wantEntry {
			t.Fatalf("got %d project / %d entry audits; want %d / %d", projects, entries, wantProject, wantEntry)
		}
	}
	// Renaming two entries must produce precisely two object-level records.
	input.Entries[0].Title += "-renamed"
	input.Entries[1].Title += "-renamed"
	checkProjectUpdate(0, 2)
	// A real project field change adds its own record, independently of entries.
	input.Name += "-renamed"
	input.Entries[0].Title += "-again"
	input.Entries[1].Title += "-again"
	checkProjectUpdate(1, 2)
	input.Description = "project description changed"
	checkProjectUpdate(1, 0)
	checkProjectUpdate(0, 0)
	input.Entries = input.Entries[:1]
	input.Entries[0].ID = first
	input.Entries[0].Remark = "updated"
	p, err = ps.Update(p.ID, input, user)
	must(err)
	if len(p.Entries) != 1 || p.Entries[0].ID != first {
		t.Fatal("entry identity lost")
	}
	var removed models.ProjectEntry
	must(tx.Unscoped().First(&removed, second).Error)
	if !removed.DeletedAt.Valid {
		t.Fatal("entry not soft deleted")
	}
	var logCount int64
	must(tx.Model(&models.AuditLog{}).Where("actor_id = ?", u.ID).Count(&logCount).Error)
	_, err = ps.Update(p.ID, input, user)
	must(err)
	var after int64
	must(tx.Model(&models.AuditLog{}).Where("actor_id = ?", u.ID).Count(&after).Error)
	if after != logCount {
		t.Fatal("unchanged project created audit noise")
	}
	dRepo := repository.NewDocumentRepository(tx)
	ds := service.NewDocumentService(dRepo)
	d, err := ds.Save(service.DocumentSaveInput{ProjectID: &p.ID, Type: "intro", Title: key + "-doc", Content: "<p>body</p>", Status: "draft"}, user)
	must(err)
	_, err = ds.Save(service.DocumentSaveInput{ID: d.ID, ProjectID: &p.ID, Type: "intro", Title: key + "-doc", Content: "<p>changed</p>", Status: "published"}, user)
	must(err)
	versions, err := ds.ListVersions(d.ID)
	must(err)
	_, err = ds.RestoreVersion(versions[len(versions)-1].ID, service.DocumentVersionRestoreInput{}, user)
	must(err)
	must(ds.Delete(d.ID, user))
	tRepo := repository.NewPublicLinkRepository(tx)
	tool := models.PublicLink{Name: key + "-tool", Type: "other", URL: "https://example.com", CreatedBy: user, IsEnabled: false}
	must(tRepo.Create(&tool))
	if tool.IsEnabled {
		t.Fatal("disabled tool became enabled")
	}
	must(tRepo.Delete(tool.ID, user))
	must(pRepo.Delete(p.ID, user))
	if err = lifecycle.Restore(tx, "entry", second, user); !errors.Is(err, lifecycle.ErrParent) {
		t.Fatalf("expected parent conflict, got %v", err)
	}
	must(lifecycle.Restore(tx, "project", p.ID, user))
	must(tx.Unscoped().First(&removed, second).Error)
	if !removed.DeletedAt.Valid {
		t.Fatal("project restore revived independently removed entry")
	}
	must(lifecycle.Restore(tx, "entry", second, user))
	if err = lifecycle.Restore(tx, "entry", second, user); !errors.Is(err, lifecycle.ErrConflict) {
		t.Fatal("duplicate restore accepted")
	}
	// Delete again to exercise all four kinds in the union list.
	input.Entries[0].ID = first
	_, err = ps.Update(p.ID, input, user)
	must(err)
	must(pRepo.Delete(p.ID, user))
	gin.SetMode(gin.TestMode)
	makeRouter := func(id uint) *gin.Engine {
		r := gin.New()
		api := r.Group("/api/v1")
		api.Use(func(c *gin.Context) { c.Set("userID", id) })
		Register(api, tx)
		return r
	}
	r := makeRouter(u.ID)
	send := func(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var b []byte
		if body != nil {
			b, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	for _, path := range []string{"/manage/templates", "/manage/trash", "/manage/audit", "/manage/audit/1"} {
		w := send(makeRouter(member.ID), "GET", path, nil)
		if w.Code != 403 {
			t.Fatalf("non-admin accessed %s: %d", path, w.Code)
		}
	}
	w := send(r, "GET", "/manage/trash?keyword="+key, nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var body struct {
		Data struct {
			List  []TrashItem
			Total int
		}
	}
	must(json.Unmarshal(w.Body.Bytes(), &body))
	if body.Data.Total != 4 {
		t.Fatalf("trash expected four kinds: %s", w.Body.String())
	}
	for _, item := range body.Data.List {
		if item.DeletedBy == nil || *item.DeletedBy != u.ID {
			t.Fatal("delete actor missing")
		}
		if item.ObjectType == "entry" && !item.ParentDeleted {
			t.Fatal("parent state missing")
		}
	}
	w = send(r, "GET", "/manage/trash?keyword="+key+"&type=tool&pageSize=1", nil)
	if !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatal("trash filter failed")
	}
	w = send(r, "POST", fmt.Sprintf("/manage/trash/document/%d/restore", d.ID), nil)
	if w.Code != 409 {
		t.Fatal("parent conflict not exposed")
	}
	must(lifecycle.Restore(tx, "project", p.ID, user))
	must(lifecycle.Restore(tx, "document", d.ID, user))
	must(lifecycle.Restore(tx, "tool", tool.ID, user))
	// Audit failures must roll back a successful-looking business mutation.
	missing := uint(4000000000)
	err = tRepo.Delete(tool.ID, &missing)
	if err == nil {
		t.Fatal("missing actor accepted")
	}
	var still models.PublicLink
	must(tx.First(&still, tool.ID).Error)
	template := TemplateInput{Name: key + "-template1", Type: "environment", Content: "<h2>部署步骤</h2>", IsEnabled: true}
	a, err := h.SaveTemplateRecord(0, template, user)
	must(err)
	template.Name = key + "-template2"
	b, err := h.SaveTemplateRecord(0, template, user)
	must(err)
	var old models.DocumentTemplate
	must(tx.First(&old, a.ID).Error)
	if old.IsEnabled {
		t.Fatal("previous template not disabled")
	}
	var enabled int64
	must(tx.Model(&models.DocumentTemplate{}).Where("type = ? AND is_enabled", template.Type).Count(&enabled).Error)
	if enabled != 1 {
		t.Fatal("multiple active templates")
	}
	template.UpdatedAt = b.UpdatedAt
	template.IsEnabled = false
	b, err = h.SaveTemplateRecord(b.ID, template, user)
	must(err)
	_, err = h.SaveTemplateRecord(b.ID, TemplateInput{Name: "stale", Type: b.Type, Content: b.Content, UpdatedAt: template.UpdatedAt}, user)
	if !errors.Is(err, errStale) {
		t.Fatal("stale template overwrote record")
	}
	w = send(makeRouter(member.ID), "GET", "/document-templates/active/environment", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"data":null`) {
		t.Fatal("disabled template exposed")
	}
	w = send(r, "GET", "/manage/templates?keyword="+key+"&enabled=false", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"total":2`) {
		t.Fatal(w.Body.String())
	}
	w = send(r, "DELETE", fmt.Sprintf("/manage/templates/%d", b.ID), map[string]any{"updatedAt": b.UpdatedAt})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = send(r, "GET", "/manage/audit?keyword="+key+"&actor="+strconvID(u.ID)+"&action=restore&type=project", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"total":2`) {
		t.Fatal(w.Body.String())
	}
	w = send(r, "GET", "/manage/audit?from=invalid", nil)
	if w.Code != 400 {
		t.Fatal("invalid time accepted")
	}
	w = send(r, "GET", "/manage/audit?pageSize=1&actor="+strconvID(u.ID), nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var auditBody struct {
		Data struct{ List []models.AuditLog }
	}
	must(json.Unmarshal(w.Body.Bytes(), &auditBody))
	if len(auditBody.Data.List) != 1 {
		t.Fatal("audit pagination failed")
	}
	w = send(r, "GET", fmt.Sprintf("/manage/audit/%d", auditBody.Data.List[0].ID), nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "管理回归") {
		t.Fatal("audit detail missing actor snapshot")
	}
}

func strconvID(id uint) string { return fmt.Sprint(id) }
