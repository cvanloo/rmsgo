package mock

import (
	"log"
	"time"

	"github.com/cvanloo/rmsgo/etag"
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
}

var _ Versioner = (*VersionLogger)(nil)

func (vl VersionLogger) Version(n etag.Node) (etag.ETag, error) {
	etag, err := vl.Versioner.Version(n)
	log.Printf("%v", map[string]any{
		"action": "Versioner",
		"date":   time.Now(),
		"result": etag,
		"error":  err,
	})
	return etag, err
}

type VersionResult struct {
	Result etag.ETag
	Err    error
}

type ReplayVersion struct {
	Queue[VersionResult]
}

var _ Versioner = (*ReplayVersion)(nil)

func (r *ReplayVersion) Version(n etag.Node) (etag.ETag, error) {
	ret := r.Dequeue()
	return ret.Result, ret.Err
}
