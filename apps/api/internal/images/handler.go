package images

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ty-notebook-api/internal/config"
)

const maxImageSize = 10 * 1024 * 1024

type Handler struct {
	Config config.Config
	HTTP   *http.Client
}

type result struct {
	Exists   bool   `json:"exists"`
	URL      string `json:"url,omitempty"`
	MD5      string `json:"md5"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Reused   bool   `json:"reused"`
}

func Register(api *gin.RouterGroup, cfg config.Config) {
	h := &Handler{cfg, &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	api.POST("/images/check", h.Check)
	api.POST("/images/upload", h.Upload)
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}

func (h *Handler) remote(c *gin.Context, path, contentType string, body io.Reader) (result, bool) {
	var out result
	if h.Config.HeliosClientID == "" || h.Config.HeliosClientSecret == "" || c.GetString("externalUserID") == "" {
		fail(c, 503, "图片上传服务尚未配置，请联系管理员")
		return out, false
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", strings.TrimRight(h.Config.HeliosImageInternalURL, "/")+path, body)
	if err != nil {
		fail(c, 503, "图片服务地址无效")
		return out, false
	}
	req.SetBasicAuth(h.Config.HeliosClientID, h.Config.HeliosClientSecret)
	req.Header.Set("X-Helios-User-ID", c.GetString("externalUserID"))
	req.Header.Set("Content-Type", contentType)
	resp, err := h.HTTP.Do(req)
	if err != nil {
		fail(c, 502, "图片服务暂时不可用，请重试")
		return out, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		status, message := 502, "图片服务暂时不可用，请重试"
		if resp.StatusCode == 400 {
			status, message = 400, "图片参数或格式无效"
		}
		if resp.StatusCode == 403 {
			status, message = 403, "当前用户没有图片上传权限"
		}
		if resp.StatusCode == 401 {
			status, message = 503, "图片服务项目凭据无效，请联系管理员"
		}
		fail(c, status, message)
		return out, false
	}
	var envelope struct {
		Code int    `json:"code"`
		Data result `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16*1024)).Decode(&envelope); err != nil || envelope.Code != 200 {
		fail(c, 502, "图片服务返回异常，请重试")
		return out, false
	}
	out = envelope.Data
	if out.URL != "" {
		// Only accept public image paths from the configured service, never a
		// URL/host supplied by the browser or an arbitrary upstream redirect.
		u, err := url.Parse(out.URL)
		if err != nil || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/uploads/images/") || strings.Contains(u.Path, "..") {
			fail(c, 502, "图片服务返回地址无效")
			return result{}, false
		}
		base, err := url.Parse(h.Config.HeliosImagePublicURL)
		if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
			fail(c, 503, "图片公共地址尚未配置")
			return result{}, false
		}
		out.URL = base.ResolveReference(u).String()
	}
	return out, true
}

func validMD5(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == md5.Size
}

func (h *Handler) check(c *gin.Context, digest string) (result, bool) {
	body, _ := json.Marshal(gin.H{"md5": digest})
	out, ok := h.remote(c, "/api/v1/upload/image/check", "application/json", bytes.NewReader(body))
	if ok && (out.MD5 != digest || (out.Exists && out.URL == "")) {
		fail(c, 502, "图片服务校验结果异常")
		return result{}, false
	}
	return out, ok
}

func (h *Handler) Check(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var in struct {
		MD5 string `json:"md5"`
	}
	if c.ShouldBindJSON(&in) != nil || !validMD5(in.MD5) {
		fail(c, 400, "请提供有效的图片 MD5")
		return
	}
	if out, ok := h.check(c, strings.ToLower(in.MD5)); ok {
		c.JSON(200, gin.H{"code": 200, "message": "操作成功", "data": out})
	}
}

func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageSize+64*1024)
	err := c.Request.ParseMultipartForm(maxImageSize + 64*1024)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		fail(c, 400, "图片不能超过 10MB，请重新选择")
		return
	}
	file, err := c.FormFile("file")
	if err != nil || file.Size <= 0 || file.Size > maxImageSize {
		fail(c, 400, "请选择不超过 10MB 的图片")
		return
	}
	src, err := file.Open()
	if err != nil {
		fail(c, 400, "无法读取图片")
		return
	}
	defer src.Close()
	content, err := io.ReadAll(io.LimitReader(src, maxImageSize+1))
	if err != nil || len(content) != int(file.Size) {
		fail(c, 400, "图片读取失败")
		return
	}
	typeExt := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}
	ext, ok := typeExt[http.DetectContentType(content)]
	original := strings.ToLower(filepath.Ext(file.Filename))
	if !ok || (original != ext && !(ext == ".jpg" && original == ".jpeg")) {
		fail(c, 400, "仅支持 JPG、PNG、GIF、WebP 图片，扩展名须与内容一致")
		return
	}
	digest := fmt.Sprintf("%x", md5.Sum(content))
	if supplied := c.PostForm("md5"); supplied != "" && !strings.EqualFold(supplied, digest) {
		fail(c, 400, "图片 MD5 校验失败，请重新选择")
		return
	}
	if out, ok := h.check(c, digest); !ok {
		return
	} else if out.Exists {
		out.Reused = true
		c.JSON(200, gin.H{"code": 200, "message": "已复用图片", "data": out})
		return
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "image"+ext)
	if err != nil {
		fail(c, 500, "图片请求创建失败")
		return
	}
	if _, err = part.Write(content); err != nil {
		fail(c, 500, "图片请求创建失败")
		return
	}
	_ = w.WriteField("md5", digest)
	_ = w.Close()
	out, ok := h.remote(c, "/api/v1/upload/image", w.FormDataContentType(), &body)
	if !ok {
		return
	}
	if out.URL == "" || out.MD5 != digest {
		fail(c, 502, "图片服务上传结果异常")
		return
	}
	c.JSON(200, gin.H{"code": 200, "message": "上传成功", "data": out})
}
