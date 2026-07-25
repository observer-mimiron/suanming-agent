package bazi

// ShenshaItem 神煞。表示命盘某一柱上的单个神煞，包含名称、吉凶属性（吉/凶/平）、推算依据和中文描述。
// 神煞是八字命理中基于干支组合的符号化判断体系，如天乙贵人、桃花、驿马等。
type ShenshaItem struct {
	Name        string `json:"name"`
	Tone        string `json:"tone"` // 吉 | 凶 | 平
	Basis       string `json:"basis"`
	Description string `json:"description"`
}

// computeShensha 计算命盘神煞。基于年干、日干、年支、日支和时支之间的关系推算各类神煞。
// gan/zhi 为[年柱, 月柱, 日柱, 时柱]的天干/地支数组；xunKong 为各柱的旬空信息。
// 返回值 byPillar 是按柱分组的神煞列表，updated pillars 是追加了神煞字段的更新后的四柱。
//
// 当前实现聚焦低争议主流神煞，优先覆盖主流排盘产品与常见古籍口径的交集。
// 规则按触发来源拆成三类：年/日干触发、年/日支触发、月支触发。
func computeShensha(gan, zhi, xunKong []string, pillars []map[string]any) (map[string][]ShenshaItem, []map[string]any) {
	yearGan := gan[0]
	dayGan := gan[2]
	monthZhi := zhi[1]
	dayZhi := zhi[2]
	yearZhi := zhi[0]

	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}
	byPillar := map[string][]ShenshaItem{
		"年柱": {}, "月柱": {}, "日柱": {}, "时柱": {},
	}

	appendUnique := func(pillar, name, tone, basis, desc string) {
		for _, item := range byPillar[pillar] {
			if item.Name == name && item.Basis == basis {
				return
			}
		}
		byPillar[pillar] = append(byPillar[pillar], ShenshaItem{
			Name: name, Tone: tone, Basis: basis, Description: desc,
		})
	}

	// addByBranch 将神煞添加到所有地支与目标匹配的柱上。
	addByBranch := func(name, tone, basis, desc string, targets []string) {
		for i, z := range zhi {
			for _, t := range targets {
				if z == t {
					appendUnique(pillarNames[i], name, tone, basis, desc)
				}
			}
		}
	}

	// addToDayPillar 将神煞仅添加到日柱上。
	addToDayPillar := func(name, tone, basis, desc string) {
		appendUnique("日柱", name, tone, basis, desc)
	}

	addByStem := func(name, tone, basis, desc, target string) {
		if target == "" {
			return
		}
		for i, g := range gan {
			if g == target {
				appendUnique(pillarNames[i], name, tone, basis, desc)
			}
		}
	}

	addByStemOrBranch := func(name, tone, basis, desc, target string) {
		if target == "" {
			return
		}
		for i := range gan {
			if gan[i] == target || zhi[i] == target {
				appendUnique(pillarNames[i], name, tone, basis, desc)
			}
		}
	}

	addByGanzhi := func(name, tone, basis, desc, targetGan, targetZhi string) {
		if targetGan == "" || targetZhi == "" {
			return
		}
		for i := range gan {
			if gan[i] == targetGan && zhi[i] == targetZhi {
				appendUnique(pillarNames[i], name, tone, basis, desc)
			}
		}
	}

	addByYearAndDayStemTargets := func(name, tone, desc string, mapping map[string][]string) {
		if targets, ok := mapping[yearGan]; ok {
			addByBranch(name, tone, "年干", desc, targets)
		}
		if targets, ok := mapping[dayGan]; ok {
			addByBranch(name, tone, "日干", desc, targets)
		}
	}

	addByYearAndDaySingleBranch := func(name, tone, desc string, mapping map[string]string) {
		if target, ok := mapping[yearGan]; ok {
			addByBranch(name, tone, "年干", desc, []string{target})
		}
		if target, ok := mapping[dayGan]; ok {
			addByBranch(name, tone, "日干", desc, []string{target})
		}
	}

	addByYearAndDayZhiTargets := func(name, tone, desc string, mapping map[string]string) {
		if target, ok := mapping[yearZhi]; ok {
			addByBranch(name, tone, "年支", desc, []string{target})
		}
		if target, ok := mapping[dayZhi]; ok {
			addByBranch(name, tone, "日支", desc, []string{target})
		}
	}

	addByMonthStem := func(name, tone, desc string, mapping map[string]string) {
		if target, ok := mapping[monthZhi]; ok {
			addByStem(name, tone, "月支", desc, target)
		}
	}

	addByMonthStemOrBranch := func(name, tone, desc string, mapping map[string]string) {
		if target, ok := mapping[monthZhi]; ok {
			addByStemOrBranch(name, tone, "月支", desc, target)
		}
	}

	addByYearAndDayGanzhiTargets := func(name, tone, desc string, mapping map[string][2]string) {
		if target, ok := mapping[yearGan]; ok {
			addByGanzhi(name, tone, "年干", desc, target[0], target[1])
		}
		if target, ok := mapping[dayGan]; ok {
			addByGanzhi(name, tone, "日干", desc, target[0], target[1])
		}
	}

	addByBranchAt := func(name, tone, basis, desc string, targets []string, indices ...int) {
		for _, idx := range indices {
			if idx < 0 || idx >= len(zhi) {
				continue
			}
			for _, target := range targets {
				if zhi[idx] == target {
					appendUnique(pillarNames[idx], name, tone, basis, desc)
				}
			}
		}
	}

	addByYearAndDayZhiTargetsExcludingSelf := func(name, tone, desc string, mapping map[string]string) {
		if target, ok := mapping[yearZhi]; ok {
			addByBranchAt(name, tone, "年支", desc, []string{target}, 1, 2, 3)
		}
		if target, ok := mapping[dayZhi]; ok {
			addByBranchAt(name, tone, "日支", desc, []string{target}, 0, 1, 3)
		}
	}

	zhiSeq := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	zhiOffset := func(base string, offset int) string {
		for i, item := range zhiSeq {
			if item == base {
				return zhiSeq[(i+offset+len(zhiSeq))%len(zhiSeq)]
			}
		}
		return ""
	}

	extractWuXing := func(value string) string {
		for _, r := range value {
			switch r {
			case '金', '木', '水', '火', '土':
				return string(r)
			}
		}
		return ""
	}

	// ---- 基于规则的神煞推算 ----

	// 1. 天乙贵人 — 年干 / 日干
	tianYi := map[string][]string{
		"甲": {"丑", "未"}, "戊": {"丑", "未"}, "庚": {"丑", "未"},
		"乙": {"子", "申"}, "己": {"子", "申"},
		"丙": {"亥", "酉"}, "丁": {"亥", "酉"},
		"壬": {"卯", "巳"}, "癸": {"卯", "巳"},
		"辛": {"午", "寅"},
	}
	addByYearAndDayStemTargets("天乙贵人", "good", "贵人相助，逢凶化吉", tianYi)

	// 2. 月德贵人 / 月德合 / 天德贵人 — 月支
	yueDe := map[string]string{
		"寅": "丙", "午": "丙", "戌": "丙",
		"申": "壬", "子": "壬", "辰": "壬",
		"亥": "甲", "卯": "甲", "未": "甲",
		"巳": "庚", "酉": "庚", "丑": "庚",
	}
	addByMonthStem("月德贵人", "good", "主逢凶化吉，得人扶持", yueDe)

	yueDeHe := map[string]string{
		"寅": "辛", "午": "辛", "戌": "辛",
		"申": "丁", "子": "丁", "辰": "丁",
		"巳": "乙", "酉": "乙", "丑": "乙",
		"亥": "己", "卯": "己", "未": "己",
	}
	addByMonthStem("月德合", "good", "主和合解厄，缓冲冲克", yueDeHe)

	tianDe := map[string]string{
		"寅": "丁", "卯": "申", "辰": "壬", "巳": "辛",
		"午": "亥", "未": "甲", "申": "癸", "酉": "寅",
		"戌": "丙", "亥": "乙", "子": "巳", "丑": "庚",
	}
	addByMonthStemOrBranch("天德贵人", "good", "主仁厚有助，遇事多解", tianDe)

	tianDeHe := map[string]string{
		"寅": "壬", "卯": "巳", "辰": "丁", "巳": "丙",
		"午": "寅", "未": "己", "申": "戊", "酉": "亥",
		"戌": "辛", "亥": "庚", "子": "申", "丑": "乙",
	}
	addByMonthStemOrBranch("天德合", "good", "主和合缓冲，助解冲克", tianDeHe)

	// 3. 太极贵人 — 年干 / 日干
	taiji := map[string][]string{
		"甲": {"子", "午"}, "乙": {"子", "午"},
		"丙": {"卯", "酉"}, "丁": {"卯", "酉"},
		"戊": {"辰", "戌", "丑", "未"}, "己": {"辰", "戌", "丑", "未"},
		"庚": {"寅", "亥"}, "辛": {"寅", "亥"},
		"壬": {"申", "巳"}, "癸": {"申", "巳"},
	}
	addByYearAndDayStemTargets("太极贵人", "good", "主聪慧悟性强，近贵近道", taiji)

	// 4. 文昌贵人 — 年干 / 日干
	wenchang := map[string]string{"甲": "巳", "乙": "午", "丙": "申", "丁": "酉", "戊": "申", "己": "酉", "庚": "亥", "辛": "子", "壬": "寅", "癸": "卯"}
	addByYearAndDaySingleBranch("文昌贵人", "good", "主学业、文才、考试运", wenchang)

	// 5. 国印贵人 — 年干 / 日干
	guoyin := map[string]string{"甲": "戌", "乙": "亥", "丙": "丑", "丁": "寅", "戊": "丑", "己": "寅", "庚": "辰", "辛": "巳", "壬": "未", "癸": "申"}
	addByYearAndDaySingleBranch("国印贵人", "good", "主守成持重，得权得信", guoyin)

	// 6. 福星贵人 — 年干 / 日干
	fuxing := map[string][]string{
		"甲": {"寅", "子"}, "丙": {"寅", "子"},
		"乙": {"卯", "丑"}, "癸": {"卯", "丑"},
		"戊": {"申"}, "己": {"未"}, "丁": {"亥"},
		"庚": {"午"}, "辛": {"巳"}, "壬": {"辰"},
	}
	addByYearAndDayStemTargets("福星贵人", "good", "主福气增益，化险添吉", fuxing)

	tianchu := map[string]string{
		"甲": "巳", "乙": "午", "丙": "巳", "丁": "午",
		"戊": "申", "庚": "亥", "辛": "子", "壬": "寅", "癸": "卯",
	}
	addByYearAndDaySingleBranch("天厨贵人", "good", "主口福享受，衣食有余", tianchu)

	// 7. 德秀贵人 / 天医 — 月支
	dexiu := map[string]struct {
		de  []string
		xiu []string
	}{
		"寅": {de: []string{"丙", "丁"}, xiu: []string{"戊", "癸"}},
		"午": {de: []string{"丙", "丁"}, xiu: []string{"戊", "癸"}},
		"戌": {de: []string{"丙", "丁"}, xiu: []string{"戊", "癸"}},
		"申": {de: []string{"壬", "癸", "戊", "己"}, xiu: []string{"丙", "辛", "甲", "己"}},
		"子": {de: []string{"壬", "癸", "戊", "己"}, xiu: []string{"丙", "辛", "甲", "己"}},
		"辰": {de: []string{"壬", "癸", "戊", "己"}, xiu: []string{"丙", "辛", "甲", "己"}},
		"巳": {de: []string{"庚", "辛"}, xiu: []string{"乙", "庚"}},
		"酉": {de: []string{"庚", "辛"}, xiu: []string{"乙", "庚"}},
		"丑": {de: []string{"庚", "辛"}, xiu: []string{"乙", "庚"}},
		"亥": {de: []string{"甲", "乙"}, xiu: []string{"丁", "壬"}},
		"卯": {de: []string{"甲", "乙"}, xiu: []string{"丁", "壬"}},
		"未": {de: []string{"甲", "乙"}, xiu: []string{"丁", "壬"}},
	}
	if cfg, ok := dexiu[monthZhi]; ok {
		for _, g := range cfg.de {
			addByStem("德秀贵人", "good", "月支", "主气质文雅，才情与声名", g)
		}
		for _, g := range cfg.xiu {
			addByStem("德秀贵人", "good", "月支", "主气质文雅，才情与声名", g)
		}
	}

	tianyi := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	for i, mz := range tianyi {
		if monthZhi == mz {
			target := tianyi[(i-1+len(tianyi))%len(tianyi)]
			addByBranch("天医", "good", "月支", "主医药缘与康复力", []string{target})
			break
		}
	}

	// 8. 桃花（咸池）— 年支 / 日支
	taohua := map[string]string{"申": "酉", "子": "酉", "辰": "酉", "寅": "卯", "午": "卯", "戌": "卯", "亥": "子", "卯": "子", "未": "子", "巳": "午", "酉": "午", "丑": "午"}
	addByYearAndDayZhiTargets("桃花", "neutral", "主人缘、魅力、感情机会", taohua)

	hongluan := map[string]string{
		"子": "卯", "丑": "寅", "寅": "丑", "卯": "子",
		"辰": "亥", "巳": "戌", "午": "酉", "未": "申",
		"申": "未", "酉": "午", "戌": "巳", "亥": "辰",
	}
	if target, ok := hongluan[yearZhi]; ok {
		addByBranch("红鸾", "good", "年支", "主喜庆婚缘，人际和合", []string{target})
	}

	tianxi := map[string]string{
		"子": "酉", "丑": "申", "寅": "未", "卯": "午",
		"辰": "巳", "巳": "辰", "午": "卯", "未": "寅",
		"申": "丑", "酉": "子", "戌": "亥", "亥": "戌",
	}
	if target, ok := tianxi[yearZhi]; ok {
		addByBranch("天喜", "good", "年支", "主喜事临门，感情助力", []string{target})
	}

	hongyan := map[string]string{
		"甲": "午", "乙": "午", "丙": "寅", "丁": "未",
		"戊": "辰", "己": "辰", "庚": "戌", "辛": "酉",
		"壬": "子", "癸": "申",
	}
	if target, ok := hongyan[dayGan]; ok {
		addByBranch("红艳煞", "neutral", "日干", "主异性缘与魅力外显，感情波动也增", []string{target})
	}

	jinyu := map[string][]string{
		"甲": {"辰"}, "乙": {"巳"}, "丙": {"未"}, "戊": {"未"},
		"丁": {"申"}, "己": {"申"}, "庚": {"戌"}, "辛": {"亥"},
		"壬": {"丑"}, "癸": {"寅"},
	}
	addByYearAndDayStemTargets("金舆", "good", "主福泽体面，配偶与衣禄助力", jinyu)

	// 9. 驿马 — 年支 / 日支
	yima := map[string]string{"申": "寅", "子": "寅", "辰": "寅", "寅": "申", "午": "申", "戌": "申", "亥": "巳", "卯": "巳", "未": "巳", "巳": "亥", "酉": "亥", "丑": "亥"}
	addByYearAndDayZhiTargets("驿马", "neutral", "主奔波、旅行、变动", yima)

	// 10. 华盖 — 年支 / 日支
	huagai := map[string]string{"申": "辰", "子": "辰", "辰": "辰", "寅": "戌", "午": "戌", "戌": "戌", "亥": "未", "卯": "未", "未": "未", "巳": "丑", "酉": "丑", "丑": "丑"}
	// 华盖按参考实现不回挂源柱自身，只看其余三柱是否落在对应墓库位。
	addByYearAndDayZhiTargetsExcludingSelf("华盖", "neutral", "主才艺、孤傲、宗教缘", huagai)

	// 11/12. 孤辰寡宿 — 年支
	// 亥子丑→孤辰寅寡宿戌, 寅卯辰→孤辰巳寡宿丑, 巳午未→孤辰申寡宿辰, 申酉戌→孤辰亥寡宿未
	guchen := map[string]string{"亥": "寅", "子": "寅", "丑": "寅", "寅": "巳", "卯": "巳", "辰": "巳", "巳": "申", "午": "申", "未": "申", "申": "亥", "酉": "亥", "戌": "亥"}
	guasu := map[string]string{"亥": "戌", "子": "戌", "丑": "戌", "寅": "丑", "卯": "丑", "辰": "丑", "巳": "辰", "午": "辰", "未": "辰", "申": "未", "酉": "未", "戌": "未"}
	if t, ok := guchen[yearZhi]; ok {
		addByBranch("孤辰", "bad", "年支", "主孤独、独立", []string{t})
	}
	if t, ok := guasu[yearZhi]; ok {
		addByBranch("寡宿", "bad", "年支", "主孤寡、独居", []string{t})
	}

	// 13. 羊刃 — 日干
	yangren := map[string]string{"甲": "卯", "乙": "寅", "丙": "午", "丁": "巳", "戊": "午", "己": "巳", "庚": "酉", "辛": "申", "壬": "子", "癸": "亥"}
	if t, ok := yangren[dayGan]; ok {
		addByBranch("羊刃", "bad", "日干", "主刚强、争执、血光", []string{t})
	}

	// 14. 禄神 — 日干
	lu := map[string]string{"甲": "寅", "乙": "卯", "丙": "巳", "丁": "午", "戊": "巳", "己": "午", "庚": "申", "辛": "酉", "壬": "亥", "癸": "子"}
	if t, ok := lu[dayGan]; ok {
		addByBranch("禄神", "good", "日干", "主食禄、福气、财富", []string{t})
	}

	// 15. 劫煞 — 年支 / 日支
	jiesha := map[string]string{"申": "巳", "子": "巳", "辰": "巳", "寅": "亥", "午": "亥", "戌": "亥", "亥": "申", "卯": "申", "未": "申", "巳": "寅", "酉": "寅", "丑": "寅"}
	addByYearAndDayZhiTargets("劫煞", "bad", "主是非、破财、意外", jiesha)

	// 16. 亡神 — 年支 / 日支
	wangshen := map[string]string{"寅": "巳", "午": "巳", "戌": "巳", "亥": "寅", "卯": "寅", "未": "寅", "巳": "申", "酉": "申", "丑": "申", "申": "亥", "子": "亥", "辰": "亥"}
	addByYearAndDayZhiTargets("亡神", "bad", "主耗散烦忧，易起暗损", wangshen)

	// 17. 灾煞 — 年支
	zaisha := map[string]string{"申": "午", "子": "午", "辰": "午", "寅": "子", "午": "子", "戌": "子", "亥": "酉", "卯": "酉", "未": "酉", "巳": "卯", "酉": "卯", "丑": "卯"}
	if target, ok := zaisha[yearZhi]; ok {
		addByBranch("灾煞", "bad", "年支", "主灾祸、疾病、横祸", []string{target})
	}

	// 18. 将星 — 年支 / 日支
	jiangxing := map[string]string{"申": "子", "子": "子", "辰": "子", "寅": "午", "午": "午", "戌": "午", "亥": "卯", "卯": "卯", "未": "卯", "巳": "酉", "酉": "酉", "丑": "酉"}
	addByYearAndDayZhiTargets("将星", "neutral", "主领导力、掌权", jiangxing)

	// 19. 天罗地网 — 年支 / 日支
	// 戌亥为天罗，辰巳为地网
	tianluodiwang := func(base, basis string) {
		switch base {
		case "戌", "亥":
			addByBranch("天罗", "bad", basis, "主困顿、官非", []string{base})
		case "辰", "巳":
			addByBranch("地网", "bad", basis, "主困顿、牢狱", []string{base})
		}
	}
	tianluodiwang(yearZhi, "年支")
	tianluodiwang(dayZhi, "日支")

	// 20. 魁罡 — 日柱（庚辰、庚戌、壬辰、戊戌）
	dayCombo := dayGan + dayZhi
	if dayCombo == "庚辰" || dayCombo == "庚戌" || dayCombo == "壬辰" || dayCombo == "戊戌" {
		addToDayPillar("魁罡", "neutral", "日柱", "主刚毅果断，行事凌厉")
	}

	xuetang := map[string][2]string{
		"甲": {"己", "亥"}, "乙": {"壬", "午"}, "丙": {"丙", "寅"}, "丁": {"丁", "酉"},
		"戊": {"戊", "寅"}, "己": {"己", "酉"}, "庚": {"辛", "巳"}, "辛": {"甲", "子"},
		"壬": {"甲", "申"}, "癸": {"乙", "卯"},
	}
	addByYearAndDayGanzhiTargets("学堂", "good", "主学习悟性、名望与文章之气", xuetang)

	ciguan := map[string][2]string{
		"甲": {"庚", "寅"}, "乙": {"辛", "卯"}, "丙": {"乙", "巳"}, "丁": {"戊", "午"},
		"戊": {"丁", "巳"}, "己": {"庚", "午"}, "庚": {"壬", "申"}, "辛": {"癸", "酉"},
		"壬": {"癸", "亥"}, "癸": {"壬", "戌"},
	}
	addByYearAndDayGanzhiTargets("词馆", "good", "主文采表达、词章与科名", ciguan)

	triads := [][]string{
		{"甲", "戊", "庚"},
		{"乙", "丙", "丁"},
		{"壬", "癸", "辛"},
	}
	hasOrderedTriad := func(stems []string) bool {
		if len(stems) != 3 {
			return false
		}
		for _, triad := range triads {
			if stems[0] == triad[0] && stems[1] == triad[1] && stems[2] == triad[2] {
				return true
			}
		}
		return false
	}
	if hasOrderedTriad([]string{gan[0], gan[1], gan[2]}) {
		appendUnique("年柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
		appendUnique("月柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
		appendUnique("日柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
	}
	if hasOrderedTriad([]string{gan[1], gan[2], gan[3]}) {
		appendUnique("月柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
		appendUnique("日柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
		appendUnique("时柱", "三奇贵人", "good", "三干顺序", "主机敏逢助，遇难有解")
	}

	feirenYangren := map[string]string{
		"甲": "卯", "乙": "寅", "丙": "午", "丁": "未", "戊": "午",
		"己": "未", "庚": "酉", "辛": "申", "壬": "子", "癸": "亥",
	}
	chong := map[string]string{
		"子": "午", "午": "子", "丑": "未", "未": "丑", "寅": "申", "申": "寅",
		"卯": "酉", "酉": "卯", "辰": "戌", "戌": "辰", "巳": "亥", "亥": "巳",
	}
	if target := chong[feirenYangren[dayGan]]; target != "" {
		addByBranch("飞刃", "bad", "日干", "主刚烈冲动，易生争执血伤", []string{target})
	}

	xueren := map[string]string{
		"寅": "丑", "卯": "未", "辰": "寅", "巳": "申", "午": "卯", "未": "酉",
		"申": "辰", "酉": "戌", "戌": "巳", "亥": "亥", "子": "午", "丑": "子",
	}
	if target, ok := xueren[monthZhi]; ok {
		addByBranchAt("血刃", "bad", "月支", "主血光损伤，需防手术磕碰", []string{target}, 0, 2, 3)
	}

	goujiaosha := map[string]string{
		"寅": "巳", "午": "巳", "戌": "巳",
		"亥": "寅", "卯": "寅", "未": "寅",
		"巳": "申", "酉": "申", "丑": "申",
		"申": "亥", "子": "亥", "辰": "亥",
	}
	addByYearAndDayZhiTargets("勾绞煞", "bad", "主牵连羁绊，是非纠葛较多", goujiaosha)

	yangGans := map[string]bool{"甲": true, "丙": true, "戊": true, "庚": true, "壬": true}
	if chongZhi := chong[yearZhi]; chongZhi != "" {
		offset := -1
		if yangGans[yearGan] {
			offset = 1
		}
		if target := zhiOffset(chongZhi, offset); target != "" {
			addByBranchAt("元辰", "bad", "年柱", "主情绪反复，人际阻滞", []string{target}, 1, 2, 3)
		}
	}

	if target := zhiOffset(yearZhi, 2); target != "" {
		addByBranchAt("丧门", "bad", "年支", "主丧忧孝服，情绪压抑", []string{target}, 1, 2, 3)
	}
	if target := zhiOffset(yearZhi, 10); target != "" {
		addByBranchAt("吊客", "bad", "年支", "主吊唁奔波，易见忧厄", []string{target}, 1, 2, 3)
	}
	if target := zhiOffset(yearZhi, 9); target != "" {
		addByBranchAt("披麻", "bad", "年支", "主服制孝耗，家宅烦忧", []string{target}, 1, 2, 3)
	}

	yearNaYin, _ := pillars[0]["naYin"].(string)
	tongziTargets := map[string]bool{}
	switch monthZhi {
	case "寅", "卯", "辰", "申", "酉", "戌":
		tongziTargets["寅"] = true
		tongziTargets["子"] = true
	case "巳", "午", "未", "亥", "子", "丑":
		tongziTargets["卯"] = true
		tongziTargets["未"] = true
		tongziTargets["辰"] = true
	}
	switch extractWuXing(yearNaYin) {
	case "金", "木":
		tongziTargets["午"] = true
		tongziTargets["卯"] = true
	case "水", "火":
		tongziTargets["酉"] = true
		tongziTargets["戌"] = true
	case "土":
		tongziTargets["辰"] = true
		tongziTargets["巳"] = true
	}
	if len(tongziTargets) > 0 {
		if tongziTargets[zhi[2]] {
			appendUnique("日柱", "童子煞", "bad", "月支/年纳音", "主性情孤洁，感情婚缘多波折")
		}
		if tongziTargets[zhi[3]] {
			appendUnique("时柱", "童子煞", "bad", "月支/年纳音", "主性情孤洁，感情婚缘多波折")
		}
	}

	jinshenTargets := map[string]bool{"乙丑": true, "己巳": true, "癸酉": true}
	timeCombo := gan[3] + zhi[3]
	if jinshenTargets[dayCombo] {
		appendUnique("日柱", "金神", "neutral", "日柱", "主刚锐果决，喜忌需配格局详断")
	}
	if jinshenTargets[timeCombo] {
		appendUnique("时柱", "金神", "neutral", "时柱", "主刚锐果决，喜忌需配格局详断")
	}

	liuxia := map[string]string{
		"甲": "酉", "乙": "戌", "丙": "未", "丁": "申", "戊": "巳",
		"己": "午", "庚": "辰", "辛": "卯", "壬": "亥", "癸": "寅",
	}
	if target, ok := liuxia[dayGan]; ok {
		addByBranch("流霞", "bad", "日干", "主情感与血光波动，需防酒色伤灾", []string{target})
	}

	springMonths := map[string]bool{"寅": true, "卯": true, "辰": true}
	summerMonths := map[string]bool{"巳": true, "午": true, "未": true}
	autumnMonths := map[string]bool{"申": true, "酉": true, "戌": true}
	winterMonths := map[string]bool{"亥": true, "子": true, "丑": true}
	switch {
	case springMonths[monthZhi] && dayCombo == "戊寅":
		addToDayPillar("天赦日", "good", "月支", "主宽赦解厄，化凶为吉")
	case summerMonths[monthZhi] && dayCombo == "甲午":
		addToDayPillar("天赦日", "good", "月支", "主宽赦解厄，化凶为吉")
	case autumnMonths[monthZhi] && dayCombo == "戊申":
		addToDayPillar("天赦日", "good", "月支", "主宽赦解厄，化凶为吉")
	case winterMonths[monthZhi] && dayCombo == "甲子":
		addToDayPillar("天赦日", "good", "月支", "主宽赦解厄，化凶为吉")
	}

	daySets := []struct {
		name   string
		tone   string
		desc   string
		target map[string]bool
	}{
		{
			name: "十灵日", tone: "good", desc: "主聪敏通灵，悟性较高",
			target: map[string]bool{"甲辰": true, "乙亥": true, "丙辰": true, "丁酉": true, "戊午": true, "庚戌": true, "庚寅": true, "辛亥": true, "壬寅": true, "癸未": true},
		},
		{
			name: "八专日", tone: "neutral", desc: "主情志专一偏执，感情欲念较强",
			target: map[string]bool{"甲寅": true, "乙卯": true, "丁未": true, "戊戌": true, "己未": true, "庚申": true, "辛酉": true, "癸丑": true},
		},
		{
			name: "六秀日", tone: "good", desc: "主秀气聪慧，才艺表达较佳",
			target: map[string]bool{"丙午": true, "丁未": true, "戊子": true, "戊午": true, "己丑": true, "己未": true},
		},
		{
			name: "九丑日", tone: "bad", desc: "主情欲是非，感情名誉易受扰",
			target: map[string]bool{"丁酉": true, "戊子": true, "戊午": true, "己卯": true, "己酉": true, "辛卯": true, "辛酉": true, "壬子": true, "壬午": true},
		},
		{
			name: "十恶大败", tone: "bad", desc: "主财禄易败，处事反复",
			target: map[string]bool{"甲辰": true, "乙巳": true, "丙申": true, "丁亥": true, "戊戌": true, "己丑": true, "庚辰": true, "辛巳": true, "壬申": true, "癸亥": true},
		},
		{
			name: "阴差阳错", tone: "bad", desc: "主婚恋沟通错位，易生误会",
			target: map[string]bool{"丙子": true, "丙午": true, "丁丑": true, "丁未": true, "戊寅": true, "戊申": true, "辛卯": true, "辛酉": true, "壬辰": true, "壬戌": true, "癸巳": true, "癸亥": true},
		},
		{
			name: "孤鸾", tone: "bad", desc: "主婚缘迟滞，感情多孤清",
			target: map[string]bool{"甲寅": true, "乙巳": true, "丙午": true, "丁巳": true, "戊午": true, "戊申": true, "辛亥": true, "壬子": true},
		},
	}
	for _, item := range daySets {
		if item.target[dayCombo] {
			addToDayPillar(item.name, item.tone, "日柱", item.desc)
		}
	}

	switch {
	case springMonths[monthZhi] && (dayCombo == "庚申" || dayCombo == "辛酉"):
		addToDayPillar("四废日", "bad", "月支", "主时气衰败，做事多阻")
	case summerMonths[monthZhi] && (dayCombo == "壬子" || dayCombo == "癸亥"):
		addToDayPillar("四废日", "bad", "月支", "主时气衰败，做事多阻")
	case autumnMonths[monthZhi] && (dayCombo == "甲寅" || dayCombo == "乙卯"):
		addToDayPillar("四废日", "bad", "月支", "主时气衰败，做事多阻")
	case winterMonths[monthZhi] && (dayCombo == "丙午" || dayCombo == "丁巳"):
		addToDayPillar("四废日", "bad", "月支", "主时气衰败，做事多阻")
	}

	gongLuPairs := []struct {
		day  string
		time string
		name string
	}{
		{day: "癸亥", time: "癸丑", name: "拱子禄"},
		{day: "癸丑", time: "癸亥", name: "拱子禄"},
		{day: "丁巳", time: "丁未", name: "拱午禄"},
		{day: "己未", time: "己巳", name: "拱午禄"},
		{day: "戊辰", time: "戊午", name: "拱巳禄"},
	}
	for _, pair := range gongLuPairs {
		if dayCombo == pair.day && timeCombo == pair.time {
			appendUnique("日柱", pair.name, "good", "日时", "主日时夹拱禄位，衣禄助力较强")
			appendUnique("时柱", pair.name, "good", "日时", "主日时夹拱禄位，衣禄助力较强")
			break
		}
	}

	// 参考项目里天转日/地转日当前使用了同一组判定表，这里先不盲从接入，
	// 避免把疑似源项目口径问题固化到主线实现中。

	// 复制四柱并附加神煞信息
	updated := make([]map[string]any, len(pillars))
	for i, p := range pillars {
		cp := make(map[string]any)
		for k, v := range p {
			cp[k] = v
		}
		name, _ := p["name"].(string)
		if name == "" {
			name = pillarNames[i]
		}
		cp["shensha"] = byPillar[name]
		updated[i] = cp
	}

	return byPillar, updated
}

func mergeShenshaForDisplay(byPillar map[string][]ShenshaItem, pillars []map[string]any) (map[string][]ShenshaItem, []map[string]any) {
	mergedByPillar := make(map[string][]ShenshaItem, len(byPillar))
	for pillar, items := range byPillar {
		mergedByPillar[pillar] = mergeShenshaItems(items)
	}

	updated := make([]map[string]any, len(pillars))
	for i, p := range pillars {
		cp := make(map[string]any, len(p)+1)
		for k, v := range p {
			cp[k] = v
		}
		if name, _ := p["name"].(string); name != "" {
			cp["shensha"] = mergedByPillar[name]
		}
		updated[i] = cp
	}

	return mergedByPillar, updated
}

func mergeShenshaItems(items []ShenshaItem) []ShenshaItem {
	if len(items) == 0 {
		return []ShenshaItem{}
	}

	order := make([]string, 0, len(items))
	indexByName := make(map[string]int, len(items))
	merged := make([]ShenshaItem, 0, len(items))

	for _, item := range items {
		if idx, ok := indexByName[item.Name]; ok {
			merged[idx].Basis = mergeShenshaBasis(merged[idx].Basis, item.Basis)
			if merged[idx].Description == "" {
				merged[idx].Description = item.Description
			}
			continue
		}

		indexByName[item.Name] = len(order)
		order = append(order, item.Name)
		merged = append(merged, item)
	}

	return merged
}

func mergeShenshaBasis(existing, next string) string {
	if existing == "" {
		return next
	}
	if next == "" || existing == next {
		return existing
	}

	seen := map[string]bool{}
	parts := make([]string, 0, 4)
	appendParts := func(value string) {
		start := 0
		for i := 0; i <= len(value); i++ {
			if i != len(value) && value[i] != '/' {
				continue
			}
			part := value[start:i]
			start = i + 1
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			parts = append(parts, part)
		}
	}

	appendParts(existing)
	appendParts(next)

	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}
