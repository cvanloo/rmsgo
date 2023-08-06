package rmsgo

import (
	"strings"
	"testing"
)

func TestCreateDocument(t *testing.T) {
	mfs = CreateMockFS()
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}
	store := NewStorage()

	const (
		docPath    = "/Documents/Homework/Assignments/2023/04/Vector Geometry.md"
		docContent = "[1 3 5] × [5 6 7]"
		docMime    = "text/markdown"
	)

	n, err := store.CreateDocument(server, docPath, []byte(docContent), docMime)
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
			name:  "Documents",
			rname: "/Documents/",
		},
		{
			name:  "Homework",
			rname: "/Documents/Homework/",
		},
		{
			name:  "Assignments",
			rname: "/Documents/Homework/Assignments/",
		},
		{
			name:  "2023",
			rname: "/Documents/Homework/Assignments/2023/",
		},
		{
			name:  "04",
			rname: "/Documents/Homework/Assignments/2023/04/",
		},
	}

	t.Logf("\n%s", store)

	p := store.Root()
	for _, c := range checks {
		n, err := store.Node(server, c.rname)
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
	store := NewStorage()

	_, err := store.CreateDocument(server, "/code/hello.go", []byte("func hello() string {\n\treturn \"Hello, World\"\n}"), "text/plain")
	if err != nil {
		t.Error(err)
	}
	_, err = store.CreateDocument(server, "/code/error.go", []byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")"), "text/plain")
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n%s", store)

	if l := len(store.Root().children); l != 1 {
		t.Errorf("got: `%d', want: 2", l)
	}

	n, err := store.Node(server, "/code/")
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
	store := NewStorage()

	const path = "/FunFacts/Part1.txt"

	n1, err := store.CreateDocument(server, path, []byte("Elephants can't jump."), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := store.UpdateDocument(server, path, []byte("Honey never spoils."), "text/plain")
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
	store := NewStorage()

	const path = "/FunFacts/Part2.txt"

	n1, err := store.CreateDocument(server, path, []byte("The first person convicted of speeding was going eight mph."), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := store.Node(server, path)
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
	store := NewStorage()

	const path = "/FunFacts/Part3.txt"

	n1, err := store.CreateDocument(server, path, []byte("The severed head of a sea slug can grow a whole new body."), "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := store.Node(server, path)
	if err != nil {
		t.Error(err)
	}

	n3, err := store.RemoveDocument(server, path)
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 || n2 != n3 {
		t.Error("expected nodes to be the same")
	}

	_, err = store.Node(server, path)
	if err == nil {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotFound)
	}
}
