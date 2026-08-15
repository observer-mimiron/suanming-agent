// 本文件属于紫微 adapter 层。
// 本文件只保留既有同包调用名，并将纯星曜规则委托给 domain；不拥有第二套算法或 map 投影。
// lunar-go、工具参数、Session、模型、trace、SSE 和最终文本仍由其他 adapter 文件负责。
package adapter

import ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"

// 这些名称保留既有 adapter 内部调用合同；数据和规则的唯一 owner 是 domain。
const (
	Water2 = ziweidomain.Water2
	Wood3  = ziweidomain.Wood3
	Metal4 = ziweidomain.Metal4
	Earth5 = ziweidomain.Earth5
	Fire6  = ziweidomain.Fire6
)

var (
	HeavenStems       = ziweidomain.HeavenStems
	EarthBranches     = ziweidomain.EarthBranches
	PalaceNames       = ziweidomain.PalaceNames
	SoulMaster        = ziweidomain.SoulMaster
	BodyMaster        = ziweidomain.BodyMaster
	BranchYinYang     = ziweidomain.BranchYinYang
	TigerRule         = ziweidomain.TigerRule
	RatRule           = ziweidomain.RatRule
	MutagenTable      = ziweidomain.MutagenTable
	ZiweiGroup        = ziweidomain.ZiweiGroup
	TianfuGroup       = ziweidomain.TianfuGroup
	StarBrightness    = ziweidomain.StarBrightness
	ChangSheng12Names = ziweidomain.ChangSheng12Names
	BoShi12Names      = ziweidomain.BoShi12Names
)

// FixIndex 将 adapter 内部的循环索引调用转发到 domain。
func FixIndex(index, max int) int { return ziweidomain.FixIndex(index, max) }

// FixIndex12 将 adapter 内部的十二宫索引调用转发到 domain。
func FixIndex12(index int) int { return ziweidomain.FixIndex12(index) }

// FixIndex10 将 adapter 内部的十干索引调用转发到 domain。
func FixIndex10(index int) int { return ziweidomain.FixIndex10(index) }

// FixEarthlyBranchIndex 将 adapter 内部的地支索引调用转发到 domain。
func FixEarthlyBranchIndex(branch string) int { return ziweidomain.FixEarthlyBranchIndex(branch) }

// branchIndex 保留 adapter 内部地支索引调用并委托 domain。
func branchIndex(branch string) int { return ziweidomain.BranchIndex(branch) }

// stemIndex 保留 adapter 内部天干索引调用并委托 domain。
func stemIndex(stem string) int { return ziweidomain.StemIndex(stem) }

// TimeToIndex 将 adapter 内部时辰索引调用转发到 domain。
func TimeToIndex(hour int) int { return ziweidomain.TimeToIndex(hour) }

// GetBrightness 将 adapter 内部亮度查询转发到 domain。
func GetBrightness(starName string, palaceIndex int) string {
	return ziweidomain.GetBrightness(starName, palaceIndex)
}

// GetMutagen 将 adapter 内部四化查询转发到 domain。
func GetMutagen(yearStemIndex int, starName string) string {
	return ziweidomain.GetMutagen(yearStemIndex, starName)
}

// GetAgeIndex 将 adapter 内部小限起始宫位查询转发到 domain。
func GetAgeIndex(yearBranch string) int { return ziweidomain.GetAgeIndex(yearBranch) }

// GetMajorStar 将 adapter 内部主星排布调用转发到 domain。
func GetMajorStar(ziweiIndex, tianfuIndex, yearStemIndex int) [12][]ZiWeiStar {
	return ziweidomain.GetMajorStar(ziweiIndex, tianfuIndex, yearStemIndex)
}
