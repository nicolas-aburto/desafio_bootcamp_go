package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"microservice/internal/order/application/dto"
	"microservice/internal/order/application/usecase"
	"microservice/pkg/apperrors"
	"microservice/pkg/httputil"
)

type OrderHandler struct {
	createOrder *usecase.CreateOrderUseCase
	getOrder    *usecase.GetOrderUseCase
}

func NewOrderHandler(createOrder *usecase.CreateOrderUseCase, getOrder *usecase.GetOrderUseCase) *OrderHandler {
	return &OrderHandler{createOrder: createOrder, getOrder: getOrder}
}

// Create godoc
// @Summary Create an order
// @Description Validates and persists a new order
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body dto.CreateOrderRequest true "Order to create"
// @Success 201 {object} dto.OrderResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) Create(c *gin.Context) {
	var request dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, apperrors.NewValidationError("ORDER_REQUEST_INVALID", "request body must contain valid JSON"))
		return
	}

	response, err := h.createOrder.Execute(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetByID godoc
// @Summary Get an order
// @Description Returns an order by ID
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetByID(c *gin.Context) {
	response, err := h.getOrder.Execute(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func writeError(c *gin.Context, err error) {
	status, response := httputil.MapErrorToHttp(err)
	c.JSON(status, response)
}
