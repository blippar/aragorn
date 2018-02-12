package main

import (
	"flag"

	"github.com/blippar/aragorn/server"
)

const listHelp = `List the test suites in the directories`

type listCommand struct{}

func (*listCommand) Name() string { return "list" }
func (*listCommand) Args() string {
	return "[file ...]"
}
func (*listCommand) ShortHelp() string { return listHelp }
func (*listCommand) LongHelp() string  { return listHelp }
func (*listCommand) Hidden() bool      { return false }

func (*listCommand) Register(fs *flag.FlagSet) {}

func (*listCommand) Run(args []string) error {
	return server.List(args)
}
