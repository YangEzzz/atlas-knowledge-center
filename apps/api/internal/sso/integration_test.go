package sso

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/models"
)

// All Atlas test data is isolated in a transaction and rolled back. Helios is
// simulated; no actual user credentials, sessions, or memberships are changed.
func TestSSOSessionLifecycle(t *testing.T) {
	if os.Getenv("ATLAS_AUTH_INTEGRATION") != "true" {
		t.Skip("set ATLAS_AUTH_INTEGRATION=true to use atlas_db with rollback")
	}
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if env["DB_NAME"] != "atlas_db" {
		t.Fatal("integration test requires dedicated atlas_db")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(env["DB_USER"], env["DB_PASSWORD"]), Host: env["DB_HOST"] + ":" + env["DB_PORT"], Path: "/atlas_db"}
	q := dsn.Query()
	q.Set("sslmode", env["DB_SSLMODE"])
	dsn.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("unable to open Atlas integration database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	denied := false
	helios := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, secret, _ := r.BasicAuth()
		if id != "test-atlas" || secret != "test-secret" {
			t.Error("missing client authentication")
		}
		if denied {
			w.WriteHeader(403)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"user": map[string]any{"id": "atlas-test-rollback", "name": "Test Admin"}, "project_id_string": "atlas-knowledge-center", "role_in_project": "admin"}})
	}))
	defer helios.Close()
	cfg := config.Config{FrontendURL: "http://localhost:5176", HeliosBaseURL: "http://localhost:5174", HeliosInternalURL: helios.URL, HeliosClientID: "test-atlas", HeliosClientSecret: "test-secret", HeliosRedirectURI: "http://localhost:5176/api/v1/auth/sso/callback", HeliosProjectID: "atlas-knowledge-center", HeliosAllowedRoles: []string{"admin"}}
	s := New(tx, cfg)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s.Register(r.Group("/auth"))
	r.GET("/auth/me", s.AuthMiddleware(), s.Me)
	cookies := map[string]*http.Cookie{}
	send := func(method, path, body string, csrf bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		if csrf {
			req.Header.Set("Origin", cfg.FrontendURL)
			req.Header.Set("X-CSRF-Token", cookies[CSRFCookie].Value)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		for _, c := range w.Result().Cookies() {
			if c.MaxAge < 0 {
				delete(cookies, c.Name)
			} else {
				cookies[c.Name] = c
			}
		}
		return w
	}
	if w := send("GET", "/auth/me", "", false); w.Code != 401 {
		t.Fatal("unauthenticated access allowed")
	}
	if w := send("GET", "/auth/csrf", "", false); w.Code != 200 {
		t.Fatal("csrf initialization failed")
	}
	if w := send("POST", "/auth/sso/start", `{"mode":"redirect","return_path":"/projects"}`, false); w.Code != 403 {
		t.Fatal("start accepted without CSRF")
	}
	w := send("POST", "/auth/sso/start", `{"mode":"redirect","return_path":"/projects"}`, true)
	var start struct {
		Data struct {
			State string `json:"transaction_id"`
			URL   string `json:"authorize_url"`
		} `json:"data"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &start); err != nil || w.Code != 200 || start.Data.State == "" {
		t.Fatal("start failed")
	}
	target, _ := url.Parse(start.Data.URL)
	if len(target.Query().Get("code_challenge")) != 43 || target.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("PKCE missing")
	}
	browser := cookies[BrowserCookie]
	delete(cookies, BrowserCookie)
	send("GET", "/auth/sso/callback?state="+start.Data.State+"&code=test", "", false)
	if cookies[SessionCookie] != nil {
		t.Fatal("callback accepted another browser")
	}
	cookies[BrowserCookie] = browser
	w = send("GET", "/auth/sso/callback?state="+start.Data.State+"&code=test", "", false)
	if w.Code != 303 || cookies[SessionCookie] == nil || !cookies[SessionCookie].HttpOnly {
		t.Fatal("session not established")
	}
	if cookies[SessionCookie].MaxAge != 30*24*60*60 {
		t.Fatal("session cookie must last 30 days")
	}
	var created Session
	if err := tx.First(&created, "id = ?", hash(cookies[SessionCookie].Value)).Error; err != nil {
		t.Fatal(err)
	}
	if remaining := time.Until(created.ExpiresAt); remaining < 30*24*time.Hour-5*time.Second || remaining > 30*24*time.Hour {
		t.Fatalf("unexpected server-side session lifetime: %v", remaining)
	}
	var count int64
	tx.Model(&Session{}).Count(&count)
	send("GET", "/auth/sso/callback?state="+start.Data.State+"&code=test", "", false)
	var after int64
	tx.Model(&Session{}).Count(&after)
	if count != after {
		t.Fatal("callback replay created session")
	}
	if w = send("GET", "/auth/me", "", false); w.Code != 200 || !strings.Contains(w.Body.String(), "Test Admin") {
		t.Fatal("profile lookup failed")
	}
	var user models.User
	tx.First(&user, "external_user_id = ?", "atlas-test-rollback")
	if user.RoleCode != "admin" {
		t.Fatal("identity mapping failed")
	}
	token := cookies[SessionCookie]
	tx.Model(&Session{}).Where("id = ?", hash(token.Value)).Update("verified_at", time.Now().Add(-6*time.Minute))
	denied = true
	if w = send("GET", "/auth/me", "", false); w.Code != 401 {
		t.Fatal("revoked membership still accepted")
	}
	if cookies[SessionCookie] != nil {
		t.Fatal("revoked cookie not cleared")
	}
	denied = false
	tx.Create(&Session{ID: hash("expired"), UserID: user.ID, ExternalID: user.ExternalUserID, VerifiedAt: time.Now(), ExpiresAt: time.Now().Add(-time.Hour)})
	cookies[SessionCookie] = &http.Cookie{Name: SessionCookie, Value: "expired"}
	if w = send("GET", "/auth/me", "", false); w.Code != 401 {
		t.Fatal("expired session accepted")
	}
	cookies[SessionCookie] = token
	if w = send("POST", "/auth/logout", `{}`, false); w.Code != 403 {
		t.Fatal("logout accepted without CSRF")
	}
	if w = send("POST", "/auth/logout", `{}`, true); w.Code != 200 {
		t.Fatal("logout failed")
	}
	fmt.Println("SSO lifecycle verified with rollback: PKCE, browser binding, profile, replay, revocation, expiry, logout")
}
