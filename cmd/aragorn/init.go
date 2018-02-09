package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/blippar/aragorn/server"
)

const initHelp = `Creates all required files to setup Aragorn for a repository`

type initCommand struct{}

func (*initCommand) Name() string { return "init" }
func (*initCommand) Args() string {
	return "(HTTP | GRPC) (default: HTTP)"
}
func (*initCommand) ShortHelp() string { return initHelp }
func (*initCommand) LongHelp() string  { return initHelp }
func (*initCommand) Hidden() bool      { return false }

func (*initCommand) Register(fs *flag.FlagSet) {}

func (*initCommand) Run(args []string) error {
	s := &server.SuiteConfig{Type: "HTTP"}
	if err := os.Mkdir(".aragorn", 0777); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	pkg := path.Base(dir)
	file, err := os.Create(".aragorn/" + pkg + ".suite.json")
	if err != nil {
		return err
	}
	defer file.Close()
	s.Name = pkg + " test suite"
	if len(args) > 0 && args[0] != "" {
		if args[0] != "HTTP" && args[0] != "GRPC" {
			return fmt.Errorf("%q is not a valid test suite type", args[0])
		}
		s.Type = args[0]
	}
	return json.NewEncoder(file).Encode(s)
}
