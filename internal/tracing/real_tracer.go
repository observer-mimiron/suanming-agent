// Package tracing 暂与 tracing.go 共享包注释，本文件提供真实的追踪实现，包含 Span 树构建和可选的磁盘持久化。

package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// realTracer 是 Tracer 的真实实现，在内存中收集 TurnTrace，
// 并可选择通过 Collector 持久化到磁盘。
type realTracer struct {
	mu        sync.Mutex
	collector *FileCollector
}

// NewRealTracer 创建一个真实 Tracer，收集追踪数据并可选的写入文件。
// collector 传 nil 则跳过持久化（追踪仍可用于前端展示）。
func NewRealTracer(collector *FileCollector) Tracer {
	return &realTracer{collector: collector}
}

func (r *realTracer) StartTrace(ctx context.Context, name string) (context.Context, Trace) {
	t := &TurnTrace{
		TraceID:   genID("trc"),
		StartedAt: time.Now(),
		Status:    "ok",
		Spans:     []TraceSpan{},
		Attributes: map[string]any{},
	}
	root := &realTrace{
		turn:     t,
		spanID:   genID("spn"),
		name:     name,
		kind:     KindAgent,
		start:    t.StartedAt,
		tracer:   r,
	}
	root.turn.Spans = append(root.turn.Spans, root.toSpan())
	return contextWithTrace(ctx, t), root
}

func (r *realTracer) finish(t *TurnTrace) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.collector != nil {
		if err := r.collector.Save(t); err != nil {
			// 非致命：追踪收集尽力而为，失败不影响主流程
			_ = err
		}
	}
}

// realTrace 是 Trace 接口的实现。
type realTrace struct {
	turn   *TurnTrace
	spanID string
	name   string
	kind   SpanKind
	start  time.Time
	end    time.Time
	status string
	err    string
	attrs  map[string]any
	tracer *realTracer
}

func (rt *realTrace) StartSpan(name string) Span {
	s := &realSpan{
		turn:   rt.turn,
		spanID: genID("spn"),
		parent: rt.spanID,
		name:   name,
		start:  time.Now(),
		status: "ok",
		attrs:  map[string]any{},
	}
	return s
}

func (rt *realTrace) SetStatus(status string) {
	rt.status = status
	rt.turn.Status = status
}

func (rt *realTrace) End() {
	rt.end = time.Now()
	if rt.status == "" || rt.status == "ok" {
		if rt.err != "" {
			rt.status = "error"
		} else {
			rt.status = "ok"
		}
	}
	// 更新追踪中的根 span 条目
	rt.turn.EndedAt = rt.end
	rt.turn.Status = rt.status
	if rt.turn.Spans[0].SpanID == rt.spanID {
		rt.turn.Spans[0] = rt.toSpan()
	}
	rt.tracer.finish(rt.turn)
}

func (rt *realTrace) toSpan() TraceSpan {
	dur := rt.end.Sub(rt.start).Milliseconds()
	if dur < 0 {
		dur = 0
	}
	return TraceSpan{
		SpanID:       rt.spanID,
		ParentSpanID: "",
		Name:         rt.name,
		Kind:         rt.kind,
		Status:       rt.status,
		StartedAt:    rt.start,
		EndedAt:      rt.end,
		DurationMs:   dur,
		Error:        rt.err,
		Attributes:   rt.attrs,
	}
}

// realSpan 是 Span 接口的实现。
type realSpan struct {
	turn   *TurnTrace
	spanID string
	parent string
	name   string
	kind   SpanKind
	start  time.Time
	end    time.Time
	status string
	err    string
	attrs  map[string]any
}

func (rs *realSpan) End() {
	rs.end = time.Now()
	if rs.err != "" && rs.status == "ok" {
		rs.status = "error"
	}
	rs.turn.Spans = append(rs.turn.Spans, rs.toSpan())
}

func (rs *realSpan) SetKind(kind SpanKind)  { rs.kind = kind }
func (rs *realSpan) SetStatus(status string) { rs.status = status }

func (rs *realSpan) SetAttribute(key string, value any) {
	if rs.attrs == nil {
		rs.attrs = map[string]any{}
	}
	rs.attrs[key] = value
}

func (rs *realSpan) RecordError(err error) {
	if err != nil {
		rs.err = err.Error()
		if rs.status == "ok" {
			rs.status = "error"
		}
	}
}

func (rs *realSpan) toSpan() TraceSpan {
	dur := rs.end.Sub(rs.start).Milliseconds()
	if dur < 0 {
		dur = 0
	}
	kind := rs.kind
	if kind == "" {
		kind = KindChain
	}
	return TraceSpan{
		SpanID:       rs.spanID,
		ParentSpanID: rs.parent,
		Name:         rs.name,
		Kind:         kind,
		Status:       rs.status,
		StartedAt:    rs.start,
		EndedAt:      rs.end,
		DurationMs:   dur,
		Error:        rs.err,
		Attributes:   rs.attrs,
	}
}

// SpanFromContext 在 ctx 中找到 TurnTrace 并创建其下的子 span。
// 当上下文中没有追踪时返回 noop span（不会返回 nil）。
func SpanFromContext(ctx context.Context, name string, kind SpanKind) Span {
	t := TraceFromContext(ctx)
	if t == nil {
		return noopSpan{}
	}
	var parentID string
	if len(t.Spans) > 0 {
		parentID = t.Spans[0].SpanID
	}
	return &realSpan{
		turn:   t,
		spanID: genID("spn"),
		parent: parentID,
		name:   name,
		kind:   kind,
		start:  time.Now(),
		status: "ok",
		attrs:  map[string]any{},
	}
}

func genID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
