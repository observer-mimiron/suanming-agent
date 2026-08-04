// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi axis verdict handling and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import "testing"

func TestValidateStaticStage_RequiresAxisVerdictFields_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:           "弃印就财",
			ClaimStrength:      "倾向成立",
			SupportLevel:       "有气",
			LimitationLevel:    "明显",
			WordingCap:         "中性",
			PatternBasis:       "先看月令印星成势，再看财星透干破印。",
			PatternOutcome:     "方向成立，但力度受限。",
			CounterEvidence:    "财星偏湿，仍属受限，不算强救。",
			AxisConsistency:    "当前仍以弃印就财为主轴，而非改取别格。",
			TiaohouAnchor:      "甲木生子月，先按寒冬木冷的月令场景审调候。",
			PatternAndQingZhuo: "浊中有清。",
			TierJudgment:       "中等",
			TierBasis:          "主轴可立，但调候不足压低层次。",
			ReasoningSummary:   "静态主轴可以成立，但限制明显。",
			ReasoningSteps:     []string{"先看月令子水当令，印星成势。"},
		},
	}

	if err := validateStaticStage(state); err == nil {
		t.Fatalf("expected missing axis verdict fields to fail static validation")
	}
}

func TestValidateStaticStage_AllowsNeutralAxisWithoutConflictReasons_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticAxisSynthesisForTest(),
	}

	if err := validateStaticStage(state); err != nil {
		t.Fatalf("expected neutral axis to pass static validation without conflict reasons, got %v", err)
	}
	if err := validateStaticAxisVerdictConsistency(state.StaticSynthesis); err != nil {
		t.Fatalf("expected neutral axis verdict to pass without conflict reasons, got %v", err)
	}
}

func TestValidateStaticStage_RequiresConflictReasonsOnlyForLimitedOrConflictAxis_NewContract(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*baziStaticSynthesis)
	}{
		{
			name: "tiaohou conflict",
			mutate: func(s *baziStaticSynthesis) {
				s.EffectOnTiaohou = "冲突"
			},
		},
		{
			name: "core disease amplification",
			mutate: func(s *baziStaticSynthesis) {
				s.EffectOnCoreDisease = "放大"
			},
		},
		{
			name: "ji-shen amplification",
			mutate: func(s *baziStaticSynthesis) {
				s.EffectOnJiShenDirection = "放大"
			},
		},
		{
			name: "structure signal ceiling",
			mutate: func(s *baziStaticSynthesis) {
				s.AxisLevel = "结构可见"
				s.AxisCeiling = "结构信号"
			},
		},
		{
			name: "restricted route ceiling",
			mutate: func(s *baziStaticSynthesis) {
				s.AxisCeiling = "受限路线"
			},
		},
		{
			name: "limited consistency flag",
			mutate: func(s *baziStaticSynthesis) {
				s.ConsistencyFlags = []string{"方向成立但力度受限"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synthesis := validStaticAxisSynthesisForTest()
			tt.mutate(&synthesis)
			state := baziCharterState{StaticSynthesis: synthesis}

			if err := validateStaticStage(state); err == nil {
				t.Fatalf("expected missing conflict reasons to fail for %s", tt.name)
			}
			if err := validateStaticAxisVerdictConsistency(synthesis); err == nil {
				t.Fatalf("expected axis verdict consistency to require conflict reasons for %s", tt.name)
			}
		})
	}
}

