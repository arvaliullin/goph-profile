# GophProfile

Сервис аватаров: HTTP API и асинхронная обработка (миниатюры, удаление объектов) через Kafka.

## Сервисы

- `profiled` - HTTP API аватаров, health, Swagger, метрики на `/metrics`.
- `avatard` - Kafka consumer для генерации миниатюр и удаления объектов.

## Архитектура

[Архитектура GophProfile](docs/dia/arch.png)

| Компонент | Роль |
| --- | --- |
| `profiled` | HTTP API, загрузка, Swagger, `/health`, `/metrics` |
| `avatard` | consumer Kafka: миниатюры, удаление в MinIO |
| PostgreSQL | метаданные аватаров |
| MinIO | объекты и превью |
| Kafka | события между API и воркером |
| UI (`web/avatars`) | фронт; в Docker Compose отдаётся через nginx на порту 8081 |

Клиент обращается к `profiled`, который пишет в БД и MinIO и публикует события в Kafka. `avatard` читает очередь, обновляет MinIO и при необходимости метаданные в PostgreSQL.

## Запуск

### Docker Compose

```bash
make up
make down
```

Стек: `deployments/docker-compose.yaml`. UI: `http://localhost:8081`, OpenAPI: `http://localhost:8081/swagger/`. Полный список целей: `make help`.

### Локальный Kubernetes (Helm)

`profiled` и `avatard` в кластере, PostgreSQL, MinIO и Kafka на хосте: `deployments/docker-compose.k8s-deps.yaml`. По умолчанию поды ходят к зависимостям через `host.docker.internal` (Docker Desktop).

Цели `make k8s-*` используют `KUBE_CONTEXT=docker-desktop`; `make k8s-assert-local` разрешает только локальный API-сервер (`127.0.0.1`, `localhost`, `kubernetes.docker.internal`). Проверка кластера:

```bash
kubectl --context docker-desktop cluster-info
make k8s-assert-local
```

| Шаг | Команда | Примечание |
| --- | --- | --- |
| Зависимости | `make up-k8s-deps` | PostgreSQL 5432, MinIO 9000, Kafka 9092 |
| Деплой | `make helm-deploy-local` | зависимости, сборка образов, `helm upgrade --install` |
| Проверка | `make k8s-status`, `make k8s-port-forward` | см. ниже |
| Остановка | `make k8s-uninstall && make down-k8s-deps` | |

После шага "Проверка": Swagger и health через `make k8s-port-forward` - `http://127.0.0.1:8080/swagger/` и `http://127.0.0.1:8080/health`. Ingress: `http://goph-profile.local` (при необходимости запись в `/etc/hosts`). Логи: `make k8s-logs`.

Чарт: `deployments/helm/goph-profile`, namespace: `goph-profile`. HPA и ServiceMonitor для локальной приёмки: `make helm-deploy-local HELM_VALUES=deployments/helm/goph-profile/values-local.yaml`.

Если поды не в Ready: нет зависимостей или `connection refused` к БД - `make up-k8s-deps`; `ImagePullBackOff` - `make k8s-build-images` и снова `make k8s-upgrade`; проблемы Kafka - пересоздать deps.

Секреты и env: `deployments/helm/goph-profile/values.yaml` (`secretEnv`, `envConfig`). Окружения: `values-dev.yaml`, `values-prod.yaml`, `values-local.yaml`.
