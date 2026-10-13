package handlers

import (
	"time"

	"ty-notebook-api/internal/api/response"

	"github.com/gin-gonic/gin"
)

type PingHandler struct{}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (h *PingHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{
		"message": "pong",
		"time":    time.Now().Format(time.RFC3339),
	})
}
