package orchestrator

import "time"

var shanghaiLoc = time.FixedZone("Asia/Shanghai", 8*60*60)

// resolveQimenTime returns the current time in Asia/Shanghai for qimen charting.
func resolveQimenTime(now time.Time) time.Time {
	return now.In(shanghaiLoc)
}
