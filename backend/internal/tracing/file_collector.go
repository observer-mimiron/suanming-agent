// Package tracing 暂与 tracing.go 共享包注释，本文件提供 Trace 的文件持久化收集器。

package tracing

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// FileCollector 将 TurnTrace 持久化为磁盘上的 JSON 文件。
type FileCollector struct {
	dir string
}

// NewFileCollector 创建一个 FileCollector，如果目录不存在则自动创建。
func NewFileCollector(dir string) *FileCollector {
	os.MkdirAll(dir, 0755)
	return &FileCollector{dir: dir}
}

// Save 将 TurnTrace 写入 logs/traces/{date}/{trace_id}.json 文件。
func (fc *FileCollector) Save(t *TurnTrace) error {
	if t == nil {
		return fmt.Errorf("nil trace")
	}
	dateDir := fc.dir + "/" + t.StartedAt.Format("2006-01-02")
	os.MkdirAll(dateDir, 0755)
	path := dateDir + "/" + t.TraceID + ".json"

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(t); err != nil {
		return err
	}
	log.Printf("[tracing] trace saved: %s", path)
	return nil
}
