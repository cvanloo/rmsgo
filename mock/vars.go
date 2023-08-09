package mock

var (
	Time            = TimeFunc()
	UUID            = UUIDFunc()
	FS   FileSystem = MockFS()
)

// Mock re-initializes all mock variables to their mocked counterparts.
// The underlying mock fileSystem is returned, so that directories and files
// may be added.
func Mock() *FakeFileSystem {
	Time = TimeFunc()
	UUID = UUIDFunc()
	fs := MockFS()
	FS = fs
	return fs
}
