package main

import (
	"flag"

	"github.com/blippar/aragorn/server"
)

const execHelp = `Execute the test suites in the directories`

type execCommand struct{}

func (*execCommand) Name() string { return "exec" }
func (*execCommand) Args() string {
	return "[file ...]"
}
func (*execCommand) ShortHelp() string { return execHelp }
func (*execCommand) LongHelp() string  { return execHelp }
func (*execCommand) Hidden() bool      { return false }

func (*execCommand) Register(fs *flag.FlagSet) {}

func (*execCommand) Run(args []string) error {
	return server.New(args).Exec()
}
