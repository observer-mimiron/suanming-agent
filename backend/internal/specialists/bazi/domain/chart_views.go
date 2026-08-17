// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责把排盘、用神、大运和流年事实裁剪为模型输入视图；
// 不负责调用模型、执行 Graph 或发送用户事件。
package domain

import "strings"

// ChartViewInput 是模型输入视图所需的最小事实载荷。
type ChartViewInput struct {
	BaziResult          map[string]any
	Yongshen            map[string]any
	Dayun               map[string]any
	Liunian             map[string]any
	SelectedRuleProfile any
}

// BuildCoreChartView 只暴露模型需要的核心命盘和已声明规则事实。
func BuildCoreChartView(input ChartViewInput) map[string]any {
	view := map[string]any{"selected_rule_profile": input.SelectedRuleProfile}
	if len(input.BaziResult) > 0 {
		if pillars, ok := input.BaziResult["pillars"]; ok && pillars != nil {
			view["pillars"] = pillars
			if monthPillar := ExtractMonthPillar(pillars); len(monthPillar) > 0 {
				view["month_pillar"] = monthPillar
			}
		}
		if dayGan, ok := input.BaziResult["dayGan"].(string); ok && dayGan != "" {
			view["day_master"] = dayGan
		}
		if dayWx, ok := input.BaziResult["dayGanWuxing"]; ok && dayWx != nil {
			view["day_master_wuxing"] = dayWx
		}
		if wx, ok := input.BaziResult["wuxing"]; ok && wx != nil {
			view["wuxing"] = wx
		}
	}
	for _, key := range coreYongshenKeys {
		value, ok := input.Yongshen[key]
		if !ok || value == nil || key == "tiao_hou" && IsTiaohouImplementationPlaceholder(value) {
			continue
		}
		view[key] = value
	}
	return view
}

// BuildDynamicFactsView 投影确定性工具已经计算的大运与流年事实。
func BuildDynamicFactsView(input ChartViewInput) map[string]any {
	view := map[string]any{}
	if dayun := BuildDayunFactsView(input.Dayun); len(dayun) > 0 {
		view["dayun"] = dayun
	}
	if liunian := BuildLiunianFactsView(input.Liunian); len(liunian) > 0 {
		view["liunian"] = liunian
	}
	return view
}

// BuildDayunFactsView 保留动态模型需要的大运目录和当前运绑定字段。
func BuildDayunFactsView(dayun map[string]any) map[string]any {
	return selectedMapFields(dayun, []string{"dayun_analyzed", "current_dayun", "periods"})
}

// BuildLiunianFactsView 保留动态模型需要的流年字段和当前大运引用。
func BuildLiunianFactsView(liunian map[string]any) map[string]any {
	return selectedMapFields(liunian, []string{
		"liunian_year", "liunian_ganzhi", "liunian_stem", "liunian_branch",
		"liunian_shi_shen", "current_dayun", "liunian_chonghe",
	})
}

// TiaohouCoverage 报告证据覆盖状态，不泄露工程实现状态。
func TiaohouCoverage(covered, missing []string) string {
	if containsString(covered, "tiaohou") {
		return "authority_evidence_covered"
	}
	if containsString(missing, "tiaohou") {
		return "missing_authority_evidence"
	}
	return "not_required"
}

// IsTiaohouImplementationPlaceholder 从模型输入中移除过期的工程状态占位文本。
func IsTiaohouImplementationPlaceholder(value any) bool {
	text := strings.TrimSpace(stringValue(value))
	return strings.Contains(text, "qiongtong_tiaohou_v1") || strings.Contains(text, "规则表实现")
}

// ExtractMonthPillar 从支持的解码形态中读取第二柱月柱。
func ExtractMonthPillar(raw any) map[string]any {
	pillars := anyMapSlice(raw)
	if len(pillars) < 2 {
		return nil
	}
	return CopyAnyMap(pillars[1])
}

// CopyAnyMap 复制一层 map，避免视图修改源事实容器。
func CopyAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// selectedMapFields 只保留非 nil 字段，并维持旧的 nil/空值合同。
func selectedMapFields(src map[string]any, keys []string) map[string]any {
	if len(src) == 0 {
		return nil
	}
	view := make(map[string]any)
	for _, key := range keys {
		if value, ok := src[key]; ok && value != nil {
			view[key] = value
		}
	}
	if len(view) == 0 {
		return nil
	}
	return view
}

var coreYongshenKeys = []string{
	"day_master", "day_master_wuxing", "strength", "strength_method", "strength_evidence",
	"month_score", "root_count", "same_element", "generate_count", "total_support",
	"balance_status", "seasonal_tiaohou_hint", "official_visibility", "season", "tiao_hou",
	"tiaohou_fire",
	"balance_yong_shen", "tiaohou_yong_shen", "conditional_yong_shen", "yong_shen", "xi_shen",
	"ji_shen", "geju", "geju_candidate", "geju_status", "geju_detail", "geju_basis",
	"geju_qing_zhuo", "geju_qing_zhuo_reason", "geju_combination", "chonghe", "shi_shen_power",
}