func validStaticAxisSynthesisForTest() baziStaticSynthesis {
	return baziStaticSynthesis{
		MainAxis:                "财星疏印，方向成立",
		ClaimStrength:           "倾向成立",
		SupportLevel:            "有气",
		LimitationLevel:         "明显",
		WordingCap:              "中性",
		AxisLevel:               "方向成立",
		EffectOnTiaohou:         "中性",
		EffectOnCoreDisease:     "缓解",
		EffectOnJiShenDirection: "缓解",
		AxisCeiling:             "可作主轴",
		PatternBasis:            "先看月令印星当令，再看财星透出疏印。",
		PatternOutcome:          "财星能疏印，方向成立。",
		CounterEvidence:         "财星力道有限，仍保留层次边界。",
		AxisConsistency:         "当前仍以财星疏印为主轴，不改判为别格。",
		TiaohouAnchor:           "甲木生冬月，先按寒木待火的月令场景审调候。",
		PatternAndQingZhuo:      "清中带浊。",
		TierJudgment:            "中等偏上",
		TierBasis:               "主轴有路，但药力未满，不宜拔高。",
		ReasoningSummary:        "静态主轴方向成立，同时保留调候和药力限制。",
		ReasoningSteps:          []string{"先看月令，再看透干与根气。"},
	}
}

func TestValidateCharterConsistency_RejectsRoutePromotedBeyondConflictCeiling_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "杀印结构可见",
			ClaimStrength:           "倾向成立",
			SupportLevel:            "有气",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "主轴成立",
			EffectOnTiaohou:         "冲突",
			EffectOnCoreDisease:     "放大",
			EffectOnJiShenDirection: "放大",
			AxisCeiling:             "可作主轴",
			ConflictReasons:         []string{"该路线继续放大寒湿与忌神方向"},
			PatternBasis:            "申子之间有承接之意。",
			PatternOutcome:          "方向可立，但同时继续放大原局病点。",
			CounterEvidence:         "调候为第一硬约束，这条路线不宜拔高。",
			AxisConsistency:         "当前只能留作结构信号，不再拔高为贵格主轴。",
			TiaohouConstraint:       "寒木待火，调候为第一硬约束。",
			TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
			StrengthBalance:         "印水偏旺，喜火土，不宜再加重水气。",
			PatternAndQingZhuo:      "结构可见，但清浊不足。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等",
			TierBasis:               "结构有其路数，但病药方向不合，层次不宜拔高。",
			ReasoningSummary:        "当前这条结构有承接之意，但同时继续放大寒湿与忌神方向，因此只能作受限结构。",
			ReasoningSteps:          []string{"先看结构有承接之意，再看其方向同时继续放大病点。"},
		},
	}

	if err := validateCharterConsistency(state); err == nil {
		t.Fatalf("expected route promoted beyond conflict ceiling to fail consistency validation")
	}
}

func TestValidateCharterConsistency_AcceptsRestrictedRouteUnderConflictCeiling_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "杀印结构可见，但只可作受限结构信号",
			ClaimStrength:           "倾向成立",
			SupportLevel:            "有气",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "方向成立",
			EffectOnTiaohou:         "冲突",
			EffectOnCoreDisease:     "放大",
			EffectOnJiShenDirection: "放大",
			AxisCeiling:             "受限路线",
			ConflictReasons:         []string{"结构继续放大原局病点"},
			PatternBasis:            "申子之间有承接之意。",
			PatternOutcome:          "方向成立，但力度受限。",
			CounterEvidence:         "这条结构只能作受限说明，不宜拔高。",
			AxisConsistency:         "当前只能留作受限结构信号，而不按贵格主轴拔高。",
			TiaohouConstraint:       "寒木待火，调候为第一硬约束。",
			TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
			StrengthBalance:         "印水偏旺，喜火土，不宜再加重水气。",
			PatternAndQingZhuo:      "结构可见，但清浊不足。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等",
			TierBasis:               "方向可立，但力度受限，层次不宜拔高。",
			ReasoningSummary:        "当前这条结构有承接之意，但同时继续放大寒湿与忌神方向，因此只能作受限结构。",
			ReasoningSteps:          []string{"先看结构有承接之意，再看其方向同时继续放大病点。"},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("expected restricted route to pass, got %v", err)
	}
}

