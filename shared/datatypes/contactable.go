package dt

// Contactable defines an interface used by dt.MailClient to send customized
// emails to users and admins
type Contactable interface {
	GetName() string
	GetEmail() string
}
