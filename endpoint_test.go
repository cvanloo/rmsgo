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
	"time"

	. "github.com/cvanloo/rmsgo/mock"
)

// @todo: test CORS

func mockServer() {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	opts.AllowAnyReadWrite()
	Reset()
}

func ExampleGetFolder() {
	mockServer()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// server url + remote root
	remoteRoot := ts.URL + g.rroot

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
		// Created ETag: ef528a27b48c1b187ef7116f7306358b
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
		// Root ETag: 6e39dd4634b22d22408e09e8cf0c9f82
		// {
		//   "@context": "http://remotestorage.io/spec/folder-description",
		//   "items": {
		//     "Documents/": {
		//       "ETag": "f7d6e4d650182817a94d6ae61fe6a772"
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
		// Documents/ ETag: f7d6e4d650182817a94d6ae61fe6a772
		// {
		//   "@context": "http://remotestorage.io/spec/folder-description",
		//   "items": {
		//     "First.txt": {
		//       "Content-Length": 18,
		//       "Content-Type": "funny/format",
		//       "ETag": "ef528a27b48c1b187ef7116f7306358b",
		//       "Last-Modified": "Mon, 01 Jan 0001 00:00:00 UTC"
		//     }
		//   }
		// }
	}

	// Output:
	// Root ETag: 03d871638b18f0b459bf8fd12a58f1d8
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{}}
	// Created ETag: ef528a27b48c1b187ef7116f7306358b
	// Root ETag: 6e39dd4634b22d22408e09e8cf0c9f82
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{"Documents/":{"ETag":"f7d6e4d650182817a94d6ae61fe6a772"}}}
	// Documents/ ETag: f7d6e4d650182817a94d6ae61fe6a772
	// {"@context":"http://remotestorage.io/spec/folder-description","items":{"First.txt":{"Content-Length":18,"Content-Type":"funny/format","ETag":"ef528a27b48c1b187ef7116f7306358b","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
}

// @todo: PUT test chunked transfer-encoding
// @todo: test requests with http1.1, and with switch to http2

func TestPutDocument(t *testing.T) {
	const (
		testContent      = "The material is classified. Its composition is classified. Its use in the weapon is classified, and the process itself is classified."
		testMime         = "top/secret"
		testDocument     = "/Classified/FOGBANK.txt"
		testDocumentEtag = "f106b4d223935ad237e8903c6a2eb36f"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentEtag1 = "9b7c85e3dab0b71cebd9743e7fad9f52"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "250e941b92252859dddfd87dad341b14"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentEtag1 = "9b7c85e3dab0b71cebd9743e7fad9f52"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "250e941b92252859dddfd87dad341b14"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentETag1 = "9b7c85e3dab0b71cebd9743e7fad9f52"

		testContent2 = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag1)
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
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag1)
		}
	}
}

