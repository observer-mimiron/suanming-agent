// Package runtime 包含 Manager 拥有的八字最终渲染。
//
// 本文件负责 Markdown 写入、展示文本清理和内部路径过滤；
// 不负责决定命理结论、选择路由或修复上游结构化结果。
package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

var baziInternalReferencePath = regexp.MustCompile(`(?:dayun\[[0-9]+\](?:\.[A-Za-z0-9_]+)+|(?:liunian|yongshen|evidence_quality)(?:\.[A-Za-z0-9_]+)+|\b(?:support_score|pressure_score|tier_status)\b)`)

func ensureSteps(src []string, fallback []string) []string {
	if len(filterNonEmpty(src)) > 0 {
		return filterNonEmpty(src)
	}
	return filterNonEmpty(fallback)
}

func filterNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func joinOrDefault(items []string, fallback string) string {
	items = uniqueText(items)
	if len(items) == 0 {
		return cleanUserVisibleText(fallback)
	}
	return cleanUserVisibleText(strings.Join(items, "；"))
}

// conciseDisplayText trims verbose model-safe boundary prose into one readable
// display sentence while preserving the upstream verdict's first-order meaning.

// conciseDisplayText trims verbose model-safe boundary prose into one readable
// display sentence while preserving the upstream verdict's first-order meaning.
func conciseDisplayText(text string, maxRunes int) string {
	text = cleanUserVisibleText(text)
	if text == "" {
		return ""
	}
	clauses := splitDisplayClauses(text)
	if len(clauses) == 0 {
		return text
	}
	out := clauses[0]
	if len([]rune(out)) < maxRunes/2 && len(clauses) > 1 {
		out = strings.TrimRight(out, "。；") + "；" + clauses[1]
	}
	if maxRunes > 0 && len([]rune(out)) > maxRunes {
		runes := []rune(out)
		out = strings.TrimRight(string(runes[:maxRunes]), "，、；。 ") + "。"
	}
	if !strings.HasSuffix(out, "。") && !strings.HasSuffix(out, "！") && !strings.HasSuffix(out, "？") {
		out += "。"
	}
	return out
}

// splitDisplayClauses splits only on Chinese sentence-level separators so the
// renderer can cap repetition without parsing or re-adjudicating the reading.

// splitDisplayClauses splits only on Chinese sentence-level separators so the
// renderer can cap repetition without parsing or re-adjudicating the reading.
func splitDisplayClauses(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '。', '；', ';', '\n':
			return true
		default:
			return false
		}
	})
	return filterNonEmpty(fields)
}

// uniqueText removes exact repeated fallback and risk lines while preserving
// their first occurrence and source order. It does not merge similar evidence.

// uniqueText removes exact repeated fallback and risk lines while preserving
// their first occurrence and source order. It does not merge similar evidence.
func uniqueText(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return cleanUserVisibleText(fallback)
	}
	return cleanUserVisibleText(strings.TrimSpace(value))
}

func firstDisplayText(values ...string) string {
	for _, value := range values {
		if text := cleanUserVisibleText(value); text != "" {
			return text
		}
	}
	return ""
}

func labeledBullet(label, value string) string {
	value = cleanUserVisibleText(value)
	if value == "" {
		return ""
	}
	return "**" + strings.TrimSpace(label) + "**：" + value
}

func conclusionOrDefault(fallback string, values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func writeHeading(b *strings.Builder, heading string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("## ")
	b.WriteString(heading)
	b.WriteString("\n")
}

// writeSubheading creates a stable second-level section without changing the page-level outline.

// writeSubheading creates a stable second-level section without changing the page-level outline.
func writeSubheading(b *strings.Builder, heading string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("### ")
	b.WriteString(heading)
	b.WriteString("\n")
}

// writeConclusion writes a sanitized conclusion, which is the final text sink
// for model-owned slot values.

// writeConclusion writes a sanitized conclusion, which is the final text sink
// for model-owned slot values.
func writeConclusion(b *strings.Builder, text string) {
	text = cleanUserVisibleText(text)
	if text == "" {
		text = "本轮未形成可展示结论。"
	}
	b.WriteString("**结论：")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("**\n")
}

// writeParagraphs writes only renderer-sanitized user-visible paragraphs.

// writeParagraphs writes only renderer-sanitized user-visible paragraphs.
func writeParagraphs(b *strings.Builder, paragraphs []string) {
	for _, paragraph := range filterNonEmpty(paragraphs) {
		paragraph = cleanUserVisibleText(paragraph)
		if paragraph == "" {
			continue
		}
		b.WriteString(paragraph)
		b.WriteString("\n")
	}
}

// writeBullets writes renderer-sanitized user-visible list items.

// writeBullets writes renderer-sanitized user-visible list items.
func writeBullets(b *strings.Builder, bullets []string) {
	for _, bullet := range filterNonEmpty(bullets) {
		bullet = cleanUserVisibleText(bullet)
		if bullet == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(bullet))
		b.WriteString("\n")
	}
}

