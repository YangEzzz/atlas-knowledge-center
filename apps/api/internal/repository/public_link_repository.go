package repository

import (
	"strings"

	"ty-notebook-api/internal/audit"
	"ty-notebook-api/internal/lifecycle"
	"ty-notebook-api/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		enabled := link.IsEnabled
		if err := tx.Create(link).Error; err != nil {
			return err
		}
		if !enabled {
			if err := tx.Model(link).Update("is_enabled", false).Error; err != nil {
				return err
			}
			link.IsEnabled = false
		}
		return audit.Write(tx, link.CreatedBy, "tool", link.ID, link.Name, nil, "create", "创建工具导航")
	})
}

func (r *PublicLinkRepository) Update(link *models.PublicLink) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var old models.PublicLink
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, link.ID).Error; err != nil {
			return err
		}
		fields := []string{}
		if old.Name != link.Name {
			fields = append(fields, "名称")
		}
		if old.Type != link.Type {
			fields = append(fields, "类型")
		}
		if old.URL != link.URL {
			fields = append(fields, "地址")
		}
		if old.Description != link.Description {
			fields = append(fields, "说明")
		}
		if old.IsEnabled != link.IsEnabled {
			fields = append(fields, "启用状态")
		}
		if old.SortOrder != link.SortOrder {
			fields = append(fields, "排序")
		}
		if err := tx.Save(link).Error; err != nil {
			return err
		}
		if len(fields) == 0 {
			return nil
		}
		return audit.Write(tx, link.UpdatedBy, "tool", link.ID, link.Name, nil, "update", "更新工具导航", fields...)
	})
}

func (r *PublicLinkRepository) Delete(id uint, deletedBy *uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var link models.PublicLink
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&link, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PublicLink{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
			return err
		}
		if err := tx.Delete(&link).Error; err != nil {
			return err
		}
		return audit.Write(tx, deletedBy, "tool", id, link.Name, nil, "delete", "删除工具导航")
	})
}

func (r *PublicLinkRepository) Restore(id uint, actor ...*uint) error {
	var user *uint
	if len(actor) > 0 {
		user = actor[0]
	}
	return lifecycle.Restore(r.db, "tool", id, user)
}
