package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blippar/aragorn/server"
)

const initHelp = `Set up a new Aragorn project`

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
	if err := os.Mkdir(".aragorn", 0777); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	pkg := filepath.Base(dir)
	path := filepath.Join(".aragorn", pkg+testSuiteJSONSuffix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	typ := "HTTP"
	if len(args) > 0 && args[0] != "" {
		if args[0] != "HTTP" && args[0] != "GRPC" {
			return fmt.Errorf("%q is not a valid test suite type", args[0])
		}
		typ = args[0]
	}
	w := json.NewEncoder(f)
	w.SetIndent("", "  ")
	s := &server.SuiteConfig{
		Type: typ,
		Name: strings.Title(pkg),
	}
	return w.Encode(s)
}
