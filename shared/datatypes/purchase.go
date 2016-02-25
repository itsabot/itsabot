package dt

import (
	"database/sql"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
)

// Purchase represents a user purchase and associated useful information such as
// a breakdown of pricing, products purchased and the time a delivery is
// expected.
type Purchase struct {
	ID                 uint64
	UserID             uint64
	User               *User
	VendorID           uint64
	Vendor             *Vendor
	ShippingAddress    *Address
	ShippingAddressID  sql.NullInt64
	Products           []string // product names
	ProductSels        ProductSels
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
	db                 *sqlx.DB
}

// PurchaseConfig is a smaller set of purchase information that packages can use
// to more easily build a full Purchase.
type PurchaseConfig struct {
	*User
	Prices          []uint64
	VendorID        uint64
	ShippingAddress *Address
	ProductSels     ProductSels
}

// statesTax represents the percentage of tax paid on a state-by-state basis.
// TODO This should be expanded beyond just California.
var statesTax = map[string]float64{
	"CA": 0.0925,
}

// NewPurchase creates a Purchase and fills in information like a pricing
// breakdown automatically based on a provided PurchaseConfig.
func NewPurchase(db *sqlx.DB, pc *PurchaseConfig) (*Purchase, error) {
	p := &Purchase{db: db}
	p.ID = uint64(rand.Int63n(8999999999) + 1000000000)
	p.User = pc.User
	p.ShippingAddress = pc.ShippingAddress
	p.VendorID = pc.VendorID
	for _, prod := range pc.ProductSels {
		p.Products = append(p.Products, prod.Name)
	}
	p.ProductSels = pc.ProductSels
	prices := pc.ProductSels.Prices(pc.ShippingAddress)
	p.Total = prices["total"]
	p.Tax = prices["tax"]
	p.Shipping = prices["shipping"]

	// Always round up fees to ensure we aren't losing money on fractional
	// cents.
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
		q := `SELECT id, businessname, contactname, contactemail
		      FROM vendors
		      WHERE id=$1`
		if err := p.db.Get((*p).Vendor, q, p.VendorID); err != nil {
			return nil, err
		}
	}
	if err := p.Create(); err != nil {
		return nil, err
	}
	return p, nil
}

// Create saves a new Purchase to the database and returns an error, if any.
func (p *Purchase) Create() error {
	q := `INSERT INTO purchases
	      (id, userid, vendorid, shippingaddressid, products, tax, shipping,
		total, avafee, creditcardfee, transferfee, vendorpayout)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, $11)`
	_, err := p.db.Exec(q, p.ID, p.User.ID, p.Vendor.ID,
		p.ShippingAddressID, nlp.StringSlice(p.Products),
		p.Tax, p.Shipping, p.Total, p.AvaFee, p.CreditCardFee,
		p.VendorPayout)
	return err
}

// Subtotal is a helper function to return the purchase price before tax and
// shipping, i.e. only the cost of the products purchased.
func (p *Purchase) Subtotal() uint64 {
	return p.Total - p.Tax - p.Shipping
}

// UpdateEmailsSent records the time at which a purchase confirmation and vendor
// request were sent. See itsabot.org/abot/shared/task/request_auth.go:makePurchase for an
// example.
func (p *Purchase) UpdateEmailsSent() error {
	t := time.Now()
	(*p).EmailsSentAt = &t
	q := `UPDATE purchases SET emailssentat=$1 WHERE id=$2`
	_, err := p.db.Exec(q, p.EmailsSentAt, p.ID)
	return err
}

// DisplayID returns a user-facing identifier for a purchase made that can be
// referenced in future communications, such as if purchased items arrive
// damaged.
func (p *Purchase) DisplayID() string {
	s := strconv.FormatUint(p.ID, 10)
	return s[:len(s)/2] + "-" + s[len(s)/2:]
}
