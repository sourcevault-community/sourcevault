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
	"path/filepath"
	"testing"
)

// TestLoad_Defaults verifies that the configuration loader applies
// the correct default values when no environment variables are set.
func TestLoad_Defaults(t *testing.T) {
	// Clear relevant environment variables to ensure defaults are used
	os.Unsetenv("SOURCEVAULT_ROOT_DIR")
	os.Unsetenv("SOURCEVAULT_LOG_FILE")
	os.Unsetenv("SOURCEVAULT_LOG_LEVEL")
	os.Unsetenv("SOURCEVAULT_WEB_ENABLED")
	os.Unsetenv("SOURCEVAULT_WEB_HOST")
	os.Unsetenv("SOURCEVAULT_WEB_PORT")
	os.Unsetenv("SOURCEVAULT_SSH_ENABLED")
	os.Unsetenv("SOURCEVAULT_SSH_HOST")
	os.Unsetenv("SOURCEVAULT_SSH_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.LogLevel != "ERROR" {
		t.Errorf("expected LogLevel ERROR, got %s", cfg.LogLevel)
	}

	if !cfg.Web.Enabled {
		t.Error("expected Web.Enabled true by default")
	}

	if cfg.Web.Port != 8080 {
		t.Errorf("expected Web.Port 8080, got %d", cfg.Web.Port)
	}

	if cfg.Ssh.Port != 2222 {
		t.Errorf("expected Ssh.Port 2222, got %d", cfg.Ssh.Port)
	}
}

// TestLoad_EnvOverrides ensures that environment variables correctly
// override the default configuration settings.
func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("SOURCEVAULT_ROOT_DIR", "/tmp/sourcevault")
	os.Setenv("SOURCEVAULT_LOG_LEVEL", "DEBUG")
	os.Setenv("SOURCEVAULT_WEB_PORT", "9090")
	defer func() {
		os.Unsetenv("SOURCEVAULT_ROOT_DIR")
		os.Unsetenv("SOURCEVAULT_LOG_LEVEL")
		os.Unsetenv("SOURCEVAULT_WEB_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	absRootDir, _ := filepath.Abs("/tmp/sourcevault")
	if cfg.RootDir != absRootDir {
		t.Errorf("expected RootDir %s, got %s", absRootDir, cfg.RootDir)
	}

	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel DEBUG, got %s", cfg.LogLevel)
	}

	if cfg.Web.Port != 9090 {
		t.Errorf("expected Web.Port 9090, got %d", cfg.Web.Port)
	}
}

// TestNormalizeLogLevel verifies the normalization logic for various
// log level string inputs.
func TestNormalizeLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"INFO", "INFO"},
		{" warn ", "WARN"},
		{"WARNING", "WARN"},
		{"error", "ERROR"},
		{"invalid", "INFO"},
		{"", "INFO"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeLogLevel(tc.input)
			if got != tc.expected {
				t.Errorf("normalizeLogLevel(%q) = %q; expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestLogLevelValue checks that LogLevelValue correctly returns a
// standardized log level or a sensible default.
func TestLogLevelValue(t *testing.T) {
	cfg := &Config{LogLevel: "debug"}
	if cfg.LogLevelValue() != "DEBUG" {
		t.Errorf("expected DEBUG, got %s", cfg.LogLevelValue())
	}

	cfg.LogLevel = ""
	if cfg.LogLevelValue() != "INFO" {
		t.Errorf("expected INFO, got %s", cfg.LogLevelValue())
	}
}

// TestSanitize verifies that configuration paths are correctly converted
// to absolute paths during the sanitization phase.
func TestSanitize(t *testing.T) {
	cfg := &Config{
		RootDir: "test-data",
		LogFile: "app.log",
	}
	cfg.sanitize()

	if !filepath.IsAbs(cfg.RootDir) {
		t.Errorf("RootDir %s should be absolute", cfg.RootDir)
	}

	if !filepath.IsAbs(cfg.LogFile) {
		t.Errorf("LogFile %s should be absolute", cfg.LogFile)
	}

	expectedLogFile := filepath.Join(cfg.RootDir, "app.log")
	if cfg.LogFile != expectedLogFile {
		t.Errorf("expected LogFile %s, got %s", expectedLogFile, cfg.LogFile)
	}
}
