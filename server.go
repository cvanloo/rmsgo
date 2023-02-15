package rmsgo

import "net/http"

type Server struct {
	webRoot     string
	storageRoot string
	auth        AuthenticationFunc
}

func NewServer(webRoot, storageRoot string, auth AuthenticationFunc) Server {
	return Server{
		webRoot: webRoot,
		storageRoot: storageRoot,
		auth: auth,
	}
}

type User interface {
	Name() string
	Quota() uint
}

type AuthenticationFunc func(r *http.Request) (User, error)
