# awesomeProject1 — RSS Feed Checker

[![CI](https://github.com/BEKZHANjavakotlin/Yeezy/actions/workflows/ci.yml/badge.svg)](https://github.com/BEKZHANjavakotlin/Yeezy/actions/workflows/ci.yml)

Сервис проверки RSS-источников. Gin + PostgreSQL (pgx) + Redis + Prometheus.

## Запуск

```bash
docker-compose up -d
go run main.go
```

## Тесты

```bash
go test -race ./...
```

## CI/CD

Каждый pull request автоматически проходит три проверки:

| Шаг | Команда | Что проверяет |
|-----|---------|---------------|
| Vet | `go vet ./...` | Подозрительный код, неправильные аргументы Printf |
| Lint | `golangci-lint run` | Качество кода: неиспользуемые переменные, игнорируемые ошибки |
| Test | `go test -race ./...` | Все тесты + race detector для горутин |

Конфигурация воркфлоу: `.github/workflows/ci.yml`  
Конфигурация линтера: `.golangci.yml`

## Observability — Grafana Dashboard

Дашборд находится в `observability/grafana-dashboard.json`.

Панели: request rate, error rate, p50/p95/p99 latency, in-flight requests, goroutines, heap memory, GC rate, CPU.

### Импорт дашборда

1. Запусти весь стек:
   ```bash
   docker compose up -d
   ```

2. Открой Grafana: http://localhost:3000  
   Логин: `admin` / Пароль: `admin`

3. Добавь Prometheus datasource:  
   **Connections → Data Sources → Add → Prometheus**  
   URL: `http://prometheus:9090`  
   Нажми **Save & Test**

4. Импортируй дашборд:  
   **Dashboards → Import → Upload JSON file**  
   Выбери `observability/grafana-dashboard.json`  
   В поле **Prometheus** выбери datasource из шага 3

5. Переменная **Instance** в верхней части фильтрует по инстансу сервиса.

### Переменные дашборда

| Переменная | Описание |
|---|---|
| `DS_PROMETHEUS` | Prometheus datasource |
| `instance` | Инстанс сервиса (из лейбла `instance` в метриках) |

## Структура

```
awesomeProject1/
├── main.go              — точка входа
├── internal/            — основная логика (CheckURLs)
├── health/              — background scheduler + repository
├── tests/               — тесты
├── rss/                 — RSS парсер
├── cache/               — Redis кеш
├── metrics/             — Prometheus метрики
├── urlcheck/            — HTTP чекер URL
├── migrations/          — SQL миграции
└── observability/       — Prometheus конфиг + Grafana dashboard
```
