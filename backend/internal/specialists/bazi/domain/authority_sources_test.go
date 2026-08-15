package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStageAuthoritySourcesPreservesStageContracts(t *testing.T) {
	tests := []struct {
		name  string
		stage string
		want  AuthoritySourceSet
	}{
		{
			name:  "static",
			stage: "static",
			want: AuthoritySourceSet{
				Primary:   []string{"子平真诠", "渊海子平", "穷通宝鉴", "滴天髓"},
				Secondary: []string{"三命通会"},
				Auxiliary: []string{"神煞", "纳音"},
			},
		},
		{
			name:  "dynamic",
			stage: "dynamic",
			want: AuthoritySourceSet{
				Primary:   []string{"三命通会"},
				Secondary: []string{"滴天髓", "子平真诠"},
				Auxiliary: []string{"神煞", "纳音"},
			},
		},
		{name: "unknown stage", stage: "unknown", want: AuthoritySourceSet{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := StageAuthoritySources(test.stage)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("StageAuthoritySources(%q) = %#v, want %#v", test.stage, got, test.want)
			}
		})
	}
}

func TestStageAuthoritySourcesKeepsJSONFieldNames(t *testing.T) {
	raw, err := json.Marshal(StageAuthoritySources("static"))
	if err != nil {
		t.Fatalf("marshal authority sources: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode authority sources: %v", err)
	}
	for _, key := range []string{"Primary", "Secondary", "Auxiliary"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("JSON field %q missing from %s", key, raw)
		}
	}
}

func TestStageAuthoritySourcesReturnsIndependentSlices(t *testing.T) {
	first := StageAuthoritySources("static")
	first.Primary[0] = "changed"
	second := StageAuthoritySources("static")
	if second.Primary[0] != "子平真诠" {
		t.Fatalf("source slices must not be shared between calls: %#v", second.Primary)
	}
}
