package pkg

import (
	"encoding/json"
	"errors"
	"net"
	"net/rpc"
	"os"
	"strconv"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
)

type PkgWrapper struct {
	P         *Pkg
	RPCClient *rpc.Client
}

// Pkg holds config options for any Ava package. Name must be globally unique.
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Pkg struct {
	Config  PkgConfig
	Vocab   *dt.Vocab
	Trigger *nlp.StructuredInput
}

type PkgConfig struct {
	Name          string
	ServerAddress string
	Route         string
	Port          int
}

type Ava int

var client *rpc.Client
var db *sqlx.DB
var (
	ErrMissingPackageName = errors.New("missing package name")
	ErrMissingPort        = errors.New("missing package port")
	ErrMissingTrigger     = errors.New("missing package trigger")
)

func NewPackage(name string, port int, trigger *nlp.StructuredInput) (
	*Pkg, error) {
	return NewPackageWithServer(name, "", port, trigger)
}

func NewPackageWithServer(name, serverAddr string, port int,
	trigger *nlp.StructuredInput) (*Pkg, error) {
	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
	}
	if trigger == nil {
		return &Pkg{}, ErrMissingTrigger
	}
	c := PkgConfig{
		Name:          name,
		Port:          port,
		ServerAddress: serverAddr,
	}
	return &Pkg{Config: c, Trigger: trigger}, nil
}

// Register with Ava to begin communicating over RPC.
func (p *Pkg) Register(pkgT interface{}) error {
	log.SetLevel(log.DebugLevel)
	log.WithFields(log.Fields{
		"port": p.Config.Port + 1,
		"pkg":  p.Config.Name,
	}).Debugln("connecting")
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p.Config.Port+1))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}
	if err := rpc.Register(pkgT); err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln(err)
		return err
	}
	client, err = rpc.Dial("tcp", ":"+strconv.Itoa(port+1))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln(err)
		return err
	}
	var notused string
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln("calling", err)
		return err
	}
	log.WithField("pkg", p.Config.Name).Debugln("connected")
	db, err = ConnectDB()
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln("connectDB", err)
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.WithField("pkg", p.Config.Name).Fatalln(err)
		}
		go rpc.ServeConn(conn)
	}
	return nil
}

// SaveMsg is handled in shared/pkg because rpc gob encoding doesn't work
// well with arbitrary interface{} types. Since a Message has a nested
// map[string]interface{} type, jsonrpc wouldn't work either. Since it's not
// easy to transfer the data from the package back to Ava for saving, the
// packages will be responsible for saving their own messages. This is not
// ideal, but it'll work for now.
func (p *Pkg) SaveMsg(respMsg *dt.RespMsg, m *dt.Msg) error {
	if len(m.Sentence) == 0 {
		log.Warnln("response sentence empty. skipping save")
		return nil
	}
	state, err := json.Marshal(m.State)
	if err != nil {
		log.WithFields(log.Fields{
			"pkg": p.Config.Name,
			"fn":  "SaveMsg",
		}).Errorln(err)
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		log.WithFields(log.Fields{
			"pkg": p.Config.Name,
			"fn":  "SaveMsg",
		}).Errorln(err)
		return err
	}
	// TODO change to use PG 9.5's UPSERT
	q := `SELECT COUNT(*) FROM states WHERE userid=$1 AND pkgname=$2`
	var tmp uint64
	if err = tx.Get(&tmp, q, m.User.ID, p.Config.Name); err != nil {
		log.WithFields(log.Fields{
			"pkg": p.Config.Name,
			"fn":  "SaveMsg",
		}).Errorln(err)
		return err
	}
	// TODO remove RETURNING id, since it's now unused
	if tmp == 0 {
		q = `INSERT INTO states (userid, state, pkgname)
		     VALUES ($1, $2, $3) RETURNING id`
		row := tx.QueryRowx(q, m.User.ID, state, p.Config.Name)
		if err = row.Scan(&tmp); err != nil {
			log.WithFields(log.Fields{
				"pkg": p.Config.Name,
				"fn":  "SaveMsg",
			}).Errorln(err)
			return err
		}
	} else {
		q = `UPDATE states
		     SET state=$1, updatedat=CURRENT_TIMESTAMP 
		     WHERE userid=$2 AND pkgname=$3 RETURNING id`
		err = tx.QueryRowx(q, state, m.User.ID, p.Config.Name).Scan(&tmp)
		if err != nil {
			log.WithFields(log.Fields{
				"pkg": p.Config.Name,
				"fn":  "SaveMsg",
			}).Errorln(err)
			return err
		}
	}
	q = `INSERT INTO messages (userid, sentence, route, avasent)
	     VALUES ($1, $2, $3, TRUE)
	     RETURNING id`
	err = tx.QueryRowx(q, m.User.ID, m.Sentence, m.Route).Scan(&tmp)
	if err != nil {
		log.WithFields(log.Fields{
			"pkg": p.Config.Name,
			"fn":  "SaveMsg",
		}).Errorln(err)
		return err
	}
	if err = tx.Commit(); err != nil {
		log.WithFields(log.Fields{
			"pkg": p.Config.Name,
			"fn":  "SaveMsg",
		}).Errorln(err)
		return err
	}
	(*respMsg).MsgID = tmp
	log.Debugln("respMsg msgid", tmp)
	(*respMsg).Sentence = m.Sentence
	return nil
}

func ConnectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=ava sslmode=disable")
	}
	if err != nil {
		log.WithFields(log.Fields{
			"fn": "ConnectDB",
		}).Errorln(err)
	}
	return db, err
}
