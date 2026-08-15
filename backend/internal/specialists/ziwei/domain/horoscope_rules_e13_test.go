// 本文件属于紫微 domain 层测试。
// 本文件锁定大限领域值对象，不验证 JSON 投影或 lunar-go 适配。
package domain

import "testing"

func TestDecadalIntervals(t *testing.T) {
	got := GetDecadalIntervals("男", 10, Fire6, 6, 6)
	if got[10].StartAge != 6 || got[10].EndAge != 15 {
		t.Fatalf("first decadal = %+v, want ages 6-15", got[10])
	}
	if got[10].HeavenlyStem != "戊" || got[10].EarthlyBranch != "子" {
		t.Fatalf("first decadal = %+v, want 戊子", got[10])
	}
}