func TestPutDocumentIfNonMatchSuccess(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag = "9b7c85e3dab0b71cebd9743e7fad9f52"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", testMime)
	req.Header.Set("If-None-Match", "*") // Set If-None-Match header!
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
		testDocumentETag1 = "9b7c85e3dab0b71cebd9743e7fad9f52"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "e6258bdf4356eeb85334f8f2de857d3f"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-None-Match", "*") // Set If-None-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag1)
		}
	}

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2)))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-None-Match", "*") // Set If-None-Match header!
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusPreconditionFailed))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag1)
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
		testDocumentEtag    = "b6c3b3eef63f4b9077cb8ad22934ba14"
		testDocumentDir     = "/Quotes/"
		testDocumentDirETag = "3868c11b85d6af588df87f4d73237da6"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
	tme, err := time.Parse(timeFormat, modt)
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
		testDocument1ETag = "e441dd5f0422b305cf30bca3bbdefd68"

		testContent2      = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2     = "/Lyrics/Raizer.srt"
		testDocument2Name = "Raizer.srt"
		testDocument2ETag = "5150b056ebc4af9674806717d7a0ecd6"

		testDocumentDir      = "/Lyrics/"
		testDocumentDirETag1 = "25878842e53f4aacdcc46940e9bbe1bd"
		testDocumentDirETag2 = "e573d0b6954cc13110e4baa481e35f9a"

		testRootETag1 = "bf101ee244dfdba96487dcb7fa0dd18f"
		testRootETag2 = "ddfa5c20ac8c789ebbb552c59b27aa4c"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentETag = "666b6cb64f90e66aa960ef47bf61bbb5"
		testMime         = "application/octet-stream"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, remoteRoot+"/Edward/M/D/Teach/", bytes.NewReader([]byte("HA! Liar. I have to write sentences with multiple dependent clauses in order to repair the damage of your 5 word rhetorical cluster grenade.")))
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
		testDocument1ETag = "3dbd75b638098b9c71ffec31e78ce412"

		testContent2  = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2 = "/Lyrics/Favourite" // this is going to clash with the already existing /Lyrics/Favourite/ folder

		expectedConflictPath = "/Lyrics/Favourite"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocument1ETag = "429bdf3f4f788d6f522fd60042169bfa"

		testContent2  = "I'm attracted to the sky To the sky To the sky Every life I learn to fly Learn to fly Learn to fly"
		testDocument2 = "/Lyrics/Favourite/STARSET.srt" // /Lyrics/Favourite/ is going to clash with the already existing /Lyrics/Favourite document

		expectedConflictPath = "/Lyrics/Favourite"
	)
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent = `> If you masturbated and went to the grocery store, and I
> ask you what you did today, and you tell me you went to the grocery
> store, that is not lying, you are just hiding implementation details.\
> -- <cite>ThePrimeagen, Twitch.tv</cite>`
		testDocument     = "/Quotes/Twitch/ThePrimeagen.md"
		testMime         = "text/plain; charset=utf-8"
		testDocumentETag = "41dfa341884bfed834d324277cc26dad"

		testDocumentDir     = "/Quotes/Twitch/"
		testDocumentDirETag = "a3e251b417a9b78281df743395a8b2b3"

		responseBody = `{"@context":"http://remotestorage.io/spec/folder-description","items":{"ThePrimeagen.md":{"Content-Length":242,"Content-Type":"text/plain; charset=utf-8","ETag":"41dfa341884bfed834d324277cc26dad","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
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

func TestGetFolderEmpty(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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

func TestGetFolderNotFound(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const responseBody = `{"data":{"rname":"/nonexistent/"},"description":"The requested folder does not exist on the server.","message":"folder not found","url":""}
` // don't forget newline

	r, err := http.Get(remoteRoot + "/nonexistent/")
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/ld+json" {
		t.Errorf("got: `%s', want: `application/ld+json'", ct)
	}
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = `You may disagree with this idiom, and that's okay, because it's enforced by the compiler. You're welcome.`
		testDocument     = "/public/go_devs_prbly"
		testMime         = "text/joke"
		testDocumentETag = "6a372e37db0bfb4240fe477f032c5ba6"

		testDocumentDir     = "/public/"
		testDocumentDirETag = "25e6318e0d6291e73d3124d9d1819d7f"
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
	req.Header.Set("If-None-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentDirETag))
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = `You may disagree with this idiom, and that's okay, because it's enforced by the compiler. You're welcome.`
		testDocument     = "/public/go_devs_prbly"
		testMime         = "text/joke"
		testDocumentETag = "6a372e37db0bfb4240fe477f032c5ba6"

		testDocumentDir     = "/public/"
		testDocumentDirETag = "25e6318e0d6291e73d3124d9d1819d7f"

		responseBody = `{"@context":"http://remotestorage.io/spec/folder-description","items":{"go_devs_prbly":{"Content-Length":105,"Content-Type":"text/joke","ETag":"6a372e37db0bfb4240fe477f032c5ba6","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
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
	req.Header.Set("If-None-Match", "03d871638b18f0b459bf8fd12a58f1d8, 3e507240501005a29cc22520bd333f79, 33f7b41f98820961b12134677ba3f231")
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

func TestGetFolderThatIsADocumentFails(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = "Since I am innocent of this crime, sir, I find it decidedly inconvenient that the gun was never found."
		testDocument     = "/Quotes/Movies/Shawshank Redemption"
		testDocumentETag = "2cd20dc9cc76d950fdbc25acc36f724f"

		testDirThatActuallyIsADocument = "/Quotes/Movies/Shawshank Redemption/"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
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

	r, err := http.Get(remoteRoot + testDirThatActuallyIsADocument)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusBadRequest))
	}
}

func TestHeadFolder(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testDocumentETag = "8bcad8e369ee8b5a6cfc069ca5b4d315"
		rootETag         = "1efea8df94b2ede344547180f4fb3002"
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = "Lisp is a perfectly logical language to use." // 😤
		testMime         = "text/plain; charset=utf-8"
		testDocument     = "/everyone/would/agree/Fridman Quote"
		testDocumentETag = "ab7e8cd739bb5d94a3a24d3a6b756df0"
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentETag = "c56336c060b8a95cc1af1f1e55136ce1"
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
	req.Header.Set("If-None-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentETag))
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
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
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
		testDocumentETag = "c56336c060b8a95cc1af1f1e55136ce1"
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
	req.Header.Set("If-None-Match", "03d871638b18f0b459bf8fd12a58f1d8, cc4c6d3bbf39189be874992479b60e2a, f0d0f717619b09cc081bb0c11d9b9c6b")
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

func TestGetDocumentThatIsAFolderFails(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = "Since I am innocent of this crime, sir, I find it decidedly inconvenient that the gun was never found."
		testDocument     = "/Quotes/Movies/Shawshank Redemption"
		testDocumentETag = "2cd20dc9cc76d950fdbc25acc36f724f"

		testDocThatActuallyIsAFolder = "/Quotes/Movies"
	)

	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent)))
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

	r, err := http.Get(remoteRoot + testDocThatActuallyIsAFolder)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusBadRequest))
	}
}

func TestHeadDocument(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testContent      = "Go is better than everything. In my opinion Go is even better than English."
		testMime         = "text/plain; charset=us-ascii"
		testDocument     = "/twitch.tv/ThePrimeagen"
		testDocumentETag = "d45b36d4fcc60d594236508e50cbcdd4"
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
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
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

func TestDeleteDocument(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testMime                = "text/plain; charset=utf-8"
		testCommonAncestor      = "/home/"
		testCommonAncestorETag1 = "c08117956226a4c41701b262f7fa9493"
		testCommonAncestorETag2 = "7c88d34bb2bbb2d1e792716f324a948c"

		testRootETag1 = "049d7da62b7e9bcaf21140b4142dfde0"
		testRootETag2 = "5f3e6abdf076d8105ff5d41fcd1e48c7"

		testContent1      = "Rien n'est plus dangereux qu'une idée, quand on n'a qu'une idée"
		testDocument1     = "/home/Chartier/idée"
		testDocumentETag1 = "14e6fca69c885d201b2cf5b30c929aa8"
		testDocumentDir1  = "/home/Chartier/"

		testContent2      = "Did you know that unsigned integers are faster than signed integers because your CPU doesn't have to autograph all of them as they go by?"
		testDocument2     = "/home/gamozo/unsigned"
		testDocumentETag2 = "85e25d4cf67c9d01290b1ca02e6bf60f"
	)

	// create document
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1)))
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
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag1)
		}
	}

	// create another document with a different parent
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2)))
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
		if e := r.Header.Get("ETag"); e != testDocumentETag2 {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag2)
		}
	}

	// check that documents exists
	{
		r, err := http.Head(remoteRoot + testDocument1)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		r, err = http.Head(remoteRoot + testDocument2)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}

	// verify common ancestor etag
	{
		r, err := http.Head(remoteRoot + testCommonAncestor)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testCommonAncestorETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testCommonAncestorETag1)
		}
	}

	// verify root etag
	{
		r, err := http.Head(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testRootETag1 {
			t.Errorf("got: `%s', want: `%s'", e, testRootETag1)
		}
	}

	// delete first document
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+testDocument1, nil)
		if err != nil {
			t.Error(err)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag1 {
			t.Errorf("got: `%s, want: `%s'", e, testDocumentETag1)
		}
	}

	// check that first document does not exist anymore
	{
		r, err := http.Head(remoteRoot + testDocument1)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
		}
	}

	// check that empty parent got removed as well
	{
		r, err := http.Head(remoteRoot + testDocumentDir1)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
		}
	}

	// check that common ancestor still exists
	{
		r, err := http.Head(remoteRoot + testCommonAncestor)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}

	// check that second document still exists
	{
		r, err := http.Head(remoteRoot + testDocument2)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}

	// check that common ancestor has an updated etag
	{
		r, err := http.Head(remoteRoot + testCommonAncestor)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testCommonAncestorETag2 {
			t.Errorf("got: `%s', want: `%s'", e, testCommonAncestorETag2)
		}
	}

	// check that root has an updated etag
	{
		r, err := http.Head(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testRootETag2 {
			t.Errorf("got: `%s', want: `%s'", e, testRootETag2)
		}
	}
}

