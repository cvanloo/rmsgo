package path

import "path/filepath"

type StoragePath interface {
	Storage() string
}

type WebPath interface {
	Web() string
}

type RemotePath interface {
	Remote() string
}

type CompletePath interface {
	StoragePath
	WebPath
	RemotePath
}

type RmsPath struct {
	Name, StoragePath, WebPath string
}

// rmsPath implements CompletePath.
var _ CompletePath = (*RmsPath)(nil)

func (rp RmsPath) Storage() string {
	return rp.StoragePath
}

func (rp RmsPath) Web() string {
	return rp.WebPath
}

func (rp RmsPath) Remote() string {
	return rp.Name
}

func NewPath(webRoot, storageRoot, webPath string) (RmsPath, error) {
	name, err := filepath.Rel(webRoot, webPath)
	if err != nil {
		var nop RmsPath
		return nop, err
	}
	return RmsPath{
		Name:        name,
		StoragePath: filepath.Join(storageRoot, name),
		WebPath:     webPath,
	}, nil
}
