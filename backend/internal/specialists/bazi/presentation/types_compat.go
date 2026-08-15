// Package presentation renders accepted Bazi results for users.
//
// This file aliases Bazi-owned value objects for renderer compatibility.
// It does not depend on runtime, models, sessions, or event transport.
package presentation

import (
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

type baziAnalysisPlan = bazidomain.AnalysisPlan
type baziCharterState = bazidomain.CharterState
type baziStaticSynthesis = bazidomain.StaticSynthesis
type baziDynamicSynthesis = bazidomain.DynamicSynthesis
type baziLifetimeDayunSynthesis = bazidomain.LifetimeDayunSynthesis
type baziCitation = bazidomain.Citation
type baziDayunJudgment = bazidomain.DayunJudgment
type baziAssertion = bazidomain.Assertion
type baziEvidenceQuality = bazidomain.EvidenceQuality

func isFactsOnlyStaticSynthesis(s baziStaticSynthesis) bool {
	return bazidomain.IsFactsOnlyStaticSynthesis(s)
}

func isFactsOnlyDynamicSynthesis(s baziDynamicSynthesis) bool {
	return bazidomain.IsFactsOnlyDynamicSynthesis(s)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsAnyText(texts []string, needles []string) bool {
	for _, text := range texts {
		for _, needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				return true
			}
		}
	}
	return false
}
