// Package orchestrator 的 turnloop_manager.go 实现 TurnLoopSessionManager：
// per-session TurnLoop 实例管理 + per-request sink 映射 + 生命周期。
//
// 设计要点：
//   - TurnLoop 是 per-session 长生命周期（eino ADK 原生）
//   - sink 是 per-request 短生命周期（每个 HTTP 请求一个 SSE 连接）
//   - 桥接：manager 维护 sessionID → sink 映射，Push 时更新
//   - 可靠性：TurnLoop 串行化保证同 session 同时只有一个 turn 在跑，sink 不会并发覆盖
//   - preempt：新 Push 时 WithPreempt(AfterChatModel) 终止旧 turn，旧 OnAgentEvents 收到 CancelError 停写
//
// 关联文档：docs/superpowers/plans/2026-06-24-eino-native-capabilities.md A2.1b

package orchestrator

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// TurnLoopCfgFactory 构建 TurnLoopConfig。由 A2.1b 的真集成提供
// （GenInput 合并 messages / PrepareAgent 调当前 runtime 执行主链 /
// OnAgentEvents 桥接事件到 sink）。
//
// getSink 返回当前 turn 的 sink（per-request），OnAgentEvents 用它桥接事件。
// 闭包捕获 manager，每次调用拿最新 sink。
type TurnLoopCfgFactory func(sessionID string, getSink func() EventSink, onIdle func()) adk.TurnLoopConfig[string, *schema.Message]

// TurnLoopSessionManager 管理 per-session TurnLoop 实例 + per-request sink 映射。
type TurnLoopSessionManager struct {
	store   state.Store
	factory TurnLoopCfgFactory
	loops   sync.Map // sessionID -> *adk.TurnLoop[string, *schema.Message]
	sinks   sync.Map // sessionID -> EventSink (per-request, 每次 Push 更新)
}

// NewTurnLoopSessionManager 创建 manager。store 用于 idle 超时落盘。
func NewTurnLoopSessionManager(store state.Store, factory TurnLoopCfgFactory) *TurnLoopSessionManager {
	return &TurnLoopSessionManager{store: store, factory: factory}
}

// Push 投递消息到 session 的 TurnLoop。sink 是本次 request 的 SSE 桥。
// WithPreempt(AfterChatModel) 保证新消息到达时终止旧 turn（在下一个 ChatModel 安全点）。
func (m *TurnLoopSessionManager) Push(_ context.Context, sessionID, msg string, sink EventSink) {
	m.sinks.Store(sessionID, sink)
	loop := m.getOrCreate(sessionID)
	loop.Push(msg, adk.WithPreempt[string, *schema.Message](adk.AfterChatModel))
}

func (m *TurnLoopSessionManager) loadSink(sessionID string) EventSink {
	v, _ := m.sinks.Load(sessionID)
	if v == nil {
		return nil
	}
	return v.(EventSink)
}

// getOrCreate 获取或创建 sessionID 的 TurnLoop。并发安全。
func (m *TurnLoopSessionManager) getOrCreate(sessionID string) *adk.TurnLoop[string, *schema.Message] {
	if v, ok := m.loops.Load(sessionID); ok {
		return v.(*adk.TurnLoop[string, *schema.Message])
	}
	onIdle := func() {
		if st := m.store.LoadOrCreate(sessionID); st != nil {
			if err := m.store.Save(st); err != nil {
				log.Printf("[turnloop_manager] idle save failed: sessionID=%s err=%v", sessionID, err)
			}
		}
		m.loops.Delete(sessionID)
		m.sinks.Delete(sessionID)
		log.Printf("[turnloop_manager] idle evicted: sessionID=%s", sessionID)
	}
	getSink := func() EventSink { return m.loadSink(sessionID) }
	cfg := m.factory(sessionID, getSink, onIdle)
	loop := adk.NewTurnLoop(cfg)
	go func() {
		// Run 阻塞直到 Stop 或 idle 超时
		loop.Run(context.Background())
	}()
	actual, loaded := m.loops.LoadOrStore(sessionID, loop)
	if loaded {
		// 并发场景：另一个 goroutine 先创建了，用已有的，关掉自己创建的
		loop.Stop(adk.WithImmediate())
		return actual.(*adk.TurnLoop[string, *schema.Message])
	}
	return loop
}

// StopAll 停止所有 TurnLoop 实例。graceful=true 等当前 turn 完成。
// 进程 SIGTERM 时调。
func (m *TurnLoopSessionManager) StopAll(graceful bool) {
	m.loops.Range(func(key, value any) bool {
		loop := value.(*adk.TurnLoop[string, *schema.Message])
		if graceful {
			loop.Stop(adk.WithGraceful(), adk.WithGracefulTimeout(10*time.Second))
		} else {
			loop.Stop(adk.WithImmediate())
		}
		return true
	})
}
