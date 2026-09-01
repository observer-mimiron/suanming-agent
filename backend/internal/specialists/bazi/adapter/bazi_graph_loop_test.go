package adapter

import (
	"context"
	"testing"
)

func TestBaziRecoverFactsNodeMarksMissingCurrentPeriod(t *testing.T) {
	in := &baziInternalGraphState{Phase: baziPhaseDynamic}

	if _, err := (&Executor{}).baziRecoverFactsNode(context.Background(), in); err != nil {
		t.Fatalf("baziRecoverFactsNode() error = %v", err)
	}
	if in.TerminationReason != "current_period_unavailable" {
		t.Fatalf("termination reason = %q, want current_period_unavailable", in.TerminationReason)
	}
}

func TestDeterministicBaziAnalysisPlanUsesStableTemplates(t *testing.T) {
	full := deterministicBaziAnalysisPlan("男，1990年5月20日巳时，北京，分析八字")
	if full.WriterTemplate != "full" || !full.NeedLifetimeDayun || !full.NeedDynamic {
		t.Fatalf("full chart plan = %+v", full)
	}
	topic := deterministicBaziAnalysisPlan("我的财运怎么样")
	if topic.WriterTemplate != "topic" || topic.NeedLifetimeDayun || topic.NeedDynamic {
		t.Fatalf("topic plan = %+v", topic)
	}
	year := deterministicBaziAnalysisPlan("今年财运如何")
	if year.WriterTemplate != "year" || year.RetrievalStage != "dynamic" || !year.NeedDynamic || year.NeedLifetimeDayun {
		t.Fatalf("year plan = %+v", year)
	}
}
