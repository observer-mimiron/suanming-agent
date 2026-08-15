// This test file belongs to the Zi Wei deterministic calculation layer.
// It verifies Zi Wei integration behavior and protects the related contract from regressions.
// It computes reproducible Zi Wei facts; it must not compose user-facing readings.
package adapter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/6tail/lunar-go/calendar"
	solartime "github.com/observer-mimiron/suanming-agent/internal/calendar"
)

// referenceCase holds iztro-generated expected values for a test case.
type referenceCase struct {
	name               string
	year, month, day   int
	hour               int
	gender             string
	fourPillars        map[string]string
	soulPalaceBranch   string
	bodyPalaceBranch   string
	fiveElementsClass  string
	fiveElementsNum    int
	soulMaster         string
	bodyMaster         string
	palaceDecadalStart map[string]int
	palaceMajorStars   map[string][]string
}

var refCases = []referenceCase{
	{
		name: "1990男_辰时", year: 1990, month: 5, day: 20, hour: 8, gender: "男",
		fourPillars:      map[string]string{"年柱": "庚午", "月柱": "辛巳", "日柱": "乙酉", "时柱": "庚辰"},
		soulPalaceBranch: "丑", bodyPalaceBranch: "酉",
		fiveElementsClass: "火六局", fiveElementsNum: Fire6, soulMaster: "巨门", bodyMaster: "火星",
		palaceDecadalStart: map[string]int{"命宫": 6, "父母": 16, "福德": 26, "田宅": 36, "官禄": 46, "交友": 56, "迁移": 66, "疾厄": 76, "财帛": 86, "子女": 96, "夫妻": 106, "兄弟": 116},
		palaceMajorStars:   map[string][]string{"命宫": {}, "父母": {"廉贞"}, "福德": {}, "田宅": {"破军"}, "官禄": {"天同"}, "交友": {"武曲", "天府"}, "迁移": {"太阳", "太阴"}, "疾厄": {"贪狼"}, "财帛": {"天机", "巨门"}, "子女": {"紫微", "天相"}, "夫妻": {"天梁"}, "兄弟": {"七杀"}},
	},
	{
		name: "1984女_未时", year: 1984, month: 2, day: 15, hour: 14, gender: "女",
		fourPillars:      map[string]string{"年柱": "甲子", "月柱": "丙寅", "日柱": "己卯", "时柱": "辛未"},
		soulPalaceBranch: "未", bodyPalaceBranch: "酉",
		fiveElementsClass: "土五局", fiveElementsNum: Earth5, soulMaster: "武曲", bodyMaster: "火星",
		palaceDecadalStart: map[string]int{"命宫": 5, "父母": 115, "福德": 105, "田宅": 95, "官禄": 85, "交友": 75, "迁移": 65, "疾厄": 55, "财帛": 45, "子女": 35, "夫妻": 25, "兄弟": 15},
		palaceMajorStars:   map[string][]string{"命宫": {"廉贞", "七杀"}, "父母": {}, "福德": {}, "田宅": {"天同"}, "官禄": {"武曲", "破军"}, "交友": {"太阳"}, "迁移": {"天府"}, "疾厄": {"天机", "太阴"}, "财帛": {"紫微", "贪狼"}, "子女": {"巨门"}, "夫妻": {"天相"}, "兄弟": {"天梁"}},
	},
	{
		name: "2000男_子时", year: 2000, month: 1, day: 1, hour: 0, gender: "男",
		fourPillars:      map[string]string{"年柱": "己卯", "月柱": "丙子", "日柱": "戊午", "时柱": "壬子"},
		soulPalaceBranch: "子", bodyPalaceBranch: "子",
		fiveElementsClass: "水二局", fiveElementsNum: Water2, soulMaster: "贪狼", bodyMaster: "天同",
		palaceDecadalStart: map[string]int{"命宫": 2, "父母": 112, "福德": 102, "田宅": 92, "官禄": 82, "交友": 72, "迁移": 62, "疾厄": 52, "财帛": 42, "子女": 32, "夫妻": 22, "兄弟": 12},
		palaceMajorStars:   map[string][]string{"命宫": {"天机"}, "父母": {"紫微", "破军"}, "福德": {}, "田宅": {"天府"}, "官禄": {"太阴"}, "交友": {"廉贞", "贪狼"}, "迁移": {"巨门"}, "疾厄": {"天相"}, "财帛": {"天同", "天梁"}, "子女": {"武曲", "七杀"}, "夫妻": {"太阳"}, "兄弟": {}},
	},
	{
		name: "1988晚子时", year: 1988, month: 8, day: 8, hour: 23, gender: "男",
		fourPillars:      map[string]string{"年柱": "戊辰", "月柱": "庚申", "日柱": "丙申", "时柱": "戊子"},
		soulPalaceBranch: "未", bodyPalaceBranch: "未",
		fiveElementsClass: "火六局", fiveElementsNum: Fire6, soulMaster: "武曲", bodyMaster: "文昌",
		palaceDecadalStart: map[string]int{"命宫": 6, "父母": 16, "福德": 26, "田宅": 36, "官禄": 46, "交友": 56, "迁移": 66, "疾厄": 76, "财帛": 86, "子女": 96, "夫妻": 106, "兄弟": 116},
		palaceMajorStars:   map[string][]string{"命宫": {"廉贞", "七杀"}, "父母": {}, "福德": {}, "田宅": {"天同"}, "官禄": {"武曲", "破军"}, "交友": {"太阳"}, "迁移": {"天府"}, "疾厄": {"天机", "太阴"}, "财帛": {"紫微", "贪狼"}, "子女": {"巨门"}, "夫妻": {"天相"}, "兄弟": {"天梁"}},
	},
	{
		// 木三局 + 阴男逆行
		name: "1995男_卯时_木三局", year: 1995, month: 11, day: 20, hour: 6, gender: "男",
		fourPillars:      map[string]string{"年柱": "乙亥", "月柱": "丁亥", "日柱": "乙卯", "时柱": "己卯"},
		soulPalaceBranch: "未", bodyPalaceBranch: "丑",
		fiveElementsClass: "木三局", fiveElementsNum: Wood3, soulMaster: "武曲", bodyMaster: "天机",
		palaceDecadalStart: map[string]int{"命宫": 3, "父母": 113, "福德": 103, "田宅": 93, "官禄": 83, "交友": 73, "迁移": 63, "疾厄": 53, "财帛": 43, "子女": 33, "夫妻": 23, "兄弟": 13},
		palaceMajorStars:   map[string][]string{"命宫": {"天相"}, "父母": {"天同", "天梁"}, "福德": {"武曲", "七杀"}, "田宅": {"太阳"}, "官禄": {}, "交友": {"天机"}, "迁移": {"紫微", "破军"}, "疾厄": {}, "财帛": {"天府"}, "子女": {"太阴"}, "夫妻": {"廉贞", "贪狼"}, "兄弟": {"巨门"}},
	},
	{
		// 木三局 + 阳女逆行
		name: "1986女_辰时_阳女", year: 1986, month: 3, day: 18, hour: 8, gender: "女",
		fourPillars:      map[string]string{"年柱": "丙寅", "月柱": "辛卯", "日柱": "辛酉", "时柱": "壬辰"},
		soulPalaceBranch: "亥", bodyPalaceBranch: "未",
		fiveElementsClass: "木三局", fiveElementsNum: Wood3, soulMaster: "巨门", bodyMaster: "天梁",
		palaceDecadalStart: map[string]int{"命宫": 3, "父母": 113, "福德": 103, "田宅": 93, "官禄": 83, "交友": 73, "迁移": 63, "疾厄": 53, "财帛": 43, "子女": 33, "夫妻": 23, "兄弟": 13},
		palaceMajorStars:   map[string][]string{"命宫": {"天同"}, "父母": {"武曲", "天府"}, "福德": {"太阳", "太阴"}, "田宅": {"贪狼"}, "官禄": {"天机", "巨门"}, "交友": {"紫微", "天相"}, "迁移": {"天梁"}, "疾厄": {"七杀"}, "财帛": {}, "子女": {"廉贞"}, "夫妻": {}, "兄弟": {"破军"}},
	},
	{
		// 金四局 + 七杀坐命 + 阳男顺行
		name: "1980男_辰时_金四局", year: 1980, month: 8, day: 20, hour: 8, gender: "男",
		fourPillars:      map[string]string{"年柱": "庚申", "月柱": "甲申", "日柱": "乙丑", "时柱": "庚辰"},
		soulPalaceBranch: "辰", bodyPalaceBranch: "子",
		fiveElementsClass: "金四局", fiveElementsNum: Metal4, soulMaster: "廉贞", bodyMaster: "天梁",
		palaceDecadalStart: map[string]int{"命宫": 4, "父母": 14, "福德": 24, "田宅": 34, "官禄": 44, "交友": 54, "迁移": 64, "疾厄": 74, "财帛": 84, "子女": 94, "夫妻": 104, "兄弟": 114},
		palaceMajorStars:   map[string][]string{"命宫": {"七杀"}, "父母": {"天机"}, "福德": {"紫微"}, "田宅": {}, "官禄": {"破军"}, "交友": {}, "迁移": {"廉贞", "天府"}, "疾厄": {"太阴"}, "财帛": {"贪狼"}, "子女": {"天同", "巨门"}, "夫妻": {"武曲", "天相"}, "兄弟": {"太阳", "天梁"}},
	},
}

