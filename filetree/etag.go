package filetree

import (
	"crypto/md5"
	"io"
	"os"
	"path/filepath"
	"time"
)

type ETag []byte

const BufSize = 1024 * 1024 * 64

var serverName string

func init() {
	hn, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	serverName = hn
}

func (etag ETag) String() string {
	return string(etag)
}

// TODO: Write tests...
// TODO: Avoid recalculating the ETags every time (only recalculate when
//   something actually changes).

func Resolve(n NodeInfo) string {
	return filepath.Join(storageRoot, n.Name())
}

func DocumentVersion(doc Document) (etag ETag, err error) {
	hash := md5.New()
	hash.Write([]byte(serverName))
	hash.Write([]byte(doc.name))
	hash.Write([]byte(doc.Mime))
	timeFmt := doc.LastMod.Format(time.RFC1123Z)
	hash.Write([]byte(timeFmt))

	file, err := os.Open(Resolve(doc))
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

func FolderVersion(fol Folder) (ETag, error) {
	hash := md5.New()
	hash.Write([]byte(serverName))

	nodes := []NodeInfo{fol}

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

	return hash.Sum(nil), nil
}
