package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/mimile-ai/mimile/rss-checker/rss"
	"github.com/mimile-ai/mimile/rss-checker/urlcheck"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Scheduler struct {
	pool     *pgxpool.Pool
	checker  urlcheck.Checker
	interval time.Duration
}

func NewScheduler(pool *pgxpool.Pool, checker urlcheck.Checker, interval time.Duration) *Scheduler {
	return &Scheduler{pool: pool, checker: checker, interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	slog.Info("health scheduler started", "interval", s.interval)

	s.runOnce(ctx)

	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			slog.Info("health scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	sources, err := s.loadSources(ctx)
	if err != nil {
		slog.Error("scheduler: failed to load sources", "error", err)
		return
	}

	for _, src := range sources {
		start := time.Now()
		code, checkErr := s.checker.Check(src.URL)
		latency := time.Since(start).Milliseconds()

		result := CheckResult{
			SourceID:  src.ID,
			CheckedAt: time.Now().UTC(),
			OK:        checkErr == nil && code >= 200 && code < 400,
			HTTPCode:  code,
			LatencyMs: latency,
		}
		if checkErr != nil {
			result.Error = checkErr.Error()
		}

		if err := SaveResult(ctx, s.pool, result); err != nil {
			slog.Error("scheduler: failed to save result", "source_id", src.ID, "error", err)
		}
	}

	slog.Info("health check cycle done", "sources", len(sources))
}

func (s *Scheduler) loadSources(ctx context.Context) ([]rss.RSSSource, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, url, is_active, language FROM rss_sources WHERE is_active = true ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []rss.RSSSource
	for rows.Next() {
		var src rss.RSSSource
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.IsActive, &src.Language); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}
