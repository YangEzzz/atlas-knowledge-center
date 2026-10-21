package audit

import (
	"gorm.io/gorm"
	"strings"
	"ty-notebook-api/internal/models"
)

// Write must use the business transaction. Never pass private content here.
func Write(tx *gorm.DB, actor *uint, kind string, id uint, name string, project *uint, action, summary string, fields ...string) error {
	actorName := "历史记录"
	if actor != nil {
		var user models.User
		if err := tx.First(&user, *actor).Error; err != nil {
			return err
		}
		actorName = user.DisplayName
		if actorName == "" {
			actorName = user.Username
		}
	}
	return tx.Create(&models.AuditLog{ActorID: actor, ActorName: actorName, ObjectType: kind, ObjectID: id, ObjectName: name, ProjectID: project, Action: action, Summary: summary, Fields: strings.Join(fields, "、")}).Error
}
