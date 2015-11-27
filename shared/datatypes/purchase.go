package dt

import (
	"database/sql"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/satori/go.uuid"
)

type Purchase struct {
	ID                 uuid.UUID
	UserID             uint64
	User               *User
	VendorID           uint64
	Vendor             *Vendor
	ShippingAddress    *Address
	ShippingAddressID  sql.NullInt64
	Products           []string // product names
	Tax                uint64
	Shipping           uint64
	Total              uint64
	AvaFee             uint64
	CreditCardFee      uint64
	TransferFee        uint64
	VendorPayout       uint64
	VendorPaidAt       *time.Time
	DeliveryExpectedAt *time.Time
	EmailsSentAt       *time.Time
	CreatedAt          *time.Time

	db *sqlx.DB
}

func NewPurchase(db *sqlx.DB) *Purchase {
	return &Purchase{db: db}
}

func (p *Purchase) Init() error {
	if p.User == nil || p.Vendor == nil {
		q := `SELECT id, name, email FROM users WHERE id=$1`
		if err := p.db.Get((*p).User, q, p.UserID); err != nil {
			return err
		}
		q = `
			SELECT id, businessname, contactname, contactemail
			FROM vendors
			WHERE id=$1`
		if err := p.db.Get((*p).Vendor, q, p.VendorID); err != nil {
			return err
		}
	}
	if p.ShippingAddressID.Valid && p.ShippingAddress == nil {
		q := `
			SELECT id, businessname, contactname, contactemail
			FROM vendors
			WHERE id=$1`
		err := p.db.Get((*p).ShippingAddress, q, p.ShippingAddressID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Purchase) Subtotal() uint64 {
	return p.Total - p.Tax - p.Shipping
}

func (p *Purchase) UpdateEmailsSent() error {
	t := time.Now()
	(*p).EmailsSentAt = &t
	q := `UPDATE purchases SET emailssentat=$1 WHERE id=$2`
	_, err := p.db.Exec(q, p.EmailsSentAt, p.ID)
	return err
}
