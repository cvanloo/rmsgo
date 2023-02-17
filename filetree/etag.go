package filetree

import (
	"crypto/md5"
	"io"
	"os"
	"time"
)

const BufSize = 1024 * 1024 * 64

type ETag []byte

func (etag ETag) String() string {
	return string(etag)
}

// TODO: Write tests...
// TODO: Avoid recalculating the ETags every time (only recalculate when
//   something actually changes).

func Resolve(n NodeInfo) string {
	panic("not implemented")
}

func DocumentVersion(n Document) (etag ETag, err error) {
	serverName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(serverName))
	hash.Write([]byte(n.name))
	hash.Write([]byte(n.Mime))
	timeFmt := n.LastMod.Format(time.RFC1123Z)
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

func FolderVersion(n Folder) (ETag, error) {
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

		if n.IsFolder() {
			f := n.Folder()
			for _, c := range f.children {
				nodes = append(nodes, c)
			}
		} else {
			d := n.Document()
			hash.Write([]byte(d.name))
			hash.Write([]byte(d.Mime))
			timeFmt := d.LastMod.Format(time.RFC1123Z)
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
	}

	return hash.Sum(nil), err
}
