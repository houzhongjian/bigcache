package model

type User struct {
	Base
	Account  string
	Name     string
	Password string
}

//TableName.
func (u *User) TableName() string {
	return "user"
}
