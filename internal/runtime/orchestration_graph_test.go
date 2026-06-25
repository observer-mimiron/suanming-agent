package runtime

import "testing"

// TestOrchestrationGraphTopology 验证 Graph 拓扑结构正确编译。
// 不验证行为（行为由现有回归测试覆盖），只验证 Runnable 可编译。
func TestOrchestrationGraphTopology(t *testing.T) {
	r, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Runnable")
	}
}
