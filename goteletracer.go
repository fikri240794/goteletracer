// Package goteletracer provides OpenTelemetry tracer functionality with GRPC exporter
package goteletracer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdk_trace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Common errors returned by the tracer package
var (
	ErrNilConfig              = errors.New("config cannot be nil")
	ErrEmptyServiceName       = errors.New("service name cannot be empty")
	ErrEmptyExporterAddress   = errors.New("exporter GRPC address cannot be empty")
	ErrInvalidExporterAddress = errors.New("exporter GRPC address is invalid")
)

// Config holds the configuration for the OpenTelemetry tracer
type Config struct {
	// ServiceName is the name of the service that will be used in telemetry data
	ServiceName string
	// ExporterGRPCAddress is the address of the OTLP GRPC exporter endpoint
	ExporterGRPCAddress string
	// ShutdownTimeout defines the maximum time to wait for graceful shutdown
	// Default is 30 seconds if not specified
	ShutdownTimeout time.Duration
}

// TracerProvider wraps the OpenTelemetry tracer provider with additional functionality
type TracerProvider struct {
	tracer          trace.Tracer
	provider        *sdk_trace.TracerProvider
	exporter        *otlptrace.Exporter
	grpcConn        *grpc.ClientConn
	shutdownOnce    sync.Once
	shutdownErr     error
	shutdownTimeout time.Duration
}

// validateConfig validates the provided configuration
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return ErrNilConfig
	}

	if strings.TrimSpace(cfg.ServiceName) == "" {
		return ErrEmptyServiceName
	}

	if strings.TrimSpace(cfg.ExporterGRPCAddress) == "" {
		return ErrEmptyExporterAddress
	}

	// Basic address validation - check if it contains host:port format
	if !strings.Contains(cfg.ExporterGRPCAddress, ":") {
		return ErrInvalidExporterAddress
	}

	return nil
}

// defaultShutdownTimeout returns the default shutdown timeout
func defaultShutdownTimeout() time.Duration {
	return 30 * time.Second
}

// NewTracer creates a new OpenTelemetry tracer with the provided configuration.
// Returns a noop tracer if config is nil.
// For production use, use NewTracerProvider for better resource management.
func NewTracer(cfg *Config) trace.Tracer {
	if cfg == nil {
		return noop.NewTracerProvider().Tracer("")
	}

	tracerProvider, err := NewTracerProvider(cfg)
	if err != nil {
		// Fallback to noop tracer to maintain backward compatibility
		return noop.NewTracerProvider().Tracer("")
	}

	return tracerProvider.Tracer()
}

// NewTracerProvider creates a new TracerProvider with the provided configuration.
// This is the recommended way to create tracers as it provides better resource management.
func NewTracerProvider(cfg *Config) (*TracerProvider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set default shutdown timeout if not provided
	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create resource with service information
	tracerResource, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer resource: %w", err)
	}

	// Create GRPC connection with timeout
	grpcConn, err := grpc.NewClient(
		cfg.ExporterGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GRPC connection: %w", err)
	}

	// Create OTLP exporter
	tracerExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(grpcConn),
	)
	if err != nil {
		// Clean up connection on error
		grpcConn.Close()
		return nil, fmt.Errorf("failed to create tracer exporter: %w", err)
	}

	// Create tracer provider with batch span processor for better performance
	tracerProvider := sdk_trace.NewTracerProvider(
		sdk_trace.WithResource(tracerResource),
		sdk_trace.WithSpanProcessor(sdk_trace.NewBatchSpanProcessor(tracerExporter)),
		sdk_trace.WithSampler(sdk_trace.AlwaysSample()),
	)

	// Set up propagators for distributed tracing
	textMapPropagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(textMapPropagator)

	// Create tracer instance
	tracer := otel.Tracer(cfg.ServiceName)

	return &TracerProvider{
		tracer:          tracer,
		provider:        tracerProvider,
		exporter:        tracerExporter,
		grpcConn:        grpcConn,
		shutdownTimeout: shutdownTimeout,
	}, nil
}

// Tracer returns the underlying OpenTelemetry tracer
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// Shutdown gracefully shuts down the tracer provider and all its components.
// It ensures all spans are flushed before closing connections.
// This method is safe to call multiple times.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	tp.shutdownOnce.Do(func() {
		// Create context with timeout if none provided
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), tp.shutdownTimeout)
			defer cancel()
		}

		// Shutdown tracer provider (this flushes remaining spans)
		if tp.provider != nil {
			if err := tp.provider.Shutdown(ctx); err != nil {
				tp.shutdownErr = fmt.Errorf("failed to shutdown tracer provider: %w", err)
				return
			}
		}

		// Close GRPC connection
		if tp.grpcConn != nil {
			if err := tp.grpcConn.Close(); err != nil {
				tp.shutdownErr = fmt.Errorf("failed to close GRPC connection: %w", err)
				return
			}
		}
	})

	return tp.shutdownErr
}
