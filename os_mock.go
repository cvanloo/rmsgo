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
	Open(name string) (file, error)
	Stat(name string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Truncate(name string, size int64) error
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
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

type mockFileSystem struct {
	parents   []string
	lastAdded string
	contents  map[string]*mockFile
}

var _ fileSystem = (*mockFileSystem)(nil)

func (m *mockFileSystem) Open(name string) (file, error) {
	if f, ok := m.contents[name]; ok {
		return f, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if f, ok := m.contents[name]; ok {
		return f, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	type fsmap struct {
		name string
		file *mockFile
	}
	fs := []fsmap{}
	for name, file := range m.contents {
		if strings.HasPrefix(name, root) {
			fs = append(fs, fsmap{name, file})
		}
	}
	sort.Slice(fs, func(i, j int) bool {
		return fs[i].name < fs[j].name
	})
	for _, v := range fs {
		fn(v.name, v.file, nil)
	}
	return nil
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
		cursor:  0,
		mode:    perm,
		lastMod: time.Now(),
	}
	return nil
}

type mockFile struct {
	isDir    bool
	name     string
	bytes    []byte
	cursor   int64
	children map[string]*mockFile
	mode     fs.FileMode
	lastMod  time.Time
}

var _ file = (*mockFile)(nil)
var _ fs.DirEntry = (*mockFile)(nil)
var _ fs.FileInfo = (*mockFile)(nil)

func (*mockFile) Close() error {
	// nop
	return nil
}

func (m *mockFile) Name() string {
	return m.name
}

func (m *mockFile) Stat() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *mockFile) Read(b []byte) (n int, err error) {
	if m.cursor == int64(len(m.bytes)) {
		return 0, io.EOF
	}
	n = copy(b, m.bytes[m.cursor:])
	m.cursor += int64(n)
	return
}

func (m *mockFile) Write(b []byte) (n int, err error) {
	nl := len(b) + len(m.bytes[:m.cursor])
	nbs := make([]byte, nl)
	copy(nbs, m.bytes[:m.cursor])
	n = copy(nbs[m.cursor:], b)
	m.bytes = nbs
	m.cursor += int64(n)
	return n, nil
}

func (m *mockFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		// relative to the origin of the file
		m.cursor = offset
	case 1:
		// relative to the current offset
		m.cursor += offset
	case 2:
		// relative to the end of the file
		m.cursor = int64(len(m.bytes)) - offset
	}
	return m.cursor, nil
}

func (m *mockFile) Info() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *mockFile) IsDir() bool {
	return m.isDir
}

func (m *mockFile) Type() fs.FileMode {
	return m.mode
}

func (m *mockFile) ModTime() time.Time {
	return m.lastMod
}

func (m *mockFile) Mode() fs.FileMode {
	return m.mode
}

func (m *mockFile) Size() int64 {
	return int64(len(m.bytes))
}

func (*mockFile) Sys() any {
	return nil
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
		cursor:  0,
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
		isDir:    true,
		name:     name,
		children: map[string]*mockFile{},
		mode:     0755,
		lastMod:  time.Now(),
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
