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

type PurchaseConfig struct {
	*User
	Prices          []uint64
	VendorID        uint64
	ShippingAddress *Address
	Products        []Product
}

func NewPurchase(ctx *Ctx, pc *PurchaseConfig) *Purchase {
	p := &Purchase{db: ctx.DB}
	p.User = pc.User
	p.ShippingAddress = pc.ShippingAddress
	p.VendorID = pc.VendorID
	for _, prod := range pc.Products {
		p.Products = append(p.Products, prod.Name)
	}
	p.Total = pc.Prices[0]
	p.Tax = pc.Prices[1]
	p.Shipping = pc.Prices[2]
	p.AvaFee = uint64(float64(p.Total) * 0.05 * 100)
	p.CreditCardFee = uint64((float64(p.Total)*0.029 + 0.3) * 100)
	p.TransferFee =
		uint64((float64(p.Total-
			p.AvaFee-
			p.CreditCardFee) * 0.005) * 100)
	p.VendorPayout = p.Total - p.AvaFee - p.CreditCardFee - p.TransferFee
	t := time.Now().Add(7 * 24 * time.Hour)
	p.DeliveryExpectedAt = &t
	return p
}

func (p *Purchase) Init() error {
	if p.User == nil {
		(*p).User = &User{}
		q := `SELECT id, name, email FROM users WHERE id=$1`
		if err := p.db.Get((*p).User, q, p.UserID); err != nil {
			return err
		}
	}
	if p.Vendor == nil {
		(*p).Vendor = &Vendor{}
		q := `
			SELECT id, businessname, contactname, contactemail
			FROM vendors
			WHERE id=$1`
		if err := p.db.Get((*p).Vendor, q, p.VendorID); err != nil {
			return err
		}
	}
	if p.ShippingAddress == nil {
		(*p).ShippingAddress = &Address{}
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
