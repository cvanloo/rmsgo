package rmsgo

import (
	"testing"
)

//func TestMain(m *testing.M) {
//	FS = &MockFileSystem{}
//	code := m.Run()
//	os.Exit(code)
//}

func TestGetFolder(t *testing.T) {
	FS = &mockFileSystem{
		contents: map[string]*mockFile{
			"test.txt": {
				isDir: false,
				name:  "test.txt",
				bytes: []byte("Hello, World!"),
			},
			"Pictures": {
				isDir: true,
				name: "Pictures",
				children: map[string]*mockFile{
					"Kittens.png": {
						isDir: false,
						name: "Kittens.png",
						bytes: []byte("Just imagine this to be a picture of kittens."),
					},
				},
			},
		},
	}
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
