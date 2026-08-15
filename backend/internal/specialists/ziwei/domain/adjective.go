// 本文件属于紫微 domain 层。
// 本文件负责年、月、日、时杂曜的纯宫位组装，输入只包含已归一的领域索引。
// 不负责 lunar-go、命盘组装、工具参数、Session、模型、trace、SSE 或最终文本。
package domain

// GetAdjectiveStar 安杂曜，返回按十二宫分组的无标签星曜值对象。
func GetAdjectiveStar(yearly YearlyStarIndex, monthly MonthlyStarIndex, daily DailyStarIndex, timely TimelyStarIndex, hongluan, tianxi int) [12][]ZiWeiStar {
	var stars [12][]ZiWeiStar

	stars[hongluan] = append(stars[hongluan], ZiWeiStar{Name: "红鸾", Type: "flower"})
	stars[tianxi] = append(stars[tianxi], ZiWeiStar{Name: "天喜", Type: "flower"})
	stars[monthly.Tianyao] = append(stars[monthly.Tianyao], ZiWeiStar{Name: "天姚", Type: "flower"})
	stars[yearly.Xianchi] = append(stars[yearly.Xianchi], ZiWeiStar{Name: "咸池", Type: "flower"})
	stars[monthly.Yuejie] = append(stars[monthly.Yuejie], ZiWeiStar{Name: "解神", Type: "helper"})
	stars[daily.Santai] = append(stars[daily.Santai], ZiWeiStar{Name: "三台", Type: "adjective"})
	stars[daily.Bazuo] = append(stars[daily.Bazuo], ZiWeiStar{Name: "八座", Type: "adjective"})
	stars[daily.Enguang] = append(stars[daily.Enguang], ZiWeiStar{Name: "恩光", Type: "adjective"})
	stars[daily.Tiangui] = append(stars[daily.Tiangui], ZiWeiStar{Name: "天贵", Type: "adjective"})
	stars[yearly.Longchi] = append(stars[yearly.Longchi], ZiWeiStar{Name: "龙池", Type: "adjective"})
	stars[yearly.Fengge] = append(stars[yearly.Fengge], ZiWeiStar{Name: "凤阁", Type: "adjective"})
	stars[yearly.Tiancai] = append(stars[yearly.Tiancai], ZiWeiStar{Name: "天才", Type: "adjective"})
	stars[yearly.Tianshou] = append(stars[yearly.Tianshou], ZiWeiStar{Name: "天寿", Type: "adjective"})
	stars[timely.Taifu] = append(stars[timely.Taifu], ZiWeiStar{Name: "台辅", Type: "adjective"})
	stars[timely.Fenggao] = append(stars[timely.Fenggao], ZiWeiStar{Name: "封诰", Type: "adjective"})
	stars[monthly.Tianwu] = append(stars[monthly.Tianwu], ZiWeiStar{Name: "天巫", Type: "adjective"})
	stars[yearly.Huagai] = append(stars[yearly.Huagai], ZiWeiStar{Name: "华盖", Type: "adjective"})
	stars[yearly.Tianguan] = append(stars[yearly.Tianguan], ZiWeiStar{Name: "天官", Type: "adjective"})
	stars[yearly.Tianfu] = append(stars[yearly.Tianfu], ZiWeiStar{Name: "天福", Type: "adjective"})
	stars[yearly.Tianchu] = append(stars[yearly.Tianchu], ZiWeiStar{Name: "天厨", Type: "adjective"})
	stars[monthly.Tianyue] = append(stars[monthly.Tianyue], ZiWeiStar{Name: "天月", Type: "adjective"})
	stars[yearly.Tiande] = append(stars[yearly.Tiande], ZiWeiStar{Name: "天德", Type: "adjective"})
	stars[yearly.Yuede] = append(stars[yearly.Yuede], ZiWeiStar{Name: "月德", Type: "adjective"})
	stars[yearly.Tiankong] = append(stars[yearly.Tiankong], ZiWeiStar{Name: "天空", Type: "adjective"})
	stars[yearly.Xunkong] = append(stars[yearly.Xunkong], ZiWeiStar{Name: "旬空", Type: "adjective"})
	stars[yearly.Jielu] = append(stars[yearly.Jielu], ZiWeiStar{Name: "截路", Type: "adjective"})
	stars[yearly.Kongwang] = append(stars[yearly.Kongwang], ZiWeiStar{Name: "空亡", Type: "adjective"})
	stars[yearly.Guchen] = append(stars[yearly.Guchen], ZiWeiStar{Name: "孤辰", Type: "adjective"})
	stars[yearly.Guasu] = append(stars[yearly.Guasu], ZiWeiStar{Name: "寡宿", Type: "adjective"})
	stars[yearly.Feilian] = append(stars[yearly.Feilian], ZiWeiStar{Name: "蜚蠊", Type: "adjective"})
	stars[yearly.Posui] = append(stars[yearly.Posui], ZiWeiStar{Name: "破碎", Type: "adjective"})
	stars[monthly.Tianxing] = append(stars[monthly.Tianxing], ZiWeiStar{Name: "天刑", Type: "tough"})
	stars[monthly.Yinsha] = append(stars[monthly.Yinsha], ZiWeiStar{Name: "阴煞", Type: "tough"})
	stars[yearly.Tianku] = append(stars[yearly.Tianku], ZiWeiStar{Name: "天哭", Type: "adjective"})
	stars[yearly.Tianxu] = append(stars[yearly.Tianxu], ZiWeiStar{Name: "天虚", Type: "adjective"})
	stars[yearly.Tianshi] = append(stars[yearly.Tianshi], ZiWeiStar{Name: "天使", Type: "adjective"})
	stars[yearly.Tianshang] = append(stars[yearly.Tianshang], ZiWeiStar{Name: "天伤", Type: "adjective"})
	stars[yearly.Nianjie] = append(stars[yearly.Nianjie], ZiWeiStar{Name: "年解", Type: "helper"})
	stars[yearly.Jiesha] = append(stars[yearly.Jiesha], ZiWeiStar{Name: "劫杀", Type: "adjective"})
	stars[yearly.Dahao] = append(stars[yearly.Dahao], ZiWeiStar{Name: "大耗", Type: "adjective"})

	return stars
}
