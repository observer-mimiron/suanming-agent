package domain

import "testing"

func TestTextListsNormalizeAndDeduplicate(t *testing.T) {
	if got := NonEmptyStrings([]string{" a ", "", " b", "  "}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("NonEmptyStrings() = %#v", got)
	}
	if got := UniqueStrings([]string{" a ", "a", "b", " b ", ""}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("UniqueStrings() = %#v", got)
	}
}
