package main

import (
	"log"
	"net/http"

	"framagit.org/attaboy/rmsgo"
)

type mockUser struct{
	name string
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
		// `/www/somesite.com/public/storage/user/kitten.png`
		StorageRoot = "/www/somesite.com/public/storage/"
	)

	srv := rmsgo.NewServer(WebRoot, StorageRoot, func(r *http.Request) (User, error) {
		authHeader := r.Header.Get("Authorization")
		// TODO: parse bearer token, validate with db, ...
		if authHeader == "Bearer testikus123" {
			return &mockUser{
				name: "testikus",
				quota: 1024*1024*64,
			}
		}
		return nil, ErrUnauthorized
	})

	mux.HandleFunc(WebRoot, func(w http.ResponseWriter, r *http.Request) {
		err := srv.Serve(w, r)
		if err != nil {
			// TODO: More intelligent logging / error handling
			//   slog maybe?
			log.Printf("%v\n", err)
		}
	})

	// TODO: Use ListenAndServeTLS()
	http.ListenAndServe(":3000", mux)
}
