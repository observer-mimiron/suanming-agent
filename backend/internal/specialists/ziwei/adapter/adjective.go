// 本文件属于紫微 adapter 层的确定性排盘实现。
// 本文件负责保留既有杂曜组装调用名，并委托 domain 的唯一规则实现。
// 不负责模型、Session、trace、SSE 或最终文本。
package adapter

import ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"

// GetAdjectiveStar 保留既有 adapter 签名，转发到 domain 的杂曜组装规则。
func GetAdjectiveStar(yearly YearlyStarIndex, monthly MonthlyStarIndex, daily DailyStarIndex, timely TimelyStarIndex, hongluan, tianxi int) [12][]ZiWeiStar {
	return ziweidomain.GetAdjectiveStar(yearly, monthly, daily, timely, hongluan, tianxi)
}
