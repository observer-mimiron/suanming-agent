package runtime

import "time"

// shanghaiLoc 是 Asia/Shanghai 的固定时区 (UTC+8)，用于奇门遁甲排盘的时间计算。
var shanghaiLoc = time.FixedZone("Asia/Shanghai", 8*60*60)

// resolveQimenTime 返回 Asia/Shanghai 时区的当前时间，用于奇门遁甲排盘。
func resolveQimenTime(now time.Time) time.Time {
	return now.In(shanghaiLoc)
}
