package ziwei

// ZiWeiStar 星曜
type ZiWeiStar struct {
	Name       string `json:"name"`
	Type       string `json:"type"`                 // major, soft, tough, adjective, helper, lucun, tianma
	Brightness string `json:"brightness,omitempty"` // 庙旺得利平不陷
	Mutagen    string `json:"mutagen,omitempty"`    // 化禄/权/科/忌
}

// DecadalInfo 大限信息
type DecadalInfo struct {
	StartAge      int    `json:"start_age"`
	EndAge        int    `json:"end_age"`
	HeavenlyStem  string `json:"heavenly_stem"`
	EarthlyBranch string `json:"earthly_branch"`
}

// ZiWeiPalace 紫微斗数宫位
type ZiWeiPalace struct {
	Index            int          `json:"index"`              // 0-11（从寅开始）
	Name             string       `json:"name"`               // 宫名（命宫/兄弟/夫妻...）
	HeavenlyStem     string       `json:"heavenly_stem"`      // 宫干
	EarthlyBranch    string       `json:"earthly_branch"`     // 宫支
	IsBodyPalace     bool         `json:"is_body_palace"`     // 是否身宫
	IsOriginalPalace bool         `json:"is_original_palace"` // 来因宫
	MajorStars       []ZiWeiStar  `json:"major_stars"`        // 主星
	MinorStars       []ZiWeiStar  `json:"minor_stars"`        // 辅星
	AdjectiveStars   []ZiWeiStar  `json:"adjective_stars"`    // 杂曜
	ChangSheng12     string       `json:"changsheng_12"`      // 长生十二神
	BoShi12          string       `json:"boshi_12"`           // 博士十二神
	Decadal          DecadalInfo  `json:"decadal"`            // 大限
}

// ZiWeiChart 紫微斗数命盘
type ZiWeiChart struct {
	Gender                string            `json:"gender"`                   // 男/女
	SolarDate             string            `json:"solar_date"`               // 公历日期
	LunarDate             string            `json:"lunar_date"`               // 农历日期
	FourPillars           map[string]string `json:"four_pillars"`             // 年柱/月柱/日柱/时柱
	SoulPalaceBranch      string            `json:"soul_palace_branch"`       // 命宫地支
	BodyPalaceBranch      string            `json:"body_palace_branch"`       // 身宫地支
	SoulPalaceGanZhi      string            `json:"soul_palace_ganzhi"`       // 命宫干支
	FiveElementsClass     string            `json:"five_elements_class"`      // 水二局/木三局...
	FiveElementsClassNum  int               `json:"five_elements_class_num"`  // 五行局数值 2-6
	SoulMaster            string            `json:"soul_master"`              // 命主
	BodyMaster            string            `json:"body_master"`              // 身主
	Palaces               []ZiWeiPalace     `json:"palaces"`                  // 十二宫
}

// ToMap 将命盘转为 map[string]any（适配 Tool 接口）
func (c *ZiWeiChart) ToMap() map[string]any {
	palaces := make([]map[string]any, 12)
	for i, p := range c.Palaces {
		palaces[i] = map[string]any{
			"index":             p.Index,
			"name":              p.Name,
			"heavenly_stem":     p.HeavenlyStem,
			"earthly_branch":    p.EarthlyBranch,
			"is_body_palace":    p.IsBodyPalace,
			"is_original_palace": p.IsOriginalPalace,
			"major_stars":       p.MajorStars,
			"minor_stars":       p.MinorStars,
			"adjective_stars":   p.AdjectiveStars,
			"changsheng_12":     p.ChangSheng12,
			"boshi_12":          p.BoShi12,
			"decadal":           p.Decadal,
		}
	}

	return map[string]any{
		"gender":                 c.Gender,
		"solar_date":             c.SolarDate,
		"lunar_date":             c.LunarDate,
		"four_pillars":           c.FourPillars,
		"soul_palace_branch":     c.SoulPalaceBranch,
		"body_palace_branch":     c.BodyPalaceBranch,
		"soul_palace_ganzhi":     c.SoulPalaceGanZhi,
		"five_elements_class":    c.FiveElementsClass,
		"five_elements_class_num": c.FiveElementsClassNum,
		"soul_master":            c.SoulMaster,
		"body_master":            c.BodyMaster,
		"palaces":                palaces,
	}
}
