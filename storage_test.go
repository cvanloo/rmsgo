package rmsgo

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	. "github.com/cvanloo/rmsgo.git/mock"
	"golang.org/x/exp/maps"
)

var genpath = func() string {
	return filepath.Join(sroot, mustVal(UUID()).String())
}

func TestAddDocument(t *testing.T) {
	Reset()

	const (
		docPath    = "/Documents/Homework/Assignments/2023/04/Vector Geometry.md"
		docContent = "[1 3 5] × [5 6 7]" // × is one character taking up two bytes
		docMime    = "text/markdown"
		docSize    = len(docContent)
		docSPath   = "/some/spath/doc"
	)

	n, err := AddDocument(docPath, docSPath, int64(docSize), docMime)
	if err != nil {
		t.Error(err)
	}

	if n.name != "Vector Geometry.md" {
		t.Errorf("got: `%s', want: `Vector Geometry.md'", n.name)
	}
	if n.rname != "/Documents/Homework/Assignments/2023/04/Vector Geometry.md" {
		t.Errorf("got: `%s', want: `/Documents/Homework/Assignments/2023/04/Vector Geometry.md'", n.rname)
	}
	if n.sname != docSPath {
		t.Errorf("got: `%s', want: `%s'", n.sname, docSPath)
	}
	if n.isFolder {
		t.Error("got: isFolder == true, want: isFolder == false")
	}
	if n.length != 18 {
		t.Errorf("got: %d, want: 18", n.length)
	}
	if n.mime != "text/markdown" {
		t.Errorf("got: `%s', want: text/markdown", n.mime)
	}

	// children must be listed after their parents… [#1]
	checks := []struct {
		name, rname string
	}{
		{
			name:  "Documents/",
			rname: "/Documents",
		},
		{
			name:  "Homework/",
			rname: "/Documents/Homework",
		},
		{
			name:  "Assignments/",
			rname: "/Documents/Homework/Assignments",
		},
		{
			name:  "2023/",
			rname: "/Documents/Homework/Assignments/2023",
		},
		{
			name:  "04/",
			rname: "/Documents/Homework/Assignments/2023/04",
		},
	}

	p := root
	for _, c := range checks {
		n, err := Retrieve(c.rname)
		if err != nil {
			t.Error(err)
		}

		if n.name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.name, c.name)
		}
		if n.rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.rname, c.rname)
		}
		if n.mime != "inode/directory" {
			t.Errorf("got: `%s', want: `inode/directory'", n.mime)
		}
		if !n.isFolder {
			t.Error("got: isFolder == false, want: isFolder == true")
		}
		if len(n.children) != 1 {
			t.Errorf("got: %d, want: 1", len(n.children))
		}
		// [#1] …for these checks to work:
		if n.parent != p {
			t.Errorf("got: `%s', want: `%s'", n.parent.rname, p.rname)
		}
		if v := maps.Values(p.children)[0]; v != n {
			t.Errorf("got: `%s', want: `%s'", v.rname, n.rname)
		}
		p = n
	}

}

func TestAddDocumentMultipleInSameFolder(t *testing.T) {
	Reset()

	_, err := AddDocument("/code/hello.go", _s(), _i64(), "text/plain")
	if err != nil {
		t.Error(err)
	}
	_, err = AddDocument("/code/error.go", _s(), _i64(), "text/plain")
	if err != nil {
		t.Error(err)
	}

	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	if codeFolder.parent != root {
		t.Errorf("got: `%s', want: `%s'", codeFolder.parent.rname, root.rname)
	}
	if l := len(root.children); l != 1 {
		t.Errorf("got: `%d', want: 1", l)
	}
	if fc := maps.Values(root.children)[0]; fc != codeFolder {
		t.Errorf("got: `%s', want: `%s'", fc, codeFolder)
	}
	if l := len(codeFolder.children); l != 2 {
		t.Errorf("got: `%d', want: 2", l)
	}
}

func TestUpdateDocument(t *testing.T) {
	Reset()

	const path = "/FunFacts/Part1.txt"

	n, err := AddDocument(path, _s(), 14, "text/plain")
	if err != nil {
		t.Error(err)
	}
	if n.mime != "text/plain" {
		t.Errorf("got: `%s', want: `text/plain'", n.mime)
	}
	if n.length != 14 {
		t.Errorf("got: %d, want: 14", n.length)
	}
	//mod1 := *n.lastMod

	UpdateDocument(n, "image/png", 5)

	if n.mime != "image/png" {
		t.Errorf("got: `%s', want: `image/png'", n.mime)
	}
	if n.length != 5 {
		t.Errorf("got: %d, want: 5", n.length)
	}
	//mod2 := *n.lastMod
	//if mod1.Compare(mod2) == 0 {
	//	t.Error("mod time did not change")
	//}
}

