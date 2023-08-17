package etag

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"
)

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("failed to read hostname: %v", err)
	}
}

// ETag is a short and unique identifier assigned to a specific version of a
// remoteStorage resource.
type ETag []byte

type Node interface {
	Reader() (io.ReadCloser, error)
	Name() string
	Length() int64
	Mime() string
	LastMod() time.Time
	IsFolder() bool
	Children() []Node
}

var hostname string

// String creates a string from an Etag e.
// To go the opposite way an obtain an ETag from a string, use ParseETag.
func (e ETag) String() string {
	return hex.EncodeToString(e)
}

// ParseETag decodes an ETag previously encoded by (ETag).String()
func ParseETag(s string) (ETag, error) {
	if len(s) != md5.Size*2 {
		return nil, fmt.Errorf("not a valid etag")
	}
	return hex.DecodeString(s)
}

func (e ETag) Equal(other ETag) bool {
	le := len(e)
	lo := len(other)
	if le != lo {
		return false
	}
	for i := 0; i < le; i++ {
		if e[i] != other[i] {
			return false
		}
	}
	return true
}

func CalculateETag(n Node) (ETag, error) {
	hash := md5.New()
	io.WriteString(hash, hostname)

	ns := []Node{n}
	for len(ns) > 0 {
		cn := ns[0]
		ns = ns[1:]

		if cn.IsFolder() {
			io.WriteString(hash, cn.Name())
			children := cn.Children()
			// Ensure that etag is deterministic by always hashing children in
			// the same order.
			sort.Slice(children, func(i, j int) bool {
				return children[i].Name() < children[j].Name()
			})
			ns = append(ns, children...)
		} else {
			io.WriteString(hash, cn.Name())
			io.WriteString(hash, cn.Mime())
			io.WriteString(hash, cn.LastMod().Format(time.RFC1123))

			fd, err := cn.Reader()
			if err != nil {
				return nil, err
			}
			n, err := io.Copy(hash, fd)
			if err != nil {
				return nil, err
			}
			if cn.Length() != int64(n) {
				return nil, fmt.Errorf("etag: expected to read %d bytes, got: %d", cn.Length(), n)
			}
			err = fd.Close()
			if err != nil {
				return nil, err
			}
		}
	}

	return hash.Sum(nil), nil
}
