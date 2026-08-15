// This test file belongs to the knowledge MCP client layer.
// It verifies external client behavior and protects the related contract from regressions.
// It calls the knowledge service; interpretation remains in runtime and specialists.
package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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

func TestSearchKnowledgeUsesRetrieveEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if q := r.URL.Query().Get("q"); q != "子平真诠 格局" {
			t.Fatalf("query = %q, want %q", q, "子平真诠 格局")
		}
		if limit := r.URL.Query().Get("limit"); limit != "3" {
			t.Fatalf("limit = %q, want %q", limit, "3")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"slug":"ziping-geju","title":"子平真诠 论格局","snippet":"格局以月令为主"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	passages, err := c.SearchKnowledge("子平真诠 格局", 3)
	if err != nil {
		t.Fatalf("SearchKnowledge error: %v", err)
	}
	if gotPath != "/api/wiki/retrieve" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/wiki/retrieve")
	}
	if len(passages) != 1 {
		t.Fatalf("len(passages) = %d, want 1", len(passages))
	}
	if passages[0].Content != "格局以月令为主" {
		t.Fatalf("content = %q", passages[0].Content)
	}
	if passages[0].Source != "knowledge://ziping-geju (子平真诠 论格局)" {
		t.Fatalf("source = %q", passages[0].Source)
	}
}

func TestSearchFailureKind_ClassifiesTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	c.client.Timeout = 10 * time.Millisecond
	_, err := c.SearchKnowledge("子平真诠 破格", 3)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := SearchFailureKind(err); got != SearchFailureTimeout {
		t.Fatalf("failure kind = %q, want %q", got, SearchFailureTimeout)
	}
}