func buildChartForCase(tc referenceCase) (*ZiWeiChart, error) {
	t := calendar.NewSolar(tc.year, tc.month, tc.day, tc.hour, 0, 0)
	ti := TimeToIndex(tc.hour)
	return BuildChart(t, ti, tc.gender)
}

func TestBuildChart_FourPillars(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			for key, want := range tc.fourPillars {
				if got := chart.FourPillars[key]; got != want {
					t.Errorf("%s: got %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestBuildChart_BasicInfo(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			if chart.SoulPalaceBranch != tc.soulPalaceBranch {
				t.Errorf("soulPalaceBranch: got %q, want %q", chart.SoulPalaceBranch, tc.soulPalaceBranch)
			}
			if chart.BodyPalaceBranch != tc.bodyPalaceBranch {
				t.Errorf("bodyPalaceBranch: got %q, want %q", chart.BodyPalaceBranch, tc.bodyPalaceBranch)
			}
			if chart.FiveElementsClass != tc.fiveElementsClass {
				t.Errorf("fiveElementsClass: got %q, want %q", chart.FiveElementsClass, tc.fiveElementsClass)
			}
			if chart.FiveElementsClassNum != tc.fiveElementsNum {
				t.Errorf("fiveElementsNum: got %d, want %d", chart.FiveElementsClassNum, tc.fiveElementsNum)
			}
			if chart.SoulMaster != tc.soulMaster {
				t.Errorf("soulMaster: got %q, want %q", chart.SoulMaster, tc.soulMaster)
			}
			if chart.BodyMaster != tc.bodyMaster {
				t.Errorf("bodyMaster: got %q, want %q", chart.BodyMaster, tc.bodyMaster)
			}
		})
	}
}

