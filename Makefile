.DEFAULT_GOAL := help

.PHONY: help
help: ## Показать список команд
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

.PHONY: install-deps
install-deps: ## Установить инструменты разработки
	- go install go.uber.org/mock/mockgen@v0.6.0
	- go install github.com/pressly/goose/v3/cmd/goose@latest
	- go install github.com/swaggo/swag/cmd/swag@latest
	- go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: generate
generate: ## Сгенерировать моки и документацию Swagger
	go generate ./...
	swag init -g internal/api/http/router.go -o docs --parseDependency --parseInternal

.PHONY: fmt
fmt: ## Форматировать код
	- go fmt ./...

.PHONY: up
up: ## Запустить полный стек в Docker Compose
	- docker compose -f deployments/docker-compose.yaml up --build -d

.PHONY: down
down: ## Остановить стек Docker Compose
	- docker compose -f deployments/docker-compose.yaml down -v

.PHONY: logs
logs: ## Показать логи Docker Compose
	- docker compose -f deployments/docker-compose.yaml logs -f

.PHONY: up-k8s-deps
up-k8s-deps: ## Запустить зависимости для локального Kubernetes
	- docker compose -f deployments/docker-compose.k8s-deps.yaml up -d

.PHONY: down-k8s-deps
down-k8s-deps: ## Остановить зависимости для локального Kubernetes
	- docker compose -f deployments/docker-compose.k8s-deps.yaml down -v

.PHONY: logs-k8s-deps
logs-k8s-deps: ## Показать логи зависимостей для локального Kubernetes
	- docker compose -f deployments/docker-compose.k8s-deps.yaml logs -f

HELM_CHART ?= deployments/helm/goph-profile
HELM_RELEASE ?= goph-profile
K8S_NAMESPACE ?= goph-profile
HELM_VALUES ?= $(HELM_CHART)/values.yaml
HELM_TIMEOUT ?= 10m
PROFILED_IMAGE ?= goph-profile/profiled:latest
AVATARD_IMAGE ?= goph-profile/avatard:latest
PROFILED_SVC ?= $(HELM_RELEASE)-profiled
K8S_LOCAL_PORT ?= 8080

# Локальный кластер по умолчанию: docker-desktop. Пример: make k8s-upgrade KUBE_CONTEXT=minikube
KUBE_CONTEXT ?= docker-desktop
KUBECTL := kubectl --context $(KUBE_CONTEXT)
HELM := helm --kube-context $(KUBE_CONTEXT)

.PHONY: k8s-assert-local
k8s-assert-local: ## Проверить, что KUBE_CONTEXT указывает на локальный API-сервер
	@kubectl config get-contexts $(KUBE_CONTEXT) >/dev/null 2>&1 || \
		(echo "ОШИБКА: контекст Kubernetes '$(KUBE_CONTEXT)' не найден в kubeconfig." >&2; exit 1)
	@server=$$($(KUBECTL) config view --minify -o jsonpath='{.clusters[0].cluster.server}'); \
	case "$$server" in \
		*127.0.0.1*|*localhost*|*kubernetes.docker.internal*) \
			echo "Локальный кластер: context=$(KUBE_CONTEXT) server=$$server" ;; \
		*) \
			echo "ОШИБКА: контекст '$(KUBE_CONTEXT)' не указывает на локальный кластер (server=$$server)." >&2; \
			echo "Ожидается 127.0.0.1, localhost или kubernetes.docker.internal в URL API-сервера." >&2; \
			exit 1 ;; \
	esac

.PHONY: helm-lint
helm-lint: ## Проверить Helm-чарт
	helm lint $(HELM_CHART)

.PHONY: helm-template
helm-template: ## Вывести манифесты Helm-чарта локально
	helm template $(HELM_RELEASE) $(HELM_CHART) -f $(HELM_VALUES)

.PHONY: k8s-build-images
k8s-build-images: ## Собрать Docker-образы profiled и avatard
	docker build -f build/profiled.Dockerfile -t $(PROFILED_IMAGE) .
	docker build -f build/avatard.Dockerfile -t $(AVATARD_IMAGE) .

.PHONY: image-build-local
image-build-local: k8s-build-images ## Собрать образы profiled и avatard

.PHONY: k8s-install
k8s-install: k8s-assert-local ## Установить релиз в локальный Kubernetes
	$(KUBECTL) create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(HELM) install $(HELM_RELEASE) $(HELM_CHART) -n $(K8S_NAMESPACE) -f $(HELM_VALUES) --timeout $(HELM_TIMEOUT)

.PHONY: k8s-upgrade
k8s-upgrade: k8s-assert-local ## Обновить или установить релиз в локальный Kubernetes
	$(KUBECTL) create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(HELM) upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(K8S_NAMESPACE) -f $(HELM_VALUES) --timeout $(HELM_TIMEOUT)

.PHONY: k8s-uninstall
k8s-uninstall: k8s-assert-local ## Удалить релиз из локального Kubernetes
	$(HELM) uninstall $(HELM_RELEASE) -n $(K8S_NAMESPACE)

.PHONY: k8s-status
k8s-status: k8s-assert-local ## Состояние ресурсов релиза в локальном кластере
	$(HELM) status $(HELM_RELEASE) -n $(K8S_NAMESPACE)
	$(KUBECTL) get pods,svc,ingress,hpa,networkpolicy -n $(K8S_NAMESPACE)
	-@$(KUBECTL) get servicemonitor -n $(K8S_NAMESPACE) 2>/dev/null || true

.PHONY: k8s-logs
k8s-logs: k8s-assert-local ## Показать логи profiled в локальном кластере
	$(KUBECTL) logs deployment/$(PROFILED_SVC) -n $(K8S_NAMESPACE) --tail=200 -f

.PHONY: k8s-port-forward
k8s-port-forward: k8s-assert-local ## Проброс порта profiled на локальный хост
	@echo "http://127.0.0.1:$(K8S_LOCAL_PORT)/swagger/"
	@echo "http://127.0.0.1:$(K8S_LOCAL_PORT)/health"
	$(KUBECTL) port-forward svc/$(PROFILED_SVC) $(K8S_LOCAL_PORT):80 -n $(K8S_NAMESPACE)

.PHONY: helm-install-local
helm-install-local: k8s-install ## Установить релиз в локальный Kubernetes

.PHONY: helm-upgrade-local
helm-upgrade-local: k8s-upgrade ## Обновить или установить релиз в локальном Kubernetes

.PHONY: helm-deploy-local
helm-deploy-local: up-k8s-deps k8s-build-images k8s-upgrade ## Зависимости, образы и установка чарта в локальный Kubernetes

.PHONY: helm-status-local
helm-status-local: k8s-status ## Состояние ресурсов релиза в локальном кластере

.PHONY: helm-uninstall-local
helm-uninstall-local: k8s-uninstall ## Удалить релиз из локального Kubernetes

.PHONY: test
test: ## Запустить тесты
	go test ./... -count=1

.PHONY: test-cover
test-cover: ## Запустить тесты с отчетом о покрытии
	go test ./internal/... ./pkg/... -count=1 -cover

.PHONY: web-avatars-build
web-avatars-build: ## Собрать веб-интерфейс аватаров
	cd web/avatars && npm ci && npm run build

.PHONY: lint
lint: ## Запустить golangci-lint
	golangci-lint run ./...

.PHONY: kafka-cluster-id
kafka-cluster-id: ## Сгенерировать CLUSTER_ID для Kafka
	docker run --rm confluentinc/cp-kafka:7.6.1 kafka-storage random-uuid
