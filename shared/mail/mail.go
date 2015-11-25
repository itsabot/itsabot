package mail

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/sendgrid/sendgrid-go"
	"github.com/avabot/ava/shared/datatypes"
)

type Client struct {
	sgc *sendgrid.SGClient
}

// TODO add shipping information and purchase identifier (UUID)
func (sg *Client) SendPurchaseConfirmation(products []dt.Product, price uint64,
	shippingInCents uint64, taxInCents uint64, u *dt.User) error {
	delivery := time.Now().Add(7 * 24 * time.Hour)
	delS := delivery.Format("Monday Jan 2, 2006")
	subj := "Order confirmation"
	text := "<html><body>"
	text += fmt.Sprintf("<p>Hi %s:</p>", u.Name)
	text += "<p>Here's a quick order summary for your records. You bought:</p>"
	text += "<ul>"
	productTotal := 0.0
	for _, product := range products {
		p := float64(product.Price) / 100
		text += fmt.Sprintf("<li>$%.2f - %s</li>", p, product.Name)
		productTotal += p
	}
	text += "</ul><table>"
	text += fmt.Sprintf("<tr><td>Subtotal: </td><td>$%.2f</td></tr>",
		productTotal)
	text += fmt.Sprintf("<tr><td>Shipping: </td><td>$%.2f</td></tr>",
		shippingInCents)
	text += fmt.Sprintf("<tr><td>Tax: </td><td>$%.2f</td></tr>",
		taxInCents)
	text += "<tr><td>My fee: </td><td>$0.00 (always)</td></tr>"
	text += fmt.Sprintf("<tr><td><b>Total: </b></td><td><b>$%.2f</b></td></tr>",
		float64(price)/100)
	text += "</table>"
	text += fmt.Sprintf("<p>Expected delivery before %s</p>", delS)
	text += "<p>Glad I could help! :)</p><p>- Ava</p>"
	text += "</body></html>"
	return sg.Send(subj, text, u)
}

func (c *Client) Send(subj, html string, u *dt.User) error {
	msg := sendgrid.NewMail()
	msg.SetFrom("ava@avabot.com")
	msg.SetFromName("Ava")
	msg.AddTo(u.Email)
	msg.AddToName(u.Name)
	msg.SetSubject(subj)
	msg.SetHTML(html)
	if err := c.sgc.Send(msg); err != nil {
		return err
	}
	return nil
}

func NewClient() *Client {
	log.Println("sendgrid", os.Getenv("SENDGRID_KEY"))
	return &Client{
		sgc: sendgrid.NewSendGridClientWithApiKey(
			os.Getenv("SENDGRID_KEY"),
		),
	}
}
