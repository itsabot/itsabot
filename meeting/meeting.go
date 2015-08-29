package main

import (
	"log"
	"os"
	"time"

	"github.com/codegangsta/cli"
	_ "github.com/lib/pq"
)

func main() {
	app := cli.NewApp()
	app.Name = "meeting"
	app.Usage = "enable Ava to schedule meetings"
	app.Action = func(c *cli.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Println("missing argument! must supply data")
				log.Println(r)
			}
		}()
		log.Println("args", c.Args())
		ScheduleMeeting(&c.Args()[0])
	}
	app.Run(os.Args)

	ap := AvaPackage{}
}

func ScheduleMeeting(content *string) (string, error) {
	receivers, err := getReceivers(content)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println("receivers", receivers)

	times, recurring, err := getTimes(content)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println("times", times)
	log.Println("recurring", recurring)

	availableTimes, err := getAvailability(times)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println("avail times", availableTimes)

	return "", err
}

func getReceivers(content *string) ([]string, error) {
	// MITIE should replace this
	/*
		firstLine := strings.SplitN(*content, "\n", 1)
		nameCandidates := strings.FieldsFunc(firstLine, splitOnCommaOrSpace)
		var n string
		var names []string
		for _, name := range nameCandidates {
			err := db.Get(&n, "SELECT name FROM names", nil)
			if err != sql.ErrNoRows {
				if err != nil {
					return names, err
				}
				// Name found in database
				names = append(names, n)
			}
		}
		return names, nil
	*/
	return []string{}, nil
}

func getTimes(content *string) ([]time.Time, bool, error) {
	return []time.Time{}, false, nil
}

func getAvailability([]time.Time) ([]time.Time, error) {
	return []time.Time{}, nil
}

func splitOnCommaOrSpace(c rune) bool {
	return c == ',' || c == ' '
}
