package rmsgo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/cvanloo/rmsgo/mock"
)

func mockServer(opts ...Option) (ts *httptest.Server, s *Server, root string) {
	hostname = "catboy"
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	s = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
		Options(opts).Combine(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts = httptest.NewServer(mux)
	return ts, s, ts.URL + g.rroot
}

// @todo: PUT test chunked transfer-encoding
// @todo: test requests with http1.1, and with switch to http2

type (
	Expectation struct {
		StatusCode int
		Headers    map[string]string
		Body       *string
	}

	ExpectedOpt func(*Expectation)
)

func Expect(opts ...ExpectedOpt) *Expectation {
	e := &Expectation{
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
		Body:       nil,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

func Status(code int) ExpectedOpt {
	return func(e *Expectation) {
		e.StatusCode = code
	}
}

func Header(k, v string) ExpectedOpt {
	return func(e *Expectation) {
		e.Headers[k] = v
	}
}

func Body(content string) ExpectedOpt {
	return func(e *Expectation) {
		e.Body = &content
	}
}

func (exp *Expectation) Validate(r *http.Response) error {
	if r.StatusCode != exp.StatusCode {
		return fmt.Errorf("got: %s, want: %s", r.Status, http.StatusText(exp.StatusCode))
	}
	for k, v := range exp.Headers {
		if hv := r.Header.Get(k); hv != v {
			return fmt.Errorf("%s got: `%s', want: `%s'", k, hv, v)
		}
	}
	if exp.Body != nil {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		if s := string(bs); len(s) != len(*exp.Body) || s != *exp.Body {
			return fmt.Errorf("got: `%s' [%dB], want: `%s' [%dB]", s, len(s), *exp.Body, len(*exp.Body))
		}
	}
	return nil
}

func TestPutDocument(t *testing.T) {
	const (
		testContent      = "The material is classified. Its composition is classified. Its use in the weapon is classified, and the process itself is classified."
		testMime         = "top/secret"
		testDocument     = "/Classified/FOGBANK.txt"
		testDocumentEtag = "60ca7ee51a4a4886d00ae2470457b206"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
	req.Header.Set("Content-Type", testMime)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusCreated),
		Header("ETag", testDocumentEtag),
	).Validate(r); err != nil {
		t.Error(err)
	}

	r, err = http.Get(remoteRoot + testDocument)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Cache-Control", "no-cache"),
		Header("Content-Length", fmt.Sprint(len(testContent))),
		Header("Content-Type", testMime),
		Header("ETag", testDocumentEtag),
		Body(testContent),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPutDocumentTwiceUpdatesIt(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)

	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{ // put first version of document
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // check that document was created by retrieving it
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("Content-Length", fmt.Sprint(len(testContent1))),
			Header("Content-Type", testMime),
			Header("ETag", testDocumentEtag1),
			Body(testContent1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // update document content
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // check that document has changed by retrieving it again
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("Content-Length", fmt.Sprint(len(testContent2))),
			Header("Content-Type", testMime),
			Header("ETag", testDocumentEtag2),
			Body(testContent2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPutDocumentIfMatchSuccessUpdatesIt(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", testDocumentEtag1) // Set If-Match header!

		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPutDocumentIfMatchFailDoesNotUpdateIt(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"
		wrongETag    = "3de26fc06d5d1e20ff96a8142cd6fabf"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2 = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", wrongETag)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusPreconditionFailed),
			Header("ETag", testDocumentEtag1), // Returns ETag of current (server) version
		).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPutDocumentIfMatchUpdateInBetween(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"
		wrongETag    = "3de26fc06d5d1e20ff96a8142cd6fabf"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"

		testContent3 = "I will travel the distance in your eyes"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{ // create document
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // @todo: check document contents
	}

	{ // update document
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", testDocumentEtag1)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // @todo: check document contents
	}

	{ // try to update document again, from a version earlier
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-Match", wrongETag)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusPreconditionFailed),
			Header("ETag", testDocumentEtag2), // Returns ETag of current (server) version
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // @todo: check document contents remain unchanged
	}
}

func TestPutDocumentIfNonMatchSuccessCreatesTheDocument(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentEtag = "33f7b41f98820961b12134677ba3f231"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{
		r, err := http.Head(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("got: %s, want: %s", r.Status, http.StatusText(http.StatusNotFound))
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-None-Match", "*")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentEtag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

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

func TestPutDocumentIfNonMatchFailDoesNotUpdateIt(t *testing.T) {
	const (
		testMime     = "application/x-subrip"
		testDocument = "/Lyrics/STARSET.txt"

		testContent1      = "I will travel the distance in your eyes Interstellar Light years from you"
		testDocumentETag1 = "33f7b41f98820961b12134677ba3f231"

		testContent2      = "I will travel the distance in your eyes Interstellar Light years from you Supernova We'll fuse when we collide Awaking in the light of all the stars aligned"
		testDocumentEtag2 = "063c77ac4aa257f9396f1b5cae956004"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-None-Match", "*")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		req.Header.Set("If-None-Match", "*")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusPreconditionFailed),
			Header("ETag", testDocumentETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{ // @todo: check document contents
	}
}

func TestPutDocumentSilentlyCreateAncestors(t *testing.T) {
	const (
		rmsContext       = "http://remotestorage.io/spec/folder-description"
		testContent      = "[...] It is written in Lisp, which is the only computer language that is beautiful." // sorry Go
		testMime         = "wise/quote"
		testDocument     = "/Quotes/Neal Stephenson.txt"
		testDocumentName = "Neal Stephenson.txt"
		testDocumentEtag = "3dc42d11db35b8354dc06c46a53c9c9d"

		testDocumentDir     = "/Quotes/"
		testDocumentDirETag = "3de26fc06d5d1e20ff96a8142cd6fabf"

		testDirListing = `{"@context":"http://remotestorage.io/spec/folder-description","items":{"Neal Stephenson.txt":{"Content-Length":83,"Content-Type":"wise/quote","ETag":"3dc42d11db35b8354dc06c46a53c9c9d","Last-Modified":"Mon, 01 Jan 0001 00:00:00 UTC"}}}
` // don't forget newline
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
	req.Header.Set("Content-Type", testMime)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusCreated),
		Header("ETag", testDocumentEtag),
	).Validate(r); err != nil {
		t.Error(err)
	}

	r, err = http.Get(remoteRoot + testDocumentDir)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Cache-Control", "no-cache"),
		Header("ETag", testDocumentDirETag),
		Body(testDirListing),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPutDocumentUpdatesAncestorETags(t *testing.T) {
	const (
		testMime = "application/x-subrip"

		testContent1      = `Run for the heavens \\ Sing to the stars \\ Love like a lover \\ Shine in the dark \\ Shout like an army \\ Sound the alarm \\ I am a burning [...] Heart`
		testDocument1     = "/Lyrics/SVRCINA.srt"
		testDocument1Name = "SVRCINA.srt"
		testDocument1ETag = "65973f0e09b0b8830949c134162d112e"

		testContent2      = `I'm attracted to the sky \\ To the sky \\ To the sky \\ Every life I learn to fly \\ Learn to fly \\ Learn to fly`
		testDocument2     = "/Lyrics/Raizer.srt"
		testDocument2Name = "Raizer.srt"
		testDocument2ETag = "19ca2805893dd1db277e23d80c2f14cd"

		testDocumentDir      = "/Lyrics/"
		testDocumentDirETag1 = "db72f39f4a2cf47b0d119946f066c894"
		testDocumentDirETag2 = "5da5bbb350f940f06f4569f8552948b0"

		testRootETag1 = "b5f199d0f2c635bf299450fe9f81da94"
		testRootETag2 = "42e40ece81f1b30afe6ae1a6de3eaffa"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	// PUT first document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocument1ETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET parent folder ETag
	{
		r, err := http.Get(remoteRoot + testDocumentDir)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("ETag", testDocumentDirETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// Get root folder ETag
	{
		r, err := http.Get(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("ETag", testRootETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT second document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocument2ETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET parent folder ETag
	{
		r, err := http.Get(remoteRoot + testDocumentDir)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("ETag", testDocumentDirETag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// Get root folder ETag
	{
		r, err := http.Get(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("ETag", testRootETag2),
		).Validate(r); err != nil {
			t.Error(err)
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
		testDocumentETag = "c1d56d2d5814cf52357a0129341402db"
		testMime         = "application/octet-stream"
	)
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		// don't set Content-Type header
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{
		r, err := http.Get(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("Cache-Control", "no-cache"),
			Header("Content-Length", fmt.Sprint(len(testContent))),
			Header("Content-Type", testMime),
			Header("ETag", testDocumentETag),
			Body(testContent),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPutDocumentAsFolderFails(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/Edward/M/D/Teach/", bytes.NewReader([]byte("HA! Liar. I have to write sentences with multiple dependent clauses in order to repair the damage of your 5 word rhetorical cluster grenade."))))
	// don't set Content-Type header
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
		t.Error(err)
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
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	// PUT first document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocument1ETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT second document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(Status(http.StatusConflict)).Validate(r); err != nil {
			t.Error(err)
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
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	// PUT first document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocument1ETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT second document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(Status(http.StatusConflict)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestGetFolder(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Get(remoteRoot + testDocumentDir)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Content-Type", "application/ld+json"),
		Header("Cache-Control", "no-cache"),
		Header("ETag", testDocumentDirETag),
		Body(responseBody),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetFolderEmpty(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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

	if err := Expect(
		Status(http.StatusOK),
		Header("Content-Type", "application/ld+json"),
		Header("Cache-Control", "no-cache"),
		Header("ETag", testDocumentDirETag),
		Body(responseBody),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetFolderNotFound(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	r, err := http.Get(remoteRoot + "/nonexistent/")
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetFolderIfNonMatchRevMatches(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+testDocumentDir, nil))
	// include revision of the folder we're about to GET
	//                                            v-- doesn't match                        v-- does match
	req.Header.Set("If-None-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentDirETag))
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNotModified)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetFolderIfNonMatchRevNoMatch(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+testDocumentDir, nil))
	// none of the revisions match our public/ folder
	req.Header.Set("If-None-Match", "03d871638b18f0b459bf8fd12a58f1d8, 3e507240501005a29cc22520bd333f79, 33f7b41f98820961b12134677ba3f231")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Content-Type", "application/ld+json"),
		Header("Cache-Control", "no-cache"),
		Header("ETag", testDocumentDirETag),
		Body(responseBody),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetFolderThatIsADocumentFails(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testContent      = "Since I am innocent of this crime, sir, I find it decidedly inconvenient that the gun was never found."
		testDocument     = "/Quotes/Movies/Shawshank Redemption"
		testDocumentETag = "2939b3af2cf45877eb61987397486084"

		testDirThatActuallyIsADocument = "/Quotes/Movies/Shawshank Redemption/"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Get(remoteRoot + testDirThatActuallyIsADocument)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestHeadFolder(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testDocumentETag = "1d8fc022c47d2abb16e03f2765575a33"
		rootETag         = "8bcad8e369ee8b5a6cfc069ca5b4d315"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/yt/rendle/citation", bytes.NewReader([]byte("In space no one can set a breakpoint."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Head(remoteRoot + "/")
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("ETag", rootETag),
		Header("Content-Length", "123"),
		Header("Content-Type", "application/ld+json"),
		Header("Cache-Control", "no-cache"),
		Body(""), // response to a head request should have an empty body
	).Validate(r); err != nil {
		t.Error(err)
	}
}

// We don't need any more HEAD folder test cases.
// The implementation logic is essentially the same: a HEAD request is also
// directed to the GetFolder handler.
// (Go's HTTP lib takes care of not including the body in the response.)

func TestGetDocument(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testContent      = "Lisp is a perfectly logical language to use." // 😤
		testMime         = "text/plain; charset=utf-8"
		testDocument     = "/everyone/would/agree/Fridman Quote"
		testDocumentETag = "1439461086c3263260ca619a30278741"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Get(remoteRoot + testDocument)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Cache-Control", "no-cache"),
		Header("Content-Length", fmt.Sprint(len(testContent))),
		Header("ETag", testDocumentETag),
		Header("Content-Type", testMime),
		Body(testContent),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	r, err := http.Get(remoteRoot + "/inexistent/document")
	if err != nil {
		t.Error(err)
	}

	if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetDocumentIfNonMatchRevMatches(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+testDocument, nil))
	// include revision of the document we're about to GET
	req.Header.Set("If-None-Match", fmt.Sprintf("03d871638b18f0b459bf8fd12a58f1d8, %s", testDocumentETag))
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNotModified)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetDocumentIfNonMatchRevNoMatch(t *testing.T) {
	ts, _, remoteRoot := mockServer()
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
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+testDocument, nil))
	// revision of our document NOT included
	req.Header.Set("If-None-Match", "03d871638b18f0b459bf8fd12a58f1d8, cc4c6d3bbf39189be874992479b60e2a, f0d0f717619b09cc081bb0c11d9b9c6b")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Cache-Control", "no-cache"),
		Header("Content-Length", fmt.Sprint(len(testContent))),
		Header("ETag", testDocumentETag),
		Header("Content-Type", testMime),
		Body(testContent),
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestGetDocumentThatIsAFolderFails(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testContent      = "Since I am innocent of this crime, sir, I find it decidedly inconvenient that the gun was never found."
		testDocument     = "/Quotes/Movies/Shawshank Redemption"
		testDocumentETag = "2939b3af2cf45877eb61987397486084"

		testDocThatActuallyIsAFolder = "/Quotes/Movies"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Get(remoteRoot + testDocThatActuallyIsAFolder)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestHeadDocument(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testContent      = "Go is better than everything. In my opinion Go is even better than English."
		testMime         = "text/plain; charset=us-ascii"
		testDocument     = "/twitch.tv/ThePrimeagen"
		testDocumentETag = "d53cc497c102d476599e7853cb3c5601"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	r, err := http.Head(remoteRoot + testDocument)
	if err != nil {
		t.Error(err)
	}

	if err := Expect(
		Status(http.StatusOK),
		Header("Content-Length", fmt.Sprint(len(testContent))),
		Header("ETag", testDocumentETag),
		Header("Content-Type", testMime),
		Body(""), // response to a head request should have an empty body
	).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestDeleteDocument(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testMime                = "text/plain; charset=utf-8"
		testCommonAncestor      = "/home/"
		testCommonAncestorETag1 = "59d054586b4316a31fcd76b434565d0e"
		testCommonAncestorETag2 = "bdf46e2f1803235eb92ac0f939101d28"

		testRootETag1 = "ed8ca43e261c8d2cf6dc7fb505859827"
		testRootETag2 = "85e25d4cf67c9d01290b1ca02e6bf60f"

		testContent1      = "Rien n'est plus dangereux qu'une idée, quand on n'a qu'une idée"
		testDocument1     = "/home/Chartier/idée"
		testDocumentETag1 = "50156bf5e641d8d33cd7929e2a2146bd"
		testDocumentDir1  = "/home/Chartier/"

		testContent2      = "Did you know that unsigned integers are faster than signed integers because your CPU doesn't have to autograph all of them as they go by?"
		testDocument2     = "/home/gamozo/unsigned"
		testDocumentETag2 = "456599fd6afcb9e611b0914147dd5550"
	)

	// create document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument1, bytes.NewReader([]byte(testContent1))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// create another document with a different parent
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument2, bytes.NewReader([]byte(testContent2))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that documents exists
	{
		r, err := http.Head(remoteRoot + testDocument1)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
			t.Error(err)
		}

		r, err = http.Head(remoteRoot + testDocument2)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// verify common ancestor etag
	{
		r, err := http.Head(remoteRoot + testCommonAncestor)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testCommonAncestorETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// verify root etag
	{
		r, err := http.Head(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testRootETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// delete first document
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+testDocument1, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testDocumentETag1),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that first document does not exist anymore
	{
		r, err := http.Head(remoteRoot + testDocument1)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that empty parent got removed as well
	{
		r, err := http.Head(remoteRoot + testDocumentDir1)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that common ancestor still exists, with an updated etag
	{
		r, err := http.Head(remoteRoot + testCommonAncestor)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testCommonAncestorETag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that second document still exists, and that it's etag remains unchanged
	{
		r, err := http.Head(remoteRoot + testDocument2)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testDocumentETag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that root has an updated etag
	{
		r, err := http.Head(remoteRoot + "/")
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testRootETag2),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteDocumentNotFound(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+"/nonexistent/document", nil))
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestDeleteFolderFails(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Did you know that unsigned integers are faster than signed integers because your CPU doesn't have to autograph all of them as they go by?"
		testDocument     = "/home/gamozo/unsigned"
		testDocumentDir  = "/home/gamozo/"
		testDocumentETag = "456599fd6afcb9e611b0914147dd5550"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+testDocumentDir, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteDocumentIfMatch(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Asking a question should not change the answer, and nor should asking it twice!"
		testDocument     = "/home/Henney/Asking Questions"
		testDocumentETag = "23527eb0b17c95022684c5b878a4c726"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// delete document, pass the correct version in if-match
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+testDocument, nil))
		// rev matches the document's current version
		req.Header.Set("If-Match", testDocumentETag)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that document really got deleted
	{
		r, err := http.Head(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteDocumentIfMatchFail(t *testing.T) {
	ts, _, remoteRoot := mockServer()
	defer ts.Close()

	const (
		testMime         = "text/plain; charset=utf-8"
		testContent      = "Tetris is an inventory management survival horror game, from the Soviet Union in 1984."
		testDocument     = "/yt/suckerpinch/Harder Drive"
		testDocumentETag = "59c0c4a04a46df78d9873e212ef3f57f"
	)

	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+testDocument, bytes.NewReader([]byte(testContent))))
		req.Header.Set("Content-Type", testMime)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// delete document, pass wrong version in if-match
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+testDocument, nil))
		// rev does NOT match the document's current version
		req.Header.Set("If-Match", "456599fd6afcb9e611b0914147dd5550")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusPreconditionFailed),
			Header("ETag", testDocumentETag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// check that document still exists
	{
		r, err := http.Head(remoteRoot + testDocument)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestUnauthorizedCanReadPublicDocument(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime           = "text/plain; charset=utf-8"
		publicDocument = "/public/somewhere/somedoc.txt"
		content        = "A person who has not done one half his day's work by ten o'clock, runs a chance of leaving the other half undone."
		etag           = "56371d17bb32d583e4131eacfdda53eb"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET public document (no authorization)
	{
		r, err := http.Get(remoteRoot + publicDocument)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD public document (no authorization)
	{
		r, err := http.Head(remoteRoot + publicDocument)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT public document (no authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE public document (no authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+publicDocument, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestUnauthorizedCannotAccessPublicFolder(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime              = "text/plain; charset=utf-8"
		publicDocument    = "/public/Napoleon/quotes.txt"
		publicDocumentDir = "/public/Napoleon/"
		content           = "You can make a stop during the ascent, but not during the descent."
		etag              = "12442797aace31d1efab9efd626c96bc"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+publicDocument, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET document's parent (no authorization)
	{
		r, err := http.Get(remoteRoot + publicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD document's parent (no authorization)
	{
		r, err := http.Head(remoteRoot + publicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestUnauthorizedCannotAccessNonPublicDocument(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime              = "text/plain; charset=utf-8"
		nonPublicDocument = "/non-public/Rebel/Nikiforova.txt"
		content           = "May every state's flag burn, leaving only ashes and the black banner as its negation. Rebel, rebel until all organs of power are eliminated."
		etag              = "a19f7c5dcf8daaba9f1411a02d6b99e1"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET document (no authorization)
	{
		r, err := http.Get(remoteRoot + nonPublicDocument)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD document (no authorization)
	{
		r, err := http.Head(remoteRoot + nonPublicDocument)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT document (no authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE document (no authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+nonPublicDocument, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestUnauthorizedCannotAccessNonPublicFolder(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime                 = "text/plain; charset=utf-8"
		nonPublicDocument    = "/non-public/Napoleon/Quotes.txt"
		nonPublicDocumentDir = "/non-public/Napoleon/"
		content              = "Death is nothing, but to live defeated and inglorious is to die daily."
		etag                 = "28c579e7f6c8906fc24b8bd0b8087013"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+nonPublicDocument, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET document's parent (no authorization)
	{
		r, err := http.Get(remoteRoot + nonPublicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD document's parent (no authorization)
	{
		r, err := http.Head(remoteRoot + nonPublicDocumentDir)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusUnauthorized)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationRead(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			if bearer == "READER" {
				return UserReadOnly{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/Pythagoras/Quotes.txt"
		content  = "Silence is the loudest answer."
		etag     = "476012c1b4644cc16a59db9315b280bc"
	)

	// PUT document with rw authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET document (with read authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD document (with read authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT document (with read authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE document (with read authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadPublicNoPerm(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return UserReadPublic{}, true
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/public/Pythagoras/Quotes.txt"
		content  = "Learn silence. With the quiet serenity of a meditative mind, listen, absorb, transcribe, and transform."
		etag     = "6681e4aec13ebde1e542809292232218"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadNonPublicNoPerm(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return UserReadPublic{}, true
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/not-public/Pythagoras/Quotes.txt"
		content  = "Learn silence. With the quiet serenity of a meditative mind, listen, absorb, transcribe, and transform."
		etag     = "6681e4aec13ebde1e542809292232218"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET non-public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD non-public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT non-public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE non-public document (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadPublicFolderNoPerm(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			return UserReadPublic{}, true
		}),
	)
	defer ts.Close()

	const (
		mime        = "text/plain; charset=utf-8"
		document    = "/public/Pythagoras/Quotes.txt"
		documentDir = "/public/Pythagoras/"
		content     = "Learn silence. With the quiet serenity of a meditative mind, listen, absorb, transcribe, and transform."
		etag        = "6681e4aec13ebde1e542809292232218"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET public folder (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+documentDir, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD public folder (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+documentDir, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT public folder (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+documentDir, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE public folder (without authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+documentDir, nil))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadPublic(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			if bearer == "READER" {
				return UserReadOnly{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/public/Pythagoras/Quotes.txt"
		content  = "Learn silence. With the quiet serenity of a meditative mind, listen, absorb, transcribe, and transform."
		etag     = "6681e4aec13ebde1e542809292232218"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadWrite(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			if bearer == "READERWRITER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/Pythagoras/Quotes.txt"
		content  = "A man is never as big as when he is on his knees to help a child."
		etag     = "d8d529c108d78c12c7356ab9f8ac3af2"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestAuthorizationReadWritePublic(t *testing.T) {
	ts, _, remoteRoot := mockServer(
		WithAuthentication(func(r *http.Request, bearer string) (User, bool) {
			if bearer == "PUTTER" {
				return UserReadWrite{}, true
			}
			if bearer == "READERWRITER" {
				return UserReadWrite{}, true
			}
			return nil, false
		}),
	)
	defer ts.Close()

	const (
		mime     = "text/plain; charset=utf-8"
		document = "/public/Pythagoras/Quotes.txt"
		content  = "Be silent, or let thy words be worth more than silence"
		etag     = "e619d8ed176ca9848f0b978a9f8712fc"
	)

	// PUT document with authorization
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte(content))))
		req.Header.Set("Content-Type", mime)
		req.Header.Set("Authorization", "Bearer PUTTER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusCreated),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// GET public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
			Body(content),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// HEAD public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodHead, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		if err := Expect(
			Status(http.StatusOK),
			Header("ETag", etag),
		).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// PUT public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+document, bytes.NewReader([]byte("Be the reason why the lights flicker when you enter a room."))))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Error(err)
		}
	}

	// DELETE public document (with authorization)
	{
		req := mustVal(http.NewRequest(http.MethodDelete, remoteRoot+document, nil))
		req.Header.Set("Authorization", "Bearer READERWRITER")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPreflightAllowAny(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNoContent)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsAllowAny(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightAllowSpecific(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowedOrigins([]string{"other.example.com", "my.example.com"}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNoContent)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsAllowSpecific(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowedOrigins([]string{"other.example.com", "my.example.com"}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightAllowCustom(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowOrigin(func(r *http.Request, origin string) bool {
			return origin == "my.example.com"
		}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNoContent)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsAllowCustom(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowOrigin(func(r *http.Request, origin string) bool {
			return origin == "my.example.com"
		}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusOK)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightAllowSpecificFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowedOrigins([]string{"other.example.com", "my.example.com"}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "wrong.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsAllowSpecificFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowedOrigins([]string{"other.example.com", "my.example.com"}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/", nil))
	req.Header.Set("Origin", "wrong.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightAllowCustomFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowOrigin(func(r *http.Request, origin string) bool {
			return origin == "my.example.com"
		}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "wrong.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsAllowCustomFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowOrigin(func(r *http.Request, origin string) bool {
			return origin == "my.example.com"
		}),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/", nil))
	req.Header.Set("Origin", "wrong.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightNotFoundFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/not/found/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestOptionsNotFoundFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/not/found", nil))
	req.Header.Set("Origin", "my.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNotFound)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightADocumentIsNotAFolderFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	// PUT a document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/hello", bytes.NewReader([]byte("Hello, World!"))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Fatal(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/hello/", nil))
		req.Header.Set("Origin", "my.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Authorization")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestOptionsADocumentIsNotAFolderFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	// PUT a document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/hello", bytes.NewReader([]byte("Hello, World!"))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Fatal(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/hello/", nil))
		req.Header.Set("Origin", "my.example.com")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestPreflightAFolderIsNotADocumentFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	// PUT a document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/hello/ignore", bytes.NewReader([]byte("Ignore"))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Fatal(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/hello", nil))
		req.Header.Set("Origin", "my.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Authorization")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

func TestOptionsAFolderIsNotADocumentFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	// PUT a document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/hello/ignore", bytes.NewReader([]byte("Ignore"))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Fatal(err)
		}
	}

	req := mustVal(http.NewRequest(http.MethodGet, remoteRoot+"/hello", nil))
	req.Header.Set("Origin", "my.example.com")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusBadRequest)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightMethodFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightHeaderFail(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Whatever")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusForbidden)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightOptions(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	opts := mustVal(Configure(rroot, sroot))
	_ = opts
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/", nil))
	req.Header.Set("Origin", "my.example.com")
	req.Header.Set("Access-Control-Request-Method", "OPTIONS")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if err := Expect(Status(http.StatusNoContent)).Validate(r); err != nil {
		t.Error(err)
	}
}

func TestPreflightDocument(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	Mock(
		WithDirectory(sroot),
	)
	_ = mustVal(Configure(rroot, sroot,
		WithAllowAnyReadWrite(),
	))
	Reset()

	mux := http.NewServeMux()
	Register(mux)
	ts := httptest.NewServer(mux)
	remoteRoot := ts.URL + g.rroot
	defer ts.Close()

	// PUT a document
	{
		req := mustVal(http.NewRequest(http.MethodPut, remoteRoot+"/hello", bytes.NewReader([]byte("Hello, World!"))))
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := Expect(Status(http.StatusCreated)).Validate(r); err != nil {
			t.Fatal(err)
		}
	}

	{
		req := mustVal(http.NewRequest(http.MethodOptions, remoteRoot+"/hello", nil))
		req.Header.Set("Origin", "my.example.com")
		req.Header.Set("Access-Control-Request-Method", "PUT")
		req.Header.Set("Access-Control-Request-Headers", "Authorization")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		if err := Expect(Status(http.StatusNoContent)).Validate(r); err != nil {
			t.Error(err)
		}
	}
}

// @todo: write tests for unhandled errors!
