package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"microservice/internal/order/domain/entity"
)

func TestPostgresOrderRepositoryCreateAndFindByID(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	repo := &PostgresOrderRepository{pool: pool}
	order, err := entity.NewOrder("integration-customer", []entity.OrderItem{
		{ProductID: "product-1", Quantity: 2, UnitPrice: 15.99},
	})
	if err != nil {
		t.Fatalf("create order entity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", order.ID)
	})

	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	found, err := repo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindByID() returned error: %v", err)
	}

	if found.ID != order.ID {
		t.Errorf("FindByID().ID = %q, want %q", found.ID, order.ID)
	}

	if found.Total != 31.98 {
		t.Errorf("FindByID().Total = %.2f, want %.2f", found.Total, 31.98)
	}

	if len(found.Items) != 1 {
		t.Fatalf("FindByID().Items length = %d, want %d", len(found.Items), 1)
	}

	if found.Items[0].Subtotal != 31.98 {
		t.Errorf("FindByID().Items[0].Subtotal = %.2f, want %.2f", found.Items[0].Subtotal, 31.98)
	}
}