func TestRetrieve(t *testing.T) {
	Reset()

	const path = "/FunFacts/Part2.txt"

	n1, err := AddDocument(path, _s(), _i64(), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Retrieve(path)
	if err != nil {
		t.Error(err)
	}
	if n1 != n2 {
		t.Error("nodes are not the same")
	}
}

func TestRemoveDocument(t *testing.T) {
	Reset()

	const path = `/Did/You/Know/That/Unix Paths May Contain Non﹣Breaking Spaces?۪txt`

	n1, err := AddDocument(path, _s(), _i64(), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Retrieve(path)
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 {
		t.Error("nodes are not the same")
	}

	RemoveDocument(n2)

	_, err = Retrieve(path)
	if err != ErrNotExist {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotExist)
	}
}

func TestUpdateDocumentUpdatesETag(t *testing.T) {
	mockServer()

	const (
		// 可愛い is 3 characters together taking up 9 bytes
		testContent = "可愛い"
	)

	var kittenDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent), 0666)
		if err != nil {
			t.Error(err)
		}
		kittenDoc, err = AddDocument("/Pictures/Kittens.png", sname, int64(len(testContent)), "text/plain")
		if err != nil {
			t.Error(err)
		}
		if kittenDoc.length != 9 {
			t.Errorf("got: %d, want: 9", kittenDoc.length)
		}
	}

	v1, err := kittenDoc.Version()
	if err != nil {
		t.Error(err)
	}

	UpdateDocument(kittenDoc, "image/avif", 9)

	if kittenDoc.Valid() {
		t.Error("etag is still valid")
	}
	v2, err := kittenDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(v2) == string(v1) {
		t.Error("etag has not changed")
	}
}

func TestAddDocumentUpdatesParentETag(t *testing.T) {
	mockServer()

	const (
		testContent1 = "func hello() string {\n\treturn \"Hello, World\"\n}"
		testContent2 = "var ErrYouSuck = errors.New(\"YOU SUCK!!\")"
	)

	var helloDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		helloDoc, err = AddDocument("/code/hello.go", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	dv1, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	fv1, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/code/error.go", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	// hello.go document etag should remain unchanged
	if !helloDoc.Valid() {
		t.Error("document etag is not valid")
	}
	dv2, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dv1) != string(dv2) {
		t.Error("document etag changed")
	}
	// parent etag should change because another document was added to its children
	if codeFolder.Valid() {
		t.Error("parent etag is still valid")
	}
	fv2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	if string(fv1) == string(fv2) {
		t.Error("parent etag has not changed")
	}
}

func TestRevomeDocumentUpdatesParentETag(t *testing.T) {
	mockServer()

	const (
		testContent1 = "func hello() string {\n\treturn \"Hello, World\"\n}"
		testContent2 = "var ErrYouSuck = errors.New(\"YOU SUCK!!\")"
	)

	var helloDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		helloDoc, err = AddDocument("/code/hello.go", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/code/error.go", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	dv1, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	v1, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}

	RemoveDocument(helloDoc)
	err = FS.Remove(helloDoc.sname)
	if err != nil {
		t.Error(err)
	}

	// hello.go document etag should remain unchanged
	if !helloDoc.Valid() {
		t.Error("document etag is not valid")
	}
	dv2, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dv1) != string(dv2) {
		t.Error("document etag changed")
	}
	// parent etag should change because another document was deleted from its children
	if codeFolder.Valid() != false {
		t.Error("etag is still valid")
	}
	v2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	if string(v1) == string(v2) {
		t.Error("etag has not changed")
	}
}

