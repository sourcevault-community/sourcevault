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

package config

import (
	"os"
	"strings"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the global application configuration.
type Config struct {
	RootDir  string    // The base directory for all sourcevault data and repositories
	LogFile  string    // The path to the application log file
	LogLevel string    // The minimum level of logs to emit (e.g. debug, info, warn, error)
	Web      WebConfig // Configuration for the web server and UI
	Ssh      SshConfig // Configuration for the Git SSH server
}

// WebConfig holds settings for the HTTP/HTTPS web interface and API.
type WebConfig struct {
	Enabled bool   // Whether the web server should be started
	Host    string // The network interface/IP to bind the web server to
	Port    int    // The port number for the web server
}

// SshConfig holds settings for the built-in SSH server used for Git operations.
type SshConfig struct {
	Enabled bool   // Whether the SSH server should be started
	Host    string // The network interface/IP to bind the SSH server to
	Port    int    // The port number for the SSH server (fixed unexported field typo)
}

// Load initializes a Config instance with default values and overrides
// them with values from environment variables if present.
func Load() (*Config, error) {
	// Determine which environment file to load, defaulting to sourcevault.env
	envFile := os.Getenv("SOURCEVAULT_CONFIG_FILE")
	if envFile == "" {
		envFile = "sourcevault.env"
	}

	// Attempt to load the environment variables from the file.
	// We ignore errors here since the file is optional.
	_ = godotenv.Load(envFile)

	// Initialize with sensible defaults
	c := &Config{
		RootDir:  "/home/sourcevault",
		LogFile:  "",
		LogLevel: "ERROR",
		Web: WebConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    8080,
		},
		Ssh: SshConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    2222,
		},
	}

	// Override global settings from environment variables
	if val := os.Getenv("SOURCEVAULT_ROOT_DIR"); val != "" {
		c.RootDir = val
	}
	if val := os.Getenv("SOURCEVAULT_LOG_FILE"); val != "" {
		c.LogFile = val
	}
	if val := os.Getenv("SOURCEVAULT_LOG_LEVEL"); val != "" {
		c.LogLevel = normalizeLogLevel(val)
	}

	// Override Web server settings from environment variables
	if val := os.Getenv("SOURCEVAULT_WEB_ENABLED"); val != "" {
		c.Web.Enabled = strings.EqualFold(val, "true")
	}
	if val := os.Getenv("SOURCEVAULT_WEB_HOST"); val != "" {
		c.Web.Host = val
	}
	if val := os.Getenv("SOURCEVAULT_WEB_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			c.Web.Port = p
		}
	}

	// Override SSH server settings from environment variables
	if val := os.Getenv("SOURCEVAULT_SSH_ENABLED"); val != "" {
		c.Ssh.Enabled = strings.EqualFold(val, "true")
	}
	if val := os.Getenv("SOURCEVAULT_SSH_HOST"); val != "" {
		c.Ssh.Host = val
	}
	if val := os.Getenv("SOURCEVAULT_SSH_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			c.Ssh.Port = p
		}
	}
	return c, nil
}

// normalizeLogLevel converts a raw log level string into a standardized
// uppercase representation supported by the system.
func normalizeLogLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "ERROR":
		return "ERROR"
	case "WARN", "WARNING":
		return "WARN"
	case "DEBUG":
		return "DEBUG"
	case "INFO":
		return "INFO"
	default:
		return "INFO"
	}
}
