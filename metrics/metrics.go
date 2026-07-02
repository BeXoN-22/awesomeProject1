package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// HTTP-запросы: метод + путь + статус-код
	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	// Время ответа: метод + путь (гистограмма → p50/p95/p99 в Grafana)
	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// Результат проверки каждого RSS-источника: имя источника + статус (200/ERR/SKIP)
	RSSCheckResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rss_check_results_total",
		Help: "RSS source check results by source name and status.",
	}, []string{"source", "status"})
)

func init() {
	prometheus.MustRegister(HTTPRequests, HTTPDuration, RSSCheckResults)
}
