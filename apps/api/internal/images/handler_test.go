package images

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"ty-notebook-api/internal/config"
)

func TestImageProxy(t *testing.T) {
	var pngFile bytes.Buffer
	if err := png.Encode(&pngFile, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", md5.Sum(pngFile.Bytes()))
	for _, tc := range []struct {
		name        string
		hit         bool
		upstream    int
		badURL      bool
		want        int
		wantUploads int
	}{
		{"upload new image", false, 200, false, 200, 1},
		{"reuse image without forwarding file", true, 200, false, 200, 0},
		{"credentials failure does not expire Atlas session", false, 401, false, 503, 0},
		{"revoked membership", false, 403, false, 403, 0},
		{"service failure", false, 500, false, 502, 0},
		{"unsafe upstream URL", true, 200, true, 502, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uploads := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				client, secret, ok := r.BasicAuth()
				if !ok || client != "atlas" || secret != "fixture-secret" || r.Header.Get("X-Helios-User-ID") != "trusted-user" {
					t.Error("missing project/user authentication")
				}
				if r.Header.Get("Cookie") != "" {
					t.Error("forwarded browser cookies")
				}
				if tc.upstream != 200 {
					w.WriteHeader(tc.upstream)
					return
				}
				path := "/uploads/images/" + digest + ".png"
				if tc.badURL {
					path = "https://untrusted.example/image.png"
				}
				out := result{MD5: digest, URL: path, Exists: tc.hit, Size: int64(pngFile.Len()), MimeType: "image/png"}
				if r.URL.Path == "/api/v1/upload/image/check" {
					var in struct {
						MD5 string `json:"md5"`
					}
					_ = json.NewDecoder(r.Body).Decode(&in)
					if in.MD5 != digest {
						t.Error("wrong computed MD5")
					}
					if !tc.hit {
						out.URL = ""
					}
				} else if r.URL.Path == "/api/v1/upload/image" {
					uploads++
					file, _, err := r.FormFile("file")
					if err != nil {
						t.Error(err)
						return
					}
					defer file.Close()
					data, _ := io.ReadAll(file)
					if !bytes.Equal(data, pngFile.Bytes()) || r.FormValue("md5") != digest {
						t.Error("image bytes changed")
					}
				} else {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(gin.H{"code": 200, "data": out})
			}))
			defer upstream.Close()
			h := &Handler{Config: config.Config{HeliosClientID: "atlas", HeliosClientSecret: "fixture-secret", HeliosImageInternalURL: upstream.URL, HeliosImagePublicURL: "https://images.example"}, HTTP: upstream.Client()}
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("externalUserID", "trusted-user") })
			r.POST("/upload", h.Upload)
			r.POST("/check", h.Check)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, _ := writer.CreateFormFile("file", "image.png")
			_, _ = part.Write(pngFile.Bytes())
			_ = writer.WriteField("md5", digest)
			_ = writer.Close()
			req := httptest.NewRequest("POST", "/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-Helios-User-ID", "forged-user")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want || uploads != tc.wantUploads {
				t.Fatalf("status=%d uploads=%d: %s", w.Code, uploads, w.Body.String())
			}
			if tc.want == 200 {
				var out struct{ Data result }
				_ = json.Unmarshal(w.Body.Bytes(), &out)
				if out.Data.URL != "https://images.example/uploads/images/"+digest+".png" || out.Data.Reused != tc.hit {
					t.Fatalf("bad response: %s", w.Body.String())
				}
			}
		})
	}
}

func TestImageValidationBeforeProxy(t *testing.T) {
	h := &Handler{}
	r := gin.New()
	r.POST("/check", h.Check)
	r.POST("/upload", h.Upload)
	for _, body := range []string{`{"md5":"../bad"}`, `{}`, `not json`} {
		req := httptest.NewRequest("POST", "/check", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatal("invalid MD5 accepted")
		}
	}
	for _, data := range [][]byte{[]byte("<svg/>"), nil, make([]byte, 10*1024*1024+1)} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("file", "fake.png")
		_, _ = part.Write(data)
		_ = writer.Close()
		req := httptest.NewRequest("POST", "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("bad file accepted: %d", w.Code)
		}
	}
}
