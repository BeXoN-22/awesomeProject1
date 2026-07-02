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

## Структура

```
awesomeProject1/
├── main.go          — точка входа
├── internal/        — основная логика (CheckURLs)
├── tests/           — тесты
├── rss/             — RSS парсер
├── cache/           — Redis кеш
├── metrics/         — Prometheus метрики
└── urlcheck/        — HTTP чекер URL
```
