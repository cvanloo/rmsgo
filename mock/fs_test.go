package mock

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
)

// @todo: many more tests needed to test the correct (complicated) behaviour
// @todo: test error cases as well

const (
	testContent  = "Ich fand es schon immer verdächtig, dass die Sonne jeden Morgen im Osten aufgeht!"
	testFilePath = "/Classified/Fakenius.txt"
	testFileName = "Fakenius.txt"
	testFileDir  = "/Classified/"
	testPerm     = 0666
)

func TestReadFile(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestWriteFile(t *testing.T) {
	m := MockFS(
		WithDirectory(testFileDir),
	)
	err := m.WriteFile(testFilePath, []byte(testContent), testPerm)
	if err != nil {
		t.Error(err)
	}
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestWriteFileAndReadBack(t *testing.T) {
	m := MockFS(
		WithDirectory(testFileDir),
	)
	err := m.WriteFile(testFilePath, []byte(testContent), testPerm)
	if err != nil {
		t.Error(err)
	}
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if fd.Name() != testFileName {
		t.Errorf("got: `%s', want: `%s'", fd.Name(), testFileName)
	}
	bs := make([]byte, 128)
	n, err := fd.Read(bs)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(testContent) {
		t.Errorf("got %d, want %d", n, len(testContent))
	}
	bs = bs[:n] // strings won't be equal otherwise
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestWriteFileExisting(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	const newContent = "Giraffe > Greif"
	err := m.WriteFile(testFilePath, []byte(newContent), testPerm)
	if err != nil {
		t.Error(err)
	}
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(bs) != newContent {
		t.Errorf("got: `%s', want: `%s'", bs, newContent)
	}
}

func TestTruncate(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	err := m.Truncate(testFilePath, 8)
	if err != nil {
		t.Fatal(err)
	}
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	expected := testContent[:8]
	if string(bs) != expected {
		t.Errorf("got: `%s', want: `%s'", bs, expected)
	}
}

func TestStat(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fi, err := m.Stat(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode() != testPerm-umask {
		t.Errorf("got: %o, want: %o", fi.Mode(), testPerm-umask)
	}
	if fi.Name() != testFileName {
		t.Errorf("got: `%s', want: `%s'", fi.Name(), testFileName)
	}
	if fi.Size() != int64(len(testContent)) {
		t.Errorf("got: %d, want: %d", fi.Size(), len(testContent))
	}
	if fi.IsDir() {
		t.Error("got: `IsDir = true', want: `IsDir = false'")
	}
	// fi.ModTime()
}

func TestOpen(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if fd.Name() != testFileName {
		t.Errorf("got: `%s', want: `%s'", fd.Name(), testFileName)
	}
	err = fd.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestOpenErrNotExist(t *testing.T) {
	m := MockFS()
	_, err := m.Open("/home/prince/Documents/Good Advice.md")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}
}

func TestFile_Read(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	bs := make([]byte, 128)
	n, err := fd.Read(bs)
	if err != nil {
		t.Error(err)
	}
	bs = bs[:n]
	if string(bs) != testContent {
		t.Errorf("got: `%s', want: `%s'", bs, testContent)
	}
}

func TestFile_ReadAtEOF(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if ret != int64(len(testContent)) {
		t.Errorf("got: %d, want: %d", ret, len(testContent))
	}
	bs := make([]byte, 128)
	n, err := fd.Read(bs)
	if err != io.EOF {
		t.Errorf("got: `%v', want: `%v'", err, io.EOF)
	}
	if n != 0 {
		t.Errorf("got: %d, want: 0", n)
	}
}

func TestFile_Seek(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := fd.Seek(5, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if ret != 5 {
		t.Errorf("got: %d, want: 5", ret)
	}
	ret, err = fd.Seek(7, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if ret != 12 {
		t.Errorf("got: %d, want: 12", ret)
	}
	ret, err = fd.Seek(4, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if ret != int64(len(testContent)+4) {
		t.Errorf("got: %d, want: %d", ret, len(testContent)-4)
	}
}

func TestFile_WriteAtSeekEnd(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.OpenFile(testFilePath, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	if ret != int64(len(testContent)) {
		t.Errorf("got: %d, want: %d", ret, len(testContent))
	}
	n, err := fd.Write([]byte("123456"))
	if err != nil {
		t.Error(err)
	}
	if n != 6 {
		t.Errorf("got: %d, want: 6", n)
	}
	err = fd.Close()
	if err != nil {
		t.Error(err)
	}
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Error(err)
	}
	const expected = testContent + "123456"
	if string(bs) != expected {
		t.Errorf("got: `%s', want: `%s'", bs, expected)
	}
}

func TestFile_WriteSequence(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.OpenFile(testFilePath, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	ret, err = fd.Seek(ret-8, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if ret != int64(len(testContent)-8) {
		t.Errorf("got: %d, want: %d", ret, len(testContent)-8)
	}
	n, err := fd.Write([]byte("abcdef"))
	if err != nil {
		t.Error(err)
	}
	if n != 6 {
		t.Errorf("got: %d, want: 6", n)
	}
	n, err = fd.Write([]byte("1234567"))
	if err != nil {
		t.Error(err)
	}
	if n != 7 {
		t.Errorf("got: %d, want: 7", n)
	}
	err = fd.Close()
	if err != nil {
		t.Error(err)
	}
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Error(err)
	}
	expected := testContent[:len(testContent)-8] + "abcdef1234567"
	if string(bs) != expected {
		t.Errorf("got: `%s', want: `%s'", bs, expected)
	}
}

func TestFile_SeekAndWrite(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte("Ich fand es schon immer verdächtig, dass die Sonne jeden Morgen im Osten aufgeht!")),
	)
	fd, err := m.OpenFile(testFilePath, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}

	// overwrite beginning of file
	ns, err := fd.Write([]byte("1234"))
	if err != nil {
		t.Error(err)
	}
	if ns != 4 {
		t.Errorf("got: %d, want: 4", ns)
	}

	// overwrite in middle of file
	ret, err := fd.Seek(5, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if ret != 9 {
		t.Errorf("got: %d, want: 9", ret)
	}
	ns, err = fd.Write([]byte("abcdef"))
	if err != nil {
		t.Error(err)
	}
	if ns != 6 {
		t.Errorf("got: %d, want: 6", ns)
	}

	// overwrite at end of file
	ret, err = fd.Seek(79, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if ret != 79 {
		t.Errorf("got: %d, want: 79", ret)
	}
	ns, err = fd.Write([]byte("___"))
	if err != nil {
		t.Error(err)
	}
	if ns != 3 {
		t.Errorf("got: %d, want: 3", ns)
	}

	// append to file
	ns, err = fd.Write([]byte("---..."))
	if err != nil {
		t.Error(err)
	}
	if ns != 6 {
		t.Errorf("got: %d, want: 6", ns)
	}

	err = fd.Close()
	if err != nil {
		t.Error(err)
	}
	const expected = "1234fand abcdefon immer verdächtig, dass die Sonne jeden Morgen im Osten aufge___---..."
	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != expected {
		t.Errorf("got: `%s', want: `%s'", bs, expected)
	}
}

func TestFile_Stat(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := fd.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode() != testPerm-umask {
		t.Errorf("got: %o, want: %o", fi.Mode(), testPerm-umask)
	}
	if fi.Name() != testFileName {
		t.Errorf("got: `%s', want: `%s'", fi.Name(), testFileName)
	}
	if fi.Size() != int64(len(testContent)) {
		t.Errorf("got: %d, want: %d", fi.Size(), len(testContent))
	}
	if fi.IsDir() != false {
		t.Error("got: `IsDir = true', want: `IsDir = false'")
	}
	//fi.ModTime()
}

func TestRemoveFile(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	_, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	err = m.Remove(testFilePath)
	if err != nil {
		t.Error(err)
	}
	_, err = m.Open(testFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}
}

func TestRemoveDir(t *testing.T) {
	m := MockFS(
		WithDirectory(testFileDir),
	)
	_, err := m.Open(testFileDir)
	if err != nil {
		t.Fatal(err)
	}
	err = m.Remove(testFileDir)
	if err != nil {
		t.Error(err)
	}
	_, err = m.Open(testFileDir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}
}

func TestRemoveDirNonEmpty(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	err := m.Remove(testFileDir)
	if err == nil {
		t.Error("expected removal of non-empty directory to report error")
	}
}

func TestRemoveAll(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	_, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}
	err = m.RemoveAll(testFileDir)
	if err != nil {
		t.Error(err)
	}
	_, err = m.Open(testFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
	}
	_, err = m.Open(testFileDir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got: `%v', want: `%v'", err, os.ErrNotExist)
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
		t.Fatalf("got: `%v', want: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("got: `%s', want: `%s'", visited[i], expected[i])
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
		t.Fatalf("visited: `%v', expected: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("got: `%s', want: `%s'", visited[i], expected[i])
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
		t.Fatalf("visited: `%v', expected: `%v'", visited, expected)
	}
	for i := range expected {
		if expected[i] != visited[i] {
			t.Errorf("got: `%s', want: `%s'", visited[i], expected[i])
		}
	}
}

func TestFile_SeekPastEndAndReadReturnsEOF(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.Open(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	// We seek 7 bytes past the end of the file
	ret, err := fd.Seek(7, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	expLen := int64(len(testContent) + 7)
	if ret != expLen {
		t.Errorf("got: %d, want: %d", ret, expLen)
	}

	bs := make([]byte, 128)
	n, err := fd.Read(bs)
	if err != io.EOF {
		t.Errorf("got: `%v', want: `%v'", err, io.EOF)
	}
	if n != 0 {
		t.Errorf("got: %d, want: 0", n)
	}
}

func TestFile_SeekPastEndAndWrite(t *testing.T) {
	m := MockFS(
		WithFile(testFilePath, []byte(testContent)),
	)
	fd, err := m.OpenFile(testFilePath, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}

	// We seek 7 bytes past the end of the file
	ret, err := fd.Seek(7, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	expLen := int64(len(testContent) + 7)
	if ret != expLen {
		t.Errorf("got: %d, want: %d", ret, expLen)
	}

	const (
		appendText = "1234----"

		//          original content + 7 bytes skipped                + appended text
		expectedResult = testContent + "\x00\x00\x00\x00\x00\x00\x00" + appendText
	)

	n, err := fd.Write([]byte(appendText))
	if err != nil {
		t.Error(err)
	}
	if n != len(appendText) {
		t.Errorf("got: %d, want: %d", n, len(appendText))
	}

	ret, err = fd.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
	}
	nl := int64(len(expectedResult))
	if ret != nl {
		t.Errorf("got: %d, want: %d", ret, nl)
	}

	err = fd.Close()
	if err != nil {
		t.Error(err)
	}

	bs, err := m.ReadFile(testFilePath)
	if err != nil {
		t.Error(err)
	}
	if string(bs) != expectedResult {
		t.Errorf("got: `%s', want: `%s'", bs, expectedResult)
	}
}
