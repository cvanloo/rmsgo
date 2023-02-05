package rmsgo

import (
	"errors"
	"time"
)

var (
	root string
	nodes map[string]*Node
)

type NodeInfo interface {
	Name() string
	Resolve() string
	Description() map[string]any
}

type Node struct {
	Name string
	Version ETag
}

type Folder struct {
	Node
	Children []*Node
}

// Folder implements Node
var _ NodeInfo = (*Folder)(nil)

type Document struct {
	Node
	Mime string
	Length uint
	LastMod time.Time
}

// Document implements Node
var _ NodeInfo = (*Document)(nil)

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

// Description generates a JSON-LD description of the node.
func (d Document) Description() map[string]any {
	// Content-Type: application/ld+json

	//  ETag: string
	//  Content-Type: string
	//  Content-Length: int (in octects)
	//  Last-Modified: string (HTTP date)

	panic("not implemented")
}

func (f Folder) Description() map[string]any {
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
