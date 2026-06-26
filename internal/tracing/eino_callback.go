// Package tracing 暂与 tracing.go 共享包注释，本文件提供 Eino 框架的回调钩子，将 LLM 调用追踪接入自定义 Span 系统。

package tracing

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einoutils "github.com/cloudwego/eino/utils/callbacks"
)

type einoCallbackSpanKey string

const (
	einoCallbackConfigKey einoCallbackSpanKey = "tracing.eino_callback_config"
	einoCallbackSpanKey_  einoCallbackSpanKey = "tracing.eino_callback_span"
)

// EinoCallbackSpanConfig 是 Eino 回调 Span 的配置，包含名称、类型和属性。
type EinoCallbackSpanConfig struct {
	Name       string
	Kind       SpanKind
	Attributes map[string]any
}

// WithEinoCallbackSpan 将 Eino 回调 Span 配置存入 context，供回调处理器在 LLM 调用时创建追踪 Span。
func WithEinoCallbackSpan(ctx context.Context, cfg EinoCallbackSpanConfig) context.Context {
	return context.WithValue(ctx, einoCallbackConfigKey, cfg)
}

func einoCallbackSpanConfigFromContext(ctx context.Context) (EinoCallbackSpanConfig, bool) {
	cfg, ok := ctx.Value(einoCallbackConfigKey).(EinoCallbackSpanConfig)
	return cfg, ok
}

func einoSpanFromContext(ctx context.Context) Span {
	span, _ := ctx.Value(einoCallbackSpanKey_).(Span)
	return span
}

// NewEinoTraceCallbackHandler 创建 Eino 回调处理器，在 LLM 调用的开始、结束、出错时创建和完结追踪 Span。
func NewEinoTraceCallbackHandler() einocallbacks.Handler {
	return einoutils.NewHandlerHelper().
		ChatModel(&einoutils.ModelCallbackHandler{
			OnStart: func(ctx context.Context, info *einocallbacks.RunInfo, input *einomodel.CallbackInput) context.Context {
				cfg, ok := einoCallbackSpanConfigFromContext(ctx)
				if !ok {
					return ctx
				}
				name := cfg.Name
				if info != nil && info.Name != "" {
					name = info.Name
				}
				span := SpanFromContext(ctx, name, cfg.Kind)
				for k, v := range cfg.Attributes {
					span.SetAttribute(k, v)
				}
				if model, ok := cfg.Attributes["model"].(string); ok && model != "" {
					span.SetAttribute("gen_ai.request.model", model)
				}
				if input != nil && input.Config != nil && input.Config.Model != "" {
					if _, exists := cfg.Attributes["model"]; !exists {
						span.SetAttribute("model", input.Config.Model)
					}
					span.SetAttribute("gen_ai.request.model", input.Config.Model)
				}
				span.SetAttribute("gen_ai.operation.name", cfg.Name)
					if input != nil && len(input.Messages) > 0 {
						summarizeLLMMessages(span, input.Messages)
						if shouldRecordLLMInputPreview() {
							span.SetAttribute("input.messages.preview", serializeLLMMessagePreview(input.Messages))
						}
					}
				return context.WithValue(ctx, einoCallbackSpanKey_, span)
			},
			OnEnd: func(ctx context.Context, _ *einocallbacks.RunInfo, output *einomodel.CallbackOutput) context.Context {
				var usage *einomodel.TokenUsage
				if output != nil {
					usage = output.TokenUsage
				}
				finishEinoCallbackSpan(ctx, nil, usage)
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, _ *einocallbacks.RunInfo, output *schema.StreamReader[*einomodel.CallbackOutput]) context.Context {
				defer output.Close()
				var last *einomodel.CallbackOutput
				for {
					item, err := output.Recv()
					if err != nil {
						break
					}
					if item != nil {
						last = item
					}
				}
				var usage *einomodel.TokenUsage
				if last != nil {
					usage = last.TokenUsage
				}
				finishEinoCallbackSpan(ctx, nil, usage)
				return ctx
			},
			OnError: func(ctx context.Context, _ *einocallbacks.RunInfo, err error) context.Context {
				finishEinoCallbackSpan(ctx, err, nil)
				return ctx
			},
		}).
		Retriever(&einoutils.RetrieverCallbackHandler{
			OnStart: func(ctx context.Context, info *einocallbacks.RunInfo, input *einoretriever.CallbackInput) context.Context {
				name := "retriever"
				if info != nil {
					if info.Name != "" {
						name = info.Name
					} else if info.Type != "" {
						name = info.Type
					}
				}

				span := SpanFromContext(ctx, name, KindRetriever)
				if input != nil {
					if input.Query != "" {
						span.SetAttribute("query", input.Query)
					}
					if input.TopK > 0 {
						span.SetAttribute("top_k", input.TopK)
					}
					if input.Filter != "" {
						span.SetAttribute("filter", input.Filter)
					}
				}
				return context.WithValue(ctx, einoCallbackSpanKey_, span)
			},
			OnEnd: func(ctx context.Context, _ *einocallbacks.RunInfo, output *einoretriever.CallbackOutput) context.Context {
				finishEinoRetrieverSpan(ctx, output)
				return ctx
			},
			OnError: func(ctx context.Context, _ *einocallbacks.RunInfo, err error) context.Context {
				finishEinoRetrieverErrorSpan(ctx, err)
				return ctx
			},
		}).Tool(&einoutils.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *einocallbacks.RunInfo, input *einotool.CallbackInput) context.Context {
			name := "tool"
			if info != nil && info.Name != "" {
				name = info.Name
			}
			span := SpanFromContext(ctx, name, KindTool)
			if input != nil && input.ArgumentsInJSON != "" {
				span.SetAttribute("args", input.ArgumentsInJSON)
			}
			return context.WithValue(ctx, einoCallbackSpanKey_, span)
		},
		OnEnd: func(ctx context.Context, _ *einocallbacks.RunInfo, output *einotool.CallbackOutput) context.Context {
			span := einoSpanFromContext(ctx)
			if span != nil {
				if output != nil && output.Response != "" {
					span.SetAttribute("response", output.Response)
				}
				span.End()
			}
			return ctx
		},
		OnError: func(ctx context.Context, _ *einocallbacks.RunInfo, err error) context.Context {
			span := einoSpanFromContext(ctx)
			if span != nil {
				if err != nil {
					span.RecordError(err)
				}
				span.SetStatus("error")
				span.End()
			}
			return ctx
		},
	}).
		Handler()
}


