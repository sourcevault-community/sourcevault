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
	"io"
	"strings"
	"log/slog"

	"github.com/charmbracelet/lipgloss"

	"sourcevault/internal/version"
	"sourcevault/internal/config"
	svlog "sourcevault/internal/log"
)

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
	// It uses a purple background with bold white text.
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
	// It uses a green color and fixed width for alignment.
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

// commands is the list of available CLI commands and their descriptions
// that are displayed when a user requests help or provides invalid arguments.
var commands = []command{
	{"help", "Display this message"},
	{"start", "Start service"},
}

// main is the primary entry point for the SourceVault application.
// It performs initial argument validation and delegates core execution
// to the run function. If run returns an error, it is printed to
// stderr and the application exits with a non-zero status.
func main() {
	// Ensure at least one command (e.g., 'start' or 'help') is provided.
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// Execute the application logic.
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run handles the core startup and execution logic of the application.
// Its responsibilities include:
// 1. Displaying the ASCII banner to the console.
// 2. Loading and validating the application configuration.
// 3. Printing build-time information (version, git hash, etc.).
// 4. Printing the active configuration for diagnostic purposes.
// 5. Initializing and starting the required services (Web, SSH, etc.).
func run(args []string, stdout, stderr io.Writer) error {
	// Display the visual identity of the application.
	fmt.Fprint(os.Stdout, banner)

	// Load the configuration from environment variables and .env files.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %s", err)
	}

	// Initialize the logger based on configuration
	closeLog := svlog.Init(cfg)
	defer closeLog()

	slog.Info("Application is starting up", "name", version.Current.AppName, "version", version.Current.AppVersion)


	// Output application build information to help with troubleshooting and support.
	slog.Info("Application information",
		"name", version.Current.AppName,
		"version", version.Current.AppVersion,
		"git_branch", version.Current.GitBranch,
		"git_commit", version.Current.GitCommit,
	)
	fmt.Printf("Application Information:\n")
	fmt.Printf("- Name          :  %s\n", version.Current.AppName)
	fmt.Printf("- Version       :  %s\n", version.Current.AppVersion)
	fmt.Printf("- Git Branch    :  %s\n", version.Current.GitBranch)
	fmt.Printf("- Git Commit    :  %s\n", version.Current.GitCommit)
	fmt.Printf("- Build Date    :  %s\n", version.Current.BuildDate)
	fmt.Printf("- Architecture  :  %s\n", version.Current.Architecture)

	// Output the full parsed configuration for diagnostic and startup verification.
	// This ensures the operator knows exactly which settings are being applied.
	fmt.Printf("Configuration:\n")
	fmt.Printf("- RootDir       :  %s\n", cfg.RootDir)
	fmt.Printf("- LogFile       :  %s\n", cfg.LogFile)
	fmt.Printf("- LogLevel      :  %s\n", cfg.LogLevel)
	fmt.Printf("- Web server configuration:\n")
	fmt.Printf("  - Enabled     :  %t\n", cfg.Web.Enabled)
	fmt.Printf("  - Host        :  %s\n", cfg.Web.Host)
	fmt.Printf("  - Port        :  %d\n", cfg.Web.Port)
	fmt.Printf("- SSH server configuration:\n")
	fmt.Printf("  - Enabled     :  %t\n", cfg.Ssh.Enabled)
	fmt.Printf("  - Host        :  %s\n", cfg.Ssh.Host)
	fmt.Printf("  - Port        :  %d\n", cfg.Ssh.Port)

	// TODO: Implement actual service startup logic based on the loaded configuration.
	return nil
}

// printUsage displays the available commands and general usage instructions.
// It leverages lipgloss styles for rich terminal output formatting.
func printUsage() {
	// Print the basic usage pattern
	fmt.Println(subheadingStyle.Render("Usage:"))
	fmt.Println(listStyle.Render("sourcevault <command> [arguments]"))

	// Print the list of available commands with their descriptions
	fmt.Println(subheadingStyle.Render("Available Commands:"))

	var sb strings.Builder
	for _, cmd := range commands {
		// Format each command line with its specific style
		sb.WriteString(commandStyle.Render(cmd.name))
		sb.WriteString(descStyle.Render(cmd.description))
		sb.WriteString("\n")
	}
	fmt.Println(listStyle.Render(sb.String()))
}
