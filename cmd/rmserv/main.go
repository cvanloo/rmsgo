package main

import (
	"log"
	"net/http"

	"framagit.org/attaboy/rmsgo"
	"framagit.org/attaboy/rmsgo/mocks"
)

type mockUser struct {
	name  string
	quota uint
}

func (mu mockUser) Name() string {
	return mu.name
}

func (mu mockUser) Quota() uint {
	return mu.quota
}

func main() {
	mux := http.NewServeMux()

	const (
		// WebRoot specifies the URL under which documents are accessed.
		// For example to retrieve the file `/user/kitten.png` the request
		// would be `GET www.somesite.com/storage/user/kitten.png`
		WebRoot = "/storage/"

		// StorageRoot specifies where on the server documents are stored.
		// The server path for `/user/kitten.png` would be
		// `/var/www/somesite.com/public/storage/user/kitten.png`
		//StorageRoot = "/var/www/somesite.com/public/storage/"
		StorageRoot = "/tmp/storage/"
	)

	srv := rmsgo.NewServer(WebRoot, StorageRoot, func(r *http.Request) (rmsgo.User, error) {
		authHeader := r.Header.Get("Authorization")
		// Parse bearer token, validate with db, ...
		_ = authHeader
		return mocks.TestUser, nil
	})

	srv.Listen(mux, func(err error) {
		// TODO: More intelligent logging / error handling
		//   slog maybe?
		log.Printf("%v\n", err)
	})

	// Alternatively:
	// mux.HandleFunc(WebRoot, func(w http.ResponseWriter, r *http.Request) {
	// 	err := srv.Serve(w, r)
	// 	if err != nil {
	// 		// TODO: More intelligent logging / error handling
	// 		//   slog maybe?
	// 		log.Printf("%v\n", err)
	// 	}
	// })

	// TODO: Use ListenAndServeTLS()
	log.Println("listening on :3000")
	log.Printf("WebRoot: %s\n", WebRoot)
	log.Printf("StorageRoot: %s\n", StorageRoot)
	http.ListenAndServe(":3000", mux)
}
