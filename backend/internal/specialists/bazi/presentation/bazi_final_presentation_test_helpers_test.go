package presentation

import (
	"strings"
)

func renderFactsOnlyDegradedTemplate(state baziCharterState) string {
	input := buildBaziFinalPresentationInput(state.AnalysisPlan, state, "")
	input.StaticSynthesis.FactsOnly = true
	return RenderFinalReply(input)
}

func renderFullTemplate(state baziCharterState) string {
	input := buildBaziFinalPresentationInput(state.AnalysisPlan, state, "")
	input.StaticSynthesis.FactsOnly = false
	return RenderFinalReply(input)
}

func writeLifetimeDayunGroups(b *strings.Builder, state baziCharterState) {
	input := buildBaziFinalPresentationInput(state.AnalysisPlan, state, "")
	input.StaticSynthesis.FactsOnly = false
	input.AnalysisPlan.NeedLifetimeDayun = true
	b.WriteString(RenderFinalReply(input))
}
