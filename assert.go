package rmsgo

import (
	"fmt"
	"runtime"
)

func assert(v bool, msg string) {
	if !v {
		pc, f, l, _ := runtime.Caller(1)
		panic(fmt.Sprintf("assertion in %s[%s:%d] failed: %s", runtime.FuncForPC(pc).Name(), f, l, msg))
	}
}

func must[T any](t T, err error) T {
	assert(err == nil, "non-nil error in must")
	return t
}
