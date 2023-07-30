package main

import (
	"log"
	"net/http"

	"github.com/cvanloo/rmsgo.git"
)

const StorageRoot = "/storage/"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(StorageRoot, func(w http.ResponseWriter, r *http.Request) {
		err := rmsgo.Serve(w, r)
		if err != nil {
			log.Printf("rms-server: %s", err)
		}
	})
	http.ListenAndServe(":8080", mux)
}
