package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/blippar/aragorn/server"
)

const listShortHelp = `List the test suites`
const listLongHelp = `List the test suites` + fileHelp

type listCommand struct {
	filter string
}

func (*listCommand) Name() string { return "list" }
func (*listCommand) Args() string {
	return "[file ...]"
}
func (*listCommand) ShortHelp() string { return listShortHelp }
func (*listCommand) LongHelp() string  { return listLongHelp }
func (*listCommand) Hidden() bool      { return false }

func (cmd *listCommand) Register(fs *flag.FlagSet) {
	fs.StringVar(&cmd.filter, "filter", "", "List only the tests that match the regular expression")
}

func (cmd *listCommand) Run(args []string) error {
	suites, err := getSuitesFromArgs(args, server.Filter(cmd.filter))
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Path\tSuite\tType\tTest\tDescription")
	for _, s := range suites {
		tests := s.Tests()
		for _, t := range tests {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", s.Path(), s.Name(), s.Type(), t.Name(), t.Description())
		}
	}
	tw.Flush()
	return nil
}
