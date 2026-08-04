// This file belongs to the intent detection layer.
// It owns intent utterance fixtures and matching for this package.
// It classifies text signals; it must not mutate session state.
package intent

// RouteUtterances 是单个术数方法的正向和负向 utterance 集合。
// 正向：用户显式请求该方法。负向：提及方法但不请求（否定/对比/疑问/价格质疑）。
type RouteUtterances struct {
	Positive []string
	Negative []string
}

// Utterances 是 semantic router 的训练数据。
// 启动时一次性 embed 所有 utterance，存内存常驻。
var Utterances = map[string]RouteUtterances{
	"ziwei": {
		Positive: []string{
			"我想看紫微",
			"排个紫微盘",
			"紫微斗数分析",
			"看下我的星盘",
			"用紫微算命",
			"紫微排盘",
		},
		Negative: []string{
			"我不看紫微",
			"紫微和八字哪个准",
			"紫微准吗",
			"什么是紫微",
			"紫微太贵了",
			"紫微和八字区别",
		},
	},
	"qimen": {
		Positive: []string{
			"用奇门看一下",
			"起个奇门局",
			"遁甲预测",
			"奇门遁甲排盘",
			"帮我起奇门",
		},
		Negative: []string{
			"奇门是什么",
			"我不信奇门",
			"奇门和紫微区别",
			"奇门准吗",
			"奇门太玄乎了",
		},
	},
	"bazi": {
		Positive: []string{
			"排八字",
			"看我的八字",
			"算算命盘",
			"八字分析",
			"帮我排个八字",
		},
		Negative: []string{
			"什么是八字",
			"八字准吗",
			"我不信八字",
			"八字和紫微哪个准",
			"八字太复杂了",
		},
	},
}
