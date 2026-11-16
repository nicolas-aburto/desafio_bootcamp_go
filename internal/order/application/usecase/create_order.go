package usecase

import (
	"context"

	"microservice/internal/order/application/dto"
	"microservice/internal/order/domain/entity"
	domain "microservice/internal/order/domain/repository"
)

type CreateOrderUseCase struct {
	repo     domain.OrderRepository
	notifier OrderNotifier
}

type OrderNotifier interface {
	Notify(ctx context.Context, orderID string)
}

func NewCreateOrderUseCase(repo domain.OrderRepository, notifier OrderNotifier) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo, notifier: notifier}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, request dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	items := make([]entity.OrderItem, len(request.Items))
	for i, item := range request.Items {
		items[i] = entity.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	order, err := entity.NewOrder(request.CustomerID, items)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	if uc.notifier != nil {
		go uc.notifier.Notify(context.WithoutCancel(ctx), order.ID)
	}

	return dto.NewOrderResponse(order), nil
}
