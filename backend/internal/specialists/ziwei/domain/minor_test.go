package domain

import "testing"

func TestMinorStars(t *testing.T) {
	stars := GetMinorStar("丙", "午", 4, 5)
	var count int
	mutagens := map[string]bool{}
	for _, palace := range stars {
		count += len(palace)
		for _, star := range palace {
			if star.Mutagen != "" {
				mutagens[star.Mutagen] = true
			}
		}
	}
	if count != 14 {
		t.Fatalf("minor star count = %d, want 14", count)
	}
	if len(mutagens) != 1 || !mutagens["化科"] {
		t.Fatalf("丙年辅星四化 = %#v, want only 化科", mutagens)
	}
}
