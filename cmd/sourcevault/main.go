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
	"strings"
	"github.com/charmbracelet/lipgloss"
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

var commands = []command{
	{"help", "Display this message"},
	{"start", "Start service"},
}

// main is the primary entry point for the SourceVault application.
// It delegates execution to the run function.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	run()
}

// run handles the core startup and execution logic,
// such as displaying the banner and initializing the server.
func run() {
	fmt.Fprint(os.Stdout, banner)
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
