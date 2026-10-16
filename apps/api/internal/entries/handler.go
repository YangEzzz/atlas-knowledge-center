package entries

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"ty-notebook-api/internal/accounts"
)

func Register(api *gin.RouterGroup, db *gorm.DB, crypt *accounts.Crypt) {
	g := api.Group("/my/entries")
	g.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, private")
		c.Header("Referrer-Policy", "no-referrer")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
		if c.GetUint("userID") == 0 {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "请先登录"})
			return
		}
		if crypt == nil {
			c.AbortWithStatusJSON(503, gin.H{"code": 503, "message": "私人入口加密配置尚未就绪"})
			return
		}
		c.Next()
	})
	s := New(db, crypt)
	respond := func(c *gin.Context, data any, err error) {
		if err == nil {
			c.JSON(200, gin.H{"code": 200, "data": data})
			return
		}
		status, msg := 500, "操作失败，请稍后重试"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status, msg = 404, "入口不存在、已删除或无权访问"
		}
		if errors.Is(err, ErrInvalid) {
			status, msg = 400, "请检查名称、网址和入口类型"
		}
		if errors.Is(err, ErrConflict) {
			status, msg = 409, "入口已更新，请刷新后重试"
		}
		c.JSON(status, gin.H{"code": status, "message": msg})
	}
	id := func(c *gin.Context) uint {
		n, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil || n == 0 {
			respond(c, nil, ErrInvalid)
			return 0
		}
		return uint(n)
	}
	g.GET("", func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "24"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 24
		}
		items, total, err := s.List(c.GetUint("userID"), c.Query("keyword"), c.Query("source"), c.Query("group"), c.Query("trash") == "true", page, size)
		respond(c, gin.H{"list": items, "total": total}, err)
	})
	g.POST("/import", func(c *gin.Context) {
		var in struct {
			Items []Input `json:"items"`
		}
		if c.ShouldBindJSON(&in) != nil {
			respond(c, nil, ErrInvalid)
			return
		}
		respond(c, gin.H{}, s.Import(c.GetUint("userID"), in.Items))
	})
	g.GET("/:id", func(c *gin.Context) {
		n := id(c)
		if n == 0 {
			return
		}
		v, err := s.Get(c.GetUint("userID"), n)
		respond(c, v, err)
	})
	save := func(c *gin.Context, n uint) {
		var in Input
		if c.ShouldBindJSON(&in) != nil {
			respond(c, nil, ErrInvalid)
			return
		}
		v, err := s.Save(c.GetUint("userID"), n, in)
		respond(c, v, err)
	}
	g.POST("", func(c *gin.Context) { save(c, 0) })
	g.PUT("/:id", func(c *gin.Context) {
		if n := id(c); n > 0 {
			save(c, n)
		}
	})
	g.DELETE("/:id", func(c *gin.Context) {
		n := id(c)
		if n == 0 {
			return
		}
		var in struct {
			Revision uint `json:"revision"`
		}
		if c.ShouldBindJSON(&in) != nil {
			respond(c, nil, ErrInvalid)
			return
		}
		respond(c, gin.H{}, s.Delete(c.GetUint("userID"), n, in.Revision))
	})
	g.POST("/:id/restore", func(c *gin.Context) {
		if n := id(c); n > 0 {
			respond(c, gin.H{}, s.Restore(c.GetUint("userID"), n))
		}
	})
}
