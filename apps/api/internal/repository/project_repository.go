package repository

import (
	"strings"

	"ty-notebook-api/internal/models"

	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

type ProjectListFilter struct {
	Keyword  string
	Type     string
	Page     int
	PageSize int
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) List(filter ProjectListFilter) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	query := r.db.Model(&models.Project{}).Preload("Owners").Preload("Entries", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc, id asc")
	})
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"name ILIKE ? OR description ILIKE ? OR keywords ILIKE ?",
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
		Order("sort_order asc, updated_at desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&projects).Error
	return projects, total, err
}

func (r *ProjectRepository) FindByID(id uint) (*models.Project, error) {
	var project models.Project
	if err := r.db.
		Preload("Owners").
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc, id asc")
		}).
		First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

func (r *ProjectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

func (r *ProjectRepository) Delete(id uint, deletedBy *uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Project{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Project{}, id).Error
	})
}

func (r *ProjectRepository) Restore(id uint) error {
	return r.db.Unscoped().
		Model(&models.Project{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
		}).Error
}

func (r *ProjectRepository) ReplaceOwners(projectID uint, owners []models.ProjectOwner) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&models.ProjectOwner{}).Error; err != nil {
			return err
		}
		if len(owners) == 0 {
			return nil
		}
		return tx.Create(&owners).Error
	})
}

func (r *ProjectRepository) ReplaceEntries(projectID uint, entries []models.ProjectEntry) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(&models.ProjectEntry{}).Error; err != nil {
			return err
		}
		if len(entries) == 0 {
			return nil
		}
		return tx.Create(&entries).Error
	})
}
