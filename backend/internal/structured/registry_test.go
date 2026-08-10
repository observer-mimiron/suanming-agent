package structured

import (
	"strings"
	"testing"
)

func TestRegisterBundleUsesDraft07AndExactRawPrompt(t *testing.T) {
	bundle := []byte("{\"example\":{\"$schema\":\"http://json-schema.org/draft-07/schema#\",\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}},\"required\":[\"name\"],\"additionalProperties\":false}}")
	if err := RegisterBundle(bundle); err != nil {
		t.Fatal(err)
	}
	prompt, err := PromptContract("example")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "draft-07/schema#") || !strings.Contains(prompt, "additionalProperties") {
		t.Fatalf("prompt omitted Draft-07 contract: %s", prompt)
	}
	if _, err := Hash("example"); err != nil {
		t.Fatal(err)
	}
}
