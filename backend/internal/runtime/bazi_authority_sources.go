package runtime

// authoritySourceSet 描述某个八字分析阶段可用的来源层级。
// Primary 是主裁判依据，Secondary 是补充交叉验证，Auxiliary 只能辅助印证。
type authoritySourceSet struct {
	Primary   []string
	Secondary []string
	Auxiliary []string
}

func stageAuthoritySources(stage string) authoritySourceSet {
	switch stage {
	case "static":
		return authoritySourceSet{
			Primary:   []string{"子平真诠", "渊海子平", "穷通宝鉴", "滴天髓"},
			Secondary: []string{"三命通会"},
			Auxiliary: []string{"神煞", "纳音"},
		}
	case "dynamic":
		return authoritySourceSet{
			Primary:   []string{"三命通会"},
			Secondary: []string{"滴天髓", "子平真诠"},
			Auxiliary: []string{"神煞", "纳音"},
		}
	default:
		return authoritySourceSet{}
	}
}
