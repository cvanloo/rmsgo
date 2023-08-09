package mock

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/maps"
)

// The respective syscall.* errors listed here: https://unix.stackexchange.com/a/326811

type FileSystem interface {
	Create(path string) (File, error)
	Open(path string) (File, error)
	Stat(path string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Truncate(path string, size int64) error
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
}

type File interface {
	Close() error
	Name() string
	Stat() (os.FileInfo, error)
	Read(b []byte) (n int, err error)                     // go doc os.File.Read
	Write(b []byte) (n int, err error)                    // go doc os.File.Write
	Seek(offset int64, whence int) (ret int64, err error) // go doc os.File.Seek
}

type RealFileSystem struct{}

var _ FileSystem = (*RealFileSystem)(nil)

func (*RealFileSystem) Create(path string) (File, error) {
	return os.Create(path)
}

func (*RealFileSystem) Open(path string) (File, error) {
	return os.Open(path)
}

func (*RealFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (*RealFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (*RealFileSystem) Truncate(path string, size int64) error {
	return os.Truncate(path, size)
}

func (*RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (*RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (*RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (*RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

type FakeFileSystem struct {
	lastAdded, parent, root *FakeFile
	contents                map[string]*FakeFile
}

var _ FileSystem = (*FakeFileSystem)(nil)

const umask = 0022

func (m *FakeFileSystem) Create(path string) (File, error) {
	path = filepath.Clean(path)

	if path[len(path)-1] == '/' {
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.EISDIR,
		}
	}

	if f, ok := m.contents[path]; ok {
		if f.isDir {
			return nil, &os.PathError{
				Op:   "open",
				Path: path,
				Err:  syscall.EISDIR,
			}
		}
		f.bytes = nil
		f.lastMod = Time()
		return f.FD(), nil
	}

	parentPath := filepath.Dir(path)
	p, ok := m.contents[parentPath]
	if !ok {
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.ENOENT,
		}
	}

	f := &FakeFile{
		isDir:   false,
		path:    path,
		name:    filepath.Base(path),
		mode:    0666 - umask,
		lastMod: Time(),
		parent:  p,
	}
	p.children[path] = f
	m.contents[path] = f
	return f.FD(), nil
}

func (m *FakeFileSystem) Open(path string) (File, error) {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		return f.FD(), nil
	}
	return nil, &os.PathError{
		Op:   "open",
		Path: path,
		Err:  syscall.ENOENT,
	}
}

func (m *FakeFileSystem) Stat(path string) (fs.FileInfo, error) {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		return f.FD(), nil
	}
	return nil, &os.PathError{
		Op:   "stat",
		Path: path,
		Err:  syscall.ENOENT,
	}
}

func readDir(d *FakeFile) []*FakeFile {
	children := maps.Values(d.children)
	// files are visited in lexicographical order
	sort.Slice(children, func(i, j int) bool {
		return children[i].path < children[j].path
	})
	return children
}

func walkDir(d *FakeFile, fn fs.WalkDirFunc) error {
	err := fn(d.path, d.FD(), nil)
	if err == fs.SkipDir {
		return nil // successfully skipped directory
	}
	if err != nil {
		return err
	}

	dirEntries := readDir(d)
	for _, d := range dirEntries {
		if d.isDir {
			// we decend into directories first, before we continue on in the
			// current directory
			err = walkDir(d, fn)
		} else {
			err = fn(d.path, d.FD(), nil)
		}
		if err != nil {
			if err == fs.SkipDir {
				return nil // successfully skipped rest of directory
			}
			return err
		}
	}
	return nil
}

func (m *FakeFileSystem) WalkDir(root string, fn fs.WalkDirFunc) (err error) {
	// @fixme: the path passed to fn should always have root as prefix
	root = filepath.Clean(root)
	r, ok := m.contents[root]
	if !ok {
		err = &os.PathError{
			Op:   "lstat",
			Path: root,
			Err:  syscall.ENOENT,
		}
	}

	if err != nil {
		err = fn(root, r.FD(), err)
	} else {
		err = walkDir(r, fn)
	}

	if err == fs.SkipAll || err == fs.SkipDir {
		return nil
	}
	return err
}

func (m *FakeFileSystem) Truncate(path string, size int64) error {
	if f, ok := m.contents[path]; ok {
		if f.isDir {
			return &os.PathError{
				Op:   "truncate",
				Path: path,
				Err:  syscall.EISDIR,
			}
		}
		f.bytes = f.bytes[:size]
		return nil
	}
	return &os.PathError{
		Op:   "truncate",
		Path: path,
		Err:  syscall.ENOENT,
	}
}

func (m *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		if f.isDir {
			return nil, &os.PathError{
				Op:   "read",
				Path: path,
				Err:  syscall.EISDIR,
			}
		}
		return f.bytes, nil
	}
	return nil, &os.PathError{
		Op:   "open",
		Path: path,
		Err:  syscall.ENOENT,
	}
}

func (m *FakeFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		if f.isDir {
			return &os.PathError{
				Op:   "open",
				Path: path,
				Err:  syscall.EISDIR,
			}
		}
		f.bytes = data
		return nil
	}
	parentPath := filepath.Dir(path)
	p, ok := m.contents[parentPath]
	if !ok {
		return &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.ENOENT,
		}
	}

	f := &FakeFile{
		isDir:   false,
		path:    path,
		name:    filepath.Base(path),
		bytes:   data,
		mode:    perm,
		lastMod: Time(),
		parent:  p,
	}
	p.children[path] = f
	m.contents[path] = f
	return nil
}

