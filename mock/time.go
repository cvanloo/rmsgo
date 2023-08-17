package mock

import (
	"time"

	"golang.org/x/exp/slog"
)

type Timer interface {
	Now() time.Time
}

type RealTime struct{}

var _ Timer = (*RealTime)(nil)

func (RealTime) Now() time.Time {
	return time.Now()
}

type TimeLogger struct {
	Timer
	Log *slog.Logger
}

var _ Timer = (*TimeLogger)(nil)

func (tl TimeLogger) Now() time.Time {
	now := tl.Timer.Now()
	tl.Log.Debug("Timer", "result", now)
	return now
}

type ReplayTime struct {
	Queue[time.Time]
}

var _ Timer = (*ReplayTime)(nil)

func (rt *ReplayTime) Now() time.Time {
	return rt.Dequeue()
}

type TimeMock struct {
	last time.Time
}

var _ Timer = (*TimeMock)(nil)

func (t *TimeMock) Now() time.Time {
	//t.last = t.last.Add(time.Minute * 5)
	return t.last
}
