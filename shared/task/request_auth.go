package task

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go/charge"
	"github.com/avabot/ava/shared/datatypes"
)

var regexNum = regexp.MustCompile(`\d+`)
var ErrNoAuth = errors.New("no auth found")
var ErrInvalidAuth = errors.New("invalid auth")

const (
	authStateNone float64 = iota
	authStateConfirm
)

const (
	// MethodZip requires the zip code associated with a credit card on
	// file. The user will be asked for a credit card if not on file.
	MethodZip = iota + 1

	// MethodWebCache allows a user to authenticate by clicking a link. If
	// their browser cookies have them already logged into Ava, they will be
	// authenticated. If they are not currently logged into Ava, they will
	// be asked to login. Once logged in, they will be authenticated.
	MethodWebCache

	// MethodWebLogin requires the user login to Ava on the web interface
	// using their username and password. This is the most secure option,
	// as it ensures no one has stolen the device or session token of a
	// user.
	MethodWebLogin
)

// RequestAuth ensures you're speaking to the correct user. Select the LOWEST
// level of authentication you'll allow based on a tolerance for fraud weighed
// against the convenience of the user experience. Methods are organized in
// least-secure to most-secure order. Therefore, MethodZip will allow any auth
// method, whereas MethodWebCache will only allow MethodWebCache and above. Ava
// will IMPROVE the quality of the authentication automatically whenever
// possible, selecting the highest authentication method for which the user has
// recently authenticated. Note that you'll never have to call RequestAuth in a
// Purchase flow. In order to drive a customer purchase, call Purchase directly,
// which will also authenticate the user.
func (t *Task) RequestAuth(m dt.Method) (bool, error) {
	t.typ = "Auth"
	// check last authentication date and method
	authenticated, err := t.ctx.Msg.User.IsAuthenticated(m)
	if err != nil {
		return false, err
	}
	if authenticated {
		return true, nil
	}
	switch t.getState() {
	case authStateNone:
		return t.askUserForAuth(m)
	case authStateConfirm:
		switch m {
		case MethodZip:
			zip5 := regexNum.FindString(t.ctx.Msg.Input.Sentence)
			if len(zip5) != 5 {
				return false, errors.New("zip code not found")
			}
			q := `
				SELECT COUNT(id) FROM addresses
				WHERE userid=$1 AND cardid<>NULL AND zip5=$2`
			var count uint64
			err := t.ctx.DB.Select(&count, q, t.ctx.Msg.User.ID, zip5)
			if err != nil {
				return false, err
			}
			if count > 0 {
				q = `
					SELECT authorizationid FROM users
					WHERE id=$1`
				var authID *sql.NullInt64
				err = t.ctx.DB.Get(authID, q, t.ctx.Msg.User.ID)
				if err == sql.ErrNoRows {
					return false, ErrNoAuth
				}
				if err != nil {
					return false, err
				}
				if !authID.Valid {
					return false, ErrNoAuth
				}
				if err = t.setAuthorized(authID); err != nil {
					return false, err
				}
				return true, nil
			} else {
				return false, ErrInvalidAuth
			}
		case MethodWebCache, MethodWebLogin:
			q := `SELECT authorizationid FROM users WHERE id=$1`
			var authID *sql.NullInt64
			err := t.ctx.DB.Get(authID, q, t.ctx.Msg.User.ID)
			if err != nil {
				return false, err
			}
			if !authID.Valid {
				return false, ErrNoAuth
			}
			q = `
				SELECT id FROM authorizations
				WHERE id=$1
					AND authmethod>=$2
					AND authorizedat<>NULL`
			err = t.ctx.DB.Get(authID, q, authID.Int64, m)
			if err != nil {
				return false, err
			}
			if !authID.Valid {
				return false, ErrNoAuth
			}
			if err = t.setAuthorized(authID); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	tmp := fmt.Sprintf("invalid auth state: %f", t.getState())
	return false, errors.New(tmp)
}

// RequestPurchase will authenticate the user and then charge a card.
func (t *Task) RequestPurchase(m dt.Method, prds []dt.Product,
	p *dt.Purchase) (bool, error) {
	if os.Getenv("AVA_ENV") == "production" {
		authenticated, err := t.RequestAuth(m)
		if err != nil {
			return false, err
		}
		if !authenticated {
			return false, nil
		}
	}
	desc := fmt.Sprintf("Purchase for %.2f", float64(p.Total)/100)
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
	chargeParams := &stripe.ChargeParams{
		Amount:   p.Total,
		Currency: "usd",
		Desc:     desc,
		Customer: t.ctx.Msg.User.StripeCustomerID,
	}
	if _, err := charge.New(chargeParams); err != nil {
		return false, err
	}
	if err := t.ctx.SG.SendVendorRequest(prds, p); err != nil {
		return false, err
	}
	if err := t.ctx.SG.SendPurchaseConfirmation(prds, p); err != nil {
		return false, err
	}
	if err := p.UpdateEmailsSent(); err != nil {
		return false, err
	}
	return true, nil
}

func (t *Task) askUserForAuth(m dt.Method) (bool, error) {
	switch m {
	case MethodZip:
		t.resp.Sentence = "Please confirm your billing zip code"
	case MethodWebCache:
		t.resp.Sentence = "Please prove you're logged in: https://www.avabot.com/?/profile"
	case MethodWebLogin:
		if err := t.ctx.Msg.User.DeleteSessions(t.ctx.DB); err != nil {
			return false, err
		}
		t.resp.Sentence = "Please log in to prove it's you: https://www.avabot.com/?/login"
	}
	tx, err := t.ctx.DB.Beginx()
	if err != nil {
		return false, err
	}
	q := `INSERT INTO authorizations (authmethod) VALUES ($1) RETURNING id`
	var aid int
	if err = tx.QueryRowx(q, m).Scan(&aid); err != nil {
		return false, err
	}
	q = `UPDATE users SET authorizationid=$1 WHERE id=$2`
	if _, err = tx.Exec(q, aid, t.ctx.Msg.User.ID); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return false, nil
}

func (t *Task) setAuthorized(authID *sql.NullInt64) error {
	tx, err := t.ctx.DB.Beginx()
	if err != nil {
		return err
	}
	q := `UPDATE users SET authorizationid=NULL WHERE id=$1`
	if _, err = tx.Exec(q, t.ctx.Msg.User.ID); err != nil {
		return err
	}
	q = `
		UPDATE authorizations
		SET authorizedat=CURRENT_TIMESTAMP
		WHERE id=$1`
	if _, err = tx.Exec(q, authID.Int64); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
