package mock

import (
	"github.com/cvanloo/rmsgo/etag"
	"golang.org/x/exp/slog"
)

type Versioner interface {
	Version(n etag.Node) (etag.ETag, error)
}

type RealVersioner struct{}

var _ Versioner = (*RealVersioner)(nil)

func (RealVersioner) Version(n etag.Node) (etag.ETag, error) {
	return etag.CalculateETag(n)
}

type VersionLogger struct {
	Versioner
	Log *slog.Logger
}

var _ Versioner = (*VersionLogger)(nil)

func (vl VersionLogger) Version(n etag.Node) (etag.ETag, error) {
	etag, err := vl.Versioner.Version(n)
	vl.Log.Debug("Versioner", "etag", etag.String(), "error", err)
	return etag, err
}

type VersionResult struct {
	Result etag.ETag `json:"etag"`
	Err    error     `json:"error"`
}

type ReplayVersion struct {
	Queue[VersionResult]
}

var _ Versioner = (*ReplayVersion)(nil)

func (r *ReplayVersion) Version(n etag.Node) (etag.ETag, error) {
	ret := r.Dequeue()
	return ret.Result, ret.Err
}
