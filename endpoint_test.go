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
	"time"

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

func mockServer() {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	must(Configure(rroot, sroot, func(err error) {
		log.Fatal(err)
	}))
	Reset()
}

func ExampleGetFolder() { // @todo: overwork
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

// @todo: PUT test chunked transfer-encoding
// @todo: test requests with http1.1, and with switch to http2

func TestPutDocument(t *testing.T) {
	const (
		testContent      = "[...] It is written in Lisp, which is the only computer language that is beautiful." // @todo: change to something else
		testMime         = "wise/quote"
		testDocument     = "/Quotes/Neal Stephenson.txt"
		testDocumentEtag = "3dc42d11db35b8354dc06c46a53c9c9d"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", testMime)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusCreated {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
	}
	if e := r.Header.Get("ETag"); e != testDocumentEtag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag)
	}

	r, err = http.Get(remoteRoot + testDocument)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
	}
	if l := r.Header.Get("Content-Length"); l != fmt.Sprint(len(testContent)) {
		t.Errorf("got: %s, want: %d", l, len(testContent))
	}
	if l := r.Header.Get("Content-Type"); l != testMime {
		t.Errorf("got: `%s', want: `%s'", l, testMime)
	}
	if e := r.Header.Get("ETag"); e != testDocumentEtag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestPutDocumentSame(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag1)
		}
	}

	{
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if l := r.Header.Get("Content-Length"); l != fmt.Sprint(len(testContent1)) {
			t.Errorf("got: %s, want: %d", l, len(testContent1))
		}
		if l := r.Header.Get("Content-Type"); l != testMime {
			t.Errorf("got: `%s', want: `%s'", l, testMime)
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag1)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != testContent1 {
			t.Errorf("got: `%s', want: `%s'", bs, testContent1)
		}
	}

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag2 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag2)
		}
	}

	{
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if l := r.Header.Get("Content-Length"); l != fmt.Sprint(len(testContent2)) {
			t.Errorf("got: %s, want: %d", l, len(testContent2))
		}
		if l := r.Header.Get("Content-Type"); l != testMime {
			t.Errorf("got: `%s', want: `%s'", l, testMime)
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag2 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag2)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != testContent2 {
			t.Errorf("got: `%s', want: `%s'", bs, testContent2)
		}
	}
}

func TestPutDocumentIfMatchSuccess(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag1)
		}
	}

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", testDocumentEtag1) // Set If-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag2 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag2)
		}
	}
}

func TestPutDocumentIfMatchFail(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"
		wrongETag    = "3de26fc06d5d1e20ff96a8142cd6fabf"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag1)
		}
	}

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", wrongETag) // Set If-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusPreconditionFailed))
		}
	}
}

func TestPutDocumentIfNonMatchSuccess(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag = "33f7b41f98820961b12134677ba3f231"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", testMime)
	req.Header.Set("If-Non-Match", "*") // Set If-Non-Match header!
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusCreated {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
	}
	if e := r.Header.Get("ETag"); e != testDocumentEtag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag)
	}
}

func TestPutDocumentIfNonMatchFail(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Non-Match", "*") // Set If-Non-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentEtag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag1)
		}
	}

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Non-Match", "*") // Set If-Non-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusPreconditionFailed))
		}
	}
}

func TestPutDocumentSilentlyCreateAncestors(t *testing.T) {
	const (
		rmsContext          = "http://remotestorage.io/spec/folder-description"
		testContent         = "[...] It is written in Lisp, which is the only computer language that is beautiful." // sorry Go
		testMime            = "wise/quote"
		testDocument        = "/Quotes/Neal Stephenson.txt"
		testDocumentName    = "Neal Stephenson.txt"
		testDocumentEtag    = "3dc42d11db35b8354dc06c46a53c9c9d"
		testDocumentDir     = "/Quotes/"
		testDocumentDirETag = "3de26fc06d5d1e20ff96a8142cd6fabf"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", testMime)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusCreated {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
	}
	if e := r.Header.Get("ETag"); e != testDocumentEtag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag)
	}

	r, err = http.Get(remoteRoot + testDocumentDir)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
	}
	if e := r.Header.Get("ETag"); e != testDocumentDirETag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentDirETag)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}

	lst := LDjson{}
	err = json.Unmarshal(bs, &lst)
	if err != nil {
		t.Error(err)
	}
	ctx, err := LDGet[string](lst, "@context")
	if err != nil {
		t.Error(err)
	}
	if ctx != rmsContext {
		t.Errorf("got: `%s', want: `%s'", ctx, rmsContext)
	}
	docLst, err := LDGet[LDjson](lst, "items", testDocumentName)
	if err != nil {
		t.Error(err)
	}

	e, err := LDGet[string](docLst, "ETag")
	if err != nil {
		t.Error(err)
	}
	if e != testDocumentEtag {
		t.Errorf("got: `%s', want: `%s'", e, testDocumentEtag)
	}
	mime, err := LDGet[string](docLst, "Content-Type")
	if err != nil {
		t.Error(err)
	}
	if mime != testMime {
		t.Errorf("got: `%s', want: `%s'", mime, testMime)
	}
	l, err := LDGet[float64](docLst, "Content-Length")
	if err != nil {
		t.Error(err)
	}
	if l != float64(len(testContent)) {
		t.Errorf("got: %f, want: %d", l, len(testContent))
	}
	modt, err := LDGet[string](docLst, "Last-Modified")
	if err != nil {
		t.Error(err)
	}
	tme, err := time.Parse(rmsTimeFormat, modt)
	if err != nil {
		t.Error(err)
	}
	_ = tme // @todo: we can't really verify this right now
}

