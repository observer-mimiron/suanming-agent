package runtime

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"

	"github.com/wikiglobal/suanming-agent/internal/tools"
	baziCalc "github.com/wikiglobal/suanming-agent/internal/tools/bazi"
	qimenTools "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
)

func TestBuildAdaptersFor_SkipsUnregisteredTools(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})
	reg.Register(&baziCalc.YongShenTool{})

	names := []string{"bazi_calc", "ziwei_calc", "yongshen"}
	adapters, err := BuildAdaptersFor(reg, names)
	if err != nil {
		t.Fatalf("BuildAdaptersFor: %v", err)
	}
	if len(adapters) != 2 {
		t.Fatalf("expected 2 adapters (bazi_calc + yongshen), got %d", len(adapters))
	}
	_ = context.Background()
}

func TestBuildAdaptersFor_BaziDomainList(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})
	reg.Register(&baziCalc.YongShenTool{})
	reg.Register(&baziCalc.DayunAnalyzer{})
	reg.Register(tools.NewKnowledgeSearchTool(nil))

	baziNames := []string{"bazi_calc", "yongshen", "dayun_analyzer", "knowledge_search"}
	adapters, err := BuildAdaptersFor(reg, baziNames)
	if err != nil {
		t.Fatalf("BuildAdaptersFor bazi: %v", err)
	}
	if len(adapters) != 4 {
		t.Fatalf("expected 4 adapters for bazi, got %d", len(adapters))
	}
}

func TestBuildAdaptersFor_QimenDomainList(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&qimenTools.Tool{})
	reg.Register(tools.NewKnowledgeSearchTool(nil))

	qimenNames := []string{"qimen_dunjia", "knowledge_search"}
	adapters, err := BuildAdaptersFor(reg, qimenNames)
	if err != nil {
		t.Fatalf("BuildAdaptersFor qimen: %v", err)
	}
	if len(adapters) != 2 {
		t.Fatalf("expected 2 adapters for qimen, got %d", len(adapters))
	}
}

func TestBuildAdaptersFor_EmptyList(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})

	adapters, err := BuildAdaptersFor(reg, nil)
	if err != nil {
		t.Fatalf("BuildAdaptersFor empty: %v", err)
	}
	if len(adapters) != 0 {
		t.Fatalf("expected 0 adapters for empty list, got %d", len(adapters))
	}
}

func TestKnowledgeSearchAdapter_BudgetEnforced(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(tools.NewKnowledgeSearchToolFromEnv())

	adapter, err := newKnowledgeSearchAdapter(reg)
	if err != nil {
		t.Fatal(err)
	}
	inv, ok := adapter.(tool.InvokableTool)
	if !ok {
		t.Fatal("adapter does not implement InvokableTool")
	}
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		ret, err := inv.InvokableRun(ctx, `{"query":"测试","top_k":1}`)
		if err != nil {
			t.Logf("call %d err: %v, result: %s", i+1, err, ret)
			continue
		}
		hasBudgetExceeded := containsStr(ret, "budget_exceeded")
		if i < 3 {
			if hasBudgetExceeded {
				t.Errorf("call %d should succeed, got budget_exceeded", i+1)
			}
		} else {
			if !hasBudgetExceeded {
				t.Errorf("call %d should be blocked by budget, got: %s", i+1, ret)
			}
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
