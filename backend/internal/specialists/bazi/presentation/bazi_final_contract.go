// package presentation 包含 Manager 拥有的八字运行时合同。
//
// 本文件校验 final writer 生成的用户可见结构和边界保留情况；
// 不负责生成 Markdown、不重新裁断命盘，也不决定模型 repair。
package presentation

import (
	"fmt"
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

// validateFinalWriterOutput 校验最终报告结构和必须保留的边界提示。
func validateFinalWriterOutput(plan baziAnalysisPlan, state baziCharterState, output string) error {
	output = strings.TrimSpace(output)
	if output == "" {
		return fmt.Errorf("final writer produced empty output")
	}
	if strings.Contains(output, "上游未提供") {
		return fmt.Errorf("final writer output leaked internal missing-field placeholder")
	}
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		if !strings.Contains(output, "只列可复算命盘事实") || !strings.Contains(output, "暂不作主轴") {
			return fmt.Errorf("facts-only final output must expose facts-only synthesis boundary")
		}
		if containsAnyText([]string{output}, []string{"优先按伤官佩印", "层次中等", "中等（保守定位）", "结构承接、压力", "倾向有利"}) {
			return fmt.Errorf("facts-only final output must not expose synthesized reading language")
		}
		return nil
	}
	switch plan.WriterTemplate {
	case "full":
		headings := []string{
			"## 总览结论",
			"## 强弱视角",
			"## 调候视角",
			"## 格局视角",
		}
		if plan.NeedLifetimeDayun {
			headings = append(headings, "## 全程运路")
		}
		headings = append(headings, "## 当前应期")
		if err := validateOrderedHeadings(output, headings); err != nil {
			return err
		}
		if strings.Count(output, "**结论：") < 6 {
			return fmt.Errorf("full writer output must expose bold conclusion lines")
		}
		overviewSection := sectionContent(output, "## 总览结论", "## 强弱视角")
		if overviewSection == "" {
			return fmt.Errorf("full writer output missing 总览结论 section body")
		}
		if err := validateOrderedHeadings(overviewSection, []string{"### 本命总断"}); err != nil {
			return fmt.Errorf("full writer output must preserve 总览结论收束格式: %w", err)
		}
		if !strings.Contains(overviewSection, "**格局评价**") || !strings.Contains(overviewSection, "**判断边界**") {
			return fmt.Errorf("full writer output must expose tier and boundary in 总览结论")
		}
		nextHeading := "## 当前应期"
		if plan.NeedLifetimeDayun {
			nextHeading = "## 全程运路"
		}
		gejuSection := sectionContent(output, "## 格局视角", nextHeading)
		if gejuSection == "" {
			return fmt.Errorf("full writer output missing 格局视角 section body")
		}
		if err := validateOrderedHeadings(gejuSection, []string{
			"**规则口径**",
			"### 格局评价",
			"**判读口径**",
			"断语所限",
		}); err != nil {
			return fmt.Errorf("full writer output must preserve 格局视角 format: %w", err)
		}
		if strings.TrimSpace(state.StaticSynthesis.PatternBasis) != "" && !containsAnyText([]string{gejuSection}, []string{"**依据**"}) {
			return fmt.Errorf("full writer output must expose concise evidence in 格局视角")
		}
		if strings.TrimSpace(state.StaticSynthesis.TierBasis) != "" && !containsAnyText([]string{gejuSection}, []string{"**判定依据**"}) {
			return fmt.Errorf("full writer output must expose a concise tier basis in 格局视角")
		}
	case "topic":
		if err := validateOrderedHeadings(output, []string{
			"## 直接回答",
			"## 命盘依据",
			"## 建议",
		}); err != nil {
			return err
		}
		if strings.Count(output, "**结论：") < 3 {
			return fmt.Errorf("topic writer output must expose bold conclusion lines")
		}
	case "year":
		if err := validateOrderedHeadings(output, []string{
			"## 年度判断",
			"## 作用机制",
			"## 重点应期",
			"## 建议",
		}); err != nil {
			return err
		}
		if strings.Count(output, "**结论：") < 1 {
			return fmt.Errorf("year writer output must expose bold conclusion line")
		}
	}
	if counterEvidence := strings.TrimSpace(state.StaticSynthesis.CounterEvidence); counterEvidence != "" &&
		!strings.Contains(output, counterEvidence) &&
		!containsAnyText([]string{output}, []string{"受限", "限制", "不足", "难以"}) {
		return fmt.Errorf("final writer output dropped limitation signals")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") &&
		!containsAnyText([]string{output}, []string{"机会伴随强变动", "吉中带险", "不宜激进"}) {
		return fmt.Errorf("final writer output dropped dynamic volatility constraint")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") &&
		!containsAnyText([]string{output}, []string{"吉中有阻", "有阻", "并存"}) {
		return fmt.Errorf("final writer output dropped mixed dynamic constraint")
	}
	if state.DynamicSynthesis.WindowLevel == "窗口年" && state.DynamicSynthesis.WordingCap != "封顶" &&
		containsAnyText([]string{output}, []string{"关键翻身年", "一飞冲天", "彻底翻身"}) {
		return fmt.Errorf("final writer output overstates a window year")
	}
	return nil
}

// ValidateFinalReplyForState checks a rendered Bazi reply at the adapter boundary.
func ValidateFinalReplyForState(plan bazidomain.AnalysisPlan, state bazidomain.CharterState, output string) error {
	return validateFinalWriterOutput(plan, state, output)
}

// validateOrderedHeadings 强制 renderer 要求的标题顺序。
func validateOrderedHeadings(output string, headings []string) error {
	last := -1
	for _, heading := range headings {
		index := strings.Index(output, heading)
		if index < 0 {
			return fmt.Errorf("final writer output missing heading: %s", heading)
		}
		if index <= last {
			return fmt.Errorf("final writer output has heading out of order: %s", heading)
		}
		last = index
	}
	return nil
}

// sectionContent 返回两个标题之间的正文。
func sectionContent(output, heading, nextHeading string) string {
	start := strings.Index(output, heading)
	if start < 0 {
		return ""
	}
	start += len(heading)
	end := len(output)
	if nextHeading != "" {
		if next := strings.Index(output[start:], nextHeading); next >= 0 {
			end = start + next
		}
	}
	return strings.TrimSpace(output[start:end])
}
