package usecase

import (
	"context"

	"github.com/google/uuid"

	"microservice/internal/order/application/dto"
	domain "microservice/internal/order/domain/repository"
	"microservice/pkg/apperrors"
)

type GetOrderUseCase struct {
	repo domain.OrderRepository
}

func NewGetOrderUseCase(repo domain.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{repo: repo}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, id string) (*dto.OrderResponse, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, apperrors.NewValidationError("ORDER_ID_INVALID", "order ID must be a valid UUID")
	}

	order, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return dto.NewOrderResponse(order), nil
}
