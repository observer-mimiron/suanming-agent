// This file belongs to the Zi Wei deterministic calculation layer.
// It owns Zi Wei minor-star placement for this package.
// It computes reproducible Zi Wei facts; it must not compose user-facing readings.
package ziwei

// GetMinorStar 安14辅星。辅星与主星配合解读命盘，影响主星的吉凶程度和具体应事。
//
// 左辅右弼（按月）、文昌文曲（按时）、天魁天钺（按年干）为吉星；
// 禄存（按年干）、天马（按年支）为财禄动星；
// 擎羊陀罗（禄存前后）、火星铃星（按年支+时支）、地空地劫（按时支）为煞星。
// 每种辅星有各自的算法规则，分布在不同宫位。
func GetMinorStar(yearStem, yearBranch string, timeIndex int, lunarMonth int) [12][]ZiWeiStar {
	var stars [12][]ZiWeiStar
	si := stemIndex(yearStem)

	lu, yang, tuo, ma := GetLuYangTuoMaIndex(yearStem, yearBranch)
	kui, yue := GetKuiYueIndex(yearStem)
	zuo, you := GetZuoYouIndex(lunarMonth)
	chang, qu := GetChangQuIndex(timeIndex)
	huo, ling := GetHuoLingIndex(yearBranch, timeIndex)
	kong, jie := GetKongJieIndex(timeIndex)

	stars[lu] = append(stars[lu], ZiWeiStar{Name: "禄存", Type: "lucun", Brightness: GetBrightness("禄存", lu)})
	stars[ma] = append(stars[ma], ZiWeiStar{Name: "天马", Type: "tianma", Brightness: GetBrightness("天马", ma)})
	stars[kui] = append(stars[kui], ZiWeiStar{Name: "天魁", Type: "soft", Brightness: GetBrightness("天魁", kui)})
	stars[yue] = append(stars[yue], ZiWeiStar{Name: "天钺", Type: "soft", Brightness: GetBrightness("天钺", yue)})
	stars[zuo] = append(stars[zuo], ZiWeiStar{Name: "左辅", Type: "soft", Brightness: GetBrightness("左辅", zuo), Mutagen: GetMutagen(si, "左辅")})
	stars[you] = append(stars[you], ZiWeiStar{Name: "右弼", Type: "soft", Brightness: GetBrightness("右弼", you), Mutagen: GetMutagen(si, "右弼")})
	stars[chang] = append(stars[chang], ZiWeiStar{Name: "文昌", Type: "soft", Brightness: GetBrightness("文昌", chang), Mutagen: GetMutagen(si, "文昌")})
	stars[qu] = append(stars[qu], ZiWeiStar{Name: "文曲", Type: "soft", Brightness: GetBrightness("文曲", qu), Mutagen: GetMutagen(si, "文曲")})
	stars[yang] = append(stars[yang], ZiWeiStar{Name: "擎羊", Type: "tough", Brightness: GetBrightness("擎羊", yang)})
	stars[tuo] = append(stars[tuo], ZiWeiStar{Name: "陀罗", Type: "tough", Brightness: GetBrightness("陀罗", tuo)})
	stars[huo] = append(stars[huo], ZiWeiStar{Name: "火星", Type: "tough", Brightness: GetBrightness("火星", huo)})
	stars[ling] = append(stars[ling], ZiWeiStar{Name: "铃星", Type: "tough", Brightness: GetBrightness("铃星", ling)})
	stars[kong] = append(stars[kong], ZiWeiStar{Name: "地空", Type: "tough", Brightness: GetBrightness("地空", kong)})
	stars[jie] = append(stars[jie], ZiWeiStar{Name: "地劫", Type: "tough", Brightness: GetBrightness("地劫", jie)})

	return stars
}
