// This file belongs to the HTTP and SSE adapter layer.
// It owns session API behavior for this package.
// It adapts HTTP/SSE; route approval and domain execution stay below this layer.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// sessionPeekStore captures the read-only store capability needed by the
// session snapshot endpoint without widening the write path surface.
type sessionPeekStore interface {
	Peek(id string) (*state.SessionState, bool)
}

// SessionSnapshot 是前端恢复会话时消费的最小只读合同。
type SessionSnapshot struct {
	SessionID string                       `json:"session_id"`
	Messages  []SessionMessage             `json:"messages"`
	Segments  []SessionSegment             `json:"segments,omitempty"`
	LastInput contracts.LastInputState     `json:"last_input,omitempty"`
	Execution *contracts.ExecutionSnapshot `json:"execution,omitempty"`
}

// SessionMessage 是前端恢复消息列表所需的最小历史消息形状。
type SessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SessionHandler 提供会话快照读取接口，供前端恢复 session 使用。
type SessionHandler struct {
	store    sessionPeekStore
	debugDir string
}

// NewSessionHandler 创建只读会话处理器。
func NewSessionHandler(store sessionPeekStore, debugDir string) *SessionHandler {
	return &SessionHandler{store: store, debugDir: debugDir}
}

// HandleGetSession 处理 GET /api/session/:sessionID，请求既有会话快照。
func (h *SessionHandler) HandleGetSession(c *gin.Context) {
	if h == nil || h.store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session store not configured"})
		return
	}

	sessionID, err := resolveSessionID(c.Param("sessionID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	st, ok := h.store.Peek(sessionID)
	if !ok || st == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, buildSessionSnapshot(h.debugDir, st))
}

func buildSessionSnapshot(debugDir string, st *state.SessionState) SessionSnapshot {
	snapshot := SessionSnapshot{
		SessionID: st.SessionID,
		Messages:  make([]SessionMessage, 0, len(st.RecentTurns)),
		LastInput: st.LastInput,
	}
	for _, turn := range st.RecentTurns {
		if turn.Role == "" || turn.Content == "" {
			continue
		}
		snapshot.Messages = append(snapshot.Messages, SessionMessage{
			Role:    turn.Role,
			Content: turn.Content,
		})
	}
	if st.Execution.HasSignal() {
		execution := st.Execution
		snapshot.Execution = &execution
	}
	snapshot.Segments = loadRecentAssistantSegments(debugDir, st, 48)
	return snapshot
}
