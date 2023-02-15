package rmsgo

import (
	"crypto/md5"
	"io"
	"os"
	"time"
)

const BufSize = 1024 * 64

type ETag []byte

// TODO: Write tests...
// TODO: Avoid recalculating the ETags every time (only recalculate when
//   something actually changes).

func DocumentVersion(n *Document) (etag ETag, err error) {
	serverName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(serverName))
	hash.Write([]byte(n.name))
	hash.Write([]byte(n.mime))
	timeFmt := n.lastMod.Format(time.RFC1123Z)
	hash.Write([]byte(timeFmt))

	file, err := os.Open(Resolve(n))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, BufSize)
	for {
		n, rerr := file.Read(buf)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			err = rerr
			break
		}
	}
	return hash.Sum(nil), err
}

func FolderVersion(n *Folder) (ETag, error) {
	serverName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(serverName))

	nodes := []NodeInfo{n}

	for len(nodes) > 0 {
		n := nodes[0]
		nodes = nodes[1:]
		hash.Write([]byte(n.Name()))

		if f, ok := n.(Folder); ok {
			for _, c := range f.children {
				nodes = append(nodes, c)
			}
		} else if d, ok := n.(Document); ok {
			hash.Write([]byte(d.name))
			hash.Write([]byte(d.mime))
			timeFmt := d.lastMod.Format(time.RFC1123Z)
			hash.Write([]byte(timeFmt))

			file, err := os.Open(Resolve(d))
			if err != nil {
				return nil, err
			}

			buf := make([]byte, BufSize)
			for {
				n, rerr := file.Read(buf)
				if n > 0 {
					hash.Write(buf[:n])
				}
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					err = rerr
					break
				}
			}
			file.Close()
		}
		// FIXME: else ... shouldn't even be possible
	}

	return hash.Sum(nil), err
}
