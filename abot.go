package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/codegangsta/cli"
	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/websocket"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/pkg"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var ner core.Classifier
var ws = websocket.NewAtomicWebSocketSet()
var offensive map[string]struct{}
var (
	errInvalidUserPass = errors.New("Invalid username/password combination")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.DebugOn(true)
	app := cli.NewApp()
	app.Name = "abot"
	app.Usage = "digital assistant framework"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "run server",
			Action: func(c *cli.Context) {
				if err := startServer(); err != nil {
					l := log.New("")
					l.SetFlags(0)
					l.Fatalf("could not start server\n%s", err)
				}
			},
		},
		{
			Name:    "package",
			Aliases: []string{"p"},
			Usage:   "manage and install packages from packages.json",
			Subcommands: []cli.Command{
				{
					Name:  "install",
					Usage: "download and install packages listed in packages.json",
					Action: func(c *cli.Context) {
						if err := installPackages(); err != nil {
							l := log.New("")
							l.SetFlags(0)
							l.Fatalf("could not start server\n%s", err)
						}
					},
				},
			},
		},
		{
			Name:    "console",
			Aliases: []string{"c"},
			Usage:   "communicate with a running abot server",
			Action: func(c *cli.Context) {
				if err := startConsole(c); err != nil {
					l := log.New("")
					l.SetFlags(0)
					l.Fatalf("could not start console\n%s", err)
				}
			},
		},
	}
	app.Action = func(c *cli.Context) {
		cli.ShowAppHelp(c)
	}
	app.Run(os.Args)
}

// startServer initializes any clients that are needed and boots packages
func startServer() error {
	var err error
	db, err = pkg.ConnectDB()
	if err != nil {
		log.Fatal("could not connect to database", err)
	}
	if err = checkRequiredEnvVars(); err != nil {
		return err
	}
	addr, err := core.BootRPCServer()
	if err != nil {
		return err
	}
	go func() {
		if err := core.BootDependencies(addr); err != nil {
			log.Debug("could not boot dependency", err)
		}
	}()
	ner, err = core.BuildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	offensive, err = core.BuildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	e := echo.New()
	initRoutes(e)
	log.Info("booted ava http server")
	e.Run(":" + os.Getenv("PORT"))
	return nil
}

func startConsole(c *cli.Context) error {
	args := c.Args()
	if len(args) != 2 {
		return errors.New("usage: abot console abot-address user-phone")
	}
	// Capture ^C interrupt to add a newline
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		for _ = range sig {
			fmt.Println("")
			os.Exit(0)
		}
	}()
	base := "http://" + args[0] + "?flexidtype=2&flexid=" + url.QueryEscape(args[1]) + "&cmd="
	scanner := bufio.NewScanner(os.Stdin)
	// Test connection
	req, err := http.NewRequest("GET", base, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if err = resp.Body.Close(); err != nil {
		return err
	}
	fmt.Print("> ")
	for scanner.Scan() {
		cmd := scanner.Text()
		req, err := http.NewRequest("POST", base+url.QueryEscape(cmd), nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		fmt.Println(string(body))
		fmt.Print("> ")
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func installPackages() error {
	l := log.New("")
	l.SetFlags(0)
	// Delete all packages in the /packages dir, packages.lock
	if err := os.RemoveAll("./packages"); err != nil {
		l.Fatal(err)
	}
	err := os.Remove("./packages.lock")
	if err != nil && err.Error() !=
		"remove ./packages.lock: no such file or directory" {
		l.Fatal(err)
	}
	// Read packages.json, unmarshal into struct
	contents, err := ioutil.ReadFile("./packages.json")
	if err != nil {
		l.Fatal(err)
	}
	var packages packageJSON
	if err = json.Unmarshal(contents, &packages); err != nil {
		l.Fatal(err)
	}
	// Remake the /packages dir
	if err = os.Mkdir("./packages", 0775); err != nil {
		l.Fatal(err)
	}
	// Fetch packages
	l.Info("Fetching", len(packages.Dependencies), "packages...")
	var wg sync.WaitGroup
	wg.Add(len(packages.Dependencies))
	rand.Seed(time.Now().UTC().UnixNano())
	for url, _ := range packages.Dependencies {
		go func(url string) {
			// Download source as a zip
			resp, err := http.Get("https://" + url + "/archive/master.zip")
			if err != nil {
				l.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				e := fmt.Sprintf("err fetching package %s: %d", url,
					resp.StatusCode)
				l.Fatal(errors.New(e))
			}
			fiName := "tmp_" + randSeq(8) + ".zip"
			fpZip := filepath.Join("./packages", fiName)
			out, err := os.Create(fpZip)
			if err != nil {
				l.Fatal(err)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				_ = out.Close()
				l.Fatal(err)
			}
			// Unzip source to directory
			if err = unzip(fpZip, "./packages"); err != nil {
				_ = out.Close()
				l.Fatal(err)
			}
			// Close zip file
			if err = out.Close(); err != nil {
				l.Fatal(err)
			}
			// Delete zip file
			if err = os.Remove(fpZip); err != nil {
				l.Fatal(err)
			}

			// Anonymously increment the package's download count
			// at itsabot.org
			p := struct {
				Path string
			}{Path: url}
			byt, err := json.Marshal(p)
			if err != nil {
				l.Info("WARN:", err)
				wg.Done()
				return
			}
			var u string
			if len(os.Getenv("ITSABOT_URL")) > 0 {
				u = os.Getenv("ITSABOT_URL") + "/api/packages.json"
			} else {
				u = "https://www.itsabot.org/api/packages.json"
			}
			resp, err = http.Post(u, "application/json",
				bytes.NewBuffer(byt))
			if err != nil {
				l.Info("WARN:", err)
				wg.Done()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				l.Info("WARN: %d - %s\n", resp.StatusCode,
					resp.Status)
			}
			wg.Done()
		}(url)
	}
	wg.Wait()
	l.Info("Success!")
	return nil
}

func checkRequiredEnvVars() error {
	port := os.Getenv("PORT")
	_, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("PORT is not set to an integer")
	}
	base := os.Getenv("ABOT_URL")
	l := len(base)
	if l == 0 {
		return errors.New("ABOT_URL not set")
	}
	if l < 4 || base[0:4] != "http" {
		return errors.New("ABOT_URL invalid. Must include http/https")
	}
	// TODO Check for ABOT_DATABASE_URL if ABOT_ENV==production
	return nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type packageJSON struct {
	Dependencies map[string]string
}

// From https://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()
	os.MkdirAll(dest, 0755)
	for _, f := range r.File {
		err = extractAndWriteFile(dest, f)
		if err != nil {
			return err
		}
	}
	return nil
}

// From https://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
func extractAndWriteFile(dest string, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			panic(err)
		}
	}()
	path := filepath.Join(dest, f.Name)
	if f.FileInfo().IsDir() {
		os.MkdirAll(path, f.Mode())
	} else {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()
		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

// From https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
