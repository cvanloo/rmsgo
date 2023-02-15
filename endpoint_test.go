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
	// TODO: create better mocks with nested dirs/files, also, we need to
	//   initialize the filetree with the same stuff as the fsMock.
	fs = fsMock{
		"/tmp/storage/someuser/kittens.png": {
			contents: []byte("if you are reading this you are cute!"),
		},
		"/tmp/storage/someuser/documents/": {
			isDir: true,
		},
	}
	srv := NewServer("/storage", "/tmp/storage", func(r *http.Request) (User, error) {
		return &mockUser{
			name: "testikus",
			quota: 1024*1024*64,
		}, nil
	})
	for _, req := range testRequests {
		rec := httptest.NewRecorder()
		err := srv.Serve(rec, req)
		if err != nil {
			t.Errorf("%s `%s' failed: %v\n", req.Method, req.URL.Path, err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("want 200 OK, got: %d\n", rec.Code)
		}
	}
}

func TestPutDocument(t *testing.T) {
	srv := NewServer("/storage", "/tmp/storage", func(r *http.Request) (User, error) {
		return &mockUser{
			name: "testikus",
			quota: 1024*1024*64,
		}, nil
	})
	fsTest := fsMock{}
	fs = fsTest
	const testContents = "If you are reading this you are cute!"
	req := httptest.NewRequest(http.MethodPut, "/storage/someuser/test.txt", strings.NewReader(testContents))
	rec := httptest.NewRecorder()
	err := srv.Serve(rec, req)
	if err != nil {
		t.Errorf("PUT document failed: %v\n", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("want 201 CREATED, got: %d\n", rec.Code)
	}
	file, err := fsTest.Open("/tmp/storage/someuser/test.txt")
	if err != nil {
		t.Fatalf("expected file to exist")
	}
	buf := make([]byte, 128)
	n, _ := file.Read(buf)
	astr := string(buf[:n])
	if astr != testContents {
		t.Errorf("invalid file contents; got: `%s', want `%s'", astr, testContents)
	}
}

func TestDeleteDocument(t *testing.T) {
	srv := NewServer("/storage", "/tmp/storage", func(r *http.Request) (User, error) {
		return &mockUser{
			name: "testikus",
			quota: 1024*1024*64,
		}, nil
	})
	fsTest := fsMock{
		"/tmp/storage/someuser/test.txt": {
			contents: []byte("if you are reading this you are cute!"),
		},
	}
	fs = fsTest
	req := httptest.NewRequest(http.MethodDelete, "/storage/someuser/test.txt", nil)
	rec := httptest.NewRecorder()
	err := srv.Serve(rec, req)
	if err != nil {
		t.Errorf("DELETE document failed: %v\n", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 OK, got: %d\n", rec.Code)
	}
	_, err = fsTest.Open("/tmp/storage/someuser/test.txt")
	if err == nil {
		t.Error("want: ErrNotExist, got: nil")
	}
}
