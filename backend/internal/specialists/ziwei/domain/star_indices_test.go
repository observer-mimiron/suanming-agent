package domain

import "testing"

func TestPureStarIndices(t *testing.T) {
	tests := []struct {
		name string
		got  [2]int
		want [2]int
	}{
		{name: "kui yue", got: pair(GetKuiYueIndex("甲")), want: [2]int{11, 5}},
		{name: "zuo you", got: pair(GetZuoYouIndex(1)), want: [2]int{2, 8}},
		{name: "chang qu", got: pair(GetChangQuIndex(0)), want: [2]int{8, 2}},
		{name: "kong jie", got: pair(GetKongJieIndex(0)), want: [2]int{9, 9}},
		{name: "huo ling", got: pair(GetHuoLingIndex("寅", 0)), want: [2]int{11, 1}},
		{name: "luan xi", got: pair(GetLuanXiIndex("子")), want: [2]int{1, 7}},
		{name: "huagai xianchi", got: pair(GetHuagaiXianchiIndex("寅")), want: [2]int{8, 1}},
		{name: "gu gua", got: pair(GetGuGuaIndex("寅")), want: [2]int{3, 11}},
		{name: "tianshi tianshang", got: pair(GetTianshiTianshangIndex("男", "子", 0)), want: [2]int{6, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	timely := GetTimelyStarIndex(0)
	if timely.Taifu != 4 || timely.Fenggao != 0 {
		t.Fatalf("unexpected timely stars: %+v", timely)
	}
}

func TestPureStarIndicesYearly(t *testing.T) {
	got := GetYearlyStarIndex("男", "甲", "子", 0, 1)
	if got.Xianchi != 7 || got.Huagai != 2 || got.Guchen != 0 || got.Guasu != 8 {
		t.Fatalf("unexpected relationship stars: %+v", got)
	}
	if got.Tiancai != 0 || got.Tianshou != 1 || got.Tianchu != 3 || got.Posui != 3 {
		t.Fatalf("unexpected derived stars: %+v", got)
	}
	if got.Jiesha != 3 || got.Nianjie != 8 || got.Dahao != 5 || got.Tianshi != 6 || got.Tianshang != 5 {
		t.Fatalf("unexpected protection stars: %+v", got)
	}
}

func pair(a, b int) [2]int { return [2]int{a, b} }
