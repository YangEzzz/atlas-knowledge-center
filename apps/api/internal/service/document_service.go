package service

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ty-notebook-api/internal/models"
	"ty-notebook-api/internal/repository"

	"gorm.io/gorm"
)

const (
	DocumentStatusDraft     = "draft"
	DocumentStatusPublished = "published"
)

type DocumentService struct {
	repo *repository.DocumentRepository
}

type DocumentDTO struct {
	ID               uint       `json:"id"`
	ProjectID        *uint      `json:"projectId"`
	Type             string     `json:"type"`
	TypeLabel        string     `json:"typeLabel"`
	Title            string     `json:"title"`
	Summary          string     `json:"summary"`
	Content          string     `json:"content"`
	Status           string     `json:"status"`
	StatusLabel      string     `json:"statusLabel"`
	CurrentVersionNo int        `json:"currentVersionNo"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	DeletedAt        *time.Time `json:"deletedAt,omitempty"`
}

type DocumentVersionDTO struct {
	ID            uint      `json:"id"`
	DocumentID    uint      `json:"documentId"`
	VersionNo     int       `json:"versionNo"`
	Title         string    `json:"title"`
	Type          string    `json:"type"`
	TypeLabel     string    `json:"typeLabel"`
	Summary       string    `json:"summary"`
	Content       string    `json:"content"`
	ChangeSummary string    `json:"changeSummary"`
	CreatedAt     time.Time `json:"createdAt"`
}

type DocumentSaveInput struct {
	ID        uint   `json:"id"`
	ProjectID *uint  `json:"projectId"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
	Status    string `json:"status"`
}

type DocumentVersionRestoreInput struct {
	ChangeSummary string `json:"changeSummary"`
}

func NewDocumentService(repo *repository.DocumentRepository) *DocumentService {
	return &DocumentService{repo: repo}
}

func (s *DocumentService) List(filter repository.DocumentListFilter) ([]DocumentDTO, int64, error) {
	documents, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapDocuments(documents), total, nil
}

func (s *DocumentService) ListDeleted(filter repository.DocumentListFilter) ([]DocumentDTO, int64, error) {
	documents, total, err := s.repo.ListDeleted(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapDocuments(documents), total, nil
}

func (s *DocumentService) Get(id uint) (*DocumentDTO, error) {
	document, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := mapDocument(*document)
	return &dto, nil
}

func (s *DocumentService) GetByProjectAndType(projectID uint, docType string) (*DocumentDTO, error) {
	document, err := s.repo.FindByProjectAndType(projectID, strings.TrimSpace(docType))
	if err != nil {
		return nil, err
	}
	dto := mapDocument(*document)
	return &dto, nil
}

func (s *DocumentService) Save(input DocumentSaveInput, actorID *uint) (*DocumentDTO, error) {
	if strings.TrimSpace(input.Type) == "" || strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("type 和 title 不能为空")
	}
	if input.ProjectID != nil && *input.ProjectID == 0 {
		return nil, errors.New("关联项目无效，不关联项目请传 null")
	}
	docType := strings.TrimSpace(input.Type)
	status := normalizeDocumentStatus(input.Status)
	var document *models.ProjectDocument
	if input.ID > 0 {
		var err error
		document, err = s.repo.FindByID(input.ID)
		if err != nil {
			return nil, err
		}
	} else {
		document = &models.ProjectDocument{CreatedBy: actorID}
	}
	if input.ProjectID != nil {
		existing, err := s.repo.FindByProjectAndType(*input.ProjectID, docType)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if err == nil && existing.ID != document.ID {
			return nil, errors.New("该项目已有同类型文档，请从文档库打开后编辑")
		}
	}
	document.ProjectID = input.ProjectID
	document.Type = docType

	document.Title = strings.TrimSpace(input.Title)
	document.Summary = strings.TrimSpace(input.Summary)
	document.Content = input.Content
	document.ContentText = markdownToText(input.Content)
	document.TOCJSON = markdownTOCJSON(input.Content)
	document.Status = status
	document.UpdatedBy = actorID
	document.CurrentVersionNo = document.CurrentVersionNo + 1

	version := &models.DocumentVersion{
		VersionNo:     document.CurrentVersionNo,
		Title:         document.Title,
		Type:          document.Type,
		Summary:       document.Summary,
		Content:       document.Content,
		ContentText:   document.ContentText,
		TOCJSON:       document.TOCJSON,
		ChangeSummary: documentChangeSummary(status, document.CurrentVersionNo),
		CreatedBy:     actorID,
	}

	if err := s.repo.SaveWithVersion(document, version); err != nil {
		return nil, err
	}
	return s.Get(document.ID)
}

func (s *DocumentService) ListVersions(documentID uint) ([]DocumentVersionDTO, error) {
	if _, err := s.repo.FindByID(documentID); err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVersions(documentID)
	if err != nil {
		return nil, err
	}
	return mapDocumentVersions(versions), nil
}

func (s *DocumentService) GetVersion(documentID uint, versionID uint) (*DocumentVersionDTO, error) {
	version, err := s.repo.FindVersion(documentID, versionID)
	if err != nil {
		return nil, err
	}
	dto := mapDocumentVersion(*version)
	return &dto, nil
}

