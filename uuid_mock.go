package rmsgo

import (
	"fmt"

	"github.com/google/uuid"
)

func CreateMockUUIDFunc() func() (uuid.UUID, error) {
	last := 0
	return func() (uuid.UUID, error) {
		last++
		lastX := fmt.Sprintf("%x", last)
		return uuid.UUID([]byte(lastX)[:16]), nil
	}
}
