package rmsgo

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fileSystem interface {
	Create(name string) (file, error)
	Open(name string) (file, error)
	Stat(name string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Truncate(name string, size int64) error
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(name string) error
}

type file interface {
	Close() error
	Name() string
	Stat() (os.FileInfo, error)
	Read(b []byte) (n int, err error)                     // go doc os.File.Read
	Write(b []byte) (n int, err error)                    // go doc os.File.Write
	Seek(offset int64, whence int) (ret int64, err error) // go doc os.File.Seek
}

type osFileSystem struct{}

var _ fileSystem = (*osFileSystem)(nil)

func (*osFileSystem) Create(name string) (file, error) {
	return os.Create(name)
}

func (*osFileSystem) Open(name string) (file, error) {
	return os.Open(name)
}

func (*osFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (*osFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (*osFileSystem) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
}

func (*osFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (*osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (*osFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (*osFileSystem) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

type mockFileSystem struct {
	parents   []string
	lastAdded string
	contents  map[string]*mockFile
}

var _ fileSystem = (*mockFileSystem)(nil)

func (m *mockFileSystem) Create(name string) (file, error) {
	if f, ok := m.contents[name]; ok {
		f.bytes = nil
		f.lastMod = time.Now()
		return f.Fd(), nil
	}
	parts := strings.Split(name, "/")
	f := &mockFile{
		isDir:   false,
		name:    parts[len(parts)-1],
		bytes:   nil,
		mode:    0666,
		lastMod: time.Now(),
	}
	m.contents[name] = f
	return f.Fd(), nil
}

func (m *mockFileSystem) Open(name string) (file, error) {
	if f, ok := m.contents[name]; ok {
		return f.Fd(), nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if f, ok := m.contents[name]; ok {
		return f.Fd(), nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	type fsmap struct {
		name string
		file *mockFile
	}
	files := []fsmap{}
	for name, file := range m.contents {
		if strings.HasPrefix(name, root) {
			files = append(files, fsmap{name, file})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})
	var walkErr error
	for _, v := range files {
		err := fn(v.name, v.file.Fd(), nil)
		switch err {
		case nil:
			continue
		case fs.SkipDir:
		// @todo: this will be much easier to implement if we have children references
		case fs.SkipAll:
			break
		default:
			walkErr = err
			break
		}
	}
	return walkErr
}

func (m *mockFileSystem) Truncate(name string, size int64) error {
	if f, ok := m.contents[name]; ok {
		f.bytes = f.bytes[:size]
		return nil
	}
	return os.ErrNotExist
}

func (m *mockFileSystem) ReadFile(name string) ([]byte, error) {
	if f, ok := m.contents[name]; ok {
		return f.bytes, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	if f, ok := m.contents[name]; ok {
		f.bytes = data
		return nil
	}
	parts := strings.Split(name, "/")
	m.contents[name] = &mockFile{
		isDir:   false,
		name:    parts[len(parts)-1],
		bytes:   data,
		mode:    perm,
		lastMod: time.Now(),
	}
	return nil
}

func (m *mockFileSystem) Remove(name string) error {
	if f, ok := m.contents[name]; ok {
		c := 0
		m.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
			c++
			return nil
		})
		if f.isDir && c > 1 {
			return &os.PathError{
				Op:   "Remove",
				Path: name,
				Err:  fmt.Errorf("cannot delete non-empty directory"),
			}
		}
		delete(m.contents, name)
	}
	return nil
}

func (m *mockFileSystem) RemoveAll(name string) error {
	if f, ok := m.contents[name]; ok {
		if f.isDir {
			m.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
				delete(m.contents, path)
				return nil
			})
		}
	}
	return nil
}

type mockFile struct {
	isDir   bool
	name    string
	bytes   []byte
	mode    fs.FileMode
	lastMod time.Time
}

func (m *mockFile) Fd() *mockFileFd {
	return &mockFileFd{m, 0}
}

type mockFileFd struct {
	file   *mockFile
	cursor int64
}

var _ file = (*mockFileFd)(nil)
var _ fs.DirEntry = (*mockFileFd)(nil)
var _ fs.FileInfo = (*mockFileFd)(nil)

func (*mockFileFd) Close() error {
	// nop
	return nil
}

func (m *mockFileFd) Name() string {
	return m.file.name
}

func (m *mockFileFd) Stat() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *mockFileFd) Read(b []byte) (n int, err error) {
	if m.cursor == int64(len(m.file.bytes)) {
		return 0, io.EOF
	}
	n = copy(b, m.file.bytes[m.cursor:])
	m.cursor += int64(n)
	return
}

func (m *mockFileFd) Write(b []byte) (n int, err error) {
	nl := len(b) + len(m.file.bytes[:m.cursor])
	nbs := make([]byte, nl)
	copy(nbs, m.file.bytes[:m.cursor])
	n = copy(nbs[m.cursor:], b)
	m.file.bytes = nbs
	m.cursor += int64(n)
	return n, nil
}

func (m *mockFileFd) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		// relative to the origin of the file
		m.cursor = offset
	case io.SeekCurrent:
		// relative to the current offset
		m.cursor += offset
	case io.SeekEnd:
		// relative to the end of the file
		m.cursor = int64(len(m.file.bytes)) - offset
	}
	return m.cursor, nil
}

func (m *mockFileFd) Info() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *mockFileFd) IsDir() bool {
	return m.file.isDir
}

func (m *mockFileFd) Type() fs.FileMode {
	return m.file.mode
}

func (m *mockFileFd) ModTime() time.Time {
	return m.file.lastMod
}

func (m *mockFileFd) Mode() fs.FileMode {
	return m.file.mode
}

func (m *mockFileFd) Size() int64 {
	return int64(len(m.file.bytes))
}

func (m *mockFileFd) Sys() any {
	return m.file
}

func CreateMockFS() (fs *mockFileSystem) {
	fs = &mockFileSystem{
		contents: map[string]*mockFile{},
	}
	//fs.AddDirectory("/")
	//fs.Into()
	return
}

func (m *mockFileSystem) AddFile(name, data string) *mockFileSystem {
	path := strings.Join(append(m.parents, name), "/")
	path = "/" + path
	m.contents[path] = &mockFile{
		isDir:   false,
		name:    name,
		bytes:   []byte(data),
		mode:    0644,
		lastMod: time.Now(),
	}
	return m
}

func (m *mockFileSystem) AddDirectory(name string) *mockFileSystem {
	path := strings.Join(append(m.parents, name), "/")
	path = "/" + path
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	m.contents[path] = &mockFile{
		isDir:   true,
		name:    name,
		mode:    0755,
		lastMod: time.Now(),
	}
	m.lastAdded = name
	return m
}

func (m *mockFileSystem) Into() *mockFileSystem {
	m.parents = append(m.parents, m.lastAdded)
	return m
}

func (m *mockFileSystem) Leave() *mockFileSystem {
	m.parents = m.parents[:len(m.parents)-1]
	return m
}

func (m *mockFileSystem) String() string {
	var pp string
	m.WalkDir("/", func(path string, d fs.DirEntry, err error) error {
		var content string
		if d.IsDir() {
			content = "[Directory]"
		} else {
			bs, err := m.ReadFile(path)
			content = string(bs)
			if err != nil {
				content = fmt.Sprintf("error reading file: %s", err)
			}
		}
		pp = fmt.Sprintf("%s\n%s: `%s'", pp, path, content)
		return nil
	})
	return pp
}
