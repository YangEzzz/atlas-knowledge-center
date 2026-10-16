package models

import (
	"gorm.io/gorm"
	"time"
)

type PersonalEntry struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"not null;index" json:"-"`
	Source         string         `gorm:"size:20;not null" json:"source"`
	TargetID       uint           `gorm:"not null;default:0" json:"targetId"`
	Kind           string         `gorm:"size:20;not null" json:"kind"`
	Name           string         `gorm:"size:200;not null" json:"name"`
	URL            string         `gorm:"size:1000;not null" json:"url"`
	GroupName      string         `gorm:"size:80;not null;default:''" json:"groupName"`
	Pinned         bool           `gorm:"not null;default:false" json:"pinned"`
	NoteCiphertext string         `gorm:"type:text;not null;default:''" json:"-"`
	Revision       uint           `gorm:"not null;default:1" json:"revision"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