func TestDeleteDocumentNotFound(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, remoteRoot+"/nonexistent/document", nil)
	if err != nil {
		t.Error(err)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
	}
}

func TestDeleteDocumentToFolder(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Did you know that unsigned integers are faster than signed integers because your CPU doesn't have to autograph all of them as they go by?"
		testDocument     = "/home/gamozo/unsigned"
		testDocumentDir  = "/home/gamozo/"
		testDocumentETag = "85e25d4cf67c9d01290b1ca02e6bf60f"
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

	req, err := http.NewRequest(http.MethodDelete, remoteRoot+testDocumentDir, nil)
	if err != nil {
		t.Error(err)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusBadRequest))
	}
}

func TestDeleteDocumentIfMatch(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Asking a question should not change the answer, and nor should asking it twice!"
		testDocument     = "/home/Henney/Asking Questions"
		testDocumentETag = "8feba501474aa57e655ef04d779196e1"
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

	// delete document, pass the correct version in if-match
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+testDocument, nil)
		if err != nil {
			t.Error(err)
		}
		// rev matches the document's current version
		req.Header.Set("If-Match", testDocumentETag)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	// check that document really got deleted
	{
		r, err := http.Head(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
		}
	}
}

func TestDeleteDocumentIfMatchFail(t *testing.T) {
	mockServer()
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Tetris is an inventory management survival horror game, from the Soviet Union in 1984."
		testDocument     = "/yt/suckerpinch/Harder Drive"
		testDocumentETag = "0daea784a3c72c074843baf751792133"
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

	// delete document, pass wrong version in if-match
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+testDocument, nil)
		if err != nil {
			t.Error(err)
		}
		// rev does NOT match the document's current version
		req.Header.Set("If-Match", "456599fd6afcb9e611b0914147dd5550")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusPreconditionFailed {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusPreconditionFailed))
		}
		if e := r.Header.Get("ETag"); e != testDocumentETag {
			t.Errorf("got: `%s', want: `%s'", e, testDocumentETag)
		}
	}

	// check that document still exists
	{
		r, err := http.Head(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}
}