// writeDayunAnalysis keeps each luck period as an independent Markdown block.
// The upstream synthesis owns the verdict and evidence; rendering must not
// flatten the full analysis into a single nested list item.

// writeDayunAnalysis keeps each luck period as an independent Markdown block.
// The upstream synthesis owns the verdict and evidence; rendering must not
// flatten the full analysis into a single nested list item.
func writeDayunAnalysis(b *strings.Builder, periods []string) {
	periods = filterNonEmpty(periods)
	if len(periods) == 0 {
		b.WriteString("当前大运尚无可解释的关系。\n")
		return
	}
	for _, period := range periods {
		b.WriteString(cleanUserVisibleText(strings.TrimSpace(period)))
		b.WriteString("\n\n")
	}
}

// writeSteps writes renderer-sanitized reasoning steps.

// writeSteps writes renderer-sanitized reasoning steps.
func writeSteps(b *strings.Builder, steps []string) {
	for i, step := range filterNonEmpty(steps) {
		step = cleanUserVisibleText(step)
		if step == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(step)))
	}
}

// writeHighlightBlock writes a sanitized short summary beneath the overview.

// writeHighlightBlock writes a sanitized short summary beneath the overview.
func writeHighlightBlock(b *strings.Builder, title, summary string, details ...string) {
	summary = cleanUserVisibleText(summary)
	if summary == "" {
		summary = "本轮未形成可展示结论。"
	}
	b.WriteString("> ")
	b.WriteString(strings.TrimSpace(title))
	b.WriteString("：")
	b.WriteString(strings.TrimSpace(summary))
	b.WriteString("\n")
	for _, detail := range filterNonEmpty(details) {
		detail = cleanUserVisibleText(detail)
		if detail == "" {
			continue
		}
		b.WriteString("> ")
		b.WriteString(strings.TrimSpace(detail))
		b.WriteString("\n")
	}
}

func sanitizeUnsupportedFlourish(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"贵人众多", "助力较多",
		"福泽深厚", "福分较厚",
		"可享清福", "后程较稳",
	)
	return replacer.Replace(strings.TrimSpace(text))
}

func cleanUserVisibleText(text string) string {
	text = sanitizeUnsupportedFlourish(text)
	text = sanitizeInternalBoundaryText(text)
	if text == "" || strings.Contains(text, "上游未提供") {
		return ""
	}
	return text
}

// sanitizeInternalBoundaryText removes engineering-oriented wording from
// user-visible renderer text without loosening any validation boundary.

// sanitizeInternalBoundaryText removes engineering-oriented wording from
// user-visible renderer text without loosening any validation boundary.
func sanitizeInternalBoundaryText(text string) string {
	text = baziInternalReferencePath.ReplaceAllString(text, "已计算的结构事实")
	replacer := strings.NewReplacer(
		"暂不定级", "暂缓定级",
		"调候规则未实现", "调候规则材料不足",
		"规则表未实现", "规则材料不足",
		"待规则表实现", "待规则材料补足",
		"待规则裁断", "待规则证据补足",
		"仅作结构观察", "只作结构说明",
		"证据不足", "证据还不够",
		"未启用运行时规则 profile", "未选择专门规则口径",
		"运行时规则 profile", "专门规则口径",
		"规则profile", "规则口径",
		"待profile裁断", "待规则证据补足",
		"动态综合未通过", "动态裁断受限",
		"模型动态综合不可用", "动态裁断受限",
		"runtime", "系统",
	)
	return replacer.Replace(strings.TrimSpace(text))
}
