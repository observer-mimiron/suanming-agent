// Package handler 提供 HTTP 请求处理层，负责任务编排器与 SSE 推送之间的桥接。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/orchestrator"
	"github.com/observer-mimiron/suanming-agent/internal/sse"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// sseEventSink 将 sse.Sender 适配为 orchestrator.EventSink。
type sseEventSink struct {
	sw        sse.Sender
	dbg       *os.File
	dbgEnc    *json.Encoder
	sessionID string
	cancel    context.CancelFunc
}

// Emit 将一个编排事件写入 SSE，并可选镜像到调试 JSONL。
//
// SSE 写入失败通常表示客户端已经断连；此处必须取消本轮 context，
// 让仍在运行的路由、模型或工具尽快停止，而不是继续占用会话锁。
func (s *sseEventSink) Emit(ctx context.Context, evt orchestrator.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.dbgEnc != nil {
		tid := tracing.TraceIDFromContext(ctx)
		s.dbgEnc.Encode(debugEntry{
			Timestamp: time.Now().Format(time.RFC3339Nano),
			SessionID: s.sessionID,
			TraceID:   tid,
			EventType: evt.Type,
			Payload:   evt.Data,
		})
	}
	if err := s.sw.Send(evt.Type, evt.Data); err != nil {
		if s.cancel != nil {
			s.cancel()
		}
		return err
	}
	return nil
}

// debugEntry is the per-SSE-event debug record written when HTTP debug mode is enabled.
type debugEntry struct {
	Timestamp string `json:"timestamp"`
	SessionID string `json:"session_id"`
	TraceID   string `json:"trace_id"`
	EventType string `json:"event_type"`
	Payload   any    `json:"payload"`
}

// ChatHandler 处理 HTTP 聊天请求，将编排器的 SSE 事件推送给前端。
type ChatHandler struct {
	orch      *orchestrator.Orchestrator
	debugHTTP bool
	debugDir  string
}

// NewChatHandler 创建用于处理 HTTP 聊天请求的 ChatHandler。
// 如果 debugHTTP 为 true，会将 SSE 事件流水记录到 debugDir 目录下的 JSONL 文件中。
func NewChatHandler(orch *orchestrator.Orchestrator, debugHTTP bool, debugDir string) *ChatHandler {
	return &ChatHandler{orch: orch, debugHTTP: debugHTTP, debugDir: debugDir}
}

// resolveSessionID normalizes optional client session IDs and rejects unsafe path-like IDs.
func resolveSessionID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "default", nil
	}
	if err := state.ValidateSessionID(id); err != nil {
		return "", err
	}
	return id, nil
}

// HandleChat 处理 POST /api/chat 请求，解析消息后通过编排器运行完整对话流程。
func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "bad request"})
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		c.JSON(400, gin.H{"error": "message is required"})
		return
	}
	sessionID, err := resolveSessionID(req.SessionID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid session_id"})
		return
	}
	req.SessionID = sessionID

	sw := sse.NewWriter(c)

	var dbgFile *os.File
	var dbgEnc *json.Encoder
	if h.debugHTTP {
		os.MkdirAll(h.debugDir, 0755)
		sid := req.SessionID
		if len(sid) > 8 {
			sid = sid[:8]
		}
		fname := fmt.Sprintf("%s/%s-%s.jsonl", h.debugDir, time.Now().Format("150405"), sid)
		dbgFile, _ = os.Create(fname)
		dbgEnc = json.NewEncoder(dbgFile)
		defer func() {
			dbgFile.Close()
		}()
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	sink := &sseEventSink{sw: sw, dbg: dbgFile, dbgEnc: dbgEnc, sessionID: req.SessionID, cancel: cancel}
	if err := h.orch.Run(ctx, sink, req.SessionID, req.Message); err != nil {
		log.Printf("[handler] orchestrator.Run session=%s error: %v", req.SessionID, err)
		if errors.Is(err, context.Canceled) {
			return
		}
	}
}
