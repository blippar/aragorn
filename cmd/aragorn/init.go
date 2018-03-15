package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/server"
)

const initHelp = `Set up a new Aragorn project`

type initCommand struct{}

func (*initCommand) Name() string { return "init" }
func (*initCommand) Args() string {

	return fmt.Sprintf("(%s) (default: HTTP)", availableTestSuites(" | "))
}
func (*initCommand) ShortHelp() string { return initHelp }
func (*initCommand) LongHelp() string  { return initHelp }
func (*initCommand) Hidden() bool      { return false }

func (*initCommand) Register(fs *flag.FlagSet) {}

func (*initCommand) Run(args []string) error {
	typ := "HTTP"
	if len(args) > 0 && args[0] != "" {
		typ = args[0]
	}
	r := plugin.Get(plugin.TestSuitePlugin, typ)
	if r == nil {
		return fmt.Errorf("%q is not a valid test suite type (supported: %s)", typ, availableTestSuites(", "))
	}
	type exampler interface {
		Example() interface{}
	}
	var suiteCfg []byte
	if ex, ok := r.Config.(exampler); ok {
		cfg := ex.Example()
		var err error
		suiteCfg, err = json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("could not marshal testsuite config: %v", err)
		}
	}

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

	w := json.NewEncoder(f)
	w.SetIndent("", "  ")
	s := &server.SuiteConfig{
		Type:  typ,
		Name:  strings.Title(pkg),
		Suite: suiteCfg,
	}
	return w.Encode(s)
}

func availableTestSuites(sep string) string {
	rs := plugin.ForType(plugin.TestSuitePlugin)
	ss := make([]string, len(rs))
	for i, r := range rs {
		ss[i] = r.ID
	}
	sort.Strings(ss)
	return strings.Join(ss, sep)
}
