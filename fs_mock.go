package rmsgo

import (
	ioFS "io/fs"
	"os"
	"path/filepath"
	"time"
)

// fs holds the fileSystem implementation to use.
// For mocking purposes this can be overwritten by a mock fs.
var fs fileSystem = osFS{}

// fileSystem implements method for working with files.
// This abstraction allows mocking the file system when testing.
type fileSystem interface {
	Open(name string) (ioFS.File, error)
	Stat(name string) (os.FileInfo, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
}

// osFS is a fileSystem implementatino that delegates to the os methods.
type osFS struct{}

func (osFS) Open(name string) (ioFS.File, error) {
	return os.Open(name)
}

func (osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (osFS) Remove(name string) error {
	return os.Remove(name)
}

type fileMock struct {
	contents []byte
	path     string
	modTime  time.Time
	isDir    bool
	ptr     int
}

var _ ioFS.File = (*fileMock)(nil)
var _ ioFS.FileInfo = (*fileMock)(nil)

func (*fileMock) Close() error {
	return nil
}

func (f *fileMock) Read(buf []byte) (int, error) {
	i := 0
	cl := len(f.contents)
	bl := len(buf)
	for i < bl && f.ptr < cl {
		buf[i] = f.contents[f.ptr];
		f.ptr++
		i++
	}
	return i, nil
}

func (f *fileMock) Stat() (ioFS.FileInfo, error) {
	return f, nil
}

func (f *fileMock) IsDir() bool {
	return f.isDir
}

func (f *fileMock) ModTime() time.Time {
	return f.modTime
}

func (*fileMock) Mode() ioFS.FileMode {
	panic("unimplemented")
}

func (f *fileMock) Name() string {
	return filepath.Base(f.path)
}

func (f *fileMock) Size() int64 {
	return int64(len(f.contents))
}

func (*fileMock) Sys() any {
	panic("unimplemented")
}

type fsMock map[string]fileMock

var _ fileSystem = (*fsMock)(nil)

func (fs fsMock) Open(name string) (ioFS.File, error) {
	file, ok := fs[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	file.path = name // FIXME: Isn't this a little ugly?
	return &file, nil
}

func (fs fsMock) Stat(name string) (ioFS.FileInfo, error) {
	file, ok := fs[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	file.path = name // FIXME: Isn't this a little ugly?
	return file.Stat()
}

func (fs fsMock) WriteFile(name string, data []byte, perm ioFS.FileMode) error {
	isDir := false
	l := len(name)
	if name[l-1] == '/' {
		isDir = true
	}
	fs[name] = fileMock{
		contents: data,
		path:     name,
		modTime:  time.Now(),
		isDir:    isDir,
	}
	return nil
}

func (fs fsMock) Remove(name string) error {
	if _, ok := fs[name]; !ok {
		return os.ErrNotExist
	}
	delete(fs, name)
	return nil
}
