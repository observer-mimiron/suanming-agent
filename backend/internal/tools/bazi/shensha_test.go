package bazi

import "testing"

func TestComputeShensha_TianYiUsesYearAndDayStem(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "戊", "乙", "庚"},
		[]string{"丑", "辰", "午", "子"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["年柱"], "天乙贵人", "年干") {
		t.Fatal("年柱 should contain 天乙贵人 with 年干 basis")
	}
	if !hasShensha(byPillar["时柱"], "天乙贵人", "日干") {
		t.Fatal("时柱 should contain 天乙贵人 with 日干 basis")
	}
}

func TestComputeShensha_MonthBranchTriggersCoreNobleStars(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"丁", "甲", "辛", "丙"},
		[]string{"子", "寅", "亥", "巳"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["时柱"], "月德贵人", "月支") {
		t.Fatal("时柱 should contain 月德贵人 triggered by 月支")
	}
	if !hasShensha(byPillar["年柱"], "天德贵人", "月支") {
		t.Fatal("年柱 should contain 天德贵人 triggered by 月支")
	}
	if !hasShensha(byPillar["日柱"], "月德合", "月支") {
		t.Fatal("日柱 should contain 月德合 triggered by 月支")
	}
}

func TestComputeShensha_CoreBranchStarsUseYearAndDayBranch(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "庚", "甲", "丙"},
		[]string{"寅", "巳", "午", "申"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["时柱"], "驿马", "年支") {
		t.Fatal("时柱 should contain 驿马 with 年支 basis")
	}
	if !hasShensha(byPillar["时柱"], "驿马", "日支") {
		t.Fatal("时柱 should contain 驿马 with 日支 basis")
	}
	if !hasShensha(byPillar["日柱"], "将星", "年支") {
		t.Fatal("日柱 should contain 将星 with 年支 basis")
	}
	if !hasShensha(byPillar["日柱"], "将星", "日支") {
		t.Fatal("日柱 should contain 将星 with 日支 basis")
	}
	if !hasShensha(byPillar["月柱"], "亡神", "年支") {
		t.Fatal("月柱 should contain 亡神 with 年支 basis")
	}
	if !hasShensha(byPillar["月柱"], "亡神", "日支") {
		t.Fatal("月柱 should contain 亡神 with 日支 basis")
	}
}

func TestComputeShensha_HuaGaiSkipsOwnYearAndDayPillars(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "乙", "丙", "丁"},
		[]string{"未", "未", "辰", "辰"},
		[]string{"", "", "", ""},
		pillars,
	)

	if hasShensha(byPillar["年柱"], "华盖", "年支") {
		t.Fatal("年柱 should not contain 华盖 with 年支 basis when only self pillar matches")
	}
	if !hasShensha(byPillar["月柱"], "华盖", "年支") {
		t.Fatal("月柱 should contain 华盖 with 年支 basis")
	}
	if hasShensha(byPillar["日柱"], "华盖", "日支") {
		t.Fatal("日柱 should not contain 华盖 with 日支 basis when only self pillar matches")
	}
	if !hasShensha(byPillar["时柱"], "华盖", "日支") {
		t.Fatal("时柱 should contain 华盖 with 日支 basis")
	}
}

func TestComputeShensha_JieShaUsesYearAndDayBranch(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "庚", "甲", "丙"},
		[]string{"寅", "辰", "午", "亥"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["时柱"], "劫煞", "年支") {
		t.Fatal("时柱 should contain 劫煞 with 年支 basis")
	}
	if !hasShensha(byPillar["时柱"], "劫煞", "日支") {
		t.Fatal("时柱 should contain 劫煞 with 日支 basis")
	}
}

func TestComputeShensha_HongLuanAndTianXiUseYearBranch(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "乙", "丙", "丁"},
		[]string{"子", "卯", "酉", "辰"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["月柱"], "红鸾", "年支") {
		t.Fatal("月柱 should contain 红鸾 with 年支 basis")
	}
	if !hasShensha(byPillar["日柱"], "天喜", "年支") {
		t.Fatal("日柱 should contain 天喜 with 年支 basis")
	}
}