func (m *FakeFileSystem) Remove(path string) error {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		if f.isDir && len(f.children) > 0 {
			return &os.PathError{
				Op:   "remove",
				Path: path,
				Err:  syscall.ENOTEMPTY,
			}
		}
		delete(m.contents, path)
		return nil
	}
	return &os.PathError{
		Op:   "remove",
		Path: path,
		Err:  syscall.ENOENT,
	}
}

func (m *FakeFileSystem) RemoveAll(path string) error {
	path = filepath.Clean(path)
	if f, ok := m.contents[path]; ok {
		delete(m.contents, path)
		if f.isDir {
			for _, c := range f.children {
				delete(m.contents, c.path)
			}
		}
	}
	return nil
}

type FakeFile struct {
	isDir      bool
	path, name string
	bytes      []byte
	mode       fs.FileMode
	lastMod    time.Time

	parent   *FakeFile
	children map[string]*FakeFile
}

func (m *FakeFile) FD() *FakeFileDescriptor {
	return &FakeFileDescriptor{m, 0}
}

type FakeFileDescriptor struct {
	file   *FakeFile
	cursor int64
}

var _ File = (*FakeFileDescriptor)(nil)
var _ fs.DirEntry = (*FakeFileDescriptor)(nil)
var _ fs.FileInfo = (*FakeFileDescriptor)(nil)

func (m *FakeFileDescriptor) Close() error {
	// m.file = nil // make sure that a closed fd can't be used anymore ???
	return nil // nop
}

func (m *FakeFileDescriptor) Name() string {
	return m.file.name
}

func (m *FakeFileDescriptor) Stat() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *FakeFileDescriptor) Read(b []byte) (n int, err error) {
	if m.file.isDir {
		return 0, &os.PathError{
			Op:   "read",
			Path: m.file.path,
			Err:  syscall.EISDIR,
		}
	}
	if m.cursor == int64(len(m.file.bytes)) {
		return 0, io.EOF
	}
	n = copy(b, m.file.bytes[m.cursor:])
	m.cursor += int64(n)
	return
}

func (m *FakeFileDescriptor) Write(b []byte) (n int, err error) {
	// @todo: write a test with multiple subsequent writes, make sure the data
	// is correctly written (nothing overlapping)
	// @todo: write a test where something in the middle of the file is
	// overwritten, beginning and end must stay the same.
	if m.file.isDir {
		return 0, &os.PathError{
			Op:   "write",
			Path: m.file.path,
			Err:  syscall.EBADF,
		}
	}
	nl := len(b) + len(m.file.bytes[:m.cursor])
	nbs := make([]byte, nl)
	copy(nbs, m.file.bytes[:m.cursor])
	n = copy(nbs[m.cursor:], b)
	m.file.bytes = nbs
	m.cursor += int64(n)
	return n, nil
}

