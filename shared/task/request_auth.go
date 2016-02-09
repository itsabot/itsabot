package task

import (
	"errors"
	"regexp"
)

var regexNum = regexp.MustCompile(`\d+`)
var ErrNoAuth = errors.New("no auth found")
var ErrInvalidAuth = errors.New("invalid auth")

const (
	authStateStart float64 = iota
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

// TODO convert to new API
/*
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
	log.Println("REQUESTAUTH")
	log.Println("state", t.GetState())
	// check last authentication date and method
	authenticated, err := t.msg.User.IsAuthenticated(m)
	if err != nil {
		log.Println("err checking last authentication")
		return false, err
	}
	if authenticated {
		return true, nil
	}
	switch t.GetState() {
	case authStateStart:
		log.Println("hit authStateStart")
		return t.askUserForAuth(m)
	case authStateConfirm:
		log.Println("hit authStateConfirm")
		switch m {
		case MethodZip:
			log.Println("methodzip")
			zip5 := []byte(regexNum.FindString(t.msg.Sentence))
			if len(zip5) != 5 {
				return false, ErrInvalidAuth
			}
			q := `SELECT zip5hash FROM cards WHERE userid=$1`
			rows, err := t.db.Queryx(q, t.msg.User.ID)
			if err != nil {
				return false, err
			}
			defer rows.Close()
			for rows.Next() {
				log.Println("hasCard is true")
				var zip5hash []byte
				if err = rows.Scan(&zip5hash); err != nil {
					return false, err
				}
				err = bcrypt.CompareHashAndPassword(zip5hash,
					zip5)
				if err == bcrypt.ErrMismatchedHashAndPassword ||
					err == bcrypt.ErrHashTooShort {
					continue
				} else if err != nil {
					return false, err
				} else {
					q = `
						SELECT authorizationid
						FROM users
						WHERE id=$1`
					authID := &sql.NullInt64{}
					err = t.db.Get(authID, q,
						t.msg.User.ID)
					if err == sql.ErrNoRows {
						return false, ErrNoAuth
					}
					if err != nil {
						return false, err
					}
					q = `
						SELECT id FROM authorizations
						WHERE id=$1 AND authmethod>=$2`
					err = t.db.Get(authID, q, *authID,
						m)
					if err == sql.ErrNoRows {
						log.Println("no authorization")
						return false, ErrInvalidAuth
					}
					if err != nil {
						return false, err
					}
					if !authID.Valid {
						return false, ErrNoAuth
					}
					err = t.setAuthorized(authID)
					if err != nil {
						return false, err
					}
					return true, nil
				}
			}
			if rows.Err() != nil {
				return false, rows.Err()
			}
			return false, ErrInvalidAuth
		case MethodWebCache, MethodWebLogin:
			q := `SELECT authorizationid FROM users WHERE id=$1`
			var authID *sql.NullInt64
			err := t.db.Get(authID, q, t.msg.User.ID)
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
			err = t.db.Get(authID, q, authID.Int64, m)
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
		default:
			return false, errors.New("invalid auth state")
		}
	}
	return false, nil
}

// RequestPurchase will authenticate the user and then charge a card.
func (t *Task) RequestPurchase(m dt.Method, p *dt.Purchase) (bool, error) {
	t.typ = "Purchase"
	done, err := t.makePurchase(m, p)
	if done {
		t.setState(authStateStart)
	}
	return done, err
}

func (t *Task) askUserForAuth(m dt.Method) (bool, error) {
	log.Println("asking user for auth")
	cards, err := t.msg.User.GetCards(t.db)
	if err != nil {
		log.Println("err getting cards")
		return false, err
	}
	if len(cards) == 0 {
		log.Println("user has no cards")
		t.msg.Sentence = "Great! I'll need you to add your card here, first: https://avabot.co/?/cards/new. Let me know when you're done!"
		return false, nil
	}
	switch m {
	case MethodZip:
		t.msg.Sentence = "To do that, please confirm your billing zip code."
		log.Println("asking user for auth: zip code")
	case MethodWebCache:
		t.msg.Sentence = "To do that, please prove you're logged in: https://www.avabot.co/?/profile"
	case MethodWebLogin:
		if err := t.msg.User.DeleteSessions(t.db); err != nil {
			return false, err
		}
		t.msg.Sentence = "To do that, please log in to prove it's you: https://www.avabot.co/?/login"
	}
	tx, err := t.db.Beginx()
	if err != nil {
		return false, err
	}
	q := `INSERT INTO authorizations (authmethod) VALUES ($1) RETURNING id`
	var aid uint64
	if err = tx.QueryRowx(q, m).Scan(&aid); err != nil {
		log.Println("err scanning")
		return false, err
	}
	q = `UPDATE users SET authorizationid=$1 WHERE id=$2`
	if _, err = tx.Exec(q, aid, t.msg.User.ID); err != nil {
		log.Println("err updating")
		return false, err
	}
	if err = tx.Commit(); err != nil {
		log.Println("err committing")
		return false, err
	}
	log.Println("asked user for auth")
	t.setState(authStateConfirm)
	return false, nil
}

func (t *Task) setAuthorized(authID *sql.NullInt64) error {
	tx, err := t.db.Beginx()
	if err != nil {
		return err
	}
	q := `UPDATE users SET authorizationid=NULL WHERE id=$1`
	if _, err = tx.Exec(q, t.msg.User.ID); err != nil {
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

func (t *Task) makePurchase(m dt.Method, p *dt.Purchase) (bool, error) {
	authenticated, err := t.RequestAuth(m)
	if err != nil {
		return false, err
	}
	if !authenticated {
		return false, nil
	}
	desc := fmt.Sprintf("Purchase for $%.2f", float64(p.Total)/100)
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
	chargeParams := &stripe.ChargeParams{
		Amount:   p.Total,
		Currency: "usd",
		Desc:     desc,
		Customer: t.msg.User.StripeCustomerID,
	}
	if _, err := charge.New(chargeParams); err != nil {
		return false, err
	}
	// TODO move functionality like sending emails to Ava core. Should pkgs
	// should request content be sent to users through Ava core?
	if err := t.sg.SendVendorRequest(p); err != nil {
		return false, err
	}
	if err := t.sg.SendPurchaseConfirmation(p); err != nil {
		return false, err
	}
	if err := p.UpdateEmailsSent(); err != nil {
		return false, err
	}
	return true, nil
}
*/
