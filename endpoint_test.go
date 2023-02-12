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
	const testContents = "If you are reading this you are cute!"
	fsTest := fsMock{}
	useFS = fsTest
	req := httptest.NewRequest(http.MethodPut, "/storage/someuser/test.txt", strings.NewReader(testContents))
	rec := httptest.NewRecorder()
	err := Serve(rec, req)
	if err != nil {
		t.Errorf("PUT document failed: %v\n", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201 CREATED, got: %d\n", rec.Code)
	}
	cf, err := fsTest.Open("/tmp/storage/someuser/test.txt")
	if err != nil {
		t.Fatalf("expected document `test.txt' to exist\n")
	}
	buf := make([]byte, 128)
	size, _ := cf.Read(buf)
	if size != len(testContents) {
		t.Errorf("want document size %d, got: %d\n", len(testContents), size)
	}
	if string(buf[:size]) != testContents {
		t.Errorf("document contains wrong contents: %s\n", string(buf))
	}
}

func TestDeleteDocument(t *testing.T) {
	fsTest := fsMock{
		"/tmp/storage/someuser/test.txt": {
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
	if _, err = fsTest.Stat("/tmp/storage/someuser/test.txt"); err == nil {
		t.Error("want err, got nil")
	}
}
