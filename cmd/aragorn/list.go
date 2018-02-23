package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

const listShortHelp = `List the test suites`
const listLongHelp = `List the test suites` + fileHelp

type listCommand struct{}

func (*listCommand) Name() string { return "list" }
func (*listCommand) Args() string {
	return "[file ...]"
}
func (*listCommand) ShortHelp() string { return listShortHelp }
func (*listCommand) LongHelp() string  { return listLongHelp }
func (*listCommand) Hidden() bool      { return false }

func (*listCommand) Register(fs *flag.FlagSet) {}

func (*listCommand) Run(args []string) error {
	suites, err := getSuitesFromArgs(args, false)
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
