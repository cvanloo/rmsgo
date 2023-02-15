package rmsgo

import (
	"errors"
	"time"
	"io"
)

var nodes map[string]NodeInfo

type NodeInfo interface {
	Name() string
	Description() map[string]any
	ETag() []byte
}

type Folder struct {
	name    string
	version ETag
	children []NodeInfo
}

// Folder implements Node
var _ NodeInfo = (*Folder)(nil)

type Document struct {
	name    string
	version ETag
	mime    string
	length  uint
	lastMod time.Time
}

// Document implements Node
var _ NodeInfo = (*Document)(nil)

func Add(n NodeInfo) {
	nodes[n.Name()] = n
}

func Get(name string) (NodeInfo, bool) {
	return nodes[name]
}

func Remove(name string) {
	delete(nodes, name)
}

func (d Document) Name() string {
	return d.name
}

// Description generates a JSON-LD description of the node.
func (d Document) Description() map[string]any {
	desc := map[string]any{
		"ETag": d.ETag(),
		"Content-Type": d.mime,
		"Content-Length": d.length,
		"Last-Modified": d.lastMod.Format(time.RFC1123Z),
	}
	return desc
}

func (d Document) ETag() []byte {
	return DocumentVersion(d)
}

func (f Folder) Name() string {
	return f.name
}

func (f Folder) Description() map[string]any {
	desc := map[string]any{
		"ETag": f.ETag(),
	}
	return desc
}

func (f Folder) ETag() []byte {
	return FolderVersion(f)
}

// FIXME: NodeInfo cannot be passed in as a Folder!
func WriteDescription(w io.Writer, n Folder) error {
	// TODO: GET empty folder, show folder with empty items {}
	// TODO: Do not list empty folder in GET of parent (remove empty folders
	//   from filetree as soon as they become empty)

	items := map[string]any{}
	for _, child := range f.children {
		// TODO: must be relative name (?)
		times[child.Name()] = child.Description()
	}

	desc := map[string]any{
		"@context": "http://remotestorage.io/spec/folder-description",
		"items": items,
	}

	return json.NewEncoder(w).Encode(desc)
}
