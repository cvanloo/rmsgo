package rmsgo

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	. "github.com/cvanloo/rmsgo.git/mock"
	"golang.org/x/exp/maps"
)

func TestCreateDocument(t *testing.T) {
	_, server := mockServer()

	const (
		docPath    = "/Documents/Homework/Assignments/2023/04/Vector Geometry.md"
		docContent = "[1 3 5] × [5 6 7]" // × is one character taking up two bytes
		docMime    = "text/markdown"
	)

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte(docContent)))
	n, err := AddDocument(docPath, sname, fsize, docMime)
	if err != nil {
		t.Error(err)
	}

	if n.Name != "Vector Geometry.md" {
		t.Errorf("got: `%s', want: `Vector Geometry.md'", n.Name)
	}
	if n.Rname != "/Documents/Homework/Assignments/2023/04/Vector Geometry.md" {
		t.Errorf("got: `%s', want: `/Documents/Homework/Assignments/2023/04/Vector Geometry.md'", n.Rname)
	}
	if !strings.HasPrefix(n.Sname, server.sroot) {
		t.Errorf("got: `%s', want a path starting with: `%s'", n.Sname, server.sroot)
	}
	if n.IsFolder {
		t.Error("got: isFolder == true, want: isFolder == false")
	}
	if n.Length != 18 {
		t.Errorf("got: `%d', want: 18", n.Length)
	}
	if n.Mime != "text/markdown" {
		t.Errorf("got: `%s', want: text/markdown", n.Mime)
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
		n, err := Retrieve(c.rname)
		if err != nil {
			t.Error(err)
		}

		if n.Name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.Name, c.name)
		}
		if n.Rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.Rname, c.rname)
		}
		if n.Mime != "inode/directory" {
			t.Errorf("got: `%s', want: inode/directory", n.Mime)
		}
		if !n.IsFolder {
			t.Error("got: isFolder == false, want: isFolder == true")
		}
		if len(n.children) != 1 {
			t.Errorf("%s has `%d' children, want: 1", n.Name, len(n.children))
		}
		if v := maps.Values(p.children)[0]; v != n {
			t.Errorf("parent `%s' has wrong child; got: `%s', want: `%s'", p.Name, v.Name, n.Name)
		}
		if n.parent != p {
			t.Errorf("wrong parent for `%s'; got: `%s', want: `%s'", n.Name, n.parent.Name, p.Name)
		}
		p = n
	}

}

func TestCreateDocuments(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World!")))
	_, err = AddDocument("/code/hello.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}
	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")))
	_, err = AddDocument("/code/error.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	t.Logf("\n%s", root)

	if l := len(root.children); l != 1 {
		t.Errorf("got: `%d', want: 1", l)
	}

	n, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}
	if l := len(n.children); l != 2 {
		t.Errorf("got: `%d', want: 2", l)
	}
}

func TestUpdateDocument(t *testing.T) {
	_, server := mockServer()

	const path = "/FunFacts/Part1.txt"

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("Elephants can't jump.")))
	n1, err := AddDocument(path, sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	sname, fsize, _, err = WriteFile(server, sname, bytes.NewReader([]byte("Honey never spoils.")))
	n2, err := UpdateDocument(path, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 {
		t.Error("expected nodes to be the same")
	}
}

func TestRetrieve(t *testing.T) {
	_, server := mockServer()

	const path = "/FunFacts/Part2.txt"

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("The first person convicted of speeding was going eight mph.")))
	n1, err := AddDocument(path, sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Retrieve(path)
	if err != nil {
		t.Error(err)
	}
	if n1 != n2 {
		t.Error("expected nodes to be the same")
	}
}

func TestRemoveDocument(t *testing.T) {
	_, server := mockServer()

	const path = "/FunFacts/Part3.txt"

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("The severed head of a sea slug can grow a whole new body.")))
	n1, err := AddDocument(path, sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	n2, err := Retrieve(path)
	if err != nil {
		t.Error(err)
	}

	n3, err := RemoveDocument(path)
	if err != nil {
		t.Error(err)
	}

	if n1 != n2 || n2 != n3 {
		t.Error("expected nodes to be the same")
	}

	_, err = Retrieve(path)
	if err != ErrNotFound {
		t.Errorf("got: `%v', want: `%v'", err, ErrNotFound)
	}
}