func TestBuildChart_MajorStars(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			for _, p := range chart.Palaces {
				wantStars := tc.palaceMajorStars[p.Name]
				gotStars := make([]string, len(p.MajorStars))
				for i, s := range p.MajorStars {
					gotStars[i] = s.Name
				}
				if len(gotStars) != len(wantStars) {
					t.Errorf("%s majorStars: got %v, want %v", p.Name, gotStars, wantStars)
					continue
				}
				for i, ws := range wantStars {
					if gotStars[i] != ws {
						t.Errorf("%s majorStars[%d]: got %q, want %q", p.Name, i, gotStars[i], ws)
					}
				}
			}
		})
	}
}

func TestBuildChart_DecadalRange(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			for _, p := range chart.Palaces {
				wantStart := tc.palaceDecadalStart[p.Name]
				if p.Decadal.StartAge != wantStart {
					t.Errorf("%s decadal start: got %d, want %d", p.Name, p.Decadal.StartAge, wantStart)
				}
			}
		})
	}
}

func TestBuildChart_BodyPalace(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			bodyCount := 0
			for _, p := range chart.Palaces {
				if p.IsBodyPalace {
					bodyCount++
					if p.EarthlyBranch != tc.bodyPalaceBranch {
						t.Errorf("bodyPalace earthly branch: got %q, want %q", p.EarthlyBranch, tc.bodyPalaceBranch)
					}
				}
			}
			if bodyCount != 1 {
				t.Errorf("expected exactly 1 body palace, got %d", bodyCount)
			}
		})
	}
}

func TestBuildChart_12Palaces(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			if len(chart.Palaces) != 12 {
				t.Fatalf("expected 12 palaces, got %d", len(chart.Palaces))
			}
			seen := make(map[string]bool)
			for _, p := range chart.Palaces {
				if seen[p.Name] {
					t.Errorf("duplicate palace name: %s", p.Name)
				}
				seen[p.Name] = true
			}
		})
	}
}

func TestBuildChart_ChangSheng12(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			for _, p := range chart.Palaces {
				if p.ChangSheng12 == "" {
					t.Errorf("%s has empty changSheng12", p.Name)
				}
			}
		})
	}
}

func TestBuildChart_BoShi12(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			for _, p := range chart.Palaces {
				if p.BoShi12 == "" {
					t.Errorf("%s has empty boShi12", p.Name)
				}
			}
		})
	}
}

