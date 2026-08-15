// 本文件属于紫微 domain 层测试。
// 本文件锁定长生十二神和博士十二神纯规则，不构造 lunar-go 或工具 adapter。
package domain

import "testing"

func TestHoroscopeRules(t *testing.T) {
	changsheng := GetChangSheng12(Fire6, "男", "午")
	if changsheng[0] != "长生" || changsheng[11] != "养" {
		t.Fatalf("GetChangSheng12(火六局, 男, 午) = %q/%q, want 长生/养", changsheng[0], changsheng[11])
	}

	boshi := GetBoShi12("男", "甲", "子")
	if boshi[0] != "博士" || boshi[11] != "官府" {
		t.Fatalf("GetBoShi12(男, 甲, 子) = %q/%q, want 博士/官府", boshi[0], boshi[11])
	}
}
