package rmsgo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type node struct {
	parent   *node
	isFolder bool

	// "Kittens.png"
	name string

	// "/Pictures/Kittens.png"
	rname string

	// "/var/rms/storage/(uuid)"
	sname string

	etag      ETag
	etagValid bool

	mime     string
	length   int64
	lastMod  time.Time
	children map[string]*node
}

func (n *node) Valid() bool {
	return n.etagValid
}

func (n *node) Invalidate() {
	n.etagValid = false
}

func (n *node) ETag() (ETag, error) {
	if !n.etagValid {
		err := calculateETag(n)
		if err != nil {
			return n.etag, err
		}
	}
	return n.etag, nil
}

func (n *node) Equal(other *node) bool {
	if !(n.etagValid && other.etagValid) {
		return false
	}
	return n.etag.Equal(other.etag)
}

var (
	files map[string]*node
	root  *node
)

var ErrFileExists = errors.New("file already exists")

func init() {
	rn := &node{
		isFolder: true,
		name:     "/",
		rname:    "/",
		mime:     "inode/directory",
		children: map[string]*node{},
	}
	files = make(map[string]*node)
	files["/"] = rn
	root = rn
}

func ResetStorage(cfg Server) error {
	for k, v := range files {
		if v != root {
			delete(files, k)
		}
	}
	root.children = make(map[string]*node)
	// @todo: re-initialize from file system cfg.Sroot
	return ErrNotImplemented
}

func CreateDocument(cfg Server, rname string, data io.Reader, mime string) (*node, error) {
	if f, ok := files[rname]; ok {
		return f, ErrFileExists
	}

	assert(rname[len(rname)-1] != '/', "CreateDocument must only be used to create files")

	u, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	sname := filepath.Join(cfg.Sroot, u.String())
	fd, err := mfs.Create(sname) // @todo: set permissions
	if err != nil {
		return nil, err
	}
	fsize, err := io.Copy(fd, data)
	if err != nil {
		return nil, err
	}

	pname := filepath.Dir(rname)
	var parts []string
	for _, s := range strings.Split(pname, string(os.PathSeparator)) {
		if s != "" {
			parts = append(parts, s)
		}
	}

	p := root

	for i := range parts {
		pname := "/" + strings.Join(parts[:i+1], string(os.PathSeparator)) + "/"
		pn, ok := files[pname]
		if !ok {
			pn = &node{
				parent:   p,
				isFolder: true,
				name:     parts[i],
				rname:    pname,
				mime:     "inode/directory",
				children: map[string]*node{},
			}
			p.children[pname] = pn
			files[pname] = pn
		}
		p = pn
	}
	// p now points to the file's immediate parent [#1]

	name := filepath.Base(rname)

	f := &node{
		parent:   p, // [#1] assign parent
		isFolder: false,
		name:     name,
		rname:    rname,
		sname:    sname,
		mime:     mime,
		length:   int64(fsize),
		lastMod:  time.Now(),
	}
	p.children[rname] = f
	files[rname] = f

	n := f
	for n != nil {
		n.Invalidate()
		n = n.parent
	}

	return f, nil
}

func UpdateDocument(cfg Server, rname string, data io.Reader, mime string) (*node, error) {
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.isFolder, "UpdateDocument must not be called on a folder")

	fd, err := mfs.Create(f.sname) // @todo: set permissions?
	if err != nil {
		return f, err
	}
	fsize, err := io.Copy(fd, data)
	if err != nil {
		return f, err
	}

	f.mime = mime
	f.length = int64(fsize)
	f.lastMod = time.Now()

	n := f
	for n != nil {
		n.Invalidate()
		n = n.parent
	}

	return f, nil
}

func RemoveDocument(cfg Server, rname string) (*node, error) {
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.isFolder, "RemoveDocument must not be called on a folder")

	p := f
	for len(p.children) == 0 && p != root {
		mfs.Remove(p.sname)
		pp := p.parent
		delete(pp.children, p.rname)
		delete(files, p.rname)
		p = pp
	}
	// p now points to the parent deepest down the ancestry that is not empty

	for p != nil {
		p.Invalidate()
		p = p.parent
	}

	return f, nil
}

func Node(rname string) (*node, error) {
	if f, ok := files[rname]; ok {
		return f, nil
	}
	return nil, ErrNotFound
}

func (n node) String() string {
	return n.StringIdent(0)
}

func (n node) StringIdent(ident int) (s string) {
	for i := 0; i < ident; i++ {
		s += "  "
	}
	if n.isFolder {
		s += fmt.Sprintf("{F} %s [%s] [%x]\n", n.name, n.rname, must(n.ETag())[:4])
		for _, c := range n.children {
			s += c.StringIdent(ident + 1)
		}
	} else {
		s += fmt.Sprintf("{D} %s (%s, %d) [%s -> %s] [%x]\n", n.name, n.mime, n.length, n.rname, n.sname, must(n.ETag())[:4])
	}
	return
}
