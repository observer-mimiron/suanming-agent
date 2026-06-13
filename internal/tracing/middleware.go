package tracing

import "github.com/gin-gonic/gin"

// Middleware creates a Gin middleware that starts a trace for each HTTP request.
func Middleware(tracer Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// /api/chat creates its own turn-level trace inside the orchestrator.
		// Skip request-level tracing here to preserve the invariant:
		// one chat request -> one turn trace.
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
