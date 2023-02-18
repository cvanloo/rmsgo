package mocks

var TestUser = User{
	name:  "Testikus",
	quota: 1024 * 16,
}

type User struct {
	name  string
	quota uint
}

func (u User) Name() string {
	return u.name
}

func (u User) Quota() uint {
	return u.quota
}
