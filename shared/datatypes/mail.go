package dt

import (
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/sendgrid/sendgrid-go"
)

// MailClient provides a higher level API over a raw SendGrid client. This has
// the following benefits:
//
// 1. It enables a standardized API when developers want to swap SendGrid for
// something else.
// 2. It reduces the workload to perform common tasks, such as sending users
// purchase confirmations or sending vendors requests for products purchased.
type MailClient struct {
	sgc *sendgrid.SGClient
}

// SendPurchaseConfirmation emails the user a purchase confirmation including
// pricing information and expected delivery dates in an HTML format.
//
// TODO Add shipping information and purchase identifier (UUID)
func (sg *MailClient) SendPurchaseConfirmation(p *Purchase) error {
	products := p.ProductSels
	if len(products) == 0 {
		return errors.New("empty products slice in purchase confirmation")
	}
	subj := fmt.Sprintf("Order confirmation: #%s", p.DisplayID())
	text := "<html><body>"
	text += fmt.Sprintf("<p>Hi %s:</p>", p.User.Name)
	text += "<p>Here's a quick order summary for your records. You bought:</p>"
	text += "<ul>"
	for _, product := range products {
		price := float64(product.Price) / 100
		var size string
		if len(product.Size) > 0 {
			size = fmt.Sprintf(" (%s)", product.Size)
		}
		text += fmt.Sprintf("<li>%d @ $%.2f - %s%s</li>", product.Count,
			price, product.Name, size)
	}
	text += "</ul><table>"
	text += fmt.Sprintf("<tr><td>Subtotal: </td><td>$%.2f</td></tr>",
		float64(p.Subtotal())/100)
	text += fmt.Sprintf("<tr><td>Shipping: </td><td>$%.2f</td></tr>",
		float64(p.Shipping)/100)
	text += fmt.Sprintf("<tr><td>Tax: </td><td>$%.2f</td></tr>",
		float64(p.Tax)/100)
	text += "<tr><td>My fee: </td><td>$0.00</td></tr>"
	text += fmt.Sprintf("<tr><td><b>Total: </b></td><td><b>$%.2f</b></td></tr>",
		float64(p.Total)/100)
	text += "</table>"
	delivery := time.Now().Add(7 * 24 * time.Hour)
	delS := delivery.Format("Monday Jan 2, 2006")
	text += fmt.Sprintf("<p>Expected delivery before %s. ", delS)
	text += fmt.Sprintf("Your order confirmation number is %s.</p>",
		p.DisplayID())
	text += "<p>Glad I could help! :)</p><p>- Ava</p>"
	text += "</body></html>"
	return sg.Send(subj, text, p.User)
}

// SendVendorRequest sends an email to a vendor informing them that payment has
// been taken for a Purchase and asking that the vendor process the order.
//
// TODO add shipping information and purchase identifier (UUID)
func (sg *MailClient) SendVendorRequest(p *Purchase) error {
	if len(p.ProductSels) == 0 {
		return errors.New("empty products slice in vendor request")
	}
	var subj string
	if os.Getenv("AVA_ENV") == "production" {
		subj = fmt.Sprintf("Order Request: #%s", p.DisplayID())
	} else {
		subj = fmt.Sprintf("[TEST - PLEASE IGNORE] Order Request: #%s",
			p.DisplayID())
		(*p.Vendor).ContactName = os.Getenv("ADMIN_NAME")
		(*p.Vendor).ContactEmail = os.Getenv("ADMIN_EMAIL")
	}
	text := "<html><body>"
	text += fmt.Sprintf("<p>Hi %s:</p>", p.Vendor.ContactName)
	text += fmt.Sprintf("<p>%s just ordered the following:</p>",
		p.User.Name)
	text += "<ul>"
	for _, product := range p.ProductSels {
		price := float64(product.Price) / 100
		var size string
		if len(product.Size) > 0 {
			size = fmt.Sprintf(" (%s)", product.Size)
		}
		text += fmt.Sprintf("<li>%d @ $%.2f - %s%s</li>", product.Count,
			price, product.Name, size)
	}
	text += "</ul><table>"
	text += fmt.Sprintf("<tr><td>Subtotal: </td><td>$%.2f</td></tr>",
		float64(p.Subtotal())/100)
	text += fmt.Sprintf("<tr><td>Shipping: </td><td>$%.2f</td></tr>",
		float64(p.Shipping)/100)
	text += fmt.Sprintf("<tr><td>Tax: </td><td>$%.2f</td></tr>",
		float64(p.Tax)/100)
	text += fmt.Sprintf("<tr><td>Ava's fee: </td><td>($%.2f)</td></tr>",
		float64(p.AvaFee)/100)
	text += fmt.Sprintf("<tr><td>Credit card fees: </td><td>($%.2f)</td></tr>",
		float64(p.CreditCardFee)/100)
	text += fmt.Sprintf("<tr><td><b>Total you'll receive: </b></td><td><b>$%.2f</b></td></tr>",
		float64(p.Total-p.AvaFee-p.CreditCardFee)/100)
	text += "</table>"
	text += fmt.Sprintf("<p>%s is expecting delivery before <b>%s</b>. ",
		p.User.Name, p.DeliveryExpectedAt.Format("Monday Jan 2, 2006"))
	text += "The order has been paid for in full and is ready to be shipped.</p>"
	text += "<p>If you have any questions or concerns with this order, "
	text += "please respond to this email.</p>"
	text += "<p>Best,</p>"
	text += "<p>- Ava</p>"
	text += "</body></html>"
	return sg.Send(subj, text, p.Vendor)
}

