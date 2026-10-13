package service

import (
	"errors"
	"strings"
	"time"

	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/repository"
)

type PublicLinkService struct {
	repo *repository.PublicLinkRepository
}

type PublicLinkDTO struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	TypeLabel   string    `json:"typeLabel"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PublicLinkSaveInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
	SortOrder   int    `json:"sortOrder"`
}

func NewPublicLinkService(repo *repository.PublicLinkRepository) *PublicLinkService {
	return &PublicLinkService{repo: repo}
}

func (s *PublicLinkService) List(filter repository.PublicLinkListFilter) ([]PublicLinkDTO, int64, error) {
	links, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapPublicLinks(links), total, nil
}

func (s *PublicLinkService) Get(id uint) (*PublicLinkDTO, error) {
	link, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := mapPublicLink(*link)
	return &dto, nil
}

func (s *PublicLinkService) Create(input PublicLinkSaveInput, actorID *uint) (*PublicLinkDTO, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Type) == "" || strings.TrimSpace(input.URL) == "" {
		return nil, errors.New("名称、类型和 URL 不能为空")
	}

	link := &models.PublicLink{
		Name:        strings.TrimSpace(input.Name),
		Type:        strings.TrimSpace(input.Type),
		URL:         strings.TrimSpace(input.URL),
		Description: strings.TrimSpace(input.Description),
		IsEnabled:   input.IsEnabled,
		SortOrder:   input.SortOrder,
		CreatedBy:   actorID,
		UpdatedBy:   actorID,
	}
	if err := s.repo.Create(link); err != nil {
		return nil, err
	}
	return s.Get(link.ID)
}

func (s *PublicLinkService) Update(id uint, input PublicLinkSaveInput, actorID *uint) (*PublicLinkDTO, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Type) == "" || strings.TrimSpace(input.URL) == "" {
		return nil, errors.New("名称、类型和 URL 不能为空")
	}

	link, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	link.Name = strings.TrimSpace(input.Name)
	link.Type = strings.TrimSpace(input.Type)
	link.URL = strings.TrimSpace(input.URL)
	link.Description = strings.TrimSpace(input.Description)
	link.IsEnabled = input.IsEnabled
	link.SortOrder = input.SortOrder
	link.UpdatedBy = actorID

	if err := s.repo.Update(link); err != nil {
		return nil, err
	}
	return s.Get(link.ID)
}

func (s *PublicLinkService) Delete(id uint, actorID *uint) error {
	return s.repo.Delete(id, actorID)
}

func (s *PublicLinkService) Restore(id uint) error {
	return s.repo.Restore(id)
}

func mapPublicLinks(links []models.PublicLink) []PublicLinkDTO {
	result := make([]PublicLinkDTO, 0, len(links))
	for _, link := range links {
		result = append(result, mapPublicLink(link))
	}
	return result
}

func mapPublicLink(link models.PublicLink) PublicLinkDTO {
	return PublicLinkDTO{
		ID:          link.ID,
		Name:        link.Name,
		Type:        link.Type,
		TypeLabel:   publicLinkTypeLabel(link.Type),
		URL:         link.URL,
		Description: link.Description,
		IsEnabled:   link.IsEnabled,
		SortOrder:   link.SortOrder,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}
}

func publicLinkTypeLabel(value string) string {
	labels := map[string]string{
		"code":     "代码",
		"deploy":   "发布",
		"monitor":  "监控",
		"logs":     "日志",
		"api":      "接口",
		"design":   "设计",
		"require":  "需求",
		"test":     "测试",
		"ops":      "运维",
		"access":   "权限",
		"document": "文档",
		"other":    "其他",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}
