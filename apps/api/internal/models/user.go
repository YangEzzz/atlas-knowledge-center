package models

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ExternalUserID string    `gorm:"uniqueIndex;size:100" json:"external_user_id"`
	Username       string    `gorm:"size:100;index" json:"username"`
	DisplayName    string    `gorm:"size:100" json:"display_name"`
	Email          string    `gorm:"size:200" json:"email"`
	Mobile         string    `gorm:"size:50" json:"mobile"`
	AvatarURL      string    `gorm:"size:500" json:"avatar_url"`
	RoleCode       string    `gorm:"size:50" json:"role_code"`
	Status         string    `gorm:"size:30;default:active" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
