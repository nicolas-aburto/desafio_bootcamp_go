package dto

import (
	"time"

	"microservice/internal/order/domain/entity"
)

type OrderResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Items      []OrderItemResponse `json:"items"`
	Status     string              `json:"status"`
	Total      float64             `json:"total"`
	CreatedAt  time.Time           `json:"created_at"`
}

type OrderItemResponse struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

func NewOrderResponse(order *entity.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.Subtotal,
		}
	}

	return &OrderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Items:      items,
		Status:     string(order.Status),
		Total:      order.Total,
		CreatedAt:  order.CreatedAt,
	}
}
