package rmsgo

import (
	"bytes"
	"strings"
	"testing"
)

func TestCreateDocument(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	const (
		docPath    = "/Documents/Homework/Assignments/2023/04/Vector Geometry.md"
		docContent = "[1 3 5] × [5 6 7]" // × is one character taking up two bytes
		docMime    = "text/markdown"
	)

	n, err := CreateDocument(server, docPath, bytes.NewReader([]byte(docContent)), docMime)
	if err != nil {
		t.Error(err)
	}

	if n.name != "Vector Geometry.md" {
		t.Errorf("got: `%s', want: `Vector Geometry.md'", n.name)
	}
	if n.rname != "/Documents/Homework/Assignments/2023/04/Vector Geometry.md" {
		t.Errorf("got: `%s', want: `/Documents/Homework/Assignments/2023/04/Vector Geometry.md'", n.rname)
	}
	if !strings.HasPrefix(n.sname, server.Sroot) {
		t.Errorf("got: `%s', want a path starting with: `%s'", n.sname, server.Sroot)
	}
	if n.isFolder {
		t.Error("got: isFolder == true, want: isFolder == false")
	}
	if n.length != 18 {
		t.Errorf("got: `%d', want: 18", n.length)
	}
	if n.mime != "text/markdown" {
		t.Errorf("got: `%s', want: text/markdown", n.mime)
	}

	checks := []struct {
		name, rname string
	}{
		{
			name:  "Documents/",
			rname: "/Documents/",
		},
		{
			name:  "Homework/",
			rname: "/Documents/Homework/",
		},
		{
			name:  "Assignments/",
			rname: "/Documents/Homework/Assignments/",
		},
		{
			name:  "2023/",
			rname: "/Documents/Homework/Assignments/2023/",
		},
		{
			name:  "04/",
			rname: "/Documents/Homework/Assignments/2023/04/",
		},
	}

	t.Logf("\n%s", root)

	p := root
	for _, c := range checks {
		n, err := Node(c.rname)
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
			t.Errorf("got: `%s', want: inode/directory", n.mime)
		}
		if !n.isFolder {
			t.Error("got: isFolder == false, want: isFolder == true")
		}
		if len(n.children) != 1 {
			t.Errorf("%s has `%d' children, want: 1", n.name, len(n.children))
		}
		if n.parent != p {
			t.Errorf("wrong parent; got: `%p', want: `%p'", n.parent, p)
		}
		p = n
	}

}

