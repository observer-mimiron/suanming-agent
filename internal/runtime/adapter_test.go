package runtime

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/tools"
	baziCalc "github.com/wikiglobal/suanming-agent/internal/tools/bazi"
)

func TestBuildAdaptersFor_SkipsUnregisteredTools(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&baziCalc.CalcTool{})
	reg.Register(&tools.YongShenTool{})

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
	reg.Register(&tools.YongShenTool{})
	reg.Register(&tools.DayunAnalyzer{})
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
	reg.Register(&tools.QimenTool{})
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
