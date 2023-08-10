package rmsgo

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	. "github.com/cvanloo/rmsgo.git/mock"
	"golang.org/x/exp/maps"
)

// ETag is a short and unique identifier assigned to a specific version of a
// remoteStorage resource.
type ETag []byte

// String creates a string from an Etag e.
// To go the opposite way an obtain an ETag from a string, use ParseETag.
func (e ETag) String() string {
	return hex.EncodeToString(e)
}

// ParseETag decodes an ETag previously encoded by (ETag).String()
func ParseETag(s string) (ETag, error) {
	if len(s) != md5.Size {
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

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("failed to read hostname: %v", err)
	}
}

func calculateETag(n *Node) error {
	hash := md5.New()
	io.WriteString(hash, hostname)

	ns := []*Node{n}
	for len(ns) > 0 {
		cn := ns[0]
		ns = ns[1:]

		if cn.IsFolder {
			io.WriteString(hash, cn.Name)
			children := maps.Values(cn.children)
			// Ensure that etag is deterministic by always hashing children in
			// the same order.
			sort.Slice(children, func(i, j int) bool {
				return children[i].Rname < children[j].Rname
			})
			ns = append(ns, children...)
		} else {
			io.WriteString(hash, cn.Name)
			io.WriteString(hash, cn.Mime)
			io.WriteString(hash, cn.LastMod.Format(rmsTimeFormat))

			fd, err := FS.Open(cn.Sname)
			if err != nil {
				return err
			}

			n, err := io.Copy(hash, fd)
			if err != nil {
				return err
			}
			if cn.Length != int64(n) {
				return fmt.Errorf("etag: expected to read %d bytes, got: %d", cn.Length, n)
			}

			err = fd.Close()
			if err != nil {
				return err
			}
		}
	}

	n.ETag = hash.Sum(nil)
	n.etagValid = true
	return nil
}

func recalculateAncestorETags(n *Node) error {
	for n != nil {
		err := calculateETag(n)
		if err != nil {
			return err
		}
		n = n.parent
	}
	return nil
}
