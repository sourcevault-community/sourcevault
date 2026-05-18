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
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config represents the global application configuration.
// It aggregates settings for data storage, logging, and the various
// servers (Web and SSH) that make up the SourceVault platform.
type Config struct {
	RootDir  string    // RootDir is the base directory for all sourcevault data and repositories.
	LogFile  string    // LogFile is the path to the application log file, relative to RootDir if not absolute.
	LogLevel string    // LogLevel specifies the minimum level of logs to emit (e.g., DEBUG, INFO, WARN, ERROR).
	Web      WebConfig     // Web contains configuration for the administrative web server and UI.
	Ssh      SshConfig     // Ssh contains configuration for the built-in Git SSH server.
	Metrics  MetricsConfig // Metrics contains configuration for the dedicated Prometheus metrics server.
}

// WebConfig holds settings for the HTTP/HTTPS web interface and API.
// This interface allows for repository management and system administration.
type WebConfig struct {
	Enabled bool   // Enabled determines whether the web server should be started upon application launch.
	Host    string // Host is the network interface or IP address to bind the web server to.
	Port    int    // Port is the port number the web server will listen on.
}

// MetricsConfig holds settings for the dedicated network-isolated metrics server.
type MetricsConfig struct {
	Enabled bool   // Enabled determines whether the metrics server should run.
	Host    string // Host is the network interface to bind to (usually 127.0.0.1 for security).
	Port    int    // Port is the port number the metrics server will listen on (default 9090).
}

// SshConfig holds settings for the built-in SSH server used for Git operations.
// This server provides secure access for cloning, pushing, and pulling repositories.
type SshConfig struct {
	Enabled bool   // Enabled determines whether the SSH server should be started upon application launch.
	Host    string // Host is the network interface or IP address to bind the SSH server to.
	Port    int    // Port is the port number the SSH server will listen on.
}

// Load initializes a Config instance by following a specific precedence order:
// 1. Sensible default values are applied.
// 2. Values are loaded from an environment file (defaulting to "sourcevault.env").
// 3. Environment variables override any previously set values.
// Finally, the configuration is sanitized to ensure valid paths and formats.
func Load() (*Config, error) {
	// Determine which environment file to load.
	// The file path can be customized via the SOURCEVAULT_CONFIG_FILE environment variable.
	envFile := os.Getenv("SOURCEVAULT_CONFIG_FILE")
	if envFile == "" {
		envFile = "sourcevault.env"
	}

	// Attempt to load the environment variables from the specified file.
	// If the file is missing or unreadable, we proceed with system environment variables and defaults.
	_ = godotenv.Load(envFile)

	// Initialize the configuration with sensible defaults for a standard deployment.
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
		Metrics: MetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    9090,
		},
	}

	// Override global settings from environment variables if they are defined.
	if val := os.Getenv("SOURCEVAULT_ROOT_DIR"); val != "" {
		c.RootDir = val
	}
	if val := os.Getenv("SOURCEVAULT_LOG_FILE"); val != "" {
		c.LogFile = val
	}
	if val := os.Getenv("SOURCEVAULT_LOG_LEVEL"); val != "" {
		c.LogLevel = normalizeLogLevel(val)
	}

	// Override Web server settings from environment variables if they are defined.
	if val := os.Getenv("SOURCEVAULT_WEB_ENABLED"); val != "" {
		// We use EqualFold for a case-insensitive boolean check.
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
	
	// Override Metrics server settings from environment variables if they are defined.
	if val := os.Getenv("SOURCEVAULT_METRICS_ENABLED"); val != "" {
		c.Metrics.Enabled = strings.EqualFold(val, "true")
	}
	if val := os.Getenv("SOURCEVAULT_METRICS_HOST"); val != "" {
		c.Metrics.Host = val
	}
	if val := os.Getenv("SOURCEVAULT_METRICS_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			c.Metrics.Port = p
		}
	}

	// Override SSH server settings from environment variables if they are defined.
	if val := os.Getenv("SOURCEVAULT_SSH_ENABLED"); val != "" {
		// We use EqualFold for a case-insensitive boolean check.
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

	// Sanitize the final configuration to ensure path consistency and validity.
	c.sanitize()
	return c, nil
}

// sanitize ensures that all directory and file paths within the configuration
// are converted to absolute paths. It also handles relative paths for the
// log file by anchoring them to the RootDir.
func (c *Config) sanitize() {
	// Convert RootDir to an absolute path for consistent filesystem operations.
	if abs, err := filepath.Abs(c.RootDir); err == nil {
		c.RootDir = abs
	}

	// If a log file is specified, ensure it is an absolute path.
	if c.LogFile != "" {
		// If the path is relative, prefix it with the RootDir.
		if !filepath.IsAbs(c.LogFile) {
			c.LogFile = filepath.Join(c.RootDir, c.LogFile)
		}
		// Convert the final path to an absolute representation.
		if abs, err := filepath.Abs(c.LogFile); err == nil {
			c.LogFile = abs
		}
	}
}

// normalizeLogLevel converts a raw log level string (e.g., from an environment variable)
// into a standardized uppercase representation (DEBUG, INFO, WARN, ERROR)
// that is recognized by the application's logging system.
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
		// Default to INFO if the provided level is unrecognized.
		return "INFO"
	}
}

func (c *Config) LogLevelValue() string {
	if c.LogLevel != "" {
		return normalizeLogLevel(c.LogLevel)
	}

	return "INFO"
}
