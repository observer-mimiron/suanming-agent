package mcp

import "testing"

// TestGetGraphContract 验证 GetGraph 返回 nodes 和 edges 的契约。
// 需要知识库服务可用（localhost:3100）；不可用时跳过。
func TestGetGraphContract(t *testing.T) {
	c := NewClient("http://localhost:3100")
	nodes, edges, err := c.GetGraph()
	if err != nil {
		t.Skipf("knowledge base not available, skipping integration test: %v", err)
	}
	if len(nodes) == 0 {
		t.Error("expected non-empty nodes")
	}
	if len(edges) == 0 {
		t.Error("expected non-empty edges")
	}
	for _, n := range nodes {
		if n.ID == "" || n.Label == "" {
			t.Errorf("node missing required fields: %+v", n)
		}
	}
	for _, e := range edges {
		if e.Source == "" || e.Target == "" {
			t.Errorf("edge missing required fields: %+v", e)
		}
	}
}
