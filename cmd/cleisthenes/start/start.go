package start

import "github.com/urfave/cli"

func Cmd() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "cleisthenes start",
		Action: func(c *cli.Context) error {
			return startCleisthenes()
		},
	}
}

func startCleisthenes() error {
	return nil
}
