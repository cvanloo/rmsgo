package rmsgo

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type fileMock struct {
	contents []byte
	path     string
	modTime  time.Time
	isDir    bool
}

var _ fs.File = (*fileMock)(nil)
var _ fs.FileInfo = (*fileMock)(nil)

func (*fileMock) Close() error {
	return nil
}

func (f *fileMock) Read(buf []byte) (int, error) {
	l := len(f.contents)
	if lb := len(buf); lb < l {
		l = lb
	}
	for i := 0; i < l; i++ {
		buf[i] = f.contents[i]
	}
	return len(f.contents), nil
}

func (f *fileMock) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f *fileMock) IsDir() bool {
	return f.isDir
}

func (f *fileMock) ModTime() time.Time {
	return f.modTime
}

func (*fileMock) Mode() fs.FileMode {
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

func (fs fsMock) Open(name string) (fs.File, error) {
	file, ok := fs[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	file.path = name // FIXME: Isn't this a little ugly?
	return &file, nil
}

func (fs fsMock) Stat(name string) (fs.FileInfo, error) {
	file, ok := fs[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	file.path = name // FIXME: Isn't this a little ugly?
	return file.Stat()
}

func (fs fsMock) WriteFile(name string, data []byte, perm fs.FileMode) error {
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
