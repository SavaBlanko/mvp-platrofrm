package product

import (
	"context"
	"errors"
	"log"
	"time"

	"cache-aside-lab/internal/metrics"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("product not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetByID(
	ctx context.Context,
	id int64,
) (*Product, error) {
	var p Product

	start := time.Now()

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			name,
			description,
			price,
			updated_at
		FROM products
		WHERE id = $1
		`,
		id,
	).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.UpdatedAt,
	)

	duration := time.Since(start)

	metrics.DBQueryDuration.
		WithLabelValues("get_product").
		Observe(duration.Seconds())

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	log.Printf(
		"DB QUERY operation=get_product id=%d duration=%s",
		id,
		duration,
	)

	return &p, nil
}