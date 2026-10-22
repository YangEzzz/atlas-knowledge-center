package management

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ty-notebook-api/internal/api/response"
	"ty-notebook-api/internal/audit"
	"ty-notebook-api/internal/lifecycle"
	"ty-notebook-api/internal/models"
)

type Handler struct{ DB *gorm.DB }

func Admin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := db.First(&user, c.GetUint("userID")).Error; err != nil {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "请重新登录"})
			return
		}
		if user.Status != "active" || (user.RoleCode != "admin" && user.RoleCode != "owner") {
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "仅 Atlas 管理员可使用管理中心"})
			return
		}
		c.Next()
	}
}

func Register(api *gin.RouterGroup, db *gorm.DB) {
	h := &Handler{DB: db}
	api.GET("/document-templates/active/:type", h.ActiveTemplate)
	g := api.Group("/manage", Admin(db))
	g.GET("/templates", h.Templates)
	g.POST("/templates", h.SaveTemplate)
	g.PUT("/templates/:id", h.SaveTemplate)
	g.DELETE("/templates/:id", h.DeleteTemplate)
	g.GET("/trash", h.Trash)
	g.POST("/trash/:type/:id/restore", h.Restore)
	g.GET("/audit", h.Audits)
	g.GET("/audit/:id", h.AuditDetail)
}

func fail(c *gin.Context, err error) {
	status, message := 500, "操作失败，请稍后重试"
	switch {
	case errors.Is(err, errStale):
		status, message = 409, err.Error()
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, message = 404, "记录不存在，请刷新列表"
	case errors.Is(err, lifecycle.ErrConflict), errors.Is(err, lifecycle.ErrParent):
		status, message = 409, err.Error()
	case strings.Contains(err.Error(), "SQLSTATE 23505"):
		status, message = 409, "存在冲突记录，请刷新列表后重试"
	}
	c.JSON(status, response.Body{Code: status, Message: message})
}

func invalid(c *gin.Context, message string) { c.JSON(400, response.Body{Code: 400, Message: message}) }
func actor(c *gin.Context) *uint             { id := c.GetUint("userID"); return &id }
func idParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		invalid(c, "无效的 ID")
		return 0, false
	}
	return uint(id), true
}
func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if p < 1 {
		p = 1
	}
	if p > 1000000 {
		p = 1000000
	}
	if s < 1 {
		s = 20
	}
	if s > 100 {
		s = 100
	}
	return p, s
}
func list(c *gin.Context, rows any, total int64, p, s int) {
	response.OK(c, gin.H{"list": rows, "total": total, "page": p, "pageSize": s})
}
func validType(t string) bool {
	switch t {
	case "intro", "environment", "deploy", "api", "access", "faq", "standard":
		return true
	}
	return false
}

func (h *Handler) ActiveTemplate(c *gin.Context) {
	var row models.DocumentTemplate
	err := h.DB.Where("type = ? AND is_enabled = true", c.Param("type")).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.OK(c, nil)
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, row)
}

func (h *Handler) Templates(c *gin.Context) {
	p, s := page(c)
	q := h.DB.Model(&models.DocumentTemplate{})
	rows := []models.DocumentTemplate{}
	var total int64
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if k := strings.TrimSpace(c.Query("keyword")); k != "" {
		q = q.Where("name ILIKE ?", "%"+k+"%")
	}
	if v := c.Query("enabled"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			invalid(c, "无效的启用状态")
			return
		}
		q = q.Where("is_enabled = ?", enabled)
	}
	if err := q.Count(&total).Error; err != nil {
		fail(c, err)
		return
	}
	if err := q.Order("updated_at desc, id desc").Offset((p - 1) * s).Limit(s).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	list(c, rows, total, p, s)
}

