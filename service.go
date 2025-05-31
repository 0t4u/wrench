package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/0t4u/wrench/internal/wrench"
	"github.com/hexops/cmder"
)

// serviceCommands contains all registered 'wrench service' subcommands.
var serviceCommands cmder.Commander

var (
	serviceFlagSet    = flag.NewFlagSet("service", flag.ExitOnError)
	serviceConfigFile = serviceFlagSet.String("config", defaultConfigFilePath(), "Path to TOML configuration file (see config.go)")
)

func init() {
	const usage = `wrench service: manage wrench as a service

Usage:

	wrench service [-config=config.toml] <command> [arguments]

The commands are:

	run          run the server now
	logs         view service logs (stderr, stdout, and system service runner logs)

Use "wrench service <command> -h" for more information about a command.
`

	usageFunc := func() {
		fmt.Printf("%s", usage)
	}
	serviceFlagSet.Usage = usageFunc

	// Handles calls to our subcommand.
	handler := func(args []string) error {
		_ = serviceFlagSet.Parse(args)
		serviceCommands.Run(serviceFlagSet, "wrench service", usage, args)
		return nil
	}

	// Register the command.
	commands = append(commands, &cmder.Command{
		FlagSet:   serviceFlagSet,
		Aliases:   []string{"svc"},
		Handler:   handler,
		UsageFunc: usageFunc,
	})
}

func defaultConfigFilePath() string {
	if _, err := os.Stat("config.toml"); err == nil {
		return "config.toml"
	}
	u, err := user.Current()
	if err == nil {
		return filepath.Join(u.HomeDir, "wrench/config.toml")
	}
	return "config.toml"
}

func newBot() *wrench.Bot {
	bot := &wrench.Bot{
		ConfigFile: *serviceConfigFile,
	}

	return bot
}

type ServiceConfig struct {
	ConfigFile string
	Executable string
}
