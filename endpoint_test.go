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

func ExampleGetFolder() {
	mockServer()

	ts := httptest.NewServer(ServeMux{})
	defer ts.Close()

	// server url + remote root
	remoteRoot := ts.URL + rroot

	// GET the currently empty root folder
	{
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
	}

	// PUT a document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+"/Documents/First.txt", bytes.NewReader([]byte("My first document.")))
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Content-Type", "funny/format") // mime type is auto-detected if not specified
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		if r.StatusCode != http.StatusCreated {
			log.Fatalf("%s %s: %s", r.Request.Method, r.Request.URL, r.Status)
		}
		fmt.Printf("Created ETag: %s\n", r.Header.Get("ETag"))
		// Created ETag: f0d0f717619b09cc081bb0c11d9b9c6b
	}

	// GET the now NON-empty root folder
	{
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
		// Root ETag: ef528a27b48c1b187ef7116f7306358b
		// {
		//   "@context": "http://remotestorage.io/spec/folder-description",
		//   "items": {
		//     "Documents/": {
		//       "ETag": "cc4c6d3bbf39189be874992479b60e2a"
		//     }
		//   }
		// }
	}

	// GET the document's folder
	{
		r, err := http.Get(remoteRoot + "/Documents/")
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
	}

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
		testContent      = "The material is classified. Its composition is classified. Its use in the weapon is classified, and the process itself is classified."
		testMime         = "top/secret"
		testDocument     = "/Classified/FOGBANK.txt"
		testDocumentEtag = "60ca7ee51a4a4886d00ae2470457b206"
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

func TestGetFolder(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent = `> If you masturbated and went to the grocery store, and I
> ask you what you did today, and you tell me you went to the grocery
> store, that is not lying, you are just hiding implementation details.\
> -- <cite>ThePrimeagen, Twitch.tv</cite>`
		testDocument     = "/Quotes/Twitch/ThePrimeagen.md"
		testMime         = "text/plain; charset=utf-8"
		testDocumentETag = "8c2d95a5232b32d1ad8c794313c0c549"

		testDocumentDir     = "/Quotes/Twitch/"
		testDocumentDirETag = "25c3d4a9bc64c223d9b8c07e8336952d"

		responseBody = `{"@context":"http://remotestorage.io/spec/folder-description","items":{"ThePrimeagen.md":{"Content-Length":242,"Content-Type":"text/plain; charset=utf-8","ETag":"8c2d95a5232b32d1ad8c794313c0c549","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
` // don't forget newline
	)

	{
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
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	r, err := http.Get(remoteRoot + testDocumentDir)
	if err != nil {
		t.Error(err)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
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
	if string(bs) != responseBody {
		t.Errorf("got: `%s', want: `%s'", bs, responseBody)
	}
}

func TestGetFolderEmpty(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testDocumentDirETag = "03d871638b18f0b459bf8fd12a58f1d8"
		responseBody        = `{"@context":"http://remotestorage.io/spec/folder-description","items":{}}
` // don't forget newline
	)

	// we can't have empty folders except for root
	r, err := http.Get(remoteRoot + "/")
	if err != nil {
		t.Error(err)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
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
	if string(bs) != responseBody {
		t.Errorf("got: `%s', want: `%s'", bs, responseBody)
	}
}

func TestGetFolderNotFound(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const responseBody = `{"data":{"rname":"/nonexistent/"},"description":"The requested folder does not exist on the server.","message":"folder not found","url":""}
` // don't forget newline

	r, err := http.Get(remoteRoot + "/nonexistent/")
	if err != nil {
		t.Error(err)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
	}
	// @todo: should error responses also include a Cache-Control header?
	//if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
	//	t.Errorf("got: `%s', want: `no-cache'", cc)
	//}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != responseBody {
		t.Errorf("got: `%s', want: `%s'", bs, responseBody)
	}
}

