// Package guidance 提供对话引导相关的轻量信号嗅探。
package guidance

import "strings"

// Signal 是面向 guidance 的最小语义信号集合。
//
// 它只覆盖少量高价值词，不承担完整路由职责。
type Signal struct {
	FateAdjacent       bool
	BroadIntent        bool
	TimingFocus        bool
	GuidanceAcceptance bool
	Topic              string
}

// ShouldOfferConsult 判断该消息是否适合先进入 offer_consult 引导。
func (s Signal) ShouldOfferConsult() bool {
	return s.FateAdjacent && !s.TimingFocus
}

// ShouldChooseTopic 判断该消息是否更适合直接进入 choose_topic 引导。
func (s Signal) ShouldChooseTopic() bool {
	return s.BroadIntent && !s.FateAdjacent && !s.TimingFocus
}

// Sniff 从消息中提取极少量 guidance 相关信号。
func Sniff(message string) Signal {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return Signal{}
	}

	signal := Signal{}
	timingScope := containsAny(trimmed, "今天", "今日", "最近", "本月", "这个月", "此刻", "现在", "当前")
	timingIntent := containsAny(trimmed, "运气", "运势", "时机", "宜忌", "适不适合", "要不要", "能不能", "行动", "推进")

	if containsAny(trimmed, "倒霉", "不顺", "运气差", "迷茫", "好衰", "喝凉水", "喝凉水都塞牙", "太背了", "诸事不顺", "运气不好", "最近很背", "点背", "霉运", "水逆") {
		signal.FateAdjacent = true
	}
	if timingScope && timingIntent {
		signal.TimingFocus = true
	}
	if containsAny(trimmed, "可以", "行", "好", "好的", "看看") {
		signal.GuidanceAcceptance = true
	}

	switch {
	case strings.Contains(trimmed, "事业"):
		signal.Topic = "事业"
	case strings.Contains(trimmed, "财运"):
		signal.Topic = "财运"
	case strings.Contains(trimmed, "姻缘"):
		signal.Topic = "感情"
	}

	if signal.FateAdjacent || signal.Topic != "" || strings.Contains(trimmed, "星座") {
		signal.BroadIntent = true
	}

	return signal
}

func containsAny(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}
