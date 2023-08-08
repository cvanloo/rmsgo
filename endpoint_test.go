package rmsgo

import (
	"fmt"
	"testing"
)

func mockServer() (*mockFileSystem, Server) {
	server := Server{
		Rroot: "/storage/",
		Sroot: "/tmp/rms/storage/",
	}

	mfs := CreateMockFS().CreateDirectories(server.Sroot)
	mfs.
		AddFile("test.txt", "Hello, World!").
		AddDirectory("Pictures").
		Into().
		AddFile("Kittens.png", "A cute kitten!").
		AddFile("Gopher.jpg", "It's Gopher!").
		Leave().
		AddDirectory("Documents").
		Into().
		AddFile("doc.txt", "Lorem ipsum dolores sit amet").
		AddFile("fakenius.txt", "Ich fand es schon immer verdächtig, dass die Sonne jeden Morgen im Osten aufgeht!")

	createUUID = CreateMockUUIDFunc()
	getTime = getMockTime

	Reset()

	return mfs, server
}

func TestGetFolder(t *testing.T) {
	fs, server := mockServer()
	_ = server

	fmt.Printf("%v\n", fs)
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
