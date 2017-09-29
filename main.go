package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/guywithnose/jira-standup/command"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = command.Name
	app.Version = fmt.Sprintf("%s-%s", command.Version, runtime.Version())
	app.Author = "Robert Bittle"
	app.Email = "guywithnose@gmail.com"
	app.Usage = "jira-standup"
	app.Action = command.CmdMain

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "%s: '%s' is not a %s command. See '%s --help'.", c.App.Name, command, c.App.Name, c.App.Name)
		os.Exit(2)
	}
	app.ErrWriter = os.Stderr

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "url",
			Usage:  "The jira url",
			EnvVar: "JIRA_URL",
		},
		cli.StringFlag{
			Name:   "username",
			Usage:  "The jira username for auth",
			EnvVar: "JIRA_USERNAME",
		},
		cli.StringFlag{
			Name:   "password",
			Usage:  "The jira password for auth",
			EnvVar: "JIRA_PASSWORD",
		},
		cli.StringFlag{
			Name:  "date",
			Usage: "The date to check",
		},
		cli.IntFlag{
			Name:  "relativeDate",
			Usage: "Check date that was this many days ago",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
