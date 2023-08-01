package rmsgo

import (
	"fmt"
	"testing"
)

func TestGetFolder(t *testing.T) {
	fs := CreateMockFS()
	fs.
		AddFile("test.txt", "Hello, World!").
		AddDirectory("Pictures").
		Into().
		AddFile("Kittens.png", "A cute kitten!").
		AddFile("Gopher.jpg", "It's Gopher!").
		Leave().
		AddDirectory("Documents").
		Into().
		AddFile("doc.txt", "Lorem ipsum dolores sit amet").
		AddFile("taxes.txt", "I ain't paying 'em!")

	fmt.Printf("%v\n", fs)

	mfs = fs
}

func TestHeadFolder(t *testing.T) {
}

func TestGetDocument(t *testing.T) {
}

func TestHeadDocument(t *testing.T) {
}

func TestPutDocument(t *testing.T) {
}

func TestDeleteDocument(t *testing.T) {
}
