package rmsgo

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

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

func calculateETag(n *node) error {
	hash := md5.New()
	io.WriteString(hash, hostname)

	ns := []*node{n}
	for len(ns) > 0 {
		cn := ns[0]
		ns = ns[1:]

		if cn.isFolder {
			io.WriteString(hash, cn.name)
			ns = append(ns, maps.Values(cn.children)...)
		} else {
			io.WriteString(hash, cn.name)
			io.WriteString(hash, cn.mime)
			io.WriteString(hash, cn.lastMod.Format(time.RFC1123))

			fd, err := mfs.Open(cn.sname)
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
