package repository

import (
	"strings"

	"ty-notebook-api/internal/models"

	"gorm.io/gorm"
)

type DocumentRepository struct {
	db *gorm.DB
}

type DocumentListFilter struct {
	Keyword   string
	Type      string
	Status    string
	ProjectID uint
	Page      int
	PageSize  int
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) List(filter DocumentListFilter) ([]models.ProjectDocument, int64, error) {
	return r.list(filter, false)
}

func (r *DocumentRepository) ListDeleted(filter DocumentListFilter) ([]models.ProjectDocument, int64, error) {
	return r.list(filter, true)
}

func (r *DocumentRepository) list(filter DocumentListFilter, deletedOnly bool) ([]models.ProjectDocument, int64, error) {
	var documents []models.ProjectDocument
	var total int64

	query := r.db.Model(&models.ProjectDocument{})
	if deletedOnly {
		query = query.Unscoped().Where("deleted_at IS NOT NULL")
	}
	if filter.ProjectID > 0 {
		query = query.Where("project_id = ?", filter.ProjectID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"title ILIKE ? OR summary ILIKE ? OR content ILIKE ?",
			like,
			like,
			like,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	err := query.
		Order("updated_at desc, id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&documents).Error
	return documents, total, err
}

func (r *DocumentRepository) FindByID(id uint) (*models.ProjectDocument, error) {
	var document models.ProjectDocument
	if err := r.db.First(&document, id).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

func (r *DocumentRepository) FindDeletedByID(id uint) (*models.ProjectDocument, error) {
	var document models.ProjectDocument
	if err := r.db.Unscoped().
		Where("deleted_at IS NOT NULL").
		First(&document, id).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

func (r *DocumentRepository) FindByProjectAndType(projectID uint, docType string) (*models.ProjectDocument, error) {
	var document models.ProjectDocument
	if err := r.db.
		Where("project_id = ? AND type = ?", projectID, docType).
		First(&document).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

func (r *DocumentRepository) Create(document *models.ProjectDocument) error {
	return r.db.Create(document).Error
}

func (r *DocumentRepository) Update(document *models.ProjectDocument) error {
	return r.db.Save(document).Error
}

func (r *DocumentRepository) SaveWithVersion(document *models.ProjectDocument, version *models.DocumentVersion) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if document.ID == 0 {
			if err := tx.Create(document).Error; err != nil {
				return err
			}
		} else if err := tx.Save(document).Error; err != nil {
			return err
		}

		version.DocumentID = document.ID
		if err := tx.Create(version).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *DocumentRepository) ListVersions(documentID uint) ([]models.DocumentVersion, error) {
	var versions []models.DocumentVersion
	err := r.db.
		Where("document_id = ?", documentID).
		Order("version_no desc, id desc").
		Find(&versions).Error
	return versions, err
}

func (r *DocumentRepository) FindVersion(documentID uint, versionID uint) (*models.DocumentVersion, error) {
	var version models.DocumentVersion
	if err := r.db.
		Where("document_id = ? AND id = ?", documentID, versionID).
		First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *DocumentRepository) FindVersionByID(versionID uint) (*models.DocumentVersion, error) {
	var version models.DocumentVersion
	if err := r.db.First(&version, versionID).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *DocumentRepository) Delete(document *models.ProjectDocument, actorID *uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		document.DeletedBy = actorID
		if err := tx.Save(document).Error; err != nil {
			return err
		}
		return tx.Delete(document).Error
	})
}

func (r *DocumentRepository) Restore(id uint) error {
	return r.db.Unscoped().
		Model(&models.ProjectDocument{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
		}).Error
}
