# Go Open Telemetry Tracer (goteletracer)
Open Telemetry Tracer with Insecure GRPC Exporter for Go.

## Installation
```bash
go get github.com/fikri240794/goteletracer
```

## Usage
```go
package main

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"github.com/fikri240794/goteletracer"
)

var (
	tracer trace.Tracer = goteletracer.NewTracer(nil)
)

func sum(ctx context.Context, a, b int) int {
	ctx, span := tracer.Start(ctx, "sum")
	defer span.End()

	return a + b
}

func main() {
	var cfg *goteletracer.Config = &goteletracer.Config{
		ServiceName:         "some-service",
		ExporterGRPCAddress: "localhost:4317",
	}

	tracer = goteletracer.NewTracer(cfg)

	var val int = sum(context.Background(), 1, 1)

	fmt.Println(val)
}
```