package rmsgo

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/exp/maps"
)

type ETag []byte

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(fmt.Errorf("rmsgo: failed to read hostname: %v", err))
	}
}

func generateETag(n *node) (ETag, error) {
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
				return nil, err
			}

			io.Copy(hash, fd)

			err = fd.Close()
			if err != nil {
				return nil, err
			}
		}
	}

	return hash.Sum(nil), nil
}
