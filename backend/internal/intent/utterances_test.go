package intent

import "testing"

func TestUtterances_EachMethodHasMinimum(t *testing.T) {
	for _, method := range []string{"ziwei", "qimen", "bazi"} {
		r, ok := Utterances[method]
		if !ok {
			t.Fatalf("method %q missing from Utterances", method)
		}
		if len(r.Positive) < 5 {
			t.Fatalf("method %q: only %d positive utterances, want >= 5", method, len(r.Positive))
		}
		if len(r.Negative) < 5 {
			t.Fatalf("method %q: only %d negative utterances, want >= 5", method, len(r.Negative))
		}
	}
}
