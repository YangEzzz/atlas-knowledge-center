package accounts

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"ty-notebook-api/internal/api/response"
)

func Register(api *gin.RouterGroup, db *gorm.DB, crypt *Crypt, revealPolicy string) {
	g := api.Group("/my/accounts")
	g.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 65536)
		c.Header("Cache-Control", "no-store, private")
		c.Header("Pragma", "no-cache")
		c.Header("Referrer-Policy", "no-referrer")
		if c.GetUint("userID") == 0 {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "请先登录"})
			return
		}
		if crypt == nil {
			c.AbortWithStatusJSON(503, gin.H{"code": 503, "message": "个人账号加密配置尚未就绪"})
			return
		}
		c.Next()
	})
	s := NewService(db, crypt)
	handle := func(c *gin.Context, err error) bool {
		if err == nil {
			return true
		}
		status := 500
		message := "操作失败，请稍后重试"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = 404
			message = "账号或所属项目、工具、私人入口不存在"
		}
		if errors.Is(err, ErrConflict) {
			status = 409
			message = "账号已更新，请关闭编辑并刷新后重试"
		}
		if errors.Is(err, ErrInvalid) {
			status = 400
			message = "请检查账号字段及关联入口"
		}
		c.JSON(status, response.Body{Code: status, Message: message})
		return false
	}
	id := func(c *gin.Context) uint {
		n, e := strconv.ParseUint(c.Param("id"), 10, 32)
		if e != nil || n == 0 {
			c.JSON(400, response.Body{Code: 400, Message: "无效的账号 ID"})
			return 0
		}
		return uint(n)
	}
	g.GET("", func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "12"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 50 {
			size = 12
		}
		project, _ := strconv.ParseUint(c.Query("projectId"), 10, 32)
		tool, toolErr := strconv.ParseUint(c.DefaultQuery("toolId", "0"), 10, 32)
		entry, entryErr := strconv.ParseUint(c.DefaultQuery("entryId", "0"), 10, 32)
		if toolErr != nil || entryErr != nil || (tool > 0 && project > 0) || (entry > 0 && (tool > 0 || project > 0)) {
			c.JSON(400, response.Body{Code: 400, Message: "请仅选择一个项目或工具"})
			return
		}
		items, total, err := s.List(c.GetUint("userID"), uint(project), c.Query("keyword"), c.Query("environment"), c.Query("trash") == "true", page, size, uint(tool), uint(entry))
		if handle(c, err) {
			response.OK(c, gin.H{"list": items, "total": total, "page": page, "pageSize": size})
		}
	})
	g.GET("/:id", func(c *gin.Context) {
		n := id(c)
		if n == 0 {
			return
		}
		a, note, err := s.Detail(c.GetUint("userID"), n)
		if handle(c, err) {
			response.OK(c, gin.H{"account": a, "note": note})
		}
	})
	save := func(c *gin.Context, n uint) {
		var input Input
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, response.Body{Code: 400, Message: "请检查必填字段和内容长度"})
			return
		}
		a, err := s.Save(c.GetUint("userID"), n, input)
		if handle(c, err) {
			response.OK(c, a)
		}
	}
	g.POST("", func(c *gin.Context) { save(c, 0) })
	g.PUT("/:id", func(c *gin.Context) {
		n := id(c)
		if n > 0 {
			save(c, n)
		}
	})
	g.POST("/:id/password", func(c *gin.Context) {
		n := id(c)
		if n == 0 {
			return
		}
		if revealPolicy != "session" {
			c.JSON(503, response.Body{Code: 503, Message: "密码查看策略尚未启用"})
			return
		}
		password, err := s.Password(c.GetUint("userID"), n)
		if handle(c, err) {
			response.OK(c, gin.H{"password": password})
		}
	})
	g.DELETE("/:id", func(c *gin.Context) {
		n := id(c)
		if n == 0 {
			return
		}
		var input struct {
			Revision uint `json:"revision" binding:"required"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, response.Body{Code: 400, Message: "请刷新账号后重试"})
			return
		}
		if handle(c, s.Delete(c.GetUint("userID"), n, input.Revision)) {
			response.OK(c, gin.H{})
		}
	})
	g.POST("/:id/restore", func(c *gin.Context) {
		n := id(c)
		if n > 0 && handle(c, s.Restore(c.GetUint("userID"), n)) {
			response.OK(c, gin.H{})
		}
	})
}
