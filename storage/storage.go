package storage

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// TODO: Recalculate ETag only when something actually changed
// TODO: Initialize file tree from file system
// TODO: Alternative file tree that uses database

const BufSize = 1024 * 1024 * 64

var storageRoot string = "/tmp/storage/"

func Setup(root string) {
	storageRoot = root
}

func Resolve(n NodeInfo) string {
	return filepath.Join(storageRoot, n.Name())
}

func Store(name string, reader io.Reader, contentType string, contentLength uint64) (node NodeInfo, err error) {
	// TODO: create ancestor directories if necessary
	path := filepath.Join(storageRoot, name)
	outFile, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer func(outFile *os.File) {
		err = outFile.Close()
	}(outFile)

	err = outFile.Chmod(0660)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(outFile)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, err
	}

	doc := &document{
		name:    name,
		mime:    contentType,
		length:  contentLength,
		lastMod: time.Now(),
	}
	doc.parent = nil
	doc.version, err = DocumentVersion(doc)
	if err != nil {
		return nil, err
	}
	node = doc

	return node, nil
}

func Retrieve(name string) (NodeInfo, bool) {
	node, found := nodes[name]
	return node, found
}

func Remove(name string) error {
	n, ok := nodes[name]
	if !ok {
		return fmt.Errorf("invalid node specified: `%s' does not exist", name)
	}

	for {
		delete(nodes, n.Name())
		p := n.Parent()
		delete(p.Children(), n.Name())
		if p == rootNode {
			break
		}
		if len(p.Children()) > 0 {
			break
		}
		n = p // delete empty parent
	}

	err := os.RemoveAll(Resolve(n))
	if err != nil {
		return fmt.Errorf("failed to remove node(s) `%s' from file system: %v", n.Name(), err)
	}
	return nil
}
