package main

import (
	"context"
	"database/sql"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mimile-ai/mimile/rss-checker/cache"
	"github.com/mimile-ai/mimile/rss-checker/health"
	appmetrics "github.com/mimile-ai/mimile/rss-checker/metrics"
	"github.com/mimile-ai/mimile/rss-checker/rss"
	"github.com/mimile-ai/mimile/rss-checker/urlcheck"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("LOG_LEVEL")),
	})))

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://mimile:mimile_secret@localhost:5433/mimile?sslmode=disable"
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		slog.Error("не удалось открыть БД для миграций", "error", err)
		os.Exit(1)
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.Up(db, "migrations"); err != nil {
		slog.Error("миграции не прошли", "error", err)
		os.Exit(1)
	}
	_ = db.Close()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		slog.Error("не удалось создать пул БД", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	c, err := cache.Connect(redisURL)
	if err != nil {
		slog.Error("неверный REDIS_URL", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	intervalStr := os.Getenv("HEALTH_INTERVAL")
	if intervalStr == "" {
		intervalStr = "5m"
	}

	healthInterval, err := time.ParseDuration(intervalStr)
	if err != nil {
		slog.Error("неверный HEALTH_INTERVAL", "value", intervalStr)
		os.Exit(1)
	}

	gin.SetMode(gin.ReleaseMode)

	myChecker := urlcheck.NewChecker()
	var checkMu sync.Mutex

	scheduler := health.NewScheduler(pool, myChecker, healthInterval)
	go scheduler.Run(ctx)

	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://mimile.ai"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(slogMiddleware(), gin.Recovery())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/check", func(ctx *gin.Context) {
		start := time.Now()
		sources, hit, err := loadSources(ctx.Request.Context(), pool, c)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		go func(srcs []rss.RSSSource) {
			if !checkMu.TryLock() {
				return
			}
			defer checkMu.Unlock()
			results, err := rss.RSSSummary(srcs, myChecker)
			if err != nil {
				slog.Error("RSSSummary failed", "error", err)
				return
			}
			saveCheckResults(context.Background(), pool, results)
		}(sources)

		ctx.JSON(http.StatusOK, gin.H{
			"status":               "success",
			"loaded_sources_count": len(sources),
			"cache_hit":            hit,
			"duration_ms":          time.Since(start).Milliseconds(),
		})
	})

	r.GET("/stats/pipeline", func(ctx *gin.Context) {
		stats, err := c.GetPipelineStats(ctx.Request.Context())
		if err != nil {
			slog.Error("не удалось прочитать stats:pipeline", "error", err)
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "redis unavailable"})
			return
		}
		if len(stats) == 0 {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "stats not yet populated — retry after Madiar's task runs (~15 min cadence)",
			})
			return
		}
		ctx.JSON(http.StatusOK, stats)
	})

	r.POST("/checks/sources/run", func(ctx *gin.Context) {
		if !checkMu.TryLock() {
			ctx.JSON(http.StatusConflict, gin.H{"status": "check already running"})
			return
		}
		defer checkMu.Unlock()

		sources, _, err := loadSources(ctx.Request.Context(), pool, c)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results, err := rss.RSSSummary(sources, myChecker)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		saveCheckResults(ctx.Request.Context(), pool, results)
		ctx.JSON(http.StatusOK, results)
	})

	r.GET("/health/sources", func(ctx *gin.Context) {
		results, err := health.LatestBySource(ctx.Request.Context(), pool)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, results)
	})

	r.GET("/health/sources/:id/history", func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
			return
		}
		limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
		if err != nil || limit < 1 || limit > 500 {
			limit = 50
		}
		results, err := health.HistoryBySource(ctx.Request.Context(), pool, id, limit)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, results)
	})

	r.GET("/health/summary", func(ctx *gin.Context) {
		summary, err := health.Summary(ctx.Request.Context(), pool)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, summary)
	})

	r.DELETE("/cache", func(ctx *gin.Context) {
		if err := c.Invalidate(ctx.Request.Context()); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "cache invalidated", "key": cache.SourcesKey})
	})

	slog.Info("сервер запущен", "port", 8080)
	if err := r.Run(":8080"); err != nil {
		slog.Error("сервер упал", "error", err)
		os.Exit(1)
	}
}

func slogMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		appmetrics.HTTPInFlight.Inc()
		ctx.Next()
		appmetrics.HTTPInFlight.Dec()

		duration := time.Since(start)
		method := ctx.Request.Method
		path := ctx.FullPath()
		status := ctx.Writer.Status()

		slog.Info("request",
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)

		appmetrics.HTTPRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
		appmetrics.HTTPDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	}
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func loadSources(ctx context.Context, pool *pgxpool.Pool, c *cache.Cache) ([]rss.RSSSource, bool, error) {
	sources, err := c.GetSources(ctx)
	if err != nil {
		return nil, false, err
	}
	if sources != nil {
		return sources, true, nil
	}

	rows, err := pool.Query(ctx, "SELECT id, name, url, is_active, language FROM rss_sources ORDER BY id")
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	for rows.Next() {
		var src rss.RSSSource
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.IsActive, &src.Language); err != nil {
			return nil, false, err
		}
		sources = append(sources, src)
	}

	_ = c.SetSources(ctx, sources)

	return sources, false, nil
}

func saveCheckResults(ctx context.Context, pool *pgxpool.Pool, results []rss.CheckResult) {
	for _, r := range results {
		if r.Status == "SKIP" {
			continue
		}
		hr := health.CheckResult{
			SourceID:  r.SourceID,
			CheckedAt: r.CheckedAt,
			OK:        r.Error == "" && r.StatusCode >= 200 && r.StatusCode < 400,
			HTTPCode:  r.StatusCode,
			LatencyMs: r.LatencyMs,
			Error:     r.Error,
		}
		if err := health.SaveResult(ctx, pool, hr); err != nil {
			slog.Error("failed to save check result", "source_id", r.SourceID, "error", err)
		}
	}
}
