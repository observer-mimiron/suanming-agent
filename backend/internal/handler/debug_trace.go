// Package handler provides HTTP request handlers.
//
// This file owns the local-only raw trace debug endpoint. It reads persisted
// TurnTrace JSON from logs/traces and does not participate in chat execution.
package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var traceIDPattern = regexp.MustCompile("^trc_[A-Za-z0-9_-]+$")

// DebugTraceHandler serves persisted raw TurnTrace JSON for local debugging.
type DebugTraceHandler struct {
	traceDir string
}

// NewDebugTraceHandler creates a read-only handler for logs/traces.
func NewDebugTraceHandler(traceDir string) *DebugTraceHandler {
	return &DebugTraceHandler{traceDir: traceDir}
}

// HandleGetTrace returns the full persisted TurnTrace JSON for a trace_id.
func (h *DebugTraceHandler) HandleGetTrace(c *gin.Context) {
	if h == nil || strings.TrimSpace(h.traceDir) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "trace debug endpoint not configured"})
		return
	}
	traceID := strings.TrimSpace(c.Param("traceID"))
	if !traceIDPattern.MatchString(traceID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trace_id"})
		return
	}

	path, ok := findTraceFile(h.traceDir, traceID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "trace not found"})
		return
	}
	c.File(path)
}

// findTraceFile resolves logs/traces/{date}/{trace_id}.json without exposing paths.
func findTraceFile(traceDir, traceID string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(traceDir, "*", traceID+".json"))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	for _, candidate := range matches {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}
