package tracing

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger" // #nosec SA1019 - Jaeger exporter migration to OTLP planned, using deprecated API temporarily
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	Enabled     bool    `json:"enabled"`
	ServiceName string  `json:"service_name"`
	Environment string  `json:"environment"`
	JaegerURL   string  `json:"jaeger_url"`
	SampleRate  float64 `json:"sample_rate"` // 0.0 to 1.0
}

// TracerProvider wraps the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	closer   io.Closer
}

// InitTracer initializes the OpenTelemetry tracer with Jaeger exporter
func InitTracer(cfg Config) (*TracerProvider, error) {
	if !cfg.Enabled {
		return &TracerProvider{}, nil
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(cfg.JaegerURL),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on configuration
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SampleRate),
	)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return &TracerProvider{
		provider: tp,
		closer:   nil, // Jaeger exporter no longer implements io.Closer; provider.Shutdown handles cleanup
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		if err := tp.provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}
	if tp.closer != nil {
		if err := tp.closer.Close(); err != nil {
			return fmt.Errorf("failed to close exporter: %w", err)
		}
	}
	return nil
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, tracerName string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetTracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithAttributes(attrs...))
}

// Common attribute keys for consistency
var (
	// HTTP attributes
	HTTPMethodKey     = attribute.Key("http.method")
	HTTPPathKey       = attribute.Key("http.path")
	HTTPStatusCodeKey = attribute.Key("http.status_code")
	HTTPUserAgentKey  = attribute.Key("http.user_agent")

	// Database attributes
	DBSystemKey    = attribute.Key("db.system")
	DBNameKey      = attribute.Key("db.name")
	DBStatementKey = attribute.Key("db.statement")
	DBOperationKey = attribute.Key("db.operation")
	DBRowCountKey  = attribute.Key("db.row_count")

	// User attributes
	UserIDKey    = attribute.Key("user.id")
	UserEmailKey = attribute.Key("user.email")

	// Custom attributes
	OrganizationIDKey = attribute.Key("organization.id")
	QueryIDKey        = attribute.Key("query.id")
	SyncOperationKey  = attribute.Key("sync.operation")
)

// Example usage:
/*
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Start a span for the entire request
	ctx, span := tracing.StartSpan(r.Context(), "http-handler", "HandleRequest")
	defer span.End()

	// Add attributes
	tracing.SetAttributes(ctx,
		tracing.HTTPMethodKey.String(r.Method),
		tracing.HTTPPathKey.String(r.URL.Path),
	)

	// Process request
	result, err := h.processRequest(ctx)
	if err != nil {
		// Record error
		tracing.RecordError(ctx, err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add success event
	tracing.AddEvent(ctx, "request_processed",
		attribute.Int("result_count", len(result)),
	)

	// Write response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) processRequest(ctx context.Context) ([]string, error) {
	// Start a child span for database query
	ctx, span := tracing.StartSpan(ctx, "database", "QueryUsers")
	defer span.End()

	tracing.SetAttributes(ctx,
		tracing.DBSystemKey.String("postgres"),
		tracing.DBOperationKey.String("SELECT"),
	)

	// Execute query...
	return []string{"result"}, nil
}
*/
