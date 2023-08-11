package rmsgo

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cvanloo/rmsgo.git/isdelve"
	. "github.com/cvanloo/rmsgo.git/mock"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

func init() {
	if !isdelve.Enabled {
		UUID = uuid.NewRandom
		Time = time.Now
	}
	Reset()
}

type ConflictError struct {
	Path         string
	ConflictPath string
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("%s: conflicts with already existing path: %s", e.Path, e.ConflictPath)
}

var ErrNotExist = errors.New("no such document or folder")

var (
	// files keeps a reference for each document or folder, allowing for easy
	// access.
	files map[string]*node

	// root keeps a reference to the root folder.
	// The reference will stay valid for the entire duration of execution once
	// Reset has been called.
	root  *node
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
	lastMod  *time.Time
	children map[string]*node
}

func (n *node) Valid() bool {
	return n.etagValid
}

func (n *node) Invalidate() {
	n.etagValid = false
}

func (n *node) Version() (e ETag, err error) {
	if !n.etagValid {
		err = calculateETag(n)
	}
	e = n.etag
	return
}

func (n *node) Equal(other *node) bool {
	if !(n.etagValid && other.etagValid) {
		return false
	}
	return n.etag.Equal(other.etag)
}

// Reset (re-) initializes the storage tree, so that it only contains a root folder.
func Reset() {
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

type NodeDTO struct {
	IsFolder    bool `xml:"IsFolder,attr"`
	Name        string
	Rname       string
	Sname       string `xml:"Sname,omitempty"`
	ETag        string
	Mime        string
	Length      int64      `xml:"Length,omitempty"`
	LastMod     *time.Time `xml:"LastMod,omitempty"`
	ParentRName string
}

// Persist serializes the storage tree to XML.
// The generated XML is written to persistFile.
func Persist(persistFile io.Writer) (err error) {
	fileDTOs := []*NodeDTO{}
	for _, n := range files {
		if n != root {
			etag, err := n.Version()
			if err != nil {
				return err
			}
			dto := &NodeDTO{
				IsFolder:    n.isFolder,
				Name:        n.name,
				Rname:       n.rname,
				Sname:       n.sname,
				ETag:        etag.String(),
				Mime:        n.mime,
				Length:      n.length,
				LastMod:     n.lastMod,
				ParentRName: n.parent.rname,
			}
			fileDTOs = append(fileDTOs, dto)
		}
	}

	// Ensure that parents are always serialized before their children, so that
	// they will also be read in first. [#parent_first]
	sort.Slice(fileDTOs, func(i, j int) bool {
		// Alphabetically, a shorter word is sorted before a longer.
		// The parent's path will always be shorter than the child's path.
		return fileDTOs[i].Rname < fileDTOs[j].Rname
	})

	type Root struct {
		Nodes []*NodeDTO
	}
	persist := Root{fileDTOs}

	var bs []byte
	if isdelve.Enabled {
		bs, err = xml.MarshalIndent(persist, "", "\t")
	} else {
		bs, err = xml.Marshal(persist)
	}
	if err != nil {
		return err
	}
	_, err = persistFile.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

// Load deserializes XML data from persistFile and adds the documents and
// folders to the storage tree.
// If storage has not been initialized before, Reset must be invoked before
// calling Load.
func Load(persistFile io.Reader) error {
	if root == nil {
		return fmt.Errorf("storage root not initialized, try calling Reset() before Load()")
	}

	bs, err := io.ReadAll(persistFile)
	if err != nil {
		return err
	}

	var persist struct {
		Nodes []*NodeDTO
	}
	err = xml.Unmarshal(bs, &persist)
	if err != nil {
		return err
	}

	for _, n := range persist.Nodes {
		etag, err := ParseETag(n.ETag)
		if err != nil {
			return err
		}
		model := &node{}
		model.isFolder = n.IsFolder
		model.name = n.Name
		model.rname = n.Rname
		model.sname = n.Sname
		model.etag = etag
		model.mime = n.Mime
		model.length = n.Length
		model.lastMod = n.LastMod
		model.children = make(map[string]*node)

		// N.b. this assumes that parents are always parsed before their
		// children! [#parent_first]
		p, ok := files[n.ParentRName]
		if !ok {
			return fmt.Errorf("node %s is missing its parent (%s), maybe it hasn't been parsed yet?", model.rname, n.ParentRName)
		}
		model.parent = p
		p.children[model.rname] = model
		files[model.rname] = model
	}

	log.Printf("Storage listing follows:\n%s\n", root)
	return nil
}

// Migrate traverses the root directory and copies any files contained therein
// into the remoteStorage root (cfg.Sroot).
func Migrate(root string) (errs []error) {
	err := FS.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fd, err := FS.Open(path)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		defer func() {
			err := fd.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}()

		u, err := UUID()
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		sname := filepath.Join(sroot, u.String())

		rmsFD, err := FS.Create(sname)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		defer func() {
			err := rmsFD.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}()

		fsize, err := io.Copy(rmsFD, fd)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		mime, err := DetectMime(rmsFD)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		rname := strings.TrimPrefix(path, root[:len(root)-1])
		_, err = AddDocument(rname, sname, fsize, mime)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

// DetectMime rewinds fd, reads from fd to detect its mime type, and finally
// rewinds fd again to the start.
func DetectMime(fd File) (mime string, err error) {
	_, err = fd.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	m, err := mimetype.DetectReader(fd)
	if err != nil {
		return m.String(), err
	}

	_, err = fd.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	return m.String(), nil
}

// AddDocument adds a new document to the storage tree and returns a reference to it.
// ETags of ancestors are invalidated.
// If the document name conflicts with any other document or folder an error of
// type ConflictPath is returned and the *node is set to nil.
func AddDocument(rname, sname string, fsize int64, mime string) (*node, error) {
	rname = filepath.Clean(rname)

	{
		assert(rname[len(rname)-1] != '/', "AddDocument must only be used to create files")
		//_, ok := files[rname]
		//assert(!ok, "AddDocument must only be used to create files that don't exist yet")
	}

	if _, ok := files[rname]; ok {
		return nil, ConflictError{
			Path:         rname,
			ConflictPath: rname,
		}
	}

	var (
		pname = filepath.Dir(rname)
		parts = strings.Split(pname, string(os.PathSeparator))[1:] // exclude empty ""
		p     = root
	)

	for i := range parts { // traverse through the hierarchy, starting at the top most ancestor (excluding root)
		pname := "/" + strings.Join(parts[:i+1], string(os.PathSeparator))
		pn, ok := files[pname]
		if ok { // a document name clashes with one of the ancestor folders
			if !pn.isFolder {
				return nil, ConflictError{
					Path:         rname,
					ConflictPath: pname,
				}
			}
		} else { // from here on downwards ancestors don't exist, we have to create them
			pn = &node{
				parent:   p,
				isFolder: true,
				name:     parts[i] + "/", // folder names must end in a slash
				rname:    pname,
				mime:     "inode/directory",
				children: map[string]*node{},
			}
			p.children[pname] = pn
			files[pname] = pn
		}
		p = pn
	}
	// p now points to the document's immediate parent [#1]

	name := filepath.Base(rname)
	tnow := Time()

	f := &node{
		parent:   p, // [#1] assign parent
		isFolder: false,
		name:     name,
		rname:    rname,
		sname:    sname,
		mime:     mime,
		length:   fsize,
		lastMod:  &tnow,
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

// UpdateDocument updates an existing document in the storage tree with new
// information and invalidates etags of the document and its ancestors.
func UpdateDocument(n *node, mime string, fsize int64) {
	assert(!n.isFolder, "UpdateDocument must not be called on a folder")

	tnow := Time()
	n.mime = mime
	n.length = int64(fsize)
	n.lastMod = &tnow

	c := n
	for c != nil {
		c.Invalidate()
		c = c.parent
	}
}

// RemoveDocument deletes a document from the storage tree and invalidates the
// etags of its ancestors.
func RemoveDocument(n *node) {
	assert(!n.isFolder, "RemoveDocument must not be called on a folder")

	p := n
	for len(p.children) == 0 && p != root {
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
}

// Retrieve a document or folder identified by rname.
// Returns ErrNotExist if rname can't be found.
func Retrieve(rname string) (*node, error) {
	rname = filepath.Clean(rname)
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotExist
	}
	return f, nil
}

func (n node) String() string {
	return n.stringIndent(0)
}

func (n node) stringIndent(ident int) (s string) {
	for i := 0; i < ident; i++ {
		s += "  "
	}
	if n.isFolder {
		s += fmt.Sprintf("{F} %s [%s] [%x]\n", n.name, n.rname, mustVal(n.Version())[:4])
		children := maps.Values(n.children)
		// Ensure that output is deterministic by always printing in the same
		// order. (Exmaple functions need this to verify their output.)
		sort.Slice(children, func(i, j int) bool {
			return children[i].rname < children[j].rname
		})
		for _, c := range children {
			s += c.stringIndent(ident + 1)
		}
	} else {
		s += fmt.Sprintf("{D} %s (%s, %d) [%s -> %s] [%x]\n", n.name, n.mime, n.length, n.rname, n.sname, mustVal(n.Version())[:4])
	}
	return
}
