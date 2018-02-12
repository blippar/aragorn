package main

import (
	"flag"

	"github.com/blippar/aragorn/server"
)

const execHelp = `Execute the test suites in the directories`

type execCommand struct {
	failfast bool
}

func (*execCommand) Name() string { return "exec" }
func (*execCommand) Args() string {
	return "[file ...]"
}
func (*execCommand) ShortHelp() string { return execHelp }
func (*execCommand) LongHelp() string  { return execHelp }
func (*execCommand) Hidden() bool      { return false }

func (cmd *execCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.failfast, "failfast", false, "stop after first test failure")
}

func (cmd *execCommand) Run(args []string) error {
	return server.Exec(args, cmd.failfast)
}
