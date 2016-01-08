package dt

type Vendor struct {
	ID           uint64
	BusinessName string `sql:"businessname"`
	ContactName  string `sql:"contactname"`
	ContactEmail string `sql:"contactemail"`
}

// GetName satisfies the Contactable interface
func (v *Vendor) GetName() string {
	return v.BusinessName
}

// GetEmail satisfies the Contactable interface
func (v *Vendor) GetEmail() string {
	return v.ContactEmail
}
