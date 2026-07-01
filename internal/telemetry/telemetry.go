package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerName         = "github.com/gemfast/server"
	defaultServiceName = "gemfast-server"
)

// Tracer is the package-wide tracer used by custom instrumentation.
func Tracer() oteltrace.Tracer {
	return otel.Tracer(tracerName)
}

// Init configures the global OpenTelemetry tracer provider and returns a
// shutdown function. The exporter is OTLP/HTTP pointing at Honeycomb by default.
//
// Configuration is read from environment variables (standard OTel vars). If
// OTEL_EXPORTER_OTLP_ENDPOINT is not set, the Honeycomb endpoint is used.
// HONEYCOMB_API_KEY is honored as a convenience so an explicit
// OTEL_EXPORTER_OTLP_HEADERS value is not required.
func Init(ctx context.Context, serviceVersion string, cfg *config.Config) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://api.honeycomb.io")
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL") == "" {
		os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_HEADERS") == "" {
		if apiKey := os.Getenv("HONEYCOMB_API_KEY"); apiKey != "" {
			os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", fmt.Sprintf("x-honeycomb-team=%s", apiKey))
		}
	}
	if os.Getenv("OTEL_SERVICE_NAME") == "" {
		os.Setenv("OTEL_SERVICE_NAME", defaultServiceName)
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp http exporter: %w", err)
	}

	deploymentEnv := os.Getenv("DEPLOYMENT_ENV")
	if deploymentEnv == "" {
		deploymentEnv = "local"
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
		semconv.ServiceVersion(serviceVersion),
		semconv.DeploymentEnvironment(deploymentEnv),
	}
	if cfg != nil {
		if cfg.Auth != nil {
			attrs = append(attrs, attribute.String("auth.type", cfg.Auth.Type))
		}
		if len(cfg.Mirrors) > 0 && cfg.Mirrors[0] != nil {
			m := cfg.Mirrors[0]
			attrs = append(attrs,
				attribute.String("mirror.upstream", m.Upstream),
				attribute.String("mirror.hostname", m.Hostname),
				attribute.Bool("mirror.enabled", m.Enabled),
			)
		}
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Info().Str("endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")).
		Str("service", os.Getenv("OTEL_SERVICE_NAME")).
		Msg("opentelemetry tracing initialized")

	return func(shutdownCtx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}, nil
}