func TestBuildChart_MinorStars(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			totalMinors := 0
			for _, p := range chart.Palaces {
				totalMinors += len(p.MinorStars)
			}
			if totalMinors < 14 {
				t.Errorf("expected at least 14 minor stars, got %d", totalMinors)
			}
		})
	}
}

func TestBuildChart_AdjectiveStars(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			totalAdj := 0
			for _, p := range chart.Palaces {
				totalAdj += len(p.AdjectiveStars)
			}
			if totalAdj < 20 {
				t.Errorf("expected at least 20 adjective stars, got %d", totalAdj)
			}
		})
	}
}

func TestBuildChart_Mutagen(t *testing.T) {
	for _, tc := range refCases {
		t.Run(tc.name, func(t *testing.T) {
			chart, err := buildChartForCase(tc)
			if err != nil {
				t.Fatalf("BuildChart() error: %v", err)
			}
			mutagenCount := 0
			mutagens := make(map[string]bool)
			for _, p := range chart.Palaces {
				for _, s := range p.MajorStars {
					if s.Mutagen != "" {
						mutagenCount++
						mutagens[s.Mutagen] = true
					}
				}
				for _, s := range p.MinorStars {
					if s.Mutagen != "" {
						mutagenCount++
						mutagens[s.Mutagen] = true
					}
				}
			}
			if mutagenCount != 4 {
				t.Errorf("expected 4 mutagen marks, got %d", mutagenCount)
			}
			for _, m := range []string{"化禄", "化权", "化科", "化忌"} {
				if !mutagens[m] {
					t.Errorf("expected mutagen %q not found", m)
				}
			}
		})
	}
}

func TestZiWeiChartToMap_StarPayloadContract(t *testing.T) {
	chart := &ZiWeiChart{Palaces: []ZiWeiPalace{{
		MajorStars: []ZiWeiStar{{Name: "紫微", Type: "major", Brightness: "庙", Mutagen: "化权"}},
		MinorStars: []ZiWeiStar{{Name: "禄存", Type: "lucun"}},
	}}}

	payload, err := json.Marshal(chart.ToMap())
	if err != nil {
		t.Fatalf("marshal chart payload: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal chart payload: %v", err)
	}
	palaces := got["palaces"].([]any)
	major := palaces[0].(map[string]any)["major_stars"].([]any)[0].(map[string]any)
	minor := palaces[0].(map[string]any)["minor_stars"].([]any)[0].(map[string]any)
	if major["name"] != "紫微" || major["type"] != "major" || major["brightness"] != "庙" || major["mutagen"] != "化权" {
		t.Fatalf("major star payload = %#v", major)
	}
	if minor["name"] != "禄存" || minor["type"] != "lucun" {
		t.Fatalf("minor star payload = %#v", minor)
	}
	if _, ok := minor["brightness"]; ok {
		t.Fatalf("minor star unexpectedly includes brightness: %#v", minor)
	}
	if _, ok := minor["mutagen"]; ok {
		t.Fatalf("minor star unexpectedly includes mutagen: %#v", minor)
	}
	if got := palaces[0].(map[string]any)["adjective_stars"]; got != nil {
		t.Fatalf("nil adjective stars payload = %#v, want nil", got)
	}
}

func TestFixIndex(t *testing.T) {
	tests := []struct{ idx, max, want int }{
		{0, 12, 0}, {11, 12, 11}, {12, 12, 0}, {13, 12, 1}, {-1, 12, 11}, {-2, 12, 10}, {-13, 12, 11},
		{0, 10, 0}, {9, 10, 9}, {10, 10, 0}, {-1, 10, 9},
	}
	for _, tt := range tests {
		if got := FixIndex(tt.idx, tt.max); got != tt.want {
			t.Errorf("FixIndex(%d, %d) = %d, want %d", tt.idx, tt.max, got, tt.want)
		}
	}
}

func TestFiveElementsClass(t *testing.T) {
	tests := []struct {
		stem, branch, className string
		classNum                int
	}{
		{"丙", "子", "水二局", Water2},
		{"辛", "未", "土五局", Earth5},
		{"庚", "申", "木三局", Wood3},
		{"甲", "子", "金四局", Metal4},
		{"戊", "辰", "木三局", Wood3},
	}
	for _, tt := range tests {
		name, num := GetFiveElementsClass(tt.stem, tt.branch)
		if name != tt.className {
			t.Errorf("GetFiveElementsClass(%s,%s) name = %q, want %q", tt.stem, tt.branch, name, tt.className)
		}
		if num != tt.classNum {
			t.Errorf("GetFiveElementsClass(%s,%s) num = %d, want %d", tt.stem, tt.branch, num, tt.classNum)
		}
	}
}

