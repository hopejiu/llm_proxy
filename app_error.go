package main

// AppError 统一错误类型，携带 code + message，前端可区分参数错误和内部错误
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

// 预定义错误
var (
	ErrBadRequest   = &AppError{Code: "BAD_REQUEST", Message: "请求参数错误"}
	ErrNotFound     = &AppError{Code: "NOT_FOUND", Message: "资源不存在"}
	ErrInternal     = &AppError{Code: "INTERNAL", Message: "内部错误"}
	ErrProxyRunning = &AppError{Code: "PROXY_RUNNING", Message: "代理服务正在运行"}
	ErrProxyStopped = &AppError{Code: "PROXY_STOPPED", Message: "代理服务未运行"}
)

func NewAppError(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}
