package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// TelemetryConfig описывает настройку OTEL.
type TelemetryConfig struct {
	ServiceName  string
	Environment  string
	OTLPEndpoint string
}

// InitTracing инициализирует tracer provider и global propagator.
func InitTracing(ctx context.Context, cfg TelemetryConfig) (func(context.Context) error, error) {
	resAttrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	}
	if strings.TrimSpace(cfg.Environment) != "" {
		resAttrs = append(resAttrs, resource.WithAttributes(semconv.DeploymentEnvironmentName(cfg.Environment)))
	}
	res, err := resource.New(ctx, append([]resource.Option{resource.WithFromEnv()}, resAttrs...)...)
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}
	if endpoint := strings.TrimSpace(cfg.OTLPEndpoint); endpoint != "" {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
