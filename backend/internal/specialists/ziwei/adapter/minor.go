// 本文件属于紫微 adapter 层的确定性排盘实现。
// 本文件负责辅星和煞星的宫位排布。
// 不负责模型、Session、trace、SSE 或最终文本。
package adapter

import ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"

// GetMinorStar 安14辅星。辅星与主星配合解读命盘，影响主星的吉凶程度和具体应事。
//
// 左辅右弼（按月）、文昌文曲（按时）、天魁天钺（按年干）为吉星；
// 禄存（按年干）、天马（按年支）为财禄动星；
// 擎羊陀罗（禄存前后）、火星铃星（按年支+时支）、地空地劫（按时支）为煞星。
// 每种辅星有各自的算法规则，分布在不同宫位。
func GetMinorStar(yearStem, yearBranch string, timeIndex int, lunarMonth int) [12][]ZiWeiStar {
	return ziweidomain.GetMinorStar(yearStem, yearBranch, timeIndex, lunarMonth)
}