func TestUpdateDocumentUpdatesParentETag(t *testing.T) {
	mockServer()

	const (
		testContent1 = "func hello() string {\n\treturn \"Hello, World\"\n}"
		testContent2 = "var ErrYouSuck = errors.New(\"YOU SUCK!!\")"
		newContents  = "var ErrExistentialCrisis = errors.New(\"why?\")"
	)

	var helloDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		helloDoc, err = AddDocument("/code/hello.go", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	var errorDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		errorDoc, err = AddDocument("/code/error.go", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	dhv1, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	dev1, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	fv1, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}

	err = FS.WriteFile(errorDoc.sname, []byte(newContents), 0666)
	if err != nil {
		t.Error(err)
	}
	UpdateDocument(errorDoc, "text/plain", int64(len(newContents)))

	// hello.go document etag should remain unchanged
	if !helloDoc.Valid() {
		t.Error("hello document etag is not valid")
	}
	dhv2, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dhv1) != string(dhv2) {
		t.Error("hello document etag changed")
	}
	// error.go document etag should change because the file content changed
	if errorDoc.Valid() {
		t.Error("error document etag is still valid")
	}
	dev2, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dev1) == string(dev2) {
		t.Error("error document etag has not changed")
	}
	// parent etag should change because one of its children changed
	if codeFolder.Valid() {
		t.Error("parent etag is still valid")
	}
	fv2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	if string(fv1) == string(fv2) {
		t.Error("parent etag has not changed")
	}
}

func TestAddDocumentDoesNotUpdateETagOfOtherFolders(t *testing.T) {
	mockServer()

	const (
		testContent1 = "func hello() string {\n\treturn \"Hello, World\"\n}"
		testContent2 = "var ErrYouSuck = errors.New(\"YOU SUCK!!\")"

		// 可愛い is 3 characters together taking up 9 bytes
		testContent3 = "可愛い"
	)

	var helloDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		helloDoc, err = AddDocument("/code/hello.go", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	var errorDoc *node
	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		errorDoc, err = AddDocument("/code/error.go", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	dhv1, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	dev1, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	fv1, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	rv1, err := root.Version()
	if err != nil {
		t.Error(err)
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent3), 0666)
		if err != nil {
			t.Error(err)
		}
		f, err := AddDocument("/Pictures/Kittens.png", sname, int64(len(testContent3)), "text/plain")
		if err != nil {
			t.Error(err)
		}
		if f.length != 9 {
			t.Errorf("got: %d, want: 9", f.length)
		}
	}

	// hello.go document etag should remain unchanged
	if !helloDoc.Valid() {
		t.Error("hello document etag is not valid")
	}
	dhv2, err := helloDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dhv1) != string(dhv2) {
		t.Error("hello document etag changed")
	}
	// error.go document etag should remain unchanged
	if !errorDoc.Valid() {
		t.Error("error document etag is not valid")
	}
	dev2, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	if string(dev1) != string(dev2) {
		t.Error("error document etag changed")
	}
	// code folder etag should remain unchanged
	if !codeFolder.Valid() {
		t.Error("code folder etag is not valid")
	}
	fv2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	if string(fv1) != string(fv2) {
		t.Error("parent etag has changed")
	}
	// root folder should change
	if root.Valid() {
		t.Error("root etag is still valid")
	}
	rv2, err := root.Version()
	if err != nil {
		t.Error(err)
	}
	if string(rv1) == string(rv2) {
		t.Error("expected root etag to have changed")
	}
}

const persistText = `<Root>
	<Nodes IsFolder="true">
		<Name>Documents/</Name>
		<Rname>/Documents</Rname>
		<ETag>86f32f54096e02778610b22d1d6c56db</ETag>
		<Mime>inode/directory</Mime>
		<ParentRName>/</ParentRName>
	</Nodes>
	<Nodes IsFolder="false">
		<Name>hello.txt</Name>
		<Rname>/Documents/hello.txt</Rname>
		<Sname>/tmp/rms/storage/32000000-0000-0000-0000-000000000000</Sname>
		<ETag>ea724748ce53d55deb465a6d045fd160</ETag>
		<Mime>text/plain</Mime>
		<Length>13</Length>
		<LastMod>0001-01-01T00:00:00Z</LastMod>
		<ParentRName>/Documents</ParentRName>
	</Nodes>
	<Nodes IsFolder="false">
		<Name>test.txt</Name>
		<Rname>/Documents/test.txt</Rname>
		<Sname>/tmp/rms/storage/31000000-0000-0000-0000-000000000000</Sname>
		<ETag>10b3bf730d787feceec1d534a876dc5f</ETag>
		<Mime>text/plain</Mime>
		<Length>20</Length>
		<LastMod>0001-01-01T00:00:00Z</LastMod>
		<ParentRName>/Documents</ParentRName>
	</Nodes>
</Root>`