func (s *DocumentService) RestoreVersion(versionID uint, input DocumentVersionRestoreInput, actorID *uint) (*DocumentDTO, error) {
	version, err := s.repo.FindVersionByID(versionID)
	if err != nil {
		return nil, err
	}

	document, err := s.repo.FindByID(version.DocumentID)
	if err != nil {
		return nil, err
	}

	document.Title = version.Title
	document.Type = version.Type
	document.Summary = version.Summary
	document.Content = version.Content
	document.ContentText = version.ContentText
	document.TOCJSON = version.TOCJSON
	document.UpdatedBy = actorID
	document.CurrentVersionNo = document.CurrentVersionNo + 1

	changeSummary := strings.TrimSpace(input.ChangeSummary)
	if changeSummary == "" {
		changeSummary = "恢复到 v" + strconv.Itoa(version.VersionNo)
	}

	restoredVersion := &models.DocumentVersion{
		VersionNo:     document.CurrentVersionNo,
		Title:         document.Title,
		Type:          document.Type,
		Summary:       document.Summary,
		Content:       document.Content,
		ContentText:   document.ContentText,
		TOCJSON:       document.TOCJSON,
		ChangeSummary: changeSummary,
		CreatedBy:     actorID,
	}

	if err := s.repo.SaveWithVersion(document, restoredVersion, true); err != nil {
		return nil, err
	}
	return s.Get(document.ID)
}

func (s *DocumentService) Delete(id uint, actorID *uint) error {
	document, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(document, actorID)
}

func (s *DocumentService) Restore(id uint, actor ...*uint) (*DocumentDTO, error) {
	if _, err := s.repo.FindDeletedByID(id); err != nil {
		return nil, err
	}
	if err := s.repo.Restore(id, actor...); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func mapDocuments(documents []models.ProjectDocument) []DocumentDTO {
	result := make([]DocumentDTO, 0, len(documents))
	for _, document := range documents {
		result = append(result, mapDocument(document))
	}
	return result
}

func mapDocument(document models.ProjectDocument) DocumentDTO {
	var deletedAt *time.Time
	if document.DeletedAt.Valid {
		value := document.DeletedAt.Time
		deletedAt = &value
	}
	return DocumentDTO{
		ID:               document.ID,
		ProjectID:        document.ProjectID,
		Type:             document.Type,
		TypeLabel:        documentTypeLabel(document.Type),
		Title:            document.Title,
		Summary:          document.Summary,
		Content:          document.Content,
		Status:           document.Status,
		StatusLabel:      documentStatusLabel(document.Status),
		CurrentVersionNo: document.CurrentVersionNo,
		CreatedAt:        document.CreatedAt,
		UpdatedAt:        document.UpdatedAt,
		DeletedAt:        deletedAt,
	}
}

func mapDocumentVersions(versions []models.DocumentVersion) []DocumentVersionDTO {
	result := make([]DocumentVersionDTO, 0, len(versions))
	for _, version := range versions {
		result = append(result, mapDocumentVersion(version))
	}
	return result
}

func mapDocumentVersion(version models.DocumentVersion) DocumentVersionDTO {
	return DocumentVersionDTO{
		ID:            version.ID,
		DocumentID:    version.DocumentID,
		VersionNo:     version.VersionNo,
		Title:         version.Title,
		Type:          version.Type,
		TypeLabel:     documentTypeLabel(version.Type),
		Summary:       version.Summary,
		Content:       version.Content,
		ChangeSummary: version.ChangeSummary,
		CreatedAt:     version.CreatedAt,
	}
}

func documentChangeSummary(status string, versionNo int) string {
	if versionNo == 1 {
		return "创建文档"
	}
	if status == DocumentStatusPublished {
		return "发布文档"
	}
	return "保存草稿"
}

func markdownToText(content string) string {
	text := content
	text = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(text, " ")
	text = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\[[^\]]+\]\([^)]+\)`).ReplaceAllStringFunc(text, func(match string) string {
		label := regexp.MustCompile(`^\[([^\]]+)\].*$`).FindStringSubmatch(match)
		if len(label) > 1 {
			return label[1]
		}
		return " "
	})
	text = regexp.MustCompile(`[#>*_\x60-]`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func markdownTOCJSON(content string) string {
	type tocItem struct {
		Level int    `json:"level"`
		Text  string `json:"text"`
	}

	items := make([]tocItem, 0)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		level := 0
		text := ""
		switch {
		case strings.HasPrefix(trimmed, "### "):
			level = 3
			text = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
		case strings.HasPrefix(trimmed, "## "):
			level = 2
			text = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
		case strings.HasPrefix(trimmed, "# "):
			level = 1
			text = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
		if level > 0 && text != "" {
			items = append(items, tocItem{Level: level, Text: text})
		}
	}

	data, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func normalizeDocumentStatus(status string) string {
	switch strings.TrimSpace(status) {
	case DocumentStatusPublished:
		return DocumentStatusPublished
	default:
		return DocumentStatusDraft
	}
}

func documentStatusLabel(status string) string {
	if status == DocumentStatusPublished {
		return "已发布"
	}
	return "草稿"
}

func documentTypeLabel(value string) string {
	labels := map[string]string{
		"general":     "通用文档",
		"intro":       "项目介绍",
		"environment": "环境说明",
		"deploy":      "部署说明",
		"api":         "API 说明",
		"access":      "账号/权限申请说明",
		"faq":         "FAQ",
		"standard":    "开发规范",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}
