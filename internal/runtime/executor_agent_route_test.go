package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
)

// testConfigs returns a minimal set of specialist configs for route-bound tests.
func testConfigs() []specialistsConfig {
	return []specialistsConfig{
		{
			Domain:      "bazi",
			Name:        "bazi_specialist",
			Description: "八字命理专家",
			ToolNames:   []string{"bazi_calc", "yongshen", "dayun_analyzer", "knowledge_search"},
		},
		{
			Domain:      "qimen",
			Name:        "qimen_specialist",
			Description: "奇门遁甲专家",
			ToolNames:   []string{"qimen_dunjia", "knowledge_search"},
		},
		{
			Domain:      "ziwei",
			Name:        "ziwei_specialist",
			Description: "紫微斗数专家",
			ToolNames:   []string{"ziwei_calc", "knowledge_search"},
		},
	}
}

// TestRouteBound_FortuneFollowupExcludesQimen 验证普通八字追问 (fortune_followup +
// QimenMode=none) 不包含 qimen_specialist。
func TestRouteBound_FortuneFollowupExcludesQimen(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	for _, cfg := range allowed {
		if cfg.Name == "qimen_specialist" {
			t.Fatalf("qimen_specialist should not be allowed for fortune_followup with QimenMode=none")
		}
	}
}

// TestRouteBound_QimenPrimaryWithSupplementIncludesQimen 验证奇门主链 (QimenMode=primary)
// 包含 qimen_specialist。
func TestRouteBound_QimenPrimaryWithSupplementIncludesQimen(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "primary",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	found := false
	for _, cfg := range allowed {
		if cfg.Name == "qimen_specialist" {
			found = true
		}
	}
	if !found {
		t.Fatal("qimen_specialist should be allowed for QimenMode=primary")
	}
}

// TestRouteBound_QimenSupplementIncludesQimen 验证奇门辅助 (QimenMode=supplement)
// 包含 qimen_specialist。
func TestRouteBound_QimenSupplementIncludesQimen(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "cross_domain_consult",
		SecondaryDomains: []string{"qimen"},
		PolicyHints: schemas.PolicyHints{
			QimenMode: "supplement",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	foundQimen := false
	for _, cfg := range allowed {
		if cfg.Name == "qimen_specialist" {
			foundQimen = true
		}
	}
	if !foundQimen {
		t.Fatal("qimen_specialist should be allowed when QimenMode=supplement and secondary domain is qimen")
	}
}

// TestRouteBound_CrossDomainAllowsBoth 验证跨领域路由 (PrimaryDomain=bazi,
// SecondaryDomains=["qimen"], QimenMode=supplement) 包含 bazi_specialist 和
// qimen_specialist，不包含 ziwei_specialist。
func TestRouteBound_CrossDomainAllowsBoth(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		SecondaryDomains: []string{"qimen"},
		TaskIntent:       "cross_domain_consult",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "supplement",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())

	var names []string
	for _, cfg := range allowed {
		names = append(names, cfg.Name)
	}

	has := func(name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}

	if !has("bazi_specialist") {
		t.Fatal("bazi_specialist should be allowed")
	}
	if !has("qimen_specialist") {
		t.Fatal("qimen_specialist should be allowed")
	}
	if has("ziwei_specialist") {
		t.Fatal("ziwei_specialist should not be allowed for cross-domain bazi+qimen")
	}
}

// TestRouteBound_ZiweiPrimary 验证紫微主域包含 ziwei_specialist。
func TestRouteBound_ZiweiPrimary(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "ziwei",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	found := false
	for _, cfg := range allowed {
		if cfg.Name == "ziwei_specialist" {
			found = true
		}
	}
	if !found {
		t.Fatal("ziwei_specialist should be allowed for ziwei primary domain")
	}
}

// TestRouteBound_UnknownPrimaryFallsBackToBazi 验证未注册的主域降级到 bazi。
func TestRouteBound_UnknownPrimaryFallsBackToBazi(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "unknown",
		TaskIntent:    "collect_profile",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	found := false
	for _, cfg := range allowed {
		if cfg.Name == "bazi_specialist" {
			found = true
		}
	}
	if !found {
		t.Fatal("bazi_specialist should be allowed as fallback for unknown primary domain")
	}
}

// TestRouteBound_CrossDomainBaziZiweiAllowsBoth 验证跨领域路由 (PrimaryDomain=bazi,
// SecondaryDomains=["ziwei"]) 同时包含 bazi_specialist 和 ziwei_specialist。
func TestRouteBound_CrossDomainBaziZiweiAllowsBoth(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"ziwei"},
		TaskIntent:         "cross_domain_consult",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())

	var names []string
	for _, cfg := range allowed {
		names = append(names, cfg.Name)
	}

	has := func(name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}

	if !has("bazi_specialist") {
		t.Fatal("bazi_specialist should be allowed")
	}
	if !has("ziwei_specialist") {
		t.Fatal("ziwei_specialist should be allowed for cross-domain bazi+ziwei")
	}
}

// TestRouteBound_PureBaziFortuneFollowupDoesNotLeakZiwei 验证普通八字追问
// (fortune_followup, primary=bazi, 无 ziwei secondary) 不泄露 ziwei_specialist。
func TestRouteBound_PureBaziFortuneFollowupDoesNotLeakZiwei(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	allowed := allowedSpecialists(route, testConfigs())
	for _, cfg := range allowed {
		if cfg.Name == "ziwei_specialist" {
			t.Fatal("ziwei_specialist should not be allowed for pure bazi fortune_followup")
		}
	}
}
