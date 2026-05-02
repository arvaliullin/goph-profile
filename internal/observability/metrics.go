package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registryOnce sync.Once
	registry     *prometheus.Registry

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goph_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"service", "method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "goph_http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route", "status"},
	)
	httpErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goph_http_errors_total",
			Help: "Total HTTP errors",
		},
		[]string{"service", "method", "route", "status"},
	)
	uploadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatars_uploads_total",
			Help: "Total number of avatar uploads",
		},
		[]string{"service", "status", "user_id"},
	)
	uploadDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "avatars_upload_duration_seconds",
			Help:    "Avatar upload duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "status"},
	)
	storageUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "avatars_storage_bytes",
			Help: "Total storage used by avatars",
		},
		[]string{"service", "user_id"},
	)
	deletesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatars_deletes_total",
			Help: "Total number of avatar deletes",
		},
		[]string{"service", "status", "user_id"},
	)
	kafkaPublishedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goph_kafka_published_total",
			Help: "Total Kafka published messages",
		},
		[]string{"service", "topic", "status"},
	)
	kafkaConsumedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goph_kafka_consumed_total",
			Help: "Total Kafka consumed messages",
		},
		[]string{"service", "topic", "status"},
	)
	workerJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goph_worker_jobs_total",
			Help: "Total worker jobs by operation and status",
		},
		[]string{"service", "operation", "status"},
	)
	dbPoolStat = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goph_db_pool_connections",
			Help: "DB pool connection stats",
		},
		[]string{"service", "state"},
	)
)

func ensureRegistry() {
	registryOnce.Do(func() {
		registry = prometheus.NewRegistry()
		registry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			httpRequestsTotal,
			httpRequestDuration,
			httpErrorsTotal,
			uploadsTotal,
			uploadDuration,
			storageUsage,
			deletesTotal,
			kafkaPublishedTotal,
			kafkaConsumedTotal,
			workerJobsTotal,
			dbPoolStat,
		)
	})
}

// MetricsHandler возвращает handler для endpoint /metrics.
func MetricsHandler() http.Handler {
	ensureRegistry()
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// ObserveHTTPRequest пишет RED-метрики HTTP.
func ObserveHTTPRequest(service, method, route string, status int, dur time.Duration) {
	ensureRegistry()
	statusStr := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(service, method, route, statusStr).Inc()
	httpRequestDuration.WithLabelValues(service, method, route, statusStr).Observe(dur.Seconds())
	if status >= http.StatusBadRequest {
		httpErrorsTotal.WithLabelValues(service, method, route, statusStr).Inc()
	}
}

// ObserveUpload пишет бизнес-метрики загрузки.
func ObserveUpload(service, status, userID string, dur time.Duration, size int64) {
	ensureRegistry()
	uploadsTotal.WithLabelValues(service, status, userID).Inc()
	uploadDuration.WithLabelValues(service, status).Observe(dur.Seconds())
	if status == "success" {
		storageUsage.WithLabelValues(service, userID).Add(float64(size))
	}
}

// ObserveDeleteStorage уменьшает метрику занятого хранилища.
func ObserveDeleteStorage(service, userID string, size int64) {
	ensureRegistry()
	storageUsage.WithLabelValues(service, userID).Sub(float64(size))
}

// ObserveDelete пишет бизнес-метрики удаления.
func ObserveDelete(service, status, userID string) {
	ensureRegistry()
	deletesTotal.WithLabelValues(service, status, userID).Inc()
}

// ObserveKafkaPublish пишет метрику публикации.
func ObserveKafkaPublish(service, topic, status string) {
	ensureRegistry()
	kafkaPublishedTotal.WithLabelValues(service, topic, status).Inc()
}

// ObserveKafkaConsume пишет метрику потребления.
func ObserveKafkaConsume(service, topic, status string) {
	ensureRegistry()
	kafkaConsumedTotal.WithLabelValues(service, topic, status).Inc()
}

// ObserveWorkerJob пишет статус обработки задач воркера.
func ObserveWorkerJob(service, operation, status string) {
	ensureRegistry()
	workerJobsTotal.WithLabelValues(service, operation, status).Inc()
}

// ObserveDBPool пишет статистику пула подключений.
func ObserveDBPool(service string, total, idle, acquired int32) {
	ensureRegistry()
	dbPoolStat.WithLabelValues(service, "total").Set(float64(total))
	dbPoolStat.WithLabelValues(service, "idle").Set(float64(idle))
	dbPoolStat.WithLabelValues(service, "acquired").Set(float64(acquired))
}
