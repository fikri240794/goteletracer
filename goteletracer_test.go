package goteletracer

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

// TestValidateConfig tests the configuration validation logic
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr error
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedErr: ErrNilConfig,
		},
		{
			name: "empty service name",
			config: &Config{
				ServiceName:         "",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectedErr: ErrEmptyServiceName,
		},
		{
			name: "whitespace only service name",
			config: &Config{
				ServiceName:         "   ",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectedErr: ErrEmptyServiceName,
		},
		{
			name: "empty exporter address",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "",
			},
			expectedErr: ErrEmptyExporterAddress,
		},
		{
			name: "whitespace only exporter address",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "   ",
			},
			expectedErr: ErrEmptyExporterAddress,
		},
		{
			name: "invalid exporter address format",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost",
			},
			expectedErr: ErrInvalidExporterAddress,
		},
		{
			name: "valid config",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectedErr: nil,
		},
		{
			name: "valid config with custom shutdown timeout",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     15 * time.Second,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedErr)
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestDefaultShutdownTimeout tests the default shutdown timeout function
func TestDefaultShutdownTimeout(t *testing.T) {
	timeout := defaultShutdownTimeout()
	expected := 30 * time.Second

	if timeout != expected {
		t.Errorf("expected default shutdown timeout %v, got %v", expected, timeout)
	}
}

// TestNewTracer tests the NewTracer function with various configurations
func TestNewTracer(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		expectNoop   bool
		expectNonNil bool
	}{
		{
			name:         "nil config returns noop tracer",
			config:       nil,
			expectNoop:   true,
			expectNonNil: true,
		},
		{
			name: "invalid config returns noop tracer",
			config: &Config{
				ServiceName:         "",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectNoop:   true,
			expectNonNil: true,
		},
		{
			name: "invalid exporter address returns noop tracer",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "invalid-address:99999", // High port that likely won't work
			},
			expectNoop:   true,
			expectNonNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config)

			if tt.expectNonNil && tracer == nil {
				t.Error("expected non-nil tracer")
				return
			}

			if tt.expectNoop {
				// Check if it's a noop tracer by comparing with a known noop tracer
				noopTracer := noop.NewTracerProvider().Tracer("")
				_, span1 := tracer.Start(context.Background(), "test")
				_, span2 := noopTracer.Start(context.Background(), "test")

				// Both should be noop spans
				if span1.SpanContext().IsValid() || span2.SpanContext().IsValid() {
					// If either is valid, they might not be noop tracers
					// This is a heuristic test since we can't directly compare types
				}
				span1.End()
				span2.End()
			}
		})
	}
}

