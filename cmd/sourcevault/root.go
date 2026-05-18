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
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"sourcevault/internal/config"
	sv_log "sourcevault/internal/log"
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
	subheadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1)

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true).
			Width(12)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A9A9A9"))

	listStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginBottom(1)
)

var (
	appCfg   *config.Config
	closeLog func()
)

var rootCmd = &cobra.Command{
	Use:   "sourcevault",
	Short: "SourceVault: The Federated Code Collaboration Platform",
	Long:  banner + "\nSourceVault is an open-source decentralized Git hosting platform.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Do not initialize heavy resources if just calling help
		if cmd.Name() == "help" {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading configuration: %w", err)
		}
		appCfg = cfg
		closeLog = sv_log.Init(cfg)

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if closeLog != nil {
			closeLog()
		}
	},
}

func init() {
	// Override the default Help function to retain the custom Lipgloss styling
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), cmd.Long)
		fmt.Fprint(cmd.OutOrStdout(), "\n\n")

		fmt.Fprintln(cmd.OutOrStdout(), subheadingStyle.Render("Usage:"))
		fmt.Fprintln(cmd.OutOrStdout(), listStyle.Render(cmd.UseLine()))

		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(cmd.OutOrStdout(), subheadingStyle.Render("Available Commands:"))
			var sb strings.Builder
			for _, subCmd := range cmd.Commands() {
				if subCmd.IsAvailableCommand() || subCmd.Name() == "help" {
					sb.WriteString(commandStyle.Render(subCmd.Name()))
					sb.WriteString(descStyle.Render(subCmd.Short))
					sb.WriteString("\n")
				}
			}
			fmt.Fprint(cmd.OutOrStdout(), listStyle.Render(sb.String()))
		}

		if cmd.HasAvailableLocalFlags() {
			fmt.Fprintln(cmd.OutOrStdout(), subheadingStyle.Render("Flags:"))
			fmt.Fprintln(cmd.OutOrStdout(), listStyle.Render(cmd.LocalFlags().FlagUsages()))
		}
	})
}
