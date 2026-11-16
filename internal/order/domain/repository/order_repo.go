package domain

import (
	"context"

	"microservice/internal/order/domain/entity"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, id string) (*entity.Order, error)
}
