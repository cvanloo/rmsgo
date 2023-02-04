package main

import (
	"log"
	"net/http"

	"framagit.org/attaboy/rmsgo"
)

func main() {
	mux := http.NewServeMux()

	server := rmsgo.Server{}

	const rmsRoot = "rms/"

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
