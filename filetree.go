package rmsgo

import (
	"errors"
	"time"
)

var (
	root  string
	nodes map[string]NodeInfo
)

type NodeInfo interface {
	Name() string
	Resolve() string
	Description() map[string]any
}

type folder struct {
	name    string
	version ETag
	children []NodeInfo
}

// Folder implements Node
var _ NodeInfo = (*folder)(nil)

type document struct {
	name    string
	version ETag
	mime    string
	length  uint
	lastMod time.Time
}

// Document implements Node
var _ NodeInfo = (*document)(nil)

func Add(n NodeInfo) {
	//n := &Document{
	//	Name:    "/user/kittens.png",
	//	Version: []byte{},
	//	Mime:    "",
	//	Length:  uint(0),
	//	LastMod: time.Time{},
	//}

	//n := &Folder{
	//	Name:     "/user/Documents/",
	//	Version:  []byte{},
	//	Children: []*Node{},
	//}

	nodes[n.Name()] = n
}

var NodeNotFound = errors.New("node not found")

func Get(name string) (NodeInfo, error) {
	for k, v := range nodes {
		if k == name {
			return v, nil
		}
	}
	return nil, NodeNotFound
}

func (d document) Name() string {
	panic("not implemented")
}

func (d document) Resolve() string {
	panic("not implemented")
}

// Description generates a JSON-LD description of the node.
func (d document) Description() map[string]any {
	// Content-Type: application/ld+json

	//  ETag: string
	//  Content-Type: string
	//  Content-Length: int (in octects)
	//  Last-Modified: string (HTTP date)

	panic("not implemented")
}

func (f folder) Name() string {
	panic("not implemented")
}

func (f folder) Resolve() string {
	panic("not implemented")
}

func (f folder) Description() map[string]any {
	//  ETag: string
	//
	// @context: http://remotestorage.io/spec/folder-description
	// items: {
	//   folder and document descriptions
	// }
	//
	// GET empty folder, show folder with empty items {}
	// Do not list empty folder in GET of parent
	panic("not implemented")
}
