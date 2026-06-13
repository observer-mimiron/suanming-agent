package tracing

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// FileCollector persists TurnTraces to disk as JSON files.
type FileCollector struct {
	dir string
}

// NewFileCollector creates a FileCollector. Dir is created if missing.
func NewFileCollector(dir string) *FileCollector {
	os.MkdirAll(dir, 0755)
	return &FileCollector{dir: dir}
}

// Save writes a TurnTrace to logs/traces/{date}/{trace_id}.json.
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
