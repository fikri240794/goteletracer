package goteletracer

import (
	"context"

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

type Config struct {
	ServiceName         string
	ExporterGRPCAddress string
}

func NewTracer(cfg *Config) trace.Tracer {
	var (
		ctx               context.Context
		tracerResource    *resource.Resource
		grpcClientConn    *grpc.ClientConn
		tracerExporter    *otlptrace.Exporter
		tracerProvider    *sdk_trace.TracerProvider
		textMapPropagator propagation.TextMapPropagator
		tracer            trace.Tracer
		err               error
	)

	if cfg == nil {
		tracer = noop.NewTracerProvider().Tracer("")
		return tracer
	}

	ctx = context.Background()

	tracerResource, err = resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
	)
	if err != nil {
		panic(err)
	}

	grpcClientConn, err = grpc.NewClient(
		cfg.ExporterGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	tracerExporter, err = otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(grpcClientConn),
	)
	if err != nil {
		panic(err)
	}

	tracerProvider = sdk_trace.NewTracerProvider(
		sdk_trace.WithResource(tracerResource),
		sdk_trace.WithSpanProcessor(sdk_trace.NewBatchSpanProcessor(tracerExporter)),
		sdk_trace.WithSampler(sdk_trace.AlwaysSample()),
	)
	textMapPropagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(textMapPropagator)
	tracer = otel.Tracer(cfg.ServiceName)

	return tracer
}