func TestPutDocumentUpdatesAncestorETags(t *testing.T) {
	const (
		testMime = "application/x-subrip"

		testContent1      = "Run for the heavens Sing to the stars Love like a lover Shine in the dark Shout like an army Sound the alarm I am a burning [...] Heart"
		testDocument1     = "/Lyrics/SVRCINA.srt"
		testDocument1Name = "SVRCINA.srt"
		testDocument1ETag = "6f9cd924b8654c70d5bec5f96491f55e"

		testContent2      = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2     = "/Lyrics/Raizer.srt"
		testDocument2Name = "Raizer.srt"
		testDocument2ETag = "9323d12cc9b79190804d1c6b9c2708f3"

		testDocumentDir      = "/Lyrics/"
		testDocumentDirETag1 = "bc42be34636d852ecd65d8b3a3857a62"
		testDocumentDirETag2 = "6dedad00af566fbfbb811661e6f88387"

		testRootETag1 = "e441dd5f0422b305cf30bca3bbdefd68"
		testRootETag2 = "628e5ebbbcf131c9103ebd51019d1b7e"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	// PUT first document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocument1ETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocument1ETag)
		}
	}

	// GET parent folder ETag
	{
		r, err := http.Get(remoteRoot + testDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if e := r.Header.Get("ETag"); e != testDocumentDirETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentDirETag1)
		}
		// @todo: maybe we want to validate this as well?
		//bs, err := io.ReadAll(r.Body)
		//if err != nil {
		//	t.Error(err)
		//}
	}

	// Get root folder ETag
	{
		r, err := http.Get(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if e := r.Header.Get("ETag"); e != testRootETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testRootETag1)
		}
		// @todo: maybe we want to validate this as well?
		//bs, err := io.ReadAll(r.Body)
		//if err != nil {
		//	t.Error(err)
		//}
	}

	// PUT second document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocument2ETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocument2ETag)
		}
	}

	// GET parent folder ETag
	{
		r, err := http.Get(remoteRoot + testDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if e := r.Header.Get("ETag"); e != testDocumentDirETag2 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentDirETag2)
		}
		// @todo: maybe we want to validate this as well?
		//bs, err := io.ReadAll(r.Body)
		//if err != nil {
		//	t.Error(err)
		//}
	}

	// Get root folder ETag
	{
		r, err := http.Get(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if e := r.Header.Get("ETag"); e != testRootETag2 {
			t.Errorf("got: `%s', want: `%s'", e, testRootETag2)
		}
		// @todo: maybe we want to validate this as well?
		//bs, err := io.ReadAll(r.Body)
		//if err != nil {
		//	t.Error(err)
		//}
	}
}

