package sse

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type Writer struct {
	c *gin.Context
}

func NewWriter(c *gin.Context) *Writer {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	return &Writer{c: c}
}

func (w *Writer) Send(eventType string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse marshal: %w", err)
	}
	safeData := strings.ReplaceAll(string(jsonData), "\n", "\\n")
	_, err = fmt.Fprintf(w.c.Writer, "event: %s\ndata: %s\n\n", eventType, safeData)
	w.c.Writer.Flush()
	return err
}