func TestCreateDocuments(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	_, err := CreateDocument(server, "/code/hello.go", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")), "text/plain")
	if err != nil {
		t.Error(err)
	}
	_, err = CreateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n%s", root)

	if l := len(root.children); l != 1 {
		t.Errorf("got: `%d', want: 1", l)
	}

	n, err := Node("/code/")
	if err != nil {
		t.Error(err)
	}
	if l := len(n.children); l != 2 {
		t.Errorf("got: `%d', want: 2", l)
	}
}

func TestUpdateDocument(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	const path = "/FunFacts/Part1.txt"

	n1, err := CreateDocument(server, path, bytes.NewReader([]byte("Elephants can't jump.")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := UpdateDocument(server, path, bytes.NewReader([]byte("Honey never spoils.")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 {
		t.Error("expected nodes to be the same")
	}
}

func TestNode(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	const path = "/FunFacts/Part2.txt"

	n1, err := CreateDocument(server, path, bytes.NewReader([]byte("The first person convicted of speeding was going eight mph.")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Node(path)
	if err != nil {
		t.Error(err)
	}
	if n1 != n2 {
		t.Error("expected nodes to be the same")
	}
}

func TestRemoveDocument(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	const path = "/FunFacts/Part3.txt"

	n1, err := CreateDocument(server, path, bytes.NewReader([]byte("The severed head of a sea slug can grow a whole new body.")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Node(path)
	if err != nil {
		t.Error(err)
	}

	n3, err := RemoveDocument(server, path)
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 || n2 != n3 {
		t.Error("expected nodes to be the same")
	}

	_, err = Node(path)
	if err != ErrNotFound {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotFound)
	}
}

func TestETagUpdatedWhenDocumentAdded(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	_, err := CreateDocument(server, "/code/hello.go", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	codeFolder, err := Node("/code/")
	if err != nil {
		t.Error(err)
	}

	v1, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v1: %x", v1)

	_, err = CreateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	if codeFolder.etagValid != false {
		t.Error("expected etag to have been invalidated")
	}
	v2, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v2: %x", v2)

	if string(v1) == string(v2) {
		t.Error("expected version to have changed")
	}
}

func TestETagUpdatedWhenDocumentRemoved(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	_, err := CreateDocument(server, "/code/hello.go", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")), "text/plain")
	if err != nil {
		t.Error(err)
	}
	_, err = CreateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	codeFolder, err := Node("/code/")
	if err != nil {
		t.Error(err)
	}

	v1, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v1: %x", v1)

	_, err = RemoveDocument(server, "/code/hello.go")
	if err != nil {
		t.Error(err)
	}

	if codeFolder.etagValid != false {
		t.Error("expected version to have been invalidated")
	}
	v2, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v2: %x", v2)

	if string(v1) == string(v2) {
		t.Error("expected version to have changed")
	}
}

func TestETagUpdatedWhenDocumentUpdated(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	_, err := CreateDocument(server, "/code/hello.go", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")), "text/plain")
	if err != nil {
		t.Error(err)
	}
	errorDoc, err := CreateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	dv1, err := errorDoc.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("document etag v1: %x", dv1)

	codeFolder, err := Node("/code/")
	if err != nil {
		t.Error(err)
	}

	fv1, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("folder etag v1: %x", fv1)

	_, err = UpdateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrExistentialCrisis = errors.New(\"why?\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	if errorDoc.etagValid != false {
		t.Error("expected document version to have been invalidated")
	}
	dv2, err := errorDoc.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("document etag v2: %x", dv2)

	if codeFolder.etagValid != false {
		t.Error("expected folder version to have been invalidated")
	}
	fv2, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("folder etag v2: %x", fv2)

	if string(fv1) == string(fv2) {
		t.Error("expected folder version to have changed")
	}
	if string(dv1) == string(dv2) {
		t.Error("expected document version to have changed")
	}
}

func TestETagNotAffected(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	ResetStorage(server) // @todo: handle error

	_, err := CreateDocument(server, "/code/hello.go", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")), "text/plain")
	if err != nil {
		t.Error(err)
	}
	_, err = CreateDocument(server, "/code/error.go", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")), "text/plain")
	if err != nil {
		t.Error(err)
	}

	codeFolder, err := Node("/code/")
	if err != nil {
		t.Error(err)
	}

	v1, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("folder etag v1: %x", v1)

	rv1, err := root.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("root etag v1: %x", rv1)

	// 可愛い is 3 characters together taking up 9 bytes
	f, err := CreateDocument(server, "/Pictures/Kittens.png", bytes.NewReader([]byte("可愛い")), "image/png")
	if err != nil {
		t.Error(err)
	}
	if f.length != 9 {
		t.Errorf("got: `%d', want: 9", f.length)
	}

	v2, err := codeFolder.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("folder etag v2: %x", v2)

	if codeFolder.etagValid != true {
		t.Error("expected etag to still be valid")
	}
	if string(v1) != string(v2) {
		t.Error("expected code folder etag to not have changed")
	}

	if root.etagValid != false {
		t.Error("expected version to have been invalidated")
	}
	rv2, err := root.ETag()
	if err != nil {
		t.Error(err)
	}
	t.Logf("root etag v2: %x", rv2)

	if string(rv1) == string(rv2) {
		t.Error("expected root etag to have changed")
	}
}
