package main

import (
	"log"
	"os"
	"time"

	initCmd "github.com/DE-labtory/cleisthenes/cmd/cleisthenes/init"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "cleisthenes"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Usage = "HoneyBadgerBFT, the first practical asynchronous BFT protocol without timing assuption"
	app.UsageText = "cleisthenes [options] command [command options] [arguments...]"
	app.Authors = []cli.Author{
		{
			Name:  "DE-labtory",
			Email: "de.labtory@gmail.com",
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "set debug mode",
		},
	}

	app.Commands = []cli.Command{}
	app.Commands = append(app.Commands, initCmd.Cmd())

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
