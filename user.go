package rmsgo

type UserStorage interface {
	Find(id string) (User, error)
}

type User interface {
	Name() string
	Quota() uint
}
