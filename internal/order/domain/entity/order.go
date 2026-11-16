package entity

import (
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"microservice/pkg/apperrors"
)

type OrderStatus string

const (
	OrderStatusInPreparation   OrderStatus = "en_preparacion"
	OrderStatusReadyToDispatch OrderStatus = "listo_para_despacho"
	OrderStatusInTransit       OrderStatus = "en_transito"
	OrderStatusDelivered       OrderStatus = "entregado"
	OrderStatusNotDelivered    OrderStatus = "no_entregado"
	OrderStatusReturned        OrderStatus = "devuelto"
	maxOrderAmount                         = 999_999_999.99
)

type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	Total      float64
	CreatedAt  time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	UnitPrice float64
	Subtotal  float64
}

func NewOrder(customerID string, items []OrderItem) (*Order, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, apperrors.NewValidationError("ORDER_CUSTOMER_REQUIRED", "customer ID is required")
	}

	if len(items) == 0 {
		return nil, apperrors.NewValidationError("ORDER_ITEMS_REQUIRED", "at least one item is required")
	}

	validatedItems := make([]OrderItem, len(items))
	var total float64

	for i, item := range items {
		if strings.TrimSpace(item.ProductID) == "" {
			return nil, apperrors.NewValidationError("ORDER_PRODUCT_REQUIRED", "product ID is required")
		}

		if item.Quantity <= 0 {
			return nil, apperrors.NewValidationError("ORDER_QUANTITY_INVALID", "quantity must be greater than zero")
		}

		if item.UnitPrice <= 0 || math.IsNaN(item.UnitPrice) || math.IsInf(item.UnitPrice, 0) {
			return nil, apperrors.NewValidationError("ORDER_UNIT_PRICE_INVALID", "unit price must be greater than zero")
		}

		item.UnitPrice = roundMoney(item.UnitPrice)
		if item.UnitPrice <= 0 {
			return nil, apperrors.NewValidationError("ORDER_UNIT_PRICE_INVALID", "unit price must be at least 0.01")
		}
		if item.UnitPrice > maxOrderAmount {
			return nil, apperrors.NewValidationError("ORDER_TOTAL_OVERFLOW", "order total exceeds the supported amount")
		}

		subtotal := roundMoney(item.UnitPrice * float64(item.Quantity))
		if math.IsInf(subtotal, 0) || subtotal > maxOrderAmount {
			return nil, apperrors.NewValidationError("ORDER_TOTAL_OVERFLOW", "order total exceeds the supported amount")
		}

		total = roundMoney(total + subtotal)
		if math.IsInf(total, 0) || total > maxOrderAmount {
			return nil, apperrors.NewValidationError("ORDER_TOTAL_OVERFLOW", "order total exceeds the supported amount")
		}

		item.Subtotal = subtotal
		validatedItems[i] = item
	}

	return &Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		Items:      validatedItems,
		Status:     OrderStatusInPreparation,
		Total:      total,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