func (m *FakeFileDescriptor) Seek(offset int64, whence int) (int64, error) {
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

func (m *FakeFileDescriptor) Info() (fs.FileInfo, error) {
	// m implements fs.FileInfo
	return m, nil
}

func (m *FakeFileDescriptor) IsDir() bool {
	return m.file.isDir
}

func (m *FakeFileDescriptor) Type() fs.FileMode {
	return m.file.mode
}

func (m *FakeFileDescriptor) ModTime() time.Time {
	return m.file.lastMod
}

func (m *FakeFileDescriptor) Mode() fs.FileMode {
	return m.file.mode
}

func (m *FakeFileDescriptor) Size() int64 {
	return int64(len(m.file.bytes))
}

func (m *FakeFileDescriptor) Sys() any {
	return m.file
}

func MockFS() (fs *FakeFileSystem) {
	r := &FakeFile{
		isDir:    true,
		path:     "/",
		name:     "/",
		mode:     0777 - umask,
		lastMod:  Time(),
		parent:   nil,
		children: map[string]*FakeFile{},
	}
	fs = &FakeFileSystem{
		lastAdded: r,
		parent:    r,
		root:      r,
		contents: map[string]*FakeFile{
			"/": r,
		},
	}
	return
}

func (m *FakeFileSystem) CreateDirectories(path string) *FakeFileSystem {
	path = filepath.Clean(path)
	parts := strings.Split(path, "/")[1:] // exclude empty ""

	p := m.root

	for i := range parts {
		pname := "/" + strings.Join(parts[:i+1], "/")
		pn, ok := m.contents[pname]
		if !ok {
			pn = &FakeFile{
				isDir:    true,
				path:     pname,
				name:     parts[i] + "/",
				mode:     0777 - umask,
				lastMod:  Time(),
				parent:   p,
				children: map[string]*FakeFile{},
			}
			p.children[pname] = pn
			m.contents[pname] = pn
		}
		p = pn
	}
	// p now points to the inner-most directory
	m.lastAdded = p
	return m
}

// @todo: Use Options pattern?
// @todo: make sure to use os.PathSeparator everywhere!!! (or fix filepath.Clean to only use /)

func (m *FakeFileSystem) AddFile(name, data string) *FakeFileSystem {
	if strings.Contains(name, "/") {
		panic("file name must not contain the Unix path separator ('/')")
	}
	path := filepath.Join(m.parent.path, name)
	f := &FakeFile{
		isDir:   false,
		path:    path,
		name:    name,
		bytes:   []byte(data),
		mode:    0666 - umask,
		lastMod: Time(),
		parent:  m.parent,
	}
	m.contents[path] = f
	m.parent.children[path] = f
	return m
}

func (m *FakeFileSystem) AddDirectory(name string) *FakeFileSystem {
	if name[len(name)-1] == '/' {
		name = name[:len(name)-1]
	}
	if strings.Contains(name, "/") {
		panic("directory name must only contain the Unix path separator ('/') as a suffix.")
	}
	path := filepath.Join(m.parent.path, name)
	d := &FakeFile{
		isDir:    true,
		path:     path,
		name:     name + "/",
		mode:     0777 - umask,
		lastMod:  Time(),
		parent:   m.parent,
		children: map[string]*FakeFile{},
	}
	m.lastAdded = d
	m.contents[path] = d
	m.parent.children[path] = d
	return m
}

func (m *FakeFileSystem) Into() *FakeFileSystem {
	m.parent = m.lastAdded
	return m
}

func (m *FakeFileSystem) Leave() *FakeFileSystem {
	m.parent = m.parent.parent
	return m
}

func (m *FakeFileSystem) String() (pp string) {
	ns := []*FakeFile{m.root}
	for len(ns) > 0 {
		n := ns[0]
		ns = ns[1:]

		var content string
		if n.isDir {
			content = "(Directory)"
		} else {
			content = "`" + string(n.bytes) + "'"
		}
		pp = fmt.Sprintf("%s\n%s: %s", pp, n.path, content)

		ns = append(ns, maps.Values(n.children)...)
	}
	return
}
