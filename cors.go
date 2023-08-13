package rmsgo

import "net/http"

func allowAll(w http.ResponseWriter) {
	// @todo: make configurable
	w.Header().Set("Access-Control-Allow-Origin", "*")
}
