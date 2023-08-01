package rmsgo

import (
	"errors"
	"path/filepath"
	"time"
)

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
	children map[string]*node
}

var (
	files         map[string]*node
	ErrFileExists = errors.New("file already exists")
)

func updateDocument(rname string, data []byte, mime string, replaceIfExists bool) (*node, error) {
	//f, ok := files[rname]
	//if ok && replaceIfExists {
	//	return nil, ErrFileExists
	//}

	sname := "" // @todo: how do we get from the rname to the sname?
	err := mfs.WriteFile(sname, data, 0640)
	if err != nil {
		return nil, err
	}

	f := &node{
		parent:   &node{}, // @todo
		isFolder: false,
		name:     filepath.Base(rname),
		rname:    rname,
		sname:    sname,
		etag:     nil, // @todo: we could make it an ETag() getter, re-calculate the etag every time we retrieve it, to make sure it is up to date
		mime:     mime,
		length:   int64(len(data)),
		lastMod:  time.Now(),
	}
	etag, err := generateETag(f)
	if err != nil {
		return f, err
	}
	f.etag = etag
	// @todo: create ancestors as necessary
	// @todo: update etag(s) of parent(s)
	return f, nil
}

func removeDocument(rname string) (*node, error) {
	if f, ok := files[rname]; ok {
		if f.isFolder {
			panic("assertion !f.isFolder failed")
		}

		mfs.Remove(f.sname)
		p := f.parent // @fixme: could parent be nil (eg at the fs root)?
		delete(p.children, f.rname)

		for p != nil {
			if len(p.children) == 0 {
				mfs.Remove(p.sname)
				pp := p.parent // @fixme: could parent be nil (eg at the fs root)?
				delete(pp.children, p.rname)
				p = pp
			} else {
				p = nil
			}
		}
		// @todo: update etag(s) of parent(s)
		return f, nil
	}
	return nil, ErrNotFound
}

func getNode(rname string) (*node, error) {
	if f, ok := files[rname]; ok {
		return f, nil
	}
	return nil, ErrNotFound
}
