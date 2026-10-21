package repository

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"sort"
	"ty-notebook-api/internal/audit"
	"ty-notebook-api/internal/models"
)

// SaveProject commits the aggregate and its audit records together. Entries keep
// their identity so removal can be undone without breaking account associations.
func (r *ProjectRepository) SaveProject(project *models.Project, owners []models.ProjectOwner, entries []models.ProjectEntry, actor *uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		action := "create"
		fields := []string{"名称", "类型", "描述", "关键词", "负责人", "资源入口"}
		if project.ID != 0 {
			action = "update"
			var old models.Project
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, project.ID).Error; err != nil {
				return err
			}
			fields = nil
			if old.Name != project.Name {
				fields = append(fields, "名称")
			}
			if old.Type != project.Type {
				fields = append(fields, "类型")
			}
			if old.Description != project.Description {
				fields = append(fields, "描述")
			}
			if old.Keywords != project.Keywords {
				fields = append(fields, "关键词")
			}
			if old.SortOrder != project.SortOrder {
				fields = append(fields, "排序")
			}
		}
		if err := tx.Omit(clause.Associations).Save(project).Error; err != nil {
			return err
		}
		var oldOwners []models.ProjectOwner
		if err := tx.Where("project_id = ?", project.ID).Order("sort_order, id").Find(&oldOwners).Error; err != nil {
			return err
		}
		ownerKey := func(items []models.ProjectOwner) []string {
			result := []string{}
			for _, o := range items {
				result = append(result, o.RoleType+":"+o.OwnerName)
			}
			sort.Strings(result)
			return result
		}
		if action == "update" && !reflect.DeepEqual(ownerKey(oldOwners), ownerKey(owners)) {
			fields = append(fields, "负责人")
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&models.ProjectOwner{}).Error; err != nil {
			return err
		}
		for i := range owners {
			owners[i].ProjectID = project.ID
		}
		if len(owners) > 0 {
			if err := tx.Create(&owners).Error; err != nil {
				return err
			}
		}
		// Entries write their own audit records. Only project-owned fields
		// contribute to the project audit, avoiding a duplicate summary record.
		_, err := saveEntries(tx, project.ID, entries, actor)
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			return nil
		}
		return audit.Write(tx, actor, "project", project.ID, project.Name, &project.ID, action, map[string]string{"create": "创建项目", "update": "更新项目资料"}[action], fields...)
	})
}

func saveEntries(tx *gorm.DB, projectID uint, entries []models.ProjectEntry, actor *uint) (bool, error) {
	var old []models.ProjectEntry
	if err := tx.Where("project_id = ?", projectID).Order("sort_order, id").Find(&old).Error; err != nil {
		return false, err
	}
	used := map[uint]bool{}
	changed := false
	for i := range entries {
		e := &entries[i]
		e.ProjectID = projectID
		var previous *models.ProjectEntry
		for j := range old {
			candidate := &old[j]
			if (e.ID != 0 && candidate.ID == e.ID) || (e.ID == 0 && !used[candidate.ID] && candidate.Type == e.Type && candidate.URL == e.URL) {
				previous = candidate
				break
			}
		}
		if e.ID != 0 && (previous == nil || used[e.ID]) {
			return false, errors.New("资源入口已变化，请刷新项目后重试")
		}
		entryAction := "create"
		fields := []string{"名称", "类型", "地址", "备注", "排序"}
		if previous != nil {
			e.ID = previous.ID
			e.CreatedAt = previous.CreatedAt
			e.CreatedBy = previous.CreatedBy
			used[e.ID] = true
			entryAction = "update"
			fields = nil
			if previous.Title != e.Title {
				fields = append(fields, "名称")
			}
			if previous.Type != e.Type {
				fields = append(fields, "类型")
			}
			if previous.URL != e.URL {
				fields = append(fields, "地址")
			}
			if previous.Remark != e.Remark {
				fields = append(fields, "备注")
			}
			if previous.SortOrder != e.SortOrder {
				fields = append(fields, "排序")
			}
			if len(fields) == 0 {
				continue
			}
		}
		e.UpdatedBy = actor
		if err := tx.Save(e).Error; err != nil {
			return false, err
		}
		if err := audit.Write(tx, actor, "entry", e.ID, e.Title, &projectID, entryAction, "维护项目资源入口", fields...); err != nil {
			return false, err
		}
		changed = true
	}
	for _, e := range old {
		if used[e.ID] {
			continue
		}
		if err := tx.Model(&e).Update("deleted_by", actor).Error; err != nil {
			return false, err
		}
		if err := tx.Delete(&e).Error; err != nil {
			return false, err
		}
		if err := audit.Write(tx, actor, "entry", e.ID, e.Title, &projectID, "delete", "从项目移除资源入口"); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}