func TestGetFolderIfNonMatchRevMatches(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent      = `You may disagree with this idiom, and that's okay, because it's enforced by the compiler. You're welcome.`
		testDocument     = "/public/go_devs_prbly"
		testMime         = "text/joke"
		testDocumentETag = "3e507240501005a29cc22520bd333f79"

		testDocumentDir     = "/public/"
		testDocumentDirETag = "660d6f3f14d933aa8e60bb17e7cae7e8"
	)

	{
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
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	req, err := http.NewRequest(http.MethodGet, remoteRoot+testDocumentDir, nil)
	if err != nil {
		t.Error(err)
	}
	// include revision of the folder we're about to GET
	req.Header.Set("If-Non-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentDirETag))
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusNotModified {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotModified))
	}
}

func TestGetFolderIfNonMatchRevNoMatch(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent      = `You may disagree with this idiom, and that's okay, because it's enforced by the compiler. You're welcome.`
		testDocument     = "/public/go_devs_prbly"
		testMime         = "text/joke"
		testDocumentETag = "3e507240501005a29cc22520bd333f79"

		testDocumentDir     = "/public/"
		testDocumentDirETag = "660d6f3f14d933aa8e60bb17e7cae7e8"

		responseBody = `{"@context":"http://remotestorage.io/spec/folder-description","items":{"go_devs_prbly":{"Content-Length":105,"Content-Type":"text/joke","ETag":"3e507240501005a29cc22520bd333f79","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
` // don't forget newline
	)

	{
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
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	req, err := http.NewRequest(http.MethodGet, remoteRoot+testDocumentDir, nil)
	if err != nil {
		t.Error(err)
	}
	// none of the revisions match our public/ folder
	req.Header.Set("If-Non-Match", "03d871638b18f0b459bf8fd12a58f1d8, 3e507240501005a29cc22520bd333f79, 33f7b41f98820961b12134677ba3f231")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
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
	if string(bs) != responseBody {
		t.Errorf("got: `%s', want: `%s'", bs, responseBody)
	}
}

func TestHeadFolder(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testDocumentETag = "eabd59d0c27b78077e391800e7cf8777"
		rootETag         = "962e336a5a324e5adf7d8eca569e0c70"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+"/yt/rendle/citation", bytes.NewReader([]byte("In space no one can set a breakpoint.")))
		if err != nil {
			t.Error(err)
		}
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

	r, err := http.Head(remoteRoot + "/")
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if etag := r.Header.Get("ETag"); etag != rootETag {
		t.Errorf("got: `%s', want: `%s'", etag, rootETag)
	}
	if l := r.Header.Get("Content-Length"); l != "123" {
		t.Errorf("got: `%s', want: 123", l)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
	}
	if len(bs) != 0 {
		t.Error("the response to a head request should have an empty body")
	}
}

// We don't need any more HEAD folder test cases.
// The implementation logic is essentially the same: a HEAD request is also
// directed to the GetFolder handler.
// (Go's HTTP lib takes care of not including the body in the response.)

func TestGetDocument(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent      = "Lisp is a perfectly logical language to use." // 😤
		testMime         = "text/plain; charset=utf-8"
		testDocument     = "/everyone/would/agree/Fridman Quote"
		testDocumentETag = "1439461086c3263260ca619a30278741"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
		req.Header.Set("Content-Type", testMime)
		if err != nil {
			t.Error(err)
		}
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
	if l := r.Header.Get("Content-Length"); l != fmt.Sprintf("%d", len(testContent)) {
		t.Errorf("got: %s, want: %d", l, len(testContent))
	}
	if e := r.Header.Get("ETag"); e != testDocumentETag {
		t.Errorf("got: `%s, want: `%s'", e, testDocumentETag)
	}
	if ct := r.Header.Get("Content-Type"); ct != testMime {
		t.Errorf("got: `%s', want: `%s'", ct, testMime)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const response = `{"data":{"rname":"/inexistent/document"},"description":"The requested document does not exist on the server.","message":"document not found","url":""}
` // don't forget newline

	r, err := http.Get(remoteRoot + "/inexistent/document")
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != response {
		t.Errorf("got: `%s', want: `%s'", bs, response)
	}
}

func TestGetDocumentIfNonMatchRevMatches(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent = `A class takes a sensible idea: defining data along
with methods that act on that data, and then drives it off a cliff by
adding inheritance and subtype polymorphism. It should be no surprise
that a bunch of class-obsessed aristocratic oldies in the 60s, who
probably spent all their time deciding which child should inherit most
of the estate, decided to add a construct named 'class' which revolved
around inheritance.`
		testMime         = "text/plain; charset=utf-8"
		testDocument     = "/gh/jesseduffield/OK"
		testDocumentETag = "9e57ccd4ec8d848d413e3e363cd48cdc"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
		req.Header.Set("Content-Type", testMime)
		if err != nil {
			t.Error(err)
		}
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

	req, err := http.NewRequest(http.MethodGet, remoteRoot+testDocument, nil)
	if err != nil {
		t.Error(err)
	}
	// include revision of the document we're about to GET
	req.Header.Set("If-Non-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentETag))
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusNotModified {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotModified))
	}
}

