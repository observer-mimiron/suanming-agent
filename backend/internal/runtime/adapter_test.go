// This test file belongs to the manager-owned runtime layer.
// It verifies runtime adapter behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"

	"github.com/observer-mimiron/suanming-agent/internal/tools"
	baziCalc "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"
	qimenTools "github.com/observer-mimiron/suanming-agent/internal/tools/qimen"
)

func TestBuildAdaptersFor_SkipsUnregisteredTools(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})
	reg.Register(&baziCalc.YongShenTool{})

	names := []string{"bazi_calc", "ziwei_calc", "yongshen"}
	adapters, err := BuildAdaptersFor(reg, names, nil)
	if err != nil {
		t.Fatalf("BuildAdaptersFor: %v", err)
	}
	if len(adapters) != 2 {
		t.Fatalf("expected 2 adapters (bazi_calc + yongshen), got %d", len(adapters))
	}
}

func TestBuildAdaptersFor_BaziDomainList(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})
	reg.Register(&baziCalc.YongShenTool{})
	reg.Register(&baziCalc.DayunAnalyzer{})
	reg.Register(tools.NewKnowledgeSearchTool(nil))

	baziNames := []string{"bazi_calc", "yongshen", "dayun_analyzer", "knowledge_search"}
	adapters, err := BuildAdaptersFor(reg, baziNames, nil)
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
	adapters, err := BuildAdaptersFor(reg, qimenNames, nil)
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

	adapters, err := BuildAdaptersFor(reg, nil, nil)
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

	adapter, err := newKnowledgeSearchAdapter(reg, nil)
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
		hasBudgetExceeded := strings.Contains(ret, "budget_exceeded")
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