func TestUnauthorizedCanReadPublicDocument(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime           = "text/plain; charset=utf-8"
		publicDocument = "/public/somewhere/somedoc.txt"
		content        = "A person who has not done one half his day's work by ten o'clock, runs a chance of leaving the other half undone."
		etag           = "02d6730d71284bfe82dd235bdefa83e1"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET public document (no authorization)
	{
		r, err := http.Get(remoteRoot + publicDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != content {
			t.Errorf("got: `%s', want: `%s'", bs, content)
		}
	}

	// HEAD public document (no authorization)
	{
		r, err := http.Head(remoteRoot + publicDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// PUT public document (no authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// DELETE public document (no authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+publicDocument, nil)
		if err != nil {
			t.Error(err)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}
}

func TestUnauthorizedCannotAccessPublicFolder(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime              = "text/plain; charset=utf-8"
		publicDocument    = "/public/Napoleon/quotes.txt"
		publicDocumentDir = "/public/Napoleon/"
		content           = "You can make a stop during the ascent, but not during the descent."
		etag              = "7ca83b7c4a8c00d6e2d8433604310841"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET document's parent (no authorization)
	{
		r, err := http.Get(remoteRoot + publicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// HEAD document's parent (no authorization)
	{
		r, err := http.Head(remoteRoot + publicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}
}

func TestUnauthorizedCannotAccessNonPublicDocument(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime              = "text/plain; charset=utf-8"
		nonPublicDocument = "/non-public/Rebel/Nikiforova.txt"
		content           = "May every state's flag burn, leaving only ashes and the black banner as its negation. Rebel, rebel until all organs of power are eliminated."
		etag              = "541857aae5f51d8ff9cfd66e1aee98b4"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET document (no authorization)
	{
		r, err := http.Get(remoteRoot + nonPublicDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// HEAD document (no authorization)
	{
		r, err := http.Head(remoteRoot + nonPublicDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// PUT document (no authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// DELETE document (no authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+nonPublicDocument, nil)
		if err != nil {
			t.Error(err)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}
}

func TestUnauthorizedCannotAccessNonPublicFolder(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime                 = "text/plain; charset=utf-8"
		nonPublicDocument    = "/non-public/Napoleon/Quotes.txt"
		nonPublicDocumentDir = "/non-public/Napoleon/"
		content              = "Death is nothing, but to live defeated and inglorious is to die daily."
		etag                 = "e160a48b29fe9dce8c6240d07b12316b"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET document's parent (no authorization)
	{
		r, err := http.Get(remoteRoot + nonPublicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}

	// HEAD document's parent (no authorization)
	{
		r, err := http.Head(remoteRoot + nonPublicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusUnauthorized {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusUnauthorized))
		}
	}
}

func TestAuthorizationRead(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		if bearer == "READER" {
			return ReadOnlyUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/Pythagoras/Quotes.txt"
		content  = "Silence is the loudest answer."
		etag     = "d2a82a88392e9c84d7a20f06bf1b5b6a"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET document (with authorization)
	{
		req, err := http.NewRequest(http.MethodGet, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != content {
			t.Errorf("got: `%s', want: `%s'", bs, content)
		}
	}

	// HEAD document (with authorization)
	{
		req, err := http.NewRequest(http.MethodHead, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// PUT document (with authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusForbidden))
		}
	}

	// DELETE document (with authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusForbidden))
		}
	}
}

func TestAuthorizationReadPublic(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		if bearer == "READER" {
			return ReadOnlyUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/public/Pythagoras/Quotes.txt"
		content  = "Learn silence. With the quiet serenity of a meditative mind, listen, absorb, transcribe, and transform."
		etag     = "d384ea5c083f09e35161e3bec30c8415"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodGet, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != content {
			t.Errorf("got: `%s', want: `%s'", bs, content)
		}
	}

	// HEAD public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodHead, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// PUT public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusForbidden))
		}
	}

	// DELETE public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusForbidden))
		}
	}
}

func TestAuthorizationReadWrite(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		if bearer == "READERWRITER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/Pythagoras/Quotes.txt"
		content  = "A man is never as big as when he is on his knees to help a child."
		etag     = "d446454ce3cb4a2032eda7417f5bb4a7"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET document (with authorization)
	{
		req, err := http.NewRequest(http.MethodGet, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != content {
			t.Errorf("got: `%s', want: `%s'", bs, content)
		}
	}

	// HEAD document (with authorization)
	{
		req, err := http.NewRequest(http.MethodHead, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// PUT document (with authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
	}

	// DELETE document (with authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}
}

func TestAuthorizationReadWritePublic(t *testing.T) {
	mockServer()
	g.UseAuthentication(func(r *http.Request, bearer string) (User, bool) {
		if bearer == "PUTTER" {
			return ReadWriteUser{}, true
		}
		if bearer == "READERWRITER" {
			return ReadWriteUser{}, true
		}
		return nil, false
	})
	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/public/Pythagoras/Quotes.txt"
		content  = "Be silent, or let thy words be worth more than silence"
		etag     = "1634d83ac7bd03d8337efab08735cd2e"
	)

	// PUT document with authorization
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content)))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
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
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// GET public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodGet, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(bs) != content {
			t.Errorf("got: `%s', want: `%s'", bs, content)
		}
	}

	// HEAD public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodHead, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
		if e := r.Header.Get("ETag"); e != etag {
			t.Errorf("got: `%s', want: `%s'", e, etag)
		}
	}

	// PUT public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room.")))
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusCreated {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusCreated))
		}
	}

	// DELETE public document (with authorization)
	{
		req, err := http.NewRequest(http.MethodDelete, remoteRoot+document, nil)
		if err != nil {
			t.Error(err)
		}
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusOK))
		}
	}
}

func TestReplay(t *testing.T) {
	t.SkipNow()

	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	opts.AllowAnyReadWrite()
	Reset()

	Time = &ReplayTime{
		Queue: []time.Time{},
	}
	UUID = &ReplayUUID{
		Queue: []UUIDResult{},
	}
	ETag = &ReplayVersion{
		Queue: []VersionResult{
			{Result: []byte("A"), Err: nil},
			{Result: []byte("B"), Err: nil},
			{Result: []byte("C"), Err: nil},
		},
	}

	log.Println(ETag.Version(nil))
	log.Println(ETag.Version(nil))
	log.Println(ETag.Version(nil))

	//mux := http.NewServeMux()
	//Register(mux)
	//ts := httptest.NewServer(mux)
	//remoteRoot := ts.URL + g.rroot
	//defer ts.Close()
}
