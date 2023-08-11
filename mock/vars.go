package mock

var (
	Time            = TimeFunc()
	UUID            = UUIDFunc()
	FS   FileSystem = MockFS()
)

// Mock re-initializes all mock variables to their mocked counterparts.
// FSOptions may be used to setup directories and files.
func Mock(fsOpts ...FSOption) {
	Time = TimeFunc()
	UUID = UUIDFunc()
	FS = MockFS(fsOpts...)
}
