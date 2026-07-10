package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckResult struct {
	SourceID  int       `json:"source_id"`
	CheckedAt time.Time `json:"checked_at"`
	OK        bool      `json:"ok"`
	HTTPCode  int       `json:"http_code"`
	LatencyMs int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
}

type SummaryResult struct {
	Total      int     `json:"total"`
	Healthy    int     `json:"healthy"`
	Unhealthy  int     `json:"unhealthy"`
	AvgLatency float64 `json:"avg_latency_ms"`
}

func SaveResult(ctx context.Context, pool *pgxpool.Pool, r CheckResult) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO source_health (source_id, checked_at, ok, http_code, latency_ms, error)
                 VALUES ($1, $2, $3, $4, $5, $6)`,
		r.SourceID, r.CheckedAt, r.OK, r.HTTPCode, r.LatencyMs, r.Error,
	)
	return err
}

func LatestBySource(ctx context.Context, pool *pgxpool.Pool) ([]CheckResult, error) {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT ON (source_id) source_id, checked_at, ok, http_code, latency_ms, error
                 FROM source_health
                 ORDER BY source_id, checked_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var r CheckResult
		if err := rows.Scan(&r.SourceID, &r.CheckedAt, &r.OK, &r.HTTPCode, &r.LatencyMs, &r.Error); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func HistoryBySource(ctx context.Context, pool *pgxpool.Pool, sourceID, limit int) ([]CheckResult, error) {
	rows, err := pool.Query(ctx,
		`SELECT source_id, checked_at, ok, http_code, latency_ms, error
                 FROM source_health
                 WHERE source_id = $1
                 ORDER BY checked_at DESC
                 LIMIT $2`,
		sourceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var r CheckResult
		if err := rows.Scan(&r.SourceID, &r.CheckedAt, &r.OK, &r.HTTPCode, &r.LatencyMs, &r.Error); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func Summary(ctx context.Context, pool *pgxpool.Pool) (SummaryResult, error) {
	row := pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (source_id) ok, latency_ms
			FROM source_health
			ORDER BY source_id, checked_at DESC
		)
		SELECT
			COUNT(*)                              AS total,
			COUNT(*) FILTER (WHERE ok)            AS healthy,
			COUNT(*) FILTER (WHERE NOT ok)        AS unhealthy,
			COALESCE(AVG(latency_ms), 0)          AS avg_latency
		FROM latest
	`)
	var s SummaryResult
	err := row.Scan(&s.Total, &s.Healthy, &s.Unhealthy, &s.AvgLatency)
	return s, err
}
