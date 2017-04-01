package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	lg "log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
	"unicode"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/websocket"

	"github.com/codegangsta/cli"
	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/plugin"
	"github.com/itsabot/itsabot.org/socket"
	_ "github.com/lib/pq" // Postgres driver
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetDebug(os.Getenv("ABOT_DEBUG") == "true")
	if err := core.LoadEnvVars(); err != nil {
		log.Fatal(err)
	}
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "new",
			Usage: "generate a new abot",
			Action: func(c *cli.Context) error {
				l := log.New("")
				l.SetFlags(0)
				if len(c.Args()) != 1 {
					l.Fatal("usage: abot new {name}")
				}
				if strings.Contains(c.Args().First(), " ") {
					l.Fatal("name cannot include a space. use camelCase")
				}
				err := newAbot(l, c.Args().First(), c.Args().Get(1))
				if err != nil {
					l.Fatalf("could not build new abot. %s", err)
				}
				l.Info("Success. Created " + c.Args().First())
				return nil
			},
		},
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "run server",
			Action: func(c *cli.Context) error {
				l := log.New("")
				l.SetFlags(0)
				cmd := exec.Command("/bin/sh", "-c", "go build")
				out, err := cmd.CombinedOutput()
				if err != nil {
					l.Info(string(out))
					return err
				}
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				_, file := filepath.Split(dir)
				cmd = exec.Command("/bin/sh", "-c", "./"+file)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err = cmd.Start(); err != nil {
					return err
				}
				return cmd.Wait()
			},
		},
		{
			Name:    "install",
			Aliases: []string{"i"},
			Usage:   "download and install plugins listed in plugins.json",
			Action: func(c *cli.Context) error {
				errChan := make(chan errMsg)
				l := log.New("")
				l.SetFlags(0)
				l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")
				go func() {
					select {
					case errC := <-errChan:
						if errC.err == nil {
							// Success
							l.Info(errC.msg)
							os.Exit(0)
						}

						// Plugins install failed, so remove incomplete plugins.go file
						errR := os.Remove("plugins.go")
						if errR != nil && !os.IsNotExist(errR) {
							l.Info("could not remove plugins.go file.", errR)
						}
						if len(errC.msg) > 0 {
							l.Fatalf("could not install plugins.\n%s\n%s", errC.msg, errC.err)
						}
						l.Fatalf("could not install plugins.\n%s", errC.err)
					}
				}()
				installPlugins(l, errChan)
				for {
					// Keep process running until a message is received
				}
			},
		},
		{
			Name:  "search",
			Usage: "search plugins indexed on itsabot.org",
			Action: func(c *cli.Context) error {
				l := log.New("")
				l.SetFlags(0)
				args := c.Args()
				if len(args) == 0 || len(args) > 2 {
					l.Fatal(errors.New(`usage: abot plugin search {term}`))
				}
				if err := searchPlugins(args.First()); err != nil {
					l.Fatalf("could not start console\n%s", err)
				}
				return nil
			},
		},
		{
			Name:    "update",
			Aliases: []string{"u", "upgrade"},
			Usage:   "update and install plugins listed in plugins.json",
			Action: func(c *cli.Context) error {
				updatePlugins()
				return nil
			},
		},
		{
			Name:    "publish",
			Aliases: []string{"p"},
			Usage:   "publish a plugin to itsabot.org",
			Action: func(c *cli.Context) error {
				publishPlugin(c)
				return nil
			},
		},
		{
			Name:    "test",
			Aliases: []string{"t"},
			Usage:   "tests plugins",
			Action: func(c *cli.Context) error {
				lg.SetFlags(0)
				count, err := testPlugin()
				if err != nil {
					return err
				}
				if count == 0 {
					lg.Println("No tests found. Did you run \"abot install\"?")
					return nil
				}
				lg.Printf("Success (%d tests).\n", count)
				return nil
			},
		},
		{
			Name:    "generate",
			Aliases: []string{"g"},
			Usage:   "generate plugin scaffolding",
			Action: func(c *cli.Context) error {
				l := log.New("")
				l.SetFlags(0)
				args := c.Args()
				if len(args) != 1 {
					l.Fatal(errors.New(`usage: abot generate {name}`))
				}
				generatePlugin(l, args.First())
				l.Info("Created", args.First(), "in",
					filepath.Join(os.Getenv("PWD"), args.First()))
				return nil
			},
		},
		{
			Name:    "login",
			Aliases: []string{"l"},
			Usage:   "log into itsabot.org to enable publishing plugins",
			Action: func(c *cli.Context) error {
				login()
				return nil
			},
		},
		{
			Name:    "console",
			Aliases: []string{"c"},
			Usage:   "communicate with a running abot server",
			Action: func(c *cli.Context) error {
				if err := startConsole(c); err != nil {
					l := log.New("")
					l.SetFlags(0)
					l.Fatalf("could not start console\n%s", err)
				}
				return nil
			},
		},
		{
			Name:    "dbconsole",
			Aliases: []string{"dbc"},
			Usage:   "communicate with a running abot server",
			Action: func(c *cli.Context) error {
				cmd := exec.Command("/bin/sh", "-c", "psql "+os.Getenv("ABOT_DATABASE_URL"))
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				if err := cmd.Start(); err != nil {
					return err
				}
				return cmd.Wait()
			},
		},
	}
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func searchPlugins(query string) error {
	byt, err := searchItsAbot(query)
	if err != nil {
		return err
	}
	if err = outputPluginResults(os.Stdout, byt); err != nil {
		return err
	}
	return nil
}