func TestComputeShensha_JinYuAndHongYanUseReferenceMappings(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "乙", "丁", "丙"},
		[]string{"辰", "巳", "子", "未"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["年柱"], "金舆", "年干") {
		t.Fatal("年柱 should contain 金舆 with 年干 basis")
	}
	if !hasShensha(byPillar["时柱"], "红艳煞", "日干") {
		t.Fatal("时柱 should contain 红艳煞 with 日干 basis")
	}
}

func TestComputeShensha_XueTangCiGuanAndTianChuFollowReferencePatterns(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "己", "庚", "丙"},
		[]string{"子", "亥", "寅", "巳"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["月柱"], "学堂", "年干") {
		t.Fatal("月柱 should contain 学堂 with 年干 basis")
	}
	if !hasShensha(byPillar["日柱"], "词馆", "年干") {
		t.Fatal("日柱 should contain 词馆 with 年干 basis")
	}
	if !hasShensha(byPillar["时柱"], "天厨贵人", "年干") {
		t.Fatal("时柱 should contain 天厨贵人 with 年干 basis")
	}
}

func TestComputeShensha_TianDeHeUsesMonthBranch(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "乙", "丙", "壬"},
		[]string{"子", "寅", "辰", "午"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["时柱"], "天德合", "月支") {
		t.Fatal("时柱 should contain 天德合 with 月支 basis")
	}
}

func TestComputeShensha_SanQiUsesOrderedConsecutiveStems(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "戊", "庚", "丙"},
		[]string{"子", "丑", "寅", "卯"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["年柱"], "三奇贵人", "三干顺序") {
		t.Fatal("年柱 should contain 三奇贵人 with 三干顺序 basis")
	}
	if !hasShensha(byPillar["月柱"], "三奇贵人", "三干顺序") {
		t.Fatal("月柱 should contain 三奇贵人 with 三干顺序 basis")
	}
	if !hasShensha(byPillar["日柱"], "三奇贵人", "三干顺序") {
		t.Fatal("日柱 should contain 三奇贵人 with 三干顺序 basis")
	}
}

func TestComputeShensha_ExtendedBranchStarsFollowReferenceProject(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "丙", "甲", "辛"},
		[]string{"子", "寅", "戌", "酉"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["月柱"], "丧门", "年支") {
		t.Fatal("月柱 should contain 丧门 with 年支 basis")
	}
	if !hasShensha(byPillar["日柱"], "吊客", "年支") {
		t.Fatal("日柱 should contain 吊客 with 年支 basis")
	}
	if !hasShensha(byPillar["时柱"], "披麻", "年支") {
		t.Fatal("时柱 should contain 披麻 with 年支 basis")
	}
	if !hasShensha(byPillar["时柱"], "飞刃", "日干") {
		t.Fatal("时柱 should contain 飞刃 with 日干 basis")
	}
	if !hasShensha(byPillar["时柱"], "流霞", "日干") {
		t.Fatal("时柱 should contain 流霞 with 日干 basis")
	}
}

func TestComputeShensha_GouJiaoAndYuanChenUseReferenceMappings(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "辛", "甲", "丁"},
		[]string{"子", "亥", "子", "未"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["月柱"], "勾绞煞", "年支") {
		t.Fatal("月柱 should contain 勾绞煞 with 年支 basis")
	}
	if !hasShensha(byPillar["月柱"], "勾绞煞", "日支") {
		t.Fatal("月柱 should contain 勾绞煞 with 日支 basis")
	}
	if !hasShensha(byPillar["时柱"], "元辰", "年柱") {
		t.Fatal("时柱 should contain 元辰 with 年柱 basis")
	}
}

