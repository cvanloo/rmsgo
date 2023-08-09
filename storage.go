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
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

func init() {
	Reset()
	if isdelve.Enabled {
		createUUID = CreateMockUUIDFunc()
		getTime = getMockTime
	}
}

var (
	files map[string]*Node
	root  *Node

	createUUID = uuid.NewRandom
	getTime    = time.Now

	ErrFileExists = errors.New("file already exists") // @todo: remove error?
)

// @todo: create separate DTO for (de)serialization
type Node struct {
	parent   *Node
	IsFolder bool `xml:"IsFolder,attr"`

	// "Kittens.png"
	Name string

	// "/Pictures/Kittens.png"
	Rname string

	// "/var/rms/storage/(uuid)"
	Sname string `xml:"Sname,omitempty"`

	ETag      ETag `xml:"ETag,omitempty"`
	etagValid bool

	Mime     string
	Length   int64      `xml:"Length,omitempty"`
	LastMod  *time.Time `xml:"LastMod,omitempty"`
	children map[string]*Node
}

func (n *Node) Valid() bool {
	return n.etagValid
}

func (n *Node) Invalidate() {
	n.etagValid = false
}

func (n *Node) Version() (e ETag, err error) {
	if !n.etagValid {
		err = calculateETag(n)
	}
	e = n.ETag
	return
}

func (n *Node) Equal(other *Node) bool {
	if !(n.etagValid && other.etagValid) {
		return false
	}
	return n.ETag.Equal(other.ETag)
}

func (n *Node) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type XMLNode struct {
		Node
		ParentRName string `xml:"ParentRName"`
	}
	if n == root {
		return nil
	}
	if n.parent == nil {
		return fmt.Errorf("node %s is missing its parent", n.Rname)
	}
	return e.EncodeElement(XMLNode{*n, n.parent.Rname}, start)
}

func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var tmp struct {
		IsFolder    bool `xml:"IsFolder,attr"`
		Name        string
		Rname       string
		Sname       string `xml:"Sname,omitempty"`
		ETag        ETag   `xml:"ETag,omitempty"`
		Mime        string
		Length      int64      `xml:"Length,omitempty"`
		LastMod     *time.Time `xml:"LastMod,omitempty"`
		ParentRName string
	}
	err := d.DecodeElement(&tmp, &start)
	if err != nil {
		return err
	}

	n.IsFolder = tmp.IsFolder
	n.Name = tmp.Name
	n.Rname = tmp.Rname
	n.Sname = tmp.Sname
	n.ETag = tmp.ETag
	n.Mime = tmp.Mime
	n.Length = tmp.Length
	n.LastMod = tmp.LastMod
	n.children = make(map[string]*Node)

	// N.b. this assumes that parents are always parsed before their
	// children! [#parent_first]
	p, ok := files[tmp.ParentRName]
	if !ok {
		return fmt.Errorf("node %s is missing its parent (%s), maybe it hasn't been parsed yet?", n.Rname, tmp.ParentRName)
	}
	p.children[n.Rname] = n
	n.parent = p
	files[n.Rname] = n
	return nil
}

func Reset() {
	rn := &Node{
		IsFolder: true,
		Name:     "/",
		Rname:    "/",
		Mime:     "inode/directory",
		children: map[string]*Node{},
	}
	files = make(map[string]*Node)
	files["/"] = rn
	root = rn
}

