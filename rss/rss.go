package rss

import (
	"strconv"
	"sync"
	"time"

	"github.com/mimile-ai/mimile/rss-checker/metrics"
	"github.com/mimile-ai/mimile/rss-checker/urlcheck"
)

type RSSSource struct {
	ID       int
	Name     string
	URL      string
	IsActive bool
	Language string
}

type CheckResult struct {
	SourceID   int       `json:"source_id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Language   string    `json:"language"`
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	CheckedAt  time.Time `json:"checked_at"`
	Error      string    `json:"error,omitempty"`
}

func RSSSummary(sources []RSSSource, checker urlcheck.Checker) ([]CheckResult, error) {
	results := make([]CheckResult, len(sources))
	var wg sync.WaitGroup

	for i, src := range sources {
		r := CheckResult{
			SourceID:  src.ID,
			Name:      src.Name,
			URL:       src.URL,
			Language:  src.Language,
			CheckedAt: time.Now().UTC(),
		}
		if !src.IsActive {
			r.Status = "SKIP"
			results[i] = r
			metrics.RSSCheckResults.WithLabelValues(src.Name, "SKIP").Inc()
			continue
		}
		wg.Add(1)
		go func(i int, src RSSSource, r CheckResult) {
			defer wg.Done()
			start := time.Now()
			code, err := checker.Check(src.URL)
			r.LatencyMs = time.Since(start).Milliseconds()
			r.CheckedAt = time.Now().UTC()
			r.StatusCode = code
			if err != nil {
				r.Status = "ERR"
				r.Error = err.Error()
			} else {
				r.Status = strconv.Itoa(code)
			}
			results[i] = r
			metrics.RSSCheckResults.WithLabelValues(src.Name, r.Status).Inc()
		}(i, src, r)
	}
	wg.Wait()
	return results, nil
}
