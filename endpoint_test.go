package rmsgo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/cvanloo/rmsgo.git/mock"
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

func mockServer() *FakeFileSystem {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	fs := Mock()
	fs.CreateDirectories(sroot)
	must(Configure(rroot, sroot, func(err error) {
		log.Fatal(err)
	}))
	Reset()
	return fs
}

func ExampleGetFolder() {
	mockServer()
	// Use: err := rmsgo.Configure(remoteRoot, storageRoot, nil)

	ts := httptest.NewServer(ServeMux{})
	defer ts.Close()

	// server url + remote root
	remoteRoot := ts.URL + rroot

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

// @todo: PUT chunked transfer coding?
// @todo: http1.1, offer switch to http2

// @todo: TestPutDocument
//  - check that parents are silently created as necessary
//  - check that all parent folders have their version updated
//  - Don't provide content type (auto-detect)
//  - put to already existing document
//  - put to folder (must fail) 4XX
//  - conditional (If-Match) with success
//  - conditional (If-Match) with failure 412
//  - conditional (If-Non-Match: *) with success (document does not exist)
//  - conditional (If-Non-Match: *) with failure 412 (document already exists)
//  - 409 when any parent folder name clashes with existing document, or
//    document clashes with existing folder

func TestPutDocument(t *testing.T) {
	fs := mockServer()

	ts := httptest.NewServer(ServeMux{})
	defer ts.Close()

	remoteRoot := ts.URL + rroot

	const content = "My first document."

	req, err := http.NewRequest(http.MethodPut, remoteRoot+"/Documents/First.txt", bytes.NewReader([]byte(content)))
	req.Header.Set("Content-Type", "funny/format")
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
	if e := r.Header.Get("ETag"); e != "f0d0f717619b09cc081bb0c11d9b9c6b" {
		t.Errorf("got: `%s', want: `f0d0f717619b09cc081bb0c11d9b9c6b'", e)
	}

	n, err := Retrieve("/Documents/First.txt")
	if err != nil {
		t.Error(err)
	}
	if n.Mime != "funny/format" {
		t.Errorf("got: `%s', want: funny/format", n.Mime)
	}
	if mustVal(n.Version()).String() != "f0d0f717619b09cc081bb0c11d9b9c6b" {
		t.Errorf("got: `%s', want: `f0d0f717619b09cc081bb0c11d9b9c6b'", n.ETag)
	}
	if n.Name != "First.txt" {
		t.Errorf("got: `%s', want: `First.txt'", n.Name)
	}
	if n.Rname != "/Documents/First.txt" {
		t.Errorf("got: `%s', want: `/Documents/First.txt'", n.Rname)
	}
	if n.Length != int64(len(content)) {
		t.Errorf("got: `%d', want: `%d'", n.Length, len(content))
	}
	if n.IsFolder {
		t.Error("a document should never be a folder")
	}

	bs, err := fs.ReadFile(n.Sname)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != content {
		t.Errorf("got: `%s', want: `%s'", bs, content)
	}
}

func TestGetFolder(t *testing.T) {
	mockServer()

	ts := httptest.NewServer(ServeMux{})
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

	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
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
	mockServer()

	ts := httptest.NewServer(ServeMux{})
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

	r, err = http.Head(ts.URL + "/storage/")
	if err != nil {
		t.Error(err)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if len(bs) != 0 {
		t.Error("the response to a head request should have an empty body")
	}

	if etag := r.Header.Get("ETag"); etag != "6495a5d8eb9f9c5343a03540b6e3dfaa" {
		t.Errorf("got: `%s', want: 6495a5d8eb9f9c5343a03540b6e3dfaa", etag)
	}

	if l := r.Header.Get("Content-Length"); l != "130" {
		t.Errorf("got: `%s', want: 130", l)
	}
}

func TestGetDocument(t *testing.T) {
	mockServer()

	ts := httptest.NewServer(ServeMux{})
	defer ts.Close()

	const content = "My first document."

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/storage/Documents/First.txt", bytes.NewReader([]byte(content)))
	req.Header.Set("Content-Type", "funny/format")
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

	r, err = http.Get(ts.URL + "/storage/Documents/First.txt")
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %d, want: %d", r.StatusCode, http.StatusOK)
	}
	if l := r.Header.Get("Content-Length"); l != fmt.Sprintf("%d", len(content)) {
		t.Errorf("got: %s, want: %d", l, len(content))
	}
	if e := r.Header.Get("ETag"); e != "f0d0f717619b09cc081bb0c11d9b9c6b" {
		t.Errorf("got: `%s, want: f0d0f717619b09cc081bb0c11d9b9c6b", e)
	}
	if ct := r.Header.Get("Content-Type"); ct != "funny/format" {
		t.Errorf("got: `%s', want: funny/format", ct)
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if len(bs) != len(content) {
		t.Errorf("mismatched content length; got: %d, want: %d", len(bs), len(content))
	}
	if string(bs) != content {
		t.Errorf("got: `%s', want: `%s'", bs, content)
	}
}

func TestHeadDocument(t *testing.T) {
	mockServer()

	ts := httptest.NewServer(ServeMux{})
	defer ts.Close()

	const content = "My first document."

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/storage/Documents/First.txt", bytes.NewReader([]byte(content)))
	req.Header.Set("Content-Type", "funny/format")
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

	r, err = http.Head(ts.URL + "/storage/Documents/First.txt")
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %d, want: %d", r.StatusCode, http.StatusOK)
	}
	if l := r.Header.Get("Content-Length"); l != fmt.Sprintf("%d", len(content)) {
		t.Errorf("got: %s, want: %d", l, len(content))
	}
	if e := r.Header.Get("ETag"); e != "f0d0f717619b09cc081bb0c11d9b9c6b" {
		t.Errorf("got: `%s, want: f0d0f717619b09cc081bb0c11d9b9c6b", e)
	}
	if ct := r.Header.Get("Content-Type"); ct != "funny/format" {
		t.Errorf("got: `%s', want: funny/format", ct)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if len(bs) != 0 {
		t.Errorf("the response to a head request should have an empty body; got: `%s'", bs)
	}
}

func TestDeleteDocument(t *testing.T) {
	fs := mockServer()

	ts := httptest.NewServer(ServeMux{})
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
	firstETag := r.Header.Get("ETag")
	if firstETag != "cccbdca11c50776583965bf7631964d6" {
		t.Errorf("got: `%s', want: `cccbdca11c50776583965bf7631964d6'", firstETag)
	}

	n, err := Retrieve("/Documents/First.txt")
	if err != nil {
		t.Error(err)
	}

	_, err = fs.Stat(n.Sname)
	if err != nil {
		t.Error(err)
	}

	req, err = http.NewRequest(http.MethodDelete, ts.URL+"/storage/Documents/First.txt", nil)
	if err != nil {
		t.Error(err)
	}

	r, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("got: `%d', want: `%d'", r.StatusCode, http.StatusOK)
	}
	if e := r.Header.Get("ETag"); e != firstETag {
		t.Errorf("got: `%s', want: `%s'", e, firstETag)
	}

	_, err = fs.Stat(n.Sname)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}

	_, err = Retrieve("/Documents/")
	if err != ErrNotExist {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotExist)
	}
}
