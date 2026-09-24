CREATE TABLE IF NOT EXISTS products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL,
    price       NUMERIC(12,2) NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Generate 100,000 test products
INSERT INTO products (name, description, price)
SELECT
    'Product ' || i,
    'Description for product ' || i,
    round((1 + random() * 999)::numeric, 2)
FROM generate_series(1, 100000) AS i;