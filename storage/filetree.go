package storage

import (
	"encoding/json"
	"io"
	"time"
)

var (
	rootNode Folder
	nodes    map[string]NodeInfo
)

func init() {
	rootF := &folder{
		parent:   nil,
		name:     "/",
		children: make(map[string]NodeInfo),
	}
	// root's parent is itself, to avoid any errors trying to go up even further.
	// FIXME: or is it better to just return an error/leave root's parent as nil?
	rootF.parent = rootF
	rootNode = rootF

	nodes = make(map[string]NodeInfo)
	nodes[rootNode.Name()] = rootNode
}

// NodeInfo represents a folder or document.
type NodeInfo interface {
	// Parent returns the Folder in which the node is contained.
	Parent() Folder
	// Name returns the name of the node.
	Name() string
	// Description produces a ld+json-like map describing the node.
	Description() map[string]any
	// Version calculates the node's etag.
	Version() ETag
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

type Folder interface {
	NodeInfo
	Children() map[string]NodeInfo
}

type Document interface {
	NodeInfo
	Mime() string
	Length() uint64
	LastMod() time.Time
	Reader() (io.Reader, error)
}

/*// Add a document to the tree.
// Any necessary ancestor directories are created automatically.
func Add(doc Document) {
	var lmp Folder
	cPath := doc.name
	for {
		pPath := filepath.Dir(cPath)
		if pPath == "." {
			pPath = "/"
		}
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

	pPath := filepath.Dir(doc.name)
	if pPath == "." {
		pPath = "/"
	}
	pn, ok := nodes[pPath]
	if !ok {
		panic("expected parent node to have been created")
	}
	pf := pn.Folder()
	pf.children[doc.name] = doc
	doc.parent = &pf
	nodes[doc.Name()] = doc
}
*/

func WriteDescription(w io.Writer, f Folder) error {
	items := map[string]any{}
	for _, child := range f.Children() {
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
