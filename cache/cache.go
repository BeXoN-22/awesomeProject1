package cache

import (
	"awesomeProject1/rss"
	"context"
	"encoding/json"
	"log/slog"
	"time"

	redisc "github.com/redis/go-redis/v9"
)

const (
	SourcesKey      = "sources:list"
	SourcesTTL      = 5 * time.Minute
	PipelineStatsKey = "stats:pipeline"
)

type Cache struct {
	rdb *redisc.Client
}

func Connect(redisURL string) (*Cache, error) {
	opts, err := redisc.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	opts.MaxRetries = 0
	opts.DialTimeout = 300 * time.Millisecond
	opts.ReadTimeout = 300 * time.Millisecond
	opts.WriteTimeout = 300 * time.Millisecond

	rdb := redisc.NewClient(opts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Redis недоступен — работаем без кэша", "error", err)
	} else {
		slog.Info("Redis подключён")
	}

	return &Cache{rdb: rdb}, nil
}

func (c *Cache) GetSources(ctx context.Context) ([]rss.RSSSource, error) {
	raw, err := c.rdb.Get(ctx, SourcesKey).Bytes()
	if err != nil {
		return nil, nil
	}
	var sources []rss.RSSSource
	if err := json.Unmarshal(raw, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func (c *Cache) SetSources(ctx context.Context, sources []rss.RSSSource) error {
	data, err := json.Marshal(sources)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, SourcesKey, data, SourcesTTL).Err()
}

func (c *Cache) Invalidate(ctx context.Context) error {
	return c.rdb.Del(ctx, SourcesKey).Err()
}

// GetPipelineStats reads stats:pipeline hash written by Madiar's Celery task (~15 min cadence).
// Returns nil map (not error) when the key doesn't exist yet.
func (c *Cache) GetPipelineStats(ctx context.Context) (map[string]string, error) {
	result, err := c.rdb.HGetAll(ctx, PipelineStatsKey).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}
