package router

import (
	"github.com/gin-gonic/gin"

	"microservice/internal/order/presentation/handler"
)

func RegisterRoutes(api *gin.RouterGroup, h *handler.OrderHandler) {
	orders := api.Group("/v1/orders")
	orders.POST("", h.Create)
	orders.GET("/:id", h.GetByID)
}
