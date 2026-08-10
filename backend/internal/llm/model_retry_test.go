package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type modelRetryStatusError struct {
	HTTPStatusCode int
}

func (e modelRetryStatusError) Error() string {
	return fmt.Sprintf("status code: %d", e.HTTPStatusCode)
}

type modelRetryTimeoutError struct{}

func (modelRetryTimeoutError) Error() string   { return "model call timeout" }
func (modelRetryTimeoutError) Timeout() bool   { return true }
func (modelRetryTimeoutError) Temporary() bool { return true }

func TestShouldRetryModelCallError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "rate limit field", err: modelRetryStatusError{HTTPStatusCode: http.StatusTooManyRequests}, want: true},
		{name: "server error text", err: errors.New("failed to create chat completion: HTTP 503: busy"), want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "net timeout", err: modelRetryTimeoutError{}, want: true},
		{name: "bad request", err: modelRetryStatusError{HTTPStatusCode: http.StatusBadRequest}, want: false},
		{name: "payment required", err: errors.New("error, status code: 402, status: , message: quota"), want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "business validation", err: errors.New("static_projection_mismatch"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryModelCallError(tt.err); got != tt.want {
				t.Fatalf("shouldRetryModelCallError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelCallRetryDecisionRetriesEmptyOutput(t *testing.T) {
	decision := ModelCallRetryDecision(context.Background(), &adk.RetryContext{
		OutputMessage: schema.AssistantMessage("  ", nil),
	})
	if decision == nil || !decision.Retry {
		t.Fatalf("decision = %+v, want retry", decision)
	}
}

func TestDefaultModelRetryConfig(t *testing.T) {
	config := DefaultModelRetryConfig()
	if config == nil {
		t.Fatal("config = nil, want default model retry config")
	}
	if config.MaxRetries != 2 {
		t.Fatalf("MaxRetries = %d, want 2", config.MaxRetries)
	}
	if config.ShouldRetry == nil {
		t.Fatal("ShouldRetry = nil, want model call retry decision")
	}
}
