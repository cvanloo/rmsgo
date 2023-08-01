package rmsgo

import "time"

// @todo: https://datatracker.ietf.org/doc/html/draft-dejong-remotestorage-21#section-3
// storage.go
// foreach node:
// - item name
// - item type (folder/document)
// - etag
// foreach document additionally:
// - mime type
// - length
// - last modified

type node struct {
	parent   *node
	isFolder bool

	// "Kittens.png"
	name string

	// "/Pictures/Kittens.png"
	rname string

	// @todo: how to store it?
	// "/var/rms/storage/Pictures/Kittens.png"
	// "/var/rms/storage/(uuid)"
	sname string

	etag ETag

	// document only properties:
	mime    string
	length  int64
	lastMod time.Time

	// folder only properties:
	children []*node
}

var files map[string]*node

func updateDocument(rname string, data []byte, mime string) (*node, error) {
	// auto-create ancestor folders as necessary
	return nil, ErrNotImplemented
}

func removeDocument(rname string) (*node, error) {
	// auto-remove empty ancestor folders
	return nil, ErrNotImplemented
}

func getDocument(rname string) (*node, error) {
	if f, ok := files[rname]; ok {
		return f, nil
	}
	return nil, ErrNotFound
}
