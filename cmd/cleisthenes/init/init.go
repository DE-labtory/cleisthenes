package init

import (
	"os"

	"github.com/DE-labtory/cleisthenes/config"
	"github.com/kyokomi/emoji"
	"github.com/urfave/cli"
)

func Cmd() cli.Command {
	return cli.Command{
		Name:      "init",
		Usage:     "Initialize cleisthenes configuration",
		UsageText: "cleisthenes init",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config",
				Usage: "Load configuration file from FILE_PATH",
			},
		},
		Action: func(c *cli.Context) error {
			return initCleisthenes(c.Args().First())
		},
	}
}

func initCleisthenes(configPath string) error {
	if err := config.Init(configPath); err != nil {
		emoji.Println(":broken_heart: initialize failed with error: %s", err)
		return err
	}
	emoji.Printf(":beer: successfully initialized at %s\n", os.Getenv("HOME")+"/.cleisthenes")
	return nil
}
