package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservice/internal/order/domain/entity"
	domain "microservice/internal/order/domain/repository"
	"microservice/pkg/apperrors"
	"microservice/pkg/database"
)

type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

var _ domain.OrderRepository = (*PostgresOrderRepository)(nil)

func NewPostgresOrderRepository(db database.Database) (*PostgresOrderRepository, error) {
	pool, ok := db.Native().(*pgxpool.Pool)
	if !ok {
		return nil, fmt.Errorf("order repository requires a PostgreSQL connection")
	}

	return &PostgresOrderRepository{pool: pool}, nil
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create order transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertOrder = `
		INSERT INTO orders (id, customer_id, status, total, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	if _, err := tx.Exec(ctx, insertOrder,
		order.ID,
		order.CustomerID,
		string(order.Status),
		order.Total,
		order.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	const insertItem = `
		INSERT INTO order_items (order_id, position, product_id, quantity, unit_price, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6)`

	for position, item := range order.Items {
		if _, err := tx.Exec(ctx, insertItem,
			order.ID,
			position,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
		); err != nil {
			return fmt.Errorf("insert order item at position %d: %w", position, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create order transaction: %w", err)
	}

	return nil
}

func (r *PostgresOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	const selectOrder = `
		SELECT id, customer_id, status, total, created_at
		FROM orders
		WHERE id = $1`

	var order entity.Order
	var status string

	err := r.pool.QueryRow(ctx, selectOrder, id).Scan(
		&order.ID,
		&order.CustomerID,
		&status,
		&order.Total,
		&order.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewNotFoundError("ORDER_NOT_FOUND", "order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("find order: %w", err)
	}

	order.Status = entity.OrderStatus(status)

	const selectItems = `
		SELECT product_id, quantity, unit_price, subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY position`

	rows, err := r.pool.Query(ctx, selectItems, id)
	if err != nil {
		return nil, fmt.Errorf("find order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item entity.OrderItem
		if err := rows.Scan(
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return &order, nil
}
