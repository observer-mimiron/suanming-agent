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

// sseEventSink adapts sse.Sender to orchestrator.EventSink.
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

// ChatHandler handles HTTP chat requests.
type ChatHandler struct {
	orch      *orchestrator.Orchestrator
	debugHTTP bool
	debugDir  string
}

// NewChatHandler creates a ChatHandler.
func NewChatHandler(orch *orchestrator.Orchestrator, debugHTTP bool, debugDir string) *ChatHandler {
	return &ChatHandler{orch: orch, debugHTTP: debugHTTP, debugDir: debugDir}
}

// HandleChat processes POST /api/chat requests.
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
