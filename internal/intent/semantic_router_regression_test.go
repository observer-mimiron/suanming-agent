package intent

import (
	"context"
	"os"
	"testing"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/embedding/openai"
)

// TestRegression_RouterAccuracy 用真 DashScope embedder 跑回归集。
// 无 EMBEDDING_API_KEY 时 skip——CI 不跑，本地手动验证。
func TestRegression_RouterAccuracy(t *testing.T) {
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Skip("EMBEDDING_API_KEY not set — skipping regression test")
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-v4"
	}

	embedder, err := einoopenai.NewEmbedder(context.Background(), &einoopenai.EmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewEmbedder err: %v", err)
	}

	router, err := NewSemanticRouter(context.Background(), embedder, Utterances, 0.75)
	if err != nil {
		t.Fatalf("NewSemanticRouter err: %v", err)
	}

	positiveCases := []struct {
		msg    string
		method string
	}{
		{"排个紫微盘", "ziwei"},
		{"我想看紫微斗数", "ziwei"},
		{"用奇门遁甲起一局", "qimen"},
		{"帮我排八字", "bazi"},
		{"看下我的八字命盘", "bazi"},
	}
	for _, c := range positiveCases {
		t.Run("positive_"+c.msg, func(t *testing.T) {
			got, err := router.Match(context.Background(), c.msg)
			if err != nil {
				t.Fatalf("Match err: %v", err)
			}
			if got.Decision != DecisionPositive {
				t.Fatalf("msg=%q Decision=%q, want positive (score=%v)", c.msg, got.Decision, got.Score)
			}
			if got.Method != c.method {
				t.Fatalf("msg=%q Method=%q, want %q", c.msg, got.Method, c.method)
			}
		})
	}

	negativeCases := []string{
		"我不看紫微",
		"紫微和八字哪个准",
		"紫微准吗",
		"什么是紫微",
		"紫微太贵了",
		"我不信奇门",
		"八字是什么",
		"八字准吗",
	}
	for _, msg := range negativeCases {
		t.Run("negative_"+msg, func(t *testing.T) {
			got, err := router.Match(context.Background(), msg)
			if err != nil {
				t.Fatalf("Match err: %v", err)
			}
			if got.Decision == DecisionPositive {
				t.Fatalf("msg=%q should NOT be positive (got positive, method=%q)", msg, got.Method)
			}
		})
	}

	edgeCases := []string{
		"今天天气怎么样",
		"排盘", // 无方法
		"",
	}
	for _, msg := range edgeCases {
		t.Run("edge_"+msg, func(t *testing.T) {
			got, _ := router.Match(context.Background(), msg)
			if got.Decision == DecisionPositive {
				t.Fatalf("msg=%q should NOT be positive", msg)
			}
		})
	}
}