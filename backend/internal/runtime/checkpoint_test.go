package runtime

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/compose"
)

// TestFileCheckPointStore_SetGetDelete 验证文件系统 Checkpoint 存储的基础操作。
func TestFileCheckPointStore_SetGetDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := newFileCheckPointStore(dir)
	if err != nil {
		t.Fatalf("newFileCheckPointStore: %v", err)
	}
	ctx := context.Background()

	// Set + Get
	data := []byte("checkpoint-data")
	if err := store.Set(ctx, "cp-1", data); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := store.Get(ctx, "cp-1")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if string(got) != string(data) {
		t.Fatalf("Get data = %q, want %q", got, data)
	}

	// 不存在的 cpID
	_, ok, err = store.Get(ctx, "cp-not-exist")
	if err != nil || ok {
		t.Fatalf("Get missing: ok=%v err=%v", ok, err)
	}

	// Delete
	if err := store.Delete(ctx, "cp-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok, err = store.Get(ctx, "cp-1")
	if err != nil || ok {
		t.Fatalf("Get after Delete: ok=%v err=%v", ok, err)
	}
}

// TestFileCheckPointStore_PersistAcrossInstances 验证文件持久化——新实例能读到旧数据。
func TestFileCheckPointStore_PersistAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	store1, err := newFileCheckPointStore(dir)
	if err != nil {
		t.Fatalf("newFileCheckPointStore 1: %v", err)
	}
	ctx := context.Background()
	if err := store1.Set(ctx, "cp-persist", []byte("persisted")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	store2, err := newFileCheckPointStore(dir)
	if err != nil {
		t.Fatalf("newFileCheckPointStore 2: %v", err)
	}
	got, ok, err := store2.Get(ctx, "cp-persist")
	if err != nil || !ok {
		t.Fatalf("Get from new instance: ok=%v err=%v", ok, err)
	}
	if string(got) != "persisted" {
		t.Fatalf("persisted data = %q, want %q", got, "persisted")
	}
}

// TestOrchestrationGraphCompilesWithCheckPointStore 验证带 cpStore 的 Graph 能编译。
func TestOrchestrationGraphCompilesWithCheckPointStore(t *testing.T) {
	dir := t.TempDir()
	store, err := newFileCheckPointStore(dir)
	if err != nil {
		t.Fatalf("newFileCheckPointStore: %v", err)
	}
	r, err := buildOrchestrationGraph(store)
	if err != nil {
		t.Fatalf("buildOrchestrationGraph(cpStore) failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Runnable")
	}
}

// TestCheckpoint_SolarTimeConfirm 验证 prefill 后中断、用户回复后恢复。
//
// TODO: 需要 mock LLM + specialist registry 才能测 agent 节点。
// 现有测试套件没有 mock agent 的基础设施（observability_test 只测 short_circuit 路径）。
// 参考 eino-agent/eino/compose/checkpoint_test.go:80 的模式补全。
func TestCheckpoint_SolarTimeConfirm(t *testing.T) {
	t.Skip("需要 mock LLM + specialist registry，参考 eino-agent/eino/compose/checkpoint_test.go:80")
	_ = compose.WithCheckPointID // 引用 compose 防止 unused import
}
