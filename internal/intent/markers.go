// Package intent 提供面向用户消息的共享 lexical detector。
//
// 本包是 birth-info / explicit method / explicit action / timing 检测的唯一 truth source，
// 供 supervisor 和 runtime 共同使用，消除重复实现导致的 split-brain。
package intent

import (
	"regexp"
	"strings"
)

// birthTimeRe 匹配中文消息中可能包含出生时间的模式。
// 覆盖：中文年月格式(2020年3月)、数字日期格式(2020-03)、农历/阴历关键词、农历月份别名。
var birthTimeRe = regexp.MustCompile(`\d{4}\s*年.*\d{1,2}\s*月|\d{4}[-/]\d{1,2}|农历|阴历|正月|腊月`)

// method regex — 使用 approved_route.go 的更完备版本。
// ziweiMethodRe 额外覆盖 紫薇 (别字) 和 星盘，比 guidance_gate.go 当前的关键词匹配更全面。
var ziweiMethodRe = regexp.MustCompile(`紫微|紫薇|斗数|星盘`)
var qimenMethodRe = regexp.MustCompile(`奇门|遁甲`)
var baziMethodRe = regexp.MustCompile(`八字`)

// explicitActionPatterns 是 guidance_gate.go 中 containsExplicitAction 的移植。
var explicitActionPatterns = []string{"帮我算", "帮我看", "排盘", "帮我看看", "算一下", "看一下"}

// timingKeywords 是 approved_route.go 中 mentionsTimingKeyword 的移植。
var timingKeywords = []string{"运势", "时机", "择日", "今天", "最近", "本月", "今年", "当下", "近期"}

// timingScopes 是 guidance.Sniff() 中 TimingFocus 的 scope 部分。
var timingScopes = []string{"今天", "今日", "最近", "本月", "这个月", "此刻", "现在", "当前"}

// timingIntents 是 guidance.Sniff() 中 TimingFocus 的 intent 部分。
var timingIntents = []string{"运气", "运势", "时机", "宜忌", "适不适合", "要不要", "能不能", "行动", "推进"}

// ContainsBirthInfo 快速检测用户消息是否可能包含出生时间信息。
func ContainsBirthInfo(msg string) bool {
	return birthTimeRe.MatchString(msg)
}

// MentionsBaziMethod 检查消息是否显式提及八字。
func MentionsBaziMethod(msg string) bool {
	return baziMethodRe.MatchString(msg)
}

// MentionsZiweiMethod 检查消息是否显式提及紫微斗数（含 紫薇 别字和 星盘）。
func MentionsZiweiMethod(msg string) bool {
	return ziweiMethodRe.MatchString(msg)
}

// MentionsQimenMethod 检查消息是否显式提及奇门遁甲。
func MentionsQimenMethod(msg string) bool {
	return qimenMethodRe.MatchString(msg)
}

// ContainsExplicitDivinationAction 检查消息是否包含显式算命请求。
func ContainsExplicitDivinationAction(msg string) bool {
	for _, a := range explicitActionPatterns {
		if strings.Contains(msg, a) {
			return true
		}
	}
	return false
}

// HasTimingFocus 用 scope + intent 双条件判断消息是否聚焦当前时机。
//
// 实现移植自 guidance.Sniff() 的 TimingFocus 字段：
// 必须同时命中时间范围词 AND 时机意图词。
// 单命中 scope（如 "今天天气"）或单命中 intent（如 "运势如何"）不算。
func HasTimingFocus(msg string) bool {
	hasScope := false
	for _, s := range timingScopes {
		if strings.Contains(msg, s) {
			hasScope = true
			break
		}
	}
	if !hasScope {
		return false
	}
	for _, i := range timingIntents {
		if strings.Contains(msg, i) {
			return true
		}
	}
	return false
}

// ContainsTimingKeyword 用关键词宽松匹配判断消息是否涉及时间/运势话题。
//
// 实现移植自 approved_route.go 的 mentionsTimingKeyword：
// 命中任意 timing 关键词即返回 true，用于触发 qimen supplement mode。
func ContainsTimingKeyword(msg string) bool {
	for _, kw := range timingKeywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
