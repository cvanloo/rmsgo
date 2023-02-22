package storage

import (
	"io"
	"os"
	"time"
)

type document struct {
	parent  Folder
	name    string
	version ETag
	mime    string
	length  uint64
	lastMod time.Time
}

// document implements Node
var _ Document = (*document)(nil)

func (document) IsFolder() bool {
	return false
}

func (document) Folder() Folder {
	panic("a document is not a folder")
}

func (d document) Document() Document {
	return d
}

func (d document) Parent() Folder {
	return d.parent
}

func (d document) Name() string {
	return d.name
}

func (d document) Description() map[string]any {
	desc := map[string]any{
		"ETag":           d.Version(),
		"Content-Type":   d.mime,
		"Content-Length": d.length,
		"Last-Modified":  d.lastMod.Format(time.RFC1123Z),
	}
	return desc
}

func (d document) Version() ETag {
	return d.version
}

func (d document) Mime() string {
	return d.mime
}

func (d document) Length() uint64 {
	return d.length
}

func (d document) LastMod() time.Time {
	return d.lastMod
}

func (d document) Reader() (io.Reader, error) {
	fi, err := os.Open(Resolve(d))
	if err != nil {
		return nil, err
	}
	return fi, nil
}
