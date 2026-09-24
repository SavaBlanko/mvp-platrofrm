package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"cache-aside-lab/internal/metrics"
)


type Service struct {
	repo  *Repository
	redis *redis.Client
	ttl   time.Duration
}

func NewService(
	repo *Repository,
	redisClient *redis.Client,
	ttl time.Duration,
) *Service {
	return &Service{
		repo:  repo,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	id int64,
	useCache bool,
) (*Product, error) {

	if !useCache {
		return s.repo.GetByID(ctx, id)
	}

	key := fmt.Sprintf("product:%d", id)

	// ------------------------------------
	// 1. READ CACHE
	// ------------------------------------

	cacheStart := time.Now()

	data, err := s.redis.Get(ctx, key).Bytes()

	cacheDuration := time.Since(cacheStart)

	metrics.CacheOperationDuration.
		WithLabelValues("product", "get").
		Observe(cacheDuration.Seconds())

	switch {
	case err == nil:
		// CACHE HIT

		metrics.CacheRequests.WithLabelValues("product", "hit").Inc()

		var p Product

		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf(
				"decode cached product: %w",
				err,
			)
		}

		log.Printf(
			"CACHE HIT key=%s latency=%s",
			key,
			cacheDuration,
		)

		return &p, nil

	case errors.Is(err, redis.Nil):
		// CACHE MISS

		metrics.CacheRequests.WithLabelValues("product", "miss").Inc()

		log.Printf(
			"CACHE MISS key=%s latency=%s",
			key,
			cacheDuration,
		)

	default:
		// Redis is unavailable.
		//
		// Cache should normally not make the
		// whole application unavailable.
		log.Printf(
			"CACHE ERROR key=%s error=%v",
			key,
			err,
		)
	}

	// ------------------------------------
	// 2. READ DATABASE
	// ------------------------------------

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// ------------------------------------
	// 3. POPULATE CACHE
	// ------------------------------------

	data, err = json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf(
			"encode product: %w",
			err,
		)
	}

	start := time.Now()

	err = s.redis.Set(ctx, key, data, s.ttl).Err()

	metrics.CacheOperationDuration.
	    WithLabelValues("product", "set").
	    Observe(time.Since(start).Seconds())

	if err != nil {
	    log.Printf(
	        "CACHE SET ERROR key=%s error=%v",
	        key,
	        err,
	    )
	}

	return p, nil
}