package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"microservice/internal/order/application/dto"
	"microservice/internal/order/domain/entity"
	"microservice/pkg/apperrors"
)

type mockOrderRepository struct {
	createFn   func(ctx context.Context, order *entity.Order) error
	findByIDFn func(ctx context.Context, id string) (*entity.Order, error)
}

type channelOrderNotifier struct {
	orderIDs chan string
}

func (n *channelOrderNotifier) Notify(_ context.Context, orderID string) {
	n.orderIDs <- orderID
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	if m.createFn == nil {
		return nil
	}

	return m.createFn(ctx, order)
}

func (m *mockOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	if m.findByIDFn == nil {
		return nil, nil
	}

	return m.findByIDFn(ctx, id)
}

func TestCreateOrderUseCaseExecute(t *testing.T) {
	var savedOrder *entity.Order
	repo := &mockOrderRepository{
		createFn: func(_ context.Context, order *entity.Order) error {
			savedOrder = order
			return nil
		},
	}
	uc := NewCreateOrderUseCase(repo, nil)
	request := dto.CreateOrderRequest{
		CustomerID: "customer-1",
		Items: []dto.CreateOrderItemRequest{
			{ProductID: "product-1", Quantity: 2, UnitPrice: 15.99},
			{ProductID: "product-2", Quantity: 1, UnitPrice: 5.50},
		},
	}

	response, err := uc.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	if savedOrder == nil {
		t.Fatal("Execute() did not send the order to the repository")
	}

	if response.ID != savedOrder.ID {
		t.Errorf("Execute().ID = %q, want %q", response.ID, savedOrder.ID)
	}

	if response.CustomerID != request.CustomerID {
		t.Errorf("Execute().CustomerID = %q, want %q", response.CustomerID, request.CustomerID)
	}

	if response.Status != string(entity.OrderStatusInPreparation) {
		t.Errorf("Execute().Status = %q, want %q", response.Status, entity.OrderStatusInPreparation)
	}

	if !amountsEqual(response.Total, 37.48) {
		t.Errorf("Execute().Total = %.2f, want %.2f", response.Total, 37.48)
	}

	if len(response.Items) != 2 {
		t.Fatalf("Execute().Items length = %d, want %d", len(response.Items), 2)
	}

	if !amountsEqual(response.Items[0].Subtotal, 31.98) {
		t.Errorf("Execute().Items[0].Subtotal = %.2f, want %.2f", response.Items[0].Subtotal, 31.98)
	}
}

func TestCreateOrderUseCaseDoesNotPersistInvalidOrder(t *testing.T) {
	createCalled := false
	repo := &mockOrderRepository{
		createFn: func(_ context.Context, _ *entity.Order) error {
			createCalled = true
			return nil
		},
	}
	uc := NewCreateOrderUseCase(repo, nil)
	request := dto.CreateOrderRequest{
		CustomerID: "",
		Items: []dto.CreateOrderItemRequest{
			{ProductID: "product-1", Quantity: 1, UnitPrice: 15.99},
		},
	}

	response, err := uc.Execute(context.Background(), request)
	if err == nil {
		t.Fatal("Execute() error = nil, expected a validation error")
	}

	if response != nil {
		t.Errorf("Execute() response = %#v, want nil", response)
	}

	if createCalled {
		t.Error("Execute() called the repository for an invalid order")
	}

	assertDomainError(t, err, apperrors.ErrorTypeValidation, "ORDER_CUSTOMER_REQUIRED")
}

func TestCreateOrderUseCaseReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	notifier := &channelOrderNotifier{orderIDs: make(chan string, 1)}
	repo := &mockOrderRepository{
		createFn: func(_ context.Context, _ *entity.Order) error {
			return wantErr
		},
	}
	uc := NewCreateOrderUseCase(repo, notifier)
	request := dto.CreateOrderRequest{
		CustomerID: "customer-1",
		Items: []dto.CreateOrderItemRequest{
			{ProductID: "product-1", Quantity: 1, UnitPrice: 15.99},
		},
	}

	response, err := uc.Execute(context.Background(), request)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}

	if response != nil {
		t.Errorf("Execute() response = %#v, want nil", response)
	}

	select {
	case orderID := <-notifier.orderIDs:
		t.Errorf("Notify() was called for order %q after persistence failed", orderID)
	default:
	}
}

