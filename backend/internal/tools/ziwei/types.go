package ziwei

// ZiWeiStar 星曜。紫微斗数中每个宫位包含的星体，包括主星（紫微天府等14颗）、
// 辅星（左辅右弼等14颗）、杂曜（红鸾天喜等数十颗）。Type 字段区分星曜类别，
// Brightness 表示庙旺利得平陷（星曜在该宫位的影响力等级），Mutagen 表示四化（化禄权科忌）。
type ZiWeiStar struct {
	Name       string `json:"name"`
	Type       string `json:"type"`                 // 主星、吉星、煞星、杂曜、辅星、禄存、天马
	Brightness string `json:"brightness,omitempty"` // 庙旺得利平不陷
	Mutagen    string `json:"mutagen,omitempty"`    // 化禄/权/科/忌
}

// DecadalInfo 大限信息。表示命盘每十年的大运区间，包含起止年龄和该限的干支。
type DecadalInfo struct {
	StartAge      int    `json:"start_age"`
	EndAge        int    `json:"end_age"`
	HeavenlyStem  string `json:"heavenly_stem"`
	EarthlyBranch string `json:"earthly_branch"`
}

// ZiWeiPalace 紫微斗数十二宫之一。每个宫位包含宫名（命宫/兄弟/夫妻/子女/财帛/疾厄/迁移/交友/官禄/田宅/福德/父母）、
// 宫干宫支、主星辅星杂曜列表、长生十二神、博士十二神以及该宫所管的大限信息。
// IsBodyPalace 标识是否为身宫，IsOriginalPalace 标识是否为来因宫。
type ZiWeiPalace struct {
	Index            int         `json:"index"`              // 0-11（从寅开始）
	Name             string      `json:"name"`               // 宫名（命宫/兄弟/夫妻...）
	HeavenlyStem     string      `json:"heavenly_stem"`      // 宫干
	EarthlyBranch    string      `json:"earthly_branch"`     // 宫支
	IsBodyPalace     bool        `json:"is_body_palace"`     // 是否身宫
	IsOriginalPalace bool        `json:"is_original_palace"` // 来因宫
	MajorStars       []ZiWeiStar `json:"major_stars"`        // 主星
	MinorStars       []ZiWeiStar `json:"minor_stars"`        // 辅星
	AdjectiveStars   []ZiWeiStar `json:"adjective_stars"`    // 杂曜
	ChangSheng12     string      `json:"changsheng_12"`      // 长生十二神
	BoShi12          string      `json:"boshi_12"`           // 博士十二神
	Decadal          DecadalInfo `json:"decadal"`            // 大限
}

// ZiWeiChart 紫微斗数命盘。包含命盘主信息：性别、公历/农历日期、四柱八字、
// 命宫身宫干支、五行局（水二局/木三局/金四局/土五局/火六局）、命主身主和十二宫详细信息。
type ZiWeiChart struct {
	Gender               string            `json:"gender"`                  // 男/女
	SolarDate            string            `json:"solar_date"`              // 公历日期
	LunarDate            string            `json:"lunar_date"`              // 农历日期
	FourPillars          map[string]string `json:"four_pillars"`            // 年柱/月柱/日柱/时柱
	SoulPalaceBranch     string            `json:"soul_palace_branch"`      // 命宫地支
	BodyPalaceBranch     string            `json:"body_palace_branch"`      // 身宫地支
	SoulPalaceGanZhi     string            `json:"soul_palace_ganzhi"`      // 命宫干支
	FiveElementsClass    string            `json:"five_elements_class"`     // 水二局/木三局...
	FiveElementsClassNum int               `json:"five_elements_class_num"` // 五行局数值 2-6
	SoulMaster           string            `json:"soul_master"`             // 命主
	BodyMaster           string            `json:"body_master"`             // 身主
	Palaces              []ZiWeiPalace     `json:"palaces"`                 // 十二宫
}

// ToMap 将紫微斗数命盘转为通用 map 结构，以适配 Tool 接口的 Execute 返回值格式。
func (c *ZiWeiChart) ToMap() map[string]any {
	palaces := make([]map[string]any, 12)
	for i, p := range c.Palaces {
		palaces[i] = map[string]any{
			"index":              p.Index,
			"name":               p.Name,
			"heavenly_stem":      p.HeavenlyStem,
			"earthly_branch":     p.EarthlyBranch,
			"is_body_palace":     p.IsBodyPalace,
			"is_original_palace": p.IsOriginalPalace,
			"major_stars":        p.MajorStars,
			"minor_stars":        p.MinorStars,
			"adjective_stars":    p.AdjectiveStars,
			"changsheng_12":      p.ChangSheng12,
			"boshi_12":           p.BoShi12,
			"decadal":            p.Decadal,
		}
	}

	return map[string]any{
		"gender":                  c.Gender,
		"solar_date":              c.SolarDate,
		"lunar_date":              c.LunarDate,
		"four_pillars":            c.FourPillars,
		"soul_palace_branch":      c.SoulPalaceBranch,
		"body_palace_branch":      c.BodyPalaceBranch,
		"soul_palace_ganzhi":      c.SoulPalaceGanZhi,
		"five_elements_class":     c.FiveElementsClass,
		"five_elements_class_num": c.FiveElementsClassNum,
		"soul_master":             c.SoulMaster,
		"body_master":             c.BodyMaster,
		"palaces":                 palaces,
	}
}
