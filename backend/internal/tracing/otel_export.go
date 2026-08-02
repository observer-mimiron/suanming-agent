package tracing

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OTelConfig describes the minimal OTLP export settings used by the local tracer mirror.
type OTelConfig struct {
	Enabled     bool
	Endpoint    string
	Headers     string
	ServiceName string
	Insecure    bool
}

type otelSDKBridge struct {
	tracer   oteltrace.Tracer
	shutdown func(context.Context) error
}

type otelSDKSpan struct {
	span oteltrace.Span
}

func NewOTelBridge(ctx context.Context, cfg OTelConfig) (otelBridge, func(context.Context) error, error) {
	opts := []otlptracehttp.Option{}
	resolvedEndpoint := normalizeOTLPHTTPEndpoint(cfg.Endpoint)
	if resolvedEndpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpointURL(resolvedEndpoint))
	}
	headers := parseOTLPHeaders(cfg.Headers)
	if isLangfuseEndpoint(cfg.Endpoint) {
		if _, ok := headers["x-langfuse-ingestion-version"]; !ok {
			headers["x-langfuse-ingestion-version"] = "4"
		}
	}
	if len(headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}
	// resource.Default() may carry a newer OpenTelemetry schema URL than the
	// semconv package imported at build time. We only need a stable service name
	// here, so attach it as schemaless attributes to avoid version-coupled
	// SchemaURL merge panics during tests and dependency bumps.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	bridge := &otelSDKBridge{
		tracer:   provider.Tracer("github.com/observer-mimiron/suanming-agent/internal/tracing"),
		shutdown: provider.Shutdown,
	}
	return bridge, provider.Shutdown, nil
}

func (b *otelSDKBridge) StartRoot(ctx context.Context, name string, kind SpanKind) (context.Context, otelSpanBridge) {
	ctx, span := b.tracer.Start(ctx, name)
	handle := &otelSDKSpan{span: span}
	handle.SetAttribute("span_kind", string(kind))
	return ctx, handle
}

func (b *otelSDKBridge) StartChild(ctx context.Context, name string, kind SpanKind) otelSpanBridge {
	_, span := b.tracer.Start(ctx, name)
	handle := &otelSDKSpan{span: span}
	handle.SetAttribute("span_kind", string(kind))
	return handle
}

func (s *otelSDKSpan) SetAttribute(key string, value any) {
	if s == nil || s.span == nil || key == "" {
		return
	}
	if kv, ok := toOTelAttr(key, value); ok {
		s.span.SetAttributes(kv)
	}
}

func (s *otelSDKSpan) SetStatus(status string) {
	if s == nil || s.span == nil {
		return
	}
	switch status {
	case "error", "fallback":
		s.span.SetStatus(codes.Error, status)
	default:
		s.span.SetStatus(codes.Ok, status)
	}
}

func (s *otelSDKSpan) RecordError(err error) {
	if s == nil || s.span == nil || err == nil {
		return
	}
	s.span.RecordError(err)
}

func (s *otelSDKSpan) End() {
	if s == nil || s.span == nil {
		return
	}
	s.span.End()
}

func parseOTLPHeaders(raw string) map[string]string {
	headers := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		headers[key] = value
	}
	return headers
}

func isLangfuseEndpoint(endpoint string) bool {
	endpoint = strings.ToLower(strings.TrimSpace(endpoint))
	if endpoint == "" {
		return false
	}
	if strings.Contains(endpoint, "langfuse") {
		return true
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "localhost", "127.0.0.1":
		return strings.HasPrefix(parsed.Path, "/api/public/otel")
	default:
		return false
	}
}

func normalizeOTLPHTTPEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	if !isLangfuseEndpoint(endpoint) {
		return endpoint
	}
	switch parsed.Path {
	case "/api/public/otel", "/api/public/otel/":
		parsed.Path = "/api/public/otel/v1/traces"
		return parsed.String()
	default:
		return endpoint
	}
}

func toOTelAttr(key string, value any) (attribute.KeyValue, bool) {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v), true
	case bool:
		return attribute.Bool(key, v), true
	case int:
		return attribute.Int(key, v), true
	case int64:
		return attribute.Int64(key, v), true
	case float64:
		return attribute.Float64(key, v), true
	case []string:
		return attribute.StringSlice(key, v), true
	default:
		return attribute.String(key, fmt.Sprint(value)), true
	}
}
