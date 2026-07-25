package tracing

import "testing"

func TestParseOTLPHeaders(t *testing.T) {
	headers := parseOTLPHeaders("Authorization=Basic abc123, X-Test = demo ,invalid")
	if got := headers["Authorization"]; got != "Basic abc123" {
		t.Fatalf("Authorization = %q, want %q", got, "Basic abc123")
	}
	if got := headers["X-Test"]; got != "demo" {
		t.Fatalf("X-Test = %q, want %q", got, "demo")
	}
	if _, ok := headers["invalid"]; ok {
		t.Fatal("invalid header entry should be ignored")
	}
}

func TestIsLangfuseEndpoint(t *testing.T) {
	if !isLangfuseEndpoint("https://cloud.langfuse.com/api/public/otel") {
		t.Fatal("expected langfuse endpoint to be detected")
	}
	if !isLangfuseEndpoint("http://localhost:3001/api/public/otel") {
		t.Fatal("expected local Langfuse OTLP endpoint to be detected")
	}
	if isLangfuseEndpoint("http://localhost:4318/v1/traces") {
		t.Fatal("plain OTLP collector should not be treated as langfuse endpoint")
	}
}

func TestNormalizeOTLPHTTPEndpoint(t *testing.T) {
	if got := normalizeOTLPHTTPEndpoint("http://localhost:3001/api/public/otel"); got != "http://localhost:3001/api/public/otel/v1/traces" {
		t.Fatalf("normalized local Langfuse endpoint = %q", got)
	}
	if got := normalizeOTLPHTTPEndpoint("https://cloud.langfuse.com/api/public/otel"); got != "https://cloud.langfuse.com/api/public/otel/v1/traces" {
		t.Fatalf("normalized cloud Langfuse endpoint = %q", got)
	}
	if got := normalizeOTLPHTTPEndpoint("http://localhost:3001/api/public/otel/v1/traces"); got != "http://localhost:3001/api/public/otel/v1/traces" {
		t.Fatalf("already-complete traces endpoint should remain unchanged, got %q", got)
	}
	if got := normalizeOTLPHTTPEndpoint("http://localhost:4318/v1/traces"); got != "http://localhost:4318/v1/traces" {
		t.Fatalf("plain OTLP collector endpoint should remain unchanged, got %q", got)
	}
}
