// Package llm 包含模型 provider 的适配与调用级合同。
//
// 本文件属于 LLM adapter 层，只负责 ADK 模型调用级有限 retry；
// 不负责业务 validator、repair loop、路由或领域语义。
package llm

import (
	"context"
	"errors"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/repair"
)

var modelRetryStatusPattern = regexp.MustCompile("(?i)(?:status code:|http)\\s*(\\d{3})")

// DefaultModelRetryConfig 返回共享的 ADK 模型调用级 retry 配置。
// MaxRetries 固定为 2，ShouldRetry 只负责 transport、timeout 和空输出分类。
func DefaultModelRetryConfig() *adk.ModelRetryConfig {
	return &adk.ModelRetryConfig{
		MaxRetries:  2,
		ShouldRetry: ModelCallRetryDecision,
	}
}

// ModelCallRetryDecision 返回 ADK 模型调用级的有限 retry 决策。
// 只允许 429、5xx、timeout 和空输出重试；业务校验失败、用户/宿主取消不重试。
func ModelCallRetryDecision(_ context.Context, rc *adk.RetryContext) *adk.RetryDecision {
	if rc == nil {
		return &adk.RetryDecision{Retry: false}
	}
	if rc.Err != nil {
		return &adk.RetryDecision{Retry: shouldRetryModelCallError(rc.Err), Backoff: time.Second}
	}
	if rc.OutputMessage == nil || strings.TrimSpace(rc.OutputMessage.Content) == "" {
		return &adk.RetryDecision{Retry: true, Backoff: time.Second}
	}
	return &adk.RetryDecision{Retry: false}
}

// shouldRetryModelCallError 为 ModelRetryConfig 分类传输错误。
// 它复用 Phase 0 HTTP taxonomy，并刻意忽略业务错误。
func shouldRetryModelCallError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if status, ok := modelRetryHTTPStatus(err); ok {
		return repair.HTTPStatusRetryable(status)
	}
	return false
}

// modelRetryHTTPStatus 从常见模型 SDK 错误中提取 HTTP 状态码。
// 它避免把供应商 SDK 类型引入 LLM retry 合同。
func modelRetryHTTPStatus(err error) (int, bool) {
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		if status, ok := statusFromErrorFields(cur); ok {
			return status, true
		}
		if status, ok := statusFromErrorText(cur.Error()); ok {
			return status, true
		}
	}
	return 0, false
}

// statusFromErrorFields 从常见错误字段中读取 HTTP 状态码。
func statusFromErrorFields(err error) (int, bool) {
	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, false
	}
	for _, name := range []string{"HTTPStatusCode", "StatusCode"} {
		field := v.FieldByName(name)
		if field.IsValid() && field.CanInt() {
			return int(field.Int()), true
		}
	}
	return 0, false
}

// statusFromErrorText 从错误文本中读取 HTTP 状态码。
func statusFromErrorText(text string) (int, bool) {
	match := modelRetryStatusPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0, false
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return status, true
}
