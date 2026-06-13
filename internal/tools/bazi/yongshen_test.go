package bazi

import (
	"context"
	"testing"
)
func TestYongShen(t *testing.T) {
    yt := &YongShenTool{}
	r, err := yt.Execute(context.Background(), map[string]any{
        "year": float64(1974), "month": float64(4), "day": float64(28),
        "hour": float64(16), "gender": "男",
    })
    if err != nil { t.Fatal(err) }
    m := r.(map[string]any)
    t.Logf("日主: %s(%s) 季节: %s 强弱: %s", m["day_master"], m["day_master_wuxing"], m["season"], m["strength"])
    t.Logf("用神: %v  喜神: %v  忌神: %v", m["yong_shen"], m["xi_shen"], m["ji_shen"])
    t.Logf("调候: %s", m["tiao_hou"])
    t.Logf("支撑分: 月令=%v 根=%v 同元=%v 生扶=%v 总分=%v", m["month_score"], m["root_count"], m["same_element"], m["generate_count"], m["total_support"])
}
