// Package intent 提供面向用户消息的共享 lexical detector 和 semantic router。
package intent

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
)

// Decision 是 semantic router 对一条用户消息的判定结果。
type Decision string

const (
	// DecisionPositive 表示用户显式提及并请求某方法，应覆盖 LLM 的 PrimaryDomain。
	DecisionPositive Decision = "positive"
	// DecisionNegative 表示用户提及方法但属于否定/对比/疑问，不应覆盖，也不退回 regex。
	DecisionNegative Decision = "negative"
	// DecisionNone 表示无足够信号，不覆盖，不退回 regex。
	DecisionNone Decision = "none"
)

// MatchResult 是 SemanticRouter.Match 的判定结果。
type MatchResult struct {
	Decision Decision
	Method   string  // positive 时为命中方法；negative 时为最佳负向匹配方法
	Score    float64 // positive 时为最佳正向相似度；negative 时为最佳负向相似度
}

// Router 是 semantic router 的接口，便于测试注入 stub。
// *SemanticRouter 默认满足此接口。
type Router interface {
	Match(ctx context.Context, msg string) (MatchResult, error)
}

// cachedRoute 是已 embed 的 utterance 向量，启动时一次性计算存内存。
type cachedRoute struct {
	Positive [][]float64
	Negative [][]float64
}

// SemanticRouter 持 embedder + 已 embed 的 utterance 向量，提供 Match 方法。
// router 是无状态、线程安全的（utterance 向量启动后只读，embedder 本身线程安全）。
type SemanticRouter struct {
	embedder  embedding.Embedder
	routes    map[string]cachedRoute
	threshold float64
}

// maxCosine 返回 msg 与 candidates 中所有向量的最大余弦相似度。
// 空候选集返回 0。维度不匹配的候选向量跳过。
func maxCosine(msg []float64, candidates [][]float64) float64 {
	best := 0.0
	for _, c := range candidates {
		if len(c) != len(msg) {
			continue
		}
		s := cosine(msg, c)
		if s > best {
			best = s
		}
	}
	return best
}

// cosine 计算两个向量的余弦相似度。
func cosine(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Match 对用户消息做语义路由判定。
// 返回 (MatchResult, error)——error 与 Decision 分离，调用方据此区分：
//   - err != nil：embedder 调用失败，调用方退回 regex 兜底（受 Confidence 守卫约束）
//   - Decision == positive：用户显式提及某方法，应覆盖 LLM 的 PrimaryDomain
//   - Decision == negative：提及但属于否定/对比/疑问，不覆盖，**不退回 regex**
//   - Decision == none：无足够信号，不覆盖，不退回 regex
func (r *SemanticRouter) Match(ctx context.Context, msg string) (MatchResult, error) {
	if r == nil || r.embedder == nil || strings.TrimSpace(msg) == "" {
		return MatchResult{Decision: DecisionNone}, nil
	}

	vectors, err := r.embedder.EmbedStrings(ctx, []string{msg})
	if err != nil || len(vectors) == 0 {
		return MatchResult{}, err
	}
	msgVec := vectors[0]

	bestMethod, bestPosScore := "", 0.0
	bestNegMethod, bestNegScore := "", 0.0

	for name, route := range r.routes {
		posScore := maxCosine(msgVec, route.Positive)
		negScore := maxCosine(msgVec, route.Negative)
		if posScore > bestPosScore {
			bestPosScore, bestMethod = posScore, name
		}
		if negScore > bestNegScore {
			bestNegScore, bestNegMethod = negScore, name
		}
	}

	switch {
	case bestPosScore >= r.threshold && bestNegScore < bestPosScore:
		return MatchResult{Decision: DecisionPositive, Method: bestMethod, Score: bestPosScore}, nil
	case bestNegScore >= bestPosScore && bestNegScore >= r.threshold:
		return MatchResult{Decision: DecisionNegative, Method: bestNegMethod, Score: bestNegScore}, nil
	default:
		return MatchResult{Decision: DecisionNone}, nil
	}
}

// NewSemanticRouter 构造 router，启动时一次性 embed 所有 utterance 存内存。
// 失败时返回 error，调用方据此决定是否走 regex 兜底（router=nil）。
func NewSemanticRouter(ctx context.Context, embedder embedding.Embedder, utterances map[string]RouteUtterances, threshold float64) (*SemanticRouter, error) {
	if embedder == nil {
		return nil, errors.New("embedder is nil")
	}

	routes := make(map[string]cachedRoute, len(utterances))
	for name, u := range utterances {
		pos, err := embedder.EmbedStrings(ctx, u.Positive)
		if err != nil {
			return nil, fmt.Errorf("embed positive utterances for %s: %w", name, err)
		}
		neg, err := embedder.EmbedStrings(ctx, u.Negative)
		if err != nil {
			return nil, fmt.Errorf("embed negative utterances for %s: %w", name, err)
		}
		routes[name] = cachedRoute{Positive: pos, Negative: neg}
	}

	return &SemanticRouter{
		embedder:  embedder,
		routes:    routes,
		threshold: threshold,
	}, nil
}
