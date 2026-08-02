package intent

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
)

// mockEmbedder 是 embedding.Embedder 的测试替身。
type mockEmbedder struct {
	vectors  map[string][][]float64
	errByMsg map[string]error
}

func (m *mockEmbedder) EmbedStrings(_ context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if err, ok := m.errByMsg[texts[0]]; ok {
		return nil, err
	}
	if v, ok := m.vectors[texts[0]]; ok {
		return v, nil
	}
	return [][]float64{{0, 0}}, nil
}

func newMockEmbedder(vectors map[string][][]float64, errByMsg map[string]error) *mockEmbedder {
	return &mockEmbedder{vectors: vectors, errByMsg: errByMsg}
}

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

func TestMatch_PositiveHit(t *testing.T) {
	embedder := newMockEmbedder(map[string][][]float64{
		"排个紫微盘": {{1, 0}},
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {
				Positive: [][]float64{{1, 0}},
				Negative: [][]float64{{-1, 0}},
			},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "排个紫微盘")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionPositive {
		t.Fatalf("Decision = %q, want positive", got.Decision)
	}
	if got.Method != "ziwei" {
		t.Fatalf("Method = %q, want ziwei", got.Method)
	}
	if math.Abs(got.Score-1.0) > 1e-9 {
		t.Fatalf("Score = %v, want 1.0", got.Score)
	}
}

func TestMatch_NegativePriority(t *testing.T) {
	embedder := newMockEmbedder(map[string][][]float64{
		"我不看紫微": {{-1, 0}},
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {
				Positive: [][]float64{{1, 0}},
				Negative: [][]float64{{-1, 0}},
			},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "我不看紫微")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNegative {
		t.Fatalf("Decision = %q, want negative", got.Decision)
	}
}

func TestMatch_BelowThreshold(t *testing.T) {
	embedder := newMockEmbedder(map[string][][]float64{
		"今天天气": {{0, 0.1}},
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {Positive: [][]float64{{1, 0}}, Negative: [][]float64{{-1, 0}}},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "今天天气")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNone {
		t.Fatalf("Decision = %q, want none", got.Decision)
	}
}

func TestMatch_EmptyMsg(t *testing.T) {
	r := &SemanticRouter{
		embedder:  newMockEmbedder(nil, nil),
		routes:    map[string]cachedRoute{"ziwei": {Positive: [][]float64{{1, 0}}}},
		threshold: 0.75,
	}
	got, err := r.Match(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNone {
		t.Fatalf("Decision = %q, want none for empty msg", got.Decision)
	}
}

func TestMatch_EmbedderError(t *testing.T) {
	embedder := newMockEmbedder(nil, map[string]error{
		"排个紫微盘": errors.New("network timeout"),
	})

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {Positive: [][]float64{{1, 0}}},
		},
		threshold: 0.75,
	}

	_, err := r.Match(context.Background(), "排个紫微盘")
	if err == nil {
		t.Fatal("expected err, got nil")
	}
}

func TestNewSemanticRouter_EmbedsAllUtterancesAtStartup(t *testing.T) {
	// mockEmbedder：为每个 utterance 字符串预设 {1, 0} 向量
	embedder := newMockEmbedder(map[string][][]float64{
		"a": {{1, 0}},
		"b": {{1, 0}},
		"c": {{1, 0}},
	}, nil)

	utterances := map[string]RouteUtterances{
		"ziwei": {
			Positive: []string{"a", "b"},
			Negative: []string{"c"},
		},
	}

	r, err := NewSemanticRouter(context.Background(), embedder, utterances, 0.75)
	if err != nil {
		t.Fatalf("NewSemanticRouter err: %v", err)
	}
	if r.routes["ziwei"].Positive == nil {
		t.Fatalf("positive vectors not cached at startup")
	}
	if r.routes["ziwei"].Negative == nil {
		t.Fatalf("negative vectors not cached at startup")
	}
}
