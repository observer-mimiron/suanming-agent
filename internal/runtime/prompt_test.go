package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func TestCurrentQuestion(t *testing.T) {
	st := state.NewSession("s1")
	if got := CurrentQuestion(st); got == "" {
		t.Fatal("CurrentQuestion should return a default prompt when session question is empty")
	}

	st.LastUserQuestion = "看看事业"
	if got := CurrentQuestion(st); got != "看看事业" {
		t.Fatalf("CurrentQuestion() = %q, want 看看事业", got)
	}
}

func TestBuildKnowledgeQuery_QimenWithoutBaziUsesQimenTerms(t *testing.T) {
	st := state.NewSession("s2")
	st.LastUserQuestion = "现在适合换工作吗"
	st.QimenResult = map[string]any{
		"value_star": "天辅",
		"value_door": "开门",
		"ju_text":    "阳遁三局",
	}

	builder := NewBuilder(&llm.NoopClient{}, "soft")
	got := builder.BuildKnowledgeQuery(nil, st, "qimen")
	if got == "" {
		t.Fatal("BuildKnowledgeQuery should return qimen terms")
	}
	if want := "奇门遁甲"; got[:len(want)] != want {
		t.Fatalf("BuildKnowledgeQuery() = %q, want prefix %q", got, want)
	}
}
