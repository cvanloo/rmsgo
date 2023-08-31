package rmsgo

import (
	"context"
	"net/http"
	"strings"
)

type (
	User interface {
		Permission(name string) Level
		// @todo: Root() string // get user's storage root?
		// @todo: Quota() int64 ?
	}

	// ReadOnlyUser is a User with read access to any folder.
	ReadOnlyUser struct{}

	// ReadWriteUser is a User with read and write access to any folder.
	ReadWriteUser struct{}

	// ReadPublicUser is a User with read permissions only to public folders.
	ReadPublicUser struct{}

	Level string
	key   int
)

const userKey key = iota

var (
	LevelNone      Level = ""
	LevelRead      Level = ":r"
	LevelReadWrite Level = ":rw"
)

var _ User = (*ReadOnlyUser)(nil)
var _ User = (*ReadWriteUser)(nil)
var _ User = (*ReadPublicUser)(nil)

func (ReadOnlyUser) Permission(name string) Level {
	return LevelRead
}

func (ReadWriteUser) Permission(name string) Level {
	return LevelReadWrite
}

func (ReadPublicUser) Permission(name string) Level {
	return LevelNone
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

func handleAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := r.Header.Get("Authorization")
		bearer = strings.TrimPrefix(bearer, "Bearer ")
		user, isAuthenticated := g.authenticate(r, bearer)
		if isAuthenticated {
			nc := context.WithValue(r.Context(), userKey, user)
			r = r.WithContext(nc)
		}

		isAuthorized := isAuthorized(r, user)
		if isAuthorized {
			next.ServeHTTP(w, r)
		} else {
			if isAuthenticated {
				w.WriteHeader(http.StatusForbidden)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	})
}

func isAuthorized(r *http.Request, user User) bool {
	rname, isPublic, isFolder := parsePath(r.URL.Path)

	isRequestRead := r.Method == http.MethodGet || r.Method == http.MethodHead

	if user != nil {
		switch user.Permission(rname) {
		case LevelNone:
			return isPublic && !isFolder && isRequestRead
		case LevelRead:
			return isRequestRead
		case LevelReadWrite:
			return true
		}
	}

	return isPublic && !isFolder && isRequestRead
}

func parsePath(path string) (rname string, isPublic, isFolder bool) {
	rname = strings.TrimPrefix(path, g.rroot)
	isPublic = strings.HasPrefix(rname, "/public/")
	// additional if-check necessary, because path could be named
	// '/publicsomethingelse' in which case the public should not be trimmed
	if isPublic {
		// rname must start with a '/', so don't trim it!
		rname = strings.TrimPrefix(rname, "/public")
	}
	isFolder = rname[len(rname)-1] == '/'
	return
}
