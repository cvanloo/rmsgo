package rmsgo

import (
	"net/http"
	"strings"
)

var (
	errCorsFail  = ErrForbidden
	allowMethods = []string{"HEAD", "GET", "PUT", "DELETE"}
	allowHeaders = []string{
		"Authorization",
		"Content-Length",
		"Content-Type",
		"Origin",
		"X-Requested-With",
		"If-Match",
		"If-None-Match",
	}
)

func handleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			err := preflight(w, r)
			if err != nil {
				g.unhandled(err)
			}
			// do NOT pass on to next handler
		} else {
			err := cors(w, r)
			if err != nil {
				g.unhandled(err)
			}
			next.ServeHTTP(w, r)
		}
	})
}

func preflight(w http.ResponseWriter, r *http.Request) error {
	path := strings.TrimPrefix(r.URL.Path, g.rroot)
	isFolder := path[len(path)-1] == '/'

	hs := w.Header()

	// always set Vary headers
	hs.Add("Vary", "Origin")
	hs.Add("Vary", "Access-Control-Request-Method")
	hs.Add("Vary", "Access-Control-Request-Headers")

	origin := r.Header.Get("Origin")
	if !(g.allowAllOrigins || g.allowOrigin(r, origin)) {
		return WriteError(w, errCorsFail)
	}

	n, err := Retrieve(path)
	if err != nil { // not found
		return WriteError(w, errCorsFail)
	}
	if n.isFolder != isFolder { // malformed request
		return WriteError(w, errCorsFail)
	}

	reqMethod := strings.ToUpper(r.Header.Get("Access-Control-Request-Method"))
	reqMethodAllowed := false
	if reqMethod == http.MethodOptions {
		reqMethodAllowed = true
	} else {
		for _, m := range allowMethods {
			if m == reqMethod {
				reqMethodAllowed = true
				break
			}
		}
	}
	if !reqMethodAllowed {
		return WriteError(w, errCorsFail)
	}

	// We might get multiple header values, but a single value might actually
	// contain multiple values itself, separated by commas.
	// By first joining all the values together, and then splitting again, we
	// ensure that all values are separate.
	reqHeaders := strings.Split(strings.Join(r.Header.Values("Access-Control-Request-Headers"), ","), ",")
	for _, reqHeader := range reqHeaders {
		reqHeader = http.CanonicalHeaderKey(strings.TrimSpace(reqHeader))
		reqHeaderAllowed := false
		for _, h := range allowHeaders {
			if h == reqHeader {
				reqHeaderAllowed = true
				break
			}
		}
		if !reqHeaderAllowed {
			return WriteError(w, errCorsFail)
		}
	}

	if g.allowAllOrigins {
		hs.Set("Access-Control-Allow-Origin", "*")
	} else {
		hs.Set("Access-Control-Allow-Origin", origin)
	}

	hs.Set("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
	hs.Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func cors(w http.ResponseWriter, r *http.Request) error {
	hs := w.Header()

	// always set Vary header
	hs.Set("Vary", "Origin")

	origin := r.Header.Get("Origin")
	if !(g.allowAllOrigins || g.allowOrigin(r, origin)) {
		return WriteError(w, errCorsFail)
	}

	if g.allowAllOrigins {
		hs.Set("Access-Control-Allow-Origin", "*")
	} else {
		hs.Set("Access-Control-Allow-Origin", origin)
	}
	return nil
}
