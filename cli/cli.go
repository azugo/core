package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Options configures the root command created by Run.
type Options struct {
	// Use is the one-line usage string for the root command.
	Use string
	// Short is the short description shown in the help output.
	Short string
	// Long is the long description shown in the help output.
	Long string
	// Version sets the version string on the root command.
	Version string
	// Flags registers persistent (global) flags on the root command.
	// Persistent flags are inherited by all subcommands.
	Flags func(*pflag.FlagSet)
}

// CommandOption is a functional option applied when registering a command.
type CommandOption func(*commandMeta)

type commandMeta struct {
	cmd       *cobra.Command
	isDefault bool
}

// AsDefault marks the registered command as the default command to run when
// the root command is invoked without a subcommand.
func AsDefault() CommandOption {
	return func(m *commandMeta) {
		m.isDefault = true
	}
}

var (
	mu         sync.Mutex
	cmds       []*commandMeta
	defaultCmd *cobra.Command
)

// Register adds a constructed *cobra.Command to the global registry.
// Call this from package init() of command packages so they are added
// automatically when the application starts.
// Use AsDefault() to mark a command as the default when no subcommand is given.
func Register(cmd *cobra.Command, opts ...CommandOption) {
	if cmd == nil {
		return
	}

	meta := &commandMeta{cmd: cmd}
	for _, opt := range opts {
		opt(meta)
	}

	mu.Lock()
	defer mu.Unlock()

	cmds = append(cmds, meta)

	if meta.isDefault {
		defaultCmd = cmd
	}
}

// Run creates a root command from opts, attaches all registered commands,
// sets a signal-aware context on the root (so subcommands inherit it), and
// executes the root command.
// If Execute returns an error the process exits with status 1.
func Run(opts Options) {
	root := &cobra.Command{
		Use:     opts.Use,
		Short:   opts.Short,
		Long:    opts.Long,
		Version: opts.Version,
	}

	if opts.Flags != nil {
		opts.Flags(root.PersistentFlags())
	}

	mu.Lock()

	for _, m := range cmds {
		root.AddCommand(m.cmd)
	}

	if defaultCmd != nil {
		root.Run = defaultCmd.Run
		root.RunE = defaultCmd.RunE
	}

	mu.Unlock()

	if code := func() int {
		// Create a context that cancels on SIGINT/SIGTERM and attach it to root.
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		root.SetContext(ctx)

		if err := root.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)

			return 1
		}

		return 0
	}(); code != 0 {
		os.Exit(code)
	}
}
