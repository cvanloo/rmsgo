package mock

import (
	"fmt"

	"github.com/google/uuid"
)

// UUIDFunc returns a new function that will create deterministic but still
// unique IDs.
func UUIDFunc() func() (uuid.UUID, error) {
	last := 0
	return func() (uuid.UUID, error) {
		last++
		lastX := fmt.Sprintf("%x", last)
		return uuid.UUID([]byte(lastX)[:16]), nil
	}
}
