package storage

import (
	"bufio"
	"io"
	"os"

	"framagit.org/attaboy/rmsgo/path"
)

const bufSize = 1024 * 1024 * 64

func Store(path path.StoragePath, reader io.Reader) error {
	name := path.Storage()
	// TODO: create ancestors?
	// TODO: permissions?
	fo, err := os.Create(name)
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

func Retrieve(path path.StoragePath) (io.Reader, error) {
	fi, err := os.Open(path.Storage())
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func Remove(path path.StoragePath) error {
	// TODO: remove empty ancestors?
	err := os.Remove(path.Storage())
	return err
}
