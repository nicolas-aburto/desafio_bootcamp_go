CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (
        status IN (
            'en_preparacion',
            'listo_para_despacho',
            'en_transito',
            'entregado',
            'no_entregado',
            'devuelto'
        )
    ),
    total NUMERIC(20, 2) NOT NULL CHECK (total > 0),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    product_id VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(20, 2) NOT NULL CHECK (unit_price > 0),
    subtotal NUMERIC(20, 2) NOT NULL CHECK (subtotal > 0),
    PRIMARY KEY (order_id, position)
);
