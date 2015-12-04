package dt

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/sendgrid/sendgrid-go"
)

type MailClient struct {
	sgc *sendgrid.SGClient
}

// TODO add shipping information and purchase identifier (UUID)
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
	text += fmt.Sprintf("Your order confirmation number is %s.</p>", p.ID)
	text += "<p>Glad I could help! :)</p><p>- Ava</p>"
	text += "</body></html>"
	return sg.Send(subj, text, p.User)
}

// TODO add shipping information and purchase identifier (UUID)
func (sg *MailClient) SendVendorRequest(p *Purchase) error {
	if len(p.ProductSels) == 0 {
		return errors.New("empty products slice in vendor request")
	}
	var subj string
	if os.Getenv("AVA_ENV") == "production" {
		subj = fmt.Sprintf("Order Request: #%s", p.DisplayID())
	} else {
		subj = fmt.Sprintf("[TEST - PLEASE IGNORE] Order Request: #%s", p.ID)
		(*p.Vendor).ContactName = "Evan"
		(*p.Vendor).ContactEmail = "egtann@gmail.com"
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

func (sg *MailClient) Send(subj, html string, c Contactable) error {
	msg := sendgrid.NewMail()
	msg.SetFrom("ava@avabot.com")
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

func NewMailClient() *MailClient {
	log.Println("sendgrid", os.Getenv("SENDGRID_KEY"))
	return &MailClient{
		sgc: sendgrid.NewSendGridClientWithApiKey(
			os.Getenv("SENDGRID_KEY"),
		),
	}
}