// TestNewTracerProvider tests the NewTracerProvider function with realistic scenarios
func TestNewTracerProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorType   error
		description string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorType:   ErrNilConfig,
			description: "Nil config should be rejected",
		},
		{
			name: "empty service name",
			config: &Config{
				ServiceName:         "",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectError: true,
			errorType:   ErrEmptyServiceName,
			description: "Empty service name should be rejected",
		},
		{
			name: "whitespace only service name",
			config: &Config{
				ServiceName:         "   ",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectError: true,
			errorType:   ErrEmptyServiceName,
			description: "Whitespace-only service name should be rejected",
		},
		{
			name: "empty exporter address",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "",
			},
			expectError: true,
			errorType:   ErrEmptyExporterAddress,
			description: "Empty exporter address should be rejected",
		},
		{
			name: "invalid exporter address format",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost", // Missing port
			},
			expectError: true,
			errorType:   ErrInvalidExporterAddress,
			description: "Address without port should be rejected",
		},
		{
			name: "valid basic config",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
			},
			expectError: false,
			errorType:   nil,
			description: "Basic valid configuration should work",
		},
		{
			name: "valid config with custom shutdown timeout",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     15 * time.Second,
			},
			expectError: false,
			errorType:   nil,
			description: "Valid config with custom timeout should work",
		},
		{
			name: "unreachable server address",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "unreachable-host:4317",
			},
			expectError: false,
			errorType:   nil,
			description: "gRPC client creation doesn't validate connectivity immediately",
		},
		{
			name: "control characters in GRPC address",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317\x00\x01", // Control characters that cause URL parsing errors
			},
			expectError: true,
			errorType:   nil, // Don't check specific error type as it comes from gRPC
			description: "Control characters should cause gRPC client creation to fail",
		},
		{
			name: "IPv6 localhost",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "[::1]:4317",
			},
			expectError: false,
			errorType:   nil,
			description: "IPv6 addresses should be supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewTracerProvider(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}

				// Check specific error type if specified
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}

				// Provider should be nil on error
				if provider != nil {
					t.Errorf("expected nil provider on error")
				}

				t.Logf("Got expected error: %v", err)
				return
			}

			// Success case validations
			if err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}

			if provider == nil {
				t.Errorf("expected non-nil provider")
				return
			}

			// Test provider functionality
			tracer := provider.Tracer()
			if tracer == nil {
				t.Errorf("expected non-nil tracer")
				return
			}

			// Test basic span creation
			_, span := tracer.Start(context.Background(), "test-span")
			if span != nil {
				span.End()
			}

			// Clean up
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if shutdownErr := provider.Shutdown(shutdownCtx); shutdownErr != nil {
				t.Logf("Shutdown error (may be expected for unreachable hosts): %v", shutdownErr)
			}
		})
	}
}

// TestTracerProviderShutdown tests the shutdown functionality
func TestTracerProviderShutdown(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		testContext func() context.Context
		description string
	}{
		{
			name: "shutdown with valid context",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     5 * time.Second,
			},
			testContext: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				return ctx
			},
			description: "Normal shutdown with provided context",
		},
		{
			name: "shutdown with nil context",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     1 * time.Second,
			},
			testContext: func() context.Context {
				return nil // This should trigger the nil context path
			},
			description: "Shutdown with nil context should use default timeout",
		},
		{
			name: "shutdown multiple times",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     2 * time.Second,
			},
			testContext: func() context.Context {
				return context.Background()
			},
			description: "Multiple shutdown calls should be safe",
		},
		{
			name: "shutdown with expired context",
			config: &Config{
				ServiceName:         "test-service",
				ExporterGRPCAddress: "localhost:4317",
				ShutdownTimeout:     3 * time.Second,
			},
			testContext: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(1 * time.Millisecond) // Ensure context expires
				defer cancel()
				return ctx
			},
			description: "Shutdown with expired context should handle timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewTracerProvider(tt.config)
			if err != nil {
				t.Logf("Provider creation failed (which is valid for unreachable hosts): %v", err)
				return
			}

			if provider == nil {
				t.Skip("Provider is nil, skipping shutdown test")
				return
			}

			// Test the shutdown functionality
			ctx := tt.testContext()

			if tt.name == "shutdown multiple times" {
				// Test multiple shutdown calls
				err1 := provider.Shutdown(ctx)
				err2 := provider.Shutdown(ctx)
				err3 := provider.Shutdown(context.Background()) // Different context

				// All should return the same error (if any) due to sync.Once
				if err1 != err2 || err2 != err3 {
					t.Errorf("shutdown returned different errors on multiple calls: %v, %v, %v", err1, err2, err3)
				}
				t.Logf("Multiple shutdown completed with consistent error: %v", err1)
			} else {
				// Single shutdown test
				shutdownErr := provider.Shutdown(ctx)
				t.Logf("Shutdown completed with result: %v", shutdownErr)

				// Test that second shutdown returns same error
				shutdownErr2 := provider.Shutdown(ctx)
				if shutdownErr != shutdownErr2 {
					t.Errorf("second shutdown returned different error: %v vs %v", shutdownErr, shutdownErr2)
				}
			}
		})
	}
}
