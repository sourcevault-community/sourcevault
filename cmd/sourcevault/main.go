// SPDX-License-Identifier: AGPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The sourcevault Authors. All rights reserved.
// ===================================================================================================================================== //
// MP""""""`MM MMP"""""YMM M""MMMMM""M MM"""""""`MM MM'""""'YMM MM""""""""`M M""MMMMM""M MMP"""""""MM M""MMMMM""M M""MMMMMMMM M""""""""M //
// M  mmmmm..M M' .mmm. `M M  MMMMM  M MM  mmmm,  M M' .mmm. `M MM  mmmmmmmM M  MMMMM  M M' .mmmm  MM M  MMMMM  M M  MMMMMMMM Mmmm  mmmM //
// M.      `YM M  MMMMM  M M  MMMMM  M M'        .M M  MMMMMooM M`      MMMM M  MMMMP  M M         `M M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// MMMMMMM.  M M  MMMMM  M M  MMMMM  M MM  MMMb. "M M  MMMMMMMM MM  MMMMMMMM M  MMMM' .M M  MMMMM  MM M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// M. .MMM'  M M. `MMM' .M M  `MMM'  M MM  MMMMM  M M. `MMM' .M MM  MMMMMMMM M  MMP' .MM M  MMMMM  MM M  `MMM'  M M  MMMMMMMM MMMM  MMMM //
// Mb.     .dM MMb     dMM Mb       dM MM  MMMMM  M MM.     .dM MM        .M M     .dMMM M  MMMMM  MM Mb       dM M         M MMMM  MMMM //
// MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMM //
// ===================================================================================================================================== //
// This program is free software: you can redistribute it and/or modify                                                                  //
// it under the terms of the GNU Affero General Public License as                                                                        //
// published by the Free Software Foundation, either version 3 of the                                                                    //
// License, or (at your option) any later version.                                                                                       //
//                                                                                                                                       //
// This program is distributed in the hope that it will be useful,                                                                       //
// but WITHOUT ANY WARRANTY; without even the implied warranty of                                                                        //
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                                                         //
// GNU Affero General Public License for more details.                                                                                   //
//                                                                                                                                       //
// You should have received a copy of the GNU Affero General Public License                                                              //
// along with this program.  If not, see <https://www.gnu.org/licenses/>.                                                                //
// ===================================================================================================================================== //

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"io"
	"strings"
	"context"
	"log/slog"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/errgroup"

	"sourcevault/internal/version"
	"sourcevault/internal/config"
	sv_log "sourcevault/internal/log"
	sv_web "sourcevault/internal/web"
)

// banner is the ASCII art visual identity displayed when the application starts.
const banner = `
===========================================================================
 ####   ####  #    # #####   ####  ###### #    #   ##   #    # #      #####
#      #    # #    # #    # #    # #      #    #  #  #  #    # #        #
 ####  #    # #    # #    # #      #####  #    # #    # #    # #        #
     # #    # #    # #####  #      #      #    # ###### #    # #        #
#    # #    # #    # #   #  #    # #       #  #  #    # #    # #        #
 ####   ####   ####  #    #  ####  ######   ##   #    #  ####  ######   #
===========================================================================
`

var (
	// titleStyle defines the look of the main application title.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginTop(1)

	// subheadingStyle is used for section headers like "Usage:" and "Available Commands:".
	subheadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1)

	// commandStyle highlights the command names in the list.
	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true).
			Width(12)

	// descStyle defines the appearance of command descriptions.
	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A9A9A9"))

	// listStyle adds indentation and spacing to the command list.
	listStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginBottom(1)
)

// command represents a CLI command with its name and a brief description.
type command struct {
	name		string
	description	string
}

// commands defines the list of supported CLI actions.
// This list is used to generate the help menu and validate user input.
var commands = []command{
	{"help", "Display this message"},
	{"start", "Start service"},
}

