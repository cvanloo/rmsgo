package rmsgo

import (
	"context"
	"math"
	"net/http"
	"strings"
)

type (
	User interface {
		Permission(name string) Level
		Quota() int64
	}

	anyUser struct{}

	Level string
	key   int
)

const userKey key = iota

var (
	LevelRead      Level = ":r"
	LevelReadWrite Level = ":rw"
)

func (anyUser) Permission(name string) Level {
	return LevelReadWrite
}

func (anyUser) Quota() int64 {
	return math.MaxInt64
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

func handleAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAuthorized := false

		rname := strings.TrimPrefix(r.URL.Path, rroot)
		isPublic := strings.HasPrefix(rname, "/public/")
		if isPublic {
			rname = strings.TrimPrefix(rname, "/public/")
		}

		bearer := r.Header.Get("Authorization")
		bearer = strings.TrimPrefix(bearer, "Bearer ")
		user, isAuthenticated := authenticate(r, bearer)

		isRequestRead := r.Method == http.MethodGet || r.Method == http.MethodHead

		if isAuthenticated {
			nc := context.WithValue(r.Context(), userKey, user)
			r = r.WithContext(nc)

			perm := user.Permission(rname)
			switch perm {
			case LevelRead:
				isAuthorized = isRequestRead
			case LevelReadWrite:
				isAuthorized = true
			}
		}

		if !isAuthorized {
			isDocument := rname[len(rname)-1] != '/'
			if isDocument && isPublic && isRequestRead {
				isAuthorized = true
			}
		}

		if isAuthorized {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
}
