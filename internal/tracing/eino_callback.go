// Package tracing 暂与 tracing.go 共享包注释，本文件提供 Eino 框架的回调钩子，将 LLM 调用追踪接入自定义 Span 系统。

package tracing

import (
	"context"
	"sync"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
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
			OnStart: func(ctx context.Context, _ *einocallbacks.RunInfo, input *einomodel.CallbackInput) context.Context {
				cfg, ok := einoCallbackSpanConfigFromContext(ctx)
				if !ok {
					return ctx
				}
				span := SpanFromContext(ctx, cfg.Name, cfg.Kind)
				for k, v := range cfg.Attributes {
					span.SetAttribute(k, v)
				}
				if input != nil && input.Config != nil && input.Config.Model != "" {
					if _, exists := cfg.Attributes["model"]; !exists {
						span.SetAttribute("model", input.Config.Model)
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
		Handler()
}

func finishEinoCallbackSpan(ctx context.Context, err error, usage *einomodel.TokenUsage) {
	span := einoSpanFromContext(ctx)
	if span == nil {
		return
	}
	if usage != nil {
		span.SetAttribute("output_tokens", usage.CompletionTokens)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error")
	}
	span.End()
}

var installEinoCallbackTracingOnce sync.Once

// InstallEinoCallbackTracing 将 Eino 回调追踪处理器注册到全局回调链中（线程安全，仅执行一次）。
func InstallEinoCallbackTracing() {
	installEinoCallbackTracingOnce.Do(func() {
		einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())
	})
}
