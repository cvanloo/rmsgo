package mock

import (
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

type UUIDer interface {
	NewRandom() (uuid.UUID, error)
}

type RealUUID struct{}

var _ UUIDer = (*RealUUID)(nil)

func (RealUUID) NewRandom() (uuid.UUID, error) {
	return uuid.NewRandom()
}

type UUIDLogger struct {
	UUIDer
	Log *slog.Logger
}

var _ UUIDer = (*UUIDLogger)(nil)

func (ul UUIDLogger) NewRandom() (uuid.UUID, error) {
	uuid, err := ul.UUIDer.NewRandom()
	ul.Log.Debug("UUIDer", "uuid", uuid, "error", err)
	return uuid, err
}

type UUIDResult struct {
	Result uuid.UUID `json:"uuid"`
	Err    error     `json:"error"`
}

type ReplayUUID struct {
	Queue[UUIDResult]
}

var _ UUIDer = (*ReplayUUID)(nil)

func (r *ReplayUUID) NewRandom() (uuid.UUID, error) {
	ret := r.Dequeue()
	return ret.Result, ret.Err
}

type UUIDMock struct {
	last int
}

var _ UUIDer = (*UUIDMock)(nil)

func (u *UUIDMock) NewRandom() (uuid.UUID, error) {
	u.last++
	lastX := fmt.Sprintf("%x", u.last)
	return uuid.UUID([]byte(lastX)[:16]), nil
}
