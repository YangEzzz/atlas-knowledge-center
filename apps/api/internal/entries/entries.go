package entries

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"ty-notebook-api/internal/accounts"
	"ty-notebook-api/internal/models"
)

var ErrInvalid = errors.New("invalid entry")
var ErrConflict = errors.New("entry changed")

type Input struct {
	Source    string `json:"source" binding:"oneof=private project tool document"`
	TargetID  uint   `json:"targetId"`
	Kind      string `json:"kind" binding:"oneof=project tool document"`
	Name      string `json:"name" binding:"required,max=200"`
	URL       string `json:"url" binding:"max=1000"`
	GroupName string `json:"groupName" binding:"max=80"`
	Pinned    bool   `json:"pinned"`
	Note      string `json:"note" binding:"max=10000"`
	Revision  uint   `json:"revision"`
}
type Item struct {
	models.PersonalEntry
	Note         string     `json:"note"`
	Available    bool       `json:"available"`
	AccountCount int64      `json:"accountCount"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}
type Service struct {
	db    *gorm.DB
	crypt *accounts.Crypt
}

func New(db *gorm.DB, crypt *accounts.Crypt) *Service {
	return &Service{db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), crypt}
}
func aad(e models.PersonalEntry) string { return fmt.Sprintf("atlas-entry:v1:%d:%d", e.UserID, e.ID) }
func validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Hostname() != "" && u.User == nil && (u.Scheme == "http" || u.Scheme == "https")
}
func resolve(db *gorm.DB, e *models.PersonalEntry) error {
	switch e.Source {
	case "private":
		return nil
	case "project":
		var p models.Project
		if err := db.Select("id,name").First(&p, e.TargetID).Error; err != nil {
			return err
		}
		e.Name, e.URL, e.Kind = p.Name, fmt.Sprintf("/projects/%d", p.ID), "project"
	case "tool":
		var p models.PublicLink
		if err := db.Select("id,name,url,is_enabled").First(&p, e.TargetID).Error; err != nil {
			return err
		}
		e.Name, e.URL, e.Kind = p.Name, p.URL, "tool"
		if !p.IsEnabled {
			return gorm.ErrRecordNotFound
		}
	case "document":
		var p models.ProjectDocument
		if err := db.Select("id,title").First(&p, e.TargetID).Error; err != nil {
			return err
		}
		e.Name, e.URL, e.Kind = p.Title, fmt.Sprintf("/documents/%d", p.ID), "document"
	default:
		return ErrInvalid
	}
	return nil
}
func (s *Service) owned(db *gorm.DB, user, id uint, trash bool) (models.PersonalEntry, error) {
	var e models.PersonalEntry
	if user == 0 {
		return e, gorm.ErrRecordNotFound
	}
	q := db.Where("user_id = ? AND id = ?", user, id)
	if trash {
		q = q.Unscoped().Where("source = 'private' AND deleted_at > ?", time.Now().Add(-7*24*time.Hour))
	}
	err := q.First(&e).Error
	return e, err
}
func (s *Service) item(e models.PersonalEntry) (Item, error) {
	i := Item{PersonalEntry: e, Available: true}
	if err := resolve(s.db, &i.PersonalEntry); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return i, err
		}
		i.Available = false
	}
	if e.NoteCiphertext != "" {
		var err error
		i.Note, err = s.crypt.Open(e.NoteCiphertext, aad(e))
		if err != nil {
			return i, err
		}
	}
	if e.DeletedAt.Valid {
		d := e.DeletedAt.Time
		i.DeletedAt = &d
		i.Available = false
	}
	if e.Source == "private" {
		if err := s.db.Model(&accounts.Account{}).Where("user_id = ? AND entry_id = ?", e.UserID, e.ID).Count(&i.AccountCount).Error; err != nil {
			return i, err
		}
	}
	return i, nil
}
func (s *Service) List(user uint, keyword, source, group string, trash bool, page, size int) ([]Item, int64, error) {
	if user == 0 {
		return nil, 0, gorm.ErrRecordNotFound
	}
	q := s.db.Model(&models.PersonalEntry{}).Where("user_id = ?", user)
	if trash {
		q = q.Unscoped().Where("source = 'private' AND deleted_at > ?", time.Now().Add(-7*24*time.Hour))
	}
	if source == "public" {
		q = q.Where("source <> 'private'")
	} else if source != "" {
		q = q.Where("source = ?", source)
	}
	if group != "" {
		q = q.Where("group_name = ?", group)
	}
	if keyword != "" {
		q = q.Where("(name ILIKE ? OR url ILIKE ? OR group_name ILIKE ? OR (source = 'project' AND EXISTS(SELECT 1 FROM projects WHERE projects.id=personal_entries.target_id AND projects.name ILIKE ?)) OR (source='tool' AND EXISTS(SELECT 1 FROM public_links WHERE public_links.id=personal_entries.target_id AND public_links.name ILIKE ?)) OR (source='document' AND EXISTS(SELECT 1 FROM project_documents WHERE project_documents.id=personal_entries.target_id AND project_documents.title ILIKE ?)))", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.PersonalEntry
	if err := q.Order("pinned DESC,updated_at DESC,id DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]Item, 0, len(rows))
	for _, e := range rows {
		i, err := s.item(e)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, nil
}
func (s *Service) Get(user, id uint) (Item, error) {
	e, err := s.owned(s.db, user, id, false)
	if err != nil {
		return Item{}, err
	}
	return s.item(e)
}
func (s *Service) Save(user, id uint, in Input) (Item, error) {
	if user == 0 || strings.TrimSpace(in.Name) == "" || len([]rune(in.Name)) > 200 || len(in.URL) > 1000 || len([]rune(in.GroupName)) > 80 || len([]rune(in.Note)) > 10000 {
		return Item{}, ErrInvalid
	}
	if in.Source == "private" {
		if in.TargetID != 0 || (in.Kind != "project" && in.Kind != "tool") || !validURL(strings.TrimSpace(in.URL)) {
			return Item{}, ErrInvalid
		}
	} else if in.TargetID == 0 || (in.Source != "project" && in.Source != "tool" && in.Source != "document") {
		return Item{}, ErrInvalid
	}
	var e models.PersonalEntry
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if id > 0 {
			e, err = s.owned(tx.Clauses(clause.Locking{Strength: "UPDATE"}), user, id, false)
			if err != nil {
				return err
			}
			if e.Revision != in.Revision {
				return ErrConflict
			}
			if e.Source != in.Source || e.TargetID != in.TargetID {
				return ErrInvalid
			}
		} else {
			e = models.PersonalEntry{UserID: user, Source: in.Source, TargetID: in.TargetID, Revision: 1}
		}
		oldName, oldURL, oldKind := e.Name, e.URL, e.Kind
		e.Name, e.URL, e.Kind = strings.TrimSpace(in.Name), strings.TrimSpace(in.URL), in.Kind
		if err = resolve(tx, &e); err != nil {
			if id == 0 || !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			e.Name, e.URL, e.Kind = oldName, oldURL, oldKind
		}
		e.GroupName, e.Pinned = strings.TrimSpace(in.GroupName), in.Pinned
		if id == 0 {
			if err = tx.Create(&e).Error; err != nil {
				return err
			}
		}
		e.NoteCiphertext = s.crypt.Seal(in.Note, aad(e))
		e.Revision++
		if err = tx.Save(&e).Error; err != nil {
			return err
		}
		if e.Source == "private" && id > 0 && oldURL != e.URL {
			return tx.Model(&accounts.Account{}).Unscoped().Where("user_id=? AND entry_id=?", user, id).Updates(map[string]any{"resource_url": e.URL, "revision": gorm.Expr("revision+1")}).Error
		}
		return nil
	})
	if err != nil {
		return Item{}, err
	}
	return s.item(e)
}

// Import preserves legacy bookmarks, including unavailable public targets. Identity and paths are server-derived.
func (s *Service) Import(user uint, inputs []Input) error {
	if user == 0 || len(inputs) > 100 {
		return ErrInvalid
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, in := range inputs {
			if in.TargetID == 0 || (in.Source != "project" && in.Source != "tool" && in.Source != "document") || len([]rune(in.Name)) > 200 {
				return ErrInvalid
			}
			e := models.PersonalEntry{UserID: user, Source: in.Source, TargetID: in.TargetID, Kind: in.Source, Name: in.Name, Revision: 1}
			if err := resolve(tx, &e); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// A missing public target remains visible as unavailable, never an arbitrary stored URL.
			if e.Source == "project" {
				e.URL = fmt.Sprintf("/projects/%d", e.TargetID)
			}
			if e.Source == "document" {
				e.URL = fmt.Sprintf("/documents/%d", e.TargetID)
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&e).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Service) Delete(user, id, revision uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.owned(tx.Clauses(clause.Locking{Strength: "UPDATE"}), user, id, false)
		if err != nil {
			return err
		}
		if e.Revision != revision {
			return ErrConflict
		}
		if e.Source != "private" {
			return tx.Unscoped().Delete(&e).Error
		}
		now := time.Now()
		if err = tx.Model(&accounts.Account{}).Where("user_id=? AND entry_id=?", user, id).Updates(map[string]any{"deleted_at": now, "entry_cascade": true, "revision": gorm.Expr("revision+1")}).Error; err != nil {
			return err
		}
		return tx.Model(&e).Updates(map[string]any{"deleted_at": now, "revision": gorm.Expr("revision+1")}).Error
	})
}
func (s *Service) Restore(user, id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.owned(tx.Clauses(clause.Locking{Strength: "UPDATE"}), user, id, true)
		if err != nil {
			return err
		}
		if err = tx.Unscoped().Model(&e).Updates(map[string]any{"deleted_at": nil, "revision": gorm.Expr("revision+1")}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Model(&accounts.Account{}).Where("user_id=? AND entry_id=? AND entry_cascade=true AND deleted_at > ?", user, id, time.Now().Add(-7*24*time.Hour)).Updates(map[string]any{"deleted_at": nil, "entry_cascade": false, "revision": gorm.Expr("revision+1")}).Error
	})
}
func PurgeExpired(db *gorm.DB) error {
	return db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Unscoped().Where("source='private' AND deleted_at <= ?", time.Now().Add(-7*24*time.Hour)).Delete(&models.PersonalEntry{}).Error
}
