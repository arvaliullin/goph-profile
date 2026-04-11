# GophProfile

## Сервисы

| Сервис | Как работает |
| --- | --- |
| `profiled` | HTTP-сервер: REST API аватаров, health, Swagger. Загрузка кладет оригинал в MinIO и метаданные в PostgreSQL, затем шлет в Kafka событие обработки. Удаление через API помечает запись в БД и публикует событие с ключами объектов в MinIO, которые нужно убрать. |
| `avatard` | Consumer group Kafka: по событию загрузки читает оригинал из MinIO, пишет миниатюры и обновляет запись в PostgreSQL; по событию удаления вызывает удаление этих ключей в MinIO. |

## Быстрый старт

### Запуск через Docker Compose

```bash
make up
```

Полный стек - в `deployments/docker-compose.yaml`. UI - `http://localhost:8081`.

### Swagger

После запуска **profiled** документация OpenAPI: `http://localhost:8081/swagger/`.
