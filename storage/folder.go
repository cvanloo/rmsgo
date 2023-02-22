package storage

type folder struct {
	parent   Folder
	name     string
	version  ETag
	children map[string]NodeInfo
}

// folder implements Folder
var _ Folder = (*folder)(nil)

func (folder) IsFolder() bool {
	return true
}

func (f folder) Folder() Folder {
	return f
}

func (folder) Document() Document {
	panic("a folder is not a document")
}

func (f folder) Parent() Folder {
	return f.parent
}

func (f folder) Name() string {
	return f.name
}

func (f folder) Description() map[string]any {
	desc := map[string]any{
		"ETag": f.Version(),
	}
	return desc
}

func (f folder) Version() ETag {
	return f.version
}

func (f folder) Children() map[string]NodeInfo {
	return f.children
}
