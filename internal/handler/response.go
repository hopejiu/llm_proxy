package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError 统一的 API 错误响应格式
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// respondError 返回统一格式的错误响应
func respondError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": APIError{Code: code, Message: message},
	})
}

// respondOK 返回成功响应（数据直接返回，不包装，保持前端兼容）
func respondOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// respondCreated 返回 201 Created 响应
func respondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// 错误码常量
const (
	ErrBadRequest   = "bad_request"
	ErrNotFound     = "not_found"
	ErrInternal     = "internal_error"
	ErrInvalidInput = "invalid_input"
)
