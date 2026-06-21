package guidance

import "testing"

func TestSniff_FateAdjacentSignalsOfferConsult(t *testing.T) {
	got := Sniff("最近好倒霉，感觉一直不顺")
	if !got.FateAdjacent {
		t.Fatal("FateAdjacent = false, want true")
	}
	if !got.ShouldOfferConsult() {
		t.Fatal("ShouldOfferConsult() = false, want true")
	}
}

func TestSniff_TodayLuckIsTimingFocusNotOfferConsult(t *testing.T) {
	got := Sniff("今天运气怎么样")
	if !got.TimingFocus {
		t.Fatal("TimingFocus = false, want true")
	}
	if got.ShouldOfferConsult() {
		t.Fatal("ShouldOfferConsult() = true, want false")
	}
}

func TestSniff_TimingStyleQuestionsDoNotBecomeOfferConsult(t *testing.T) {
	cases := []string{
		"最近运气怎么样",
		"本月运势如何",
		"此刻适不适合行动",
	}

	for _, message := range cases {
		t.Run(message, func(t *testing.T) {
			got := Sniff(message)
			if !got.TimingFocus {
				t.Fatalf("TimingFocus = false, want true for %q", message)
			}
			if got.ShouldOfferConsult() {
				t.Fatalf("ShouldOfferConsult() = true, want false for %q", message)
			}
		})
	}
}

func TestSniff_TransferLuckQuestionDoesNotBecomeOfferConsult(t *testing.T) {
	got := Sniff("什么时候能转运")
	if got.ShouldOfferConsult() {
		t.Fatal("ShouldOfferConsult() = true, want false")
	}
}

func TestSniff_DetectsGuidanceAcceptanceAndTopic(t *testing.T) {
	got := Sniff("可以，先看看财运")
	if !got.GuidanceAcceptance {
		t.Fatal("GuidanceAcceptance = false, want true")
	}
	if !got.BroadIntent {
		t.Fatal("BroadIntent = false, want true")
	}
	if got.Topic != "财运" {
		t.Fatalf("Topic = %q, want 财运", got.Topic)
	}
}

func TestSniff_BroadIntentWithoutFateAdjacent(t *testing.T) {
	got := Sniff("我想看看星座和姻缘")
	if !got.BroadIntent {
		t.Fatal("BroadIntent = false, want true")
	}
	if got.ShouldOfferConsult() {
		t.Fatal("ShouldOfferConsult() = true, want false")
	}
	if !got.ShouldChooseTopic() {
		t.Fatal("ShouldChooseTopic() = false, want true")
	}
}

func TestSniff_ExplicitMethodNotGuidanceSignal(t *testing.T) {
	msgs := []string{"用奇门看看", "八字排盘", "紫微斗数分析"}
	for _, msg := range msgs {
		got := Sniff(msg)
		if got.ShouldOfferConsult() {
			t.Fatalf("ShouldOfferConsult() = true for %q, want false", msg)
		}
		if got.ShouldChooseTopic() {
			t.Fatalf("ShouldChooseTopic() = true for %q, want false", msg)
		}
	}
}

func TestSniff_MangranCountsAsFateAdjacent(t *testing.T) {
	got := Sniff("最近有点迷茫，想看看方向")
	if !got.FateAdjacent {
		t.Fatal("FateAdjacent = false, want true")
	}
	if !got.ShouldOfferConsult() {
		t.Fatal("ShouldOfferConsult() = false, want true")
	}
}
