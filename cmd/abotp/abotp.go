// abotp fetches and manages Abot packages.
//
// Eventually support will be added for versioning, package uploads, searching,
// etc.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type packageJSON struct {
	Dependencies map[string]string
}

func main() {
	log.SetFlags(0)
	// Delete all packages in the /packages dir, packages.lock
	if err := os.RemoveAll("./packages"); err != nil {
		log.Fatalln(err)
	}
	err := os.Remove("./packages.lock")
	if err != nil && err.Error() !=
		"remove ./packages.lock: no such file or directory" {
		log.Fatalln(err)
	}
	// Read packages.json, unmarshal into struct
	contents, err := ioutil.ReadFile("./packages.json")
	if err != nil {
		log.Fatalln(err)
	}
	var packages packageJSON
	if err = json.Unmarshal(contents, &packages); err != nil {
		log.Fatalln(err)
	}
	// Remake the /packages dir
	if err = os.Mkdir("./packages", 0775); err != nil {
		log.Fatalln(err)
	}
	// Fetch packages
	log.Println("Fetching", len(packages.Dependencies), "packages...")
	var wg sync.WaitGroup
	wg.Add(len(packages.Dependencies))
	rand.Seed(time.Now().UTC().UnixNano())
	for url, _ := range packages.Dependencies {
		go func(url string) {
			// Download source as a zip
			resp, err := http.Get("https://" + url + "/archive/master.zip")
			if err != nil {
				log.Fatalln(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				e := fmt.Sprintf("err fetching package %s: %d", url,
					resp.StatusCode)
				log.Fatalln(errors.New(e))
			}
			fiName := "tmp_" + randSeq(8) + ".zip"
			fpZip := filepath.Join("./packages", fiName)
			out, err := os.Create(fpZip)
			if err != nil {
				log.Fatalln(err)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				_ = out.Close()
				log.Fatalln(err)
			}
			// Unzip source to directory
			if err = unzip(fpZip, "./packages"); err != nil {
				_ = out.Close()
				log.Fatalln(err)
			}
			// Close zip file
			if err = out.Close(); err != nil {
				log.Fatalln(err)
			}
			// Delete zip file
			if err = os.Remove(fpZip); err != nil {
				log.Fatalln(err)
			}

			// Anonymously increment the package's download count
			// at itsabot.org
			p := struct {
				Path string
			}{Path: url}
			byt, err := json.Marshal(p)
			if err != nil {
				log.Println("WARN:", err)
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
				log.Println("WARN:", err)
				wg.Done()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				log.Println("WARN:", resp.StatusCode, "-",
					resp.Status)
			}
			wg.Done()
		}(url)
	}
	wg.Wait()
	log.Println("Success!")
	// TODO Create a packages.lock file including versioning if present.
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
