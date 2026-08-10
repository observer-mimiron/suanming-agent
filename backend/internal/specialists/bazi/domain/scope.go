// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责按年龄计算可解释范围；不读取命盘、不选择路由，也不生成命理结论。
package domain

// SubjectContextInput 是年龄授权计算所需的最小输入。
type SubjectContextInput struct {
	BirthYear  int
	TargetYear int
}

// SubjectContext 限制模型可讨论的生活领域，不改变任何命盘事实。
type SubjectContext struct {
	BirthYear             int      `json:"birth_year,omitempty"`
	TargetYear            int      `json:"target_year,omitempty"`
	Age                   int      `json:"age,omitempty"`
	AgeBand               string   `json:"age_band"`
	AllowedOutcomeDomains []string `json:"allowed_outcome_domains"`
}

// BuildSubjectContext 根据出生年和目标年生成稳定的年龄授权范围。
func BuildSubjectContext(input SubjectContextInput) SubjectContext {
	context := SubjectContext{
		BirthYear:             input.BirthYear,
		TargetYear:            input.TargetYear,
		AgeBand:               "unknown",
		AllowedOutcomeDomains: []string{"structure"},
	}
	if input.BirthYear <= 0 || input.TargetYear < input.BirthYear {
		return context
	}
	context.Age = input.TargetYear - input.BirthYear
	switch {
	case context.Age <= 2:
		context.AgeBand = "infant"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 12:
		context.AgeBand = "child"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 17:
		context.AgeBand = "adolescent"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 64:
		context.AgeBand = "adult"
		context.AllowedOutcomeDomains = []string{"structure", "user_requested_authorized_domain"}
	default:
		context.AgeBand = "senior"
		context.AllowedOutcomeDomains = []string{"structure", "user_requested_authorized_domain"}
	}
	return context
}
