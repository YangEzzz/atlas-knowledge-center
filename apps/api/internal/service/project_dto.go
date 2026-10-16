package service

import (
	"strings"
	"time"

	"ty-notebook-api/internal/models"
)

type ProjectOwnerDTO struct {
	ID        uint   `json:"id"`
	UserID    *uint  `json:"userId"`
	OwnerName string `json:"ownerName"`
	RoleType  string `json:"roleType"`
}

type ProjectOwnersDTO struct {
	Frontend []ProjectOwnerDTO `json:"frontend"`
	Backend  []ProjectOwnerDTO `json:"backend"`
	Tester   []ProjectOwnerDTO `json:"tester"`
}

type ProjectEntryDTO struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Remark    string `json:"remark"`
	SortOrder int    `json:"sortOrder"`
}

type ProjectDTO struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	TypeLabel   string            `json:"typeLabel"`
	Description string            `json:"description"`
	Keywords    []string          `json:"keywords"`
	Owners      ProjectOwnersDTO  `json:"owners"`
	Entries     []ProjectEntryDTO `json:"entries"`
	SortOrder   int               `json:"sortOrder"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func mapProjects(projects []models.Project) []ProjectDTO {
	result := make([]ProjectDTO, 0, len(projects))
	for _, project := range projects {
		result = append(result, mapProject(project))
	}
	return result
}

func mapProject(project models.Project) ProjectDTO {
	dto := ProjectDTO{
		ID:          project.ID,
		Name:        project.Name,
		Type:        project.Type,
		TypeLabel:   projectTypeLabel(project.Type),
		Description: project.Description,
		Keywords:    splitKeywords(project.Keywords),
		Owners: ProjectOwnersDTO{
			Frontend: []ProjectOwnerDTO{},
			Backend:  []ProjectOwnerDTO{},
			Tester:   []ProjectOwnerDTO{},
		},
		Entries:   []ProjectEntryDTO{},
		SortOrder: project.SortOrder,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}

	for _, owner := range project.Owners {
		item := ProjectOwnerDTO{
			ID:        owner.ID,
			UserID:    owner.UserID,
			OwnerName: owner.OwnerName,
			RoleType:  owner.RoleType,
		}
		switch owner.RoleType {
		case ProjectOwnerRoleFrontend:
			dto.Owners.Frontend = append(dto.Owners.Frontend, item)
		case ProjectOwnerRoleBackend:
			dto.Owners.Backend = append(dto.Owners.Backend, item)
		case ProjectOwnerRoleTester:
			dto.Owners.Tester = append(dto.Owners.Tester, item)
		}
	}

	for _, entry := range project.Entries {
		dto.Entries = append(dto.Entries, ProjectEntryDTO{
			ID:        entry.ID,
			Type:      entry.Type,
			Title:     entry.Title,
			URL:       entry.URL,
			Remark:    entry.Remark,
			SortOrder: entry.SortOrder,
		})
	}

	return dto
}

func splitKeywords(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func projectTypeLabel(value string) string {
	labels := map[string]string{
		"frontend":     "前端应用",
		"backend":      "后端服务",
		"admin":        "后台系统",
		"user_client":  "用户端",
		"website":      "官网",
		"mobile":       "移动端",
		"mini_program": "小程序",
		"shared_lib":   "公共库/组件",
		"other":        "其他",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}
