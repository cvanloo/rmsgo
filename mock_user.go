package rmsgo

type mockUser struct{
	name string
	quota uint
}

func (mu mockUser) Name() string {
	return mu.name
}

func (mu mockUser) Quota() uint {
	return mu.quota
}
