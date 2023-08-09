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

// @todo: should the server be a struct?
//   it doesn't make sense to have multiple servers
//   it's more a config than a server,
//   but it allows to use it as an http.Handler (ServeHTTP)
// @todo: Test Migrate/Load/Persist
// @todo: Creating a server (-config) should also Reset()?
// @todo: how exactly should a server be configured and setup?

func mockServer() (*mockFileSystem, Server) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	createUUID = CreateMockUUIDFunc()
	getTime = getMockTime
	server, _ := New(rroot, sroot)
	mfs = CreateMockFS().CreateDirectories(server.sroot)
	Reset()
	return mfs.(*mockFileSystem), server
}

func ExampleServer_GetFolder() {
	_, server := mockServer()
	// Use: server, err := rmsgo.New(remoteRoot, storageRoot)

	// @todo: correct error handling, but we first have to implement good errors

	ts := httptest.NewServer(server)
	defer ts.Close()

	remoteRoot := ts.URL + server.Rroot()

	// GET the currently empty root folder
	r, err := http.Get(remoteRoot + "/")
	if err != nil {
		log.Fatal(err)
	}
	if r.StatusCode != http.StatusOK {
		log.Fatalf("%s %s: %s", r.Request.Method, r.Request.URL, r.Status)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Root ETag: %s\n", r.Header.Get("ETag"))
	fmt.Print(string(bs))
	// Root ETag: 03d871638b18f0b459bf8fd12a58f1d8
	// {
	//   "@context": "http://remotestorage.io/spec/folder-description",
	//   "items": {}
	// }

	// PUT a document
	req, err := http.NewRequest(http.MethodPut, remoteRoot+"/Documents/First.txt", bytes.NewReader([]byte("My first document.")))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "funny/format")

	r, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if r.StatusCode != http.StatusCreated {
		log.Fatalf("%s %s: %s", r.Request.Method, r.Request.URL, r.Status)
	}

	fmt.Printf("Created ETag: %s\n", r.Header.Get("ETag"))
	// Created ETag: f0d0f717619b09cc081bb0c11d9b9c6b

	// GET the now non-empty root folder
	r, err = http.Get(remoteRoot + "/")
	if err != nil {
		log.Fatal(err)
	}
	if r.StatusCode != http.StatusOK {
		log.Fatalf("%s %s: %s", r.Request.Method, r.Request.URL, r.Status)
	}

	bs, err = io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Root ETag: %s\n", r.Header.Get("ETag"))
	fmt.Print(string(bs))
	// Root ETag: ef528a27b48c1b187ef7116f7306358b
	// {
	//   "@context": "http://remotestorage.io/spec/folder-description",
	//   "items": {
	//     "Documents/": {
	//       "ETag": "cc4c6d3bbf39189be874992479b60e2a"
	//     }
	//   }
	// }

	// GET the document's folder
	r, err = http.Get(remoteRoot + "/Documents/")
	if err != nil {
		log.Fatal(err)
	}
	if r.StatusCode != http.StatusOK {
		log.Fatalf("%s %s: %s", r.Request.Method, r.Request.URL, r.Status)
	}

	bs, err = io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Documents/ ETag: %s\n", r.Header.Get("ETag"))
	fmt.Print(string(bs))
	// Documents/ ETag: cc4c6d3bbf39189be874992479b60e2a
	// {
	//   "@context": "http://remotestorage.io/spec/folder-description",
	//   "items": {
	//     "First.txt": {
	//       "Content-Length": 18,
	//       "Content-Type": "funny/format",
	//       "ETag": "f0d0f717619b09cc081bb0c11d9b9c6b",
	//       "Last-Modified": "Mon, 01 Jan 0001 00:00:00 UTC"
	//     }
	//   }
	// }

	// Output:
	// Root ETag: 03d871638b18f0b459bf8fd12a58f1d8
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{}}
	// Created ETag: f0d0f717619b09cc081bb0c11d9b9c6b
	// Root ETag: ef528a27b48c1b187ef7116f7306358b
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{"Documents/":{"ETag":"cc4c6d3bbf39189be874992479b60e2a"}}}
	// Documents/ ETag: cc4c6d3bbf39189be874992479b60e2a
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