func TestETagUpdatedWhenDocumentAdded(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")))
	_, err = AddDocument("/code/hello.go", sname, fsize, "text/plain")
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
	t.Logf("etag v1: %x", v1)

	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")))
	_, err = AddDocument("/code/error.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	if codeFolder.etagValid != false {
		t.Error("expected etag to have been invalidated")
	}
	v2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v2: %x", v2)

	if string(v1) == string(v2) {
		t.Error("expected version to have changed")
	}
}

func TestETagUpdatedWhenDocumentRemoved(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")))
	_, err = AddDocument("/code/hello.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}
	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")))
	_, err = AddDocument("/code/error.go", sname, fsize, "text/plain")
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
	t.Logf("etag v1: %x", v1)

	_, err = RemoveDocument("/code/hello.go")
	if err != nil {
		t.Error(err)
	}

	if codeFolder.etagValid != false {
		t.Error("expected version to have been invalidated")
	}
	v2, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("etag v2: %x", v2)

	if string(v1) == string(v2) {
		t.Error("expected version to have changed")
	}
}

func TestETagUpdatedWhenDocumentUpdated(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")))
	_, err = AddDocument("/code/hello.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}
	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")))
	errorDoc, err := AddDocument("/code/error.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	dv1, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("document etag v1: %x", dv1)

	codeFolder, err := Retrieve("/code/")
	if err != nil {
		t.Error(err)
	}

	fv1, err := codeFolder.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("folder etag v1: %x", fv1)

	_, fsize, _, err = WriteFile(server, sname, bytes.NewReader([]byte("var ErrExistentialCrisis = errors.New(\"why?\")")))
	_, err = UpdateDocument("/code/error.go", fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	if errorDoc.etagValid != false {
		t.Error("expected document version to have been invalidated")
	}
	dv2, err := errorDoc.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("document etag v2: %x", dv2)

	if codeFolder.etagValid != false {
		t.Error("expected folder version to have been invalidated")
	}
	fv2, err := codeFolder.Version()
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
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("func hello() string {\n\treturn \"Hello, World\"\n}")))
	_, err = AddDocument("/code/hello.go", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}
	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("var ErrYouSuck = errors.New(\"YOU SUCK!!\")")))
	_, err = AddDocument("/code/error.go", sname, fsize, "text/plain")
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
	t.Logf("folder etag v1: %x", v1)

	rv1, err := root.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("root etag v1: %x", rv1)

	// 可愛い is 3 characters together taking up 9 bytes
	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("可愛い")))
	f, err := AddDocument("/Pictures/Kittens.png", sname, fsize, "image/png")
	if err != nil {
		t.Error(err)
	}
	if f.Length != 9 {
		t.Errorf("got: `%d', want: 9", f.Length)
	}

	v2, err := codeFolder.Version()
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
	rv2, err := root.Version()
	if err != nil {
		t.Error(err)
	}
	t.Logf("root etag v2: %x", rv2)

	if string(rv1) == string(rv2) {
		t.Error("expected root etag to have changed")
	}
}

const persistText = `<Root>
	<Nodes IsFolder="true">
		<Name>Documents/</Name>
		<Rname>/Documents/</Rname>
		<Mime>inode/directory</Mime>
		<ParentRName>/</ParentRName>
	</Nodes>
	<Nodes IsFolder="false">
		<Name>hello.txt</Name>
		<Rname>/Documents/hello.txt</Rname>
		<Sname>/tmp/rms/storage/32000000-0000-0000-0000-000000000000</Sname>
		<Mime>text/plain</Mime>
		<Length>13</Length>
		<LastMod>0001-01-01T00:00:00Z</LastMod>
		<ParentRName>/Documents/</ParentRName>
	</Nodes>
	<Nodes IsFolder="false">
		<Name>test.txt</Name>
		<Rname>/Documents/test.txt</Rname>
		<Sname>/tmp/rms/storage/31000000-0000-0000-0000-000000000000</Sname>
		<Mime>text/plain</Mime>
		<Length>15</Length>
		<LastMod>0001-01-01T00:00:00Z</LastMod>
		<ParentRName>/Documents/</ParentRName>
	</Nodes>
</Root>`

