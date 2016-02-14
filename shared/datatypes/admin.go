package dt

import "os"

type admin struct {
	name  string
	email string
}

func GetAdmin() *admin {
	return &admin{
		name:  os.Getenv("ADMIN_NAME"),
		email: os.Getenv("ADMIN_EMAIL"),
	}
}

func (a *admin) GetName() string {
	return a.name
}

func (a *admin) GetEmail() string {
	return a.email
}
