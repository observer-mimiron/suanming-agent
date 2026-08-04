// This file belongs to the manager-owned runtime layer.
// It owns BaZi value normalization helpers for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

// stringValue returns a string only when the raw value is already string typed.
func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

// intValue converts common decoded JSON number shapes into int for guards.
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