// SendBug is called every time an unhandled error occurs, particularly when
// that error happens as a result of the message a user sends to Ava
func (sg *MailClient) SendBug(err error) {
	if os.Getenv("AVA_ENV") != "production" {
		return
	}
	subj := "[Bug] " + err.Error()
	if len(subj) > 30 {
		subj = subj[0:27] + "..."
	}
	text := "<html><body>"
	text += fmt.Sprintf("<p>%s</p>", err.Error())
	text += "</body></html>"
	if err := sg.Send(subj, text, Admin()); err != nil {
		log.Error("sending bug report", err)
	}
}

// SendTrainingNotification is called every time an unhandled error occurs,
// particularly when that error happens as a result of the message a user sends
// to Ava
func (sg *MailClient) SendTrainingNotification(db *sqlx.DB, m *Msg) error {
	subj := "[Train] " + m.Sentence
	if len(m.Sentence) > 30 {
		subj = subj[0:27] + "..."
	}
	text := "<html><body>"
	text += fmt.Sprintf(
		"<p>We received a request that needs your help: %s</p>",
		m.Sentence)
	var url string
	if len(os.Getenv("ABOT_PORT")) > 0 {
		url = fmt.Sprintf("%s:%s/train/%d", os.Getenv("ABOT_URL"),
			os.Getenv("ABOT_PORT"), m.ID)
	} else {
		url = fmt.Sprintf("%s/train/%d", os.Getenv("ABOT_URL"), m.ID)
	}
	text += fmt.Sprintf(
		"<p><a href=\"%s\">Click here to help.</a></p>", url)
	text += "</body></html>"
	q := `SELECT name, email FROM users WHERE trainer IS TRUE`
	rows, err := db.Queryx(q)
	if err != nil {
		return err
	}
	defer rows.Close()
	user := &User{}
	for rows.Next() {
		if err = rows.Scan(&user.Name, &user.Email); err != nil {
			return err
		}
		if err := sg.Send(subj, text, user); err != nil {
			return err
		}
	}
	return nil
}

// Send a custom HTML email to any Contactable (user, vendor, admin, etc.) from
// Ava
func (sg *MailClient) Send(subj, html string, c Contactable) error {
	msg := sendgrid.NewMail()
	msg.SetFrom("ava@avabot.co")
	msg.SetFromName("Ava")
	msg.AddTo(c.GetEmail())
	msg.AddToName(c.GetName())
	msg.SetSubject(subj)
	msg.SetHTML(html)
	if err := sg.sgc.Send(msg); err != nil {
		return err
	}
	return nil
}

// NewMailClient returns a new MailClient initialized with any private API keys
// necessary to be immediately useful.
func NewMailClient() *MailClient {
	return &MailClient{
		sgc: sendgrid.NewSendGridClientWithApiKey(
			os.Getenv("SENDGRID_KEY"),
		),
	}
}
