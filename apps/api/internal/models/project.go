package models

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:200;not null;index" json:"name"`
	Type        string         `gorm:"size:50;not null;index" json:"type"`
	Description string         `gorm:"type:text" json:"description"`
	Keywords    string         `gorm:"size:1000" json:"keywords"`
	Visibility  string         `gorm:"size:50" json:"visibility"`
	OwnerID     *uint          `gorm:"index" json:"owner_id"`
	Source      string         `gorm:"size:50;default:manual" json:"source"`
	SortOrder   int            `gorm:"default:0;index" json:"sort_order"`
	CreatedBy   *uint          `gorm:"index" json:"created_by"`
	UpdatedBy   *uint          `gorm:"index" json:"updated_by"`
	DeletedBy   *uint          `gorm:"index" json:"deleted_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Owners    []ProjectOwner    `gorm:"foreignKey:ProjectID" json:"owners"`
	Entries   []ProjectEntry    `gorm:"foreignKey:ProjectID" json:"entries"`
	Documents []ProjectDocument `gorm:"foreignKey:ProjectID" json:"documents"`
}

type ProjectOwner struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index;uniqueIndex:idx_project_owner_unique" json:"project_id"`
	UserID    *uint     `gorm:"index;uniqueIndex:idx_project_owner_unique" json:"user_id"`
	OwnerName string    `gorm:"size:100;not null;uniqueIndex:idx_project_owner_unique" json:"owner_name"`
	RoleType  string    `gorm:"size:50;not null;index;uniqueIndex:idx_project_owner_unique" json:"role_type"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedBy *uint     `gorm:"index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectEntry struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProjectID uint           `gorm:"not null;index" json:"project_id"`
	Type      string         `gorm:"size:50;not null;index" json:"type"`
	Title     string         `gorm:"size:100;not null" json:"title"`
	URL       string         `gorm:"size:1000;not null" json:"url"`
	Remark    string         `gorm:"type:text;not null;default:''" json:"remark"`
	SortOrder int            `gorm:"default:0;index" json:"sort_order"`
	CreatedBy *uint          `gorm:"index" json:"created_by"`
	UpdatedBy *uint          `gorm:"index" json:"updated_by"`
	DeletedBy *uint          `gorm:"index" json:"deleted_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProjectDocument struct {
	ID               uint              `gorm:"primaryKey" json:"id"`
	ProjectID        *uint             `gorm:"index;uniqueIndex:idx_project_document_type" json:"project_id"`
	Type             string            `gorm:"size:80;not null;index;uniqueIndex:idx_project_document_type" json:"type"`
	Title            string            `gorm:"size:200;not null;index" json:"title"`
	Summary          string            `gorm:"size:500" json:"summary"`
	Content          string            `gorm:"type:text" json:"content"`
	ContentText      string            `gorm:"type:text" json:"content_text"`
	TOCJSON          string            `gorm:"type:text" json:"toc_json"`
	CurrentVersionNo int               `gorm:"not null;default:0" json:"current_version_no"`
	Status           string            `gorm:"size:50;not null;default:draft;index" json:"status"`
	CreatedBy        *uint             `gorm:"index" json:"created_by"`
	UpdatedBy        *uint             `gorm:"index" json:"updated_by"`
	DeletedBy        *uint             `gorm:"index" json:"deleted_by"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `gorm:"index" json:"-"`
	Versions         []DocumentVersion `gorm:"foreignKey:DocumentID" json:"versions"`
}

type DocumentVersion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DocumentID    uint      `gorm:"not null;index;uniqueIndex:idx_document_version_unique" json:"document_id"`
	VersionNo     int       `gorm:"not null;uniqueIndex:idx_document_version_unique" json:"version_no"`
	Title         string    `gorm:"size:200;not null" json:"title"`
	Type          string    `gorm:"size:80;not null;index" json:"type"`
	Summary       string    `gorm:"size:500" json:"summary"`
	Content       string    `gorm:"type:text" json:"content"`
	ContentText   string    `gorm:"type:text" json:"content_text"`
	TOCJSON       string    `gorm:"type:text" json:"toc_json"`
	ChangeSummary string    `gorm:"size:500" json:"change_summary"`
	CreatedBy     *uint     `gorm:"index" json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type PublicLink struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:200;not null;index" json:"name"`
	Type        string         `gorm:"size:50;not null;index" json:"type"`
	URL         string         `gorm:"size:1000;not null" json:"url"`
	Description string         `gorm:"size:500" json:"description"`
	IsEnabled   bool           `gorm:"not null;default:true;index" json:"is_enabled"`
	SortOrder   int            `gorm:"default:0;index" json:"sort_order"`
	CreatedBy   *uint          `gorm:"index" json:"created_by"`
	UpdatedBy   *uint          `gorm:"index" json:"updated_by"`
	DeletedBy   *uint          `gorm:"index" json:"deleted_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
