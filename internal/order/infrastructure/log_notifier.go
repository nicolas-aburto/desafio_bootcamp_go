package infrastructure

import (
	"context"

	"go.uber.org/zap"
)

type LogOrderNotifier struct {
	log *zap.Logger
}

func NewLogOrderNotifier(log *zap.Logger) *LogOrderNotifier {
	return &LogOrderNotifier{log: log}
}

func (n *LogOrderNotifier) Notify(_ context.Context, orderID string) {
	n.log.Info("order notification simulated", zap.String("order_id", orderID))
}