func TestPutDocumentAutodetectContentType(t *testing.T) {
	const (
		testContent = `“But the plans were on display…”
“On display? I eventually had to go down to the cellar to find them.”
“That’s the display department.”
“With a flashlight.”
“Ah, well, the lights had probably gone.”
“So had the stairs.”
“But look, you found the notice, didn’t you?”
“Yes,” said Arthur, “yes I did. It was on display in the bottom of a locked filing cabinet stuck in a disused lavatory with a sign on the door saying ‘Beware of the Leopard.”`
		testDocument     = "/Quotes/Douglas Adams"
		testDocumentETag = "e3e1c1d7f6952350b93d4935aa412497"
		testMime         = "text/plain; charset=utf-8"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
		if err != nil {
			t.Error(err)
		}
		// don't set Content-Type header
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	{
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("got: `%s', want: `no-cache'", cc)
		}
		if l := r.Header.Get("Content-Length"); l != fmt.Sprint(len(testContent)) {
			t.Errorf("got: %s, want: %d", l, len(testContent))
		}
		if l := r.Header.Get("Content-Type"); l != testMime {
			t.Errorf("got: `%s', want: `%s'", l, testMime)
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != testContent {
			t.Errorf("got: `%s', want: `%s'", bs, testContent)
		}
	}
}

func TestPutDocumentAsFolderFails(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+"/Edward/M/D/Teach/", bytes.NewReader([]byte("HA! Liar. I have to write sentences with multiple dependend clausse in order to repair the damage of your 5 word rhetorical cluster grenade.")))
	if err != nil {
		t.Error(err)
	}
	// (don't set Content-Type header)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusBadRequest))
	}
}

func TestPutDocumentClashesWithFolderFails(t *testing.T) {
	const (
		testMime = "application/x-subrip"

		testContent1      = "Run for the heavens Sing to the stars Love like a lover Shine in the dark Shout like an army Sound the alarm I am a burning [...] Heart"
		testDocument1     = "/Lyrics/Favourite/SVRCINA.srt"
		testDocument1ETag = "6f9cd924b8654c70d5bec5f96491f55e"

		testContent2  = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2 = "/Lyrics/Favourite" // this is going to clash with the already existing /Lyrics/Favourite/ folder

		expectedConflictPath = "/Lyrics/Favourite"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	// PUT first document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocument1ETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocument1ETag)
		}
	}

	// PUT second document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusConflict {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusConflict))
		}

		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		errLst := LDjson{}
		err = json.Unmarshal(bs, &errLst)
		if err != nil {
			t.Error(err)
		}
		conflictPath, err := LDGet[string](errLst, "data", "conflict")
		if err != nil {
			t.Error(err)
		}
		if conflictPath != expectedConflictPath {
			t.Errorf("got: `%s', want: `%s'", conflictPath, expectedConflictPath)
		}
	}
}

func TestPutDocumentAncestorFolderClashesWithDocumentFails(t *testing.T) {
	const (
		testMime = "application/x-subrip"

		testContent1      = "Run for the heavens Sing to the stars Love like a lover Shine in the dark Shout like an army Sound the alarm I am a burning [...] Heart"
		testDocument1     = "/Lyrics/Favourite"
		testDocument1ETag = "421432bf1a9f22883bac81ad1714dd90"

		testContent2  = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2 = "/Lyrics/Favourite/STARSET.srt" // /Lyrics/Favourite/ is going to clash with the already existing /Lyrics/Favourite document

		expectedConflictPath = "/Lyrics/Favourite"
	)
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	// PUT first document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocument1ETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocument1ETag)
		}
	}

	// PUT second document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusConflict {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusConflict))
		}

		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		errLst := LDjson{}
		err = json.Unmarshal(bs, &errLst)
		if err != nil {
			t.Error(err)
		}
		conflictPath, err := LDGet[string](errLst, "data", "conflict")
		if err != nil {
			t.Error(err)
		}
		if conflictPath != expectedConflictPath {
			t.Errorf("got: `%s', want: `%s'", conflictPath, expectedConflictPath)
		}
	}
}

// -------

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

	lst := LDjson{}
	err = json.Unmarshal(bs, &lst)
	if err != nil {
		t.Error(err)
	}

	items, ok := lst["items"]
	if !ok {
		t.Error("response is missing items field")
	}

	itemsLd, ok := items.(LDjson)
	if !ok {
		t.Error("items field cannot be cast to ldjson")
	}

	doc, ok := itemsLd["Documents/"]
	if !ok {
		t.Error("Documents/ folder missing from items")
	}

	docLd, ok := doc.(LDjson)
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
	firstETag := r.Header.Get("ETag")
	if firstETag != "cccbdca11c50776583965bf7631964d6" {
		t.Errorf("got: `%s', want: `cccbdca11c50776583965bf7631964d6'", firstETag)
	}

	n, err := Retrieve("/Documents/First.txt")
	if err != nil {
		t.Error(err)
	}

	_, err = FS.Stat(n.sname)
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

	_, err = FS.Stat(n.sname)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}

	_, err = Retrieve("/Documents/")
	if err != ErrNotExist {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotExist)
	}
}
