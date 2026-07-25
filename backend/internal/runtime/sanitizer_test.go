package runtime

import "testing"

func TestSanitizeFinalOutput_DeepSeekDisclaimer(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain disclaimer with fullwidth period",
			input: "命盘分析结束。\n\n**以上内容由 DeepSeek 生成，仅供参考。**",
			want:  "命盘分析结束。",
		},
		{
			name:  "plain disclaimer no bold",
			input: "命盘分析结束。\n\n以上内容由 DeepSeek 生成，仅供参考。",
			want:  "命盘分析结束。",
		},
		{
			name:  "disclaimer with ASCII period",
			input: "命盘分析结束。\n以上内容由 DeepSeek 生成, 仅供参考.",
			want:  "命盘分析结束。",
		},
		{
			name:  "disclaimer after horizontal rule",
			input: "命盘分析结束。\n\n---\n\n**以上内容由 DeepSeek 生成，仅供参考。**",
			want:  "命盘分析结束。\n\n---",
		},
		{
			name:  "multiple trailing disclaimers",
			input: "命盘分析结束。\n以上内容由 DeepSeek 生成，仅供参考。\n以上内容由 DeepSeek 生成，仅供参考。",
			want:  "命盘分析结束。",
		},
		{
			name:  "no disclaimer leaves text intact",
			input: "命盘分析结束。以上仅供参考，不做投资建议。",
			want:  "命盘分析结束。以上仅供参考，不做投资建议。",
		},
		{
			name:  "mid-text 仅供参考 not stripped",
			input: "财运方位仅供参考，主方位为东南。\n\n以上内容由 DeepSeek 生成，仅供参考。",
			want:  "财运方位仅供参考，主方位为东南。",
		},
		{
			name:  "empty string stays empty",
			input: "",
			want:  "",
		},
		{
			name:  "only disclaimer collapses to empty",
			input: "**以上内容由 DeepSeek 生成，仅供参考。**",
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeFinalOutput(c.input)
			if got != c.want {
				t.Errorf("sanitizeFinalOutput(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}
