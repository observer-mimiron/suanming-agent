package intent

import (
	"math"
	"testing"
)

func TestMaxCosine_ReturnsHighestSimilarity(t *testing.T) {
	msg := []float64{1, 0}
	candidates := [][]float64{
		{1, 0},  // 相同方向，cos=1.0
		{0, 1},  // 正交，cos=0
		{-1, 0}, // 反向，cos=-1
	}
	got := maxCosine(msg, candidates)
	if math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("maxCosine = %v, want 1.0", got)
	}
}

func TestMaxCosine_EmptyCandidates(t *testing.T) {
	got := maxCosine([]float64{1, 0}, nil)
	if got != 0 {
		t.Fatalf("maxCosine(nil) = %v, want 0", got)
	}
}