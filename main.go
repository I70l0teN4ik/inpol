package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/I70l0teN4ik/inpol/pkg"
	"github.com/joho/godotenv"
)

const (
	host = "inpol.mazowieckie.pl"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file:" + err.Error())
	}

	conf := &pkg.Config{
		Host:        host,
		Queue:       os.Getenv("QUEUE"),
		Case:        os.Getenv("CASE"),
		Name:        os.Getenv("NAME"),
		LastName:    os.Getenv("LAST_NAME"),
		DateOfBirth: os.Getenv("DATE_OF_BIRTH"),
		MFA:         os.Getenv("MFA"),
	}
	token := os.Getenv("JWT")

	var cmd string

	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	if token == "" {
		os.Stdin.Read(make([]byte, 1))
	}

	client, err := pkg.NewClient(*conf, token)
	if err != nil {
		fmt.Println(err)
		return
	}

	svc := pkg.NewReserver(client)

	if cmd == "dates" || cmd == "d" {
		err = svc.CheckDates()
	} else if cmd == "watch" || cmd == "w" {
		sleep := 5
		if len(os.Args) > 2 {
			sleep, _ = strconv.Atoi(os.Args[2])
		}
		err = svc.WatchDates(sleep)
	} else if cmd == "mfa" {
		fmt.Println(svc.GetMFA())
	} else if cmd == "async" || cmd == "a" {
		limit := 10
		if len(os.Args) > 2 {
			limit, _ = strconv.Atoi(os.Args[2])
		}
		err = svc.AsyncReserve(limit)
	} else {
		err = svc.ReserveResidence()
	}

	if err != nil {
		fmt.Println(err)
	}

	return
}
