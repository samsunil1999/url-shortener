package repository

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type ClickEvent struct {
	ShortCode string
	IPAddress string
	UserAgent string
	Referer   string
	ClickedAt time.Time
}

type DailyStats struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

type AnalyticsRepository interface {
	RecordClick(ctx context.Context, event *ClickEvent) error
	GetDailyStats(ctx context.Context, code string) ([]DailyStats, error)
	GetTotalClicks(ctx context.Context, code string) (int64, error)
}

type analyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) RecordClick(ctx context.Context, event *ClickEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO click_events (short_code, ip_address, user_agent, referer)
         VALUES ($1, $2, $3, $4)`,
		event.ShortCode, event.IPAddress, event.UserAgent, event.Referer,
	)
	return err
}

func (r *analyticsRepository) GetDailyStats(ctx context.Context, code string) ([]DailyStats, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT DATE(clicked_at)::text, COUNT(*)
        FROM click_events
        WHERE short_code = $1
          AND clicked_at >= NOW() - INTERVAL '30 days'
        GROUP BY DATE(clicked_at)
        ORDER BY DATE(clicked_at)`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		rows.Scan(&s.Date, &s.Clicks)
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *analyticsRepository) GetTotalClicks(ctx context.Context, code string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM click_events WHERE short_code = $1`, code,
	).Scan(&count)
	return count, err
}

// Helper to build a ClickEvent from an HTTP request
func ClickEventFromRequest(code string, req *http.Request) *ClickEvent {
	return &ClickEvent{
		ShortCode: code,
		IPAddress: req.RemoteAddr,
		UserAgent: req.UserAgent(),
		Referer:   req.Referer(),
	}
}
