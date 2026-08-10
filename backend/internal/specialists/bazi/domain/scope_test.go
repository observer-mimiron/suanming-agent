package domain

import "testing"

func TestBuildSubjectContextKeepsMinorDomainsBounded(t *testing.T) {
	context := BuildSubjectContext(SubjectContextInput{BirthYear: 2020, TargetYear: 2025})
	if context.AgeBand != "child" || context.Age != 5 {
		t.Fatalf("context = %+v", context)
	}
	if len(context.AllowedOutcomeDomains) != 4 || context.AllowedOutcomeDomains[1] != "growth_environment" {
		t.Fatalf("minor domains = %+v", context.AllowedOutcomeDomains)
	}
}

func TestBuildSubjectContextUsesStructureOnlyWhenYearsAreInvalid(t *testing.T) {
	context := BuildSubjectContext(SubjectContextInput{BirthYear: 0, TargetYear: 2025})
	if context.AgeBand != "unknown" || len(context.AllowedOutcomeDomains) != 1 || context.AllowedOutcomeDomains[0] != "structure" {
		t.Fatalf("invalid-year context = %+v", context)
	}
}
