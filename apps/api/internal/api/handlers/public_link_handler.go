package handlers

import (
	"errors"
	"net/http"

	"ty-notebook-api/internal/api/response"
	"ty-notebook-api/internal/repository"
	"ty-notebook-api/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PublicLinkHandler struct {
	service *service.PublicLinkService
}

func NewPublicLinkHandler(service *service.PublicLinkService) *PublicLinkHandler {
	return &PublicLinkHandler{service: service}
}

func (h *PublicLinkHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	link, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "工具不存在或已删除")
		} else {
			response.Error(c, http.StatusInternalServerError, "工具暂时无法加载，请重试")
		}
		return
	}
	response.OK(c, link)
}

func (h *PublicLinkHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)

	links, total, err := h.service.List(repository.PublicLinkListFilter{
		Keyword:  c.Query("keyword"),
		Type:     c.Query("type"),
		Enabled:  queryBoolPtr(c, "enabled"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取工具导航列表失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"list":     links,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *PublicLinkHandler) Create(c *gin.Context) {
	var input service.PublicLinkSaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	link, err := h.service.Create(input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "创建工具导航失败: "+err.Error())
		return
	}

	response.Created(c, link)
}

func (h *PublicLinkHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var input service.PublicLinkSaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	link, err := h.service.Update(id, input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "更新工具导航失败: "+err.Error())
		return
	}

	response.OK(c, link)
}

func (h *PublicLinkHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(id, currentUserID(c)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除工具导航失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"id": id})
}

func (h *PublicLinkHandler) Restore(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Restore(id); err != nil {
		response.Error(c, http.StatusInternalServerError, "恢复工具导航失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"id": id})
}
