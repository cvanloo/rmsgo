package rmsgo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testRequests = [...]*http.Request{
	httptest.NewRequest(http.MethodGet, "/storage/someuser/kittens.png", nil),
	httptest.NewRequest(http.MethodHead, "/storage/someuser/kittens.png", nil),
	httptest.NewRequest(http.MethodGet, "/storage/someuser/documents/", nil),
	httptest.NewRequest(http.MethodHead, "/storage/someuser/documents/", nil),
}

func TestServer(t *testing.T) {
	for _, req := range testRequests {
		rec := httptest.NewRecorder()
		err := Serve(rec, req)
		if err != nil {
			t.Errorf("%s `%s' failed: %v\n", req.Method, req.URL.Path, err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("want 200 OK, got: %d\n", rec.Code)
		}
	}
}

func TestPutDocument(t *testing.T) {
	fsTest := fsMock{
		"test.txt": {
			contents: []byte("if you are reading this you are cute!"),
		},
	}
	useFS = fsTest
	const testContents = "If you are reading this you are cute!"
	req := httptest.NewRequest(http.MethodPut, "/storage/someuser/test.txt", strings.NewReader(testContents))
	rec := httptest.NewRecorder()
	err := Serve(rec, req)
	if err != nil {
		t.Errorf("PUT document failed: %v\n", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201 CREATED, got: %d\n", rec.Code)
	}
}

func TestDeleteDocument(t *testing.T) {
	fsTest := fsMock{
		"test.txt": {
			contents: []byte("if you are reading this you are cute!"),
		},
	}
	useFS = fsTest
	req := httptest.NewRequest(http.MethodDelete, "/storage/someuser/test.txt", nil)
	rec := httptest.NewRecorder()
	err := Serve(rec, req)
	if err != nil {
		t.Errorf("DELETE document failed: %v\n", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 OK, got: %d\n", rec.Code)
	}
}
