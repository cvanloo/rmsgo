package rmsgo

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"

	"golang.org/x/exp/maps"
)

type ETag []byte

func ParseETag(s string) (ETag, error) {
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

func (e ETag) String() string {
	return hex.EncodeToString(e)
}

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(fmt.Errorf("rmsgo: failed to read hostname: %v", err))
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
			// Ensure that etag is deterministic by always hashing the chilren
			// in the same order.
			sort.Slice(children, func(i, j int) bool {
				return children[i].Rname < children[j].Rname
			})
			ns = append(ns, children...)
		} else {
			io.WriteString(hash, cn.Name)
			io.WriteString(hash, cn.Mime)
			io.WriteString(hash, cn.LastMod.Format(rmsTimeFormat))

			fd, err := mfs.Open(cn.Sname)
			if err != nil {
				return err
			}

			io.Copy(hash, fd)

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
