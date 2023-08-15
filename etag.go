package rmsgo

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	. "github.com/cvanloo/rmsgo/mock"
	"golang.org/x/exp/maps"
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

func calculateETag(n *node) error {
	hash := md5.New()
	io.WriteString(hash, hostname)

	ns := []*node{n}
	for len(ns) > 0 {
		cn := ns[0]
		ns = ns[1:]

		if cn.isFolder {
			io.WriteString(hash, cn.name)
			children := maps.Values(cn.children)
			// Ensure that etag is deterministic by always hashing children in
			// the same order.
			sort.Slice(children, func(i, j int) bool {
				return children[i].rname < children[j].rname
			})
			ns = append(ns, children...)
		} else {
			io.WriteString(hash, cn.name)
			io.WriteString(hash, cn.mime)
			io.WriteString(hash, cn.lastMod.Format(rmsTimeFormat))

			fd, err := FS.Open(cn.sname)
			if err != nil {
				return err
			}

			n, err := io.Copy(hash, fd)
			if err != nil {
				return err
			}
			if cn.length != int64(n) {
				return fmt.Errorf("etag: expected to read %d bytes, got: %d", cn.length, n)
			}

			err = fd.Close()
			if err != nil {
				return err
			}
		}
	}

	n.etag = hash.Sum(nil)
	n.etagValid = true
	return nil
}

func recalculateAncestorETags(n *node) error {
	for n != nil {
		err := calculateETag(n)
		if err != nil {
			return err
		}
		n = n.parent
	}
	return nil
}
