package mock

import (
	"fmt"
	"github.com/cvanloo/go-ffs"
	"time"

	"github.com/google/uuid"
)

// Alias into this namespace, to make importing easier for users of mock.
type (
	FileSystem     = ffs.FileSystem
	File           = ffs.File
	RealFileSystem = ffs.RealFileSystem
	FakeFileSystem = ffs.FakeFileSystem
	FSOption       = ffs.FSOption
)

// Alias into this namespace, to make importing easier for users of mock.
var (
	MockFS        = ffs.MockFS
	WithFile      = ffs.WithFile
	WithDirectory = ffs.WithDirectory
)

// These variables might hold a concrete implementation or a mock.
// Use these variables instead of using the concrete implementations directly.
var (
	Time            = TimeFunc()
	UUID            = UUIDFunc()
	FS   FileSystem = MockFS()
)

func init() {
	ffs.Time = Time
}

// Mock re-initializes all mock variables to their mocked counterparts.
// FSOptions may be used to setup directories and files.
func Mock(fsOpts ...FSOption) {
	Time = TimeFunc()
	ffs.Time = TimeFunc()
	UUID = UUIDFunc()
	FS = MockFS(fsOpts...)
}

// UUIDFunc returns a mock function that creates predictable UUIDs, simply a
// number starting at one, being increased for each new UUID.
func UUIDFunc() func() (uuid.UUID, error) {
	last := 0
	return func() (uuid.UUID, error) {
		last++
		lastX := fmt.Sprintf("%016x", last) // @nocheckin
		return uuid.UUID([]byte(lastX)[:16]), nil
	}
}

// TimeFunc returns a mock function that will always return the zero value of
// time.Time.
func TimeFunc() func() (t time.Time) {
	return func() (t time.Time) {
		return
	}
}
