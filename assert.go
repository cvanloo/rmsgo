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

func mustVal[T any](t T, err error) T {
	if err != nil {
		pc, f, l, _ := runtime.Caller(1)
		panic(fmt.Sprintf("must in %s[%s:%d] failed: %v", runtime.FuncForPC(pc).Name(), f, l, err))
	}
	return t
}

func must(err error) {
	if err != nil {
		pc, f, l, _ := runtime.Caller(1)
		panic(fmt.Sprintf("must in %s[%s:%d] failed: %v", runtime.FuncForPC(pc).Name(), f, l, err))
	}
}
