package api

import (
	"net/http"

	"go-whois/internal/engine"
	"go-whois/internal/errors"
	"go-whois/internal/model"
	"go-whois/internal/service"
	"go-whois/pkg/validator"

	"github.com/gin-gonic/gin"
)

// Handler 定义 HTTP 请求处理器
type Handler struct {
	lookupService service.LookupService
}

// NewHandler 创建新的处理器实例
func NewHandler(lookupService service.LookupService) *Handler {
	return &Handler{
		lookupService: lookupService,
	}
}

// Lookup 单域名查询处理
func (h *Handler) Lookup(c *gin.Context) {
	domain := c.Param("domain")
	protocol := c.DefaultQuery("protocol", "auto")
	if protocol == "" {
		protocol = "auto"
	}
	// 验证域名
	if err := validator.ValidateDomain(domain); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    string(errors.ErrCodeInvalidDomain),
				Message: "域名格式无效",
				Details: err.Error(),
			},
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// 创建查询请求
	req := &engine.QueryRequest{
		Domain:   domain,
		Protocol: engine.Protocol(protocol),
	}

	// 执行查询
	result, err := h.lookupService.Lookup(c.Request.Context(), req)
	if err != nil {
		// 处理应用错误
		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.HTTPStatus, model.APIResponse{
				Success: false,
				Error: &model.APIError{
					Code:    string(appErr.Code),
					Message: appErr.Message,
					Details: appErr.Details,
				},
				RequestID: c.GetString("request_id"),
			})
			return
		}

		// 处理其他错误
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    string(errors.ErrCodeInternalError),
				Message: "内部服务器错误",
			},
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Success:   true,
		Data:      result,
		RequestID: c.GetString("request_id"),
	})
}

// HealthCheck 健康检查处理
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	})
}
