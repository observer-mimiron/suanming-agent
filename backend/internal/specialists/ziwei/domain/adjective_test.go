package domain

import "testing"

func TestAdjectiveStarAssembly(t *testing.T) {
	indices := YearlyStarIndex{Xianchi: 0, Huagai: 1, Guchen: 2, Guasu: 3, Tiancai: 4, Tianshou: 5, Tianchu: 6, Posui: 7, Feilian: 8, Longchi: 9, Fengge: 10, Tianku: 11, Tianxu: 0, Tianguan: 1, Tianfu: 2, Tiande: 3, Yuede: 4, Tiankong: 5, Jielu: 6, Kongwang: 7, Xunkong: 8, Tianshi: 9, Tianshang: 10, Jiesha: 11, Nianjie: 0, Dahao: 1}
	monthly := MonthlyStarIndex{Yuejie: 2, Tianyao: 3, Tianxing: 4, Yinsha: 5, Tianyue: 6, Tianwu: 7}
	daily := DailyStarIndex{Santai: 8, Bazuo: 9, Enguang: 10, Tiangui: 11}
	timely := TimelyStarIndex{Taifu: 0, Fenggao: 1}

	stars := GetAdjectiveStar(indices, monthly, daily, timely, 10, 11)
	wantNames := [12][]string{
		{"咸池", "台辅", "天虚", "年解"},
		{"封诰", "华盖", "天官", "大耗"},
		{"解神", "天福", "孤辰"},
		{"天姚", "天德", "寡宿"},
		{"天才", "月德", "天刑"},
		{"天寿", "天空", "阴煞"},
		{"天厨", "天月", "截路"},
		{"天巫", "空亡", "破碎"},
		{"三台", "旬空", "蜚蠊"},
		{"八座", "龙池", "天使"},
		{"红鸾", "恩光", "凤阁", "天伤"},
		{"天喜", "天贵", "天哭", "劫杀"},
	}
	for palace, want := range wantNames {
		if len(stars[palace]) != len(want) {
			t.Fatalf("宫位 %d 杂曜数 = %d, want %d", palace, len(stars[palace]), len(want))
		}
		for i, name := range want {
			if stars[palace][i].Name != name {
				t.Fatalf("宫位 %d 第 %d 星曜 = %q, want %q", palace, i, stars[palace][i].Name, name)
			}
		}
	}

	want := map[string]string{"红鸾": "flower", "天喜": "flower", "解神": "helper", "天刑": "tough", "阴煞": "tough", "大耗": "adjective"}
	for name, starType := range want {
		found := false
		for _, palace := range stars {
			for _, star := range palace {
				if star.Name == name {
					found = true
					if star.Type != starType {
						t.Fatalf("%s 类型 = %q, want %q", name, star.Type, starType)
					}
				}
			}
		}
		if !found {
			t.Fatalf("缺少杂曜 %s", name)
		}
	}
}
