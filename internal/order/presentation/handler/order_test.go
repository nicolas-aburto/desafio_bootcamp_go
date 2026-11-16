package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"microservice/internal/order/application/dto"
	"microservice/internal/order/application/usecase"
	"microservice/internal/order/domain/entity"
	orderHandler "microservice/internal/order/presentation/handler"
	orderRouter "microservice/internal/order/presentation/router"
	"microservice/pkg/apperrors"
	"microservice/pkg/httputil"
)

type memoryOrderRepository struct {
	mu        sync.RWMutex
	orders    map[string]*entity.Order
	createErr error
}

func newMemoryOrderRepository() *memoryOrderRepository {
	return &memoryOrderRepository{orders: make(map[string]*entity.Order)}
}

func (r *memoryOrderRepository) Create(_ context.Context, order *entity.Order) error {
	if r.createErr != nil {
		return r.createErr
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *memoryOrderRepository) FindByID(_ context.Context, id string) (*entity.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}

	return order, nil
}

func TestCreateAndGetOrder(t *testing.T) {
	repo := newMemoryOrderRepository()
	router := setupRouter(repo)
	body := []byte(`{
		"customer_id": "customer-1",
		"items": [
			{"product_id": "product-1", "quantity": 2, "unit_price": 15.99}
		]
	}`)

	createResponse := performRequest(router, http.MethodPost, "/api/v1/orders", body)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST /orders status = %d, want %d; body=%s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}

	var created dto.OrderResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	if created.Status != string(entity.OrderStatusInPreparation) {
		t.Errorf("created status = %q, want %q", created.Status, entity.OrderStatusInPreparation)
	}

	if created.Total != 31.98 {
		t.Errorf("created total = %.2f, want %.2f", created.Total, 31.98)
	}

	getResponse := performRequest(router, http.MethodGet, "/api/v1/orders/"+created.ID, nil)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /orders/:id status = %d, want %d; body=%s", getResponse.Code, http.StatusOK, getResponse.Body.String())
	}

	var found dto.OrderResponse
	if err := json.Unmarshal(getResponse.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode get response: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("found ID = %q, want %q", found.ID, created.ID)
	}
}

func TestCreateOrderValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{
			name:     "malformed JSON",
			body:     `{"customer_id":`,
			wantCode: "ORDER_REQUEST_INVALID",
		},
		{
			name:     "invalid quantity",
			body:     `{"customer_id":"customer-1","items":[{"product_id":"product-1","quantity":0,"unit_price":15.99}]}`,
			wantCode: "ORDER_QUANTITY_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(newMemoryOrderRepository())
			response := performRequest(router, http.MethodPost, "/api/v1/orders", []byte(tt.body))

			assertHTTPError(t, response, http.StatusBadRequest, tt.wantCode)
		})
	}
}

func TestGetOrderNotFound(t *testing.T) {
	router := setupRouter(newMemoryOrderRepository())
	response := performRequest(router, http.MethodGet, "/api/v1/orders/d4e12c89-e768-45e7-b13c-f30f2d290da8", nil)

	assertHTTPError(t, response, http.StatusNotFound, "ORDER_NOT_FOUND")
}

func TestCreateOrderHidesRepositoryError(t *testing.T) {
	repo := newMemoryOrderRepository()
	repo.createErr = errors.New("secret database detail")
	router := setupRouter(repo)
	body := []byte(`{"customer_id":"customer-1","items":[{"product_id":"product-1","quantity":1,"unit_price":15.99}]}`)

	response := performRequest(router, http.MethodPost, "/api/v1/orders", body)

	assertHTTPError(t, response, http.StatusInternalServerError, "INTERNAL_001")
	if bytes.Contains(response.Body.Bytes(), []byte("secret database detail")) {
		t.Error("HTTP response exposed an internal repository error")
	}
}

func setupRouter(repo *memoryOrderRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	createOrder := usecase.NewCreateOrderUseCase(repo, nil)
	getOrder := usecase.NewGetOrderUseCase(repo)
	handler := orderHandler.NewOrderHandler(createOrder, getOrder)
	router := gin.New()
	orderRouter.RegisterRoutes(router.Group("api"), handler)
	return router
}

func performRequest(router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertHTTPError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("HTTP status = %d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}

	var errorResponse httputil.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &errorResponse); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if errorResponse.Error.Code != wantCode {
		t.Errorf("error code = %q, want %q", errorResponse.Error.Code, wantCode)
	}
}
