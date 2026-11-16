package container

import (
	"microservice/cmd/config"
	healthUseCase "microservice/internal/health/application/usecase"
	healthInfra "microservice/internal/health/infrastructure"
	healthHandler "microservice/internal/health/presentation/handler"
	orderUseCase "microservice/internal/order/application/usecase"
	orderInfra "microservice/internal/order/infrastructure"
	orderHandler "microservice/internal/order/presentation/handler"
	"microservice/pkg/database"
	"microservice/pkg/logger"
	"microservice/pkg/pubsub"

	"go.uber.org/zap"
)

type Container struct {
	DB            database.Database
	PubSub        pubsub.PubSub
	HealthHandler *healthHandler.HealthHandler
	OrderHandler  *orderHandler.OrderHandler
	Log           *zap.Logger
}

func NewContainer(cfg *config.Configuration, db database.Database, ps pubsub.PubSub, log *zap.Logger) (*Container, error) {

	repo := healthInfra.NewPostgresHealthRepository(db)
	healthUC := healthUseCase.NewHealthUseCase(repo)
	healthHdlr := healthHandler.NewHealthHandler(healthUC)

	orderRepo, err := orderInfra.NewPostgresOrderRepository(db)
	if err != nil {
		return nil, err
	}
	orderNotifier := orderInfra.NewLogOrderNotifier(log)
	createOrderUC := orderUseCase.NewCreateOrderUseCase(orderRepo, orderNotifier)
	getOrderUC := orderUseCase.NewGetOrderUseCase(orderRepo)
	orderHdlr := orderHandler.NewOrderHandler(createOrderUC, getOrderUC)

	logger.Success("Container de dependencias inicializado...")

	return &Container{
		DB:            db,
		PubSub:        ps,
		HealthHandler: healthHdlr,
		OrderHandler:  orderHdlr,
		Log:           log,
	}, nil
}
