package main

import (
	"bufio"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/codegangsta/cli"
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var bayes *bayesian.Classifier

var (
	ErrInvalidClass        = errors.New("invalid class")
	ErrInvalidCommand      = errors.New("invalid command")
	ErrInvalidOddParameter = errors.New("parameter count must be even")
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	app := cli.NewApp()
	app.Name = "ava"
	app.Usage = "general purpose ai platform"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "server, s",
			Usage: "run server",
		},
		cli.StringFlag{
			Name:  "port, p",
			Value: "4000",
			Usage: "set port for server",
		},
		cli.BoolFlag{
			Name:  "install, i",
			Usage: "install packages in package.conf",
		},
	}
	app.Action = func(c *cli.Context) {
		showHelp := true
		if c.Bool("install") {
			log.Println("TODO: Install packages")
			showHelp = false
		}
		if c.Bool("server") {
			startServer(c.String("port"))
			showHelp = false
		}
		if showHelp {
			cli.ShowAppHelp(c)
		}
	}
	app.Run(os.Args)
}

func startServer(port string) {
	var err error
	db = connectDB()
	// Load packages
	/*
		bc, err := loadConfig("packages.conf")
		if err != nil {
			log.Fatalln("could not load package", err)
		}
	*/
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Fatalln("error loading classifier", err)
	}
	/*
		si, err := classify(bayes, "train _C(Order) _O(an Uber).")
		if err != nil {
			log.Fatalln("error classifying sentence", err)
		}
		log.Println(si)
	*/

	e := echo.New()
	initRoutes(e)
	e.Run(":" + port)
}

// route will determine what kind of request it is based on text.
// Content can belong to multiple classes. Route returns []string,
// which is used by os.Exec to run the commands.
func route(content string) []string {
	var pkgs []string

	return pkgs
}

func connectDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres",
		"user=egtann dbname=ava sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	return db
	// Run schema while testing
}

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger())
	e.Use(mw.Gzip())
	e.Use(mw.Recover())
	e.Post("/", handlerMain)
}

func handlerMain(c *echo.Context) error {
	var ret string
	var err error
	si := &StructuredInput{}
	cmd := c.Form("cmd")
	if len(cmd) == 0 {
		return ErrInvalidCommand
	}
	if strings.ToLower(cmd)[0:5] == "train" {
		if err := train(bayes, cmd[7:]); err != nil {
			return err
		}
		goto Response
	}
	si, err = classify(bayes, cmd)
	if err != nil {
		log.Fatalln("error classifying sentence", err)
	}
	ret = si.String()
	// Update state machine
	// Save last command (save structured input)
	// Send to packages
	/*
		for _, pkg := range pkgs {
			path := path.Join("packages", pkg)
			out, err := exec.Command(path, cmd).CombinedOutput()
			if err != nil {
				log.Println("unable to run package", err)
				return err
			}
			ret += string(out) + "\n\n"
		}
	*/

Response:
	err = c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}
	return nil
}

func loadClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	var err error

	filename := path.Join("data", "common", "bayes.dat")
	c, err = bayesian.NewClassifierFromFile(filename)
	if err.Error() == "open data/common/bayes.dat: no such file or directory" {
		c, err = buildClassifier(c)
		if err != nil {
			return c, err
		}
		log.Println("c2", c)
	} else if err != nil {
		log.Println("!!", err)
		return c, err
	}
	return c, nil
}

func buildClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	c = bayesian.NewClassifier(Command, Actor, Object, Time, None)
	filename := path.Join("training", "imperative.txt")
	fi, err := os.Open(filename)
	if err != nil {
		return c, err
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	line := 1
	for scanner.Scan() {
		if err := trainClassifier(c, scanner.Text()); err != nil {
			log.Fatalln("line", line, "::", err)
		}
		line++
	}
	if err = scanner.Err(); err != nil {
		return c, err
	}
	if err = c.WriteClassesToFile(path.Join("data")); err != nil {
		return c, err
	}
	return c, nil
}

func trainClassifier(c *bayesian.Classifier, s string) error {
	if len(s) == 0 {
		return nil
	}
	if s[0] == '/' {
		return nil
	}
	ws, err := extractFields(s)
	if err != nil {
		return err
	}
	l := len(ws)
	for i := 0; i < l; i++ {
		var word2 string
		var word3 string
		word1, entity, err := extractEntity(ws[i])
		if err != nil {
			return err
		}
		if entity == "" {
			continue
		}
		trigram := word1
		if i+1 < l {
			word2, _, err = extractEntity(ws[i+1])
			if err != nil {
				return err
			}
			trigram += " " + word2
		}
		if i+2 < l {
			word3, _, err = extractEntity(ws[i+2])
			if err != nil {
				return err
			}
			trigram += " " + word3
		}
		c.Learn([]string{word1}, entity)
		if word2 != "" {
			c.Learn([]string{word1 + " " + word2}, entity)
		}
		if word3 != "" {
			c.Learn([]string{trigram}, entity)
		}
	}
	return nil
}

func extractFields(s string) ([]string, error) {
	var ss []string

	if len(s) == 0 {
		return ss, errors.New("sentence too short to classify")
	}
	wordBuf := ""
	ws := strings.Fields(s)
	for _, w := range ws {
		r, _ := utf8.DecodeRuneInString(w)
		if r == '_' {
			r, _ = utf8.DecodeRuneInString(w[3:])
		}
		if unicode.IsNumber(r) {
			wordBuf += w + " "
			continue
		}
		word, _, err := extractEntity(w)
		if err != nil {
			return ss, err
		}
		switch strings.ToLower(word) {
		// Articles and prepositions
		case "a", "an", "the", "before", "at", "after", "next":
			wordBuf += w + " "
		default:
			ss = append(ss, wordBuf+w)
			wordBuf = ""
		}
	}
	return ss, nil
}

func classify(c *bayesian.Classifier, s string) (*StructuredInput, error) {
	si := &StructuredInput{}
	ws, err := extractFields(s)
	if err != nil {
		return si, err
	}
	var wc []wordclass
	for i := range ws {
		tmp, err := classifyTrigram(c, ws, i)
		if err != nil {
			return si, err
		}
		wc = append(wc, tmp)
	}
	if err = si.Add(wc); err != nil {
		return si, err
	}
	return si, nil
}

func extractEntity(w string) (string, bayesian.Class, error) {
	w = strings.TrimRight(w, ").,;")
	if w[0] != '_' {
		return w, "", nil
	}
	switch w[1] {
	case 'C': // Command
		return w[3:], Command, nil
	case 'O': // Object
		return w[3:], Object, nil
	case 'A': // Actor
		return w[3:], Actor, nil
	case 'T': // Time
		return w[3:], Time, nil
	case 'N': // None
		return w[3:], None, nil
	}
	return w, "", errors.New("syntax error in entity")
}

func classifyTrigram(c *bayesian.Classifier, ws []string, i int) (wordclass,
	error) {

	var wc wordclass
	l := len(ws)
	word1, _, err := extractEntity(ws[i])
	if err != nil {
		return wc, err
	}
	trigram := word1
	var word2 string
	var word3 string
	if i+1 < l {
		word2, _, err = extractEntity(ws[i+1])
		if err != nil {
			return wc, err
		}
		trigram += " " + word2
	}
	if i+2 < l {
		word3, _, err = extractEntity(ws[i+2])
		if err != nil {
			return wc, err
		}
		trigram += " " + word3
	}
	_, likely, _ := c.LogScores([]string{trigram})
	return wordclass{word1, likely}, nil
}
