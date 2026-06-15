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
// 包含的15种神煞：天乙贵人、桃花、驿马、华盖、孤辰、寡宿、文昌贵人、羊刃、禄神、劫煞、灾煞、
// 将星、天罗地网、魁罡、空亡。每种均有特定的干支规则推算。
func computeShensha(gan, zhi, xunKong []string, pillars []map[string]any) (map[string][]ShenshaItem, []map[string]any) {
	dayGan := gan[2]
	dayZhi := zhi[2]
	yearZhi := zhi[0]

	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}
	byPillar := map[string][]ShenshaItem{
		"年柱": {}, "月柱": {}, "日柱": {}, "时柱": {},
	}

	// addByBranch 将神煞添加到所有地支与目标匹配的柱上。
	addByBranch := func(name, tone, basis, desc string, targets []string) {
		for i, z := range zhi {
			for _, t := range targets {
				if z == t {
					byPillar[pillarNames[i]] = append(byPillar[pillarNames[i]], ShenshaItem{
						Name: name, Tone: tone, Basis: basis, Description: desc,
					})
				}
			}
		}
	}

	// addToDayPillar 将神煞仅添加到日柱上。
	addToDayPillar := func(name, tone, basis, desc string) {
		byPillar["日柱"] = append(byPillar["日柱"], ShenshaItem{
			Name: name, Tone: tone, Basis: basis, Description: desc,
		})
	}

	// ---- 基于规则的神煞推算 ----

	// 1. 天乙贵人 — 日干
	tianYi := map[string][]string{
		"甲": {"丑", "未"}, "戊": {"丑", "未"}, "庚": {"丑", "未"},
		"乙": {"子", "申"}, "己": {"子", "申"},
		"丙": {"亥", "酉"}, "丁": {"亥", "酉"},
		"壬": {"卯", "巳"}, "癸": {"卯", "巳"},
		"辛": {"午", "寅"},
	}
	if t, ok := tianYi[dayGan]; ok {
		addByBranch("天乙贵人", "good", "日干", "贵人相助，逢凶化吉", t)
	}

	// 2. 桃花（咸池）— 日支
	taohua := map[string]string{"申":"酉","子":"酉","辰":"酉","寅":"卯","午":"卯","戌":"卯","亥":"子","卯":"子","未":"子","巳":"午","酉":"午","丑":"午"}
	if t, ok := taohua[dayZhi]; ok {
		addByBranch("桃花", "neutral", "日支", "主人缘、魅力、感情机会", []string{t})
	}

	// 3. 驿马 — 日支
	yima := map[string]string{"申":"寅","子":"寅","辰":"寅","寅":"申","午":"申","戌":"申","亥":"巳","卯":"巳","未":"巳","巳":"亥","酉":"亥","丑":"亥"}
	if t, ok := yima[dayZhi]; ok {
		addByBranch("驿马", "neutral", "日支", "主奔波、旅行、变动", []string{t})
	}

	// 4. 华盖 — 日支
	huagai := map[string]string{"申":"辰","子":"辰","辰":"辰","寅":"戌","午":"戌","戌":"戌","亥":"未","卯":"未","未":"未","巳":"丑","酉":"丑","丑":"丑"}
	if t, ok := huagai[dayZhi]; ok {
		addByBranch("华盖", "neutral", "日支", "主才艺、孤傲、宗教缘", []string{t})
	}

	// 5/6. 孤辰寡宿 — 年支 (三会局)
	// 亥子丑→孤辰寅寡宿戌, 寅卯辰→孤辰巳寡宿丑, 巳午未→孤辰申寡宿辰, 申酉戌→孤辰亥寡宿未
	guchen := map[string]string{"亥":"寅","子":"寅","丑":"寅","寅":"巳","卯":"巳","辰":"巳","巳":"申","午":"申","未":"申","申":"亥","酉":"亥","戌":"亥"}
	guasu := map[string]string{"亥":"戌","子":"戌","丑":"戌","寅":"丑","卯":"丑","辰":"丑","巳":"辰","午":"辰","未":"辰","申":"未","酉":"未","戌":"未"}
	if t, ok := guchen[yearZhi]; ok {
		addByBranch("孤辰", "bad", "年支", "主孤独、独立", []string{t})
	}
	if t, ok := guasu[yearZhi]; ok {
		addByBranch("寡宿", "bad", "年支", "主孤寡、独居", []string{t})
	}

	// 7. 文昌贵人 — 日干
	wenchang := map[string]string{"甲":"巳","乙":"午","丙":"申","丁":"酉","戊":"申","己":"酉","庚":"亥","辛":"子","壬":"寅","癸":"卯"}
	if t, ok := wenchang[dayGan]; ok {
		addByBranch("文昌贵人", "good", "日干", "主学业、文才、考试运", []string{t})
	}

	// 8. 羊刃 — 日干
	yangren := map[string]string{"甲":"卯","乙":"寅","丙":"午","丁":"巳","戊":"午","己":"巳","庚":"酉","辛":"申","壬":"子","癸":"亥"}
	if t, ok := yangren[dayGan]; ok {
		addByBranch("羊刃", "bad", "日干", "主刚强、争执、血光", []string{t})
	}

	// 9. 禄神 — 日干
	lu := map[string]string{"甲":"寅","乙":"卯","丙":"巳","丁":"午","戊":"巳","己":"午","庚":"申","辛":"酉","壬":"亥","癸":"子"}
	if t, ok := lu[dayGan]; ok {
		addByBranch("禄神", "good", "日干", "主食禄、福气、财富", []string{t})
	}

	// 10. 劫煞 — 日支
	jiesha := map[string]string{"申":"巳","子":"巳","辰":"巳","寅":"亥","午":"亥","戌":"亥","亥":"申","卯":"申","未":"申","巳":"寅","酉":"寅","丑":"寅"}
	if t, ok := jiesha[dayZhi]; ok {
		addByBranch("劫煞", "bad", "日支", "主是非、破财、意外", []string{t})
	}

	// 11. 灾煞 — 日支
	zaisha := map[string]string{"申":"午","子":"午","辰":"午","寅":"子","午":"子","戌":"子","亥":"酉","卯":"酉","未":"酉","巳":"卯","酉":"卯","丑":"卯"}
	if t, ok := zaisha[dayZhi]; ok {
		addByBranch("灾煞", "bad", "日支", "主灾祸、疾病、横祸", []string{t})
	}

	// 12. 将星 — 日支
	jiangxing := map[string]string{"申":"子","子":"子","辰":"子","寅":"午","午":"午","戌":"午","亥":"卯","卯":"卯","未":"卯","巳":"酉","酉":"酉","丑":"酉"}
	if t, ok := jiangxing[dayZhi]; ok {
		addByBranch("将星", "neutral", "日支", "主领导力、掌权", []string{t})
	}

	// 13. 天罗地网 — 日支
	// 戌亥为天罗，辰巳为地网
	dayZhiTLDW := map[string]struct{ Name, Desc string }{
		"戌": {"天罗", "主困顿、官非"},
		"亥": {"天罗", "主困顿、官非"},
		"辰": {"地网", "主困顿、牢狱"},
		"巳": {"地网", "主困顿、牢狱"},
	}
	if info, ok := dayZhiTLDW[dayZhi]; ok {
		addToDayPillar(info.Name, "bad", "日支", info.Desc)
	}

	// 14. 魁罡 — 日柱（庚辰、庚戌、壬辰、戊戌）
	dayCombo := dayGan + dayZhi
	if dayCombo == "庚辰" || dayCombo == "庚戌" || dayCombo == "壬辰" || dayCombo == "戊戌" {
		addToDayPillar("魁罡", "neutral", "日柱", "主刚毅果断，行事凌厉")
	}

	// 15. 空亡 — from xunKong data
	for i, xk := range xunKong {
		if xk != "" && xk != "—" && xk != "无" {
			byPillar[pillarNames[i]] = append(byPillar[pillarNames[i]], ShenshaItem{
				Name: "空亡", Tone: "neutral", Basis: "日柱", Description: "主虚浮、不实、落空",
			})
		}
	}

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