func TestCreateOrderUseCaseDispatchesNotification(t *testing.T) {
	repo := &mockOrderRepository{}
	notifier := &channelOrderNotifier{orderIDs: make(chan string, 1)}
	uc := NewCreateOrderUseCase(repo, notifier)
	request := dto.CreateOrderRequest{
		CustomerID: "customer-1",
		Items: []dto.CreateOrderItemRequest{
			{ProductID: "product-1", Quantity: 1, UnitPrice: 15.99},
		},
	}

	response, err := uc.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	select {
	case orderID := <-notifier.orderIDs:
		if orderID != response.ID {
			t.Errorf("Notify() order ID = %q, want %q", orderID, response.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("Notify() was not called")
	}
}

func TestGetOrderUseCaseExecute(t *testing.T) {
	order, err := entity.NewOrder("customer-1", []entity.OrderItem{
		{ProductID: "product-1", Quantity: 2, UnitPrice: 15.99},
	})
	if err != nil {
		t.Fatalf("NewOrder() returned unexpected error: %v", err)
	}

	repo := &mockOrderRepository{
		findByIDFn: func(_ context.Context, id string) (*entity.Order, error) {
			if id != order.ID {
				t.Errorf("FindByID() id = %q, want %q", id, order.ID)
			}
			return order, nil
		},
	}
	uc := NewGetOrderUseCase(repo)

	response, err := uc.Execute(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	if response.ID != order.ID {
		t.Errorf("Execute().ID = %q, want %q", response.ID, order.ID)
	}

	if !amountsEqual(response.Total, 31.98) {
		t.Errorf("Execute().Total = %.2f, want %.2f", response.Total, 31.98)
	}
}

func TestGetOrderUseCaseRejectsInvalidID(t *testing.T) {
	findCalled := false
	repo := &mockOrderRepository{
		findByIDFn: func(_ context.Context, _ string) (*entity.Order, error) {
			findCalled = true
			return nil, nil
		},
	}
	uc := NewGetOrderUseCase(repo)

	response, err := uc.Execute(context.Background(), "invalid-id")
	if err == nil {
		t.Fatal("Execute() error = nil, expected a validation error")
	}

	if response != nil {
		t.Errorf("Execute() response = %#v, want nil", response)
	}

	if findCalled {
		t.Error("Execute() called the repository with an invalid ID")
	}

	assertDomainError(t, err, apperrors.ErrorTypeValidation, "ORDER_ID_INVALID")
}

func TestGetOrderUseCaseReturnsNotFoundError(t *testing.T) {
	wantErr := apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	repo := &mockOrderRepository{
		findByIDFn: func(_ context.Context, _ string) (*entity.Order, error) {
			return nil, wantErr
		},
	}
	uc := NewGetOrderUseCase(repo)

	response, err := uc.Execute(context.Background(), "d4e12c89-e768-45e7-b13c-f30f2d290da8")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}

	if response != nil {
		t.Errorf("Execute() response = %#v, want nil", response)
	}
}

func assertDomainError(t *testing.T, err error, wantType apperrors.ErrorType, wantCode string) {
	t.Helper()

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want *apperrors.DomainError", err)
	}

	if domainErr.Type != wantType {
		t.Errorf("error type = %q, want %q", domainErr.Type, wantType)
	}

	if domainErr.Code != wantCode {
		t.Errorf("error code = %q, want %q", domainErr.Code, wantCode)
	}
}

func amountsEqual(got, want float64) bool {
	const tolerance = 0.000001

	difference := got - want
	if difference < 0 {
		difference = -difference
	}

	return difference < tolerance
}
