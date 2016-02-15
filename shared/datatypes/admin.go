package dt

import "os"

type admin struct {
	name  string
	email string
}

// Admin is a singleton that represents the current admin running the deployed
// service. Its name and email are set through the environment variables
// ADMIN_NAME and ADMIN_EMAIL.
func Admin() *admin {
	return &admin{
		name:  os.Getenv("ADMIN_NAME"),
		email: os.Getenv("ADMIN_EMAIL"),
	}
}

// GetName satisfies the Contactable interface, making it possible to email the
// admin.
func (a *admin) GetName() string {
	return a.name
}

// GetEmail satisfies the Contactable interface, making it possible to email the
// admin.
func (a *admin) GetEmail() string {
	return a.email
}
