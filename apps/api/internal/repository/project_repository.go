package repository

import (
	"strings"

	"ty-notebook-api/internal/audit"
	"ty-notebook-api/internal/lifecycle"
	"ty-notebook-api/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			`projects.name ILIKE ? OR projects.description ILIKE ? OR projects.keywords ILIKE ?
            OR EXISTS (SELECT 1 FROM project_entries WHERE project_entries.project_id = projects.id AND project_entries.deleted_at IS NULL AND (project_entries.title ILIKE ? OR project_entries.url ILIKE ?))
            OR EXISTS (SELECT 1 FROM project_owners WHERE project_owners.project_id = projects.id AND project_owners.owner_name ILIKE ?)`,
			like,
			like,
			like,
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
		var project models.Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Project{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
			return err
		}
		if err := tx.Delete(&project).Error; err != nil {
			return err
		}
		return audit.Write(tx, deletedBy, "project", id, project.Name, &id, "delete", "删除项目；子项保持原状态")
	})
}

func (r *ProjectRepository) Restore(id uint, actor ...*uint) error {
	var user *uint
	if len(actor) > 0 {
		user = actor[0]
	}
	return lifecycle.Restore(r.db, "project", id, user)
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
		_, err := saveEntries(tx, projectID, entries, nil)
		return err
	})
}
