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
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/codegangsta/cli"
	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/log"
	_ "github.com/lib/pq"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetDebug(true)
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
						if err := installPlugins(); err != nil {
							l := log.New("")
							l.SetFlags(0)
							l.Fatalf("could not install plugins\n%s", err)
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
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// startServer initializes any clients that are needed, sets up routes, and
// boots plugins.
func startServer() error {
	e, _, err := core.NewServer()
	if err != nil {
		return err
	}
	e.Run(":" + os.Getenv("PORT"))
	return nil
}

func startConsole(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 || len(args) >= 3 {
		return errors.New("usage: abot console abot-address user-phone")
	}
	var addr, phone string
	if len(args) == 1 {
		addr = "localhost:" + os.Getenv("PORT")
		phone = args[0]
	} else if len(args) == 2 {
		addr = args[0]
		phone = args[1]
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
	base := "http://" + addr + "?flexidtype=2&flexid=" + url.QueryEscape(phone) + "&cmd="
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

func installPlugins() error {
	l := log.New("")
	l.SetFlags(0)
	// Delete all plugins in the /plugins and /public directories
	err := os.RemoveAll("./plugins")
	if err != nil && err.Error() !=
		"remove ./plugins: no such file or directory" {
		l.Fatal(err)
	}
	err = os.RemoveAll("./public")
	if err != nil && err.Error() !=
		"remove ./public: no such file or directory" {
		l.Fatal(err)
	}
	// Read plugins.json, unmarshal into struct
	contents, err := ioutil.ReadFile("./plugins.json")
	if err != nil {
		l.Fatal(err)
	}
	var plugins pluginJSON
	if err = json.Unmarshal(contents, &plugins); err != nil {
		l.Fatal(err)
	}
	// Remake the /plugins dir for plugin Go code
	if err = os.Mkdir("./plugins", 0775); err != nil {
		l.Fatal(err)
	}
	// Remake the /public dir for assets
	if err = os.Mkdir("./public", 0775); err != nil {
		l.Fatal(err)
	}
	// Fetch plugins
	l.Info("Fetching", len(plugins.Dependencies), "plugins...")
	var wg sync.WaitGroup
	wg.Add(len(plugins.Dependencies))
	rand.Seed(time.Now().UTC().UnixNano())
	for url, version := range plugins.Dependencies {
		go func(url, version string) {
			// Download source as a zip
			var resp *http.Response
			resp, err = http.Get("https://" + url + "/archive/master.zip")
			if err != nil {
				l.Fatal(err)
			}
			defer func() {
				if err = resp.Body.Close(); err != nil {
					l.Fatal(err)
				}
			}()
			if resp.StatusCode != 200 {
				e := fmt.Sprintf("err fetching plugin %s: %d", url,
					resp.StatusCode)
				l.Fatal(errors.New(e))
			}
			fiName := "tmp_" + core.RandSeq(8) + ".zip"
			fpZip := filepath.Join("./plugins", fiName)
			var out *os.File
			out, err = os.Create(fpZip)
			if err != nil {
				l.Fatal(err)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				_ = out.Close()
				l.Fatal(err)
			}

			// Unzip source to directory
			if err = unzip(fpZip, "./plugins"); err != nil {
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

			// Sync to get dependencies
			var outC []byte
			outC, err = exec.
				Command("/bin/sh", "-c", "glock sync $(pwd | sed 's/^.*src\\///')").
				CombinedOutput()
			if err != nil {
				l.Debug(string(outC))
				l.Fatal(err)
			}

			// Anonymously increment the plugin's download count
			// at itsabot.org
			p := struct {
				Path string
			}{Path: url}
			outC, err = json.Marshal(p)
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
			resp, err = http.Post(u, "application/json",
				bytes.NewBuffer(outC))
			if err != nil {
				l.Info("failed to update itsabot.org.", err)
				wg.Done()
				return
			}
			defer func() {
				if err = resp.Body.Close(); err != nil {
					l.Fatal(err)
				}
			}()
			if resp.StatusCode != 200 {
				l.Info("WARN: %d - %s\n", resp.StatusCode,
					resp.Status)
			}
			wg.Done()
		}(url, version)
	}
	wg.Wait()
	l.Info("Fetching dependencies...")
	outC, err := exec.
		Command("/bin/sh", "-c", "go get ./...").
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}
	l.Info("Installing plugins...")
	outC, err = exec.
		Command("/bin/sh", "-c", "go install ./...").
		CombinedOutput()
	if err != nil {
		l.Info(string(outC))
		l.Fatal(err)
	}
	l.Info("Success!")
	return nil
}

type pluginJSON struct {
	Dependencies map[string]string
}

// From https://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = r.Close(); err != nil {
			panic(err)
		}
	}()
	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}
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
		if err = rc.Close(); err != nil {
			panic(err)
		}
	}()
	path := filepath.Join(dest, f.Name)
	if f.FileInfo().IsDir() {
		if err = os.MkdirAll(path, f.Mode()); err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err = f.Close(); err != nil {
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
