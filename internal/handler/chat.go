// Package handler 提供 HTTP 请求处理层，负责任务编排器与 SSE 推送之间的桥接。
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wikiglobal/suanming-agent/internal/orchestrator"
	"github.com/wikiglobal/suanming-agent/internal/sse"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// sseEventSink 将 sse.Sender 适配为 orchestrator.EventSink。
type sseEventSink struct {
	sw        sse.Sender
	dbg       *os.File
	dbgEnc    *json.Encoder
	sessionID string
}

func (s *sseEventSink) Emit(ctx context.Context, evt orchestrator.Event) error {
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
	return s.sw.Send(evt.Type, evt.Data)
}

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
	if req.SessionID == "" {
		req.SessionID = "default"
	}

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

	sink := &sseEventSink{sw: sw, dbg: dbgFile, dbgEnc: dbgEnc, sessionID: req.SessionID}
	ctx := c.Request.Context()
	if err := h.orch.Run(ctx, sink, req.SessionID, req.Message); err != nil {
		log.Printf("[handler] orchestrator.Run session=%s error: %v", req.SessionID, err)
	}
}
