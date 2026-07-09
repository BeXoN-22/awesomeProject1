CREATE TABLE source_health (
                               id          BIGSERIAL PRIMARY KEY,
                               source_id   INT NOT NULL,
                               checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                               ok          BOOLEAN NOT NULL,
                               http_code   INT NOT NULL DEFAULT 0,
                               latency_ms  INT NOT NULL DEFAULT 0,
                               error       TEXT
);

CREATE INDEX ON source_health (source_id, checked_at DESC);