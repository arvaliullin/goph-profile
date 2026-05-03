.DEFAULT_GOAL := help

.PHONY: help
help: ## Показать список доступных команд
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

.PHONY: install-deps
install-deps: ## Установить инструменты (mockgen, goose, swag, golangci-lint)
	- go install go.uber.org/mock/mockgen@v0.6.0
	- go install github.com/pressly/goose/v3/cmd/goose@latest
	- go install github.com/swaggo/swag/cmd/swag@latest
	- go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: generate
generate: ## Сгенерировать моки и Swagger (go generate, swag init)
	go generate ./...
	swag init -g internal/api/http/router.go -o docs --parseDependency --parseInternal

.PHONY: fmt
fmt: ## Форматировать код
	- go fmt ./...

.PHONY: up
up: ## Запустить стек через Docker Compose (postgres, minio, kafka, profiled, avatard, nginx)
	- docker compose -f deployments/docker-compose.yaml up --build -d

.PHONY: down
down: ## Остановить и удалить контейнеры Docker Compose
	- docker compose -f deployments/docker-compose.yaml down -v

.PHONY: logs
logs: ## Логи Docker Compose
	- docker compose -f deployments/docker-compose.yaml logs -f

.PHONY: test
test: ## Запустить тесты
	go test ./... -count=1

# Покрытие по пакетам приложения (без cmd и glue миграций в оценке приемки).
.PHONY: test-cover
test-cover: ## Тесты с покрытием internal + pkg
	go test ./internal/... ./pkg/... -count=1 -cover

.PHONY: web-avatars-build
web-avatars-build: ## Собрать Vite UI в web/avatars/dist
	cd web/avatars && npm ci && npm run build

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run ./...

.PHONY: kafka-cluster-id
kafka-cluster-id: ## Случайный CLUSTER_ID для Kafka KRaft (docker run)
	docker run --rm confluentinc/cp-kafka:7.6.1 kafka-storage random-uuid
