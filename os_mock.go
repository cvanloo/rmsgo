package rmsgo

import (
	"io/fs"
	"os"
	"path/filepath"
)

type fileSystem interface {
	Open(name string) (file, error)
	Stat(name string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type file interface {
	Close() error
	Name() string
	Read(b []byte) (n int, err error)
	Stat() (os.FileInfo, error)
	Write(b []byte) (n int, err error)
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

type mockFileSystem struct {
	contents map[string]*mockFile
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
		return f.Stat()
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	// @todo: ensure we walk the directory in lexical order
	for name, file := range m.contents {
		fn(name, file, nil)
	}
	return nil
}

type mockFile struct {
	isDir    bool
	name     string
	bytes    []byte
	children map[string]*mockFile
}

var _ file = (*mockFile)(nil)
var _ fs.DirEntry = (*mockFile)(nil)

func (*mockFile) Close() error {
	// nop
	return nil
}

func (m *mockFile) Name() string {
	return m.name
}

func (*mockFile) Read(b []byte) (n int, err error) {
	panic("unimplemented")
}

func (*mockFile) Stat() (fs.FileInfo, error) {
	panic("unimplemented")
}

func (*mockFile) Write(b []byte) (n int, err error) {
	// @todo: we probably just want to use os.ReadFile, os.Truncate, and
	// os.WriteFile instead
	panic("unimplemented")
}

func (m *mockFile) Info() (fs.FileInfo, error) {
	return m.Stat()
}

func (m *mockFile) IsDir() bool {
	return m.isDir
}

func (*mockFile) Type() fs.FileMode {
	panic("unimplemented")
}
