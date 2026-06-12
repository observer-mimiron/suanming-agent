package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Client calls the LLM supervisor and returns structured decisions.
type Client struct {
	flash llm.Chat
}

// NewClient creates a supervisor client backed by the given flash model.
func NewClient(flash llm.Chat) *Client {
	return &Client{flash: flash}
}

// Decide runs the layered supervisor (conversation → domain → task) and returns a structured decision.
func (c *Client) Decide(ctx context.Context, msg string, st *state.SessionState) (schemas.SupervisorDecision, error) {
	prompt, err := buildSupervisorPrompt()
	if err != nil {
		return safeFallback(st), fmt.Errorf("supervisor prompt: %w", err)
	}

	messages := []llm.Message{
		{Role: "user", Content: msg},
	}

	resp, _, err := c.flash.Generate(ctx, prompt, messages)
	if err != nil {
		return safeFallback(st), fmt.Errorf("supervisor call: %w", err)
	}

	decision := parseDecision(resp)
	return decision, nil
}

// parseDecision unmarshals raw JSON into a normalized SupervisorDecision.
// On failure it returns a zero-value decision with safe defaults applied.
func parseDecision(raw string) schemas.SupervisorDecision {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		d := schemas.SupervisorDecision{}
		d.Normalize()
		return d
	}

	// Strip markdown code fences if present.
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var d schemas.SupervisorDecision
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		d = schemas.SupervisorDecision{}
	}
	d.Normalize()
	return d
}

// safeFallback returns a conservative fallback decision when the model is unavailable.
func safeFallback(st *state.SessionState) schemas.SupervisorDecision {
	needsClarification := !st.IsProfileComplete() && !st.HasBaziResult()
	taskIntent := "collect_profile"
	if st.IsProfileComplete() || st.HasBaziResult() {
		taskIntent = "interpret_chart"
		needsClarification = false
	}

	return schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{},
		TaskIntent:         taskIntent,
		NeedsClarification: needsClarification,
		Confidence:         0.5,
		Slots: schemas.DecisionSlots{
			Profile:    map[string]any{},
			Language:   "zh",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsKnowledge:         true,
			CanReuseSessionProfile: st.IsProfileComplete() || st.HasBaziResult(),
			CanReuseCachedResult:   st.HasBaziResult(),
		},
	}
}

// supervisorPromptFiles lists the prompt assets to load and concatenate.
var supervisorPromptFiles = []string{
	"prompts/supervisor/conversation_router.md",
	"prompts/supervisor/domain_router.md",
	"prompts/supervisor/task_router.md",
}

// buildSupervisorPrompt loads and concatenates the supervisor prompt assets.
func buildSupervisorPrompt() (string, error) {
	var parts []string
	for _, path := range supervisorPromptFiles {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}
