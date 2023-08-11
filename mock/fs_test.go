package mock

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
)

func mockStorage() *FakeFileSystem {
	m := MockFS(
		WithFile("/test.txt", []byte("Hello, World!")),
		WithFile("/Pictures/Kittens.png", []byte("A cute kitten!")),
		WithFile("/Pictures/Gopher.jpg", []byte("It's Gopher!")),
		WithFile("/Documents/doc.txt", []byte("Lorem ipsum dolores sit amet")),
		WithFile("/Documents/fakenius.txt", []byte("Ich fand es schon immer verdächtig, dass die Sonne jeden Morgen im Osten aufgeht!")),
	)
	return m
}

func TestReadFile(t *testing.T) {
	m := mockStorage()
	c, err := m.ReadFile("/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != "Hello, World!" {
		t.Errorf("file contents don't match; got: `%s', want: `%s'", c, "Hello, World!")
	}
}

func TestWriteExistingFile(t *testing.T) {
	m := mockStorage()
	const path string = "/Documents/fakenius.txt"
	err := m.WriteFile(path, []byte("Giraffe > Greif"), 0644)
	if err != nil {
		t.Error(err)
	}
	c, err := m.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != "Giraffe > Greif" {
		t.Errorf("file contents don't match; got: `%s', want: `%s'", c, "Giraffe > Greif")
	}
}

func TestWriteNewFile(t *testing.T) {
	ms := mockStorage()
	const path string = "/Documents/new.md"
	err := ms.WriteFile(path, []byte("Cats > Dogs"), 0644)
	if err != nil {
		t.Error(err)
	}
	c, err := ms.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != "Cats > Dogs" {
		t.Errorf("file contents don't match; got: `%s', want: `%s'", c, "Cats > Dogs")
	}
}

func TestWriteNewFileAndStats(t *testing.T) {
	ms := mockStorage()
	const path string = "/Documents/new.md"
	err := ms.WriteFile(path, []byte("Cats > Dogs"), 0644)
	if err != nil {
		t.Error(err)
	}

	fd, err := ms.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if fd.Name() != "new.md" {
		t.Errorf("incorrect name; got: `%s', want: `%s'", fd.Name(), "new.md")
	}

	content := make([]byte, 11)
	n, err := fd.Read(content)
	if err != nil {
		t.Fatal(err)
	}
	if n != 11 {
		t.Errorf("incorrect number of bytes read: got `%d', want `%d'", n, 11)
	}
	if string(content) != "Cats > Dogs" {
		t.Errorf("file contents don't match; got: `%s', want: `%s'", content, "Cats > Dogs")
	}
}

func TestTruncate(t *testing.T) {
	m := mockStorage()
	const path string = "/Pictures/Gopher.jpg"
	err := m.Truncate(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != "It" {
		t.Errorf("file contents don't match; got: `%s', want: `%s'", c, "It")
	}
}

func TestStat(t *testing.T) {
	m := mockStorage()
	s, err := m.Stat("/Pictures/Kittens.png")
	if err != nil {
		t.Fatal(err)
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
	m := mockStorage()
	_, err := m.Open("/Does/Not/Exist")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}
}

func TestOpenExistent(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Name() != "doc.txt" {
		t.Errorf("got: `%s', want: `%s'", fd.Name(), "doc.txt")
	}
	err = fd.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestFileRead(t *testing.T) {
	m := mockStorage()
	const expectedContent string = "Lorem ipsum dolores sit amet"
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatalf("got error: `%v'", err)
	}
	content := make([]byte, 30)
	nread, err := fd.Read(content)
	if err != nil {
		t.Error(err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileReadEOF(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}

	nc, err := fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if nc != 28 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 28)
	}

	content := make([]byte, 30)
	nread, err := fd.Read(content)
	if err != io.EOF {
		t.Errorf("got: `%v', want: `%v'", err, io.EOF)
	}
	if nread != 0 {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, 0)
	}
}

func TestFileSeek(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}

	nc, err := fd.Seek(5, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if nc != 5 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 5)
	}

	nc, err = fd.Seek(7, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if nc != 12 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 12)
	}

	nc, err = fd.Seek(4, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}
}

