package ziwei

// GetMajorStar 安主星（紫微星系 + 天府星系）
//
// 紫微逆去天机星，隔一太阳武曲辰，连接天同空二宫，廉贞居处方是真。
// 天府顺行有太阴，贪狼而后巨门临，随来天相天梁继，七杀空三是破军。
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
