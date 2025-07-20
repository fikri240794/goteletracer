# GoTeleTracer 🔍

Open Telemetry Tracer with Insecure GRPC Exporter for Go.

## 📦 Installation

```bash
go get github.com/fikri240794/goteletracer
```

## 🚀 Quick Start

### Basic Usage (Simple)

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/fikri240794/goteletracer"
)

func main() {
    // Create tracer with configuration
    config := &goteletracer.Config{
        ServiceName:         "my-awesome-service",
        ExporterGRPCAddress: "localhost:4317",
    }
    
    tracer := goteletracer.NewTracer(config)
    
    // Your business logic here
    result := doSomething(context.Background())
    fmt.Println("Result:", result)
}

func doSomething(ctx context.Context) string {
    // Create and use spans
    ctx, span := tracer.Start(ctx, "doSomething")
    defer span.End()

    return "Hello, World!"
}
```

## 🔧 Configuration

### Config Structure

```go
type Config struct {
    // ServiceName is the name of your service (required)
    ServiceName string
    
    // ExporterGRPCAddress is the OTLP collector endpoint (required)
    // Example: "localhost:4317", "jaeger:14250"
    ExporterGRPCAddress string
    
    // ShutdownTimeout defines maximum time for graceful shutdown
    // Default: 30 seconds
    ShutdownTimeout time.Duration
}
```

### Environment Setup

For local development with OTLP collector and Jaeger, check out the complete setup example at:
🔗 **[go-otel-tracer-example](https://github.com/fikri240794/go-otel-tracer-example)**

This repository provides a complete Docker Compose setup for local development with OpenTelemetry.

## 🎯 API Reference

### Functions

#### `NewTracer(cfg *Config) trace.Tracer`
Creates a new OpenTelemetry tracer. Returns a noop tracer if config is nil or invalid.

#### `NewTracerProvider(cfg *Config) (*TracerProvider, error)`
Creates a new TracerProvider with proper resource management. **Recommended for production use.**

### TracerProvider Methods

#### `Tracer() trace.Tracer`
Returns the underlying OpenTelemetry tracer.

#### `Shutdown(ctx context.Context) error`
Gracefully shuts down the provider and flushes all spans. Safe to call multiple times.

### Error Types

```go
var (
    ErrNilConfig             = errors.New("config cannot be nil")
    ErrEmptyServiceName      = errors.New("service name cannot be empty")
    ErrEmptyExporterAddress  = errors.New("exporter GRPC address cannot be empty")
    ErrInvalidExporterAddress = errors.New("exporter GRPC address is invalid")
)
```

## 📚 Examples

Check out the [examples](./examples/) directory for comprehensive usage examples:

```bash
# Run the examples
cd examples
go run main.go
```

The examples demonstrate:
- Basic tracer usage
- Advanced TracerProvider usage
- Error handling patterns
- Complex nested spans
- Graceful shutdown

---