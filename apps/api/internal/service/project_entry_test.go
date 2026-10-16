package service

import (
	"testing"
	"ty-notebook-api/internal/models"
)

func TestProjectEntryRemarkMapping(t *testing.T) {
	entries := buildEntries(42, []ProjectEntryInput{
		{Type: "production", URL: "https://example.com", Remark: "  仅内网访问\n请先连接 VPN  "},
		{Type: "gitlab", URL: "https://git.example.com"},
	}, nil)
	if len(entries) != 2 || entries[0].Remark != "仅内网访问\n请先连接 VPN" || entries[1].Remark != "" {
		t.Fatalf("unexpected optional remark mapping: %#v", entries)
	}
	dto := mapProject(models.Project{Entries: entries})
	if dto.Entries[0].Remark != entries[0].Remark || dto.Entries[1].Remark != "" {
		t.Fatal("remark was lost from the project response")
	}
}
