package rmsgo

import (
	"io"
	"io/fs"
	"os"
	"testing"
)

func createMock() *mockFileSystem {
	m := CreateMockFS()
	m.
		AddFile("test.txt", "Hello, World!").
		AddDirectory("Pictures").
		Into().
		AddFile("Kittens.png", "A cute kitten!").
		AddFile("Gopher.jpg", "It's Gopher!").
		Leave().
		AddDirectory("Documents").
		Into().
		AddFile("doc.txt", "Lorem ipsum dolores sit amet").
		AddFile("fakenius.txt", "Ich fand es schon immer verdächtig, dass die Sonne jeden Morgen im Osten aufgeht!")
	return m
}

func TestReadFile(t *testing.T) {
	m := createMock()
	c, err := m.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile: got error: `%v'", err)
	}
	if string(c) != "Hello, World!" {
		t.Errorf("ReadFile: file contents don't match; got: `%s', want: `%s'", c, "Hello, World!")
	}
}

func TestWriteExistingFile(t *testing.T) {
	m := createMock()
	const path string = "/Documents/fakenius.txt"
	err := m.WriteFile(path, []byte("Giraffe > Greif"), 0644)
	if err != nil {
		t.Errorf("WriteFile: got error: `%v'", err)
	}
	c, err := m.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: got error: `%v'", err)
	}
	if string(c) != "Giraffe > Greif" {
		t.Errorf("ReadFile: file contents don't match; got: `%s', want: `%s'", c, "Giraffe > Greif")
	}
}

func TestWriteNewFile(t *testing.T) {
	ms := createMock()
	const path string = "/Documents/new.md"
	err := ms.WriteFile(path, []byte("Cats > Dogs"), 0644)
	if err != nil {
		t.Errorf("WriteFile: got error: `%v'", err)
	}
	c, err := ms.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: got error: `%v'", err)
	}
	if string(c) != "Cats > Dogs" {
		t.Errorf("ReadFile: file contents don't match; got: `%s', want: `%s'", c, "Cats > Dogs")
	}
}

func TestWriteNewFileAndStats(t *testing.T) {
	ms := createMock()
	const path string = "/Documents/new.md"
	err := ms.WriteFile(path, []byte("Cats > Dogs"), 0644)
	if err != nil {
		t.Errorf("WriteFile: got error: `%v'", err)
	}

	fd, err := ms.Open(path)
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}
	if fd.Name() != "new.md" {
		t.Errorf("Name: incorrect name; got: `%s', want: `%s'", fd.Name(), "new.md")
	}

	content := make([]byte, 11)
	n, err := fd.Read(content)
	if err != nil {
		t.Fatalf("Read: got error: `%v'", err)
	}
	if n != 11 {
		t.Errorf("incorrect number of bytes read: got `%d', want `%d'", n, 11)
	}
	if string(content) != "Cats > Dogs" {
		t.Errorf("ReadFile: file contents don't match; got: `%s', want: `%s'", content, "Cats > Dogs")
	}
}

func TestTruncate(t *testing.T) {
	m := createMock()
	const path string = "/Pictures/Gopher.jpg"
	err := m.Truncate(path, 2)
	if err != nil {
		t.Fatalf("Truncate: got error: `%v'", err)
	}
	c, err := m.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: got error: `%v'", err)
	}
	if string(c) != "It" {
		t.Errorf("ReadFile: file contents don't match; got: `%s', want: `%s'", c, "It")
	}
}

func TestWalkDir(t *testing.T) {
	m := createMock()
	idx := 0
	expected := []string{
		"/Pictures/",
		"/Pictures/Gopher.jpg",
		"/Pictures/Kittens.png",
	}
	m.WalkDir("/Pictures/", func(path string, d fs.DirEntry, err error) error {
		if expected[idx] != path {
			t.Errorf("WalkDir: wrong path; got: `%s', want: `%s'", path, expected[idx])
		}
		idx++
		return nil
	})
}

