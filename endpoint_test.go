package rmsgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

//fs, server := mockServer()
//
// Option 1:
//ts := httptest.NewServer(server)
//defer ts.Close()
//
//r, err := http.Get(ts.URL)
//if err != nil {
//	t.Error(err)
//}
//_ = r
//
// Option 2:
//req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/storage/", nil)
//w := httptest.NewRecorder()
//server.ServeHTTP(w, req)
//
//resp := w.Result()
//body, err := io.ReadAll(resp.Body)
//if err != nil {
//	t.Error(err)
//}
//_ = body

func mockServer() (*mockFileSystem, Server) {
	var (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	server, _ := New(rroot, sroot)
	mfs := CreateMockFS().CreateDirectories(server.Sroot)
	createUUID = CreateMockUUIDFunc()
	getTime = getMockTime
	Reset()
	return mfs, server
}

func ExampleServer_GetFolder() {
	_, server := mockServer()

	ts := httptest.NewServer(server)
	defer ts.Close()

	// GET the currently empty root folder
	r, err := http.Get(ts.URL + "/storage/")
	if err != nil {
		log.Fatal(err)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(bs))

	// PUT a document
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/storage/Documents/First.txt", bytes.NewReader([]byte("My first document.")))
	req.Header.Set("Content-Type", "funny/format")
	if err != nil {
		log.Fatal(err)
	}

	r, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if r.StatusCode != http.StatusCreated {
		log.Fatal("creating document failed")
	}

	etag := r.Header.Get("ETag")
	if etag == "" {
		log.Fatal("etag missing")
	}
	fmt.Printf("Created ETag: %s\n", etag)

	// GET the now non-empty root folder
	r, err = http.Get(ts.URL + "/storage/")
	if err != nil {
		log.Fatal(err)
	}

	bs, err = io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(bs))

	// GET the document's folder
	r, err = http.Get(ts.URL + "/storage/Documents/")
	if err != nil {
		log.Fatal(err)
	}

	bs, err = io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(bs))

	// Output:
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{}}
	// Created ETag: f0d0f717619b09cc081bb0c11d9b9c6b
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{"Documents/":{"ETag":"cc4c6d3bbf39189be874992479b60e2a"}}}
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{"First.txt":{"Content-Length":18,"Content-Type":"funny/format","ETag":"f0d0f717619b09cc081bb0c11d9b9c6b","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
}

func TestGetFolder(t *testing.T) {
	fs, server := mockServer()
	_ = fs

	ts := httptest.NewServer(server)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/storage/Documents/First.txt", bytes.NewReader([]byte("My first document.")))
	if err != nil {
		t.Error(err)
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusCreated {
		t.Errorf("got: %d, want: %d", r.StatusCode, http.StatusCreated)
	}

	r, err = http.Get(ts.URL + "/storage/")
	if err != nil {
		t.Error(err)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}

	lst := ldjson{}
	json.Unmarshal(bs, &lst)

	items, ok := lst["items"]
	if !ok {
		t.Error("response is missing items field")
	}

	itemsLd, ok := items.(ldjson)
	if !ok {
		t.Error("items field cannot be cast to ldjson")
	}

	doc, ok := itemsLd["Documents/"]
	if !ok {
		t.Error("Documents/ folder missing from items")
	}

	docLd, ok := doc.(ldjson)
	if !ok {
		t.Error("Documents/ field cannot be cast to ldjson")
	}

	etag, ok := docLd["ETag"]
	if !ok {
		t.Error("Documents/ is missing ETag field")
	}

	etagStr, ok := etag.(string)
	if !ok {
		t.Error("ETag is not of type string")
	}

	if etagStr != "b819cb916b90ce59d4064481754672b9" {
		t.Errorf("got: `%s', want: %s", etagStr, "b819cb916b90ce59d4064481754672b9")
	}
}

func TestHeadFolder(t *testing.T) {
}

func TestGetDocument(t *testing.T) {
}

func TestHeadDocument(t *testing.T) {
}

func TestPutDocument(t *testing.T) {
}

func TestDeleteDocument(t *testing.T) {
}
