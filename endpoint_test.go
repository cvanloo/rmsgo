package rmsgo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"framagit.org/attaboy/rmsgo"
)

func TestHeadDocument(t *testing.T) {
	req, err := http.NewRequest(http.MethodHead, "/storage/someuser/kittens.png", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v\n", err)
	}

	recorder := httptest.NewRecorder()

	//mux := http.NewServeMux()

	//s := rmsgo.Server{}

	//mux.HandleFunc("/storage/", func(w http.ResponseWriter, r *http.Request) {
	//	err := s.Serve(w, r)
	//	if err != nil {
	//		t.Fatalf("failed to serve request: %v\n", err)
	//	}
	//})
	//testServer := httptest.NewServer(mux)

	//err = s.Serve(recorder, req)
	err = rmsgo.HeadDocument(recorder, req)
	if err != nil {
		t.Fatalf("HEAD document failed: %v\n", err)
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200 OK, got: %d\n", recorder.Code)
	}
}
