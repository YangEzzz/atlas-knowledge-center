package handlers

import (
	"net/http"

	"ty-notebook-api/internal/api/response"
	"ty-notebook-api/internal/repository"
	"ty-notebook-api/internal/service"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	service *service.DocumentService
}

func NewDocumentHandler(service *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

func (h *DocumentHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)
	projectID := uint(queryInt(c, "projectId", 0))

	documents, total, err := h.service.List(repository.DocumentListFilter{
		Keyword:   c.Query("keyword"),
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		ProjectID: projectID,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取文档列表失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"list":     documents,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *DocumentHandler) ListDeleted(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)
	projectID := uint(queryInt(c, "projectId", 0))

	documents, total, err := h.service.ListDeleted(repository.DocumentListFilter{
		Keyword:   c.Query("keyword"),
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		ProjectID: projectID,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取回收站列表失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"list":     documents,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *DocumentHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	document, err := h.service.Get(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文档不存在")
		return
	}

	response.OK(c, document)
}

func (h *DocumentHandler) GetByProjectAndType(c *gin.Context) {
	projectID, ok := parseIDParam(c, "projectId")
	if !ok {
		return
	}
	docType := c.Param("type")
	if docType == "" {
		response.Error(c, http.StatusBadRequest, "文档类型不能为空")
		return
	}

	document, err := h.service.GetByProjectAndType(projectID, docType)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文档不存在")
		return
	}

	response.OK(c, document)
}

func (h *DocumentHandler) Save(c *gin.Context) {
	var input service.DocumentSaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	document, err := h.service.Save(input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "保存文档失败: "+err.Error())
		return
	}

	response.OK(c, document)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(id, currentUserID(c)); err != nil {
		response.Error(c, http.StatusNotFound, "文档不存在")
		return
	}

	response.OK(c, gin.H{
		"deleted": true,
	})
}

func (h *DocumentHandler) Restore(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	document, err := h.service.Restore(id, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "恢复文档失败: "+err.Error())
		return
	}

	response.OK(c, document)
}

func (h *DocumentHandler) ListVersions(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	versions, err := h.service.ListVersions(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文档不存在")
		return
	}

	response.OK(c, gin.H{
		"list": versions,
	})
}

func (h *DocumentHandler) GetVersion(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	versionID, ok := parseIDParam(c, "versionId")
	if !ok {
		return
	}

	version, err := h.service.GetVersion(id, versionID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文档版本不存在")
		return
	}

	response.OK(c, version)
}

func (h *DocumentHandler) RestoreVersion(c *gin.Context) {
	versionID, ok := parseIDParam(c, "versionId")
	if !ok {
		return
	}

	var input service.DocumentVersionRestoreInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	document, err := h.service.RestoreVersion(versionID, input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "恢复文档版本失败: "+err.Error())
		return
	}

	response.OK(c, document)
}

func (h *DocumentHandler) RestoreDocumentVersion(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	versionID, ok := parseIDParam(c, "versionId")
	if !ok {
		return
	}

	version, err := h.service.GetVersion(id, versionID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文档版本不存在")
		return
	}
	if version.DocumentID != id {
		response.Error(c, http.StatusBadRequest, "文档版本不属于当前文档")
		return
	}

	var input service.DocumentVersionRestoreInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	document, err := h.service.RestoreVersion(versionID, input, currentUserID(c))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "恢复文档版本失败: "+err.Error())
		return
	}

	response.OK(c, document)
}
