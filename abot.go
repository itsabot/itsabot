package main

import (
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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/codegangsta/cli"
	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

var conf *core.PluginJSON

// PathError is thrown when GOPATH cannot be located
var PathError = errors.New("GOPATH env variable not set")

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetDebug(os.Getenv("ABOT_DEBUG") == "true")

	var err error
	conf, err = core.LoadConf()
	if err != nil {
		log.Fatal(err)
	}
	err = core.LoadEnvVars()
	if err != nil {
		log.Fatal(err)
	}

	app := cli.NewApp()
	app.Name = conf.Name
	app.Usage = conf.Description
	app.Version = conf.Version
	app.Commands = []cli.Command{
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "run server",
			Action: func(c *cli.Context) {
				var err error
				if err = startServer(); err != nil {
					l := log.New("")
					l.SetFlags(0)
					l.Fatalf("could not start server\n%s", err)
				}
			},
		},
		{
			Name:    "plugin",
			Aliases: []string{"p"},
			Usage:   "manage and install plugins from plugins.json",
			Subcommands: []cli.Command{
				{
					Name:  "install",
					Usage: "download and install plugins listed in plugins.json",
					Action: func(c *cli.Context) {
						installPlugins()
					},
				},
				{
					Name:    "search",
					Aliases: []string{"s"},
					Usage:   "search plugins indexed on itsabot.org",
					Action: func(c *cli.Context) {
						l := log.New("")
						l.SetFlags(0)
						args := c.Args()
						if len(args) == 0 || len(args) > 2 {
							l.Fatal(errors.New(`usage: abot plugin search "{term}"`))
						}
						if err := searchPlugins(args.First()); err != nil {
							l.Fatalf("could not start console\n%s", err)
						}
					},
				},
				{
					Name:    "update",
					Aliases: []string{"u", "upgrade"},
					Usage:   "update and install plugins listed in plugins.json",
					Action: func(c *cli.Context) {
						updatePlugins()
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
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// startServer initializes any clients that are needed, sets up routes, and
// boots plugins.
func startServer() error {
	hr, err := core.NewServer()
	if err != nil {
		return err
	}
	log.Info("started", conf.Name)
	if err = http.ListenAndServe(":"+os.Getenv("PORT"), hr); err != nil {
		return err
	}
	return nil
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
		Username      string
		Description   string
		Path          string
		Readme        string
		DownloadCount uint64
		Similarity    float64
	}
	if err := json.Unmarshal(byt, &results); err != nil {
		return err
	}
	writer := tabwriter.Writer{}
	writer.Init(w, 0, 8, 1, '\t', 0)
	_, err := writer.Write([]byte("NAME\tDESCRIPTION\tUSERNAME\tDOWNLOADS\n"))
	if err != nil {
		return err
	}
	for _, result := range results {
		d := result.Description
		if len(result.Description) >= 30 {
			d = d[:27] + "..."
		}
		_, err = writer.Write([]byte(fmt.Sprintf("%s\t%s\t%s\t%d\n",
			result.Name, d, result.Username, result.DownloadCount)))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func searchItsAbot(q string) ([]byte, error) {
	u := fmt.Sprintf("https://www.itsabot.org/api/search.json?q=%s",
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
	if len(args) == 0 || len(args) >= 3 {
		return errors.New("usage: abot console {abotAddress} {userPhone}")
	}
	var addr, phone string
	if len(args) == 1 {
		addr = "http://localhost:" + os.Getenv("PORT")
		phone = args[0]
	} else if len(args) == 2 {
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

func installPlugins() {
	l := log.New("")
	l.SetFlags(0)
	l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")

	plugins := buildPluginFile(l)

	// Fetch all plugins
	l.Info("Fetching", len(plugins.Dependencies), "plugins...")
	outC, err := exec.
		Command("/bin/sh", "-c", "go get ./...").
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}

	// Sync each of them to get dependencies
	var wg sync.WaitGroup
	wg.Add(len(plugins.Dependencies))
	rand.Seed(time.Now().UTC().UnixNano())
	for url, version := range plugins.Dependencies {
		go func(url, version string) {
			// Check out specific commit
			var outB []byte
			if version != "*" {
				l.Debug("checking out", url, "at", version)
				path := os.Getenv("GOPATH")
				if path == "" {
					l.Fatal(PathError)
				}
				tokenizedPath := strings.Split(path, string(os.PathListSeparator))
				p := filepath.Join(tokenizedPath[0], "src", url)
				c := fmt.Sprintf("git -C %s checkout %s", p, version)
				outB, err = exec.
					Command("/bin/sh", "-c", c).
					CombinedOutput()
				if err != nil {
					l.Debug(string(outB))
					l.Fatal(err)
				}
			}

			// Anonymously increment the plugin's download count
			// at itsabot.org
			l.Debug("incrementing download count", url)
			p := struct{ URL string }{URL: url}
			outB, err = json.Marshal(p)
			if err != nil {
				l.Info("failed to build itsabot.org JSON.", err)
				wg.Done()
				return
			}
			var u string
			if len(os.Getenv("ITSABOT_URL")) > 0 {
				u = os.Getenv("ITSABOT_URL") + "/api/plugins.json"
			} else {
				u = "https://www.itsabot.org/api/plugins.json"
			}
			resp, errB := http.Post(u, "application/json",
				bytes.NewBuffer(outB))
			if errB != nil {
				l.Info("failed to update itsabot.org.", errB)
				wg.Done()
				return
			}
			defer func() {
				if errB = resp.Body.Close(); errB != nil {
					l.Fatal(errB)
				}
			}()
			if resp.StatusCode != 200 {
				l.Infof("WARN: %d - %s\n", resp.StatusCode,
					resp.Status)
			}
			wg.Done()
		}(url, version)
	}
	wg.Wait()

	// Ensure dependencies are still there with the latest checked out
	// versions, and install the plugins
	l.Info("Installing plugins...")
	outC, err = exec.
		Command("/bin/sh", "-c", "go get ./...").
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}

	updateGlockfileAndInstall(l)
	l.Info("Success!")
}

func updatePlugins() {
	l := log.New("")
	l.SetFlags(0)
	l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")

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

	updateGlockfileAndInstall(l)
	l.Info("Success!")
}

func updateGlockfileAndInstall(l *log.Logger) {
	path := os.Getenv("GOPATH")
	if path == "" {
		l.Fatal(PathError)
	}
	tokenizedPath := strings.Split(path, string(os.PathListSeparator))
	outC, err := exec.
		Command("/bin/sh", "-c", `pwd | sed "s|`+tokenizedPath[0]+`/src/||"`).
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

func buildPluginFile(l *log.Logger) *core.PluginJSON {
	plugins, err := core.LoadConf()
	if err != nil {
		l.Fatal(err)
	}

	// Create plugin.go file, truncate if exists
	fi, err := os.Create("plugins.go")
	if err != nil {
		l.Fatal(err)
	}
	defer func() {
		if err = fi.Close(); err != nil {
			l.Fatal(err)
		}
	}()

	// Insert _ imports
	s := "// This file is generated by `abot plugin install`. Do not edit.\n\n"
	s += "package main\n\nimport (\n"
	for url := range plugins.Dependencies {
		s += fmt.Sprintf("\t_ \"%s\"\n", url)
	}
	s += ")"
	_, err = fi.WriteString(s)
	if err != nil {
		l.Fatal(err)
	}

	return plugins
}
