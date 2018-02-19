package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/blippar/aragorn/log"

	_ "github.com/blippar/aragorn/notifier/slack"
	_ "github.com/blippar/aragorn/testsuite/grpcexpect"
	_ "github.com/blippar/aragorn/testsuite/httpexpect"
)

const (
	successExitCode = 0
	errorExitCode   = 1
)

const fileHelp = `

For each operand that names a file of type directory,
all the files with the extension .suite.json in the directory will be used as test suites.

For each operand that names a file of a type other than directory,
the file will be used as a test suite.

If no operands are given, the current directory is used.
`

type command interface {
	Name() string           // "foobar"
	Args() string           // "<baz> [quux...]"
	ShortHelp() string      // "Foo the first bar"
	LongHelp() string       // "Foo the first bar meeting the following conditions..."
	Register(*flag.FlagSet) // command-specific flags
	Hidden() bool           // indicates whether the command should be hidden from help output
	Run([]string) error
}

func main() {
	os.Exit(run())
}

func run() int {
	// Build the list of available commands.
	commands := [...]command{
		&runCommand{},
		&execCommand{},
		&watchCommand{},
		&listCommand{},
		&versionCommand{},
		&initCommand{},
	}

	usage := func(w io.Writer) {
		fmt.Fprintln(w, "Aragorn is a regression testing tool")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage: \"aragorn [command]\"")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Commands:")
		fmt.Fprintln(w)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, cmd := range commands {
			if !cmd.Hidden() {
				fmt.Fprintf(tw, "\t%s\t%s\n", cmd.Name(), cmd.ShortHelp())
			}
		}
		tw.Flush()
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Use \"aragorn help [command]\" for more information about a command.")
	}

	cmdName, printCommandHelp, exit := parseArgs(os.Args)
	if exit {
		usage(os.Stderr)
		return errorExitCode
	}

	for _, cmd := range commands {
		if cmd.Name() == cmdName {
			// Build flag set with global flags in there.
			fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
			fs.SetOutput(os.Stderr)
			debug := fs.Bool("debug", false, "Enable debug mode")
			logLevel := fs.String("log-level", "info", `Set the logging level ("debug"|"info"|"warn"|"error"|"fatal")`)

			// Register the subcommand flags in there, too.
			cmd.Register(fs)

			// Override the usage text to something nicer.
			resetUsage(os.Stderr, fs, cmdName, cmd.Args(), cmd.LongHelp())

			if printCommandHelp {
				fs.Usage()
				return errorExitCode
			}

			// Parse the flags the user gave us.
			// flag package automatically prints usage and error message in err != nil
			// or if '-h' flag provided
			if err := fs.Parse(os.Args[2:]); err != nil {
				return errorExitCode
			}

			if err := log.Init(*logLevel, *debug); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return errorExitCode
			}
			defer log.L().Sync()
			// Run the command with the post-flag-processing args.
			if err := cmd.Run(fs.Args()); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return errorExitCode
			}

			// Easy peasy livin' breezy.
			return successExitCode
		}
	}

	fmt.Fprintf(os.Stderr, "aragorn: %s: no such command\n", cmdName)
	usage(os.Stderr)
	return errorExitCode
}

func resetUsage(w io.Writer, fs *flag.FlagSet, name, args, longHelp string) {
	var (
		hasFlags   bool
		flagBlock  bytes.Buffer
		flagWriter = tabwriter.NewWriter(&flagBlock, 0, 4, 2, ' ', 0)
	)
	fs.VisitAll(func(f *flag.Flag) {
		hasFlags = true
		// Default-empty string vars should read "(default: <none>)"
		// rather than the comparatively ugly "(default: )".
		defValue := f.DefValue
		if defValue == "" {
			defValue = "<none>"
		}
		fmt.Fprintf(flagWriter, "\t-%s\t%s (default: %s)\n", f.Name, f.Usage, defValue)
	})
	flagWriter.Flush()
	fs.Usage = func() {
		fmt.Fprintf(w, "Usage: aragorn %s %s\n", name, args)
		fmt.Fprintln(w)
		fmt.Fprintln(w, strings.TrimSpace(longHelp))
		fmt.Fprintln(w)
		if hasFlags {
			fmt.Fprintln(w, "Flags:")
			fmt.Fprintln(w)
			fmt.Fprintln(w, flagBlock.String())
		}
	}
}

// parseArgs determines the name of the dep command and whether the user asked for
// help to be printed.
func parseArgs(args []string) (cmdName string, printCmdUsage bool, exit bool) {
	isHelpArg := func() bool {
		return strings.Contains(strings.ToLower(args[1]), "help") || strings.ToLower(args[1]) == "-h"
	}

	switch len(args) {
	case 0, 1:
		exit = true
	case 2:
		if isHelpArg() {
			exit = true
		} else {
			cmdName = args[1]
		}
	default:
		if isHelpArg() {
			cmdName = args[2]
			printCmdUsage = true
		} else {
			cmdName = args[1]
		}
	}
	return cmdName, printCmdUsage, exit
}
