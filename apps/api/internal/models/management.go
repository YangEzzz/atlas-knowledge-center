package models

import "time"

type DocumentTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:200;not null" json:"name"`
	Type      string    `gorm:"size:80;not null;index" json:"type"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsEnabled bool      `gorm:"not null;default:false" json:"isEnabled"`
	UpdatedBy *uint     `json:"updatedBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    *uint     `gorm:"index" json:"actorId"`
	ActorName  string    `gorm:"size:100" json:"actorName"`
	ObjectType string    `gorm:"size:40;not null;index" json:"objectType"`
	ObjectID   uint      `gorm:"not null;index" json:"objectId"`
	ObjectName string    `gorm:"size:200" json:"objectName"`
	ProjectID  *uint     `gorm:"index" json:"projectId"`
	Action     string    `gorm:"size:40;not null;index" json:"action"`
	Summary    string    `gorm:"size:1000" json:"summary"`
	Fields     string    `gorm:"type:text" json:"fields"`
	CreatedAt  time.Time `gorm:"index" json:"createdAt"`
}
