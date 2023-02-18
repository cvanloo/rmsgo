package rmsgo

import (
	"net/http"
)

func NewServer(webRoot, storageRoot string, auth AuthenticationFunc) Server {
	return Server{
		webRoot:     webRoot,
		storageRoot: storageRoot,
		auth:        auth,
	}
}

type Server struct {
	webRoot     string
	storageRoot string
	auth        AuthenticationFunc
}

type AuthenticationFunc func(r *http.Request) (User, error)

type ErrorHandler func(err error)

func (srv Server) Listen(mux *http.ServeMux, handler ErrorHandler) {
	mux.HandleFunc(srv.webRoot, func(w http.ResponseWriter, r *http.Request) {
		err := srv.Serve(w, r)
		if err != nil {
			handler(err)
		}
	})
}

type User interface {
	Name() string
	Quota() uint
}
