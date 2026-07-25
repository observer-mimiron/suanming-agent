package guidance

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func TestRenderGuidance_OfferConsult(t *testing.T) {
	text := RenderGuidance(&state.GuidanceState{DirectiveKind: "offer_consult"}, Context{})
	want := "如果您愿意，我可以按命理咨询的方式继续帮您看。您可以直接告诉我这次最想重点了解什么，比如事业、感情、财运、健康或流年。"
	if text != want {
		t.Fatalf("RenderGuidance() = %q, want %q", text, want)
	}
}

func TestRenderGuidance_ChooseTopic(t *testing.T) {
	text := RenderGuidance(&state.GuidanceState{DirectiveKind: "choose_topic"}, Context{})
	want := "我可以先从几个常见方向来帮您看：事业、感情、财运、健康、流年。您这次最想先聊哪一项？"
	if text != want {
		t.Fatalf("RenderGuidance() = %q, want %q", text, want)
	}
}

func TestRenderGuidance_CollectSlot(t *testing.T) {
	text := RenderGuidance(&state.GuidanceState{
		DirectiveKind: "collect_slot",
		PendingSlot:   "birth_time",
	}, Context{})
	want := "还差一个关键信息：出生时辰。请告诉我大概几点出生；如果只记得范围，也可以直接告诉我。"
	if text != want {
		t.Fatalf("RenderGuidance() = %q, want %q", text, want)
	}
}

func TestRenderGuidance_GuidedFallback(t *testing.T) {
	text := RenderGuidance(&state.GuidanceState{DirectiveKind: "guided_fallback"}, Context{
		ClarificationQuestion: "请问您想了解什么？",
	})
	want := "请问您想了解什么？"
	if text != want {
		t.Fatalf("RenderGuidance() = %q, want %q", text, want)
	}
}

func TestRenderBoundary_AskBaziProfile(t *testing.T) {
	text := RenderBoundary(BoundaryAskBaziProfile, Context{})
	want := "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。"
	if text != want {
		t.Fatalf("RenderBoundary() = %q, want %q", text, want)
	}
}

func TestRender_BoundaryCollectGenderPrompt(t *testing.T) {
	text := Render(Request{Boundary: BoundaryCollectGenderFromBirthTime})
	want := "已经收到您的出生时间，还需要确认一项关键信息：性别是男还是女？\n\n八字排盘的大运顺逆排法因性别而异，请告知后再为您完整分析。"
	if text != want {
		t.Fatalf("Render() = %q, want %q", text, want)
	}
}

func TestRender_BoundaryCollectBirthplacePrompt(t *testing.T) {
	text := Render(Request{Boundary: BoundaryCollectBirthplaceFromProfile})
	want := "已经收到您的出生年月日时和性别，还差一个关键信息：出生地。请告诉我出生城市或地区，我再继续为您排盘分析。"
	if text != want {
		t.Fatalf("Render() = %q, want %q", text, want)
	}
}

func TestRender_BoundaryClarificationFallbackUsesTrimmedQuestion(t *testing.T) {
	text := Render(Request{
		Boundary: BoundaryClarificationFallback,
		Context: Context{
			ClarificationQuestion: "  请问您这次想重点看事业还是感情？\n",
		},
	})
	want := "请问您这次想重点看事业还是感情？"
	if text != want {
		t.Fatalf("Render() = %q, want %q", text, want)
	}
}
