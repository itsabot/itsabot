package dt

import (
	"database/sql"
	"log"
	"math"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

type Purchase struct {
	ID                 string `sql:"type:varchar(36);primary key"`
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

func NewPurchase(ctx *Ctx, pc *PurchaseConfig) (*Purchase, error) {
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
	// always round up fees to ensure we aren't losing money on fractional
	// cents
	p.AvaFee = uint64(math.Ceil(float64(p.Total) * 0.05))
	p.CreditCardFee = uint64(math.Ceil((float64(p.Total)*0.029 + 0.3)))
	p.TransferFee = uint64(math.Ceil((float64(p.Total-
		p.AvaFee-
		p.CreditCardFee) * 0.005)))
	p.VendorPayout = p.Total - p.AvaFee - p.CreditCardFee - p.TransferFee
	t := time.Now().Add(7 * 24 * time.Hour)
	p.DeliveryExpectedAt = &t
	if p.User == nil {
		(*p).User = &User{}
		q := `SELECT id, name, email FROM users WHERE id=$1`
		if err := p.db.Get((*p).User, q, p.UserID); err != nil {
			return nil, err
		}
	}
	if p.Vendor == nil {
		(*p).Vendor = &Vendor{}
		q := `
			SELECT id, businessname, contactname, contactemail
			FROM vendors
			WHERE id=$1`
		if err := p.db.Get((*p).Vendor, q, p.VendorID); err != nil {
			return nil, err
		}
	}
	id, err := p.Create()
	if err != nil {
		return nil, err
	}
	p.ID = id
	return p, nil
}

func (p *Purchase) Create() (string, error) {
	q := `
		INSERT INTO purchases
		(userid, vendorid, shippingaddressid, products, tax, shipping,
		total, avafee, creditcardfee, transferfee, vendorpayout)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, $10)
		RETURNING id`
	log.Println("tax", p.Tax)
	log.Println("shipping", p.Shipping)
	log.Println("total", p.Total)
	log.Println("avafee", p.AvaFee)
	log.Println("creditcardfee", p.CreditCardFee)
	log.Println("vendorpayout", p.VendorPayout)
	var id string
	err := p.db.QueryRowx(q, p.User.ID, p.Vendor.ID, p.ShippingAddressID,
		StringSlice(p.Products), p.Tax, p.Shipping, p.Total, p.AvaFee,
		p.CreditCardFee, p.VendorPayout).Scan(&id)
	if err != nil {
		log.Println("ERR HERE")
		return "", err
	}
	return id, nil
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
