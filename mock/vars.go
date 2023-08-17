package mock

var (
	Time Timer      = &TimeMock{}
	UUID UUIDer     = &UUIDMock{}
	ETag Versioner  = &RealVersioner{} // not mocked per default
	FS   FileSystem = MockFS()
)

// Mock re-initializes all mock variables to their mocked counterparts.
// FSOptions may be used to setup directories and files.
func Mock(fsOpts ...FSOption) {
	Time = &TimeMock{}
	UUID = &UUIDMock{}
	ETag = &RealVersioner{} // not mocked per default
	FS = MockFS(fsOpts...)
}
