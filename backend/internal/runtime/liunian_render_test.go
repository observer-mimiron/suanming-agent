// This test file belongs to the manager-owned runtime layer.
// It verifies LiuNian rendering behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"strings"
	"testing"
)

func TestBuildLiuNianBlock_FullOutput(t *testing.T) {
	t.Run("全部字段完整", func(t *testing.T) {
		ln := map[string]interface{}{
			"liunian_year":     2026,
			"liunian_ganzhi":   "丙午",
			"liunian_stem":     "丙",
			"liunian_branch":   "午",
			"liunian_shi_shen": "正印",
			"current_dayun": map[string]interface{}{
				"startAge": 53,
				"endAge":   62,
				"ganZhi":   "甲戌",
			},
			"liunian_chonghe": []map[string]string{
				{"type": "三合", "description": "流年参与寅午戌合火局"},
				{"type": "六冲", "description": "流年子午冲大运子"},
			},
		}

		var sb strings.Builder
		buildLiuNianBlock(&sb, ln)
		output := sb.String()

		// 验证所有关键字段都出现在渲染结果中
		checks := []string{
			"**流年应期**",
			"2026年",
			"丙午",
			"流年天干十神=正印",
			"当前大运=",
			"53-62",
			"甲戌",
			"流年冲合刑害=",
			"[三合]",
			"[六冲]",
			"流年参与寅午戌合火局",
			"流年子午冲大运子",
		}
		for _, c := range checks {
			if !strings.Contains(output, c) {
				t.Errorf("渲染输出缺少 %q\n完整输出:\n%s", c, output)
			}
		}
		t.Logf("渲染输出:\n%s", output)
	})

	t.Run("无 current_dayun（空 map）", func(t *testing.T) {
		ln := map[string]interface{}{
			"liunian_year":     2026,
			"liunian_ganzhi":   "丙午",
			"liunian_stem":     "丙",
			"liunian_branch":   "午",
			"liunian_shi_shen": "正印",
			"current_dayun":    map[string]interface{}{},
		}

		var sb strings.Builder
		buildLiuNianBlock(&sb, ln)
		output := sb.String()

		if strings.Contains(output, "当前大运") {
			t.Errorf("current_dayun 为空 map 时不应渲染当前大运，但输出了:\n%s", output)
		}
	})

	t.Run("无 liunian_chonghe（空切片）", func(t *testing.T) {
		ln := map[string]interface{}{
			"liunian_year":     2026,
			"liunian_ganzhi":   "丙午",
			"liunian_stem":     "丙",
			"liunian_branch":   "午",
			"liunian_shi_shen": "正印",
			"current_dayun": map[string]interface{}{
				"startAge": 53,
				"endAge":   62,
				"ganZhi":   "甲戌",
			},
			"liunian_chonghe": []map[string]string{},
		}

		var sb strings.Builder
		buildLiuNianBlock(&sb, ln)
		output := sb.String()

		if strings.Contains(output, "冲合刑害") {
			t.Errorf("空 liunian_chonghe 不应渲染流年冲合刑害，但输出了:\n%s", output)
		}
	})

	t.Run("仅最简字段", func(t *testing.T) {
		// 构造极端简略的数据：只有 liunian_year + liunian_ganzhi
		ln := map[string]interface{}{
			"liunian_year":   2026,
			"liunian_ganzhi": "丙午",
		}

		var sb strings.Builder
		buildLiuNianBlock(&sb, ln)
		output := sb.String()

		if !strings.Contains(output, "**流年应期**") {
			t.Errorf("最简数据也应输出标题，got:\n%s", output)
		}
		if !strings.Contains(output, "2026年") {
			t.Errorf("最简数据应包含年份，got:\n%s", output)
		}
	})
}
