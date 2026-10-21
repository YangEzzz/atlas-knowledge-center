package service

import (
	"strings"

	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/repository"
)

const (
	ProjectOwnerRoleFrontend = "frontend"
	ProjectOwnerRoleBackend  = "backend"
	ProjectOwnerRoleTester   = "tester"

	ProjectEntryTypeProduction = "production"
	ProjectEntryTypeStaging    = "staging"
	ProjectEntryTypeGitLab     = "gitlab"
)

type ProjectService struct {
	repo *repository.ProjectRepository
}

type ProjectOwnerInput struct {
	UserID    *uint  `json:"userId"`
	OwnerName string `json:"ownerName"`
}

type ProjectEntryInput struct {
	ID     uint   `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Remark string `json:"remark"`
}

type ProjectSaveInput struct {
	Name           string              `json:"name"`
	Type           string              `json:"type"`
	Description    string              `json:"description"`
	Keywords       []string            `json:"keywords"`
	Entries        []ProjectEntryInput `json:"entries"`
	FrontendOwners []ProjectOwnerInput `json:"frontendOwners"`
	BackendOwners  []ProjectOwnerInput `json:"backendOwners"`
	TesterOwners   []ProjectOwnerInput `json:"testerOwners"`
	SortOrder      int                 `json:"sortOrder"`
}

func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) List(filter repository.ProjectListFilter) ([]ProjectDTO, int64, error) {
	projects, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapProjects(projects), total, nil
}

func (s *ProjectService) Get(id uint) (*ProjectDTO, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := mapProject(*project)
	return &dto, nil
}

func (s *ProjectService) Create(input ProjectSaveInput, actorID *uint) (*ProjectDTO, error) {
	project := &models.Project{
		Name:        strings.TrimSpace(input.Name),
		Type:        input.Type,
		Description: input.Description,
		Keywords:    strings.Join(input.Keywords, ","),
		SortOrder:   input.SortOrder,
		Source:      "manual",
		CreatedBy:   actorID,
		UpdatedBy:   actorID,
	}
	if err := s.repo.SaveProject(project, buildOwners(project.ID, input, actorID), buildEntries(project.ID, input.Entries, actorID), actorID); err != nil {
		return nil, err
	}
	return s.Get(project.ID)
}

func (s *ProjectService) Update(id uint, input ProjectSaveInput, actorID *uint) (*ProjectDTO, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	project.Name = strings.TrimSpace(input.Name)
	project.Type = input.Type
	project.Description = input.Description
	project.Keywords = strings.Join(input.Keywords, ",")
	project.SortOrder = input.SortOrder
	project.UpdatedBy = actorID

	if err := s.repo.SaveProject(project, buildOwners(project.ID, input, actorID), buildEntries(project.ID, input.Entries, actorID), actorID); err != nil {
		return nil, err
	}
	return s.Get(project.ID)
}

func (s *ProjectService) Delete(id uint, actorID *uint) error {
	return s.repo.Delete(id, actorID)
}

func (s *ProjectService) Restore(id uint, actor ...*uint) error {
	return s.repo.Restore(id, actor...)
}

func buildOwners(projectID uint, input ProjectSaveInput, actorID *uint) []models.ProjectOwner {
	var owners []models.ProjectOwner
	owners = appendOwners(owners, projectID, ProjectOwnerRoleFrontend, input.FrontendOwners, actorID)
	owners = appendOwners(owners, projectID, ProjectOwnerRoleBackend, input.BackendOwners, actorID)
	owners = appendOwners(owners, projectID, ProjectOwnerRoleTester, input.TesterOwners, actorID)
	return owners
}

func appendOwners(owners []models.ProjectOwner, projectID uint, roleType string, inputs []ProjectOwnerInput, actorID *uint) []models.ProjectOwner {
	for index, input := range inputs {
		name := strings.TrimSpace(input.OwnerName)
		if name == "" {
			continue
		}
		owners = append(owners, models.ProjectOwner{
			ProjectID: projectID,
			UserID:    input.UserID,
			OwnerName: name,
			RoleType:  roleType,
			SortOrder: index,
			CreatedBy: actorID,
		})
	}
	return owners
}

func buildEntries(projectID uint, inputs []ProjectEntryInput, actorID *uint) []models.ProjectEntry {
	entries := make([]models.ProjectEntry, 0, len(inputs))
	for index, input := range inputs {
		entryType := strings.TrimSpace(input.Type)
		url := strings.TrimSpace(input.URL)
		if entryType == "" || url == "" {
			continue
		}
		title := strings.TrimSpace(input.Title)
		if title == "" {
			title = projectEntryTitle(entryType)
		}
		entries = append(entries, models.ProjectEntry{
			ID:        input.ID,
			ProjectID: projectID,
			Type:      entryType,
			Title:     title,
			URL:       url,
			Remark:    strings.TrimSpace(input.Remark),
			SortOrder: index,
			CreatedBy: actorID,
			UpdatedBy: actorID,
		})
	}
	return entries
}

func projectEntryTitle(entryType string) string {
	switch entryType {
	case ProjectEntryTypeProduction:
		return "线上地址"
	case ProjectEntryTypeStaging:
		return "测试地址"
	case ProjectEntryTypeGitLab:
		return "GitLab"
	default:
		return "入口"
	}
}
