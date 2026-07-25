// Package tracing 暂与 tracing.go 共享包注释，本文件提供 Gin 中间件，为 HTTP 请求创建追踪跨度。

package tracing

import "github.com/gin-gonic/gin"

// Middleware 创建 Gin 中间件，为每个 HTTP 请求自动创建追踪。
func Middleware(tracer Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// /api/chat 在 orchestrator 内部自行创建轮次级别的追踪。
		// 此处跳过请求级别追踪，以保证一个聊天请求对应一个轮次追踪的不变约束。
		if c.FullPath() == "/api/chat" {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		ctx, trace := tracer.StartTrace(ctx, c.Request.Method+" "+c.FullPath())
		defer trace.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