func TestValidateCharterConsistency_RejectsDynamicEscalationBeyondStaticCeiling_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "杀印结构可见，但只可作受限结构信号",
			ClaimStrength:           "倾向成立",
			SupportLevel:            "有气",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "方向成立",
			EffectOnTiaohou:         "冲突",
			EffectOnCoreDisease:     "放大",
			EffectOnJiShenDirection: "放大",
			AxisCeiling:             "受限路线",
			ConflictReasons:         []string{"动态不得越过受限路线的天花板"},
			PatternBasis:            "申子之间有承接之意。",
			PatternOutcome:          "方向成立，但力度受限。",
			CounterEvidence:         "这条结构只能作受限说明，不宜拔高。",
			AxisConsistency:         "当前只能留作受限结构信号，而不按贵格主轴拔高。",
			TiaohouConstraint:       "寒木待火，调候为第一硬约束。",
			TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
			StrengthBalance:         "印水偏旺，喜火土，不宜再加重水气。",
			PatternAndQingZhuo:      "结构可见，但清浊不足。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等",
			TierBasis:               "方向可立，但力度受限，层次不宜拔高。",
			ReasoningSummary:        "当前这条结构有承接之意，但同时继续放大寒湿与忌神方向，因此只能作受限结构。",
			ReasoningSteps:          []string{"先看结构有承接之意，再看其方向同时继续放大病点。"},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前这条路线已全面大成，后运权位可期。",
			ClaimStrength:    "明确成立",
			SupportLevel:     "得力",
			LimitationLevel:  "轻微",
			WordingCap:       "明确",
			DayunPath:        []string{"某步运有承托，但仍有明显限制。"},
			LiunianFocus:     "后续最多也只能是机会伴随变动的窗口。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "动态只能在受限路线内承托。",
			ReasoningSteps:   []string{"先看大运承托，再看它仍未改写静态的限制天花板。"},
		},
	}

	if err := validateCharterConsistency(state); err == nil {
		t.Fatalf("expected dynamic escalation beyond static ceiling to fail")
	}
}

func TestValidateFinalWriterOutput_RejectsWriterEscalationBeyondStaticCeiling_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "杀印结构可见，但只可作结构信号",
			ClaimStrength:           "保守判断",
			SupportLevel:            "出现",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "结构可见",
			EffectOnTiaohou:         "冲突",
			EffectOnCoreDisease:     "放大",
			EffectOnJiShenDirection: "放大",
			AxisCeiling:             "结构信号",
			ConflictReasons:         []string{"只能作结构说明，不可拔高"},
			PatternBasis:            "申子之间有承接之意。",
			PatternOutcome:          "只可作结构信号。",
			CounterEvidence:         "这条结构不能拔高为贵格主轴。",
			AxisConsistency:         "当前只保留结构可见，不再拔高。",
			TiaohouConstraint:       "寒木待火，调候为第一硬约束。",
			TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
			StrengthBalance:         "印水偏旺，喜火土，不宜再加重水气。",
			PatternAndQingZhuo:      "结构可见，但清浊不足。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等",
			TierBasis:               "结构有其意，但不足以拔高。",
			ReasoningSummary:        "当前只能保留结构信号层面的说明。",
			ReasoningSteps:          []string{"先看结构可见，再看其不足以改变全局病药方向。"},
		},
	}
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := "## 强弱视角\n**结论：身强。**\n- 印水偏旺。\n## 调候视角\n**结论：火为第一调候。**\n- 寒木待火。\n## 格局视角\n**结论：这条结构已化杀为权，可成纯主轴贵格。**\n- 杀印相生路线已经完整成局。\n## 大运验证\n**结论：后运大成。**\n- 权位显露。\n## 流年应期\n**结论：机会年。**\n- 变动中有机。\n## 综合判定\n**结论：命格层次中上。**\n- 主轴贵气已成。\n## 命格总结\n- 最大优点：杀印主轴清纯，可化杀为权。"

	if err := validateFinalWriterOutput(plan, state, output); err == nil {
		t.Fatalf("expected writer escalation beyond static ceiling to fail")
	}
}

