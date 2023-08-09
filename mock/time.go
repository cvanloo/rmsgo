package mock

import "time"

// TimeFunc returns a new function which when called always returns the zero
// value of time.Time.
func TimeFunc() func() (t time.Time) {
	return func() (t time.Time) {
		return
	}
}
