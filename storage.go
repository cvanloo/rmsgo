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

	etag     ETag
	mime     string
	length   int64
	lastMod  time.Time
	children map[string]*node
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
	return s.root
}

func (s Storage) CreateDocument(cfg Server, rname string, data []byte, mime string) (*node, error) {
	if f, ok := s.files[rname]; ok {
		return f, ErrFileExists
	}

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
	if pname != "/" {
		parts = strings.Split(pname, string(os.PathSeparator))[1:] // don't include "" before first "/"
	}
	p := s.files["/"]
	assert(p != nil, "/ (root) exists")
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
		e, err := generateETag(n) // @perf(#etag)
		if err != nil {
			return f, err
		}
		n.etag = e
		n = n.parent
	}

	return f, nil
}

func (s Storage) UpdateDocument(cfg Server, rname string, data []byte, mime string) (*node, error) {
	f, ok := s.files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	err := mfs.WriteFile(f.sname, data, 0640)
	if err != nil {
		return f, err
	}

	f.mime = mime
	f.length = int64(len(data))
	f.lastMod = time.Now()

	n := f
	for n != nil {
		e, err := generateETag(n) // @perf(#etag)
		if err != nil {
			return n, err
		}
		n.etag = e
		n = n.parent
	}

	return f, nil
}

func (s Storage) RemoveDocument(cfg Server, rname string) (*node, error) {
	if f, ok := s.files[rname]; ok {
		assert(!f.isFolder, "removeDocument must not be called on a folder")
		p := f
		for len(p.children) == 0 && p != s.root {
			mfs.Remove(p.sname)
			pp := p.parent
			delete(pp.children, p.rname)
			delete(s.files, p.rname)
			p = pp
		}
		// p now points to the parent deepest down the ancestry that is not empty

		// @perf(#etag): maybe don't do the re-calculation here, only mark the etags as invalid
		//   then use a getter ETag() that re-calculates when the invalid flag is set.
		for p != nil {
			e, err := generateETag(p)
			if err != nil {
				return f, err
			}
			p.etag = e
			p = p.parent
		}

		return f, nil
	}
	return nil, ErrNotFound
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
		s += fmt.Sprintf("{F} %s [%s] [%x]\n", n.name, n.rname, n.etag[:4])
		for _, c := range n.children {
			s += c.StringIdent(ident + 1)
		}
	} else {
		s += fmt.Sprintf("{D} %s (%s, %d) [%s -> %s] [%x]\n", n.name, n.mime, n.length, n.rname, n.sname, n.etag[:4])
	}
	return
}