func TestPersist(t *testing.T) {
	mockServer()

	const (
		testContent1 = "Whole life's a test."
		testContent2 = "Hello, World!"
	)

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/Documents/test.txt", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/Documents/hello.txt", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	bs := &bytes.Buffer{}
	err := Persist(bs)
	if err != nil {
		t.Error(err)
	}
	if persistText != bs.String() {
		t.Errorf("got:\n%s\nwant:\n%s\n", bs, persistText)
	}
}

func TestLoad(t *testing.T) {
	mockServer()

	const (
		testContent1 = "Whole life's a test."
		testContent2 = "Hello, World!"
	)

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/Documents/test.txt", sname, int64(len(testContent1)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		if err != nil {
			t.Error(err)
		}
		_, err = AddDocument("/Documents/hello.txt", sname, int64(len(testContent2)), "text/plain")
		if err != nil {
			t.Error(err)
		}
	}

	bs := &bytes.Buffer{}
	err := Persist(bs)
	if err != nil {
		t.Error(err)
	}
	Reset()
	err = Load(bs)
	if err != nil {
		t.Error(err)
	}
	if len(files) != 4 {
		t.Errorf("got: %d, want: 4", len(files))
	}

	// ensure root is set correctly
	r, err := Retrieve("/")
	if err != nil {
		t.Error(err)
	}
	if r != root {
		t.Errorf("got: `%v', want: `%v'", r, root)
	}
	if len(r.children) != 1 {
		t.Errorf("got: %d, want: 1", len(r.children))
	}

	// Children must be listed immediately after their parents for parent/child
	// check to work correctly. [#child_after]
	checks := []struct {
		name, rname string
		mime        string
		nchildren   int
		isDir       bool
	}{
		{
			name:      "Documents/",
			rname:     "/Documents",
			mime:      "inode/directory",
			nchildren: 2,
			isDir:     true,
		},
		{
			name:  "test.txt",
			rname: "/Documents/test.txt",
			mime:  "text/plain",
		},
		{
			name:  "hello.txt",
			rname: "/Documents/hello.txt",
			mime:  "text/plain",
		},
	}

	t.Logf("Root Listing:\n%s", root)

	p := root
	for _, c := range checks {
		n, err := Retrieve(c.rname)
		if err != nil {
			t.Error(err)
		}

		if n.name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.name, c.name)
		}
		if n.rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.rname, c.rname)
		}
		if n.mime != c.mime {
			t.Errorf("got: `%s', want: `%s'", n.mime, c.mime)
		}
		if n.isFolder != c.isDir {
			t.Errorf("got: isFolder == %t, want: isFolder == %t", n.isFolder, c.isDir)
		}
		if len(n.children) != c.nchildren {
			t.Errorf("%s has %d children, want: %d", n.name, len(n.children), c.nchildren)
		}
		hasAsChild := false
		for _, v := range p.children {
			if v == n {
				hasAsChild = true
				break
			}
		}
		if !hasAsChild {
			t.Errorf("parent `%s' is missing `%s' as a child", p.name, n.name)
		}
		if n.parent != p {
			t.Errorf("wrong parent for `%s'; got: `%s', want: `%s'", n.name, n.parent.name, p.name)
		}
		if c.isDir {
			p = n // [#child_after]
		}
	}
}

