package main

import (
	"flag"
	"log"
	"os"

	"github.com/hexops/cmder"
)

// commands contains all registered subcommands.
var commands cmder.Commander

var usageText = `wrench (zig mirror)

Usage:

	wrench <command> [arguments]

The commands are:

	service    manage the wrench service (also 'wrench svc')
	version    print the wrench version

Use "wrench <command> -h" for more information about a command.
`

func main() {
	// Configure logging.
	log.SetFlags(0)
	log.SetPrefix("")

	commands.Run(flag.CommandLine, "wrench", usageText, os.Args[1:])
}
