package rmsgo

import (
	"net/http"

	"framagit.org/attaboy/rmsgo/filetree"
	"framagit.org/attaboy/rmsgo/storage"
)

func NewServer(webRoot, storageRoot string, auth AuthenticationFunc) Server {
	storage.Setup(storageRoot)
	filetree.Setup(storageRoot)
	return Server{
		webRoot: webRoot,
		auth:    auth,
	}
}

type Server struct {
	webRoot string
	auth    AuthenticationFunc
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