func Persist(persistFile file) (err error) {
	files := maps.Values(files)
	// Ensure that parents are always serialized before their children, so that
	// they will also be read in first. [#parent_first]
	sort.Slice(files, func(i, j int) bool {
		// Alphabetically, a shorter word is sorted before a longer.
		// The parent's path will always be shorter than the child's path.
		return files[i].Rname < files[j].Rname
	})
	type Root struct {
		Nodes []*Node
	}
	var (
		bs      []byte
		persist = Root{files}
	)
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

func Load(persistFile file) error {
	if root == nil {
		return fmt.Errorf("storage root not initialized, try calling Reset() before Load()")
	}

	bs, err := io.ReadAll(persistFile)
	if err != nil {
		return err
	}

	var persist struct {
		Nodes []*Node
	}
	err = xml.Unmarshal(bs, &persist)
	if err != nil {
		return err
	}

	log.Printf("Storage listing follows:\n%s\n", root)
	return nil
}

// Migrate traverses the root directory and copies any files contained therein
// into the remoteStorage root (cfg.Sroot).
func Migrate(cfg Server, root string) (errs []error) {
	//root = filepath.Clean(root)
	err := mfs.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, fmt.Errorf("error encountered, skipping directory: %v", err))
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		fd, err := mfs.Open(path)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		sname, fsize, mime, err := WriteFile(cfg, "", fd)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		rname := strings.TrimPrefix(path, root[:len(root)-1])
		AddDocument(rname, sname, fsize, mime)
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

func WriteFile(cfg Server, sname string, data io.Reader) (nsname string, fsize int64, detectedMime string, err error) {
	if sname == "" {
		u, err := createUUID()
		if err != nil {
			return "", 0, "", err
		}
		nsname = filepath.Join(cfg.sroot, u.String())
	} else {
		nsname = sname
	}

	fd, err := mfs.Create(nsname) // @todo: set permissions
	if err != nil {
		return nsname, 0, "", err
	}

	fsize, err = io.Copy(fd, data)
	if err != nil {
		return nsname, fsize, "", err
	}

	fd.Seek(0, io.SeekStart)
	mime, err := mimetype.DetectReader(fd)
	if err != nil {
		return nsname, fsize, mime.String(), err
	}

	return nsname, fsize, mime.String(), nil
}

func DeleteDocument(sname string) error {
	return mfs.Remove(sname)
}

func AddDocument(rname, sname string, fsize int64, mime string) (*Node, error) {
	if f, ok := files[rname]; ok {
		return f, ErrFileExists
	}

	assert(rname[len(rname)-1] != '/', "AddDocument must only be used to create files")

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
			pn = &Node{
				parent:   p,
				IsFolder: true,
				Name:     parts[i] + "/", // folder names must end in a slash
				Rname:    pname,
				Mime:     "inode/directory",
				children: map[string]*Node{},
			}
			p.children[pname] = pn
			files[pname] = pn
		}
		p = pn
	}
	// p now points to the file's immediate parent [#1]

	name := filepath.Base(rname)
	tnow := getTime()

	f := &Node{
		parent:   p, // [#1] assign parent
		IsFolder: false,
		Name:     name,
		Rname:    rname,
		Sname:    sname,
		Mime:     mime,
		Length:   fsize,
		LastMod:  &tnow,
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

func UpdateDocument(rname string, fsize int64, mime string) (*Node, error) {
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.IsFolder, "UpdateDocument must not be called on a folder")

	tnow := getTime()
	f.Mime = mime
	f.Length = int64(fsize)
	f.LastMod = &tnow

	n := f
	for n != nil {
		n.Invalidate()
		n = n.parent
	}

	return f, nil
}

func RemoveDocument(rname string) (*Node, error) {
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotFound
	}

	assert(!f.IsFolder, "RemoveDocument must not be called on a folder")

	p := f
	for len(p.children) == 0 && p != root {
		pp := p.parent
		delete(pp.children, p.Rname)
		delete(files, p.Rname)
		p = pp
	}
	// p now points to the parent deepest down the ancestry that is not empty

	for p != nil {
		p.Invalidate()
		p = p.parent
	}

	err := mfs.Remove(f.Sname)
	return f, err
}

func Retrieve(rname string) (*Node, error) {
	if f, ok := files[rname]; ok {
		return f, nil
	}
	return nil, ErrNotFound // @todo: wrap with rname (WHAT resource was not found?)
}

func (n Node) String() string {
	return n.StringIdent(0)
}

func (n Node) StringIdent(ident int) (s string) {
	for i := 0; i < ident; i++ {
		s += "  "
	}
	if n.IsFolder {
		s += fmt.Sprintf("{F} %s [%s] [%x]\n", n.Name, n.Rname, must(n.Version())[:4])
		children := maps.Values(n.children)
		// Ensure that output is deterministic by always printing in the same
		// order. (Exmaple functions need this to verify their output.)
		sort.Slice(children, func(i, j int) bool {
			return children[i].Rname < children[j].Rname
		})
		for _, c := range children {
			s += c.StringIdent(ident + 1)
		}
	} else {
		s += fmt.Sprintf("{D} %s (%s, %d) [%s -> %s] [%x]\n", n.Name, n.Mime, n.Length, n.Rname, n.Sname, must(n.Version())[:4])
	}
	return
}
