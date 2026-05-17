# GophProfile

## Сервисы

- `profiled` - HTTP API аватаров, health, Swagger, метрики Prometheus.
- `avatard` - Kafka consumer для генерации миниатюр и удаления объектов.

## Быстрый старт

### Запуск полного стека через Docker Compose

```bash
make up
```

Полный стек - в `deployments/docker-compose.yaml`. UI - `http://localhost:8081`.

### Swagger

После запуска **profiled** документация OpenAPI: `http://localhost:8081/swagger/`.

## Локальный Kubernetes + Helm

Сценарий для спринта: `profiled` и `avatard` работают в Kubernetes, внешние зависимости запускаются отдельно через Docker Compose. Без запущенных зависимостей приложение в подах получит отказ соединения к PostgreSQL (`connection refused` на `5432`).

Цели `make k8s-*` всегда используют контекст `docker-desktop` (переменная `KUBE_CONTEXT`) и отказываются работать, если API-сервер не локальный (`127.0.0.1`, `localhost`, `kubernetes.docker.internal`). Текущий контекст `kubectl` и Teleport на это не влияют.

Проверка локального кластера:

```bash
kubectl --context docker-desktop cluster-info
make k8s-assert-local
```

### 1) Поднять внешние зависимости

```bash
make up-k8s-deps
```

Файл: `deployments/docker-compose.k8s-deps.yaml`.

Поднимаются:

- PostgreSQL на `localhost:5432`
- MinIO на `localhost:9000`
- Kafka на `localhost:9092`

Chart по умолчанию включает initContainer `wait-for-deps` (см. `waitForDeps` в `values.yaml`): пока с хоста недоступны TCP-порты PostgreSQL, MinIO и Kafka, основные контейнеры не стартуют. Так проще не ловить `CrashLoop` из-за гонки с `docker compose`.

По умолчанию поды обращаются к зависимостям через `host.docker.internal` (Docker Desktop). Для Rancher Desktop добавьте overlay:

```bash
make k8s-upgrade HELM_VALUES=deployments/helm/goph-profile/values-rancher.yaml
```

или объедините файлы: `-f values.yaml -f values-rancher.yaml`.

### 2) Собрать образы и установить chart

```bash
make k8s-build-images
make helm-lint
make helm-template
make k8s-upgrade
```

Либо одной командой:

```bash
make helm-deploy-local
```

С HPA и ServiceMonitor для проверки приёмки:

```bash
make helm-deploy-local HELM_VALUES=deployments/helm/goph-profile/values-local.yaml
```

Chart: `deployments/helm/goph-profile`. Namespace: `goph-profile`.

Миграции БД выполняются Helm hook Job (`migration-hook-job.yaml`) до установки/обновления релиза.

### 3) Проверить доступность

```bash
make k8s-status
kubectl --context docker-desktop -n goph-profile get hpa,servicemonitor,ingress
```

- браузер: `make k8s-port-forward` → http://127.0.0.1:8080/swagger/ и http://127.0.0.1:8080/health
- ingress: `http://goph-profile.local` (добавьте запись в `/etc/hosts`, если нужно)
- логи: `make k8s-logs`

### 4) Остановить

```bash
make k8s-uninstall
make down-k8s-deps
```

### Если поды не становятся Ready

- В логах `profiled`/`avatard`: `connection refused` к `host.docker.internal:5432` - не запущен PostgreSQL из шага 1 или порт занят другим процессом. Выполните `make up-k8s-deps`, проверьте `docker compose -f deployments/docker-compose.k8s-deps.yaml ps`.
- Под зависает в `Init` на контейнере `wait-for-deps` - с хоста не открываются порты `5432`, `9000` или `9092`. Дождитесь готовности сервисов или поправьте `waitForDeps.host` (для Rancher - `values-rancher.yaml`).
- `ImagePullBackOff` / локальные теги `latest` - соберите образы: `make k8s-build-images` и повторите `make k8s-upgrade`.
- Hook миграций в `Error` - проверьте `kubectl --context docker-desktop -n goph-profile logs job/goph-profile-goph-profile-migrate`.
- `avatard` в `Error` / рестарты - часто Kafka: в `k8s-deps` брокер должен рекламировать `host.docker.internal:9092`, а не `localhost`. Пересоздайте Kafka: `make down-k8s-deps && make up-k8s-deps`. Для Rancher: `KAFKA_ADVERTISED_HOST=host.rancher-desktop.internal make up-k8s-deps`.
- `k8s-status` и ServiceMonitor - CRD Prometheus Operator в локальном кластере может отсутствовать; это нормально, если `serviceMonitor.enabled=false`.

## Переменные конфигурации

Основные значения задаются в `deployments/helm/goph-profile/values.yaml`.

Файлы окружений:

| Файл | Назначение |
|------|------------|
| `values.yaml` | Базовые настройки, Docker Desktop (`host.docker.internal`) |
| `values-local.yaml` | 2 реплики, HPA и ServiceMonitor для локальной приёмки |
| `values-dev.yaml` | 1 реплика, autoscaling выключен |
| `values-prod.yaml` | 3+ реплики, HPA, увеличенные resource limits |
| `values-rancher.yaml` | Overlay для `host.rancher-desktop.internal` |

Секреты задаются в блоке `secretEnv`:

- `databaseURI`
- `minioAccessKey`
- `minioSecretKey`

Нечувствительные настройки задаются в `envConfig`, включая:

- `minioEndpoint`
- `kafkaBrokers`
- `maxUploadBytes`
- `publicBaseURL`

## Мониторинг и алерты

- `profiled` отдает метрики на `/metrics`.
- В chart есть `ServiceMonitor` (`templates/servicemonitor.yaml`), включается через `serviceMonitor.enabled` или `values-local.yaml`.
- Базовая проверка мониторинга:

```bash
kubectl --context docker-desktop -n goph-profile get servicemonitor
kubectl --context docker-desktop -n goph-profile describe servicemonitor <name>
```

## Безопасность

- non-root контейнеры (`runAsNonRoot`, `runAsUser`).
- ограничение привилегий (`allowPrivilegeEscalation: false`, drop all capabilities).
- `NetworkPolicy` для ingress/egress.
- отдельный `ServiceAccount` и минимальный `Role/RoleBinding`.

## Архитектура

Kubernetes запускает два приложения: HTTP-сервис `profiled` и воркер `avatard`. Трафик снаружи попадает на Ingress с хостом `goph-profile.local`, который проксирует запросы в Service `profiled`. Оба Deployment читают общие настройки из ConfigMap и секреты из Secret. PostgreSQL, MinIO и Kafka поднимаются вне кластера (`deployments/docker-compose.k8s-deps.yaml`) и доступны подам через `host.docker.internal`. Миграции схемы выполняет Helm hook Job перед деплоем. Метрики `profiled` отдаются на `/metrics`; при включенном `serviceMonitor.enabled` Prometheus Operator находит поды через ServiceMonitor.