func TestMigrate(t *testing.T) {
	const (
		rroot = "/storage/"
		sroot = "/tmp/rms/storage/"
	)
	must(Setup(rroot, sroot))
	Mock(
		WithDirectory(sroot),
		WithFile("/somewhere/Documents/hello.txt", []byte("Hello, World!")),
		WithFile("/somewhere/Documents/test.txt", []byte("Whole life's a test.")),
	)

	Reset()
	errs := Migrate("/somewhere/")
	for _, err := range errs {
		t.Error(err)
	}

	t.Log(root)

	// Children must be listed immediately after their parents for parent/child
	// check to work correctly. [#child_after]
	checks := []struct {
		name, rname string
		mime        string
		nchildren   int
		isDir       bool
	}{
		{
			name:      "Documents/",
			rname:     "/Documents",
			mime:      "inode/directory",
			nchildren: 2,
			isDir:     true,
		},
		{
			name:  "test.txt",
			rname: "/Documents/test.txt",
			mime:  "application/octet-stream",
		},
		{
			name:  "hello.txt",
			rname: "/Documents/hello.txt",
			mime:  "application/octet-stream",
		},
	}

	p := root
	for _, c := range checks {
		n, err := Retrieve(c.rname)
		if err != nil {
			t.Error(err)
		}

		if n.name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.name, c.name)
		}
		if n.rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.rname, c.rname)
		}
		if n.mime != c.mime {
			t.Errorf("got: `%s', want: `%s'", n.mime, c.mime)
		}
		if n.isFolder != c.isDir {
			t.Errorf("got: isFolder == %t, want: isFolder == %t", n.isFolder, c.isDir)
		}
		if len(n.children) != c.nchildren {
			t.Errorf("%s has %d children, want: %d", n.name, len(n.children), c.nchildren)
		}
		hasAsChild := false
		for _, v := range p.children {
			if v == n {
				hasAsChild = true
				break
			}
		}
		if !hasAsChild {
			t.Errorf("parent `%s' is missing `%s' as a child", p.name, n.name)
		}
		if n.parent != p {
			t.Errorf("wrong parent for `%s'; got: `%s', want: `%s'", n.name, n.parent.name, p.name)
		}
		if c.isDir {
			p = n // [#child_after]
		}
	}
}

func ExamplePersist() {
	mockServer()

	panicIf := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	const (
		testContent1 = "Whole life's a test."
		testContent2 = "Hello, World!"
	)

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent1), 0666)
		panicIf(err)
		_, err = AddDocument("/Documents/test.txt", sname, int64(len(testContent1)), "text/plain")
		panicIf(err)
	}

	{
		sname := genpath()
		err := FS.WriteFile(sname, []byte(testContent2), 0666)
		panicIf(err)
		_, err = AddDocument("/Documents/hello.txt", sname, int64(len(testContent2)), "text/plain")
		panicIf(err)
	}

	fd, err := FS.Create(sroot + "/marshalled.xml")
	panicIf(err)
	defer fd.Close()
	err = Persist(fd)
	panicIf(err)

	fd.Seek(0, io.SeekStart)
	bs, err := io.ReadAll(fd)
	panicIf(err)
	fmt.Printf("XML follows:\n%s\n", bs)

	fd.Seek(0, io.SeekStart)
	Reset()
	err = Load(fd)
	panicIf(err)
	fmt.Printf("Storage listing follows:\n%s", root)

	// Output: XML follows:
	// <Root>
	// 	<Nodes IsFolder="true">
	// 		<Name>Documents/</Name>
	// 		<Rname>/Documents</Rname>
	// 		<ETag>86f32f54096e02778610b22d1d6c56db</ETag>
	// 		<Mime>inode/directory</Mime>
	// 		<ParentRName>/</ParentRName>
	// 	</Nodes>
	// 	<Nodes IsFolder="false">
	// 		<Name>hello.txt</Name>
	// 		<Rname>/Documents/hello.txt</Rname>
	// 		<Sname>/tmp/rms/storage/32000000-0000-0000-0000-000000000000</Sname>
	// 		<ETag>ea724748ce53d55deb465a6d045fd160</ETag>
	// 		<Mime>text/plain</Mime>
	// 		<Length>13</Length>
	// 		<LastMod>0001-01-01T00:00:00Z</LastMod>
	// 		<ParentRName>/Documents</ParentRName>
	// 	</Nodes>
	// 	<Nodes IsFolder="false">
	// 		<Name>test.txt</Name>
	// 		<Rname>/Documents/test.txt</Rname>
	// 		<Sname>/tmp/rms/storage/31000000-0000-0000-0000-000000000000</Sname>
	// 		<ETag>10b3bf730d787feceec1d534a876dc5f</ETag>
	// 		<Mime>text/plain</Mime>
	// 		<Length>20</Length>
	// 		<LastMod>0001-01-01T00:00:00Z</LastMod>
	// 		<ParentRName>/Documents</ParentRName>
	// 	</Nodes>
	// </Root>
	// Storage listing follows:
	// {F} / [/] [6330643033303764]
	//   {F} Documents/ [/Documents] [3836663332663534]
	//     {D} hello.txt (text/plain, 13) [/Documents/hello.txt -> /tmp/rms/storage/32000000-0000-0000-0000-000000000000] [6561373234373438]
	//     {D} test.txt (text/plain, 20) [/Documents/test.txt -> /tmp/rms/storage/31000000-0000-0000-0000-000000000000] [3130623362663733]
}
