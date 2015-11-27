package dt

type Vendor struct {
	ID           uint64
	BusinessName string
	ContactName  string
	ContactEmail string
}

// GetName satisfies the Contactable interface
func (v *Vendor) GetName() string {
	return v.BusinessName
}

// GetEmail satisfies the Contactable interface
func (v *Vendor) GetEmail() string {
	return v.ContactEmail
}
