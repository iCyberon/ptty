package main

import (
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iCyberon/ptty/internal/cli"
	"github.com/iCyberon/ptty/internal/scanner"
	"github.com/iCyberon/ptty/internal/tui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	version = "dev"

	jsonOutput bool
	noColor    bool
	showAll    bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "ptty",
		Short:   "Port Analyzer — see what's running on your ports",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				s := newScanner()
				ports, err := s.ListPorts(!showAll)
				if err != nil {
					return err
				}
				cli.PrintJSON(ports)
				return nil
			}
			return runTUI(0)
		},
	}

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&showAll, "all", "a", false, "Include system processes")

	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(detailCmd())
	rootCmd.AddCommand(psCmd())
	rootCmd.AddCommand(cleanCmd())
	rootCmd.AddCommand(watchCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newScanner() scanner.Scanner {
	return scanner.New()
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List listening ports",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := newScanner()
			devOnly := !showAll
			ports, err := s.ListPorts(devOnly)
			if err != nil {
				return err
			}

			if jsonOutput {
				cli.PrintJSON(ports)
			} else {
				cli.PrintPortTable(ports, !noColor)
			}
			return nil
		},
	}
}

func detailCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detail [port]",
		Short: "Show details for a specific port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid port: %s", args[0])
			}

			s := newScanner()
			info, err := s.GetPortDetail(port)
			if err != nil {
				return err
			}

			if jsonOutput {
				cli.PrintJSON(info)
			} else {
				cli.PrintPortDetail(info, !noColor)
			}
			return nil
		},
	}
}

func psCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List running dev processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := newScanner()
			devOnly := !showAll
			procs, err := s.GetAllProcesses(devOnly)
			if err != nil {
				return err
			}

			if jsonOutput {
				cli.PrintJSON(procs)
			} else {
				cli.PrintProcessTable(procs, !noColor)
			}
			return nil
		},
	}
}

func cleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Find and kill orphaned processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := newScanner()
			orphans, err := s.FindOrphans()
			if err != nil {
				return err
			}

			if jsonOutput {
				cli.PrintJSON(orphans)
				return nil
			}

			return cli.RunCleanInteractive(s, orphans, !noColor)
		},
	}
}

func watchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Monitor port changes in real time",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return cli.RunWatchJSON(newScanner(), !showAll)
			}
			return runTUI(2) // Watch tab
		},
	}
}

func runTUI(initialTab int) error {
	s := newScanner()
	app := tui.NewApp(s, initialTab)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
