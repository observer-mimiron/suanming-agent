// This file belongs to the server bootstrap layer.
// It owns the command entrypoint for this package.
// It wires dependencies and routes; domain decisions stay under internal packages.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/container"
)

func main() {
	c := container.BuildContainer()
	for _, line := range c.TraceStartupLines() {
		log.Println(line)
	}

	// Register debug endpoints in development mode
	if c.Config.DebugHTTP {
		debugDir := c.DebugDir
		os.MkdirAll(debugDir, 0755)
		c.Router.GET("/api/debug", func(ctx *gin.Context) {
			files, _ := os.ReadDir(debugDir)
			var list []string
			for _, f := range files {
				list = append(list, f.Name())
			}
			ctx.JSON(200, gin.H{"sessions": list})
		})
		c.Router.GET("/api/debug/files/:file", func(ctx *gin.Context) {
			ctx.File(debugDir + "/" + ctx.Param("file"))
		})
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		if c.Shutdown != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := c.Shutdown(ctx); err != nil {
				log.Printf("shutdown tracing exporter: %v", err)
			}
		}
		os.Exit(0)
	}()

	log.Println("server starting on " + c.Config.ListenAddr)
	log.Fatal(c.Router.Run(c.Config.ListenAddr))
}
