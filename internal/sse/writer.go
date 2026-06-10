package sse

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type Sender interface {
	Send(eventType string, data any) error
}

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

// DebugWriter wraps Writer and logs events to a file
type DebugWriter struct {
	*Writer
	dbg *os.File
}

func NewWriterWithDebug(c *gin.Context, dbg *os.File) *DebugWriter {
	return &DebugWriter{Writer: NewWriter(c), dbg: dbg}
}

func (dw *DebugWriter) Send(eventType string, data any) error {
	if dw.dbg != nil {
		jd, _ := json.Marshal(data)
		fmt.Fprintf(dw.dbg, "[%s] %s\n", eventType, string(jd))
	}
	return dw.Writer.Send(eventType, data)
}
