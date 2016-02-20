package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: abotc abot-address user-phone")
		os.Exit(0)
	}
	base := "http://" + os.Args[1] + "?flexidtype=2&flexid=" + url.QueryEscape(os.Args[2]) + "&cmd="

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		cmd := scanner.Text()
		req, err := http.NewRequest("POST", base+url.QueryEscape(cmd), nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		resp.Body.Close()

		fmt.Println(string(body))
		fmt.Print("> ")
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("scanner error:", err)
	}
}