func TestPersist(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("This is a test.")))
	if err != nil {
		t.Error(err)
	}
	_, err = AddDocument("/Documents/test.txt", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("Hello, World!")))
	if err != nil {
		t.Error(err)
	}
	_, err = AddDocument("/Documents/hello.txt", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	fd, err := FS.Create(server.sroot + "/marshalled.xml")
	if err != nil {
		t.Error(err)
	}
	defer fd.Close()

	err = Persist(fd)
	if err != nil {
		t.Error(err)
	}

	Reset()

	fd.Seek(0, io.SeekStart)
	bs, err := io.ReadAll(fd)
	if err != nil {
		t.Error(err)
	}
	if persistText != string(bs) {
		t.Errorf("incorrect XML generated:\ngot:\n%s\n----\nwant:\n%s\n----", bs, persistText)
	}
}

func TestLoad(t *testing.T) {
	_, server := mockServer()

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("This is a test.")))
	if err != nil {
		t.Error(err)
	}
	_, err = AddDocument("/Documents/test.txt", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("Hello, World!")))
	if err != nil {
		t.Error(err)
	}
	_, err = AddDocument("/Documents/hello.txt", sname, fsize, "text/plain")
	if err != nil {
		t.Error(err)
	}

	fd, err := FS.Create(server.sroot + "/marshalled.xml")
	if err != nil {
		t.Error(err)
	}
	defer fd.Close() // ignore close err!

	err = Persist(fd)
	if err != nil {
		t.Error(err)
	}

	Reset()

	fd.Seek(0, io.SeekStart)
	bs, err := io.ReadAll(fd)
	if err != nil {
		t.Error(err)
	}
	if persistText != string(bs) {
		t.Errorf("incorrect XML generated:\ngot:\n%s\n----\nwant:\n%s\n----", bs, persistText)
	}

	fd.Seek(0, io.SeekStart)
	err = Load(fd)
	if err != nil {
		t.Error(err)
	}
	if len(files) != 4 {
		t.Errorf("wrong number of nodes; got: %d, want: 4", len(files))
	}

	// Ensure root is set correctly.
	r, err := Retrieve("/")
	if err != nil {
		t.Error(err)
	}
	if r != root {
		t.Errorf("root not set correctly; got: `%p', want: `%p'", r, root)
	}
	if len(r.children) != 1 {
		t.Errorf("root has %d children, want: 1", len(r.children))
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
			rname:     "/Documents/",
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

		if n.Name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.Name, c.name)
		}
		if n.Rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.Rname, c.rname)
		}
		if n.Mime != c.mime {
			t.Errorf("got: `%s', want: `%s'", n.Mime, c.mime)
		}
		if n.IsFolder != c.isDir {
			t.Errorf("got: isFolder == %t, want: isFolder == %t", n.IsFolder, c.isDir)
		}
		if len(n.children) != c.nchildren {
			t.Errorf("%s has %d children, want: %d", n.Name, len(n.children), c.nchildren)
		}
		hasAsChild := false
		for _, v := range p.children {
			if v == n {
				hasAsChild = true
				break
			}
		}
		if !hasAsChild {
			t.Errorf("parent `%s' is missing `%s' as a child", p.Name, n.Name)
		}
		if n.parent != p {
			t.Errorf("wrong parent for `%s'; got: `%s', want: `%s'", n.Name, n.parent.Name, p.Name)
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
	server, _ := New(rroot, sroot)
	fs := Mock()
	fs.CreateDirectories(server.sroot).
		AddDirectory("somewhere").Into().
		AddDirectory("Documents").Into().
		AddFile("hello.txt", "Hello, World!").
		AddFile("test.txt", "Whole life's a test.")

	Reset()
	errs := Migrate(server, "/somewhere/")
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
			rname:     "/Documents/",
			mime:      "inode/directory",
			nchildren: 2,
			isDir:     true,
		},
		{
			name:  "test.txt",
			rname: "/Documents/test.txt",
			mime:  "text/plain; charset=utf-8",
		},
		{
			name:  "hello.txt",
			rname: "/Documents/hello.txt",
			mime:  "text/plain; charset=utf-8",
		},
	}

	p := root
	for _, c := range checks {
		n, err := Retrieve(c.rname)
		if err != nil {
			t.Error(err)
		}

		if n.Name != c.name {
			t.Errorf("got: `%s', want: `%s'", n.Name, c.name)
		}
		if n.Rname != c.rname {
			t.Errorf("got: `%s', want: `%s'", n.Rname, c.rname)
		}
		if n.Mime != c.mime {
			t.Errorf("got: `%s', want: `%s'", n.Mime, c.mime)
		}
		if n.IsFolder != c.isDir {
			t.Errorf("got: isFolder == %t, want: isFolder == %t", n.IsFolder, c.isDir)
		}
		if len(n.children) != c.nchildren {
			t.Errorf("%s has %d children, want: %d", n.Name, len(n.children), c.nchildren)
		}
		hasAsChild := false
		for _, v := range p.children {
			if v == n {
				hasAsChild = true
				break
			}
		}
		if !hasAsChild {
			t.Errorf("parent `%s' is missing `%s' as a child", p.Name, n.Name)
		}
		if n.parent != p {
			t.Errorf("wrong parent for `%s'; got: `%s', want: `%s'", n.Name, n.parent.Name, p.Name)
		}
		if c.isDir {
			p = n // [#child_after]
		}
	}
}

