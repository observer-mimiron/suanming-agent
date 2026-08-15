// 本文件属于紫微 domain 层测试。
// 本文件锁定五行局、宫名旋转和起紫微纯规则，不构造 lunar-go 或工具 adapter。
package domain

import "testing"

func TestPalaceRules(t *testing.T) {
	if name, number := GetFiveElementsClass("丙", "子"); name != "水二局" || number != Water2 {
		t.Fatalf("GetFiveElementsClass(丙, 子) = %q/%d, want 水二局/%d", name, number, Water2)
	}

	names := GetPalaceNames(10)
	if names[0] != "福德" || names[4] != "迁移" {
		t.Fatalf("GetPalaceNames(10)[0,4] = %q/%q, want 福德/迁移", names[0], names[4])
	}

	tests := []struct {
		name                  string
		lunarDay, fiveElemNum int
		wantZiwei, wantTianfu int
	}{
		{name: "wood three", lunarDay: 8, fiveElemNum: Wood3, wantZiwei: 1, wantTianfu: 11},
		{name: "fire six", lunarDay: 8, fiveElemNum: Fire6, wantZiwei: 5, wantTianfu: 7},
		{name: "late child adjusted day", lunarDay: 9, fiveElemNum: Fire6, wantZiwei: 10, wantTianfu: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ziwei, tianfu := GetZiweiStartIndex(tt.lunarDay, tt.fiveElemNum)
			if ziwei != tt.wantZiwei || tianfu != tt.wantTianfu {
				t.Fatalf("GetZiweiStartIndex(%d, %d) = %d/%d, want %d/%d", tt.lunarDay, tt.fiveElemNum, ziwei, tianfu, tt.wantZiwei, tt.wantTianfu)
			}
		})
	}
}

func TestSoulAndBodyRules(t *testing.T) {
	tests := []struct {
		name                                       string
		monthIndex, timeBranchIndex, yearStemIndex int
		wantSoul, wantBody                         int
		wantStem, wantBranch                       int
	}{
		{name: "寅月子时甲年", monthIndex: 0, timeBranchIndex: 0, yearStemIndex: 0, wantSoul: 0, wantBody: 0, wantStem: 2, wantBranch: 2},
		{name: "归一月序与逆行命宫", monthIndex: 4, timeBranchIndex: 3, yearStemIndex: 6, wantSoul: 1, wantBody: 7, wantStem: 5, wantBranch: 3},
		{name: "负向循环", monthIndex: 1, timeBranchIndex: 11, yearStemIndex: 9, wantSoul: 2, wantBody: 0, wantStem: 2, wantBranch: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSoulAndBody(tt.monthIndex, tt.timeBranchIndex, tt.yearStemIndex)
			if got.SoulIndex != tt.wantSoul || got.BodyIndex != tt.wantBody {
				t.Fatalf("GetSoulAndBody(%d, %d, %d) indexes = %d/%d, want %d/%d", tt.monthIndex, tt.timeBranchIndex, tt.yearStemIndex, got.SoulIndex, got.BodyIndex, tt.wantSoul, tt.wantBody)
			}
			if got.HeavenlyStemIndex != tt.wantStem || got.EarthlyBranchIndex != tt.wantBranch {
				t.Fatalf("GetSoulAndBody(%d, %d, %d) ganzhi = %d/%d, want %d/%d", tt.monthIndex, tt.timeBranchIndex, tt.yearStemIndex, got.HeavenlyStemIndex, got.EarthlyBranchIndex, tt.wantStem, tt.wantBranch)
			}
		})
	}
}
