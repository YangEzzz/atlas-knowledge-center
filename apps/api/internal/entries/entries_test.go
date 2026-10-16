package entries

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"ty-notebook-api/internal/accounts"
	"ty-notebook-api/internal/models"
)

func fixture(t *testing.T) (*gorm.DB, *accounts.Crypt) {
	t.Helper()
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires atlas_db; all fixture writes roll back")
	}
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil || env["DB_NAME"] != "atlas_db" {
		t.Fatal("requires dedicated atlas_db")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(env["DB_USER"], env["DB_PASSWORD"]), Host: env["DB_HOST"] + ":" + env["DB_PORT"], Path: "/atlas_db"}
	q := dsn.Query()
	q.Set("sslmode", env["DB_SSLMODE"])
	dsn.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("database unavailable")
	}
	conn, _ := db.DB()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback(); conn.Close() })
	key, _ := json.Marshal(map[string]string{"fixture": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{5}, 32))})
	crypt, err := accounts.NewCrypt("fixture", string(key))
	if err != nil {
		t.Fatal(err)
	}
	return tx, crypt
}
func TestPrivateEntryOwnershipEncryptionAndCascade(t *testing.T) {
	db, crypt := fixture(t)
	s := New(db, crypt)
	a := accounts.NewService(db, crypt)
	const owner, other = uint(740000001), uint(740000002)
	input := Input{Source: "private", Kind: "tool", Name: "Private fixture", URL: "https://private.example.test", GroupName: "Work", Note: "private-entry-note"}
	entry, err := s.Save(owner, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	var stored models.PersonalEntry
	if err = db.First(&stored, entry.ID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.NoteCiphertext, input.Note) {
		t.Fatal("entry note persisted as plaintext")
	}
	if _, err := s.Get(other, entry.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user entry read allowed")
	}
	if list, _, err := s.List(other, "", "", "", false, 1, 24); err != nil || len(list) != 0 {
		t.Fatal("cross-user entry list leaked")
	}
	input.Revision = entry.Revision
	if _, err := s.Save(other, entry.ID, input); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user edit allowed")
	}
	if err := s.Delete(other, entry.ID, entry.Revision); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user delete allowed")
	}
	create := func(name string) accounts.Account {
		t.Helper()
		result, err := a.Save(owner, 0, accounts.Input{EntryID: entry.ID, Name: name, Environment: "other", Username: "fixture", ResourceURL: entry.URL, PasswordAction: "replace", Password: "private-fixture-password"})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	active := create("active fixture")
	preDeleted := create("already deleted fixture")
	if err := a.Delete(owner, preDeleted.ID, preDeleted.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Save(other, 0, accounts.Input{EntryID: entry.ID, Name: "forged", Environment: "other", Username: "fixture", PasswordAction: "keep"}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("account attached to someone else's private entry")
	}
	input.URL = "https://new-private.example.test"
	input.Pinned = true
	entry, err = s.Save(owner, entry.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	detail, _, err := a.Detail(owner, active.ID)
	if err != nil || detail.ResourceURL != input.URL {
		t.Fatal("entry URL change not reflected in account")
	}
	if password, err := a.Password(owner, active.ID); err != nil || password != "private-fixture-password" {
		t.Fatal("entry edit broke account encryption")
	}
	if _, err := s.Save(owner, entry.ID, input); !errors.Is(err, ErrConflict) {
		t.Fatal("stale entry edit allowed")
	}
	if err := s.Delete(owner, entry.ID, entry.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Password(owner, active.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cascade-deleted password still readable")
	}
	if err := a.Restore(owner, active.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("account restored under deleted parent")
	}
	if err := s.Restore(other, entry.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user restore allowed")
	}
	if err := s.Restore(owner, entry.ID); err != nil {
		t.Fatal(err)
	}
	if password, err := a.Password(owner, active.ID); err != nil || password != "private-fixture-password" {
		t.Fatal("cascade restore lost account password")
	}
	if _, err := a.Password(owner, preDeleted.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("parent restore revived independently deleted account")
	}
	list, total, err := a.List(owner, 0, "", "", false, 1, 12, 0, entry.ID)
	if err != nil || total != 1 || len(list) != 1 || list[0].EntryName != entry.Name {
		t.Fatal("private entry account scope failed", err)
	}
	fresh, err := s.Get(owner, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(owner, entry.ID, fresh.Revision); err != nil {
		t.Fatal(err)
	}
	if err = db.Unscoped().Model(&models.PersonalEntry{}).Where("id=?", entry.ID).Update("deleted_at", time.Now().Add(-8*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err = s.Restore(owner, entry.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("expired entry restored")
	}
}
func TestFavoriteImportAndLiveReference(t *testing.T) {
	db, crypt := fixture(t)
	s := New(db, crypt)
	const owner = uint(740000003)
	p := models.Project{Name: "Favorite project fixture", Type: "backend"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	imports := []Input{{Source: "project", TargetID: p.ID, Name: "old name"}, {Source: "tool", TargetID: 4294000000, Name: "deleted tool"}}
	for i := 0; i < 2; i++ {
		if err := s.Import(owner, imports); err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err := s.List(owner, "", "public", "", false, 1, 24)
	if err != nil || total != 2 {
		t.Fatal("import duplicated or lost favorites", err)
	}
	var ref Item
	for _, r := range rows {
		if r.Source == "project" {
			ref = r
		} else if r.Available || r.URL != "" {
			t.Fatal("missing tool gained arbitrary navigation")
		}
	}
	if ref.Name != p.Name {
		t.Fatal("favorite name was not resolved from public target")
	}
	if err := db.Model(&p).Update("name", "Updated public name").Error; err != nil {
		t.Fatal(err)
	}
	updated, err := s.Get(owner, ref.ID)
	if err != nil || updated.Name != "Updated public name" {
		t.Fatal("favorite name did not follow public update")
	}
	in := Input{Source: ref.Source, TargetID: ref.TargetID, Kind: ref.Kind, Name: ref.Name, URL: ref.URL, Note: "my personal reference note", Pinned: true, GroupName: "Pinned", Revision: ref.Revision}
	if _, err := s.Save(owner, ref.ID, in); err != nil {
		t.Fatal(err)
	}
	if err := s.Import(owner, imports); err != nil {
		t.Fatal(err)
	}
	updated, err = s.Get(owner, ref.ID)
	if err != nil || !updated.Pinned || updated.Note != in.Note || updated.GroupName != "Pinned" {
		t.Fatal("reimport overwrote personal metadata")
	}
	if err := s.Delete(owner, ref.ID, updated.Revision); err != nil {
		t.Fatal(err)
	}
	if err := s.Import(owner, imports[:1]); err != nil {
		t.Fatal(err)
	}
	if _, total, err := s.List(owner, "", "public", "", false, 1, 24); err != nil || total != 2 {
		t.Fatal("refavoriting failed")
	}
}
func TestEntryURLValidation(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "//example.com", "https://user:secret@example.com", "file:///tmp/example", "https://"} {
		if validURL(raw) {
			t.Fatal("unsafe URL accepted")
		}
	}
	if !validURL("https://example.com/path?q=test") {
		t.Fatal("ordinary URL rejected")
	}
}

func TestEntryHTTPOwnership(t *testing.T) {
	db, crypt := fixture(t)
	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		n, _ := strconv.Atoi(c.GetHeader("Test-User"))
		c.Set("userID", uint(n))
		c.Next()
	})
	Register(api, db, crypt)
	send := func(method, path, body string, user int) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, "/api/v1/my/entries"+path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Test-User", strconv.Itoa(user))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	body := `{"source":"private","kind":"tool","name":"HTTP fixture","url":"https://http.example.test","note":"private HTTP note"}`
	if w := send("POST", "", body, 0); w.Code != 401 {
		t.Fatal("anonymous entry creation accepted")
	}
	w := send("POST", "", body, 750000001)
	if w.Code != 200 {
		t.Fatal("HTTP entry create failed")
	}
	var result struct {
		Data Item `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	path := "/" + strconv.Itoa(int(result.Data.ID))
	if result.Data.ID == 0 || result.Data.Note != "private HTTP note" {
		t.Fatal("owner response lost private note")
	}
	if w := send("GET", path, "", 750000002); w.Code != 404 {
		t.Fatal("cross-user HTTP note read allowed")
	}
	if w := send("DELETE", path, `{"revision":2}`, 750000002); w.Code != 404 {
		t.Fatal("cross-user HTTP entry delete allowed")
	}
	if w := send("GET", path, "", 750000001); w.Code != 200 || w.Header().Get("Cache-Control") != "no-store, private" || strings.Contains(w.Body.String(), "ciphertext") {
		t.Fatal("owner entry response invalid or cacheable")
	}
	if w := send("POST", "/import", `{"items":[{"source":"private","kind":"tool","name":"bad","targetId":1}]}`, 750000001); w.Code != 400 {
		t.Fatal("import used to forge private entry")
	}
}
