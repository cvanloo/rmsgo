package rmsgo

type Node struct {
	IsFolder bool
	Name string
	Length   int64
	Mime     string
	LastMod time.Time
	Version string
}

type Listing []Node

func Save(name string, r io.Reader) error {
}

func Delete(name string) error {
}

func Folder(name string) (Listing, error) {
}

func Document(name string) (io.Reader, error) {
}

var cache = map[string]Listing{}

func Cache(name string) (lst Listing, ok bool) {
	lst, ok = cache[name]
	if ok {
		// @todo: head folder to check version
		serverVersion := "todo!"
		return lst, lst.Version == serverVersion
	}
	return
}
