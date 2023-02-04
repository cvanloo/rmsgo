package rmsgo

type UserStorage interface {
	Find(id string) User
}

type User interface {
	Name() string
	Quota() uint
}
