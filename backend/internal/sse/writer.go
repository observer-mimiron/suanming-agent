// Package sse 实现 Server-Sent Events 推送，提供 Sender 接口和 Gin 上下文绑定的 Writer。
// 支持 6 种事件类型：thinking / tool_call / component / text / error / done。

package sse

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Sender 是 SSE 事件推送的抽象接口。
type Sender interface {
	Send(eventType string, data any) error
}

type Writer struct {
	c *gin.Context
}

// NewWriter 创建一个 SSE Writer，设置 Gin 响应的 SSE 相关头信息。
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

// DebugWriter 包装 Writer 并将事件同时记录到日志文件。
type DebugWriter struct {
	*Writer
	dbg *os.File
}

// NewWriterWithDebug 创建一个 DebugWriter，将 SSE 事件同时写入日志文件。
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
