package rmsgo

import (
	"fmt"

	"github.com/google/uuid"
)

var last int

func createMockUUID() (uuid.UUID, error) {
	last++
	lastX := fmt.Sprintf("%x", last)
	return uuid.UUID([]byte(lastX)[:16]), nil
}