// AppendCurrentSpanAttribute 向当前活跃的 Eino 回调 Span 追加字符串属性。
// 如果 span 不存在，静默忽略。
func AppendCurrentSpanAttribute(ctx context.Context, key string, value any) {
	span := einoSpanFromContext(ctx)
	if span == nil {
		return
	}
	span.SetAttribute(key, value)
}

func finishEinoCallbackSpan(ctx context.Context, err error, usage *einomodel.TokenUsage) {
	span := einoSpanFromContext(ctx)
	if span == nil {
		return
	}
	if usage != nil {
		span.SetAttribute("output_tokens", usage.CompletionTokens)
		span.SetAttribute("gen_ai.usage.output_tokens", usage.CompletionTokens)
		span.SetAttribute("gen_ai.usage.input_tokens", usage.PromptTokens)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error")
	}
	span.End()
}

func finishEinoRetrieverSpan(ctx context.Context, output *einoretriever.CallbackOutput) {
	span := einoSpanFromContext(ctx)
	if span == nil {
		return
	}

	hits := 0
	hasExplicitHits := false
	if output != nil {
		if output.Docs != nil {
			hits = len(output.Docs)
		}
		if output.Extra != nil {
			if v, ok := output.Extra["hits"]; ok {
				hasExplicitHits = true
				span.SetAttribute("hits", v)
				if n, ok := v.(int); ok {
					hits = n
				}
			}
			if v, ok := output.Extra["degrade_reason"]; ok {
				span.SetAttribute("degrade_reason", v)
			}
			if v, ok := output.Extra["status"].(string); ok && v != "" {
				span.SetStatus(v)
			}
		} else {
			span.SetAttribute("hits", hits)
		}
	}

	if !hasExplicitHits {
		span.SetAttribute("hits", hits)
	}
	if hits == 0 {
		span.SetStatus("degraded")
		if output == nil || output.Extra == nil || output.Extra["degrade_reason"] == nil {
			span.SetAttribute("degrade_reason", "no_results")
		}
	}
	span.End()
}

func finishEinoRetrieverErrorSpan(ctx context.Context, err error) {
	span := einoSpanFromContext(ctx)
	if span == nil {
		return
	}
	span.SetAttribute("hits", 0)
	if err != nil {
		span.RecordError(err)
	}
	span.SetStatus("degraded")
	span.End()
}

// summarizeLLMMessages 记录 LLM 输入 messages 的结构信号，不包含原文内容。
func summarizeLLMMessages(span Span, msgs []*schema.Message) {
	if span == nil {
		return
	}
	roles := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		roles = append(roles, string(m.Role))
	}
	span.SetAttribute("input.message_count", len(roles))
	span.SetAttribute("input.message_roles", strings.Join(roles, ","))
}

// shouldRecordLLMInputPreview 控制是否记录脱敏后的 message 内容预览。
// 默认关闭，避免 OTel mirror 外发完整 prompt、出生资料和历史上下文。
func shouldRecordLLMInputPreview() bool {
	return os.Getenv("TRACE_LLM_INPUT_MESSAGES") == "1"
}

// serializeLLMMessagePreview 把 LLM 输入 messages 序列化为脱敏截断预览，用于本地排障。
func serializeLLMMessagePreview(msgs []*schema.Message) string {
	var b strings.Builder
	for i, m := range msgs {
		if m == nil {
			continue
		}
		role := string(m.Role)
		content := redactTracePreview(m.Content)
		if r := []rune(content); len(r) > 200 {
			content = string(r[:200]) + "...(truncated)"
		}
		fmt.Fprintf(&b, "[%d] %s: %s", i, role, content)
		if i < len(msgs)-1 {
			b.WriteString("\n")
		}
		if r := []rune(b.String()); len(r) > 3000 {
			return string(r[:3000]) + "...(truncated)"
		}
	}
	return b.String()
}

func redactTracePreview(s string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)authorization:\s*bearer\s+[^\s]+`),
		regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*[^\s]+`),
		regexp.MustCompile(`sk-[A-Za-z0-9_-]+`),
		regexp.MustCompile(`\b1[3-9]\d{9}\b`),
		regexp.MustCompile(`\b\d{17}[\dXx]\b`),
	}
	out := s
	for _, p := range patterns {
		out = p.ReplaceAllString(out, "[redacted]")
	}
	return out
}

var installEinoCallbackTracingOnce sync.Once

// InstallEinoCallbackTracing 将 Eino 回调追踪处理器注册到全局回调链中（线程安全，仅执行一次）。
func InstallEinoCallbackTracing() {
	installEinoCallbackTracingOnce.Do(func() {
		einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())
	})
}
