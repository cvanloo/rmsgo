package rmsgo

import (
	"crypto/md5"
	"io"
	"os"
	"time"
)

const BufSize = 1024 * 32

type ETag []byte

// TODO: Write tests...

func DocumentVersion(n *document) (etag ETag, err error) {
	serverName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(n.name))
	hash.Write([]byte(n.mime))
	timeFmt := n.lastMod.Format(time.RFC822Z)
	hash.Write([]byte(timeFmt))
	hash.Write([]byte(serverName))

	file, err := os.Open(n.Resolve())
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

func FolderVersion(n *folder) (ETag, error) {
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

		if f, ok := n.(folder); ok {
			for _, c := range f.children {
				if cdoc, ok := c.(document); ok {
					hash.Write([]byte(cdoc.name))
					hash.Write([]byte(cdoc.mime))
					timeFmt := cdoc.lastMod.Format(time.RFC822Z)
					hash.Write([]byte(timeFmt))

					file, err := os.Open(cdoc.Resolve())
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
				} else {
					nodes = append(nodes, c)
				}
			}
		}

	}

	return hash.Sum(nil), err
}
