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
	fmt.Fprintln(tw, "Path\tName\tType")
	for _, s := range suites {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", s.Path(), s.Name(), s.Type())
	}
	tw.Flush()
	return nil
}
