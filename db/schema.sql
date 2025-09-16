CREATE TABLE IF NOT EXISTS session (
    id SERIAL PRIMARY KEY,
    cinema_name TEXT NOT NULL,
    film_name TEXT NOT NULL,
    session_id TEXT NOT NULL,
    seats INTEGER NOT NULL,
    logged_at TIMESTAMPTZ NOT NULL,
    start_hour TIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_session_id ON session(session_id);
CREATE INDEX IF NOT EXISTS idx_session_cinema_name ON session(cinema_name);
CREATE INDEX IF NOT EXISTS idx_session_film_name ON session(film_name);