func TestLiuNian_2026(t *testing.T) {
	// 1990-05-20 08:00 男 → 2026年(丙午)流年
	solar := calendar.NewSolar(1990, 5, 20, 8, 0, 0)
	ti := TimeToIndex(8)
	chart, err := BuildChart(solar, ti, "男")
	if err != nil {
		t.Fatalf("BuildChart() error: %v", err)
	}

	ln := GetLiuNian(chart, 2026, 37) // 2026年虚岁37

	// 2026 = 丙午年
	if ln.YearStem != "丙" || ln.YearBranch != "午" {
		t.Errorf("year stem/branch: got %s%s, want 丙午", ln.YearStem, ln.YearBranch)
	}

	// 丙年四化: 天同化禄, 天机化权, 文昌化科, 廉贞化忌
	expected := [4]LiuNianMutagen{
		{"天同", "化禄"}, {"天机", "化权"}, {"文昌", "化科"}, {"廉贞", "化忌"},
	}
	for i, exp := range expected {
		if ln.Mutagens[i] != exp {
			t.Errorf("mutagen[%d]: got %v, want %v", i, ln.Mutagens[i], exp)
		}
	}

	// 小限：男顺行，午年→辰起，37-1=36步 → FixIndex12(4+36)=FixIndex12(40)=4 → 官禄宫(index 4)
	// 但要注意命盘实际的 soulIndex：1990男 soulIndex=10(丑)
	// index 4 in the chart is at earthly branch 午 (4+2=6→午), name depends on soulIndex
	// GetPalaceNames(10): idx=fixIndex(4-10)=fixIndex(-6)=6 → PALACES[6]="迁移"
	// Wait, that means index 4 in physical palace is "迁移"宫
	if ln.AgePalace == "" {
		t.Error("agePalace should not be empty")
	}
	t.Logf("1990男 2026年(丙午) 37岁 小限宫位: %s", ln.AgePalace)
	t.Logf("流年四化: %v", ln.Mutagens)
}

func TestLiuNian_YearStemCorrect(t *testing.T) {
	// 验证多个年份的干支正确
	tests := []struct {
		year                 int
		wantStem, wantBranch string
	}{
		{2024, "甲", "辰"},
		{2025, "乙", "巳"},
		{2026, "丙", "午"},
		{2027, "丁", "未"},
	}
	for _, tt := range tests {
		solar := calendar.NewSolar(tt.year, 6, 15, 12, 0, 0)
		lunar := solar.GetLunar()
		gotStem := lunar.GetYearGan()
		gotBranch := lunar.GetYearZhi()
		if gotStem != tt.wantStem || gotBranch != tt.wantBranch {
			t.Errorf("year %d: got %s%s, want %s%s", tt.year, gotStem, gotBranch, tt.wantStem, tt.wantBranch)
		}
	}
}

func TestTimeToIndex(t *testing.T) {
	tests := []struct{ hour, want int }{
		{0, 0}, {1, 1}, {3, 2}, {5, 3}, {7, 4}, {9, 5},
		{11, 6}, {13, 7}, {15, 8}, {17, 9}, {19, 10}, {21, 11}, {23, 12},
	}
	for _, tt := range tests {
		if got := TimeToIndex(tt.hour); got != tt.want {
			t.Errorf("TimeToIndex(%d) = %d, want %d", tt.hour, got, tt.want)
		}
	}
}

func TestZiWeiCalc_TrueSolarTimeCrossesMidnightInShanghai(t *testing.T) {
	tool := &ZiWeiCalcTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"year":      float64(2025),
		"month":     float64(11),
		"day":       float64(10),
		"hour":      float64(23),
		"minute":    float64(53),
		"longitude": 121.4737,
		"gender":    "男",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data := result.(map[string]any)
	if got := data["solar_time_version"]; got != solartime.TrueSolarTimeVersion {
		t.Fatalf("solar_time_version=%v, want %s", got, solartime.TrueSolarTimeVersion)
	}
	if got := data["birthday"]; got != "2025-11-11 00:15:00" {
		t.Fatalf("birthday=%v, want 2025-11-11 00:15:00", got)
	}
	pillars := data["four_pillars"].(map[string]string)
	if got := pillars["日柱"]; got != "甲申" {
		t.Fatalf("day pillar=%s, want 甲申 after true-solar midnight crossing", got)
	}
	if got := pillars["时柱"]; got != "甲子" {
		t.Fatalf("time pillar=%s, want 甲子 after true-solar midnight crossing", got)
	}
}
