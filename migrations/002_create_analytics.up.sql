CREATE TABLE click_events (
    id         BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    referer    TEXT,
    country    VARCHAR(2),
    clicked_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_click_short_code ON click_events(short_code);
CREATE INDEX idx_clicked_at       ON click_events(clicked_at);