func TestValidateFinalWriterOutput_RejectsMissingMethodologyAndEvidenceLabels_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "弃印就财，方向成立但力度受限",
			ClaimStrength:           "倾向成立",
			SupportLevel:            "有气",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "方向成立",
			EffectOnTiaohou:         "中性",
			EffectOnCoreDisease:     "缓解",
			EffectOnJiShenDirection: "抑制",
			AxisCeiling:             "受限路线",
			ConflictReasons:         []string{"财星偏湿，仍受调候压制"},
			PatternBasis:            "先看月令印星成势，再看财星透干破印。",
			PatternOutcome:          "方向成立，但力度受限。",
			CounterEvidence:         "财星偏湿，仍难拔高。",
			AxisConsistency:         "当前仍以弃印就财为主轴，而非改取杀印相生。",
			TiaohouConstraint:       "寒木待火，调候为第一硬约束。",
			TiaohouAnchor:           "甲木生子月，先按寒冬木冷的月令场景审调候。",
			StrengthBalance:         "印比偏旺，喜火土泄耗。",
			PatternAndQingZhuo:      "浊中有清。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等偏上，但受调候压制。",
			TierBasis:               "主轴有路，但药力未充足，层次不宜拔高。",
			ReasoningSummary:        "静态主轴可以成立，但必须保留调候与药力不足的限制。",
			ReasoningSteps:          []string{"先看月令子水当令，印星成势。"},
		},
	}
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := "## 强弱视角\n**结论：身旺。**\n- 根气深。\n## 调候视角\n**结论：火为先。**\n- 调候优先。\n## 格局视角\n**结论：命局主轴为弃印就财。**\n1. 先看月令印星成势。\n2. 再看财星透干破印。\n- **为何成立**：财星有疏滞之用。\n- **限制在哪里**：药力不足。\n## 大运验证\n**结论：吉中有阻。**\n- 当前运仍有内耗。\n## 流年应期\n**结论：这一年更像窗口年。**\n- **年性**：动中有机。\n## 综合判定\n**结论：命格层次中等偏上。**\n- **能成在哪里**：主轴有路。\n- **受限在哪里**：调候未足。\n## 命格总结\n- **最大优点**：主轴清晰。"

	if err := validateFinalWriterOutput(plan, state, output); err == nil {
		t.Fatalf("expected output missing methodology/evidence labels to fail")
	}
}

func TestValidateCharterConsistency_AcceptsJiShenDirectionHuJie_NewContract(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:                "弃印就财，方向成立但力度受限",
			ClaimStrength:           "倾向成立",
			SupportLevel:            "有气",
			LimitationLevel:         "明显",
			WordingCap:              "中性",
			ConsistencyFlags:        []string{"方向成立但力度受限"},
			AxisLevel:               "方向成立",
			EffectOnTiaohou:         "中性",
			EffectOnCoreDisease:     "缓解",
			EffectOnJiShenDirection: "缓解",
			AxisCeiling:             "受限路线",
			ConflictReasons:         []string{"财星虽透，但湿气未净，仍需保留限制语"},
			PatternBasis:            "先看月令印星当令，再看财星透干疏印。",
			PatternOutcome:          "方向成立，但药力有限，仍属受限路线。",
			CounterEvidence:         "财星虽透，但仍不足以把原局限制完全化开，难以拔高。",
			AxisConsistency:         "当前仍以弃印就财为主轴，不改判为别格。",
			TiaohouConstraint:       "寒木待火，调候仍是第一硬约束。",
			TiaohouAnchor:           "甲木生子月，先按寒冬木冷的月令场景审调候。",
			StrengthBalance:         "印比偏旺，喜火土泄耗，不宜再增水木。",
			PatternAndQingZhuo:      "浊中有清。",
			QiShiOrCongHua:          "不从化，仍按正格处理。",
			TierJudgment:            "中等偏上",
			TierBasis:               "主轴有路，但限制仍在，层次不宜拔高。",
			ReasoningSummary:        "静态主轴可以成立，且对忌神方向已有缓解，但还不足以解除全部限制。",
			ReasoningSteps: []string{
				"先看月令印星当令，命局根气仍厚。",
				"再看财星透干疏印，忌神方向有所缓解，但药力尚未充足。",
			},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("expected 缓解 to be accepted for ji-shen direction, got %v", err)
	}
}