func outputPluginResults(w io.Writer, byt []byte) error {
	var results []struct {
		ID            uint64
		Name          string
		Description   sql.NullString
		Path          string
		DownloadCount uint64
		Similarity    float64
	}
	if err := json.Unmarshal(byt, &results); err != nil {
		return err
	}
	writer := tabwriter.Writer{}
	writer.Init(w, 0, 8, 1, '\t', 0)
	_, err := writer.Write([]byte("NAME\tDESCRIPTION\tDOWNLOADS\n"))
	if err != nil {
		return err
	}
	for _, result := range results {
		d := result.Description
		if len(d.String) >= 30 {
			d.String = d.String[:27] + "..."
		}
		_, err = writer.Write([]byte(fmt.Sprintf("%s\t%s\t%d\n",
			result.Name, d.String, result.DownloadCount)))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func searchItsAbot(q string) ([]byte, error) {
	u := fmt.Sprintf("%s/api/plugins/search/%s", os.Getenv("ITSABOT_URL"),
		url.QueryEscape(q))
	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()
	return ioutil.ReadAll(res.Body)
}

func startConsole(c *cli.Context) error {
	args := c.Args()
	if len(args) >= 3 {
		return errors.New("usage: abot console {abotAddress} {userPhone}")
	}
	var addr, phone string
	switch len(args) {
	case 0:
		addr = "http://localhost:" + os.Getenv("PORT")
		phone = "+15555551234"
	case 1:
		addr = "http://localhost:" + os.Getenv("PORT")
		phone = args[0]
	case 2:
		addr = args[0]
		phone = args[1]
	}

	// Capture ^C interrupt to add a newline
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		for range sig {
			fmt.Printf("\n")
			os.Exit(0)
		}
	}()

	body := struct {
		CMD        string
		FlexID     string
		FlexIDType dt.FlexIDType
	}{
		FlexID:     phone,
		FlexIDType: 2,
	}
	scanner := bufio.NewScanner(os.Stdin)

	// Test connection
	req, err := http.NewRequest("GET", addr, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if err = resp.Body.Close(); err != nil {
		return err
	}
	fmt.Print("> ")

	// Handle each user input
	for scanner.Scan() {
		body.CMD = scanner.Text()
		byt, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", addr, bytes.NewBuffer(byt))
		if err != nil {
			return err
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err = resp.Body.Close(); err != nil {
			return err
		}
		fmt.Println(string(body))
		fmt.Print("> ")
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func installPlugins(l *log.Logger, errChan chan errMsg) {
	if err := core.LoadConf(); err != nil {
		errChan <- errMsg{msg: "", err: err}
		return
	}
	plugins := buildPluginFile(l)

	// Fetch all plugins
	if len(plugins.Dependencies) == 1 {
		l.Infof("Fetching 1 plugin...\n")
	} else {
		l.Infof("Fetching %d plugins...\n", len(plugins.Dependencies))
	}
	outC, err := exec.
		Command("/bin/sh", "-c", "go get ./...").
		CombinedOutput()
	if err == nil {
		syncDependencies(plugins, l, errChan)
		return
	}

	// Show errors only when it's not a private repo issue
	if !strings.Contains(string(outC), "fatal: could not read Username for") {
		errChan <- errMsg{msg: string(outC), err: err}
		return
	}

	// TODO enable versioning for private repos
	l.Infof("Fetching private repos...\n\n")
	l.Info("*** This will delete your local plugins to fetch remote copies.")
	_, err = fmt.Print("Continue? [n]: ")
	if err != nil {
		errChan <- errMsg{msg: "", err: err}
		return
	}
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		errChan <- errMsg{msg: "", err: errors.New("failed to read from stdin")}
		return
	}
	if text[0] != 'y' && text[0] != 'Y' {
		errChan <- errMsg{msg: "Canceled", err: nil}
		return
	}
	wg := &sync.WaitGroup{}
	for name, _ := range plugins.Dependencies {
		go clonePrivateRepo(name, errChan, wg)
	}
	wg.Wait()
	syncDependencies(plugins, l, errChan)
}

func clonePrivateRepo(name string, errChan chan errMsg, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	parts := strings.Split(name, "/")
	if len(parts) < 2 {
		errChan <- errMsg{msg: "", err: errors.New("invalid dependency path: too few parts")}
		return
	}

	// Ensure we don't delete a lower level directory
	p := filepath.Join(os.Getenv("GOPATH"), "src")
	tmp := filepath.Join(p, name)
	if len(tmp)+4 <= len(p) {
		errChan <- errMsg{msg: "", err: errors.New("invalid dependency path: too short")}
		return
	}
	if strings.Contains(tmp, "..") {
		errChan <- errMsg{msg: name, err: errors.New("invalid dependency path: contains '..'")}
		return
	}
	cmd := fmt.Sprintf("rm -rf %s", tmp)
	l.Debug("running:", cmd)
	outC, err := exec.
		Command("/bin/sh", "-c", cmd).
		CombinedOutput()
	if err != nil {
		tmp = fmt.Sprintf("failed to fetch %s\n%s", name, string(outC))
		errChan <- errMsg{msg: tmp, err: err}
		return
	}
	name = strings.Join(parts[1:], "/")
	cmd = fmt.Sprintf("git clone git@github.com:%s.git %s", name, tmp)
	l.Debug("running:", cmd)
	outC, err = exec.
		Command("/bin/sh", "-c", cmd).
		CombinedOutput()
	if err != nil {
		tmp = fmt.Sprintf("failed to fetch %s\n%s", name, string(outC))
		errChan <- errMsg{msg: tmp, err: err}
	}
}

func syncDependencies(plugins *core.PluginJSON, l *log.Logger,
	errChan chan errMsg) {

	// Sync each of them to get dependencies
	l.Debug("syncing dependencies")
	wg := &sync.WaitGroup{}
	rand.Seed(time.Now().UTC().UnixNano())
	for url, version := range plugins.Dependencies {
		wg.Add(1)
		go func(url, version string) {
			defer wg.Done()
			// Check out specific commit
			var outB []byte
			var errB error
			if version != "*" {
				l.Debug("checking out", url, "at", version)
				p := filepath.Join(os.Getenv("GOPATH"), "src",
					url)
				c := fmt.Sprintf("git -C %s checkout %s", p, version)
				outB, errB = exec.
					Command("/bin/sh", "-c", c).
					CombinedOutput()
				if errB != nil {
					l.Debug(string(outB))
					errChan <- errMsg{msg: "", err: errB}
					return
				}
			}

			// Anonymously increment the plugin's download count
			// at itsabot.org
			l.Debug("incrementing download count", url)
			p := struct{ Path string }{Path: url}
			outB, errB = json.Marshal(p)
			if errB != nil {
				errChan <- errMsg{msg: "failed to build itsabot.org JSON.", err: errB}
				return
			}
			var u string
			u = os.Getenv("ITSABOT_URL") + "/api/plugins.json"
			req, errB := http.NewRequest("PUT", u, bytes.NewBuffer(outB))
			if errB != nil {
				errChan <- errMsg{msg: "failed to build request to itsabot.org.", err: errB}
				return
			}
			client := &http.Client{Timeout: 10 * time.Second}
			resp, errB := client.Do(req)
			if errB != nil {
				errChan <- errMsg{msg: "failed to update itsabot.org.", err: errB}
				return
			}
			defer func() {
				if errB = resp.Body.Close(); errB != nil {
					errChan <- errMsg{msg: "", err: errB}
				}
			}()
			if resp.StatusCode != 200 {
				l.Infof("WARN: %d - %s\n", resp.StatusCode,
					resp.Status)
			}
		}(url, version)
	}
	wg.Wait()

	// Ensure dependencies are still there with the latest checked out
	// versions, and install the plugins
	l.Info("Installing plugins...")
	outC, err := exec.
		Command("/bin/sh", "-c", "go get ./...").
		CombinedOutput()
	if err != nil {
		errChan <- errMsg{msg: string(outC), err: err}
		return
	}
	embedPluginConfs(plugins, l)
	updateGlockfileAndInstall(l)
	errChan <- errMsg{msg: "Success!"}
}

func updatePlugins() {
	l := log.New("")
	l.SetFlags(0)
	l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")

	if err := core.LoadConf(); err != nil {
		l.Fatal(err)
	}
	plugins := buildPluginFile(l)

	l.Info("Updating plugins...")
	for path, version := range plugins.Dependencies {
		if version != "*" {
			continue
		}
		l.Infof("Updating %s...\n", path)
		outC, err := exec.
			Command("/bin/sh", "-c", "go get -u "+path).
			CombinedOutput()
		if err != nil {
			l.Info(string(outC))
			l.Fatal(err)
		}
	}
	embedPluginConfs(plugins, l)
	updateGlockfileAndInstall(l)
	l.Info("Success!")
}

func updateGlockfileAndInstall(l *log.Logger) {
	outC, err := exec.
		Command("/bin/sh", "-c", `pwd | sed "s|$GOPATH/src/||"`).
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}

	// Update plugin dependency versions in GLOCKFILE
	p := string(outC)
	outC, err = exec.
		Command("/bin/sh", "-c", "glock save "+p).
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}

	outC, err = exec.
		Command("/bin/sh", "-c", "go install").
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}
}

func embedPluginConfs(plugins *core.PluginJSON, l *log.Logger) {
	l.Debug("embedding plugin confs")

	// Open plugins.go file for writing
	fi, err := os.OpenFile("plugins.go", os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		l.Fatal(err)
	}
	defer func() {
		if err = fi.Close(); err != nil {
			l.Fatal(err)
		}
	}()

	p := os.Getenv("GOPATH")
	tokenizedPath := strings.Split(p, string(os.PathListSeparator))

	// Insert plugin.json text as comments
	s := "\n\n/*\n"
	for u := range plugins.Dependencies {
		s += u + "\n"
		l.Debug("reading file", p)
		p = filepath.Join(tokenizedPath[0], "src", u, "plugin.json")
		fi2, err2 := os.Open(p)
		if err2 != nil {
			l.Fatal(err2)
		}
		scn := bufio.NewScanner(fi2)
		var tmp string
		for scn.Scan() {
			line := scn.Text() + "\n"
			s += line
			tmp += line
		}
		if err2 = scn.Err(); err2 != nil {
			l.Fatal(err2)
		}
		if err2 = fi2.Close(); err2 != nil {
			l.Fatal(err2)
		}

		var plg struct{ Name string }
		if err2 = json.Unmarshal([]byte(tmp), &plg); err2 != nil {
			l.Fatal(err2)
		}

		// Fetch remote plugin IDs to be included in the plugin confs
		plg.Name = url.QueryEscape(plg.Name)
		ul := os.Getenv("ITSABOT_URL") + "/api/plugins/by_name/" + plg.Name
		req, err2 := http.NewRequest("GET", ul, nil)
		if err2 != nil {
			l.Fatal(err2)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err2 := client.Do(req)
		if err2 != nil {
			l.Fatal(err2)
		}
		var data struct{ ID uint64 }
		if err2 := json.NewDecoder(resp.Body).Decode(&data); err2 != nil {
			l.Fatal(err2)
		}
		id := strconv.FormatUint(data.ID, 10)

		// Remove closing characters to insert additional ID data
		s = s[:len(s)-3]
		s += ",\n\t\"ID\": " + id + "\n}\n"
	}
	s += "*/"
	_, err = fi.WriteString(s)
	if err != nil {
		l.Fatal(err)
	}
}

func buildPluginFile(l *log.Logger) *core.PluginJSON {
	// Create plugins.go file, truncate if exists
	fi, err := os.Create("plugins.go")
	if err != nil {
		l.Fatal(err)
	}
	defer func() {
		if err = fi.Close(); err != nil {
			l.Fatal(err)
		}
	}()

	s := "// This file is generated by `abot plugin install`. Do not edit.\n"
	s += "package main\n\nimport (\n"
	for url := range core.Conf().Dependencies {
		// Insert _ imports
		s += fmt.Sprintf("\t_ \"%s\"\n", url)
	}
	s += ")"
	_, err = fi.WriteString(s)
	if err != nil {
		l.Fatal(err)
	}

	return core.Conf()
}

func login() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Password: ")
	pass, err := terminal.ReadPassword(0)
	if err != nil {
		log.Fatal(err)
	}
	email = email[:len(email)-1]
	req := struct {
		Email    string
		Password string
	}{
		Email:    email,
		Password: string(pass),
	}
	fmt.Println()
	byt, err := json.Marshal(req)
	if err != nil {
		log.Fatal(err)
	}
	u := os.Getenv("ITSABOT_URL") + "/api/users/login.json"
	resp, err := http.Post(u, "application/json", bytes.NewBuffer(byt))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	var data struct {
		ID        uint64
		Email     string
		Scopes    []string
		AuthToken string
		IssuedAt  uint64
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode == 401 {
		log.Fatal(errors.New("invalid email/password combination"))
	}

	// Create abot.conf file, truncate if exists
	fi, err := os.Create(filepath.Join(os.Getenv("HOME"), ".abot.conf"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = fi.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Insert auth data
	s := fmt.Sprintf("%d\n%s\n%s\n%d", data.ID, data.Email, data.AuthToken,
		data.IssuedAt)
	_, err = fi.WriteString(s)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Success!")
}

func publishPlugin(c *cli.Context) error {
	p := filepath.Join(os.Getenv("HOME"), ".abot.conf")
	fi, err := os.Open(p)
	if err != nil {
		if err.Error() == fmt.Sprintf("open %s: no such file or directory", p) {
			login()
			publishPlugin(c)
			return nil
		}
		log.Fatal(err)
	}
	defer func() {
		if err = fi.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Prepare request
	if len(c.Args().First()) == 0 {
		log.Fatal("missing plugin's `go get` path")
	}
	reqData := struct {
		Path   string
		Secret string
	}{
		Path:   c.Args().First(),
		Secret: core.RandSeq(24),
	}
	byt, err := json.Marshal(reqData)
	if err != nil {
		log.Fatal(err)
	}
	u := os.Getenv("ITSABOT_URL") + "/api/plugins.json"
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(byt))
	if err != nil {
		log.Fatal(err)
	}

	// Populate req with login credentials from ~/.abot.conf
	scn := bufio.NewScanner(fi)
	var lineNum int
	for scn.Scan() {
		line := scn.Text()
		cookie := &http.Cookie{}
		switch lineNum {
		case 0:
			cookie.Name = "iaID"
		case 1:
			cookie.Name = "iaEmail"
		case 2:
			req.Header.Set("Authorization", "Bearer "+line)
		case 3:
			cookie.Name = "iaIssuedAt"
		default:
			log.Fatal("unknown line in abot.conf")
		}
		if lineNum != 2 {
			cookie.Value = url.QueryEscape(line)
			req.AddCookie(cookie)
		}
		lineNum++
	}
	if err = scn.Err(); err != nil {
		log.Fatal(err)
	}
	cookie := &http.Cookie{}
	cookie.Name = "iaScopes"
	req.AddCookie(cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	if resp.StatusCode == 401 {
		login()
		publishPlugin(c)
	} else if resp.StatusCode != 202 {
		log.Info("something went wrong. status code", resp.StatusCode)
		var msg string
		if err = json.NewDecoder(resp.Body).Decode(&msg); err != nil {
			log.Fatal(err)
		}
		log.Fatal(msg)
	}

	// Make a websocket request to get updates about the publishing process
	uri, err := url.Parse(os.Getenv("ITSABOT_URL"))
	if err != nil {
		log.Fatal(err)
	}
	ws, err := websocket.Dial("ws://"+uri.Host+"/api/ws", "",
		os.Getenv("ABOT_URL"))
	if err != nil {
		log.Fatal(err)
	}
	if err = websocket.Message.Send(ws, reqData.Secret); err != nil {
		log.Fatal(err)
	}
	var msg socket.Msg
	l := log.New("")
	l.SetFlags(0)
	l.Info("> Establishing connection with server...")
	var established bool
	var lastMsg string
	for {
		websocket.JSON.Receive(ws, &msg)
		if !established {
			l.Info("OK")
			established = true
		}
		if msg.Content == lastMsg {
			log.Info("server hung up. please try again")
			if err = ws.Close(); err != nil {
				log.Info("failed to close socket.", err)
			}
			return nil
		}
		lastMsg = msg.Content
		if msg.Type == socket.MsgTypeFinishedSuccess ||
			msg.Type == socket.MsgTypeFinishedFailed {
			if err = ws.Close(); err != nil {
				log.Info("failed to close socket.", err)
			}
			return nil
		}
		if len(msg.Content) < 2 {
			l.Info(msg.Content)
			continue
		}
		tmp := msg.Content[0:2]
		if tmp == "> " || tmp == "==" {
			l.Info("")
		}
		l.Info(msg.Content)
	}
}

func generatePlugin(l *log.Logger, name string) error {
	// Log in to get the maintainer email
	if os.Getenv("ABOT_ENV") != "test" {
		p := filepath.Join(os.Getenv("HOME"), ".abot.conf")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			login()
		}
	}

	// Ensure the name and path are unique globally
	var words []string
	var lastIdx int
	name = strings.Replace(name, " ", "_", -1)
	dirName := name
	for i, letter := range name {
		if i == 0 {
			continue
		}
		if unicode.IsUpper(letter) {
			words = append(words, name[lastIdx:i])
			lastIdx = i
		}
	}
	words = append(words, name[lastIdx:])
	dirName = strings.Join(words, "_")
	dirName = strings.ToLower(dirName)
	name = strings.ToLower(name)

	// Create the directory
	if err := os.Mkdir(dirName, 0744); err != nil {
		return err
	}

	// Generate a plugin.json file
	if err := buildPluginJSON(dirName, name); err != nil {
		log.Info("failed to create plugin.json")
		return err
	}

	// Generate name.go and name_test.go files with starter keywords and
	// state machines
	if err := buildPluginScaffoldFile(dirName, name); err != nil {
		log.Info("failed to create plugin scaffold file")
		return err
	}
	return nil
}

func buildPluginJSON(dirName, name string) error {
	var maintainer string
	if os.Getenv("ABOT_ENV") == "test" {
		maintainer = "test@example.com"
	} else {
		fi, err := os.Open(filepath.Join(os.Getenv("HOME"), ".abot.conf"))
		if err != nil {
			return err
		}
		defer func() {
			if err = fi.Close(); err != nil {
				log.Info("failed to close plugin.json file.", err)
			}
		}()
		scn := bufio.NewScanner(fi)
		var lineNum int
		for scn.Scan() {
			if lineNum < 1 {
				lineNum++
				continue
			}
			maintainer = scn.Text()
			break
		}
		if scn.Err() != nil {
			return err
		}
	}
	b := []byte(`{
	"Name": "` + name + `",
	"Maintainer": "` + maintainer + `",
	"Usage": ["Show me a demo"],
	"Tests": [
		{"Show me a demo": ["demo"]}
	]
}`)
	return ioutil.WriteFile(filepath.Join(dirName, "plugin.json"), b, 0744)
}

func buildPluginScaffoldFile(dirName, name string) error {
	fi, err := os.Create(filepath.Join(dirName, dirName+".go"))
	if err != nil {
		return err
	}
	defer func() {
		err = fi.Close()
		if err != nil {
			log.Info("failed to close plugin.json.", err)
		}
	}()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, name)
	_, err = fi.WriteString(pluginScaffoldFile(dir, name))
	if err != nil {
		return err
	}
	return nil
}

// newAbot creates a new directory for a new Abot project. It's similar to
// `rails new`.
func newAbot(l *log.Logger, name, dbconnstr string) error {
	// Create a new directory for the project
	if err := os.Mkdir(name, 0777); err != nil {
		return err
	}
	if err := os.Chdir(name); err != nil {
		return err
	}

	// Generate abot.env
	fi, err := os.Create("abot.env")
	if err != nil {
		return err
	}
	defer func() {
		if err = fi.Close(); err != nil {
			l.Info("failed to close abot.env.", err)
		}
	}()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	_, err = fi.WriteString(serverAbotEnv(name, dir))
	if err != nil {
		return err
	}

	// Copy and modify base files
	p := filepath.Join(os.Getenv("ABOT_PATH"), "base", "plugins.json")
	if err = core.CopyFileContents(p, "plugins.json"); err != nil {
		return err
	}
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "server.go.x")
	if err = core.CopyFileContents(p, "server.go"); err != nil {
		return err
	}
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", ".gitignore")
	if err = core.CopyFileContents(p, ".gitignore"); err != nil {
		return err
	}
	fi2, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer func() {
		if err = fi2.Close(); err != nil {
			l.Info("failed to close .gitignore.", err)
		}
	}()
	_, err = fi2.WriteString(name)
	if err != nil {
		l.Fatal("failed to write to .gitignore.", err)
	}

	// Walk the base/assets dir, copying all files
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "assets")
	if err = filepath.Walk(p, recursiveCopy); err != nil {
		return err
	}
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "cmd")
	if err = filepath.Walk(p, recursiveCopy); err != nil {
		return err
	}
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "data")
	if err = filepath.Walk(p, recursiveCopy); err != nil {
		return err
	}
	p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "db")
	if err = filepath.Walk(p, recursiveCopy); err != nil {
		return err
	}

	// Run cmd/dbsetup.sh
	cmd := exec.Command("/bin/sh", "-c", "cmd/dbsetup.sh "+dbconnstr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		l.Info("Fix the errors above, then re-run cmd/dbsetup.sh")
		return err
	}

	// TODO analytics on a new Abot project
	return nil
}

func recursiveCopy(pth string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	parts := strings.Split(pth, string(os.PathSeparator))
	pth = ""
	for i := len(parts) - 1; i > 0; i-- {
		if parts[i] == "base" {
			break
		}
		pth = filepath.Join(parts[i], pth)
	}

	// Handle directories
	if info.IsDir() {
		if err = os.Mkdir(pth, 0777); err != nil {
			return err
		}
		return nil
	}

	// Handle files
	p := filepath.Join(os.Getenv("ABOT_PATH"), "base", pth)
	core.CopyFileContents(p, pth)
	return nil
}

func testPlugin() (int, error) {
	if err := core.LoadPluginsGo(); err != nil {
		return 0, err
	}
	r := plugin.TestPrepare()
	plugin.TestCleanup()
	var count int
	for _, plg := range core.PluginsGo {
		log.Info("loading", plg.Name)
		for _, test := range plg.Tests {
			log.Info("running", test)
			count++
			for in, exps := range test {
				err := plugin.TestReq(r, in, exps)
				if err != nil {
					return count, err
				}
			}
		}
	}
	plugin.TestCleanup()
	return count, nil
}

type errMsg struct {
	msg string
	err error
}
