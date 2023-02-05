package rmsgo

import (
	"crypto/md5"
	"io"
	"os"
	"time"
)

const BufSize = 1024*32

type ETag []byte

// TODO: Write tests...

func DocumentVersion(n *Node) (etag ETag, err error) {
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

	file, err := os.Open(n.path)
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

func FolderVersion(n *Node) (ETag, error) {
	// FIXME: Make separate types for folders and documents, to avoid
	//  having passed in nodes of the wrong type...
	serverName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	hash.Write([]byte(serverName))

	folders := []*Node{n}

	for len(folders) > 0 {
		n := folders[0]
		hash.Write([]byte(n.name))
		for _, c := range n.children {
			if c.ntype == Document {
				hash.Write([]byte(c.name))
				hash.Write([]byte(c.mime))
				timeFmt := n.lastMod.Format(time.RFC822Z)
				hash.Write([]byte(timeFmt))

				file, err := os.Open(n.path)
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
			} else if c.ntype == Folder {
				folders = append(folders, c)
			}
		}
	}

	return hash.Sum(nil), err
}
