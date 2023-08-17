package mock

import (
	"os"
	"time"

	"golang.org/x/exp/slog"
)

var (
	Time Timer      = &TimeMock{}
	UUID UUIDer     = &UUIDMock{}
	ETag Versioner  = &RealVersioner{} // not mocked per default
	FS   FileSystem = MockFS()
)

// Mock re-initializes all mock variables to their mocked counterparts.
// FSOptions may be used to setup directories and files.
func Mock(fsOpts ...FSOption) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	Time = &TimeMock{}
	UUID = &UUIDMock{}
	ETag = &RealVersioner{} // not mocked per default
	FS = MockFS(fsOpts...)
}

type LogDTO struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
}
