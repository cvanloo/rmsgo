package rmsgo

type UserStorage interface {
	Find(id, secret string) (User, error)
}

type User interface {
	Name() string
	Quota() uint
}
