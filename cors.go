package rmsgo

import (
	"net/http"
	"strings"
)

func handleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			preflight(w, r)
			w.WriteHeader(http.StatusNoContent) // do NOT pass on to next handler
		} else {
			cors(w, r)
			next.ServeHTTP(w, r)
		}
	})
}

func preflight(w http.ResponseWriter, r *http.Request) {
	allowedHeaders := []string{"Authorization", "Content-Length", "Content-Type", "Origin", "X-Requested-With", "If-Match", "If-None-Match"}

	path := r.URL.Path
	isFolder := false
	if path[len(path)-1] == '/' {
		isFolder = true
	}

	var allowedMethods []string // @fixme: HEAD implied by GET?
	if isFolder {
		allowedMethods = []string{"GET", "PUT", "DELETE"}
	} else {
		allowedMethods = []string{"GET"}
	}

	hs := w.Header()
	origin := r.Header.Get("Origin")

	hs.Add("Vary", "Origin")
	hs.Add("Vary", "Access-Control-Request-Method")
	hs.Add("Vary", "Access-Control-Request-Headers")

	if origin == "" {
		return
	}

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	reqMethod = strings.ToUpper(reqMethod)
	reqMethodAllowed := false
	if reqMethod == http.MethodOptions {
		reqMethodAllowed = true
	} else {
		for _, m := range allowedMethods {
			if m == reqMethod {
				reqMethodAllowed = true
				break
			}
		}
	}
	if !reqMethodAllowed {
		return
	}

	reqHeadersStr := strings.Join(r.Header.Values("Access-Control-Request-Headers"), ",")
	reqHeaders := strings.Split(reqHeadersStr, ",")
	for _, reqHeader := range reqHeaders {
		reqHeader = strings.TrimSpace(reqHeader)
		reqHeaderAllowed := false
		for _, h := range allowedHeaders {
			if h == reqHeader {
				reqHeaderAllowed = true
			}
		}
		if !reqHeaderAllowed {
			return
		}
	}

	if allowAllOrigins {
		hs.Set("Access-Control-Allow-Origin", "*")
	} else {
		hs.Set("Access-Control-Allow-Origin", origin)
	}

	hs.Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
	hs.Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
}

func cors(w http.ResponseWriter, r *http.Request) {
	hs := w.Header()
	origin := r.Header.Get("Origin")

	hs.Set("Vary", "Origin")

	if origin == "" {
		return
	}

	if !(allowAllOrigins || allowOriginFunc(r, origin)) {
		return
	}

	if allowAllOrigins {
		hs.Set("Access-Control-Allow-Origin", "*")
	} else {
		hs.Set("Access-Control-Allow-Origin", origin)
	}
}
