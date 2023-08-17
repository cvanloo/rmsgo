package main

import (
	"log"
	"net/http"

	"github.com/cvanloo/rmsgo"
	//"github.com/cvanloo/rmsgo/mock"
)

const (
	RemoteRoot  = "/storage/"
	StorageRoot = "/tmp/rms/storage/"
)

func main() {
	log.Println("starting listener on :8080")

	//mock.Mock(
	//	mock.WithDirectory("/tmp/rms/storage/"),
	//)

	opts, err := rmsgo.Configure(RemoteRoot, StorageRoot)
	if err != nil {
		log.Fatal(err)
	}
	opts.UseErrorHandler(func(err error) {
		log.Fatalf("remote storage: unhandled error: %v", err)
	})
	opts.AllowAnyReadWrite()

	rmsgo.Register(nil)
	http.ListenAndServe(":8080", nil) // @todo: use TLS
}
