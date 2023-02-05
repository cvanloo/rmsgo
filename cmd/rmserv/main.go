package main

import (
	"log"
	"net/http"

	"framagit.org/attaboy/rmsgo"
)

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

	server := rmsgo.Server{
		WebRoot: WebRoot,
		StorageRoot: ServerRoot
	}

	mux.HandleFunc(rmsRoot, func(w http.ResponseWriter, r *http.Request) {
		err := server.Serve(w, r)
		if err != nil {
			// TODO: More intelligent logging / error handling
			log.Printf("%v\n", err)
		}
	})

	// TODO: Use ListenAndServeTLS()
	http.ListenAndServe(":3000", mux)
}
