// This test file belongs to the Zi Wei domain layer.
// It protects the pure star placement contract without constructing lunar calendar or tool adapters.
package domain

import "testing"

func TestPureStarCore(t *testing.T) {
	if got := FixIndex(-1, 12); got != 11 {
		t.Fatalf("FixIndex(-1, 12) = %d, want 11", got)
	}
	if got := TimeToIndex(23); got != 12 {
		t.Fatalf("TimeToIndex(23) = %d, want 12", got)
	}
	if got := GetMutagen(0, "廉贞"); got != "化禄" {
		t.Fatalf("GetMutagen(0, 廉贞) = %q, want 化禄", got)
	}

	stars := GetMajorStar(0, 0, 0)[0]
	if len(stars) != 2 || stars[0].Name != "紫微" || stars[1].Name != "天府" {
		t.Fatalf("GetMajorStar(0, 0, 0)[0] = %#v, want 紫微/天府", stars)
	}
}
