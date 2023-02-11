package rmsgo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"framagit.org/attaboy/rmsgo"
)

var testRequests = [...]*http.Request{
	httptest.NewRequest(http.MethodGet, "/storage/someuser/kittens.png", nil),
	httptest.NewRequest(http.MethodHead, "/storage/someuser/kittens.png", nil),
	httptest.NewRequest(http.MethodPut, "/storage/someuser/kittens.png", nil), // TODO: Put needs to read from somewhere
	httptest.NewRequest(http.MethodDelete, "/storage/someuser/kittens.png", nil),
	httptest.NewRequest(http.MethodGet, "/storage/someuser/documents/", nil),
	httptest.NewRequest(http.MethodHead, "/storage/someuser/documents/", nil),
}

func TestServer(t *testing.T) {
	for _, req := range testRequests {
		rec := httptest.NewRecorder()
		err := rmsgo.Serve(rec, req)
		if err != nil {
			t.Errorf("%s `%s' failed: %v\n", req.Method, req.URL.Path, err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("want 200 OK, got: %d\n", rec.Code)
		}
	}
}
