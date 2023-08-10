package rmsgo

import (
	"encoding/xml"
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

var (
	files map[string]*Node
	root  *Node
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

func Persist(persistFile io.Writer) (err error) {
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

func Load(persistFile io.Reader) error {
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
		sname := filepath.Join(cfg.sroot, u.String())

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

func DeleteDocument(sname string) error {
	return FS.Remove(sname)
}

func AddDocument(rname, sname string, fsize int64, mime string) (*Node, error) {
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
			if !pn.IsFolder {
				return nil, ConflictError{
					Path:         rname,
					ConflictPath: pname,
				}
			}
		} else { // from here on downwards ancestors don't exist, we have to create them
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
	// p now points to the document's immediate parent [#1]

	name := filepath.Base(rname)
	tnow := Time()

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

func UpdateDocument(n *Node, mime string, fsize int64) {
	assert(!n.IsFolder, "UpdateDocument must not be called on a folder")

	tnow := Time()
	n.Mime = mime
	n.Length = int64(fsize)
	n.LastMod = &tnow

	c := n
	for c != nil {
		c.Invalidate()
		c = c.parent
	}
}

func RemoveDocument(n *Node) {
	assert(!n.IsFolder, "RemoveDocument must not be called on a folder")

	p := n
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
}

func Retrieve(rname string) (*Node, error) {
	rname = filepath.Clean(rname)
	f, ok := files[rname]
	if !ok {
		return nil, ErrNotExist
	}
	return f, nil
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
