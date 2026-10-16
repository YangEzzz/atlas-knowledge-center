package accounts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/models"
)

type Account struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"not null;index" json:"-"`
	ProjectID    uint           `gorm:"not null;index" json:"projectId"`
	ToolID       uint           `gorm:"not null;default:0;index" json:"toolId"`
	EntryID      uint           `gorm:"not null;default:0;index;check:accounts_single_target,(project_id > 0 AND tool_id = 0 AND entry_id = 0) OR (project_id = 0 AND tool_id > 0 AND entry_id = 0) OR (project_id = 0 AND tool_id = 0 AND entry_id > 0)" json:"entryId"`
	EntryCascade bool           `gorm:"not null;default:false" json:"-"`
	Name         string         `gorm:"size:200;not null" json:"name"`
	Environment  string         `gorm:"size:30;not null" json:"environment"`
	Username     string         `gorm:"size:300;not null" json:"username"`
	ResourceURL  string         `gorm:"size:1000" json:"resourceUrl"`
	Ciphertext   string         `gorm:"type:text;not null" json:"-"`
	HasPassword  bool           `gorm:"not null" json:"hasPassword"`
	Revision     uint           `gorm:"not null;default:1" json:"revision"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
type Item struct {
	Account
	ProjectName string     `json:"projectName"`
	ToolName    string     `json:"toolName"`
	EntryName   string     `json:"entryName"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}
type Input struct {
	ProjectID      uint   `json:"projectId"`
	ToolID         uint   `json:"toolId"`
	EntryID        uint   `json:"entryId"`
	Name           string `json:"name" binding:"max=200"`
	Environment    string `json:"environment" binding:"oneof=production staging other"`
	Username       string `json:"username" binding:"required,max=300"`
	ResourceURL    string `json:"resourceUrl" binding:"max=1000"`
	Note           string `json:"note" binding:"max=10000"`
	PasswordAction string `json:"passwordAction" binding:"oneof=keep replace clear"`
	Password       string `json:"password" binding:"max=4096"`
	Revision       uint   `json:"revision"`
}
type secret struct {
	Password string `json:"password"`
	Note     string `json:"note"`
}
type Service struct {
	db    *gorm.DB
	crypt *Crypt
}

var ErrConflict = errors.New("record changed")
var ErrInvalid = errors.New("invalid account")

func NewService(db *gorm.DB, crypt *Crypt) *Service {
	return &Service{db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), crypt}
}
func aad(a Account) string {
	if a.EntryID > 0 {
		return fmt.Sprintf("atlas-account:entry:v1:%d:%d:%d", a.UserID, a.EntryID, a.ID)
	}
	if a.ToolID > 0 {
		return fmt.Sprintf("atlas-account:tool:v1:%d:%d:%d", a.UserID, a.ToolID, a.ID)
	}
	return fmt.Sprintf("atlas-account:v1:%d:%d:%d", a.UserID, a.ProjectID, a.ID)
}
func (s *Service) owned(db *gorm.DB, user, id uint, trash bool) (Account, error) {
	var a Account
	if user == 0 {
		return a, gorm.ErrRecordNotFound
	}
	q := db.Where("user_id = ? AND id = ?", user, id)
	if trash {
		q = q.Unscoped().Where("deleted_at > ?", time.Now().Add(-7*24*time.Hour))
	}
	err := q.First(&a).Error
	return a, err
}
func validTarget(project, tool, entry uint) bool {
	n := 0
	for _, id := range []uint{project, tool, entry} {
		if id > 0 {
			n++
		}
	}
	return n == 1
}
func (s *Service) activeTarget(db *gorm.DB, a Account) error {
	if !validTarget(a.ProjectID, a.ToolID, a.EntryID) {
		return ErrInvalid
	}
	if a.EntryID > 0 {
		return db.Clauses(clause.Locking{Strength: "SHARE"}).Select("id").Where("user_id=? AND source='private'", a.UserID).First(&models.PersonalEntry{}, a.EntryID).Error
	}
	if a.ToolID > 0 {
		return db.Select("id").First(&models.PublicLink{}, a.ToolID).Error
	}
	return db.Select("id").First(&models.Project{}, a.ProjectID).Error
}
func (s *Service) decrypt(a Account) (secret, error) {
	var v secret
	raw, err := s.crypt.Open(a.Ciphertext, aad(a))
	if err != nil {
		return v, err
	}
	err = json.Unmarshal([]byte(raw), &v)
	return v, err
}
func (s *Service) List(user, project uint, keyword, environment string, trash bool, page, size int, tools ...uint) ([]Item, int64, error) {
	q := s.db.Model(&Account{}).Where("user_id = ?", user)
	if user == 0 {
		return nil, 0, gorm.ErrRecordNotFound
	}
	if trash {
		q = q.Unscoped().Where("deleted_at > ?", time.Now().Add(-7*24*time.Hour))
	} else {
		q = q.Where("((tool_id = 0 AND EXISTS (SELECT 1 FROM projects WHERE projects.id = accounts.project_id AND projects.deleted_at IS NULL)) OR (project_id = 0 AND EXISTS (SELECT 1 FROM public_links WHERE public_links.id = accounts.tool_id AND public_links.deleted_at IS NULL)) OR (entry_id>0 AND EXISTS (SELECT 1 FROM personal_entries WHERE personal_entries.id=accounts.entry_id AND personal_entries.user_id=accounts.user_id AND personal_entries.source='private' AND personal_entries.deleted_at IS NULL)))")
	}
	if project > 0 {
		q = q.Where("project_id = ?", project)
	}
	if len(tools) > 0 && tools[0] > 0 {
		q = q.Where("tool_id = ?", tools[0])
	}
	if len(tools) > 1 && tools[1] > 0 {
		q = q.Where("entry_id=?", tools[1])
	}
	if keyword != "" {
		term := "%" + keyword + "%"
		q = q.Where("(name ILIKE ? OR username ILIKE ? OR EXISTS (SELECT 1 FROM projects WHERE projects.id = accounts.project_id AND projects.name ILIKE ?) OR EXISTS (SELECT 1 FROM public_links WHERE public_links.id = accounts.tool_id AND public_links.name ILIKE ?) OR EXISTS (SELECT 1 FROM personal_entries WHERE personal_entries.id=accounts.entry_id AND personal_entries.user_id=accounts.user_id AND personal_entries.name ILIKE ?))", term, term, term, term, term)
	}
	if environment != "" {
		q = q.Where("environment = ?", environment)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Account
	if err := q.Order("updated_at DESC, id DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]Item, 0, len(rows))
	ids := []uint{}
	toolIDs := []uint{}
	entryIDs := []uint{}
	for _, a := range rows {
		if a.EntryID > 0 {
			entryIDs = append(entryIDs, a.EntryID)
		}
		if a.ProjectID > 0 {
			ids = append(ids, a.ProjectID)
		}
		if a.ToolID > 0 {
			toolIDs = append(toolIDs, a.ToolID)
		}
	}
	var projects []models.Project
	if len(ids) > 0 {
		if err := s.db.Unscoped().Select("id,name").Where("id IN ?", ids).Find(&projects).Error; err != nil {
			return nil, 0, err
		}
	}
	names := map[uint]string{}
	for _, p := range projects {
		names[p.ID] = p.Name
	}
	toolNames := map[uint]string{}
	if len(toolIDs) > 0 {
		var links []models.PublicLink
		if err := s.db.Unscoped().Select("id,name").Where("id IN ?", toolIDs).Find(&links).Error; err != nil {
			return nil, 0, err
		}
		for _, link := range links {
			toolNames[link.ID] = link.Name
		}
	}
	entryNames := map[uint]string{}
	if len(entryIDs) > 0 {
		var entries []models.PersonalEntry
		if err := s.db.Unscoped().Where("user_id=? AND id IN ?", user, entryIDs).Find(&entries).Error; err != nil {
			return nil, 0, err
		}
		for _, e := range entries {
			entryNames[e.ID] = e.Name
		}
	}
	for _, a := range rows {
		item := Item{Account: a, ProjectName: names[a.ProjectID], ToolName: toolNames[a.ToolID], EntryName: entryNames[a.EntryID]}
		if a.DeletedAt.Valid {
			t := a.DeletedAt.Time
			item.DeletedAt = &t
		}
		result = append(result, item)
	}
	return result, total, nil
}
func (s *Service) Detail(user, id uint) (Account, string, error) {
	a, err := s.owned(s.db, user, id, false)
	if err != nil {
		return a, "", err
	}
	if err = s.activeTarget(s.db, a); err != nil {
		return a, "", err
	}
	v, err := s.decrypt(a)
	return a, v.Note, err
}
func (s *Service) Password(user, id uint) (string, error) {
	a, err := s.owned(s.db, user, id, false)
	if err != nil {
		return "", err
	}
	if err = s.activeTarget(s.db, a); err != nil {
		return "", err
	}
	v, err := s.decrypt(a)
	return v.Password, err
}
func (s *Service) Save(user, id uint, input Input) (Account, error) {
	var a Account
	if input.ResourceURL != "" {
		u, err := url.Parse(input.ResourceURL)
		if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return a, ErrInvalid
		}
	}
	if !validTarget(input.ProjectID, input.ToolID, input.EntryID) {
		return a, ErrInvalid
	}
	if user == 0 || strings.TrimSpace(input.Username) == "" || (input.PasswordAction != "keep" && input.PasswordAction != "replace" && input.PasswordAction != "clear") || (input.Environment != "production" && input.Environment != "staging" && input.Environment != "other") || (input.PasswordAction == "replace" && input.Password == "") {
		return a, ErrInvalid
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		var v secret
		if id > 0 {
			a, err = s.owned(tx, user, id, false)
			if err != nil {
				return err
			}
			if a.ProjectID != input.ProjectID || a.ToolID != input.ToolID || a.EntryID != input.EntryID {
				return ErrInvalid
			}
			if a.Revision != input.Revision {
				return ErrConflict
			}
			v, err = s.decrypt(a)
			if err != nil {
				return err
			}
		} else {
			a = Account{UserID: user, ProjectID: input.ProjectID, ToolID: input.ToolID, EntryID: input.EntryID, Revision: 1}
		}
		if err = s.activeTarget(tx, a); err != nil {
			return err
		}
		if input.ResourceURL != "" {
			var count int64
			var query *gorm.DB
			if a.EntryID > 0 {
				query = tx.Model(&models.PersonalEntry{}).Where("id=? AND user_id=? AND url=? AND source='private'", a.EntryID, user, input.ResourceURL)
			} else if a.ToolID > 0 {
				query = tx.Model(&models.PublicLink{}).Where("id = ? AND url = ?", a.ToolID, input.ResourceURL)
			} else {
				query = tx.Model(&models.ProjectEntry{}).Where("project_id = ? AND url = ?", input.ProjectID, input.ResourceURL)
			}
			if err = query.Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalid
			}
		}
		a.Name = strings.TrimSpace(input.Name)
		a.Environment = input.Environment
		a.Username = strings.TrimSpace(input.Username)
		a.ResourceURL = input.ResourceURL
		if input.PasswordAction == "replace" {
			v.Password = input.Password
		}
		if input.PasswordAction == "clear" {
			v.Password = ""
		}
		v.Note = input.Note
		a.HasPassword = v.Password != ""
		if id == 0 {
			if err = tx.Create(&a).Error; err != nil {
				return err
			}
		}
		raw, _ := json.Marshal(v)
		a.Ciphertext = s.crypt.Seal(string(raw), aad(a))
		updates := map[string]any{"name": a.Name, "environment": a.Environment, "username": a.Username, "resource_url": a.ResourceURL, "ciphertext": a.Ciphertext, "has_password": a.HasPassword, "revision": gorm.Expr("revision + 1"), "updated_at": time.Now()}
		result := tx.Model(&Account{}).Where("id = ? AND user_id = ? AND revision = ?", a.ID, user, a.Revision).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrConflict
		}
		a.Revision++
		return tx.First(&a, a.ID).Error
	})
	return a, err
}
func (s *Service) Delete(user, id, revision uint) error {
	a, err := s.owned(s.db, user, id, false)
	if err != nil {
		return err
	}
	if a.Revision != revision {
		return ErrConflict
	}
	result := s.db.Model(&Account{}).Where("id = ? AND user_id = ? AND revision = ?", id, user, revision).Updates(map[string]any{"deleted_at": time.Now(), "revision": gorm.Expr("revision + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Service) Restore(user, id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error { scoped := *s; scoped.db = tx; return scoped.restore(user, id) })
}
func (s *Service) restore(user, id uint) error {
	a, err := s.owned(s.db, user, id, true)
	if err != nil {
		return err
	}
	if err = s.activeTarget(s.db, a); err != nil {
		return err
	}
	result := s.db.Unscoped().Model(&Account{}).Where("id = ? AND user_id = ? AND deleted_at > ?", id, user, time.Now().Add(-7*24*time.Hour)).Updates(map[string]any{"deleted_at": nil, "revision": gorm.Expr("revision + 1"), "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}

// Expired records are never listed or restored, even between cleanup runs.
func PurgeExpired(db *gorm.DB) error {
	return db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Unscoped().Where("deleted_at <= ?", time.Now().Add(-7*24*time.Hour)).Delete(&Account{}).Error
}
