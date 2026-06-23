package intent

import "testing"

func TestContainsBirthInfo(t *testing.T) {
	cases := []string{
		"1990年5月20日早上8点出生",
		"我出生于农历1990年正月十五",
		"2000-03-15生的",
		"1990/05/20",
		"1990年5月",
	}
	for _, c := range cases {
		if !ContainsBirthInfo(c) {
			t.Errorf("ContainsBirthInfo(%q) = false, want true", c)
		}
	}
}

func TestContainsBirthInfo_False(t *testing.T) {
	cases := []string{
		"最近好倒霉",
		"帮我算算运势",
		"我今年28岁",
		"生日快乐",
	}
	for _, c := range cases {
		if ContainsBirthInfo(c) {
			t.Errorf("ContainsBirthInfo(%q) = true, want false", c)
		}
	}
}

func TestMentionsBaziMethod(t *testing.T) {
	if !MentionsBaziMethod("看看八字") {
		t.Error("MentionsBaziMethod(看看八字) = false, want true")
	}
	if !MentionsBaziMethod("帮我排个八字") {
		t.Error("MentionsBaziMethod(帮我排个八字) = false, want true")
	}
	if MentionsBaziMethod("奇门遁甲") {
		t.Error("MentionsBaziMethod(奇门遁甲) = true, want false")
	}
}

func TestMentionsZiweiMethod(t *testing.T) {
	// 正名 + 别字 + 术语 + 星盘
	cases := []string{"紫微斗数", "紫薇斗数", "看看斗数", "排个星盘"}
	for _, c := range cases {
		if !MentionsZiweiMethod(c) {
			t.Errorf("MentionsZiweiMethod(%q) = false, want true", c)
		}
	}
	if MentionsZiweiMethod("八字排盘") {
		t.Error("MentionsZiweiMethod(八字排盘) = true, want false")
	}
}

func TestMentionsQimenMethod(t *testing.T) {
	cases := []string{"奇门遁甲", "奇门排盘", "遁甲"}
	for _, c := range cases {
		if !MentionsQimenMethod(c) {
			t.Errorf("MentionsQimenMethod(%q) = false, want true", c)
		}
	}
	if MentionsQimenMethod("八字排盘") {
		t.Error("MentionsQimenMethod(八字排盘) = true, want false")
	}
}

func TestContainsExplicitDivinationAction(t *testing.T) {
	cases := []string{
		"帮我算一下",
		"帮我看一下八字",
		"帮我排盘",
		"帮我看看运势",
		"算一下",
		"看一下",
	}
	for _, c := range cases {
		if !ContainsExplicitDivinationAction(c) {
			t.Errorf("ContainsExplicitDivinationAction(%q) = false, want true", c)
		}
	}
}

func TestContainsExplicitDivinationAction_False(t *testing.T) {
	cases := []string{
		"最近运气不好",
		"今天怎么样",
		"可以，看看事业",
	}
	for _, c := range cases {
		if ContainsExplicitDivinationAction(c) {
			t.Errorf("ContainsExplicitDivinationAction(%q) = true, want false", c)
		}
	}
}

func TestHasTimingFocus(t *testing.T) {
	// scope + intent 双命中
	cases := []string{
		"今天运气怎么样",
		"本月运势如何",
		"最近运势怎么样",
		"当前适不适合行动",
		"今天适不适合推进项目",
	}
	for _, c := range cases {
		if !HasTimingFocus(c) {
			t.Errorf("HasTimingFocus(%q) = false, want true", c)
		}
	}
}

func TestHasTimingFocus_False(t *testing.T) {
	// 只有 scope 没 intent、或只有 intent 没 scope
	cases := []string{
		"运势如何",       // 没 scope
		"时机成熟了吗",     // 没 scope
		"今天开会",       // scope only, not timing intent
		"最近好倒霉",     // fate-adjacent, not timing
		"最近很背",       // fate-adjacent, not timing
	}
	for _, c := range cases {
		if HasTimingFocus(c) {
			t.Errorf("HasTimingFocus(%q) = true, want false", c)
		}
	}
}

func TestContainsTimingKeyword(t *testing.T) {
	// 宽松关键词命中 — 对标 approved_route.go 里的 mentionsTimingKeyword
	cases := []string{
		"运势怎么样",
		"看看时机",
		"择日",
		"择日吧",
		"今天有什么讲究",
		"最近怎么样",
		"本月运程",
		"今年有什么变化",
		"当下如何",
		"近期情况",
	}
	for _, c := range cases {
		if !ContainsTimingKeyword(c) {
			t.Errorf("ContainsTimingKeyword(%q) = false, want true", c)
		}
	}
}

func TestContainsTimingKeyword_False(t *testing.T) {
	cases := []string{
		"帮我算一下八字",
		"看看事业",
		"感情怎么样",
	}
	for _, c := range cases {
		if ContainsTimingKeyword(c) {
			t.Errorf("ContainsTimingKeyword(%q) = true, want false", c)
		}
	}
}
