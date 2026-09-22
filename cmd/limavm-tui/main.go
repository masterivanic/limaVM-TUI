package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/epel/limavm-tui/internal/limactl"
	"github.com/epel/limavm-tui/internal/store"
	"github.com/epel/limavm-tui/internal/ui"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		flagDataDir = flag.String("data-dir", "", "Path to state and configuration directory")
		flagRefresh = flag.Duration("refresh", 3*time.Second, "Poll interval for VM and network status")
		flagLog     = flag.String("log", "", "Optional log file path for debug logs")
		flagVersion = flag.Bool("version", false, "Print limavm-tui version and exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Lima VM Manager — Terminal UI (TUI) for limactl\n\n")
		fmt.Fprintf(os.Stderr, "Usage: limavm-tui [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *flagVersion {
		fmt.Printf("limavm-tui %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Setup logging if requested
	if *flagLog != "" {
		f, err := tea.LogToFile(*flagLog, "limavm-tui")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open log file %q: %v\n", *flagLog, err)
		} else {
			defer f.Close()
			log.Printf("limavm-tui starting up (refresh=%v, data-dir=%q)", *flagRefresh, *flagDataDir)
		}
	}

	// Initialize limactl client
	client, err := limactl.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please ensure Lima (limactl) is installed and available in your PATH.\n")
		fmt.Fprintf(os.Stderr, "See https://github.com/lima-vm/lima for installation instructions.\n")
		os.Exit(1)
	}

	// Initialize local store
	st, err := store.New(*flagDataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing local storage: %v\n", err)
		os.Exit(1)
	}

	// Create root app model
	app := ui.NewApp(client, st, *flagRefresh)

	// Run full-screen Bubble Tea program
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running limavm-tui: %v\n", err)
		os.Exit(1)
	}
}
