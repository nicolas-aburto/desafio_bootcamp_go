package entity

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"

	"microservice/pkg/apperrors"
)

func TestNewOrder(t *testing.T) {
	items := []OrderItem{
		{ProductID: "product-1", Quantity: 2, UnitPrice: 15.99},
		{ProductID: "product-2", Quantity: 1, UnitPrice: 5.50},
	}

	order, err := NewOrder("customer-1", items)
	if err != nil {
		t.Fatalf("NewOrder() returned unexpected error: %v", err)
	}

	if _, err := uuid.Parse(order.ID); err != nil {
		t.Errorf("NewOrder().ID = %q, expected a valid UUID", order.ID)
	}

	if order.CustomerID != "customer-1" {
		t.Errorf("NewOrder().CustomerID = %q, want %q", order.CustomerID, "customer-1")
	}

	if order.Status != OrderStatusInPreparation {
		t.Errorf("NewOrder().Status = %q, want %q", order.Status, OrderStatusInPreparation)
	}

	if !amountsEqual(order.Total, 37.48) {
		t.Errorf("NewOrder().Total = %.2f, want %.2f", order.Total, 37.48)
	}

	if !amountsEqual(order.Items[0].Subtotal, 31.98) {
		t.Errorf("NewOrder().Items[0].Subtotal = %.2f, want %.2f", order.Items[0].Subtotal, 31.98)
	}

	if !amountsEqual(order.Items[1].Subtotal, 5.50) {
		t.Errorf("NewOrder().Items[1].Subtotal = %.2f, want %.2f", order.Items[1].Subtotal, 5.50)
	}

	if order.CreatedAt.IsZero() {
		t.Error("NewOrder().CreatedAt is zero, expected a valid timestamp")
	}
}

func TestNewOrderDoesNotModifyInputItems(t *testing.T) {
	items := []OrderItem{{ProductID: "product-1", Quantity: 2, UnitPrice: 15.99}}

	order, err := NewOrder("customer-1", items)
	if err != nil {
		t.Fatalf("NewOrder() returned unexpected error: %v", err)
	}

	if !amountsEqual(items[0].Subtotal, 0) {
		t.Errorf("input item subtotal = %.2f, want %.2f", items[0].Subtotal, 0.0)
	}

	if !amountsEqual(order.Items[0].Subtotal, 31.98) {
		t.Errorf("NewOrder().Items[0].Subtotal = %.2f, want %.2f", order.Items[0].Subtotal, 31.98)
	}
}

func TestNewOrderRoundsPricesToTwoDecimals(t *testing.T) {
	order, err := NewOrder("customer-1", []OrderItem{
		{ProductID: "product-1", Quantity: 2, UnitPrice: 10.999},
	})
	if err != nil {
		t.Fatalf("NewOrder() returned unexpected error: %v", err)
	}

	if !amountsEqual(order.Items[0].UnitPrice, 11.00) {
		t.Errorf("NewOrder().Items[0].UnitPrice = %.2f, want %.2f", order.Items[0].UnitPrice, 11.00)
	}

	if !amountsEqual(order.Total, 22.00) {
		t.Errorf("NewOrder().Total = %.2f, want %.2f", order.Total, 22.00)
	}
}

func TestNewOrderValidation(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		items      []OrderItem
		wantCode   string
	}{
		{
			name:       "empty customer",
			customerID: " ",
			items:      validItems(),
			wantCode:   "ORDER_CUSTOMER_REQUIRED",
		},
		{
			name:       "empty items",
			customerID: "customer-1",
			items:      nil,
			wantCode:   "ORDER_ITEMS_REQUIRED",
		},
		{
			name:       "empty product",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: " ", Quantity: 1, UnitPrice: 100}},
			wantCode:   "ORDER_PRODUCT_REQUIRED",
		},
		{
			name:       "zero quantity",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 0, UnitPrice: 100}},
			wantCode:   "ORDER_QUANTITY_INVALID",
		},
		{
			name:       "negative quantity",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: -1, UnitPrice: 100}},
			wantCode:   "ORDER_QUANTITY_INVALID",
		},
		{
			name:       "zero price",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: 0}},
			wantCode:   "ORDER_UNIT_PRICE_INVALID",
		},
		{
			name:       "negative price",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: -100}},
			wantCode:   "ORDER_UNIT_PRICE_INVALID",
		},
		{
			name:       "price rounds to zero",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: 0.001}},
			wantCode:   "ORDER_UNIT_PRICE_INVALID",
		},
		{
			name:       "subtotal overflow",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 2, UnitPrice: math.MaxFloat64}},
			wantCode:   "ORDER_TOTAL_OVERFLOW",
		},
		{
			name:       "total overflow",
			customerID: "customer-1",
			items: []OrderItem{
				{ProductID: "product-1", Quantity: 1, UnitPrice: math.MaxFloat64 / 2},
				{ProductID: "product-2", Quantity: 1, UnitPrice: math.MaxFloat64 / 2},
			},
			wantCode: "ORDER_TOTAL_OVERFLOW",
		},
		{
			name:       "NaN price",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: math.NaN()}},
			wantCode:   "ORDER_UNIT_PRICE_INVALID",
		},
		{
			name:       "infinite price",
			customerID: "customer-1",
			items:      []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: math.Inf(1)}},
			wantCode:   "ORDER_UNIT_PRICE_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := NewOrder(tt.customerID, tt.items)
			if err == nil {
				t.Fatal("NewOrder() error = nil, expected a validation error")
			}

			if order != nil {
				t.Errorf("NewOrder() order = %#v, want nil", order)
			}

			var domainErr *apperrors.DomainError
			if !errors.As(err, &domainErr) {
				t.Fatalf("NewOrder() error type = %T, want *apperrors.DomainError", err)
			}

			if domainErr.Type != apperrors.ErrorTypeValidation {
				t.Errorf("NewOrder() error type = %q, want %q", domainErr.Type, apperrors.ErrorTypeValidation)
			}

			if domainErr.Code != tt.wantCode {
				t.Errorf("NewOrder() error code = %q, want %q", domainErr.Code, tt.wantCode)
			}
		})
	}
}

func validItems() []OrderItem {
	return []OrderItem{{ProductID: "product-1", Quantity: 1, UnitPrice: 15.99}}
}

func amountsEqual(got, want float64) bool {
	return math.Abs(got-want) < 0.000001
}