type TemplateInput struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	IsEnabled bool      `json:"isEnabled"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var errStale = errors.New("模板已被其他人修改，请关闭表单并刷新列表后重试")

func (h *Handler) SaveTemplate(c *gin.Context) {
	var input TemplateInput
	if c.ShouldBindJSON(&input) != nil {
		invalid(c, "模板参数格式错误")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 200 || !validType(input.Type) || strings.TrimSpace(input.Content) == "" || len(input.Content) > 1024*1024 {
		invalid(c, "请填写模板名称、有效文档类型和正文（最多 1MB）")
		return
	}
	var id uint
	if c.Param("id") != "" {
		var ok bool
		id, ok = idParam(c)
		if !ok {
			return
		}
		if input.UpdatedAt.IsZero() {
			invalid(c, "缺少更新时间，请重新打开模板")
			return
		}
	}
	row, err := h.SaveTemplateRecord(id, input, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, row)
}

func (h *Handler) SaveTemplateRecord(id uint, input TemplateInput, user *uint) (*models.DocumentTemplate, error) {
	var row models.DocumentTemplate
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		action := "create"
		// Lock before reading a row so concurrent activations cannot deadlock.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('atlas-document-templates'))").Error; err != nil {
			return err
		}
		if id != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
				return err
			}
			if !row.UpdatedAt.Equal(input.UpdatedAt) {
				return errStale
			}
			action = "update"
		}
		if input.IsEnabled {
			var active []models.DocumentTemplate
			if err := tx.Where("type = ? AND is_enabled = true AND id <> ?", input.Type, id).Find(&active).Error; err != nil {
				return err
			}
			for _, old := range active {
				if err := tx.Model(&old).Updates(map[string]any{"is_enabled": false, "updated_by": user}).Error; err != nil {
					return err
				}
				if err := audit.Write(tx, user, "template", old.ID, old.Name, nil, "update", "启用同类型新模板，自动停用此模板", "启用状态"); err != nil {
					return err
				}
			}
		}
		row.Name = input.Name
		row.Type = input.Type
		row.Content = input.Content
		row.IsEnabled = input.IsEnabled
		row.UpdatedBy = user
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		// Return PostgreSQL's persisted timestamp precision for optimistic edits.
		if err := tx.First(&row, row.ID).Error; err != nil {
			return err
		}
		return audit.Write(tx, user, "template", row.ID, row.Name, nil, action, "维护文档模板", "名称", "文档类型", "正文", "启用状态")
	})
	return &row, err
}

func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var input TemplateInput
	if c.ShouldBindJSON(&input) != nil || input.UpdatedAt.IsZero() {
		invalid(c, "缺少更新时间，请刷新模板列表")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var row models.DocumentTemplate
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if !row.UpdatedAt.Equal(input.UpdatedAt) {
			return errStale
		}
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		return audit.Write(tx, actor(c), "template", id, row.Name, nil, "delete", "删除文档模板；已保存文档不受影响")
	})
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

type TrashItem struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	ObjectType    string    `json:"objectType"`
	ProjectID     *uint     `json:"projectId"`
	ProjectName   string    `json:"projectName"`
	DeletedBy     *uint     `json:"deletedBy"`
	DeletedByName string    `json:"deletedByName"`
	DeletedAt     time.Time `json:"deletedAt"`
	ParentDeleted bool      `json:"parentDeleted"`
}

const trashSQL = `SELECT p.id, p.name, 'project' AS object_type, p.id AS project_id, p.name AS project_name, p.deleted_by, p.deleted_at, false AS parent_deleted FROM projects p WHERE p.deleted_at IS NOT NULL
UNION ALL SELECT e.id, e.title, 'entry', e.project_id, COALESCE(p.name,''), e.deleted_by, e.deleted_at, (p.id IS NULL OR p.deleted_at IS NOT NULL) FROM project_entries e LEFT JOIN projects p ON p.id=e.project_id WHERE e.deleted_at IS NOT NULL
UNION ALL SELECT d.id, d.title, 'document', d.project_id, COALESCE(p.name,''), d.deleted_by, d.deleted_at, (d.project_id IS NOT NULL AND (p.id IS NULL OR p.deleted_at IS NOT NULL)) FROM project_documents d LEFT JOIN projects p ON p.id=d.project_id WHERE d.deleted_at IS NOT NULL
UNION ALL SELECT t.id, t.name, 'tool', NULL::bigint, '', t.deleted_by, t.deleted_at, false FROM public_links t WHERE t.deleted_at IS NOT NULL`

func (h *Handler) Trash(c *gin.Context) {
	p, s := page(c)
	rows := []TrashItem{}
	var total int64
	q := h.DB.Table("(" + trashSQL + ") AS trash").Joins("LEFT JOIN users u ON u.id=trash.deleted_by")
	if k := strings.TrimSpace(c.Query("keyword")); k != "" {
		q = q.Where("trash.name ILIKE ?", "%"+k+"%")
	}
	if t := c.Query("type"); t != "" {
		q = q.Where("trash.object_type = ?", t)
	}
	if err := q.Count(&total).Error; err != nil {
		fail(c, err)
		return
	}
	if err := q.Select("trash.*, COALESCE(NULLIF(u.display_name,''),NULLIF(u.username,''),'历史记录') AS deleted_by_name").Order("trash.deleted_at desc, trash.object_type, trash.id desc").Offset((p - 1) * s).Limit(s).Scan(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	list(c, rows, total, p, s)
}

func (h *Handler) Restore(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	kind := c.Param("type")
	if kind != "project" && kind != "entry" && kind != "document" && kind != "tool" {
		invalid(c, "不支持的对象类型")
		return
	}
	if err := lifecycle.Restore(h.DB, kind, id, actor(c)); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

func (h *Handler) Audits(c *gin.Context) {
	p, s := page(c)
	rows := []models.AuditLog{}
	var total int64
	q := h.DB.Model(&models.AuditLog{})
	if k := strings.TrimSpace(c.Query("keyword")); k != "" {
		like := "%" + k + "%"
		q = q.Where("(object_name ILIKE ? OR summary ILIKE ? OR CAST(object_id AS TEXT) = ?)", like, like, k)
	}
	if a := strings.TrimSpace(c.Query("actor")); a != "" {
		q = q.Where("(actor_name ILIKE ? OR CAST(actor_id AS TEXT) = ?)", "%"+a+"%", a)
	}
	if t := c.Query("type"); t != "" {
		q = q.Where("object_type = ?", t)
	}
	if a := c.Query("action"); a != "" {
		q = q.Where("action = ?", a)
	}
	var start, end *time.Time
	for _, key := range []string{"from", "to"} {
		if raw := c.Query(key); raw != "" {
			value, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				invalid(c, "时间格式错误")
				return
			}
			if key == "from" {
				start = &value
			} else {
				end = &value
			}
		}
	}
	if start != nil && end != nil && start.After(*end) {
		invalid(c, "开始时间不能晚于结束时间")
		return
	}
	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("created_at <= ?", *end)
	}
	if err := q.Count(&total).Error; err != nil {
		fail(c, err)
		return
	}
	if err := q.Order("created_at desc, id desc").Offset((p - 1) * s).Limit(s).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	list(c, rows, total, p, s)
}

func (h *Handler) AuditDetail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var row models.AuditLog
	if err := h.DB.First(&row, id).Error; err != nil {
		fail(c, err)
		return
	}
	response.OK(c, row)
}
