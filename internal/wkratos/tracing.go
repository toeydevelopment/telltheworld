package wkratos

import (
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

type TracingConfig struct {
	ProjectID string
	// 0 < SampleRatio < 1
	SampleRatio float64
	ServiceName string
}

func NewTracing(cfg TracingConfig) error {
	if cfg.ProjectID == "" {
		return nil
	}
	// Create the Jaeger exporter
	exp, err := texporter.New(texporter.WithProjectID(cfg.ProjectID))

	if err != nil {
		return err
	}

	tp := tracesdk.NewTracerProvider(
		// Set the sampling rate based on the parent span to 100%
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(1.0))),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
	)

	otel.SetTracerProvider(tp)

	return nil
}
