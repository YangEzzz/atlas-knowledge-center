package accounts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/models"
)

func testCrypt(t *testing.T) *Crypt {
	t.Helper()
	raw, _ := json.Marshal(map[string]string{"v1": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))})
	c, err := NewCrypt("v1", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHTTPAccountPrivacyAndValidation(t *testing.T) {
	db := integrationDB(t)
	p := models.Project{Name: "HTTP account test", Type: "backend"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		id, _ := strconv.Atoi(c.GetHeader("Test-User"))
		c.Set("userID", uint(id))
		c.Next()
	})
	Register(api, db, testCrypt(t), "session")
	req := func(method, path, body string, user int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/api/v1/my/accounts"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Test-User", strconv.Itoa(user))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if w := req("GET", "", "", 0); w.Code != 401 {
		t.Fatal("anonymous account access allowed")
	}
	body := fmt.Sprintf(`{"projectId":%d,"name":"HTTP test","environment":"staging","username":"test","passwordAction":"replace","password":"fixture-password","note":"fixture-note"}`, p.ID)
	w := req("POST", "", body, 91)
	if w.Code != 200 {
		t.Fatal("HTTP create failed")
	}
	var saved struct {
		Data Account `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	path := "/" + strconv.Itoa(int(saved.Data.ID))
	// A purpose label is optional; neither omitted nor empty names require a made-up title.
	for _, withoutName := range []string{strings.Replace(body, `,"name":"HTTP test"`, "", 1), strings.Replace(body, `"name":"HTTP test"`, `"name":""`, 1)} {
		if w := req("POST", "", withoutName, 91); w.Code != 200 {
			t.Fatal("account without purpose label rejected")
		}
	}
	w = req("GET", "", "", 91)
	if w.Code != 200 || strings.Contains(w.Body.String(), "fixture-password") || strings.Contains(w.Body.String(), "fixture-note") || strings.Contains(w.Body.String(), "ciphertext") {
		t.Fatal("HTTP list leaked private content")
	}
	if w.Header().Get("Cache-Control") != "no-store, private" {
		t.Fatal("private response can be cached")
	}
	for _, path := range []string{path, path + "/password"} {
		method := http.MethodGet
		if strings.HasSuffix(path, "password") {
			method = http.MethodPost
		}
		if w := req(method, path, "{}", 92); w.Code != 404 {
			t.Fatal("cross-user HTTP read allowed")
		}
	}
	if w := req("POST", path+"/password", "{}", 91); w.Code != 200 || !strings.Contains(w.Body.String(), "fixture-password") {
		t.Fatal("owner cannot read password")
	}
	if w := req("PUT", path, body, 91); w.Code != 409 {
		t.Fatal("missing edit revision accepted")
	}
	if w := req("POST", "", `{"name":"invalid"}`, 91); w.Code != 400 {
		t.Fatal("invalid input accepted")
	}
	var dbRow Account
	if err := db.First(&dbRow, saved.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dbRow.Ciphertext, "fixture-") {
		t.Fatal("plaintext saved")
	}
	tool := models.PublicLink{Name: "HTTP private tool fixture", Type: "gitlab", URL: "https://private-tool.example.test", IsEnabled: true}
	if err := db.Create(&tool).Error; err != nil {
		t.Fatal(err)
	}
	toolBody := fmt.Sprintf(`{"toolId":%d,"name":"HTTP tool account","environment":"production","username":"tool-user","passwordAction":"replace","password":"tool-password-fixture","note":"tool-note-fixture","resourceUrl":%q}`, tool.ID, tool.URL)
	w = req("POST", "", toolBody, 91)
	if w.Code != 200 {
		t.Fatal("HTTP tool create failed")
	}
	var toolSaved struct {
		Data Account `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &toolSaved); err != nil {
		t.Fatal(err)
	}
	if toolSaved.Data.ProjectID != 0 || toolSaved.Data.ToolID != tool.ID {
		t.Fatal("HTTP tool target not preserved")
	}
	w = req("GET", fmt.Sprintf("?toolId=%d", tool.ID), "", 91)
	if w.Code != 200 || !strings.Contains(w.Body.String(), tool.Name) || strings.Contains(w.Body.String(), "HTTP test") || strings.Contains(w.Body.String(), "tool-password-fixture") || strings.Contains(w.Body.String(), "tool-note-fixture") {
		t.Fatal("HTTP scoped tool list failed or leaked private content")
	}
	if w := req("GET", fmt.Sprintf("?toolId=%d&projectId=%d", tool.ID, p.ID), "", 91); w.Code != 400 {
		t.Fatal("ambiguous list scope allowed")
	}
	if w := req("POST", fmt.Sprintf("/%d/password", toolSaved.Data.ID), "{}", 92); w.Code != 404 {
		t.Fatal("cross-user tool password read allowed")
	}
}
func TestAuthenticatedEncryption(t *testing.T) {
	c := testCrypt(t)
	one := c.Seal(" password\n ", "owner:record1")
	two := c.Seal(" password\n ", "owner:record1")
	if one == two {
		t.Fatal("encryption reused nonce")
	}
	if value, err := c.Open(one, "owner:record1"); err != nil || value != " password\n " {
		t.Fatal("roundtrip changed password")
	}
	if _, err := c.Open(one, "other:record1"); err == nil {
		t.Fatal("wrong owner context accepted")
	}
	changed := []byte(one)
	changed[len(changed)-3] ^= 1
	if _, err := c.Open(string(changed), "owner:record1"); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err := NewCrypt("v1", `{"v1":"short"}`); err == nil {
		t.Fatal("weak key accepted")
	}
	raw, _ := json.Marshal(map[string]string{"v1": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)), "v2": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32))})
	rotated, err := NewCrypt("v2", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = rotated.Open(one, "owner:record1"); err != nil {
		t.Fatal("old key lost after rotation")
	}
}
func integrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("requires dedicated atlas_db; transaction rollback")
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
	sql, _ := db.DB()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback(); sql.Close() })
	return tx
}
func TestPrivateAccountLifecycle(t *testing.T) {
	db := integrationDB(t)
	s := NewService(db, testCrypt(t))
	suffix := fmt.Sprint(time.Now().UnixNano())
	owner := models.User{Username: "account-owner-" + suffix, ExternalUserID: "owner-" + suffix, Status: "active"}
	admin := models.User{Username: "account-other-" + suffix, ExternalUserID: "other-" + suffix, RoleCode: "admin", Status: "active"}
	p := models.Project{Name: "Account lifecycle", Type: "backend"}
	for _, v := range []any{&owner, &admin, &p} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	input := Input{ProjectID: p.ID, Name: "测试账号", Username: "test-user", Environment: "staging", PasswordAction: "replace", Password: "  private-test-password  ", Note: "private-test-note"}
	a, err := s.Save(owner.ID, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(a.Ciphertext), []byte(input.Password)) || bytes.Contains([]byte(a.Ciphertext), []byte(input.Note)) {
		t.Fatal("plaintext persisted")
	}
	items, _, err := s.List(owner.ID, p.ID, "", "", false, 1, 12)
	if err != nil || len(items) != 1 {
		t.Fatal("owner list failed")
	}
	encoded, _ := json.Marshal(items)
	for _, bad := range []string{"ciphertext", "private-test-password", "private-test-note", "\"password\"", "\"note\""} {
		if bytes.Contains(bytes.ToLower(encoded), []byte(bad)) {
			t.Fatal("secret leaked in list")
		}
	}
	if list, _, _ := s.List(admin.ID, p.ID, "", "", false, 1, 12); len(list) != 0 {
		t.Fatal("admin listed another user's records")
	}
	if _, _, err = s.Detail(admin.ID, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user detail allowed")
	}
	if _, err = s.Password(admin.ID, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user password allowed")
	}
	input.Revision = a.Revision
	if _, err = s.Save(admin.ID, a.ID, input); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user update allowed")
	}
	if err = s.Delete(admin.ID, a.ID, a.Revision); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user delete allowed")
	}
	password, err := s.Password(owner.ID, a.ID)
	if err != nil || password != input.Password {
		t.Fatal("password changed during encryption")
	}
	input.PasswordAction = "keep"
	input.Password = "ignored"
	input.Note = "updated-note"
	oldRevision := a.Revision
	a, err = s.Save(owner.ID, a.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := s.Password(owner.ID, a.ID); value != password {
		t.Fatal("keep action changed password")
	}
	if _, err = s.Save(owner.ID, a.ID, input); !errors.Is(err, ErrConflict) {
		t.Fatal("stale edit accepted")
	}
	if err = s.Delete(owner.ID, a.ID, oldRevision); !errors.Is(err, ErrConflict) {
		t.Fatal("stale delete accepted")
	}
	input.Revision = a.Revision
	input.PasswordAction = "clear"
	a, err = s.Save(owner.ID, a.ID, input)
	if err != nil || a.HasPassword {
		t.Fatal("clear failed")
	}
	if value, _ := s.Password(owner.ID, a.ID); value != "" {
		t.Fatal("cleared password remains")
	}
	if err = s.Delete(owner.ID, a.ID, a.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Password(owner.ID, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("deleted password accessible")
	}
	if err = s.Restore(admin.ID, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user restore allowed")
	}
	if err = s.Restore(owner.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if err = db.Model(&Account{}).Where("id = ?", a.ID).Update("deleted_at", time.Now().Add(-8*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err = s.Restore(owner.ID, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("expired record restored")
	}
	if list, _, _ := s.List(owner.ID, 0, "", "", true, 1, 12); len(list) != 0 {
		t.Fatal("expired record listed")
	}
	if err = PurgeExpired(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Unscoped().Model(&Account{}).Where("id = ?", a.ID).Count(&count)
	if count != 0 {
		t.Fatal("expired record not purged")
	}
}