func ExamplePersist() {
	_, server := mockServer()

	panicIf := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	sname, fsize, _, err := WriteFile(server, "", bytes.NewReader([]byte("This is a test.")))
	panicIf(err)

	_, err = AddDocument("/Documents/test.txt", sname, fsize, "text/plain")
	panicIf(err)

	sname, fsize, _, err = WriteFile(server, "", bytes.NewReader([]byte("Hello, World!")))
	panicIf(err)

	_, err = AddDocument("/Documents/hello.txt", sname, fsize, "text/plain")
	panicIf(err)

	fd, err := FS.Create(server.sroot + "/marshalled.xml")
	panicIf(err)
	defer fd.Close() // ignore close err!

	err = Persist(fd)
	panicIf(err)

	Reset()

	fd.Seek(0, io.SeekStart)
	bs, err := io.ReadAll(fd)
	panicIf(err)
	fmt.Printf("XML follows:\n%s\n", bs)

	fd.Seek(0, io.SeekStart)
	err = Load(fd)
	panicIf(err)

	fmt.Printf("Storage listing follows:\n%s", root)
	// Output: XML follows:
	// <Root>
	// 	<Nodes IsFolder="true">
	// 		<Name>Documents/</Name>
	// 		<Rname>/Documents/</Rname>
	// 		<Mime>inode/directory</Mime>
	// 		<ParentRName>/</ParentRName>
	// 	</Nodes>
	// 	<Nodes IsFolder="false">
	// 		<Name>hello.txt</Name>
	// 		<Rname>/Documents/hello.txt</Rname>
	// 		<Sname>/tmp/rms/storage/32000000-0000-0000-0000-000000000000</Sname>
	// 		<Mime>text/plain</Mime>
	// 		<Length>13</Length>
	// 		<LastMod>0001-01-01T00:00:00Z</LastMod>
	// 		<ParentRName>/Documents/</ParentRName>
	// 	</Nodes>
	// 	<Nodes IsFolder="false">
	// 		<Name>test.txt</Name>
	// 		<Rname>/Documents/test.txt</Rname>
	// 		<Sname>/tmp/rms/storage/31000000-0000-0000-0000-000000000000</Sname>
	// 		<Mime>text/plain</Mime>
	// 		<Length>15</Length>
	// 		<LastMod>0001-01-01T00:00:00Z</LastMod>
	// 		<ParentRName>/Documents/</ParentRName>
	// 	</Nodes>
	// </Root>
	// Storage listing follows:
	// {F} / [/] [6462373162316434]
	//   {F} Documents/ [/Documents/] [3466353235626333]
	//     {D} hello.txt (text/plain, 13) [/Documents/hello.txt -> /tmp/rms/storage/32000000-0000-0000-0000-000000000000] [6561373234373438]
	//     {D} test.txt (text/plain, 15) [/Documents/test.txt -> /tmp/rms/storage/31000000-0000-0000-0000-000000000000] [6530353836306537]
}
