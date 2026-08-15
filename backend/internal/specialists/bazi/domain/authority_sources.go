// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责按分析阶段提供权威来源层级合同；
// 不读取 SessionState，不调用检索、模型、trace 或输出传输。
package domain

// AuthoritySourceSet 描述某个八字分析阶段可用的来源层级。
// Primary 是主裁判依据，Secondary 是补充交叉验证，Auxiliary 只能辅助印证。
type AuthoritySourceSet struct {
	Primary   []string
	Secondary []string
	Auxiliary []string
}

// StageAuthoritySources 返回指定分析阶段的固定来源层级；未知阶段返回空合同。
func StageAuthoritySources(stage string) AuthoritySourceSet {
	switch stage {
	case "static":
		return AuthoritySourceSet{
			Primary:   []string{"子平真诠", "渊海子平", "穷通宝鉴", "滴天髓"},
			Secondary: []string{"三命通会"},
			Auxiliary: []string{"神煞", "纳音"},
		}
	case "dynamic":
		return AuthoritySourceSet{
			Primary:   []string{"三命通会"},
			Secondary: []string{"滴天髓", "子平真诠"},
			Auxiliary: []string{"神煞", "纳音"},
		}
	default:
		return AuthoritySourceSet{}
	}
}
