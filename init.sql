CREATE TABLE IF NOT EXISTS sites (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    status TEXT DEFAULT 'unknown',
    is_up BOOLEAN DEFAULT false,
    response_time FLOAT DEFAULT 0,
    check_count INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS checks (
    id SERIAL PRIMARY KEY,
    site_id INT REFERENCES sites(id) ON DELETE CASCADE,
    status_code INT,
    response_time FLOAT,
    is_up BOOLEAN,
    checked_at TIMESTAMP DEFAULT NOW()
);