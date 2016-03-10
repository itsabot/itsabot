package dt

import "os"

// Admin is a special type of user that has access to Abot's admin panel.
type Admin struct {
	name  string
	email string
}

// NewAdmin returns a singleton that represents the current admin running the
// deployed service. Its name and email are set through the environment
// variables ADMIN_NAME and ADMIN_EMAIL.
func NewAdmin() *Admin {
	return &Admin{
		name:  os.Getenv("ADMIN_NAME"),
		email: os.Getenv("ADMIN_EMAIL"),
	}
}

// GetName satisfies the Contactable interface, making it possible to email the
// admin.
func (a *Admin) GetName() string {
	return a.name
}

// GetEmail satisfies the Contactable interface, making it possible to email the
// admin.
func (a *Admin) GetEmail() string {
	return a.email
}
