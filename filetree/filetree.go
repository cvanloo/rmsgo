package filetree

import (
	"encoding/json"
	"io"
	"path/filepath"
	"time"
)

// TODO: how do we represent the root?
//var root *Folder = &Folder{
//	parent: nil, // ?
//	name: "/",
//}

// NodeInfo represents a folder or document.
type NodeInfo interface {
	// Parent returns the Folder in which the node is contained.
	Parent() *Folder
	// Name returns the name of the node.
	Name() string
	// Description produces a ld+json-like map describing the node.
	Description() map[string]any
	// Version calculates the node's etag.
	Version() []byte
	// IsFolder returns true only when the node is of type Folder;
	// false if it is a Document.
	IsFolder() bool
	// Document casts the node as a Document.
	// This method will panic if the node is a Folder.
	// Call IsFolder first to verify the type.
	Document() Document
	// Folder casts the node as a Folder.
	// This method will panic if the node is a Document.
	// Call IsFolder first to verify the type.
	Folder() Folder
}

type Folder struct {
	parent   *Folder
	name     string
	version  []byte
	children map[string]NodeInfo
}

// Folder implements Node
var _ NodeInfo = (*Folder)(nil)

type Document struct {
	parent  *Folder
	name    string
	version []byte
	Mime    string
	Length  uint
	LastMod time.Time
}

// Document implements Node
var _ NodeInfo = (*Document)(nil)

var (
	root                 *Folder
	nodes                map[string]NodeInfo
	storageRoot, webRoot string
)

func init() {
	root = &Folder{
		parent: nil,
		name:   "/",
	}
	root.parent = root
	nodes = make(map[string]NodeInfo)
	nodes[root.name] = root
}

// NewDocument creates a new document node.
// If mime is left empty, it will be detected based on the file's contents.
func NewDocument(name, mime string, length uint, lastMod time.Time) Document {
	return Document{
		name:    name,
		Mime:    mime,
		Length:  length,
		LastMod: lastMod,
	}
}

// Add a document to the tree.
// Any necessary ancestor directories are created automatically.
func Add(doc Document) {
	var lmp *Folder
	cPath := doc.name
	for {
		pPath := filepath.Dir(cPath)
		pn, ok := nodes[pPath]
		if !ok {
			nf := &Folder{
				parent:   nil,
				name:     pPath,
				children: make(map[string]NodeInfo),
			}
			if lmp != nil {
				nf.children[lmp.name] = lmp
				lmp.parent = nf
			}
			nodes[nf.name] = nf
			lmp = nf
			cPath = pPath
		} else {
			pf := pn.Folder()
			if lmp != nil {
				lmp.parent = &pf
				pf.children[lmp.name] = lmp
			}
			break
		}
	}

	pn, ok := nodes[filepath.Dir(doc.name)]
	if !ok {
		panic("expected parent node to have been created")
	}
	pf := pn.Folder()
	pf.children[doc.name] = doc
	doc.parent = &pf
	nodes[doc.Name()] = doc
}

// Get a node from its name.
func Get(name string) (NodeInfo, bool) {
	n, ok := nodes[name]
	return n, ok
}

// Remove a node from the tree.
// If the node's parent is left empty, it is removed as well, and its parent,
// and so on...
func Remove(name string) {
	n, ok := nodes[name]
	if !ok {
		return
	}

	for {
		delete(nodes, n.Name())
		p := n.Parent()
		delete(p.children, n.Name())
		if len(p.children) == 0 {
			// delete empty parent
			n = p
		} else {
			break
		}
	}
}

func (d Document) IsFolder() bool {
	return false
}

func (d Document) Parent() *Folder {
	return d.parent
}

func (d Document) Document() Document {
	return d
}

func (d Document) Folder() Folder {
	panic("document is not a folder")
}

func (d Document) Name() string {
	return d.name
}

func (d Document) Description() map[string]any {
	desc := map[string]any{
		"ETag":           d.Version(),
		"Content-Type":   d.Mime,
		"Content-Length": d.Length,
		"Last-Modified":  d.LastMod.Format(time.RFC1123Z),
	}
	return desc
}

func (d Document) Version() []byte {
	return d.version
}

func (f Folder) IsFolder() bool {
	return true
}

func (f Folder) Folder() Folder {
	return f
}

func (f Folder) Document() Document {
	panic("folder is not a document")
}

func (f Folder) Parent() *Folder {
	return f.parent
}

func (f Folder) Name() string {
	return f.name
}

func (f Folder) Description() map[string]any {
	desc := map[string]any{
		"ETag": f.Version(),
	}
	return desc
}

func (f Folder) Version() []byte {
	return f.version
}

func WriteDescription(w io.Writer, f Folder) error {
	items := map[string]any{}
	for _, child := range f.children {
		// TODO: must be relative name (?)
		//   Maybe we should already store just the relative name in the node.name?
		items[child.Name()] = child.Description()
	}

	desc := map[string]any{
		"@context": "http://remotestorage.io/spec/folder-description",
		"items":    items,
	}

	return json.NewEncoder(w).Encode(desc)
}
