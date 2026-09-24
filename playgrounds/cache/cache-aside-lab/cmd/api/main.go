package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cache-aside-lab/internal/product"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	appmetrics "cache-aside-lab/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()
	appmetrics.Register()

	// ------------------------------------
	// PostgreSQL
	// ------------------------------------

	databaseURL := env(
		"DATABASE_URL",
		"postgres://app:app@localhost:5432/app?sslmode=disable",
	)

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal("postgres:", err)
	}

	// ------------------------------------
	// Redis
	// ------------------------------------

	redisAddr := env(
		"REDIS_ADDR",
		"localhost:6379",
	)

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis:", err)
	}

	// ------------------------------------
	// Application
	// ------------------------------------

	repo := product.NewRepository(db)

	service := product.NewService(
		repo,
		rdb,
		60*time.Second, // TTL of key in Redis
	)

	// ------------------------------------
	// HTTP
	// ------------------------------------

	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /api/products/{id}",
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			id, err := strconv.ParseInt(
				r.PathValue("id"),
				10,
				64,
			)

			if err != nil {
				http.Error(
					w,
					"invalid product id",
					http.StatusBadRequest,
				)
				return
			}

			useCache := true

			if strings.EqualFold(
				r.URL.Query().Get("cache"),
				"false",
			) {
				useCache = false
			}

			p, err := service.GetByID(
				r.Context(),
				id,
				useCache,
			)

			if errors.Is(err, product.ErrNotFound) {
				http.Error(
					w,
					"product not found",
					http.StatusNotFound,
				)
				return
			}

			if err != nil {
				log.Printf("request error: %v", err)

				http.Error(
					w,
					"internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			// Useful while experimenting.
			w.Header().Set(
				"X-Response-Time",
				time.Since(start).String(),
			)

			if err := json.NewEncoder(w).Encode(p); err != nil {
				log.Printf("encode response: %v", err)
			}

			log.Printf(
				"REQUEST id=%d cache=%t duration=%s",
				id,
				useCache,
				time.Since(start),
			)
		},
	)

	mux.Handle("GET /metrics", promhttp.Handler())

	log.Println("HTTP server: :8080")

	if err := http.ListenAndServe(
		":8080",
		mux,
	); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}