func TestGetDocumentIfNonMatchRevNoMatch(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent = `A class takes a sensible idea: defining data along
with methods that act on that data, and then drives it off a cliff by
adding inheritance and subtype polymorphism. It should be no surprise
that a bunch of class-obsessed aristocratic oldies in the 60s, who
probably spent all their time deciding which child should inherit most
of the estate, decided to add a construct named 'class' which revolved
around inheritance.`
		testMime         = "text/plain; charset=utf-8"
		testDocument     = "/gh/jesseduffield/OK"
		testDocumentETag = "9e57ccd4ec8d848d413e3e363cd48cdc"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
		req.Header.Set("Content-Type", testMime)
		if err != nil {
			t.Error(err)
		}
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

	req, err := http.NewRequest(http.MethodGet, remoteRoot+testDocument, nil)
	if err != nil {
		t.Error(err)
	}
	// revision of our document NOT included
	req.Header.Set("If-Non-Match", "03d871638b18f0b459bf8fd12a58f1d8, cc4c6d3bbf39189be874992479b60e2a, f0d0f717619b09cc081bb0c11d9b9c6b")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("got: `%s', want: `no-cache'", cc)
	}
	if l := r.Header.Get("Content-Length"); l != fmt.Sprintf("%d", len(testContent)) {
		t.Errorf("got: %s, want: %d", l, len(testContent))
	}
	if e := r.Header.Get("ETag"); e != testDocumentETag {
		t.Errorf("got: `%s, want: `%s'", e, testDocumentETag)
	}
	if ct := r.Header.Get("Content-Type"); ct != testMime {
		t.Errorf("got: `%s', want: `%s'", ct, testMime)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestHeadDocument(t *testing.T) {
	mockServer()
	ts := httptest.NewServer(ServeMux{})
	remoteRoot := ts.URL + rroot
	defer ts.Close()

	const (
		testContent      = "Go is better than everything. In my opinion Go is even better than English."
		testMime         = "text/plain; charset=us-ascii"
		testDocument     = "/twitch.tv/ThePrimeagen"
		testDocumentETag = "d53cc497c102d476599e7853cb3c5601"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
		req.Header.Set("Content-Type", testMime)
		if err != nil {
			t.Error(err)
		}
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

	r, err := http.Head(remoteRoot + testDocument)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("got: %d, want: %d", r.StatusCode, http.StatusOK)
	}
	if l := r.Header.Get("Content-Length"); l != fmt.Sprintf("%d", len(testContent)) {
		t.Errorf("got: %s, want: %d", l, len(testContent))
	}
	if e := r.Header.Get("ETag"); e != testDocumentETag {
		t.Errorf("got: `%s, want: `%s'", e, testDocumentETag)
	}
	if ct := r.Header.Get("Content-Type"); ct != testMime {
		t.Errorf("got: `%s', want: `%s'", ct, testMime)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if len(bs) != 0 {
		t.Error("the response to a head request should have an empty body")
	}
}

// @todo: write tests for DELETE document

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