func TestFileWriteAtSeekEnd(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}

	nc, err := fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if nc != 28 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 28)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Error(err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit ametabcdef"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Error(err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileWriteOverwriteParts(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}

	nc, err := fd.Seek(4, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Error(err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit abcdef"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Error(err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileMultipleWrite(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Documents/doc.txt")
	if err != nil {
		t.Fatal(err)
	}

	nc, err := fd.Seek(4, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if nc != 24 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 24)
	}

	nw, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Error(err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	// test that Write advances the cursor
	nw, err = fd.Write([]byte("ghijkl"))
	if err != nil {
		t.Error(err)
	}
	if nw != 6 {
		t.Errorf("incorrect number of bytes written; got: `%d', want: `%d'", nw, 6)
	}

	nc, err = fd.Seek(0, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if nc != 0 {
		t.Errorf("incorrect cursor position; got: `%d', want: `%d'", nc, 0)
	}

	const expectedContent string = "Lorem ipsum dolores sit abcdefghijkl"
	content := make([]byte, 40)
	nread, err := fd.Read(content)
	if err != nil {
		t.Error(err)
	}
	if nread != len(expectedContent) {
		t.Errorf("incorrect number of bytes read; got: `%d', want: `%d'", nread, len(expectedContent))
	}
	if string(content[:nread]) != expectedContent {
		t.Errorf("incorrect content read; got: `%s', want: `%s'", content[:nread], expectedContent)
	}
}

func TestFileStat(t *testing.T) {
	m := mockStorage()
	fd, err := m.Open("/Pictures/Kittens.png")
	if err != nil {
		t.Fatal(err)
	}
	s, err := fd.Stat()
	if err != nil {
		t.Fatal(err)
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

func TestRemove(t *testing.T) {
	m := mockStorage()
	const path = "/Documents/fakenius.txt"
	_, err := m.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	err = m.Remove(path)
	if err != nil {
		t.Error(err)
	}
	_, err = m.Open(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%s' (expected file to be inexistent)", err, os.ErrNotExist)
	}
}

func TestRemoveNonEmpty(t *testing.T) {
	m := mockStorage()
	err := m.Remove("/Pictures/")
	if err == nil {
		t.Error("expected removal of non-empty directory to report error")
	}
}

func TestRemoveAll(t *testing.T) {
	m := mockStorage()
	_, err := m.Open("/Documents/fakenius.txt")
	if err != nil {
		t.Fatal(err)
	}
	err = m.RemoveAll("/Documents/")
	if err != nil {
		t.Error(err)
	}
	_, err = m.Open("/Documents/fakenius.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%s' (expected file to be inexistent)", err, os.ErrNotExist)
	}
	_, err = m.Open("/Documents/doc.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%s' (expected file to be inexistent)", err, os.ErrNotExist)
	}
	_, err = m.Open("/Documents/")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%s' (expected file to be inexistent)", err, os.ErrNotExist)
	}
}

func TestWalkDir(t *testing.T) {
	m := MockFS(
		WithFile("/tmp/t/1", []byte("")),
		WithFile("/tmp/t/3", []byte("")),
		WithFile("/tmp/t/2/4", []byte("")),
		WithFile("/tmp/t/2/5", []byte("")),
		WithFile("/tmp/t/2/6", []byte("")),
		WithFile("/tmp/t/2/7/8/9", []byte("")),
	)

	expected, visited := []string{
		"/tmp/t",
		"/tmp/t/1",
		"/tmp/t/2",
		"/tmp/t/2/4",
		"/tmp/t/2/5",
		"/tmp/t/2/6",
		"/tmp/t/2/7",
		"/tmp/t/2/7/8",
		"/tmp/t/2/7/8/9",
		"/tmp/t/3",
	}, []string{}

	err := m.WalkDir("/tmp/t/", func(path string, d fs.DirEntry, err error) error {
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if len(expected) != len(visited) {
		t.Fatalf("incorrect paths visited: visited: `%v', expected: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("wrong path: got: `%s', want: `%s'", visited[i], expected[i])
		}
	}
}

func TestWalkDirSkipOnDir(t *testing.T) {
	m := MockFS(
		WithFile("/tmp/t/1", []byte("")),
		WithFile("/tmp/t/3", []byte("")),
		WithFile("/tmp/t/2/4", []byte("")),
		WithFile("/tmp/t/2/5", []byte("")),
		WithFile("/tmp/t/2/6", []byte("")),
		WithFile("/tmp/t/2/7/8/9", []byte("")),
	)

	expected, visited := []string{
		"/tmp/t",
		"/tmp/t/1",
		"/tmp/t/2",
		"/tmp/t/3",
	}, []string{}

	err := m.WalkDir("/tmp/t/", func(path string, d fs.DirEntry, err error) error {
		visited = append(visited, path)
		if path == "/tmp/t/2" {
			return fs.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Error(err)
	}
	if len(expected) != len(visited) {
		t.Fatalf("incorrect paths visited: visited: `%v', expected: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("wrong path: got: `%s', want: `%s'", visited[i], expected[i])
		}
	}
}

func TestWalkDirSkipWithinDir(t *testing.T) {
	m := MockFS(
		WithFile("/tmp/t/1", []byte("")),
		WithFile("/tmp/t/3", []byte("")),
		WithFile("/tmp/t/2/4", []byte("")),
		WithFile("/tmp/t/2/5", []byte("")),
		WithFile("/tmp/t/2/6", []byte("")),
		WithFile("/tmp/t/2/7/8/9", []byte("")),
	)

	expected, visited := []string{
		"/tmp/t",
		"/tmp/t/1",
		"/tmp/t/2",
		"/tmp/t/2/4",
		"/tmp/t/2/5",
		"/tmp/t/3",
	}, []string{}

	err := m.WalkDir("/tmp/t/", func(path string, d fs.DirEntry, err error) error {
		visited = append(visited, path)
		if path == "/tmp/t/2/5" {
			return fs.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Error(err)
	}
	if len(expected) != len(visited) {
		t.Fatalf("incorrect paths visited: visited: `%v', expected: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("wrong path: got: `%s', want: `%s'", visited[i], expected[i])
		}
	}
}
