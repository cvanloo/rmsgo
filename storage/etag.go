package storage

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"
)

type ETag []byte

func (e ETag) Base64() string {
	return base64.StdEncoding.EncodeToString(e)
}

var serverName string

func init() {
	name, err := os.Hostname()
	if err != nil {
		panic(fmt.Errorf("could not obtain hostname information: %v", err))
	}
	serverName = name
}

func DocumentVersion(doc Document) (etag ETag, err error) {
	hash := md5.New()
	hash.Write([]byte(serverName))
	hash.Write([]byte(doc.Name()))
	hash.Write([]byte(doc.Mime()))
	timeFmt := doc.LastMod().Format(time.RFC1123Z)
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
			for _, c := range f.Children() {
				nodes = append(nodes, c)
			}
		} else {
			d := n.Document()
			hash.Write([]byte(d.Name()))
			hash.Write([]byte(d.Mime()))
			timeFmt := d.LastMod().Format(time.RFC1123Z)
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
					return nil, rerr
				}
			}
			file.Close()
		}
	}

	return hash.Sum(nil), nil
}
