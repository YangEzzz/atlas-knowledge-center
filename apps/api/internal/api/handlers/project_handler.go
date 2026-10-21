package handlers

import (
	"net/http"
	"strconv"

	"ty-notebook-api/internal/api/response"
	"ty-notebook-api/internal/repository"
	"ty-notebook-api/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)

	projects, total, err := h.service.List(repository.ProjectListFilter{
		Keyword:  c.Query("keyword"),
		Type:     c.Query("type"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取项目列表失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"list":     projects,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	project, err := h.service.Get(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "项目不存在")
		return
	}

	response.OK(c, project)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var input service.ProjectSaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if input.Name == "" || input.Type == "" {
		response.Error(c, http.StatusBadRequest, "项目名称和项目形态不能为空")
		return
	}

	project, err := h.service.Create(input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建项目失败: "+err.Error())
		return
	}

	response.Created(c, project)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var input service.ProjectSaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if input.Name == "" || input.Type == "" {
		response.Error(c, http.StatusBadRequest, "项目名称和项目形态不能为空")
		return
	}

	project, err := h.service.Update(id, input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新项目失败: "+err.Error())
		return
	}

	response.OK(c, project)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(id, currentUserID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除项目失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"id": id})
}

func (h *ProjectHandler) Restore(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Restore(id, currentUserID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "恢复项目失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"id": id})
}

func parseIDParam(c *gin.Context, key string) (uint, bool) {
	value := c.Param(key)
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "无效的 ID")
		return 0, false
	}
	return uint(id), true
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func queryBoolPtr(c *gin.Context, key string) *bool {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}

func currentUserID(c *gin.Context) *uint {
	value, ok := c.Get("userID")
	if !ok {
		return nil
	}
	userID, ok := value.(uint)
	if !ok {
		return nil
	}
	return &userID
}
