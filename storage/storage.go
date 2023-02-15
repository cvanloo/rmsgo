package storage

import (
	"errors"
	"io"
)

func Store(name string, reader io.Reader) error {
	return errors.New("not implemented")
}

func Retrieve(name string) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func Remove(name string) error {
	return errors.New("not implemented")
}
