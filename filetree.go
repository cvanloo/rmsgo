package rmsgo

import "time"

var root string

type Node struct {
	isDir        bool
	name         string
	path         string
	size         uint
	etag         []byte
	lastModified time.Time
	children     []*Node
}

func AddPath(path string) *Node {
	panic("not implemented")
}

// Resolve returns the full path of where the node is stored on the server's
// file system.
func (n Node) Resolve() string {
	panic("not implemented")
}

func (n Node) Description() map[string]any {
	panic("not implemented")
}
