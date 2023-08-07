package rmsgo

import (
	"errors"
	"fmt"
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

type Storage struct {
	files map[string]*node
	root  *node
}

var ErrFileExists = errors.New("file already exists")

func NewStorage() (s Storage) {
	rn := &node{
		isFolder: true,
		name:     "/",
		rname:    "/",
		mime:     "inode/directory",
		children: map[string]*node{},
	}
	s.files = make(map[string]*node)
	s.files["/"] = rn
	s.root = rn
	return
}

func (s Storage) Root() *node {
	assert(s.root != nil, "/ (root) exists")
	return s.root
}

func (s Storage) CreateDocument(cfg Server, rname string, data []byte, mime string) (*node, error) {
	if f, ok := s.files[rname]; ok {
		return f, ErrFileExists
	}

	assert(rname[len(rname)-1] != '/', "CreateDocument must only be used to create files")

	u, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	sname := filepath.Join(cfg.Sroot, u.String())
	err = mfs.WriteFile(sname, data, 0640)
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

	p := s.root

	for i := range parts {
		pname := "/" + strings.Join(parts[:i+1], string(os.PathSeparator)) + "/"
		pn, ok := s.files[pname]
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
			s.files[pname] = pn
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
		length:   int64(len(data)),
		lastMod:  time.Now(),
	}
	p.children[rname] = f
	s.files[rname] = f

	n := f
	for n != nil {
		n.Invalidate()
		n = n.parent
	}

	return f, nil
}

func (s Storage) UpdateDocument(cfg Server, rname string, data []byte, mime string) (*node, error) {
	f, ok := s.files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.isFolder, "UpdateDocument must not be called on a folder")

	err := mfs.WriteFile(f.sname, data, 0640)
	if err != nil {
		return f, err
	}

	f.mime = mime
	f.length = int64(len(data))
	f.lastMod = time.Now()

	n := f
	for n != nil {
		n.Invalidate()
		n = n.parent
	}

	return f, nil
}

func (s Storage) RemoveDocument(cfg Server, rname string) (*node, error) {
	f, ok := s.files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.isFolder, "RemoveDocument must not be called on a folder")

	p := f
	for len(p.children) == 0 && p != s.root {
		mfs.Remove(p.sname)
		pp := p.parent
		delete(pp.children, p.rname)
		delete(s.files, p.rname)
		p = pp
	}
	// p now points to the parent deepest down the ancestry that is not empty

	for p != nil {
		p.Invalidate()
		p = p.parent
	}

	return f, nil
}

func (s Storage) Node(cfg Server, rname string) (*node, error) {
	if f, ok := s.files[rname]; ok {
		return f, nil
	}
	return nil, ErrNotFound
}

func (s Storage) String() string {
	return s.root.StringIdent(0)
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