func TestStat(t *testing.T) {
	m := createMock()
	s, err := m.Stat("/Pictures/Kittens.png")
	if err != nil {
		t.Fatalf("Stat: got error: `%v'", err)
	}
	if s.Mode() != 0644 {
		t.Errorf("incorrect mode; got: `%o', want: `%o'", s.Mode(), 0644)
	}
	if s.Name() != "Kittens.png" {
		t.Errorf("incorrect name; got: `%s', want: `%s'", s.Name(), "Kittens.png")
	}
	if s.Size() != int64(len("A cute kitten!")) {
		t.Errorf("incorrect size; got: `%d', want: `%d'", s.Size(), len("A cute kitten!"))
	}
	if s.IsDir() != false {
		t.Error("incorrect type; got: `IsDir = true', want: `IsDir = false'")
	}
	//s.ModTime()
}

func TestOpenNonExistent(t *testing.T) {
	m := createMock()
	_, err := m.Open("/Does/Not/Exist")
	if err != os.ErrNotExist {
		t.Fatalf("Open: got: `%v', want: `%v'", err, os.ErrNotExist)
	}
}

func TestOpenExistent(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}
	if fd.Name() != "doc.txt" {
		t.Errorf("Name: got: `%s', want: `%s'", fd.Name(), "doc.txt")
	}
	err = fd.Close()
	if err != nil {
		t.Errorf("Close: got error: `%v'", err)
	}
}

func TestFileRead(t *testing.T) {
	m := createMock()
	const expectedContent string = "Lorem ipsum dolores sit amet"
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}
	content := make([]byte, 30)
	nread, err := fd.Read(content)
	if err != nil {
		t.Errorf("Read: got error: `%v'", err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileReadEOF(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}

	nc, err := fd.Seek(0, 2) // seek end
	if err != nil {
		t.Fatalf("Seek: got error: `%v'", err)
	}
	if nc != 28 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 28)
	}

	content := make([]byte, 30)
	nread, err := fd.Read(content)
	if err != io.EOF {
		t.Errorf("EOF: got: `%v', want: `%v'", err, io.EOF)
	}
	if nread != 0 {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, 0)
	}
}

func TestFileSeek(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}

	nc, err := fd.Seek(5, 0)
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 5 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 5)
	}

	nc, err = fd.Seek(7, 1)
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 12 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 12)
	}

	nc, err = fd.Seek(4, 2)
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}
}

func TestFileWriteAtSeekEnd(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}

	nc, err := fd.Seek(0, 2) // Seek to end
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 28 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 28)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Errorf("Write: got error: `%v'", err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, 0) // Seek to beginning
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit ametabcdef"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Errorf("Read: got error: `%v'", err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileWriteOverwriteParts(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}

	nc, err := fd.Seek(4, 2) // Seek to end-4
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Errorf("Write: got error: `%v'", err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, 0) // Seek to beginning
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit abcdef"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Errorf("Read: got error: `%v'", err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileMultipleWrite(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}

	nc, err := fd.Seek(4, 2) // Seek to end-4
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Errorf("Write: got error: `%v'", err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	// test that Write advances the cursor
	nw, err = fd.Write([]byte("ghijkl"))
	if err != nil {
		t.Errorf("Write: got error: `%v'", err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, 0) // Seek to beginning
	if err != nil {
		t.Errorf("Seek: got error: `%v'", err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit abcdefghijkl"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Errorf("Read: got error: `%v'", err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileStat(t *testing.T) {
	m := createMock()
	fd, err := m.Open("/Pictures/Kittens.png")
	if err != nil {
		t.Fatalf("Open: got error: `%v'", err)
	}
	s, err := fd.Stat()
	if err != nil {
		t.Fatalf("Stat: got error: `%v'", err)
	}
	if s.Mode() != 0644 {
		t.Errorf("incorrect mode; got: `%o', want: `%o'", s.Mode(), 0644)
	}
	if s.Name() != "Kittens.png" {
		t.Errorf("incorrect name; got: `%s', want: `%s'", s.Name(), "Kittens.png")
	}
	if s.Size() != int64(len("A cute kitten!")) {
		t.Errorf("incorrect size; got: `%d', want: `%d'", s.Size(), len("A cute kitten!"))
	}
	if s.IsDir() != false {
		t.Error("incorrect type; got: `IsDir = true', want: `IsDir = false'")
	}
	//s.ModTime()
}
