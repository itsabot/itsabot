package dt

// Vendor represents some seller of a product or service Ava provides that is
// contactable via email to notify them of a new user purchase or transaction.
type Vendor struct {
	ID           uint64
	BusinessName string `sql:"businessname"`
	ContactName  string `sql:"contactname"`
	ContactEmail string `sql:"contactemail"`
}

// GetName satisfies the Contactable interface enabling a vendor to be emailed.
func (v *Vendor) GetName() string {
	return v.BusinessName
}

// GetEmail satisfies the Contactable interface enabling a vendor to be emailed.
func (v *Vendor) GetEmail() string {
	return v.ContactEmail
}