// main is the primary entry point for the SourceVault application.
func main() {
	// Simple argument check: the user must provide at least one command.
	if len(os.Args) < 2 {
		printUsage(os.Stdout)
		return
	}

	// Delegate the core logic to the run function.
	// This approach facilitates easier testing and cleaner error handling.
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run orchestrates the application's bootstrap process.
func run(args []string, stdout, stderr io.Writer) error {
	// The first argument after the binary name is the command.
	cmd := args[1]

	// Handle the 'help' command early to avoid unnecessary setup.
	if cmd == "help" {
		printUsage(stdout)
		return nil
	}

	// Print the ASCII banner to the provided stdout writer.
	fmt.Fprint(stdout, banner)
	fmt.Fprint(stdout, "\n\n")

	// Step 1: Load application configuration.
	// Configuration is sourced from environment variables and optional .env files.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Step 2: Initialize the logging system.
	// The logger is configured based on the LogFile and LogLevel settings.
	closeLog := sv_log.Init(cfg)
	defer closeLog()

	// Step 3: Log application metadata.
	// This helps in identifying the exact build running in a given environment.
	slog.Info("Application is starting up", "application_name", version.Current.AppName, "application_version", version.Current.AppVersion)

	slog.Info("Application information",
		"application_name", version.Current.AppName,
		"application_version", version.Current.AppVersion,
		"git_branch", version.Current.GitBranch,
		"git_commit", version.Current.GitCommit,
		"build_date", version.Current.BuildDate,
		"architecture", version.Current.Architecture,
	)

	// Step 4: Log the parsed configuration at DEBUG level.
	// This is invaluable for troubleshooting environment-specific setup issues.
	slog.Debug("Base configuration",
		"root_dir", cfg.RootDir,
		"log_file", cfg.LogFile,
		"log_level", cfg.LogLevel,
	)
	slog.Debug("Web server configuration",
		"enabled", cfg.Web.Enabled,
		"host", cfg.Web.Host,
		"port", cfg.Web.Port,
	)
	slog.Debug("SSH server configuration",
		"enabled", cfg.Ssh.Enabled,
		"host", cfg.Ssh.Host,
		"port", cfg.Ssh.Port,
	)

	// Set up signal handling for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Use an errgroup to manage background services.
	g, ctx := errgroup.WithContext(ctx)

	// Keep track of how many services were successfully started.
	var started int

	switch cmd {
	case "start":
		// Launch the Web server if enabled.
		if cfg.Web.Enabled {
			started++
			g.Go(func() error {
				return sv_web.Run(ctx, cfg)
			})
		}
		// TODO: Launch the SSH server if enabled.
		if cfg.Ssh.Enabled {
			// started++
			// g.Go(func() error { ... })
		}
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", cmd)
	}

	// If no services were enabled to start, we still wait for a signal
	// to keep the application "running" as requested by the user,
	// rather than exiting immediately.
	if started == 0 {
		slog.Warn("No services (Web/SSH) are enabled; waiting for interrupt signal")
		<-ctx.Done()
	} else {
		// Wait for all background services to complete or for a termination signal.
		if err := g.Wait(); err != nil {
			return fmt.Errorf("application error: %w", err)
		}
	}

	slog.Info("Application shut down gracefully")
	return nil
}

// printUsage displays the available commands and general usage instructions.
// It leverages lipgloss styles for rich terminal output formatting.
func printUsage(out io.Writer) {
	// Print the basic usage pattern
	fmt.Fprintln(out, subheadingStyle.Render("Usage:"))
	fmt.Fprintln(out, listStyle.Render("sourcevault <command> [arguments]"))

	// Print the list of available commands with their descriptions
	fmt.Fprintln(out, subheadingStyle.Render("Available Commands:"))

	var sb strings.Builder
	for _, cmd := range commands {
		// Format each command line with its specific style
		sb.WriteString(commandStyle.Render(cmd.name))
		sb.WriteString(descStyle.Render(cmd.description))
		sb.WriteString("\n")
	}
	fmt.Fprintln(out, listStyle.Render(sb.String()))
}
