package storage

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
)

const bufSize = 1024 * 1024 * 64
var storageRoot string

func Store(name string, reader io.Reader) error {
	path := filepath.Join(storageRoot, name)
	// TODO: create ancestors?
	// TODO: permissions?
	fo, err := os.Create(path)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(fo)
	buf := make([]byte, bufSize)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, err := w.Write(buf); err != nil {
			return err
		}
	}

	return nil
}

func Retrieve(name string) (io.Reader, error) {
	path := filepath.Join(storageRoot, name)
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func Remove(name string) error {
	path := filepath.Join(storageRoot, name)
	// TODO: remove empty ancestors?
	err := os.Remove(path)
	return err
}
