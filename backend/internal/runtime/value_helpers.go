// Package runtime contains Manager-owned execution coordination.
//
// This file provides scalar and map helpers shared by runtime plan, prefill, and follow-up code;
// it does not contain Bazi rules, Graph nodes, or specialist behavior.
package runtime

import (
	"fmt"
	"strings"
)

// stringValue returns a string only when the raw value is already textual.
func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

// intValue converts the JSON number forms used by runtime asset payloads.
func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// copyAnyMap returns a shallow copy so runtime projections cannot mutate an asset payload.
func copyAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// containsAnyText reports whether any source text contains a target fragment.
func containsAnyText(texts []string, needles []string) bool {
	for _, text := range texts {
		for _, needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				return true
			}
		}
	}
	return false
}

// anyToString formats a dynamic runtime payload value for follow-up matching.
func anyToString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
