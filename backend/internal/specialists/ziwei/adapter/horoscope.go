// 本文件属于紫微 adapter 层的确定性排盘实现。
// 本文件负责长生十二神、博士十二神和大限的确定性计算。
// 不负责模型、Session、trace、SSE 或最终文本。
package adapter

import (
	"github.com/6tail/lunar-go/calendar"
	ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"
)

// GetChangSheng12 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetChangSheng12(fiveElemNum int, gender string, yearBranch string) [12]string {
	return ziweidomain.GetChangSheng12(fiveElemNum, gender, yearBranch)
}

// GetBoShi12 保留未读取的 lunar-go 参数并转发到 domain 唯一规则实现。
func GetBoShi12(solar *calendar.Solar, gender, yearStem, yearBranch string) [12]string {
	_ = solar
	return ziweidomain.GetBoShi12(gender, yearStem, yearBranch)
}
