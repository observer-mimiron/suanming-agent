package handler

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// SessionSegment 是前端恢复 assistant 结构化内容时消费的最小片段合同。
type SessionSegment struct {
	Type          string         `json:"type"`
	Content       string         `json:"content,omitempty"`
	Text          string         `json:"text,omitempty"`
	Agent         string         `json:"agent,omitempty"`
	Tool          string         `json:"tool,omitempty"`
	Params        map[string]any `json:"params,omitempty"`
	Result        string         `json:"result,omitempty"`
	ComponentType string         `json:"component_type,omitempty"`
	Payload       any            `json:"payload,omitempty"`
	Message       string         `json:"message,omitempty"`
}

type sessionDebugEntry struct {
	SessionID string         `json:"session_id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}

func loadRecentAssistantSegments(debugDir string, st *state.SessionState, limit int) []SessionSegment {
	if strings.TrimSpace(debugDir) == "" || st == nil || strings.TrimSpace(st.SessionID) == "" || limit <= 0 {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(debugDir, "*.jsonl"))
	if err != nil || len(files) == 0 {
		return nil
	}
	sort.Strings(files)

	for i := len(files) - 1; i >= 0; i-- {
		segments, ok := loadSegmentsFromDebugFile(files[i], st.SessionID, limit)
		if !ok || len(segments) == 0 {
			continue
		}
		return segments
	}
	return nil
}

func loadSegmentsFromDebugFile(path, sessionID string, limit int) ([]SessionSegment, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var segments []SessionSegment
	var sawDone bool
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry sessionDebugEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.SessionID != sessionID {
			continue
		}

		if entry.EventType == "done" {
			sawDone = true
		}
		if seg, ok := mapDebugEntryToSegment(entry); ok {
			segments = append(segments, seg)
		}
	}
	if !sawDone || len(segments) == 0 {
		return nil, false
	}
	return trimSessionSegments(segments, limit), true
}

func trimSessionSegments(segments []SessionSegment, limit int) []SessionSegment {
	if limit <= 0 || len(segments) <= limit {
		return segments
	}
	return append([]SessionSegment(nil), segments[len(segments)-limit:]...)
}

func mapDebugEntryToSegment(entry sessionDebugEntry) (SessionSegment, bool) {
	payload := entry.Payload
	switch entry.EventType {
	case "text":
		content, _ := payload["content"].(string)
		if strings.TrimSpace(content) == "" {
			return SessionSegment{}, false
		}
		return SessionSegment{Type: "text", Content: content}, true
	case "thinking":
		text, _ := payload["text"].(string)
		if strings.TrimSpace(text) == "" {
			return SessionSegment{}, false
		}
		agent, _ := payload["agent"].(string)
		return SessionSegment{Type: "thinking", Text: text, Agent: agent}, true
	case "tool_call":
		tool, _ := payload["tool"].(string)
		result, _ := payload["result"].(string)
		params, _ := payload["params"].(map[string]any)
		return SessionSegment{Type: "tool_call", Tool: tool, Params: params, Result: result}, true
	case "component":
		componentType, _ := payload["type"].(string)
		if strings.TrimSpace(componentType) == "" {
			return SessionSegment{}, false
		}
		return SessionSegment{
			Type:          "component",
			ComponentType: componentType,
			Payload:       payload["payload"],
		}, true
	case "error":
		message, _ := payload["message"].(string)
		if strings.TrimSpace(message) == "" {
			return SessionSegment{}, false
		}
		return SessionSegment{Type: "error", Message: message}, true
	default:
		return SessionSegment{}, false
	}
}
