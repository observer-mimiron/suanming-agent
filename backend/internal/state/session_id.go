// This file belongs to the session state layer.
// It owns session ID validation for this package.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

import (
	"fmt"
	"regexp"
	"strings"
)

var validSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidateSessionID enforces the storage-safe session id contract shared by the
// HTTP boundary and file-backed session persistence.
func ValidateSessionID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("session id is required")
	}
	if id == "." || id == ".." {
		return fmt.Errorf("session id %q is invalid", id)
	}
	if !validSessionIDPattern.MatchString(id) {
		return fmt.Errorf("session id %q is invalid", id)
	}
	return nil
}
