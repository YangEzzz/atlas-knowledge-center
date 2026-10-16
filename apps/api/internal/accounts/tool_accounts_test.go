package accounts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
	"ty-notebook-api/internal/models"
)

func TestTargetEncryptionCompatibility(t *testing.T) {
	c := testCrypt(t)
	legacy := "atlas-account:v1:7:11:15"
	a := Account{ID: 15, UserID: 7, ProjectID: 11}
	ciphertext := c.Seal(`{"password":"legacy"}`, legacy)
	if _, err := c.Open(ciphertext, aad(a)); err != nil {
		t.Fatal("existing project ciphertext became unreadable")
	}
	a.ProjectID, a.ToolID = 0, 11
	if _, err := c.Open(ciphertext, aad(a)); err == nil {
		t.Fatal("project ciphertext accepted as tool ciphertext")
	}
	ciphertext = c.Seal("tool secret", aad(a))
	a.ToolID++
	if _, err := c.Open(ciphertext, aad(a)); err == nil {
		t.Fatal("tool ciphertext accepted under another tool")
	}
}

func TestAccountRequiresExactlyOneTarget(t *testing.T) {
	s := &Service{}
	for _, input := range []Input{{}, {ProjectID: 1, ToolID: 2}} {
		if _, err := s.Save(7, 0, input); !errors.Is(err, ErrInvalid) {
			t.Fatal("ambiguous account target accepted")
		}
	}
}

func TestToolAccountPrivacyAndLifecycle(t *testing.T) {
	db := integrationDB(t)
	s := NewService(db, testCrypt(t))
	tool := models.PublicLink{Name: "Private account tool fixture", Type: "gitlab", URL: "https://tool.example.test", IsEnabled: true}
	other := models.PublicLink{Name: "Other tool fixture", Type: "monitor", URL: "https://other.example.test", IsEnabled: true}
	for _, record := range []any{&tool, &other} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	const owner, stranger = uint(720000001), uint(720000002)
	input := Input{ToolID: tool.ID, Name: "Tool account fixture", Username: "fixture-user", Environment: "production", ResourceURL: tool.URL, PasswordAction: "replace", Password: "private-tool-password", Note: "private-tool-note"}
	a, err := s.Save(owner, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	items, total, err := s.List(owner, 0, "", "", false, 1, 12, tool.ID)
	if err != nil || total != 1 || len(items) != 1 || items[0].ToolName != tool.Name || items[0].ProjectID != 0 {
		t.Fatal("tool-filtered list failed", err)
	}
	serialized, _ := json.Marshal(items)
	if strings.Contains(string(serialized), input.Password) || strings.Contains(string(serialized), input.Note) || strings.Contains(string(serialized), "ciphertext") {
		t.Fatal("private fields leaked in tool list")
	}
	if list, _, err := s.List(owner, 0, tool.Name, "", false, 1, 12); err != nil || len(list) != 1 {
		t.Fatal("tool name search failed", err)
	}
	if list, _, _ := s.List(stranger, 0, "", "", false, 1, 12, tool.ID); len(list) != 0 {
		t.Fatal("another user listed tool accounts")
	}
	if _, _, err := s.Detail(stranger, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("another user read tool note")
	}
	if _, err := s.Password(stranger, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("another user read tool password")
	}
	input.Revision = a.Revision
	if _, err := s.Save(stranger, a.ID, input); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("another user edited tool account")
	}
	if err := s.Delete(stranger, a.ID, a.Revision); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("another user deleted tool account")
	}
	input.PasswordAction, input.Password = "keep", ""
	a, err = s.Save(owner, a.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if password, err := s.Password(owner, a.ID); err != nil || password != "private-tool-password" {
		t.Fatal("tool password keep failed")
	}
	input.Revision, input.ToolID = a.Revision, other.ID
	if _, err := s.Save(owner, a.ID, input); !errors.Is(err, ErrInvalid) {
		t.Fatal("tool account reassignment allowed")
	}
	input.ToolID = tool.ID
	input.ResourceURL = other.URL
	if _, err := s.Save(owner, a.ID, input); !errors.Is(err, ErrInvalid) {
		t.Fatal("another tool's URL accepted")
	}
	if err := s.Delete(owner, a.ID, a.Revision); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(stranger, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("another user restored tool account")
	}
	if err := s.Restore(owner, a.ID); err != nil {
		t.Fatal(err)
	}
	if password, err := s.Password(owner, a.ID); err != nil || password != "private-tool-password" {
		t.Fatal("tool password restore failed")
	}
	if err := db.Delete(&tool).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Password(owner, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("deleted tool password remains accessible")
	}
	if items, _, _ := s.List(owner, 0, "", "", false, 1, 12, tool.ID); len(items) != 0 {
		t.Fatal("deleted tool account remains active")
	}
	var stored Account
	if err := db.First(&stored, a.ID).Error; err != nil {
		t.Fatal("tool deletion removed private account", err)
	}
	if err := s.Delete(owner, a.ID, stored.Revision); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(owner, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(fmt.Sprint("account restored while its tool is deleted: ", err))
	}
}
