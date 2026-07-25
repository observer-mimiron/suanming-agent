package ziwei

// GetMajorStar 安主星（紫微星系 + 天府星系共14主星）。主星分布决定命盘的基本格局。
//
// 紫微星系（6颗，逆时针排列）：紫微→天机→（空一宫）→太阳→武曲→天同→（空二宫）→廉贞。
// 口诀：紫微逆去天机星，隔一太阳武曲辰，连接天同空二宫，廉贞居处方是真。
//
// 天府星系（8颗，顺时针排列）：天府→太阴→贪狼→巨门→天相→天梁→七杀→（空三宫）→破军。
// 口诀：天府顺行有太阴，贪狼而后巨门临，随来天相天梁继，七杀空三是破军。
//
// 紫微星位置由命宫干支和农历日数推算，天府星位置与紫微星位置成镜像对称（紫微+天府=12）。
func GetMajorStar(ziweiIndex, tianfuIndex int, yearStemIndex int) [12][]ZiWeiStar {
	var stars [12][]ZiWeiStar

	// 安紫微星系（逆时针）
	for i, name := range ZiweiGroup {
		if name == "" {
			continue
		}
		idx := FixIndex12(ziweiIndex - i)
		stars[idx] = append(stars[idx], ZiWeiStar{
			Name:       name,
			Type:       "major",
			Brightness: GetBrightness(name, idx),
			Mutagen:    GetMutagen(yearStemIndex, name),
		})
	}

	// 安天府星系（顺时针）
	for i, name := range TianfuGroup {
		if name == "" {
			continue
		}
		idx := FixIndex12(tianfuIndex + i)
		stars[idx] = append(stars[idx], ZiWeiStar{
			Name:       name,
			Type:       "major",
			Brightness: GetBrightness(name, idx),
			Mutagen:    GetMutagen(yearStemIndex, name),
		})
	}

	return stars
}
