package main

import (
	"flag"
	"fmt"
	"runtime"
)

var (
	version    = "devel"
	buildDate  string
	commitHash string
)

const versionHelp = `Show the aragorn version information`

type versionCommand struct{}

func (*versionCommand) Name() string { return "version" }
func (*versionCommand) Args() string {
	return ""
}
func (*versionCommand) ShortHelp() string { return versionHelp }
func (*versionCommand) LongHelp() string  { return versionHelp }
func (*versionCommand) Hidden() bool      { return false }

func (*versionCommand) Register(fs *flag.FlagSet) {}

func (*versionCommand) Run(args []string) error {
	fmt.Printf(`aragorn:
 version     : %s
 build date  : %s
 git hash    : %s
 go version  : %s
 go compiler : %s
 platform    : %s/%s
`, version, buildDate, commitHash,
		runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
	return nil
}
