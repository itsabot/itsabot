package pkg

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/shared/datatypes"
)

type PkgWrapper struct {
	P         *Pkg
	RPCClient *rpc.Client
}

// Pkg holds config options for any Ava package. Name must be globally unique
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Pkg struct {
	Config  PkgConfig
	Trigger *dt.StructuredInput
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

func NewPackage(name string, port int, trigger *dt.StructuredInput) (
	*Pkg, error) {
	return NewPackageWithServer(name, "", port, trigger)
}

func NewPackageWithServer(name, serverAddr string, port int,
	trigger *dt.StructuredInput) (*Pkg, error) {
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
	log.Println("connecting to port", p.Config.Port+1, "for", p.Config.Name)
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p.Config.Port+1))
	if err != nil {
		log.Fatalln("rpc listen:", err, p.Config.Name)
	}
	if err := rpc.Register(pkgT); err != nil {
		log.Fatalln(err, p.Config.Name)
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		return err
	}
	client, err = rpc.Dial("tcp", ":"+strconv.Itoa(port+1))
	if err != nil {
		return err
	}
	var notused string
	log.Println("calling register", p.Config.Name)
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil {
		log.Println("err: registering package", p.Config.Name, err)
		return err
	}
	log.Println("connected with ava", p.Config.Name)
	db, err = ConnectDB()
	if err != nil {
		return err
	}
	log.Println("connected with database", p.Config.Name)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go rpc.ServeConn(conn)
	}
	return nil
}

// SaveResponse is handled in shared/pkg because rpc gob encoding doesn't work
// well with arbitrary interface{} types. Since a Response had a nested
// map[string]interface{} type, jsonrpc wouldn't work either. Since it's not
// easy to transfer the data from the package back to Ava for saving, the
// packages will be responsible for saving their own responses. This is not
// ideal, but it'll work for now.
func SaveResponse(respMsg *dt.RespMsg, r *dt.Resp) error {
	q := `
		INSERT INTO responses (userid, inputid, sentence, route, state)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	state, err := json.Marshal(r.State)
	if err != nil {
		return err
	}
	var rid int
	err = db.QueryRowx(q, r.UserID, r.InputID, r.Sentence, r.Route, state).
		Scan(&rid)
	if err != nil {
		log.Println("ERR SAVING", err)
		return err
	}
	log.Println("saved route", r.Route)
	respMsg.ResponseID = rid
	respMsg.Sentence = r.Sentence
	return nil
}

func ConnectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	return db, err
}
