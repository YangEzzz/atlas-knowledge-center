package lifecycle

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ty-notebook-api/internal/audit"
	"ty-notebook-api/internal/models"
)

var ErrConflict = errors.New("记录已恢复或状态已变化，请刷新列表")
var ErrParent = errors.New("所属项目已删除，请先恢复项目")

func Restore(db *gorm.DB, kind string, id uint, actor *uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var target any
		var name string
		var project *uint
		switch kind {
		case "project":
			target = &models.Project{}
		case "entry":
			target = &models.ProjectEntry{}
		case "document":
			target = &models.ProjectDocument{}
		case "tool":
			target = &models.PublicLink{}
		default:
			return errors.New("不支持的对象类型")
		}
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("deleted_at IS NOT NULL").First(target, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrConflict
			}
			return err
		}
		switch item := target.(type) {
		case *models.Project:
			name = item.Name
			project = &item.ID
		case *models.ProjectEntry:
			name = item.Title
			project = &item.ProjectID
		case *models.ProjectDocument:
			name = item.Title
			project = &item.ProjectID
		case *models.PublicLink:
			name = item.Name
		}
		if project != nil && kind != "project" {
			var parent models.Project
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&parent, *project).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrParent
				}
				return err
			}
		}
		result := tx.Unscoped().Model(target).Where("id = ? AND deleted_at IS NOT NULL", id).Updates(map[string]any{"deleted_at": nil, "deleted_by": nil, "updated_by": actor})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrConflict
		}
		return audit.Write(tx, actor, kind, id, name, project, "restore", "从回收站恢复")
	})
}
