package repository

import (
	"strings"

	"ty-notebook-api/internal/models"

	"gorm.io/gorm"
)

type PublicLinkRepository struct {
	db *gorm.DB
}

type PublicLinkListFilter struct {
	Keyword  string
	Type     string
	Enabled  *bool
	Page     int
	PageSize int
}

func NewPublicLinkRepository(db *gorm.DB) *PublicLinkRepository {
	return &PublicLinkRepository{db: db}
}

func (r *PublicLinkRepository) List(filter PublicLinkListFilter) ([]models.PublicLink, int64, error) {
	var links []models.PublicLink
	var total int64

	query := r.db.Model(&models.PublicLink{})
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Enabled != nil {
		query = query.Where("is_enabled = ?", *filter.Enabled)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ? OR url ILIKE ?", like, like, like)
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
		Order("sort_order asc, updated_at desc, id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&links).Error
	return links, total, err
}

func (r *PublicLinkRepository) FindByID(id uint) (*models.PublicLink, error) {
	var link models.PublicLink
	if err := r.db.First(&link, id).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *PublicLinkRepository) Create(link *models.PublicLink) error {
	return r.db.Create(link).Error
}

func (r *PublicLinkRepository) Update(link *models.PublicLink) error {
	return r.db.Save(link).Error
}

func (r *PublicLinkRepository) Delete(id uint, deletedBy *uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PublicLink{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&models.PublicLink{}, id).Error
	})
}

func (r *PublicLinkRepository) Restore(id uint) error {
	return r.db.Unscoped().
		Model(&models.PublicLink{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
		}).Error
}
