package repository

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type URL struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	CustomAlias string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

type URLRepository interface {
	Create(ctx context.Context, url *URL) error
	GetByShortCode(ctx context.Context, code string) (*URL, error)
	GetExpiredURLs(ctx context.Context) ([]string, error)
	DeleteByShortCode(ctx context.Context, code string) error
	UpdateShortCode(ctx context.Context, id int64, shortCode string) error
}

type urlRepository struct {
	db *sql.DB
}

func NewURLRepository(db *sql.DB) URLRepository {
	return &urlRepository{db: db}
}

func (r *urlRepository) Create(ctx context.Context, url *URL) error {
	query := `
        INSERT INTO urls (short_code, original_url, custom_alias, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		url.ShortCode, url.OriginalURL, url.CustomAlias, url.ExpiresAt,
	).Scan(&url.ID, &url.CreatedAt)
}

func (r *urlRepository) GetByShortCode(ctx context.Context, code string) (*URL, error) {
	url := &URL{}
	query := `
        SELECT id, short_code, original_url, expires_at, created_at
        FROM urls WHERE short_code = $1`
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&url.ID, &url.ShortCode, &url.OriginalURL, &url.ExpiresAt, &url.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func (r *urlRepository) GetExpiredURLs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT short_code FROM urls WHERE expires_at IS NOT NULL AND expires_at < NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func (r *urlRepository) DeleteByShortCode(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM urls WHERE short_code = $1`, code)
	return err
}

func (r *urlRepository) UpdateShortCode(ctx context.Context, id int64, shortCode string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE urls SET short_code = $1 WHERE id = $2`, shortCode, id)
	return err
}
