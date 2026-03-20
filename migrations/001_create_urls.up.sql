CREATE TABLE urls (
    id           BIGSERIAL PRIMARY KEY,
    short_code   VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    expires_at   TIMESTAMP,
    created_at   TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_short_code ON urls(short_code);