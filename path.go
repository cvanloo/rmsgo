package rmsgo

type Path struct {
	path string
}

func (p Path) System() string {
	panic("not implemented")
}

func (p Path) Remote() string {
	panic("not implemented")
}

func (p Path) Relative(to string) string {
	panic("not implemented")
}

func (p Path) Parent() Path {
	panic("not implemented")
}
