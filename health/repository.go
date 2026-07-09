package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckResult struct {
	SourceID  int
	CheckedAt time.Time
	OK        bool
	HTTPCode  int
	LatencyMs int64
	Error     string
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
