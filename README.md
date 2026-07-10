# RSS Checker — mimile health service

[![CI](https://github.com/BEKZHANjavakotlin/Yeezy/actions/workflows/ci.yml/badge.svg)](https://github.com/BEKZHANjavakotlin/Yeezy/actions/workflows/ci.yml)

Сервис проверки RSS-источников. Gin + PostgreSQL (pgx) + Redis + Prometheus.

Периодически (раз в `HEALTH_INTERVAL`) проверяет каждый активный источник HEAD-запросом и сохраняет результат в `source_health`. Результаты доступны через REST API.

---

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | `postgres://mimile:mimile_secret@localhost:5433/mimile?sslmode=disable` | PostgreSQL DSN |
| `REDIS_URL` | `redis://localhost:6379` | Redis DSN |
| `HEALTH_INTERVAL` | `5m` | Интервал автопроверки (формат Go: `30s`, `1m`, `5m`) |
| `LOG_LEVEL` | `info` | Уровень логов: `debug`, `info`, `warn`, `error` |

---

## Запуск локально

**Шаг 1 — поднять PostgreSQL и Redis:**
```bash
docker-compose up -d postgres redis
```

**Шаг 2 — запустить сервис:**
```bash
go run .
```

Миграции применяются автоматически при старте. Сервер слушает `:8080`.

**Всё сразу (включая Grafana и Prometheus):**
```bash
docker-compose up -d
```

---

## Тесты

```bash
go test -race ./...
```

---

## API

### `POST /checks/sources/run`
Запускает проверку всех активных источников прямо сейчас. Синхронный — ждёт результата.

```bash
curl -s -X POST http://localhost:8080/checks/sources/run | jq .
```

```json
[
  {
    "source_id": 1,
    "name": "Tengrinews",
    "url": "https://tengrinews.kz/rss",
    "language": "ru",
    "status": "200",
    "status_code": 200,
    "latency_ms": 312,
    "checked_at": "2026-07-10T10:00:00Z"
  },
  {
    "source_id": 2,
    "name": "BrokenFeed",
    "url": "https://broken.example.com/rss",
    "language": "en",
    "status": "ERR",
    "status_code": 0,
    "latency_ms": 3001,
    "checked_at": "2026-07-10T10:00:01Z",
    "error": "context deadline exceeded"
  }
]
```

Если проверка уже идёт — `409 Conflict`:
```json
{ "status": "check already running" }
```

---

### `GET /health/sources`
Последний статус каждого активного источника.

```bash
curl -s http://localhost:8080/health/sources | jq .
```

```json
[
  {
    "source_id": 1,
    "checked_at": "2026-07-10T10:00:00Z",
    "ok": true,
    "http_code": 200,
    "latency_ms": 312,
    "error": ""
  }
]
```

---

### `GET /health/sources/:id/history`
История проверок одного источника. Query-параметр `limit` (по умолчанию 50, максимум 500).

```bash
curl -s "http://localhost:8080/health/sources/1/history?limit=10" | jq .
```

```json
[
  { "source_id": 1, "checked_at": "2026-07-10T10:05:00Z", "ok": true,  "http_code": 200, "latency_ms": 290 },
  { "source_id": 1, "checked_at": "2026-07-10T10:00:00Z", "ok": false, "http_code": 0,   "latency_ms": 3001, "error": "timeout" }
]
```

---

### `GET /health/summary`
Агрегированная сводка по всем источникам.

```bash
curl -s http://localhost:8080/health/summary | jq .
```

```json
{
  "total": 12,
  "healthy": 10,
  "unhealthy": 2,
  "avg_latency_ms": 143.5
}
```

---

### `GET /stats/pipeline`
Статистика пайплайна из Redis (пишет Madiar's Celery task, ~15 мин цикл).

```bash
curl -s http://localhost:8080/stats/pipeline | jq .
```

---

### `GET /metrics`
Prometheus метрики.

```bash
curl -s http://localhost:8080/metrics | grep rss_check
```

---

### `DELETE /cache`
Инвалидирует Redis-кеш источников.

```bash
curl -s -X DELETE http://localhost:8080/cache | jq .
```

```json
{ "status": "cache invalidated", "key": "sources:list" }
```

---

## CI/CD

| Шаг | Команда | Что проверяет |
|-----|---------|---------------|
| Vet | `go vet ./...` | Подозрительный код |
| Lint | `golangci-lint run` | Качество кода |
| Test | `go test -race ./...` | Тесты + race detector |

Конфигурация: `.github/workflows/ci.yml`, `.golangci.yml`

---

## Observability — Grafana Dashboard

Дашборд: `observability/grafana-dashboard.json`

Панели: request rate, error rate, p50/p95/p99 latency, in-flight requests, goroutines, heap, GC rate, CPU.

**Импорт:**
1. `docker compose up -d`
2. Grafana: http://localhost:3000 (admin/admin)
3. **Connections → Data Sources → Prometheus** → URL: `http://prometheus:9090` → Save & Test
4. **Dashboards → Import** → выбери `observability/grafana-dashboard.json`

---

## Структура

```
.
├── main.go              — точка входа, HTTP роутер
├── health/              — scheduler (фоновые проверки) + repository (SQL)
├── rss/                 — RSSSummary: проверяет источники, возвращает []CheckResult
├── urlcheck/            — HTTP чекер (HEAD-запрос, таймаут 3s)
├── cache/               — Redis: кеш источников + pipeline stats
├── metrics/             — Prometheus счётчики
├── migrations/          — SQL миграции (goose)
├── tests/               — все тесты
└── observability/       — Prometheus конфиг + Grafana dashboard
```
