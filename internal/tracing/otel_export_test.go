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
	if isLangfuseEndpoint("http://localhost:4318/v1/traces") {
		t.Fatal("plain OTLP collector should not be treated as langfuse endpoint")
	}
}
