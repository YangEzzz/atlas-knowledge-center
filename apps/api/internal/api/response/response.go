package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    http.StatusOK,
		Message: "created",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Body{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Body{
		Code:    http.StatusUnauthorized,
		Message: message,
		Data:    nil,
	})
}