func TestComputeShensha_TianSheAndDaySpecialSets(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "丙", "戊", "庚"},
		[]string{"子", "寅", "寅", "申"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["日柱"], "天赦日", "月支") {
		t.Fatal("日柱 should contain 天赦日 with 月支 basis")
	}

	byPillar, _ = computeShensha(
		[]string{"甲", "庚", "甲", "丙"},
		[]string{"子", "申", "寅", "子"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["日柱"], "八专日", "日柱") {
		t.Fatal("日柱 should contain 八专日 with 日柱 basis")
	}
	if !hasShensha(byPillar["日柱"], "四废日", "月支") {
		t.Fatal("日柱 should contain 四废日 with 月支 basis")
	}
	if !hasShensha(byPillar["日柱"], "孤鸾", "日柱") {
		t.Fatal("日柱 should contain 孤鸾 with 日柱 basis")
	}
}

func TestComputeShensha_JinShenAndGongLuUseExactGanzhiPairs(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "丙", "乙", "己"},
		[]string{"子", "寅", "丑", "巳"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["日柱"], "金神", "日柱") {
		t.Fatal("日柱 should contain 金神 with 日柱 basis")
	}
	if !hasShensha(byPillar["时柱"], "金神", "时柱") {
		t.Fatal("时柱 should contain 金神 with 时柱 basis")
	}

	byPillar, _ = computeShensha(
		[]string{"甲", "乙", "癸", "癸"},
		[]string{"子", "卯", "亥", "丑"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["日柱"], "拱子禄", "日时") {
		t.Fatal("日柱 should contain 拱子禄 with 日时 basis")
	}
	if !hasShensha(byPillar["时柱"], "拱子禄", "日时") {
		t.Fatal("时柱 should contain 拱子禄 with 日时 basis")
	}
}

func TestComputeShensha_TongZiShaUsesMonthAndNaYinTargets(t *testing.T) {
	pillars := testPillarsWithNaYin("海中金")

	byPillar, _ := computeShensha(
		[]string{"甲", "乙", "丙", "庚"},
		[]string{"子", "寅", "寅", "午"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["日柱"], "童子煞", "月支/年纳音") {
		t.Fatal("日柱 should contain 童子煞 with 月支/年纳音 basis")
	}
	if !hasShensha(byPillar["时柱"], "童子煞", "月支/年纳音") {
		t.Fatal("时柱 should contain 童子煞 with 月支/年纳音 basis")
	}
}

func TestComputeShensha_XueRenAndOtherDaySetsCanCoexist(t *testing.T) {
	pillars := testPillars()

	byPillar, _ := computeShensha(
		[]string{"甲", "辛", "戊", "丁"},
		[]string{"子", "亥", "午", "亥"},
		[]string{"", "", "", ""},
		pillars,
	)

	if !hasShensha(byPillar["时柱"], "血刃", "月支") {
		t.Fatal("时柱 should contain 血刃 with 月支 basis")
	}
	if !hasShensha(byPillar["日柱"], "十灵日", "日柱") {
		t.Fatal("日柱 should contain 十灵日 with 日柱 basis")
	}
	if !hasShensha(byPillar["日柱"], "六秀日", "日柱") {
		t.Fatal("日柱 should contain 六秀日 with 日柱 basis")
	}
	if !hasShensha(byPillar["日柱"], "九丑日", "日柱") {
		t.Fatal("日柱 should contain 九丑日 with 日柱 basis")
	}
}

func testPillars() []map[string]any {
	return []map[string]any{
		{"name": "年柱"},
		{"name": "月柱"},
		{"name": "日柱"},
		{"name": "时柱"},
	}
}

func testPillarsWithNaYin(naYin string) []map[string]any {
	pillars := testPillars()
	pillars[0]["naYin"] = naYin
	return pillars
}

func hasShensha(items []ShenshaItem, name, basis string) bool {
	for _, item := range items {
		if item.Name == name && item.Basis == basis {
			return true
		}
	}
	return false
}
