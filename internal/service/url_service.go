package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samsunil1999/url-shortener/internal/repository"
)

var (
	ErrNotFound      = errors.New("URL not found")
	ErrAliasConflict = errors.New("alias already taken")
	ErrAliasInvalid  = errors.New("alias must be 3-30 alphanumeric characters")
)

type ShortenRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
}

type StatsResponse struct {
	ShortCode   string                  `json:"short_code"`
	TotalClicks int64                   `json:"total_clicks"`
	DailyStats  []repository.DailyStats `json:"daily_stats"`
}

type URLService interface {
	Shorten(ctx context.Context, req *ShortenRequest) (*repository.URL, error)
	Resolve(ctx context.Context, code string) (string, error)
	GetStats(ctx context.Context, code string) (*StatsResponse, error)
	Delete(ctx context.Context, code string) error
	RecordClick(code string, req *http.Request)
	StartExpirationWorker(ctx context.Context)
}

type urlService struct {
	repo      repository.URLRepository
	analytics repository.AnalyticsRepository
	redis     *redis.Client
	logger    *slog.Logger
}

func NewURLService(repo repository.URLRepository, analytics repository.AnalyticsRepository, redis *redis.Client, logger *slog.Logger) URLService {
	return &urlService{repo: repo, analytics: analytics, redis: redis, logger: logger}
}

func (s *urlService) Shorten(ctx context.Context, req *ShortenRequest) (*repository.URL, error) {
	url := &repository.URL{
		OriginalURL: req.OriginalURL,
	}

	// if req.CustomAlias != "" {
	// 	if err := s.validateAlias(req.CustomAlias); err != nil {
	// 		return nil, err
	// 	}
	// 	url.ShortCode = req.CustomAlias
	// 	url.CustomAlias = req.CustomAlias
	// }

	if err := s.repo.Create(ctx, url); err != nil {
		s.logger.Error("failed to create URL", "error", err, "original_url", req.OriginalURL)
		return nil, err
	}

	// If no custom alias, encode the DB ID to base62
	// if req.CustomAlias == "" {
	// 	url.ShortCode = shortener.Encode(uint64(url.ID))
	// 	// Update the short_code in DB
	// 	if err := s.repo.UpdateShortCode(ctx, url.ID, url.ShortCode); err != nil {
	// 		return nil, err
	// 	}
	// }

	return url, nil
}

func (s *urlService) Resolve(ctx context.Context, code string) (string, error) {
	// Cache-aside: check Redis first
	cacheKey := "url:" + code
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached, nil
	}

	// Cache miss — query Postgres
	url, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	// Check expiry
	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return "", ErrNotFound
	}

	// Warm the cache
	s.redis.Set(ctx, cacheKey, url.OriginalURL, 24*time.Hour)
	return url.OriginalURL, nil
}

func (s *urlService) GetStats(ctx context.Context, code string) (*StatsResponse, error) {
	total, err := s.analytics.GetTotalClicks(ctx, code)
	if err != nil {
		return nil, err
	}
	daily, err := s.analytics.GetDailyStats(ctx, code)
	if err != nil {
		return nil, err
	}
	return &StatsResponse{
		ShortCode:   code,
		TotalClicks: total,
		DailyStats:  daily,
	}, nil
}

func (s *urlService) Delete(ctx context.Context, code string) error {
	s.redis.Del(ctx, "url:"+code)
	return s.repo.DeleteByShortCode(ctx, code)
}

func (s *urlService) RecordClick(code string, req *http.Request) {
	event := repository.ClickEventFromRequest(code, req)
	ctx := context.Background()
	if err := s.analytics.RecordClick(ctx, event); err != nil {
		s.logger.Error("failed to record click", "error", err, "code", code)
	}
}

func (s *urlService) StartExpirationWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				expired, err := s.repo.GetExpiredURLs(ctx)
				if err != nil {
					s.logger.Error("expiration worker error", "error", err)
					continue
				}
				for _, code := range expired {
					s.redis.Del(ctx, "url:"+code)
					s.repo.DeleteByShortCode(ctx, code)
				}
				s.logger.Info("expiration sweep done", "expired_count", len(expired))
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
